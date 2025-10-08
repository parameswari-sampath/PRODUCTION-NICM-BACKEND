package handlers

import (
	"context"
	"encoding/json"
	"log"
	"mcq-exam/db"
	"mcq-exam/utils"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SendEmailRequest struct {
	ToEmail  string `json:"to_email"`
	ToName   string `json:"to_name"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
}

// SendEmailHandler handles POST /api/mail/send
func SendEmailHandler(c *fiber.Ctx) error {
	var req SendEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if strings.TrimSpace(req.ToEmail) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to_email is required"})
	}
	if strings.TrimSpace(req.Subject) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subject is required"})
	}
	if strings.TrimSpace(req.HTMLBody) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "html_body is required"})
	}

	// Send email
	params := utils.SendEmailParams{
		ToEmail:  req.ToEmail,
		ToName:   req.ToName,
		Subject:  req.Subject,
		HTMLBody: req.HTMLBody,
	}

	zeptoResp, err := utils.SendEmail(params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to send email",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":    "Email sent successfully",
		"to":         req.ToEmail,
		"subject":    req.Subject,
		"request_id": zeptoResp.RequestID,
	})
}

type SendAllRequest struct {
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
}

// SendAllEmailsHandler handles POST /api/mail/send-all
// Sends personalized emails to all students with {{name}} replacement
func SendAllEmailsHandler(c *fiber.Ctx) error {
	var req SendAllRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if strings.TrimSpace(req.Subject) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subject is required"})
	}
	if strings.TrimSpace(req.HTMLBody) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "html_body is required"})
	}

	// Get all students from database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT id, name, email FROM students ORDER BY id`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch students"})
	}
	defer rows.Close()

	type Student struct {
		ID    int
		Name  string
		Email string
	}

	var students []Student
	for rows.Next() {
		var student Student
		if err := rows.Scan(&student.ID, &student.Name, &student.Email); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to scan student"})
		}
		students = append(students, student)
	}

	if len(students) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No students found in database"})
	}

	// Send emails to all students
	sentCount := 0

	for _, student := range students {
		// Personalize email by replacing {{name}}
		personalizedBody := strings.ReplaceAll(req.HTMLBody, "{{name}}", student.Name)

		// Send email
		params := utils.SendEmailParams{
			ToEmail:  student.Email,
			ToName:   student.Name,
			Subject:  req.Subject,
			HTMLBody: personalizedBody,
		}

		zeptoResp, err := utils.SendEmail(params)

		// All emails marked as "sent" initially
		// Webhook will update to "bounced" if delivery fails
		status := "sent"
		var requestID, responseCode, responseMessage *string
		var zeptoResponseJSON *string

		if err == nil {
			sentCount++
			requestID = &zeptoResp.RequestID
			if len(zeptoResp.Data) > 0 {
				responseCode = &zeptoResp.Data[0].Code
				responseMessage = &zeptoResp.Data[0].Message
			}
			// Store full response as JSON
			jsonBytes, _ := json.Marshal(zeptoResp)
			jsonStr := string(jsonBytes)
			zeptoResponseJSON = &jsonStr
		}

		// Log to database (even if API call failed, log for tracking)
		logQuery := `
			INSERT INTO email_logs (student_id, email, subject, status, request_id, response_code, response_message, zepto_response, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`
		_, _ = db.Pool.Exec(context.Background(), logQuery, student.ID, student.Email, req.Subject, status, requestID, responseCode, responseMessage, zeptoResponseJSON)

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return c.JSON(fiber.Map{
		"message": "All emails sent successfully",
		"total":   len(students),
		"sent":    sentCount,
	})
}

// ResendConferenceInvitationHandler handles POST /api/mail/resend-conference
// Resends conference invitation to students who haven't opened the first email
// Reuses existing conference tokens (no new token generation)
func ResendConferenceInvitationHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get students who have NOT attended the conference but have existing tokens
	query := `
		SELECT et.student_id, s.name, s.email, et.conference_token
		FROM email_tracking et
		JOIN students s ON et.student_id = s.id
		WHERE et.email_type = 'firstMail'
		  AND et.conference_attended = false
		  AND et.conference_token IS NOT NULL
		ORDER BY et.student_id ASC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch students"})
	}
	defer rows.Close()

	type StudentWithToken struct {
		ID             int
		Name           string
		Email          string
		ConferenceToken string
	}

	var students []StudentWithToken
	for rows.Next() {
		var st StudentWithToken
		if err := rows.Scan(&st.ID, &st.Name, &st.Email, &st.ConferenceToken); err != nil {
			continue
		}
		students = append(students, st)
	}

	if len(students) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No students found who need resend",
			"total":   0,
			"sent":    0,
		})
	}

	// Get frontend URL from environment
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://nicm.smart-mcq.com"
	}

	sentCount := 0
	for _, student := range students {
		// Reuse existing conference token
		conferenceLink := frontendURL + "/live?token=" + student.ConferenceToken

		// Email body - same as Phase 1 first mail
		htmlBody := `
		<div style="font-family: Arial, sans-serif; max-width: 700px; margin: 0 auto; padding: 20px;">
			<h2 style="color: #2c3e50;">Invitation to the Inaugural Virtual Meeting – CoopQuest - An International Online Cooperative Conclave</h2>

			<p>Dear ` + student.Name + `,</p>

			<p><strong>Greetings from Natesan Institute of Cooperative Management (NICM), Chennai!</strong></p>

			<p>In commemoration of the <strong>International Year of Cooperatives</strong> and in alignment with the vision of <strong>"Sahakar Se Samriddhi"</strong> (Prosperity through Cooperation), we are delighted to host the <strong>International Online Quiz on Cooperatives</strong>. This event celebrates the strength of the cooperative movement in fostering inclusive growth, empowerment, and sustainable development across the globe.</p>

			<p>We cordially invite you to join the <strong>Inaugural Virtual Meeting</strong> of the International Online Quiz:</p>

			<div style="background-color: #f8f9fa; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0;">
				<p style="margin: 5px 0;"><strong>📅 Date:</strong> 8th October 2025</p>
				<p style="margin: 5px 0;"><strong>🕒 Login Time:</strong> 1:45 PM (IST) onwards</p>
				<p style="margin: 5px 0;"><strong>🎤 Inauguration:</strong> 2:00 PM (IST)</p>
				<p style="margin: 5px 0;"><strong>🔗 Join Link:</strong> <a href="` + conferenceLink + `" style="color: #4CAF50; font-weight: bold;">Click here to join</a></p>
			</div>

			<h3 style="color: #2c3e50;">Important Instructions for Participants:</h3>
			<ul style="line-height: 1.8;">
				<li>At the end of this inaugural session, you will receive your link for the International Online Quiz.</li>
				<li>The quiz will be conducted between <strong>3:00 PM and 3:50 PM</strong> (your local time).</li>
				<li>Upon completion, you can view your responses, the correct answers, and your overall score.</li>
				<li>All participants will receive a <strong>Participation Certificate</strong>.</li>
				<li>The <strong>Top 10 scorers</strong> will be awarded <strong>Merit Certificates</strong>.</li>
				<li>The <strong>Winner</strong> will be selected based on the highest score and the time taken to complete the quiz (in case of a tie, faster completion time will be considered).</li>
			</ul>

			<p>This international event is not just a competition but also a platform to celebrate the spirit of cooperation and its role in creating a sustainable and equitable world.</p>

			<p>We look forward to your enthusiastic participation and presence in the inaugural session.</p>

			<p style="margin-top: 30px;">With warm regards,</p>
			<p><strong>Dr. U. Homiga</strong><br>
			Event Convenor,<br>
			Natesan Institute of Cooperative Management (NICM), Chennai</p>

			<p style="text-align: center; color: #4CAF50; font-style: italic; margin-top: 30px; font-size: 16px;">
				"Cooperatives: Building a Better World Together"
			</p>
		</div>
		`

		params := utils.SendEmailParams{
			ToEmail:  student.Email,
			ToName:   student.Name,
			Subject:  "Invitation: CoopQuest- An International Online Cooperative  Conclave",
			HTMLBody: htmlBody,
		}

		_, err := utils.SendEmail(params)
		if err != nil {
			log.Printf("Failed to resend email to %s: %v", student.Email, err)
		} else {
			sentCount++
		}

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return c.JSON(fiber.Map{
		"message": "Conference invitations resent successfully",
		"total":   len(students),
		"sent":    sentCount,
	})
}

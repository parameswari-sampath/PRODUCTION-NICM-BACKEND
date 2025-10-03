package handlers

import (
	"context"
	"encoding/json"
	"mcq-exam/db"
	"mcq-exam/utils"
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
	failedCount := 0
	var failedEmails []string

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

		// Prepare log entry
		status := "sent"
		var requestID, responseCode, responseMessage, errorMessage *string
		var zeptoResponseJSON *string

		if err != nil {
			status = "failed"
			errMsg := err.Error()
			errorMessage = &errMsg
			failedCount++
			failedEmails = append(failedEmails, student.Email)
		} else {
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

		// Log to database
		logQuery := `
			INSERT INTO email_logs (student_id, email, subject, status, request_id, response_code, response_message, zepto_response, error_message, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		`
		_, _ = db.Pool.Exec(context.Background(), logQuery, student.ID, student.Email, req.Subject, status, requestID, responseCode, responseMessage, zeptoResponseJSON, errorMessage)

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	response := fiber.Map{
		"message":      "Email sending completed",
		"total":        len(students),
		"sent":         sentCount,
		"failed":       failedCount,
	}

	if failedCount > 0 {
		response["failed_emails"] = failedEmails
	}

	return c.JSON(response)
}

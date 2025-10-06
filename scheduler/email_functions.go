package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"mcq-exam/db"
	"mcq-exam/utils"
	"os"
	"time"
)

// SendFirstEmailToAll sends conference email to all students with tracking pixel
func SendFirstEmailToAll() {
	log.Printf("[%s] EXECUTING: SendFirstEmailToAll - Sending conference emails", time.Now().Format(time.RFC3339))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all students
	query := `SELECT id, name, email FROM students ORDER BY id`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("ERROR: Failed to fetch students: %v", err)
		return
	}
	defer rows.Close()

	type Student struct {
		ID    int
		Name  string
		Email string
	}

	var students []Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.ID, &s.Name, &s.Email); err != nil {
			continue
		}
		students = append(students, s)
	}

	if len(students) == 0 {
		log.Printf("WARNING: No students found to send emails")
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://nicm.smart-mcq.com"
	}

	sentCount := 0
	for _, student := range students {
		// Generate conference token
		token := generateConferenceToken()

		// Store token in email_tracking
		insertQuery := `
			INSERT INTO email_tracking (student_id, email_type, conference_token, opened, created_at)
			VALUES ($1, 'first', $2, false, NOW())
			ON CONFLICT (student_id, email_type)
			DO UPDATE SET conference_token = $2, updated_at = NOW()
		`
		// Note: Need unique constraint on (student_id, email_type) - will add in migration
		_, err := db.Pool.Exec(context.Background(), insertQuery, student.ID, token)
		if err != nil {
			log.Printf("Failed to store token for student %d: %v", student.ID, err)
			continue
		}

		// Conference link with token
		conferenceLink := fmt.Sprintf("%s/live?token=%s", frontendURL, token)

		// Email body
		htmlBody := fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2>Conference Invitation</h2>
				<p>Dear %s,</p>
				<p>You are invited to attend our live conference session!</p>
				<p>Click the button below to join:</p>
				<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Join Conference Now</a></p>
				<p>This link is unique to you and can only be used once.</p>
				<p>Best regards,<br>SmartMCQ Team</p>
			</div>
		`, student.Name, conferenceLink)

		params := utils.SendEmailParams{
			ToEmail:  student.Email,
			ToName:   student.Name,
			Subject:  "Conference Invitation - SmartMCQ",
			HTMLBody: htmlBody,
		}

		_, err = utils.SendEmail(params)
		if err != nil {
			log.Printf("Failed to send email to %s: %v", student.Email, err)
		} else {
			sentCount++
		}

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[%s] COMPLETED: SendFirstEmailToAll - Sent %d/%d emails", time.Now().Format(time.RFC3339), sentCount, len(students))
}

// generateConferenceToken generates a secure random token
func generateConferenceToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SendSecondEmailToEligible sends test invitation to students who attended conference
func SendSecondEmailToEligible() {
	log.Printf("[%s] EXECUTING: SendSecondEmailToEligible - Sending test invitations", time.Now().Format(time.RFC3339))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get students who attended conference (verified token)
	query := `
		SELECT et.student_id, s.name, s.email, et.access_code
		FROM email_tracking et
		JOIN students s ON et.student_id = s.id
		WHERE et.email_type = 'first' AND et.conference_attended = true AND et.access_code IS NOT NULL
		ORDER BY et.conference_attended_at DESC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("ERROR: Failed to fetch eligible students: %v", err)
		return
	}
	defer rows.Close()

	type EligibleStudent struct {
		ID         int
		Name       string
		Email      string
		AccessCode string
	}

	var students []EligibleStudent
	for rows.Next() {
		var s EligibleStudent
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.AccessCode); err != nil {
			continue
		}
		students = append(students, s)
	}

	if len(students) == 0 {
		log.Printf("WARNING: No eligible students found (no one attended conference)")
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://nicm.smart-mcq.com"
	}

	sentCount := 0
	for _, student := range students {
		// Email body with access code
		htmlBody := fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2>Test Invitation - SmartMCQ</h2>
				<p>Dear %s,</p>
				<p>Thank you for attending the conference!</p>
				<p>You are now eligible to take the test. Your access code is:</p>
				<div style="background-color: #f4f4f4; padding: 20px; text-align: center; font-size: 32px; font-weight: bold; letter-spacing: 5px; margin: 20px 0;">
					%s
				</div>
				<p>Please use this code to start your test session.</p>
				<p><a href="%s/test" style="background-color: #2196F3; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Start Test</a></p>
				<p>Best regards,<br>SmartMCQ Team</p>
			</div>
		`, student.Name, student.AccessCode, frontendURL)

		params := utils.SendEmailParams{
			ToEmail:  student.Email,
			ToName:   student.Name,
			Subject:  "Test Invitation - Your Access Code",
			HTMLBody: htmlBody,
		}

		_, err := utils.SendEmail(params)
		if err != nil {
			log.Printf("Failed to send email to %s: %v", student.Email, err)
		} else {
			sentCount++
		}

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[%s] COMPLETED: SendSecondEmailToEligible - Sent %d/%d emails", time.Now().Format(time.RFC3339), sentCount, len(students))
}

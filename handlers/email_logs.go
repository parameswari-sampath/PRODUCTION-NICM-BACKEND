package handlers

import (
	"context"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

type EmailLog struct {
	ID              int       `json:"id"`
	StudentID       int       `json:"student_id"`
	Email           string    `json:"email"`
	Subject         string    `json:"subject"`
	Status          string    `json:"status"`
	RequestID       *string   `json:"request_id"`
	ResponseCode    *string   `json:"response_code"`
	ResponseMessage *string   `json:"response_message"`
	SentAt          time.Time `json:"sent_at"`
}

// GetEmailLogsHandler handles GET /api/mail/logs?status=sent
// Returns email logs filtered by status (default: sent)
// Special case: status=failed returns emails where request_id IS NULL
func GetEmailLogsHandler(c *fiber.Ctx) error {
	status := c.Query("status", "sent")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, student_id, email, subject, status, request_id, response_code, response_message, sent_at
		FROM email_logs
		WHERE status = $1
		ORDER BY id DESC
		LIMIT 1000
	`

	rows, err := db.Pool.Query(ctx, query, status)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch email logs"})
	}
	defer rows.Close()

	var logs []EmailLog
	for rows.Next() {
		var log EmailLog
		if err := rows.Scan(&log.ID, &log.StudentID, &log.Email, &log.Subject, &log.Status, &log.RequestID, &log.ResponseCode, &log.ResponseMessage, &log.SentAt); err != nil {
			continue
		}
		logs = append(logs, log)
	}

	return c.JSON(fiber.Map{
		"count": len(logs),
		"logs":  logs,
	})
}

package handlers

import (
	"context"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetEmailStatsHandler handles GET /api/mail/stats
// Returns total email addresses in students table
func GetEmailStatsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get total email addresses from students table
	var totalEmails int
	query := `SELECT COUNT(*) FROM students`
	if err := db.Pool.QueryRow(ctx, query).Scan(&totalEmails); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get email count"})
	}

	return c.JSON(fiber.Map{
		"total_emails": totalEmails,
	})
}

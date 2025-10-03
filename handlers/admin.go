package handlers

import (
	"mcq-exam/db"

	"github.com/gofiber/fiber/v2"
)

// ResetDatabaseHandler handles POST /api/admin/reset-db
// WARNING: This drops all tables and re-runs migrations
func ResetDatabaseHandler(c *fiber.Ctx) error {
	// Optional: Add authentication/authorization here
	// For now, it's open - SECURE THIS IN PRODUCTION!

	if err := db.ResetDatabase(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to reset database",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Database reset successfully",
		"status":  "All tables dropped and migrations re-run",
	})
}

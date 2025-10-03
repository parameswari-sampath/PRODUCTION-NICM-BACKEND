package handlers

import (
	"context"
	"mcq-exam/db"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SearchEmailHandler handles GET /api/mail/search?email=parames
// Supports partial search (finds emails containing the search term)
func SearchEmailHandler(c *fiber.Ctx) error {
	searchTerm := c.Query("email")

	if strings.TrimSpace(searchTerm) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email query parameter is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Partial search for emails in students table
	query := `SELECT email FROM students WHERE email ILIKE $1 ORDER BY email LIMIT 50`
	rows, err := db.Pool.Query(ctx, query, "%"+searchTerm+"%")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to search emails"})
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			continue
		}
		emails = append(emails, email)
	}

	if len(emails) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No emails found"})
	}

	return c.JSON(fiber.Map{
		"count":  len(emails),
		"emails": emails,
	})
}

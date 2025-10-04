package middleware

import (
	"context"
	"mcq-exam/db"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SessionMiddlewareResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ValidateSessionToken middleware validates the session token from Authorization header
func ValidateSessionToken(c *fiber.Ctx) error {
	// Get Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(SessionMiddlewareResponse{
			Success: false,
			Message: "Authorization header required",
		})
	}

	// Extract token from "Bearer <token>" format
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	if token == "" || token == authHeader {
		return c.Status(fiber.StatusUnauthorized).JSON(SessionMiddlewareResponse{
			Success: false,
			Message: "Invalid authorization format. Use: Bearer <token>",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate token exists in sessions table
	var studentID int
	var completed bool
	query := `
		SELECT student_id, completed
		FROM sessions
		WHERE session_token = $1
	`
	err := db.Pool.QueryRow(ctx, query, token).Scan(&studentID, &completed)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(SessionMiddlewareResponse{
			Success: false,
			Message: "Invalid or expired session token",
		})
	}

	// Check if test already completed
	if completed {
		return c.Status(fiber.StatusForbidden).JSON(SessionMiddlewareResponse{
			Success: false,
			Message: "Test already completed",
		})
	}

	// Store student_id and session_token in context for use in handlers
	c.Locals("student_id", studentID)
	c.Locals("session_token", token)

	return c.Next()
}

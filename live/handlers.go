package live

import (
	"context"
	"crypto/rand"
	"log"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

// generateAccessCode generates a 6-character alphanumeric code
func generateAccessCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	randomBytes := make([]byte, 6)
	rand.Read(randomBytes)
	for i := range code {
		code[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(code)
}

type VerifyTokenRequest struct {
	Token string `json:"token"`
}

type VerifyTokenResponse struct {
	Success  bool   `json:"success"`
	VideoURL string `json:"video_url,omitempty"`
	Message  string `json:"message,omitempty"`
}

// VerifyFirstMailTokenHandler handles POST /api/live/verify-first-mail
func VerifyFirstMailTokenHandler(c *fiber.Ctx) error {
	var req VerifyTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyTokenResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyTokenResponse{
			Success: false,
			Message: "Token is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Validate token exists in DB
	var studentId int
	var attended bool
	query := `
		SELECT student_id, conference_attended
		FROM email_tracking
		WHERE conference_token = $1 AND email_type = 'firstMail'
	`
	err := db.Pool.QueryRow(ctx, query, req.Token).Scan(&studentId, &attended)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(VerifyTokenResponse{
			Success: false,
			Message: "Invalid or expired token",
		})
	}

	// Step 2: Mark conference_attended as true and generate access code
	if !attended {
		// Generate 6-digit alphanumeric access code
		accessCode := generateAccessCode()

		updateQuery := `
			UPDATE email_tracking
			SET conference_attended = true, conference_attended_at = NOW(), access_code = $1, updated_at = NOW()
			WHERE conference_token = $2 AND email_type = 'firstMail'
		`
		_, err = db.Pool.Exec(ctx, updateQuery, accessCode, req.Token)
		if err != nil {
			log.Printf("Failed to mark attendance: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(VerifyTokenResponse{
				Success: false,
				Message: "Failed to update verification status",
			})
		}
	}

	// Step 3: Get YouTube URL from schedule table
	var videoURL string
	scheduleQuery := `SELECT video_url FROM event_schedule ORDER BY id DESC LIMIT 1`
	err = db.Pool.QueryRow(ctx, scheduleQuery).Scan(&videoURL)
	if err != nil || videoURL == "" {
		log.Printf("Failed to get video URL: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(VerifyTokenResponse{
			Success: false,
			Message: "Video URL not configured",
		})
	}

	// Step 4: Return success with YouTube URL
	return c.JSON(VerifyTokenResponse{
		Success:  true,
		VideoURL: videoURL,
		Message:  "Token verified successfully",
	})
}

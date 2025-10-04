package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

type VerifyTokenRequest struct {
	Token string `json:"token"`
}

type VerifyTokenResponse struct {
	Valid     bool   `json:"valid"`
	VideoURL  string `json:"video_url,omitempty"`
	Message   string `json:"message,omitempty"`
	StudentID int    `json:"student_id,omitempty"`
}

// VerifyConferenceTokenHandler handles POST /api/verify-token
// Verifies conference token and returns video URL
func VerifyConferenceTokenHandler(c *fiber.Ctx) error {
	var req VerifyTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyTokenResponse{
			Valid:   false,
			Message: "Invalid request body",
		})
	}

	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyTokenResponse{
			Valid:   false,
			Message: "Token is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find student by conference token
	var studentID int
	var attended bool
	query := `SELECT student_id, conference_attended FROM email_tracking WHERE conference_token = $1 AND email_type = 'first'`
	err := db.Pool.QueryRow(ctx, query, req.Token).Scan(&studentID, &attended)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(VerifyTokenResponse{
			Valid:   false,
			Message: "Invalid or expired token",
		})
	}

	// Get video URL from event schedule
	var videoURL string
	scheduleQuery := `SELECT video_url FROM event_schedule ORDER BY id DESC LIMIT 1`
	err = db.Pool.QueryRow(ctx, scheduleQuery).Scan(&videoURL)
	if err != nil || videoURL == "" {
		log.Printf("Failed to get video URL: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(VerifyTokenResponse{
			Valid:   false,
			Message: "Video URL not configured",
		})
	}

	// Mark as attended if not already
	if !attended {
		// Generate 6-character alphanumeric access code
		accessCode := generateAccessCode()
		updateQuery := `UPDATE email_tracking SET conference_attended = true, conference_attended_at = NOW(), access_code = $1, updated_at = NOW() WHERE conference_token = $2`
		_, err = db.Pool.Exec(context.Background(), updateQuery, accessCode, req.Token)
		if err != nil {
			log.Printf("Failed to mark attendance: %v", err)
		}
	}

	return c.JSON(VerifyTokenResponse{
		Valid:     true,
		VideoURL:  videoURL,
		StudentID: studentID,
		Message:   "Token verified successfully",
	})
}

// GenerateConferenceToken generates a secure random token
func GenerateConferenceToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

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

// generateSessionToken generates a unique session token
func generateSessionToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, 64)
	randomBytes := make([]byte, 64)
	rand.Read(randomBytes)
	for i := range token {
		token[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(token)
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

type VerifyOTPRequest struct {
	OTP string `json:"otp"`
}

type VerifyOTPResponse struct {
	Success      bool   `json:"success"`
	SessionToken string `json:"session_token,omitempty"`
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	Message      string `json:"message,omitempty"`
}

// VerifyOTPHandler handles POST /api/live/verify-otp
func VerifyOTPHandler(c *fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if req.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyOTPResponse{
			Success: false,
			Message: "OTP is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Verify OTP exists and get student details
	var studentID int
	var name, email string
	query := `
		SELECT et.student_id, s.name, s.email
		FROM email_tracking et
		JOIN students s ON et.student_id = s.id
		WHERE et.access_code = $1 AND et.email_type = 'firstMail' AND et.conference_attended = true
	`
	err := db.Pool.QueryRow(ctx, query, req.OTP).Scan(&studentID, &name, &email)
	if err != nil {
		log.Printf("OTP validation failed: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Already test completed or invalid OTP",
		})
	}

	// Step 2: Check if session already exists for this student
	var existingSessionID int
	checkSessionQuery := `SELECT id FROM sessions WHERE student_id = $1 LIMIT 1`
	err = db.Pool.QueryRow(ctx, checkSessionQuery, studentID).Scan(&existingSessionID)
	if err == nil {
		// Session exists
		return c.Status(fiber.StatusBadRequest).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Already test completed or invalid OTP",
		})
	}

	// Step 3: Validate test time (within 15 minutes of second_scheduled_time)
	var secondScheduledTime time.Time
	timeCheckQuery := `SELECT second_scheduled_time FROM event_schedule ORDER BY id DESC LIMIT 1`
	err = db.Pool.QueryRow(ctx, timeCheckQuery).Scan(&secondScheduledTime)
	if err != nil {
		log.Printf("Failed to get scheduled time: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Failed to validate test time",
		})
	}

	// Calculate time window: second_scheduled_time to second_scheduled_time + 15 minutes
	currentTime := time.Now()
	testEndTime := secondScheduledTime.Add(15 * time.Minute)

	if currentTime.Before(secondScheduledTime) {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Test has not started yet",
		})
	}

	if currentTime.After(testEndTime) {
		return c.Status(fiber.StatusBadRequest).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Test time expired",
		})
	}

	// Step 4: Generate session token and create new session
	sessionToken := generateSessionToken()

	createSessionQuery := `
		INSERT INTO sessions (student_id, session_token, access_code, started_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`
	var sessionID int
	err = db.Pool.QueryRow(ctx, createSessionQuery, studentID, sessionToken, req.OTP).Scan(&sessionID)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(VerifyOTPResponse{
			Success: false,
			Message: "Failed to create session",
		})
	}

	// Step 5: Return success with session token
	return c.JSON(VerifyOTPResponse{
		Success:      true,
		SessionToken: sessionToken,
		Email:        email,
		Name:         name,
		Message:      "OTP verified successfully",
	})
}

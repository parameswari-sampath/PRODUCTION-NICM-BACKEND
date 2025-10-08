package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TrackEmailOpenHandler handles GET /api/track-open?student_id=123&type=first
// Returns 1x1 transparent PNG and tracks email open + generates access code for first email
func TrackEmailOpenHandler(c *fiber.Ctx) error {
	studentIDStr := c.Query("student_id")
	emailType := c.Query("type") // 'first' or 'second'

	if studentIDStr == "" || emailType == "" {
		// Return pixel anyway but don't track
		return returnTransparentPixel(c)
	}

	var studentID int
	if _, err := fmt.Sscanf(studentIDStr, "%d", &studentID); err != nil {
		return returnTransparentPixel(c)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check if tracking record exists
	var trackingID int
	var opened bool
	checkQuery := `SELECT id, opened FROM email_tracking WHERE student_id = $1 AND email_type = $2`
	err := db.Pool.QueryRow(ctx, checkQuery, studentID, emailType).Scan(&trackingID, &opened)

	if err != nil {
		// Create new tracking record
		accessCode := ""
		if emailType == "first" {
			accessCode = generateAccessCode()
		}

		insertQuery := `
			INSERT INTO email_tracking (student_id, email_type, opened, opened_at, access_code)
			VALUES ($1, $2, true, NOW(), $3)
			RETURNING id
		`
		err = db.Pool.QueryRow(context.Background(), insertQuery, studentID, emailType, nullString(accessCode)).Scan(&trackingID)
		if err != nil {
			log.Printf("Failed to create email tracking: %v", err)
		}
	} else if !opened {
		// Update existing record to opened
		accessCode := ""
		if emailType == "first" {
			accessCode = generateAccessCode()
		}

		updateQuery := `UPDATE email_tracking SET opened = true, opened_at = NOW(), access_code = $1, updated_at = NOW() WHERE id = $2`
		_, _ = db.Pool.Exec(context.Background(), updateQuery, nullString(accessCode), trackingID)
	}

	return returnTransparentPixel(c)
}

// generateAccessCode generates a random 6-character alphanumeric code
func generateAccessCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	code := make([]byte, 6)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// nullString returns nil if string is empty, otherwise returns the string
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// returnTransparentPixel returns a 1x1 transparent PNG image
func returnTransparentPixel(c *fiber.Ctx) error {
	// 1x1 transparent PNG in base64
	transparentPNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	imgData, _ := base64.StdEncoding.DecodeString(transparentPNG)

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.Send(imgData)
}

// GetStudentsWhoOpenedHandler handles GET /api/tracking/opened-first
// Returns students who opened first email with their access codes
func GetStudentsWhoOpenedHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT et.student_id, s.name, s.email, et.access_code, et.opened_at
		FROM email_tracking et
		JOIN students s ON et.student_id = s.id
		WHERE et.email_type = 'first' AND et.opened = true
		ORDER BY et.opened_at DESC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch tracking data"})
	}
	defer rows.Close()

	type StudentTracking struct {
		StudentID  int       `json:"student_id"`
		Name       string    `json:"name"`
		Email      string    `json:"email"`
		AccessCode string    `json:"access_code"`
		OpenedAt   time.Time `json:"opened_at"`
	}

	var students []StudentTracking
	for rows.Next() {
		var st StudentTracking
		if err := rows.Scan(&st.StudentID, &st.Name, &st.Email, &st.AccessCode, &st.OpenedAt); err != nil {
			continue
		}
		students = append(students, st)
	}

	return c.JSON(fiber.Map{
		"count":    len(students),
		"students": students,
	})
}

// GetStudentsNotAttendedHandler handles GET /api/tracking/not-attended
// Returns students who did NOT attend the conference (fail-safe mechanism)
func GetStudentsNotAttendedHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT s.id, s.name, s.email, et.opened, et.opened_at, et.email_type
		FROM students s
		LEFT JOIN email_tracking et ON s.id = et.student_id
		WHERE et.conference_attended = false OR et.conference_attended IS NULL
		ORDER BY s.id ASC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch non-attendees"})
	}
	defer rows.Close()

	type NonAttendee struct {
		StudentID  int        `json:"student_id"`
		Name       string     `json:"name"`
		Email      string     `json:"email"`
		Opened     *bool      `json:"opened"`
		OpenedAt   *time.Time `json:"opened_at"`
		EmailType  *string    `json:"email_type"`
	}

	var students []NonAttendee
	for rows.Next() {
		var st NonAttendee
		if err := rows.Scan(&st.StudentID, &st.Name, &st.Email, &st.Opened, &st.OpenedAt, &st.EmailType); err != nil {
			continue
		}
		students = append(students, st)
	}

	return c.JSON(fiber.Map{
		"count":    len(students),
		"students": students,
	})
}

// GetStudentsNotStartedTestHandler handles GET /api/tracking/not-started-test
// Returns students who attended conference but did NOT start the test (no session created)
func GetStudentsNotStartedTestHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT et.student_id, s.name, s.email, et.access_code, et.conference_attended_at
		FROM email_tracking et
		JOIN students s ON et.student_id = s.id
		LEFT JOIN sessions sess ON sess.student_id = et.student_id
		WHERE et.email_type = 'firstMail'
		  AND et.conference_attended = true
		  AND et.access_code IS NOT NULL
		  AND sess.student_id IS NULL
		ORDER BY et.student_id ASC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch students"})
	}
	defer rows.Close()

	type StudentNotStarted struct {
		StudentID            int       `json:"student_id"`
		Name                 string    `json:"name"`
		Email                string    `json:"email"`
		AccessCode           string    `json:"access_code"`
		ConferenceAttendedAt time.Time `json:"conference_attended_at"`
	}

	var students []StudentNotStarted
	for rows.Next() {
		var st StudentNotStarted
		if err := rows.Scan(&st.StudentID, &st.Name, &st.Email, &st.AccessCode, &st.ConferenceAttendedAt); err != nil {
			continue
		}
		students = append(students, st)
	}

	return c.JSON(fiber.Map{
		"count":    len(students),
		"students": students,
	})
}

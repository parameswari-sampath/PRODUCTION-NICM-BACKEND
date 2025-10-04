package handlers

import (
	"context"
	"log"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CreateScheduleRequest struct {
	FirstFunction       string `json:"first_function"`
	FirstScheduledTime  string `json:"first_scheduled_time"`   // ISO8601 format
	SecondFunction      string `json:"second_function"`
	SecondScheduledTime string `json:"second_scheduled_time"` // ISO8601 format
	VideoURL            string `json:"video_url"`
}

// CreateEventScheduleHandler handles POST /api/event/schedule
// Creates a new event schedule with 2 timed function calls
func CreateEventScheduleHandler(c *fiber.Ctx) error {
	var req CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Load IST timezone
	istLocation, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("Failed to load IST timezone: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Server timezone error"})
	}

	// Parse times as IST (Asia/Kolkata)
	firstTime, err := time.ParseInLocation("2006-01-02T15:04:05", req.FirstScheduledTime, istLocation)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid first_scheduled_time format. Use YYYY-MM-DDTHH:MM:SS in IST (e.g., 2025-10-05T15:30:00)"})
	}

	secondTime, err := time.ParseInLocation("2006-01-02T15:04:05", req.SecondScheduledTime, istLocation)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid second_scheduled_time format. Use YYYY-MM-DDTHH:MM:SS in IST (e.g., 2025-10-05T18:00:00)"})
	}

	// Validate second time is after first time
	if secondTime.Before(firstTime) || secondTime.Equal(firstTime) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "second_scheduled_time must be after first_scheduled_time"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Validate video URL
	if req.VideoURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "video_url is required"})
	}

	// Insert schedule
	query := `
		INSERT INTO event_schedule (first_function, first_scheduled_time, second_function, second_scheduled_time, video_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var scheduleID int
	err = db.Pool.QueryRow(ctx, query, req.FirstFunction, firstTime, req.SecondFunction, secondTime, req.VideoURL).Scan(&scheduleID)
	if err != nil {
		log.Printf("Failed to create schedule: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create schedule"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":              "Schedule created successfully",
		"schedule_id":          scheduleID,
		"first_function":       req.FirstFunction,
		"first_scheduled_time": firstTime,
		"second_function":      req.SecondFunction,
		"second_scheduled_time": secondTime,
		"video_url":            req.VideoURL,
	})
}

// GetEventScheduleHandler handles GET /api/event/schedule
// Returns the current event schedule
func GetEventScheduleHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, first_function, first_scheduled_time, first_executed, first_executed_at,
		       second_function, second_scheduled_time, second_executed, second_executed_at,
		       created_at
		FROM event_schedule
		ORDER BY id DESC
		LIMIT 1
	`

	var schedule struct {
		ID                  int        `json:"id"`
		FirstFunction       string     `json:"first_function"`
		FirstScheduledTime  time.Time  `json:"first_scheduled_time"`
		FirstExecuted       bool       `json:"first_executed"`
		FirstExecutedAt     *time.Time `json:"first_executed_at"`
		SecondFunction      string     `json:"second_function"`
		SecondScheduledTime time.Time  `json:"second_scheduled_time"`
		SecondExecuted      bool       `json:"second_executed"`
		SecondExecutedAt    *time.Time `json:"second_executed_at"`
		CreatedAt           time.Time  `json:"created_at"`
	}

	err := db.Pool.QueryRow(ctx, query).Scan(
		&schedule.ID,
		&schedule.FirstFunction,
		&schedule.FirstScheduledTime,
		&schedule.FirstExecuted,
		&schedule.FirstExecutedAt,
		&schedule.SecondFunction,
		&schedule.SecondScheduledTime,
		&schedule.SecondExecuted,
		&schedule.SecondExecutedAt,
		&schedule.CreatedAt,
	)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No schedule found"})
	}

	return c.JSON(schedule)
}

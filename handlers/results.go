package handlers

import (
	"context"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetAllResultsHandler handles GET /api/results
// Returns all completed test results ranked by score (DESC) then time (ASC)
func GetAllResultsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT s.email, sess.score, sess.total_time_taken_seconds
		FROM sessions sess
		JOIN students s ON sess.student_id = s.id
		WHERE sess.completed = true
		ORDER BY sess.score DESC, sess.total_time_taken_seconds ASC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch results"})
	}
	defer rows.Close()

	type StudentResult struct {
		Email               string `json:"email"`
		Score               int    `json:"score"`
		TotalTimeTakenSeconds int    `json:"total_time_taken_seconds"`
	}

	var results []StudentResult
	for rows.Next() {
		var result StudentResult
		if err := rows.Scan(&result.Email, &result.Score, &result.TotalTimeTakenSeconds); err != nil {
			continue
		}
		results = append(results, result)
	}

	return c.JSON(fiber.Map{
		"count":   len(results),
		"results": results,
	})
}

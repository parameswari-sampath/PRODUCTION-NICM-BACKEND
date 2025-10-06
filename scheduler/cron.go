package scheduler

import (
	"context"
	"log"
	"mcq-exam/db"
	"time"
)

// StartScheduler starts the cron job that checks for scheduled functions every minute
func StartScheduler() {
	log.Println("Starting event scheduler (checks every minute)...")

	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			checkAndExecuteSchedules()
		}
	}()

	// Also check immediately on start
	go checkAndExecuteSchedules()
}

// checkAndExecuteSchedules checks for pending scheduled functions and executes them
func checkAndExecuteSchedules() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use UTC for comparison (database stores times in UTC)
	now := time.Now().UTC()

	// Check for first function to execute
	query := `
		SELECT id, first_function
		FROM event_schedule
		WHERE first_executed = false
		  AND first_scheduled_time <= $1
		ORDER BY first_scheduled_time ASC
		LIMIT 1
	`

	var scheduleID int
	var functionName string
	err := db.Pool.QueryRow(ctx, query, now).Scan(&scheduleID, &functionName)
	if err == nil {
		log.Printf("Found scheduled first function: %s (schedule_id: %d)", functionName, scheduleID)

		// Execute function
		success := ExecuteFunction(functionName)

		if success {
			// Mark as executed
			updateQuery := `UPDATE event_schedule SET first_executed = true, first_executed_at = NOW() WHERE id = $1`
			_, _ = db.Pool.Exec(context.Background(), updateQuery, scheduleID)
			log.Printf("Marked first function as executed (schedule_id: %d)", scheduleID)
		}
	}

	// Check for second function to execute
	query = `
		SELECT id, second_function
		FROM event_schedule
		WHERE second_executed = false
		  AND second_scheduled_time <= $1
		  AND first_executed = true
		ORDER BY second_scheduled_time ASC
		LIMIT 1
	`

	err = db.Pool.QueryRow(ctx, query, now).Scan(&scheduleID, &functionName)
	if err == nil {
		log.Printf("Found scheduled second function: %s (schedule_id: %d)", functionName, scheduleID)

		// Execute function
		success := ExecuteFunction(functionName)

		if success {
			// Mark as executed
			updateQuery := `UPDATE event_schedule SET second_executed = true, second_executed_at = NOW() WHERE id = $1`
			_, _ = db.Pool.Exec(context.Background(), updateQuery, scheduleID)
			log.Printf("Marked second function as executed (schedule_id: %d)", scheduleID)
		}
	}
}

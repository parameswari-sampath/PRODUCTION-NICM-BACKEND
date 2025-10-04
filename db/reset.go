package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
)

// ResetDatabase drops all tables and re-runs migrations
func ResetDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Drop all tables (CASCADE will handle indexes and constraints)
	dropQuery := `
		DROP TABLE IF EXISTS answers CASCADE;
		DROP TABLE IF EXISTS sessions CASCADE;
		DROP TABLE IF EXISTS email_tracking CASCADE;
		DROP TABLE IF EXISTS event_schedule CASCADE;
		DROP TABLE IF EXISTS email_logs CASCADE;
		DROP TABLE IF EXISTS students CASCADE;
		DROP TABLE IF EXISTS schema_migrations CASCADE;
	`

	if _, err := Pool.Exec(ctx, dropQuery); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	log.Println("All tables dropped successfully")

	// Re-run migrations
	if err := RunMigrations(""); err != nil {
		// Ignore ErrNoChange as migrations might already be applied
		if err != migrate.ErrNoChange {
			return fmt.Errorf("failed to run migrations after reset: %w", err)
		}
	}

	log.Println("Database reset completed successfully")
	return nil
}

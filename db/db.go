package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var Pool *pgxpool.Pool

// InitDB initializes the database connection pool optimized for high traffic
func InitDB() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// Parse and configure pool settings for 2k req/sec peak load
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("unable to parse DATABASE_URL: %w", err)
	}

	// Connection pool settings optimized for 2 vCPU + MCQ exam load
	config.MaxConns = 25                          // 2-3x vCPUs, handles 800 writes/sec peak
	config.MinConns = 5                           // Keep warm connections ready
	config.MaxConnLifetime = 5 * time.Minute      // Recycle connections
	config.MaxConnIdleTime = 2 * time.Minute      // Close idle connections
	config.HealthCheckPeriod = 1 * time.Minute    // Periodic health checks
	config.ConnConfig.ConnectTimeout = 3 * time.Second

	// Create pool
	Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	log.Printf("Database connection pool initialized (max: %d, min: %d)", config.MaxConns, config.MinConns)
	return nil
}

// Close closes the database connection pool
func Close() {
	if Pool != nil {
		Pool.Close()
		log.Println("Database connection pool closed")
	}
}

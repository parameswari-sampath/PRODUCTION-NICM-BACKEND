package main

import (
	"log"
	"mcq-exam/db"
	"mcq-exam/handlers"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Initialize database
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	databaseURL := os.Getenv("DATABASE_URL")
	if err := db.RunMigrations(databaseURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "MCQ Exam API",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "*",
	}))

	// Routes
	api := app.Group("/api")

	// Student endpoints
	students := api.Group("/students")
	students.Post("/bulk", handlers.BulkCreateStudentsFiber)
	students.Get("/", handlers.GetAllStudentsFiber)
	students.Post("/", handlers.CreateStudentFiber)
	students.Get("/:id", handlers.GetStudentFiber)
	students.Put("/:id", handlers.UpdateStudentFiber)
	students.Delete("/:id", handlers.DeleteStudentFiber)

	// Admin endpoints
	admin := api.Group("/admin")
	admin.Post("/reset-db", handlers.ResetDatabaseHandler)

	// Mail endpoints
	mail := api.Group("/mail")
	mail.Post("/send", handlers.SendEmailHandler)
	mail.Post("/send-all", handlers.SendAllEmailsHandler)
	mail.Get("/stats", handlers.GetEmailStatsHandler)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")
		app.Shutdown()
	}()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

package main

import (
	"log"
	"mcq-exam/db"
	"mcq-exam/handlers"
	"mcq-exam/live"
	"mcq-exam/scheduler"
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

	// Start scheduler
	scheduler.StartScheduler()

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
	mail.Post("/resend-conference", handlers.ResendConferenceInvitationHandler)
	mail.Post("/resend-test-invitation", handlers.ResendTestInvitationHandler)
	mail.Get("/stats", handlers.GetEmailStatsHandler)
	mail.Get("/search", handlers.SearchEmailHandler)
	mail.Get("/logs", handlers.GetEmailLogsHandler)

	// Webhook endpoints
	webhooks := api.Group("/webhooks")
	webhooks.Post("/zeptomail", handlers.ZeptoMailWebhookHandler)

	// Event scheduling endpoints
	event := api.Group("/event")
	event.Post("/schedule", handlers.CreateEventScheduleHandler)
	event.Get("/schedule", handlers.GetEventScheduleHandler)

	// Email tracking endpoints
	api.Get("/track-open", handlers.TrackEmailOpenHandler)
	tracking := api.Group("/tracking")
	tracking.Get("/opened-first", handlers.GetStudentsWhoOpenedHandler)
	tracking.Get("/not-attended", handlers.GetStudentsNotAttendedHandler)
	tracking.Get("/not-started-test", handlers.GetStudentsNotStartedTestHandler)

	// Conference token verification
	api.Post("/verify-token", handlers.VerifyConferenceTokenHandler)

	// Live endpoints
	liveAPI := api.Group("/live")
	liveAPI.Post("/verify-first-mail", live.VerifyFirstMailTokenHandler)
	liveAPI.Post("/verify-otp", live.VerifyOTPHandler)
	liveAPI.Post("/start-session", live.StartSessionHandler)
	liveAPI.Post("/submit-answer", live.SubmitAnswerHandler)
	liveAPI.Post("/end-session", live.EndSessionHandler)
	liveAPI.Post("/result", live.GetResultHandler)

	// Leaderboard endpoints
	leaderboard := api.Group("/leaderboard")
	leaderboard.Get("/overall", handlers.GetOverallLeaderboardHandler)
	leaderboard.Get("/section/:section_id", handlers.GetSectionLeaderboardHandler)
	leaderboard.Get("/user-sections", handlers.GetUserSectionRanksHandler)

	// Results endpoints
	api.Get("/results", handlers.GetAllResultsHandler)

	// Load test endpoints (isolated)
	loadTest := api.Group("/load-test")
	loadTest.Post("/individual", handlers.LoadTestIndividualHandler)
	loadTest.Post("/batch", handlers.LoadTestBatchHandler)
	loadTest.Get("/metrics/individual", handlers.GetIndividualMetricsHandler)
	loadTest.Get("/metrics/batch", handlers.GetBatchMetricsHandler)
	loadTest.Post("/metrics/reset", handlers.ResetLoadTestMetricsHandler)
	loadTest.Delete("/cleanup", handlers.CleanupLoadTestDataHandler)
	loadTest.Post("/results/save", handlers.SaveTestResultsHandler)
	loadTest.Get("/results", handlers.GetAllTestResultsHandler)

	// Serve static files
	app.Static("/", "./public")

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

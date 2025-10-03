package handlers

import (
	"context"
	"fmt"
	"mcq-exam/db"
	"mcq-exam/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// CreateStudentFiber handles POST /api/students
func CreateStudentFiber(c *fiber.Ctx) error {
	var req models.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and email are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var student models.Student
	query := `
		INSERT INTO students (name, email, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, name, email, created_at, updated_at
	`
	err := db.Pool.QueryRow(ctx, query, req.Name, req.Email).Scan(
		&student.ID,
		&student.Name,
		&student.Email,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create student"})
	}

	return c.Status(fiber.StatusCreated).JSON(student)
}

// GetStudentFiber handles GET /api/students/:id
func GetStudentFiber(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var student models.Student
	query := `SELECT id, name, email, created_at, updated_at FROM students WHERE id = $1`
	err = db.Pool.QueryRow(ctx, query, id).Scan(
		&student.ID,
		&student.Name,
		&student.Email,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Student not found"})
	}

	return c.JSON(student)
}

// GetAllStudentsFiber handles GET /api/students?limit=10&offset=0
func GetAllStudentsFiber(c *fiber.Ctx) error {
	// Get limit and offset from query params (default: limit=100, offset=0)
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	// Validate limit
	if limit < 1 || limit > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Limit must be between 1 and 1000"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM students`
	if err := db.Pool.QueryRow(ctx, countQuery).Scan(&totalCount); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get total count"})
	}

	// Get paginated results
	query := `SELECT id, name, email, created_at, updated_at FROM students ORDER BY id LIMIT $1 OFFSET $2`
	rows, err := db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch students"})
	}
	defer rows.Close()

	students := []models.Student{}
	for rows.Next() {
		var student models.Student
		if err := rows.Scan(&student.ID, &student.Name, &student.Email, &student.CreatedAt, &student.UpdatedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to scan student"})
		}
		students = append(students, student)
	}

	return c.JSON(fiber.Map{
		"students": students,
		"total":    totalCount,
		"limit":    limit,
		"offset":   offset,
		"count":    len(students),
	})
}

// UpdateStudentFiber handles PUT /api/students/:id
func UpdateStudentFiber(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	var req models.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and email are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var student models.Student
	query := `
		UPDATE students
		SET name = $1, email = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, email, created_at, updated_at
	`
	err = db.Pool.QueryRow(ctx, query, req.Name, req.Email, id).Scan(
		&student.ID,
		&student.Name,
		&student.Email,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Student not found"})
	}

	return c.JSON(student)
}

// DeleteStudentFiber handles DELETE /api/students/:id
func DeleteStudentFiber(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM students WHERE id = $1`
	result, err := db.Pool.Exec(ctx, query, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete student"})
	}

	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Student not found"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// BulkCreateStudentsFiber handles POST /api/students/bulk
func BulkCreateStudentsFiber(c *fiber.Ctx) error {
	var req struct {
		Students []models.CreateStudentRequest `json:"students"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(req.Students) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No students provided"})
	}

	// Validate max limit for bulk upload
	if len(req.Students) > 2000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Maximum 2000 students allowed per bulk upload"})
	}

	// Validate all students
	for i, student := range req.Students {
		if strings.TrimSpace(student.Name) == "" || strings.TrimSpace(student.Email) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Student at index %d has invalid name or email", i)})
		}
	}

	// Deduplicate emails within the request
	emailMap := make(map[string]models.CreateStudentRequest)
	var duplicatesInRequest []string

	for _, student := range req.Students {
		email := strings.ToLower(strings.TrimSpace(student.Email))
		if _, exists := emailMap[email]; exists {
			duplicatesInRequest = append(duplicatesInRequest, fmt.Sprintf("%s (%s)", student.Name, student.Email))
		} else {
			emailMap[email] = student
		}
	}

	// Convert map back to slice for insertion
	uniqueStudents := make([]models.CreateStudentRequest, 0, len(emailMap))
	for _, student := range emailMap {
		uniqueStudents = append(uniqueStudents, student)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use batch insert for performance with ON CONFLICT DO NOTHING
	batch := &pgx.Batch{}
	for _, student := range uniqueStudents {
		query := `INSERT INTO students (name, email, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (email) DO NOTHING`
		batch.Queue(query, student.Name, student.Email)
	}

	results := db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	// Execute all batched queries
	successCount := 0
	skippedCount := 0
	for i := range uniqueStudents {
		cmdTag, err := results.Exec()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to insert student at index %d: %s", i, err.Error())})
		}
		// Check rows affected - 0 means skipped due to conflict
		if cmdTag.RowsAffected() == 0 {
			skippedCount++
		} else {
			successCount++
		}
	}

	// Prepare response
	response := fiber.Map{
		"message":                  "Students processed successfully",
		"total_received":           len(req.Students),
		"duplicates_in_request":    len(duplicatesInRequest),
		"unique_emails":            len(uniqueStudents),
		"successfully_inserted":    successCount,
		"already_exists_skipped":   skippedCount,
	}

	if len(duplicatesInRequest) > 0 {
		response["duplicate_emails_in_request"] = duplicatesInRequest
	}

	if skippedCount > 0 || len(duplicatesInRequest) > 0 {
		return c.Status(fiber.StatusPartialContent).JSON(response)
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

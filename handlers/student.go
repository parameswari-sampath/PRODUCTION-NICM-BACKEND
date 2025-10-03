package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"mcq-exam/db"
	"mcq-exam/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateStudent handles POST /api/students
func CreateStudent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
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
		http.Error(w, "Failed to create student", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(student)
}

// GetStudent handles GET /api/students/{id}
func GetStudent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/students/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
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
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// GetAllStudents handles GET /api/students
func GetAllStudents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	query := `SELECT id, name, email, created_at, updated_at FROM students ORDER BY id`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	students := []models.Student{}
	for rows.Next() {
		var student models.Student
		if err := rows.Scan(&student.ID, &student.Name, &student.Email, &student.CreatedAt, &student.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan student", http.StatusInternalServerError)
			return
		}
		students = append(students, student)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// UpdateStudent handles PUT /api/students/{id}
func UpdateStudent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/students/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
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
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// DeleteStudent handles DELETE /api/students/{id}
func DeleteStudent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/students/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM students WHERE id = $1`
	result, err := db.Pool.Exec(ctx, query, id)
	if err != nil {
		http.Error(w, "Failed to delete student", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkCreateStudents handles POST /api/students/bulk
func BulkCreateStudents(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Students []models.CreateStudentRequest `json:"students"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Students) == 0 {
		http.Error(w, "No students provided", http.StatusBadRequest)
		return
	}

	// Validate all students
	for i, student := range req.Students {
		if strings.TrimSpace(student.Name) == "" || strings.TrimSpace(student.Email) == "" {
			http.Error(w, fmt.Sprintf("Student at index %d has invalid name or email", i), http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Use batch insert for performance
	batch := &pgx.Batch{}
	for _, student := range req.Students {
		query := `INSERT INTO students (name, email, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`
		batch.Queue(query, student.Name, student.Email)
	}

	results := db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	// Execute all batched queries
	for range req.Students {
		if _, err := results.Exec(); err != nil {
			http.Error(w, "Failed to insert students", http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"message": "Students created successfully",
		"count":   len(req.Students),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

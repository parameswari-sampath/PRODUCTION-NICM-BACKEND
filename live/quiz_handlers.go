package live

import (
	"context"
	"encoding/json"
	"log"
	"mcq-exam/db"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SubmitAnswerRequest struct {
	SessionToken        string `json:"session_token"`
	QuestionID          int    `json:"question_id"`
	SelectedOptionIndex int    `json:"selected_option_index"`
	IsCorrect           bool   `json:"is_correct"`
	TimeTakenSeconds    int    `json:"time_taken_seconds"`
}

type SubmitAnswerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type EndSessionRequest struct {
	SessionToken string `json:"session_token"`
}

type EndSessionResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	Score              *int   `json:"score,omitempty"`
	TotalTimeTaken     *int   `json:"total_time_taken_seconds,omitempty"`
	TotalQuestions     *int   `json:"total_questions_answered,omitempty"`
}

type GetResultRequest struct {
	Email string `json:"email"`
}

type StudentInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SessionInfo struct {
	Score                  int  `json:"score"`
	TotalTimeTakenSeconds  int  `json:"total_time_taken_seconds"`
	TotalQuestionsAnswered int  `json:"total_questions_answered"`
	Completed              bool `json:"completed"`
}

type QuestionResult struct {
	ID               int     `json:"id"`
	Question         string  `json:"question"`
	Description      string  `json:"description"`
	Options          []string `json:"options"`
	CorrectAnswer    int     `json:"correctAnswer"`
	SelectedAnswer   *int    `json:"selected_answer"`
	IsCorrect        *bool   `json:"is_correct"`
	TimeTakenSeconds *int    `json:"time_taken_seconds"`
}

type SectionResult struct {
	ID        int              `json:"id"`
	Name      string           `json:"name"`
	TimeLimit int              `json:"time_limit"`
	Questions []QuestionResult `json:"questions"`
}

type GetResultResponse struct {
	Success  bool            `json:"success"`
	Message  string          `json:"message,omitempty"`
	Student  *StudentInfo    `json:"student,omitempty"`
	Session  *SessionInfo    `json:"session,omitempty"`
	Sections []SectionResult `json:"sections,omitempty"`
}

// SubmitAnswerHandler handles POST /api/live/submit-answer
func SubmitAnswerHandler(c *fiber.Ctx) error {
	var req SubmitAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	// Validate required fields
	if req.SessionToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Session token is required",
		})
	}

	if req.QuestionID <= 0 || req.QuestionID > 120 {
		return c.Status(fiber.StatusBadRequest).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Invalid question ID (must be 1-120)",
		})
	}

	if req.SelectedOptionIndex < 0 || req.SelectedOptionIndex > 3 {
		return c.Status(fiber.StatusBadRequest).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Invalid option index (must be 0-3)",
		})
	}

	if req.TimeTakenSeconds < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Invalid time taken",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Validate session token and get session_id
	var sessionID int
	var completed bool
	sessionQuery := `
		SELECT id, completed
		FROM sessions
		WHERE session_token = $1
	`
	err := db.Pool.QueryRow(ctx, sessionQuery, req.SessionToken).Scan(&sessionID, &completed)
	if err != nil {
		log.Printf("Session validation failed: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Invalid session token",
		})
	}

	// Step 2: Check if test is already completed
	if completed {
		return c.Status(fiber.StatusForbidden).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Test already completed",
		})
	}

	// Step 3: Check if answer already submitted for this question
	var existingAnswerID int
	checkQuery := `SELECT id FROM answers WHERE session_id = $1 AND question_id = $2 LIMIT 1`
	err = db.Pool.QueryRow(ctx, checkQuery, sessionID, req.QuestionID).Scan(&existingAnswerID)
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Answer already submitted for this question",
		})
	}

	// Step 4: Insert answer into database
	insertQuery := `
		INSERT INTO answers (session_id, question_id, selected_option_index, is_correct, time_taken_seconds)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = db.Pool.Exec(ctx, insertQuery, sessionID, req.QuestionID, req.SelectedOptionIndex, req.IsCorrect, req.TimeTakenSeconds)
	if err != nil {
		log.Printf("Failed to insert answer: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(SubmitAnswerResponse{
			Success: false,
			Message: "Failed to save answer",
		})
	}

	// Step 5: Return success
	return c.Status(fiber.StatusCreated).JSON(SubmitAnswerResponse{
		Success: true,
		Message: "Answer submitted successfully",
	})
}

// EndSessionHandler handles POST /api/live/end-session
func EndSessionHandler(c *fiber.Ctx) error {
	var req EndSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(EndSessionResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	// Validate required fields
	if req.SessionToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(EndSessionResponse{
			Success: false,
			Message: "Session token is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Validate session token and get session_id and started_at
	var sessionID int
	var completed bool
	var startedAt time.Time
	sessionQuery := `
		SELECT id, completed, started_at
		FROM sessions
		WHERE session_token = $1
	`
	err := db.Pool.QueryRow(ctx, sessionQuery, req.SessionToken).Scan(&sessionID, &completed, &startedAt)
	if err != nil {
		log.Printf("Session validation failed: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(EndSessionResponse{
			Success: false,
			Message: "Invalid session token",
		})
	}

	// Step 2: Check if test is already completed
	if completed {
		return c.Status(fiber.StatusConflict).JSON(EndSessionResponse{
			Success: false,
			Message: "Test already completed",
		})
	}

	// Step 3: Calculate total score (count of correct answers)
	var score int
	scoreQuery := `
		SELECT COUNT(*)
		FROM answers
		WHERE session_id = $1 AND is_correct = true
	`
	err = db.Pool.QueryRow(ctx, scoreQuery, sessionID).Scan(&score)
	if err != nil {
		log.Printf("Failed to calculate score: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(EndSessionResponse{
			Success: false,
			Message: "Failed to calculate score",
		})
	}

	// Step 4: Calculate total time taken (sum of all answer times)
	var totalTimeTaken int
	timeQuery := `
		SELECT COALESCE(SUM(time_taken_seconds), 0)
		FROM answers
		WHERE session_id = $1
	`
	err = db.Pool.QueryRow(ctx, timeQuery, sessionID).Scan(&totalTimeTaken)
	if err != nil {
		log.Printf("Failed to calculate total time: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(EndSessionResponse{
			Success: false,
			Message: "Failed to calculate total time",
		})
	}

	// Step 5: Get total questions answered
	var totalQuestions int
	countQuery := `
		SELECT COUNT(*)
		FROM answers
		WHERE session_id = $1
	`
	err = db.Pool.QueryRow(ctx, countQuery, sessionID).Scan(&totalQuestions)
	if err != nil {
		log.Printf("Failed to count questions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(EndSessionResponse{
			Success: false,
			Message: "Failed to count questions answered",
		})
	}

	// Step 6: Update session with completion data
	updateQuery := `
		UPDATE sessions
		SET completed = true,
		    completed_at = NOW(),
		    score = $1,
		    total_time_taken_seconds = $2,
		    updated_at = NOW()
		WHERE id = $3
	`
	_, err = db.Pool.Exec(ctx, updateQuery, score, totalTimeTaken, sessionID)
	if err != nil {
		log.Printf("Failed to update session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(EndSessionResponse{
			Success: false,
			Message: "Failed to end session",
		})
	}

	// Step 7: Return success with results
	return c.Status(fiber.StatusOK).JSON(EndSessionResponse{
		Success:        true,
		Message:        "Test completed successfully",
		Score:          &score,
		TotalTimeTaken: &totalTimeTaken,
		TotalQuestions: &totalQuestions,
	})
}

// GetResultHandler handles POST /api/live/result
func GetResultHandler(c *fiber.Ctx) error {
	var req GetResultRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(GetResultResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	// Validate email
	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(GetResultResponse{
			Success: false,
			Message: "Email is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Get student by email
	var studentID int
	var studentName string
	studentQuery := `
		SELECT id, name
		FROM students
		WHERE email = $1
	`
	err := db.Pool.QueryRow(ctx, studentQuery, req.Email).Scan(&studentID, &studentName)
	if err != nil {
		log.Printf("Student not found: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(GetResultResponse{
			Success: false,
			Message: "Student not found",
		})
	}

	// Step 2: Get session by student_id
	var sessionID int
	var score, totalTimeTaken int
	var completed bool
	sessionQuery := `
		SELECT id, COALESCE(score, 0), COALESCE(total_time_taken_seconds, 0), completed
		FROM sessions
		WHERE student_id = $1
	`
	err = db.Pool.QueryRow(ctx, sessionQuery, studentID).Scan(&sessionID, &score, &totalTimeTaken, &completed)
	if err != nil {
		log.Printf("Session not found: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(GetResultResponse{
			Success: false,
			Message: "No session found for this student",
		})
	}

	// Step 3: Get all answers for this session
	answersQuery := `
		SELECT question_id, selected_option_index, is_correct, time_taken_seconds
		FROM answers
		WHERE session_id = $1
	`
	rows, err := db.Pool.Query(ctx, answersQuery, sessionID)
	if err != nil {
		log.Printf("Failed to fetch answers: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(GetResultResponse{
			Success: false,
			Message: "Failed to fetch answers",
		})
	}
	defer rows.Close()

	// Create map of answers by question_id
	answersMap := make(map[int]struct {
		SelectedOption int
		IsCorrect      bool
		TimeTaken      int
	})
	answeredCount := 0

	for rows.Next() {
		var questionID, selectedOption, timeTaken int
		var isCorrect bool
		if err := rows.Scan(&questionID, &selectedOption, &isCorrect, &timeTaken); err != nil {
			log.Printf("Failed to scan answer: %v", err)
			continue
		}
		answersMap[questionID] = struct {
			SelectedOption int
			IsCorrect      bool
			TimeTaken      int
		}{selectedOption, isCorrect, timeTaken}
		answeredCount++
	}

	// Step 4: Load questions from JSON file
	questionsFile, err := os.ReadFile("questions_with_timer.json")
	if err != nil {
		log.Printf("Failed to read questions file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(GetResultResponse{
			Success: false,
			Message: "Failed to load questions",
		})
	}

	// Define structure for parsing JSON
	type JSONQuestion struct {
		ID          int      `json:"id"`
		Question    string   `json:"question"`
		Description string   `json:"description"`
		Options     []string `json:"options"`
		CorrectAnswer int    `json:"correctAnswer"`
	}
	type JSONSection struct {
		ID        int            `json:"id"`
		Name      string         `json:"name"`
		TimeLimit int            `json:"time_limit"`
		Questions []JSONQuestion `json:"questions"`
	}
	var jsonSections []JSONSection

	if err := json.Unmarshal(questionsFile, &jsonSections); err != nil {
		log.Printf("Failed to parse questions JSON: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(GetResultResponse{
			Success: false,
			Message: "Failed to parse questions",
		})
	}

	// Step 5: Merge answers into questions
	var sections []SectionResult
	for _, jsonSection := range jsonSections {
		section := SectionResult{
			ID:        jsonSection.ID,
			Name:      jsonSection.Name,
			TimeLimit: jsonSection.TimeLimit,
			Questions: make([]QuestionResult, 0),
		}

		for _, jsonQ := range jsonSection.Questions {
			question := QuestionResult{
				ID:            jsonQ.ID,
				Question:      jsonQ.Question,
				Description:   jsonQ.Description,
				Options:       jsonQ.Options,
				CorrectAnswer: jsonQ.CorrectAnswer,
			}

			// Check if student answered this question
			if answer, exists := answersMap[jsonQ.ID]; exists {
				question.SelectedAnswer = &answer.SelectedOption
				question.IsCorrect = &answer.IsCorrect
				question.TimeTakenSeconds = &answer.TimeTaken
			} else {
				// Not answered - leave as null
				question.SelectedAnswer = nil
				question.IsCorrect = nil
				question.TimeTakenSeconds = nil
			}

			section.Questions = append(section.Questions, question)
		}

		sections = append(sections, section)
	}

	// Step 6: Return complete result
	return c.Status(fiber.StatusOK).JSON(GetResultResponse{
		Success: true,
		Student: &StudentInfo{
			Name:  studentName,
			Email: req.Email,
		},
		Session: &SessionInfo{
			Score:                  score,
			TotalTimeTakenSeconds:  totalTimeTaken,
			TotalQuestionsAnswered: answeredCount,
			Completed:              completed,
		},
		Sections: sections,
	})
}

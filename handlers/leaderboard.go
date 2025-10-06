package handlers

import (
	"context"
	"encoding/json"
	"log"
	"mcq-exam/db"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ============================================
// OVERALL LEADERBOARD
// ============================================

type LeaderboardEntry struct {
	Rank                  int    `json:"rank"`
	StudentID             int    `json:"student_id"`
	Name                  string `json:"name"`
	Email                 string `json:"email"`
	Score                 int    `json:"score"`
	TotalTimeTakenSeconds int    `json:"total_time_taken_seconds"`
}

type OverallLeaderboardResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
	Total   int                `json:"total,omitempty"`
	Data    []LeaderboardEntry `json:"data,omitempty"`
}

// GetOverallLeaderboardHandler handles GET /api/leaderboard/overall
func GetOverallLeaderboardHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query to get top 100 students ordered by score DESC, then time ASC
	query := `
		SELECT
			s.id,
			s.name,
			s.email,
			COALESCE(sess.score, 0) as score,
			COALESCE(sess.total_time_taken_seconds, 0) as total_time_taken_seconds
		FROM students s
		INNER JOIN sessions sess ON s.id = sess.student_id
		WHERE sess.completed = true
		ORDER BY sess.score DESC, sess.total_time_taken_seconds ASC
		LIMIT 100
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to fetch leaderboard: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(OverallLeaderboardResponse{
			Success: false,
			Message: "Failed to fetch leaderboard",
		})
	}
	defer rows.Close()

	leaderboard := make([]LeaderboardEntry, 0)
	rank := 1

	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.StudentID, &entry.Name, &entry.Email, &entry.Score, &entry.TotalTimeTakenSeconds); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		entry.Rank = rank
		leaderboard = append(leaderboard, entry)
		rank++
	}

	// Get total count of completed sessions
	var total int
	countQuery := `SELECT COUNT(*) FROM sessions WHERE completed = true`
	err = db.Pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		log.Printf("Failed to count sessions: %v", err)
		total = len(leaderboard)
	}

	return c.Status(fiber.StatusOK).JSON(OverallLeaderboardResponse{
		Success: true,
		Total:   total,
		Data:    leaderboard,
	})
}

// ============================================
// SECTION-BASED TOP 100
// ============================================

type SectionLeaderboardEntry struct {
	Rank                  int    `json:"rank"`
	StudentID             int    `json:"student_id"`
	Name                  string `json:"name"`
	Email                 string `json:"email"`
	SectionScore          int    `json:"section_score"`
	SectionTimeTakenSeconds int  `json:"section_time_taken_seconds"`
}

type SectionLeaderboardResponse struct {
	Success     bool                       `json:"success"`
	Message     string                     `json:"message,omitempty"`
	SectionID   int                        `json:"section_id,omitempty"`
	SectionName string                     `json:"section_name,omitempty"`
	Total       int                        `json:"total,omitempty"`
	Data        []SectionLeaderboardEntry  `json:"data,omitempty"`
}

// GetSectionLeaderboardHandler handles GET /api/leaderboard/section/:section_id
func GetSectionLeaderboardHandler(c *fiber.Ctx) error {
	sectionID, err := c.ParamsInt("section_id")
	if err != nil || sectionID < 1 || sectionID > 4 {
		return c.Status(fiber.StatusBadRequest).JSON(SectionLeaderboardResponse{
			Success: false,
			Message: "Invalid section ID (must be 1-4)",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load questions to get section info and question IDs
	questionsFile, err := os.ReadFile("questions_with_timer.json")
	if err != nil {
		log.Printf("Failed to read questions file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(SectionLeaderboardResponse{
			Success: false,
			Message: "Failed to load questions",
		})
	}

	type JSONQuestion struct {
		ID int `json:"id"`
	}
	type JSONSection struct {
		ID        int            `json:"id"`
		Name      string         `json:"name"`
		Questions []JSONQuestion `json:"questions"`
	}
	var sections []JSONSection

	if err := json.Unmarshal(questionsFile, &sections); err != nil {
		log.Printf("Failed to parse questions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(SectionLeaderboardResponse{
			Success: false,
			Message: "Failed to parse questions",
		})
	}

	// Find the requested section
	var targetSection *JSONSection
	for i := range sections {
		if sections[i].ID == sectionID {
			targetSection = &sections[i]
			break
		}
	}

	if targetSection == nil {
		return c.Status(fiber.StatusNotFound).JSON(SectionLeaderboardResponse{
			Success: false,
			Message: "Section not found",
		})
	}

	// Extract question IDs for this section
	questionIDs := make([]int, len(targetSection.Questions))
	for i, q := range targetSection.Questions {
		questionIDs[i] = q.ID
	}

	// Query to calculate section scores and times
	query := `
		WITH section_scores AS (
			SELECT
				sess.student_id,
				COUNT(CASE WHEN a.is_correct = true THEN 1 END) as section_score,
				COALESCE(SUM(a.time_taken_seconds), 0) as section_time_taken_seconds
			FROM sessions sess
			LEFT JOIN answers a ON sess.id = a.session_id
			WHERE sess.completed = true
			AND a.question_id = ANY($1)
			GROUP BY sess.student_id
		)
		SELECT
			s.id,
			s.name,
			s.email,
			COALESCE(sc.section_score, 0) as section_score,
			COALESCE(sc.section_time_taken_seconds, 0) as section_time_taken_seconds
		FROM students s
		INNER JOIN section_scores sc ON s.id = sc.student_id
		ORDER BY sc.section_score DESC, sc.section_time_taken_seconds ASC
		LIMIT 100
	`

	rows, err := db.Pool.Query(ctx, query, questionIDs)
	if err != nil {
		log.Printf("Failed to fetch section leaderboard: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(SectionLeaderboardResponse{
			Success: false,
			Message: "Failed to fetch section leaderboard",
		})
	}
	defer rows.Close()

	leaderboard := make([]SectionLeaderboardEntry, 0)
	rank := 1

	for rows.Next() {
		var entry SectionLeaderboardEntry
		if err := rows.Scan(&entry.StudentID, &entry.Name, &entry.Email, &entry.SectionScore, &entry.SectionTimeTakenSeconds); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		entry.Rank = rank
		leaderboard = append(leaderboard, entry)
		rank++
	}

	// Get total count for this section
	countQuery := `
		SELECT COUNT(DISTINCT sess.student_id)
		FROM sessions sess
		INNER JOIN answers a ON sess.id = a.session_id
		WHERE sess.completed = true
		AND a.question_id = ANY($1)
	`
	var total int
	err = db.Pool.QueryRow(ctx, countQuery, questionIDs).Scan(&total)
	if err != nil {
		log.Printf("Failed to count section participants: %v", err)
		total = len(leaderboard)
	}

	return c.Status(fiber.StatusOK).JSON(SectionLeaderboardResponse{
		Success:     true,
		SectionID:   sectionID,
		SectionName: targetSection.Name,
		Total:       total,
		Data:        leaderboard,
	})
}

// ============================================
// USER SECTION RANKS
// ============================================

type UserSectionRank struct {
	SectionID             int    `json:"section_id"`
	SectionName           string `json:"section_name"`
	Score                 int    `json:"score"`
	TimeTakenSeconds      int    `json:"time_taken_seconds"`
	Rank                  int    `json:"rank"`
	TotalParticipants     int    `json:"total_participants"`
}

type UserSectionRanksResponse struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message,omitempty"`
	StudentID    int               `json:"student_id,omitempty"`
	StudentName  string            `json:"student_name,omitempty"`
	StudentEmail string            `json:"student_email,omitempty"`
	Sections     []UserSectionRank `json:"sections,omitempty"`
}

// GetUserSectionRanksHandler handles GET /api/leaderboard/user-sections?email=student@example.com
func GetUserSectionRanksHandler(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(UserSectionRanksResponse{
			Success: false,
			Message: "Email parameter is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get student by email
	var studentID int
	var studentName string
	studentQuery := `SELECT id, name FROM students WHERE email = $1`
	err := db.Pool.QueryRow(ctx, studentQuery, email).Scan(&studentID, &studentName)
	if err != nil {
		log.Printf("Student not found: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(UserSectionRanksResponse{
			Success: false,
			Message: "Student not found",
		})
	}

	// Check if student has a completed session
	var sessionID int
	sessionQuery := `SELECT id FROM sessions WHERE student_id = $1 AND completed = true`
	err = db.Pool.QueryRow(ctx, sessionQuery, studentID).Scan(&sessionID)
	if err != nil {
		log.Printf("No completed session found: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(UserSectionRanksResponse{
			Success: false,
			Message: "No completed session found for this student",
		})
	}

	// Load questions to get section info
	questionsFile, err := os.ReadFile("questions_with_timer.json")
	if err != nil {
		log.Printf("Failed to read questions file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(UserSectionRanksResponse{
			Success: false,
			Message: "Failed to load questions",
		})
	}

	type JSONQuestion struct {
		ID int `json:"id"`
	}
	type JSONSection struct {
		ID        int            `json:"id"`
		Name      string         `json:"name"`
		Questions []JSONQuestion `json:"questions"`
	}
	var sections []JSONSection

	if err := json.Unmarshal(questionsFile, &sections); err != nil {
		log.Printf("Failed to parse questions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(UserSectionRanksResponse{
			Success: false,
			Message: "Failed to parse questions",
		})
	}

	// Calculate ranks for each section
	userSectionRanks := make([]UserSectionRank, 0, len(sections))

	for _, section := range sections {
		// Extract question IDs for this section
		questionIDs := make([]int, len(section.Questions))
		for i, q := range section.Questions {
			questionIDs[i] = q.ID
		}

		// Get user's score and time for this section
		userScoreQuery := `
			SELECT
				COUNT(CASE WHEN a.is_correct = true THEN 1 END) as section_score,
				COALESCE(SUM(a.time_taken_seconds), 0) as section_time_taken_seconds
			FROM answers a
			WHERE a.session_id = $1
			AND a.question_id = ANY($2)
		`
		var userScore, userTime int
		err = db.Pool.QueryRow(ctx, userScoreQuery, sessionID, questionIDs).Scan(&userScore, &userTime)
		if err != nil {
			log.Printf("Failed to get user section score: %v", err)
			continue
		}

		// Calculate rank: count how many students scored better
		rankQuery := `
			WITH section_scores AS (
				SELECT
					sess.student_id,
					COUNT(CASE WHEN a.is_correct = true THEN 1 END) as section_score,
					COALESCE(SUM(a.time_taken_seconds), 0) as section_time_taken_seconds
				FROM sessions sess
				LEFT JOIN answers a ON sess.id = a.session_id
				WHERE sess.completed = true
				AND a.question_id = ANY($1)
				GROUP BY sess.student_id
			)
			SELECT COUNT(*) + 1
			FROM section_scores
			WHERE (section_score > $2)
			   OR (section_score = $2 AND section_time_taken_seconds < $3)
		`
		var rank int
		err = db.Pool.QueryRow(ctx, rankQuery, questionIDs, userScore, userTime).Scan(&rank)
		if err != nil {
			log.Printf("Failed to calculate rank: %v", err)
			rank = 0
		}

		// Get total participants for this section
		totalQuery := `
			SELECT COUNT(DISTINCT sess.student_id)
			FROM sessions sess
			INNER JOIN answers a ON sess.id = a.session_id
			WHERE sess.completed = true
			AND a.question_id = ANY($1)
		`
		var total int
		err = db.Pool.QueryRow(ctx, totalQuery, questionIDs).Scan(&total)
		if err != nil {
			log.Printf("Failed to count participants: %v", err)
			total = 0
		}

		userSectionRanks = append(userSectionRanks, UserSectionRank{
			SectionID:         section.ID,
			SectionName:       section.Name,
			Score:             userScore,
			TimeTakenSeconds:  userTime,
			Rank:              rank,
			TotalParticipants: total,
		})
	}

	return c.Status(fiber.StatusOK).JSON(UserSectionRanksResponse{
		Success:      true,
		StudentID:    studentID,
		StudentName:  studentName,
		StudentEmail: email,
		Sections:     userSectionRanks,
	})
}

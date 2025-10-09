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
		Email                 string `json:"email"`
		Score                 int    `json:"score"`
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

// GetComprehensiveStatsHandler handles GET /api/stats/comprehensive
// Returns all statistics in a single response:
// 1. Top 100 overall ranks
// 2. Section-wise top 100 ranks (all 4 sections)
// 3. Total attended conference
// 4. Total completed vs incomplete users
func GetComprehensiveStatsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response := fiber.Map{
		"success": true,
	}

	// ============================================
	// 1. TOP 100 OVERALL RANKS
	// ============================================
	overallQuery := `
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

	rows, err := db.Pool.Query(ctx, overallQuery)
	if err != nil {
		log.Printf("Failed to fetch overall leaderboard: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch overall leaderboard",
		})
	}

	type LeaderboardEntry struct {
		Rank                  int    `json:"rank"`
		StudentID             int    `json:"student_id"`
		Name                  string `json:"name"`
		Email                 string `json:"email"`
		Score                 int    `json:"score"`
		TotalTimeTakenSeconds int    `json:"total_time_taken_seconds"`
	}

	overallLeaderboard := make([]LeaderboardEntry, 0)
	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.StudentID, &entry.Name, &entry.Email, &entry.Score, &entry.TotalTimeTakenSeconds); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		entry.Rank = rank
		overallLeaderboard = append(overallLeaderboard, entry)
		rank++
	}
	rows.Close()

	response["top_100_overall"] = overallLeaderboard

	// ============================================
	// 2. SECTION-WISE TOP 100 RANKS (ALL 4 SECTIONS)
	// ============================================

	// Load questions to get section info
	questionsFile, err := os.ReadFile("questions_with_timer.json")
	if err != nil {
		log.Printf("Failed to read questions file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load questions",
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse questions",
		})
	}

	type SectionLeaderboardEntry struct {
		Rank                    int    `json:"rank"`
		StudentID               int    `json:"student_id"`
		Name                    string `json:"name"`
		Email                   string `json:"email"`
		SectionScore            int    `json:"section_score"`
		SectionTimeTakenSeconds int    `json:"section_time_taken_seconds"`
	}

	sectionLeaderboards := make(map[string]interface{})

	for _, section := range sections {
		// Extract question IDs for this section
		questionIDs := make([]int, len(section.Questions))
		for i, q := range section.Questions {
			questionIDs[i] = q.ID
		}

		// Query to calculate section scores and times
		sectionQuery := `
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

		sectionRows, err := db.Pool.Query(ctx, sectionQuery, questionIDs)
		if err != nil {
			log.Printf("Failed to fetch section %d leaderboard: %v", section.ID, err)
			continue
		}

		sectionLeaderboard := make([]SectionLeaderboardEntry, 0)
		sectionRank := 1

		for sectionRows.Next() {
			var entry SectionLeaderboardEntry
			if err := sectionRows.Scan(&entry.StudentID, &entry.Name, &entry.Email, &entry.SectionScore, &entry.SectionTimeTakenSeconds); err != nil {
				log.Printf("Failed to scan section row: %v", err)
				continue
			}
			entry.Rank = sectionRank
			sectionLeaderboard = append(sectionLeaderboard, entry)
			sectionRank++
		}
		sectionRows.Close()

		// Get total count for this section
		countQuery := `
			SELECT COUNT(DISTINCT sess.student_id)
			FROM sessions sess
			INNER JOIN answers a ON sess.id = a.session_id
			WHERE sess.completed = true
			AND a.question_id = ANY($1)
		`
		var sectionTotal int
		err = db.Pool.QueryRow(ctx, countQuery, questionIDs).Scan(&sectionTotal)
		if err != nil {
			log.Printf("Failed to count section participants: %v", err)
			sectionTotal = len(sectionLeaderboard)
		}

		sectionLeaderboards[section.Name] = fiber.Map{
			"section_id":   section.ID,
			"section_name": section.Name,
			"total":        sectionTotal,
			"top_100":      sectionLeaderboard,
		}
	}

	response["section_leaderboards"] = sectionLeaderboards

	// ============================================
	// 3. TOTAL ATTENDED CONFERENCE
	// ============================================
	var totalAttended int
	attendedQuery := `
		SELECT COUNT(DISTINCT student_id)
		FROM email_tracking
		WHERE conference_attended = true
	`
	err = db.Pool.QueryRow(ctx, attendedQuery).Scan(&totalAttended)
	if err != nil {
		log.Printf("Failed to count attended: %v", err)
		totalAttended = 0
	}

	response["total_attended_conference"] = totalAttended

	// ============================================
	// 4. COMPLETION STATISTICS
	// ============================================

	// Total who started test (have a session)
	var totalStarted int
	startedQuery := `SELECT COUNT(*) FROM sessions`
	err = db.Pool.QueryRow(ctx, startedQuery).Scan(&totalStarted)
	if err != nil {
		log.Printf("Failed to count started: %v", err)
		totalStarted = 0
	}

	// Total who completed test
	var totalCompleted int
	completedQuery := `SELECT COUNT(*) FROM sessions WHERE completed = true`
	err = db.Pool.QueryRow(ctx, completedQuery).Scan(&totalCompleted)
	if err != nil {
		log.Printf("Failed to count completed: %v", err)
		totalCompleted = 0
	}

	// Total incomplete (started but not completed)
	totalIncomplete := totalStarted - totalCompleted

	// Total who got access code but never started
	var totalNeverStarted int
	neverStartedQuery := `
		SELECT COUNT(*)
		FROM email_tracking et
		WHERE et.conference_attended = true
		AND et.access_code IS NOT NULL
		AND NOT EXISTS (
			SELECT 1 FROM sessions s WHERE s.student_id = et.student_id
		)
	`
	err = db.Pool.QueryRow(ctx, neverStartedQuery).Scan(&totalNeverStarted)
	if err != nil {
		log.Printf("Failed to count never started: %v", err)
		totalNeverStarted = 0
	}

	response["completion_stats"] = fiber.Map{
		"total_attended_conference": totalAttended,
		"total_started_test":        totalStarted,
		"total_completed_test":      totalCompleted,
		"total_incomplete_test":     totalIncomplete,
		"total_never_started":       totalNeverStarted,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

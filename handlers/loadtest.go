package handlers

import (
	"context"
	"fmt"
	"mcq-exam/db"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Test MCQ response structure
type TestMCQResponse struct {
	QuestionText string `json:"question_text"`
	OptionA      string `json:"option_a"`
	OptionB      string `json:"option_b"`
	OptionC      string `json:"option_c"`
	OptionD      string `json:"option_d"`
}

// Metrics structure
type LoadTestMetrics struct {
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulReqs    int64         `json:"successful_requests"`
	FailedReqs        int64         `json:"failed_requests"`
	TotalDBTime       time.Duration `json:"total_db_time_ms"`
	MinDBTime         time.Duration `json:"min_db_time_ms"`
	MaxDBTime         time.Duration `json:"max_db_time_ms"`
	AvgDBTime         time.Duration `json:"avg_db_time_ms"`
	P50DBTime         time.Duration `json:"p50_db_time_ms"`
	P95DBTime         time.Duration `json:"p95_db_time_ms"`
	P99DBTime         time.Duration `json:"p99_db_time_ms"`
	ErrorRate         float64       `json:"error_rate"`
	mu                sync.RWMutex
	dbTimes           []time.Duration
}

var (
	individualMetrics = &LoadTestMetrics{dbTimes: make([]time.Duration, 0)}
	batchMetrics      = &LoadTestMetrics{dbTimes: make([]time.Duration, 0)}
)

// Individual insert test - inserts 5 records one by one
func LoadTestIndividualHandler(c *fiber.Ctx) error {
	startTime := time.Now()

	// Parse request body (5 MCQ responses)
	var responses []TestMCQResponse
	if err := c.BodyParser(&responses); err != nil {
		individualMetrics.recordFailure()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(responses) != 5 {
		individualMetrics.recordFailure()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Expected exactly 5 MCQ responses",
		})
	}

	// Insert each record individually
	dbStartTime := time.Now()
	ctx := context.Background()
	for _, resp := range responses {
		query := `
			INSERT INTO test_mcq_responses (question_text, option_a, option_b, option_c, option_d)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err := db.Pool.Exec(ctx, query, resp.QuestionText, resp.OptionA, resp.OptionB, resp.OptionC, resp.OptionD)
		if err != nil {
			individualMetrics.recordFailure()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database insert failed",
			})
		}
	}
	dbDuration := time.Since(dbStartTime)

	individualMetrics.recordSuccess(dbDuration)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":         "Individual inserts completed",
		"records_created": 5,
		"response_time":   time.Since(startTime).Milliseconds(),
		"db_time":         dbDuration.Milliseconds(),
	})
}

// Batch insert test - inserts 5 records in a single query
func LoadTestBatchHandler(c *fiber.Ctx) error {
	startTime := time.Now()

	// Parse request body (5 MCQ responses)
	var responses []TestMCQResponse
	if err := c.BodyParser(&responses); err != nil {
		batchMetrics.recordFailure()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(responses) != 5 {
		batchMetrics.recordFailure()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Expected exactly 5 MCQ responses",
		})
	}

	// Batch insert using single query
	dbStartTime := time.Now()
	ctx := context.Background()
	query := `
		INSERT INTO test_mcq_responses (question_text, option_a, option_b, option_c, option_d)
		VALUES
			($1, $2, $3, $4, $5),
			($6, $7, $8, $9, $10),
			($11, $12, $13, $14, $15),
			($16, $17, $18, $19, $20),
			($21, $22, $23, $24, $25)
	`
	_, err := db.Pool.Exec(ctx, query,
		responses[0].QuestionText, responses[0].OptionA, responses[0].OptionB, responses[0].OptionC, responses[0].OptionD,
		responses[1].QuestionText, responses[1].OptionA, responses[1].OptionB, responses[1].OptionC, responses[1].OptionD,
		responses[2].QuestionText, responses[2].OptionA, responses[2].OptionB, responses[2].OptionC, responses[2].OptionD,
		responses[3].QuestionText, responses[3].OptionA, responses[3].OptionB, responses[3].OptionC, responses[3].OptionD,
		responses[4].QuestionText, responses[4].OptionA, responses[4].OptionB, responses[4].OptionC, responses[4].OptionD,
	)
	if err != nil {
		batchMetrics.recordFailure()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database batch insert failed",
		})
	}
	dbDuration := time.Since(dbStartTime)

	batchMetrics.recordSuccess(dbDuration)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":         "Batch insert completed",
		"records_created": 5,
		"response_time":   time.Since(startTime).Milliseconds(),
		"db_time":         dbDuration.Milliseconds(),
	})
}

// Get metrics for individual test
func GetIndividualMetricsHandler(c *fiber.Ctx) error {
	return c.JSON(individualMetrics.getMetrics())
}

// Get metrics for batch test
func GetBatchMetricsHandler(c *fiber.Ctx) error {
	return c.JSON(batchMetrics.getMetrics())
}

// Reset metrics
func ResetLoadTestMetricsHandler(c *fiber.Ctx) error {
	individualMetrics.reset()
	batchMetrics.reset()
	return c.JSON(fiber.Map{
		"message": "Metrics reset successfully",
	})
}

// Helper methods for metrics
func (m *LoadTestMetrics) recordSuccess(dbTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRequests++
	m.SuccessfulReqs++
	m.dbTimes = append(m.dbTimes, dbTime)
}

func (m *LoadTestMetrics) recordFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRequests++
	m.FailedReqs++
}

func (m *LoadTestMetrics) getMetrics() fiber.Map {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.dbTimes) == 0 {
		return fiber.Map{
			"total_requests":      m.TotalRequests,
			"successful_requests": m.SuccessfulReqs,
			"failed_requests":     m.FailedReqs,
			"error_rate":          0.0,
			"message":             "No data collected yet",
		}
	}

	// Calculate percentiles
	p50, p95, p99 := calculatePercentiles(m.dbTimes)

	// Calculate min, max, avg
	var total time.Duration
	min := m.dbTimes[0]
	max := m.dbTimes[0]

	for _, t := range m.dbTimes {
		total += t
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}

	avg := total / time.Duration(len(m.dbTimes))
	errorRate := 0.0
	if m.TotalRequests > 0 {
		errorRate = float64(m.FailedReqs) / float64(m.TotalRequests) * 100
	}

	return fiber.Map{
		"total_requests":      m.TotalRequests,
		"successful_requests": m.SuccessfulReqs,
		"failed_requests":     m.FailedReqs,
		"error_rate":          fmt.Sprintf("%.2f%%", errorRate),
		"db_metrics": fiber.Map{
			"min_ms": min.Milliseconds(),
			"max_ms": max.Milliseconds(),
			"avg_ms": avg.Milliseconds(),
			"p50_ms": p50.Milliseconds(),
			"p95_ms": p95.Milliseconds(),
			"p99_ms": p99.Milliseconds(),
		},
	}
}

func (m *LoadTestMetrics) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRequests = 0
	m.SuccessfulReqs = 0
	m.FailedReqs = 0
	m.dbTimes = make([]time.Duration, 0)
}

// Calculate percentiles (simple implementation)
func calculatePercentiles(times []time.Duration) (p50, p95, p99 time.Duration) {
	if len(times) == 0 {
		return 0, 0, 0
	}

	// Create a sorted copy
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)

	// Simple bubble sort (good enough for metrics)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p50Index := int(float64(len(sorted)) * 0.50)
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	if p50Index >= len(sorted) {
		p50Index = len(sorted) - 1
	}
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	return sorted[p50Index], sorted[p95Index], sorted[p99Index]
}

// Cleanup test data
func CleanupLoadTestDataHandler(c *fiber.Ctx) error {
	ctx := context.Background()
	query := `DELETE FROM test_mcq_responses`
	result, err := db.Pool.Exec(ctx, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to cleanup test data",
		})
	}

	rowsDeleted := result.RowsAffected()
	return c.JSON(fiber.Map{
		"message":      "Test data cleaned up successfully",
		"rows_deleted": rowsDeleted,
	})
}

// Save test results to database
func SaveTestResultsHandler(c *fiber.Ctx) error {
	// Request body structure
	type SaveTestResultRequest struct {
		TestType     string  `json:"test_type"`
		Notes        string  `json:"notes"`
		TestDuration int     `json:"test_duration_seconds"`
	}

	var req SaveTestResultRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate test type
	if req.TestType != "individual" && req.TestType != "batch" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "test_type must be 'individual' or 'batch'",
		})
	}

	// Get current metrics based on test type
	var metrics *LoadTestMetrics
	if req.TestType == "individual" {
		metrics = individualMetrics
	} else {
		metrics = batchMetrics
	}

	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	if metrics.TotalRequests == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No test data available. Run a test first.",
		})
	}

	// Calculate metrics
	p50, p95, p99 := calculatePercentiles(metrics.dbTimes)
	var total time.Duration
	min := metrics.dbTimes[0]
	max := metrics.dbTimes[0]

	for _, t := range metrics.dbTimes {
		total += t
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}

	avg := total / time.Duration(len(metrics.dbTimes))
	errorRate := 0.0
	if metrics.TotalRequests > 0 {
		errorRate = float64(metrics.FailedReqs) / float64(metrics.TotalRequests) * 100
	}

	// Save to database
	ctx := context.Background()
	query := `
		INSERT INTO test_results (
			test_type, total_requests, successful_requests, failed_requests,
			error_rate, min_db_time_ms, max_db_time_ms, avg_db_time_ms,
			p50_db_time_ms, p95_db_time_ms, p99_db_time_ms,
			test_duration_seconds, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`

	var resultID int
	var createdAt time.Time

	err := db.Pool.QueryRow(ctx, query,
		req.TestType,
		metrics.TotalRequests,
		metrics.SuccessfulReqs,
		metrics.FailedReqs,
		errorRate,
		min.Milliseconds(),
		max.Milliseconds(),
		avg.Milliseconds(),
		p50.Milliseconds(),
		p95.Milliseconds(),
		p99.Milliseconds(),
		req.TestDuration,
		req.Notes,
	).Scan(&resultID, &createdAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save test results",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Test results saved successfully",
		"result_id":  resultID,
		"created_at": createdAt,
		"summary": fiber.Map{
			"test_type":           req.TestType,
			"total_requests":      metrics.TotalRequests,
			"successful_requests": metrics.SuccessfulReqs,
			"failed_requests":     metrics.FailedReqs,
			"error_rate":          fmt.Sprintf("%.2f%%", errorRate),
			"db_metrics": fiber.Map{
				"min_ms": min.Milliseconds(),
				"max_ms": max.Milliseconds(),
				"avg_ms": avg.Milliseconds(),
				"p50_ms": p50.Milliseconds(),
				"p95_ms": p95.Milliseconds(),
				"p99_ms": p99.Milliseconds(),
			},
		},
	})
}

// Get all test results from database
func GetAllTestResultsHandler(c *fiber.Ctx) error {
	ctx := context.Background()

	// Optional query params for filtering
	testType := c.Query("test_type") // "individual" or "batch"
	limit := c.QueryInt("limit", 50)

	query := `
		SELECT
			id, test_type, total_requests, successful_requests, failed_requests,
			error_rate, min_db_time_ms, max_db_time_ms, avg_db_time_ms,
			p50_db_time_ms, p95_db_time_ms, p99_db_time_ms,
			test_duration_seconds, notes, created_at
		FROM test_results
	`

	args := []interface{}{}
	argIndex := 1

	if testType != "" {
		query += fmt.Sprintf(" WHERE test_type = $%d", argIndex)
		args = append(args, testType)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch test results",
		})
	}
	defer rows.Close()

	type TestResult struct {
		ID                   int       `json:"id"`
		TestType             string    `json:"test_type"`
		TotalRequests        int64     `json:"total_requests"`
		SuccessfulRequests   int64     `json:"successful_requests"`
		FailedRequests       int64     `json:"failed_requests"`
		ErrorRate            float64   `json:"error_rate"`
		MinDBTimeMs          *int64    `json:"min_db_time_ms"`
		MaxDBTimeMs          *int64    `json:"max_db_time_ms"`
		AvgDBTimeMs          *int64    `json:"avg_db_time_ms"`
		P50DBTimeMs          *int64    `json:"p50_db_time_ms"`
		P95DBTimeMs          *int64    `json:"p95_db_time_ms"`
		P99DBTimeMs          *int64    `json:"p99_db_time_ms"`
		TestDurationSeconds  *int      `json:"test_duration_seconds"`
		Notes                *string   `json:"notes"`
		CreatedAt            time.Time `json:"created_at"`
	}

	results := []TestResult{}
	for rows.Next() {
		var r TestResult
		err := rows.Scan(
			&r.ID, &r.TestType, &r.TotalRequests, &r.SuccessfulRequests,
			&r.FailedRequests, &r.ErrorRate, &r.MinDBTimeMs, &r.MaxDBTimeMs,
			&r.AvgDBTimeMs, &r.P50DBTimeMs, &r.P95DBTimeMs, &r.P99DBTimeMs,
			&r.TestDurationSeconds, &r.Notes, &r.CreatedAt,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to parse test results",
			})
		}
		results = append(results, r)
	}

	return c.JSON(fiber.Map{
		"total":   len(results),
		"results": results,
	})
}

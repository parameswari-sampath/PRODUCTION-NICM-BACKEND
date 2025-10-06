-- Test MCQ responses table for load testing
CREATE TABLE IF NOT EXISTS test_mcq_responses (
    id SERIAL PRIMARY KEY,
    question_text TEXT NOT NULL,
    option_a TEXT NOT NULL,
    option_b TEXT NOT NULL,
    option_c TEXT NOT NULL,
    option_d TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_test_mcq_responses_created_at ON test_mcq_responses(created_at);

-- Test results table to store load test run results
CREATE TABLE IF NOT EXISTS test_results (
    id SERIAL PRIMARY KEY,
    test_type VARCHAR(50) NOT NULL,
    total_requests BIGINT NOT NULL,
    successful_requests BIGINT NOT NULL,
    failed_requests BIGINT NOT NULL,
    error_rate DECIMAL(5,2) NOT NULL,
    min_db_time_ms BIGINT,
    max_db_time_ms BIGINT,
    avg_db_time_ms BIGINT,
    p50_db_time_ms BIGINT,
    p95_db_time_ms BIGINT,
    p99_db_time_ms BIGINT,
    test_duration_seconds INT,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_test_results_test_type ON test_results(test_type);
CREATE INDEX IF NOT EXISTS idx_test_results_created_at ON test_results(created_at);

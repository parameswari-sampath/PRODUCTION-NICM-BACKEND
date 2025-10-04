-- Students table
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_students_email ON students(email);

-- Email logs table
CREATE TABLE IF NOT EXISTS email_logs (
    id SERIAL PRIMARY KEY,
    student_id INT REFERENCES students(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    status VARCHAR(50) DEFAULT 'sent',
    request_id VARCHAR(255),
    response_code VARCHAR(50),
    response_message TEXT,
    sent_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_logs_student_id ON email_logs(student_id);

-- Event schedule table
CREATE TABLE IF NOT EXISTS event_schedule (
    id SERIAL PRIMARY KEY,
    first_function VARCHAR(100) NOT NULL,
    first_scheduled_time TIMESTAMPTZ NOT NULL,
    first_executed BOOLEAN DEFAULT false,
    first_executed_at TIMESTAMPTZ,
    second_function VARCHAR(100) NOT NULL,
    second_scheduled_time TIMESTAMPTZ NOT NULL,
    second_executed BOOLEAN DEFAULT false,
    second_executed_at TIMESTAMPTZ,
    video_url VARCHAR(500),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_schedule_first_time ON event_schedule(first_scheduled_time) WHERE first_executed = false;
CREATE INDEX IF NOT EXISTS idx_event_schedule_second_time ON event_schedule(second_scheduled_time) WHERE second_executed = false;

-- Email tracking table
CREATE TABLE IF NOT EXISTS email_tracking (
    id SERIAL PRIMARY KEY,
    student_id INT REFERENCES students(id) ON DELETE CASCADE,
    email_type VARCHAR(50) NOT NULL,
    conference_token VARCHAR(100),
    conference_attended BOOLEAN DEFAULT false,
    conference_attended_at TIMESTAMPTZ,
    access_code VARCHAR(10),
    opened BOOLEAN DEFAULT false,
    opened_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_student_email_type UNIQUE (student_id, email_type)
);

CREATE INDEX IF NOT EXISTS idx_email_tracking_student_email ON email_tracking(student_id, email_type);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    student_id INT REFERENCES students(id) ON DELETE CASCADE,
    session_token VARCHAR(100) UNIQUE NOT NULL,
    access_code VARCHAR(10),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed BOOLEAN DEFAULT false,
    completed_at TIMESTAMPTZ,
    score INT,
    total_time_taken_seconds INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_student_id ON sessions(student_id);
CREATE INDEX IF NOT EXISTS idx_sessions_access_code ON sessions(access_code);
CREATE INDEX IF NOT EXISTS idx_sessions_session_token ON sessions(session_token);

-- Answers table
CREATE TABLE IF NOT EXISTS answers (
    id SERIAL PRIMARY KEY,
    session_id INT REFERENCES sessions(id) ON DELETE CASCADE,
    question_id INT NOT NULL,
    selected_option_index INT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    time_taken_seconds INT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_answers_session_id ON answers(session_id);
CREATE INDEX IF NOT EXISTS idx_answers_question_id ON answers(question_id);

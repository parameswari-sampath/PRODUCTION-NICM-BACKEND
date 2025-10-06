# Load Test Endpoints

Complete API reference for load testing endpoints.

## Endpoints

### 1. Individual Insert Test
**POST** `/api/load-test/individual`

Inserts 5 MCQ records one by one (5 separate INSERT queries).

**Request Body:**
```json
[
  {
    "question_text": "What is the capital of France?",
    "option_a": "London",
    "option_b": "Paris",
    "option_c": "Berlin",
    "option_d": "Madrid"
  },
  {
    "question_text": "What is 2 + 2?",
    "option_a": "3",
    "option_b": "4",
    "option_c": "5",
    "option_d": "6"
  },
  {
    "question_text": "Which planet is closest to the Sun?",
    "option_a": "Venus",
    "option_b": "Earth",
    "option_c": "Mercury",
    "option_d": "Mars"
  },
  {
    "question_text": "What is the largest ocean?",
    "option_a": "Atlantic",
    "option_b": "Pacific",
    "option_c": "Indian",
    "option_d": "Arctic"
  },
  {
    "question_text": "Who wrote Romeo and Juliet?",
    "option_a": "Charles Dickens",
    "option_b": "William Shakespeare",
    "option_c": "Mark Twain",
    "option_d": "Jane Austen"
  }
]
```

**Response:**
```json
{
  "message": "Individual inserts completed",
  "records_created": 5,
  "response_time": 45,
  "db_time": 42
}
```

---

### 2. Batch Insert Test
**POST** `/api/load-test/batch`

Inserts 5 MCQ records in a single batch query (1 INSERT query).

**Request Body:** Same as individual test

**Response:**
```json
{
  "message": "Batch insert completed",
  "records_created": 5,
  "response_time": 15,
  "db_time": 12
}
```

---

### 3. Get Individual Test Metrics
**GET** `/api/load-test/metrics/individual`

Returns real-time metrics for individual insert tests.

**Response:**
```json
{
  "total_requests": 10000,
  "successful_requests": 9950,
  "failed_requests": 50,
  "error_rate": "0.50%",
  "db_metrics": {
    "min_ms": 5,
    "max_ms": 450,
    "avg_ms": 45,
    "p50_ms": 38,
    "p95_ms": 120,
    "p99_ms": 280
  }
}
```

---

### 4. Get Batch Test Metrics
**GET** `/api/load-test/metrics/batch`

Returns real-time metrics for batch insert tests.

**Response:** Same format as individual metrics

---

### 5. Reset Metrics
**POST** `/api/load-test/metrics/reset`

Resets all metrics counters (individual and batch).

**Response:**
```json
{
  "message": "Metrics reset successfully"
}
```

---

### 6. Cleanup Test Data
**DELETE** `/api/load-test/cleanup`

Deletes all test MCQ records from the database.

**Response:**
```json
{
  "message": "Test data cleaned up successfully",
  "rows_deleted": 50000
}
```

---

### 7. Save Test Results (NEW)
**POST** `/api/load-test/results/save`

Saves current test metrics to database for historical tracking.

**Request Body:**
```json
{
  "test_type": "individual",
  "test_duration_seconds": 300,
  "notes": "2k req/sec test on 2 vCPU"
}
```

**Fields:**
- `test_type`: `"individual"` or `"batch"`
- `test_duration_seconds`: How long the test ran (optional)
- `notes`: Any notes about the test (optional)

**Response:**
```json
{
  "message": "Test results saved successfully",
  "result_id": 42,
  "created_at": "2025-10-06T12:30:00Z",
  "summary": {
    "test_type": "individual",
    "total_requests": 600000,
    "successful_requests": 598500,
    "failed_requests": 1500,
    "error_rate": "0.25%",
    "db_metrics": {
      "min_ms": 2,
      "max_ms": 450,
      "avg_ms": 45,
      "p50_ms": 38,
      "p95_ms": 120,
      "p99_ms": 280
    }
  }
}
```

---

### 8. Get All Test Results (NEW)
**GET** `/api/load-test/results`

Retrieves all saved test results from database.

**Query Parameters:**
- `test_type` (optional): Filter by `"individual"` or `"batch"`
- `limit` (optional): Max results to return (default: 50)

**Examples:**
```bash
# Get all results
GET /api/load-test/results

# Get only batch test results
GET /api/load-test/results?test_type=batch

# Get last 10 results
GET /api/load-test/results?limit=10
```

**Response:**
```json
{
  "total": 15,
  "results": [
    {
      "id": 42,
      "test_type": "individual",
      "total_requests": 600000,
      "successful_requests": 598500,
      "failed_requests": 1500,
      "error_rate": 0.25,
      "min_db_time_ms": 2,
      "max_db_time_ms": 450,
      "avg_db_time_ms": 45,
      "p50_db_time_ms": 38,
      "p95_db_time_ms": 120,
      "p99_db_time_ms": 280,
      "test_duration_seconds": 300,
      "notes": "2k req/sec test on 2 vCPU",
      "created_at": "2025-10-06T12:30:00Z"
    }
  ]
}
```

---

## Usage Flow

### Running a Test

1. **Reset metrics (optional but recommended)**
   ```bash
   curl -X POST http://your-server/api/load-test/metrics/reset
   ```

2. **Hit the endpoint from local (2k req/sec for 5 min)**
   ```bash
   # Use your load testing tool from local machine
   # Send requests to: http://your-server/api/load-test/individual
   # OR: http://your-server/api/load-test/batch
   ```

3. **Check real-time metrics**
   ```bash
   curl http://your-server/api/load-test/metrics/individual
   curl http://your-server/api/load-test/metrics/batch
   ```

4. **Save results to database**
   ```bash
   curl -X POST http://your-server/api/load-test/results/save \
     -H "Content-Type: application/json" \
     -d '{
       "test_type": "individual",
       "test_duration_seconds": 300,
       "notes": "First test on production hardware"
     }'
   ```

5. **View all saved results**
   ```bash
   curl http://your-server/api/load-test/results
   ```

6. **Cleanup**
   ```bash
   curl -X DELETE http://your-server/api/load-test/cleanup
   ```

---

## Database Tables

### test_mcq_responses
Stores the actual MCQ data inserted during tests.

### test_results
Stores historical test run results with metrics (p50, p95, p99, etc).

---

## Metrics Explained

- **p50 (50th percentile)**: 50% of requests were faster than this
- **p95 (95th percentile)**: 95% of requests were faster than this (only 5% slower)
- **p99 (99th percentile)**: 99% of requests were faster than this (worst 1%)
- **error_rate**: Percentage of failed requests
- **db_time**: Time spent in database queries only (excludes network, parsing, etc)
- **response_time**: Total time for the entire request-response cycle

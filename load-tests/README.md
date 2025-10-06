# Load Testing Suite

This directory contains load testing scripts for the MCQ application using k6.

## Prerequisites

1. Install k6: https://k6.io/docs/getting-started/installation/
   ```bash
   # macOS
   brew install k6

   # Linux
   sudo gpg -k
   sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
   echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
   sudo apt-get update
   sudo apt-get install k6

   # Docker
   docker pull grafana/k6
   ```

## Test Configuration

- **Target Load**: 2,000 requests/second
- **Duration**: 5 minutes
- **Total Requests**: ~600,000 requests
- **Data per Request**: 5 MCQ responses
- **Total DB Inserts**: ~3,000,000 records

## Available Tests

### 1. Individual Insert Test
Tests inserting 5 records one by one (5 separate INSERT statements per request)

```bash
k6 run individual-test.js
```

### 2. Batch Insert Test
Tests inserting 5 records in a single batch query (1 INSERT statement per request)

```bash
k6 run batch-test.js
```

## Running the Tests

### Step 1: Start the Application
```bash
# Make sure your app is running
go run main.go

# Or with Docker
docker-compose up
```

### Step 2: Reset Metrics (Optional)
```bash
curl -X POST http://localhost:8080/api/load-test/metrics/reset
```

### Step 3: Run a Test
```bash
# Run individual insert test
k6 run individual-test.js

# Run batch insert test
k6 run batch-test.js
```

### Step 4: View App Metrics
```bash
# Individual test metrics
curl http://localhost:8080/api/load-test/metrics/individual | jq

# Batch test metrics
curl http://localhost:8080/api/load-test/metrics/batch | jq
```

### Step 5: Cleanup Test Data
```bash
curl -X DELETE http://localhost:8080/api/load-test/cleanup
```

## Understanding the Metrics

### k6 Metrics (from terminal output)

- **http_reqs**: Total HTTP requests made
- **http_req_duration**: Response time metrics
  - `p(50)`: 50% of requests finished faster than this
  - `p(95)`: 95% of requests finished faster than this
  - `p(99)`: 99% of requests finished faster than this
- **http_req_failed**: Percentage of failed requests
- **iteration_duration**: Total time per iteration

### Application Metrics (from API endpoints)

Query the metrics endpoints to get detailed stats:

```bash
# Individual test metrics
curl http://localhost:8080/api/load-test/metrics/individual

# Example response:
{
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
```

## Performance Comparison

Run both tests and compare:

| Metric | Individual | Batch | Winner |
|--------|-----------|-------|--------|
| Avg Response Time | ? ms | ? ms | ? |
| P95 Response Time | ? ms | ? ms | ? |
| P99 Response Time | ? ms | ? ms | ? |
| Error Rate | ?% | ?% | ? |
| Throughput | ? req/s | ? req/s | ? |

## Expected Results on 2 vCPU

Based on the hardware (2 vCPU + Docker Postgres), you can expect:

- **Individual Inserts**: Higher latency due to 5 separate DB calls
  - Expected: 100-200ms average response time
  - May struggle to maintain 2k req/sec consistently

- **Batch Inserts**: Lower latency with single DB call
  - Expected: 20-50ms average response time
  - Better chance of sustaining 2k req/sec

## Troubleshooting

### Test fails immediately
- Check if app is running: `curl http://localhost:8080/health`
- Check database connection: `docker-compose ps`

### High error rates
- Database connection pool exhausted
- CPU maxed out
- Check app logs: `docker-compose logs -f app`

### Can't install k6
- Use Docker: `docker run --rm -i grafana/k6 run - <individual-test.js`

## Cleanup

After testing, clean up the test data:

```bash
# Delete all test records
curl -X DELETE http://localhost:8080/api/load-test/cleanup

# Reset metrics
curl -X POST http://localhost:8080/api/load-test/metrics/reset
```

## Test Result Files

After each test run, k6 generates:
- `individual-test-results.json` - Detailed metrics for individual test
- `batch-test-results.json` - Detailed metrics for batch test

These files contain comprehensive performance data for analysis.

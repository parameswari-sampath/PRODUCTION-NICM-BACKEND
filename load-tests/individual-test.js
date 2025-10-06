import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const dbTime = new Trend('db_time');
const responseTime = new Trend('response_time');

// Test configuration
export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 2000,              // 2000 requests per second
      timeUnit: '1s',
      duration: '5m',          // 5 minutes
      preAllocatedVUs: 100,    // Initial number of VUs
      maxVUs: 500,             // Maximum VUs if needed
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<500'],  // 95% of requests should be below 500ms
    'errors': ['rate<0.1'],               // Error rate should be below 10%
  },
};

// Test data - 5 MCQ responses
const payload = JSON.stringify([
  {
    question_text: "What is the capital of France?",
    option_a: "London",
    option_b: "Paris",
    option_c: "Berlin",
    option_d: "Madrid"
  },
  {
    question_text: "What is 2 + 2?",
    option_a: "3",
    option_b: "4",
    option_c: "5",
    option_d: "6"
  },
  {
    question_text: "Which planet is closest to the Sun?",
    option_a: "Venus",
    option_b: "Earth",
    option_c: "Mercury",
    option_d: "Mars"
  },
  {
    question_text: "What is the largest ocean?",
    option_a: "Atlantic",
    option_b: "Pacific",
    option_c: "Indian",
    option_d: "Arctic"
  },
  {
    question_text: "Who wrote Romeo and Juliet?",
    option_a: "Charles Dickens",
    option_b: "William Shakespeare",
    option_c: "Mark Twain",
    option_d: "Jane Austen"
  }
]);

const params = {
  headers: {
    'Content-Type': 'application/json',
  },
};

export default function () {
  const response = http.post(
    'http://localhost:8080/api/load-test/individual',
    payload,
    params
  );

  // Check response
  const success = check(response, {
    'status is 201': (r) => r.status === 201,
    'response has message': (r) => r.json('message') !== undefined,
  });

  errorRate.add(!success);

  // Track metrics if successful
  if (success && response.json()) {
    const body = response.json();
    if (body.db_time) {
      dbTime.add(body.db_time);
    }
    responseTime.add(body.response_time);
  }
}

export function handleSummary(data) {
  return {
    'individual-test-results.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options?.indent || '';
  const enableColors = options?.enableColors || false;

  let summary = '\n';
  summary += indent + '========== INDIVIDUAL INSERT TEST RESULTS ==========\n\n';

  summary += indent + 'Requests:\n';
  summary += indent + `  Total: ${data.metrics.http_reqs.values.count}\n`;
  summary += indent + `  Successful: ${data.metrics.http_reqs.values.count - (data.metrics.errors ? data.metrics.errors.values.count : 0)}\n`;
  summary += indent + `  Failed: ${data.metrics.errors ? data.metrics.errors.values.count : 0}\n`;
  summary += indent + `  Error Rate: ${((data.metrics.errors?.values.rate || 0) * 100).toFixed(2)}%\n\n`;

  summary += indent + 'Response Time:\n';
  summary += indent + `  Min: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
  summary += indent + `  Max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
  summary += indent + `  Avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += indent + `  P50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
  summary += indent + `  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += indent + `  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n\n`;

  if (data.metrics.db_time) {
    summary += indent + 'Database Time:\n';
    summary += indent + `  Min: ${data.metrics.db_time.values.min.toFixed(2)}ms\n`;
    summary += indent + `  Max: ${data.metrics.db_time.values.max.toFixed(2)}ms\n`;
    summary += indent + `  Avg: ${data.metrics.db_time.values.avg.toFixed(2)}ms\n`;
    summary += indent + `  P50: ${data.metrics.db_time.values['p(50)'].toFixed(2)}ms\n`;
    summary += indent + `  P95: ${data.metrics.db_time.values['p(95)'].toFixed(2)}ms\n`;
    summary += indent + `  P99: ${data.metrics.db_time.values['p(99)'].toFixed(2)}ms\n\n`;
  }

  summary += indent + 'Throughput:\n';
  summary += indent + `  Requests/sec: ${data.metrics.http_reqs.values.rate.toFixed(2)}\n\n`;

  summary += indent + '===================================================\n';

  return summary;
}

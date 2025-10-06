#!/usr/bin/env python3
"""
Load Test Script for MCQ API
Sends 2000 requests/sec for 30 seconds to test individual and batch endpoints
"""

import requests
import time
import threading
import statistics
from datetime import datetime
from collections import defaultdict
import json

# Configuration
API_BASE_URL = "https://api.smart-mcq.com"
TARGET_RPS = 2000  # Requests per second
TEST_DURATION = 300  # Seconds (5 minutes)
THREADS = 50  # Number of concurrent threads

# Test payload - 5 MCQ questions
PAYLOAD = [
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

# Global metrics
class Metrics:
    def __init__(self):
        self.response_times = []
        self.db_times = []
        self.successful_requests = 0
        self.failed_requests = 0
        self.status_codes = defaultdict(int)
        self.lock = threading.Lock()
        self.start_time = None
        self.end_time = None

    def record_success(self, response_time, db_time, status_code):
        with self.lock:
            self.response_times.append(response_time)
            self.db_times.append(db_time)
            self.successful_requests += 1
            self.status_codes[status_code] += 1

    def record_failure(self, status_code=0):
        with self.lock:
            self.failed_requests += 1
            self.status_codes[status_code] += 1

    def get_percentile(self, data, percentile):
        if not data:
            return 0
        sorted_data = sorted(data)
        index = int(len(sorted_data) * percentile)
        if index >= len(sorted_data):
            index = len(sorted_data) - 1
        return sorted_data[index]

    def print_summary(self, test_type):
        with self.lock:
            total_requests = self.successful_requests + self.failed_requests
            duration = (self.end_time - self.start_time) if self.end_time and self.start_time else 0

            print("\n" + "="*60)
            print(f"{test_type.upper()} TEST RESULTS")
            print("="*60)
            print(f"\nTest Duration: {duration:.2f} seconds")
            print(f"Total Requests: {total_requests}")
            print(f"Successful: {self.successful_requests}")
            print(f"Failed: {self.failed_requests}")
            print(f"Error Rate: {(self.failed_requests/total_requests*100) if total_requests > 0 else 0:.2f}%")

            if duration > 0:
                print(f"Actual Throughput: {total_requests/duration:.2f} req/sec")

            if self.response_times:
                print(f"\nResponse Time (ms):")
                print(f"  Min: {min(self.response_times):.2f}")
                print(f"  Max: {max(self.response_times):.2f}")
                print(f"  Avg: {statistics.mean(self.response_times):.2f}")
                print(f"  P50: {self.get_percentile(self.response_times, 0.50):.2f}")
                print(f"  P95: {self.get_percentile(self.response_times, 0.95):.2f}")
                print(f"  P99: {self.get_percentile(self.response_times, 0.99):.2f}")

            if self.db_times:
                print(f"\nDatabase Time (ms):")
                print(f"  Min: {min(self.db_times):.2f}")
                print(f"  Max: {max(self.db_times):.2f}")
                print(f"  Avg: {statistics.mean(self.db_times):.2f}")
                print(f"  P50: {self.get_percentile(self.db_times, 0.50):.2f}")
                print(f"  P95: {self.get_percentile(self.db_times, 0.95):.2f}")
                print(f"  P99: {self.get_percentile(self.db_times, 0.99):.2f}")

            print(f"\nStatus Codes:")
            for code, count in sorted(self.status_codes.items()):
                print(f"  {code}: {count}")

            print("="*60 + "\n")

    def get_summary_dict(self):
        """Return summary as dictionary for API calls"""
        with self.lock:
            total_requests = self.successful_requests + self.failed_requests
            duration = (self.end_time - self.start_time) if self.end_time and self.start_time else 0

            return {
                "total_requests": total_requests,
                "successful_requests": self.successful_requests,
                "failed_requests": self.failed_requests,
                "error_rate": (self.failed_requests/total_requests*100) if total_requests > 0 else 0,
                "duration": duration,
                "response_time_avg": statistics.mean(self.response_times) if self.response_times else 0,
                "response_time_p50": self.get_percentile(self.response_times, 0.50),
                "response_time_p95": self.get_percentile(self.response_times, 0.95),
                "response_time_p99": self.get_percentile(self.response_times, 0.99),
                "db_time_avg": statistics.mean(self.db_times) if self.db_times else 0,
                "db_time_p50": self.get_percentile(self.db_times, 0.50),
                "db_time_p95": self.get_percentile(self.db_times, 0.95),
                "db_time_p99": self.get_percentile(self.db_times, 0.99),
            }


def worker(endpoint, metrics, stop_event, requests_per_thread):
    """Worker thread that sends requests"""
    session = requests.Session()

    while not stop_event.is_set():
        try:
            start = time.time()
            response = session.post(
                endpoint,
                json=PAYLOAD,
                headers={"Content-Type": "application/json"},
                timeout=10
            )
            response_time = (time.time() - start) * 1000  # Convert to ms

            if response.status_code == 201:
                data = response.json()
                db_time = data.get('db_time', 0)
                metrics.record_success(response_time, db_time, response.status_code)
            else:
                metrics.record_failure(response.status_code)

        except requests.exceptions.Timeout:
            metrics.record_failure(408)  # Timeout
        except requests.exceptions.RequestException as e:
            metrics.record_failure(0)  # Connection error
        except Exception as e:
            metrics.record_failure(0)

        # Sleep to maintain target RPS
        time.sleep(1 / requests_per_thread)


def run_load_test(test_type, endpoint):
    """Run load test for a specific endpoint"""
    print(f"\n{'='*60}")
    print(f"Starting {test_type.upper()} Load Test")
    print(f"Target: {TARGET_RPS} req/sec for {TEST_DURATION} seconds")
    print(f"Endpoint: {endpoint}")
    print(f"{'='*60}\n")

    metrics = Metrics()
    stop_event = threading.Event()
    threads = []

    # Calculate requests per thread
    requests_per_thread = TARGET_RPS / THREADS

    # Start metrics
    metrics.start_time = time.time()

    # Start worker threads
    for i in range(THREADS):
        t = threading.Thread(
            target=worker,
            args=(endpoint, metrics, stop_event, requests_per_thread)
        )
        t.daemon = True
        t.start()
        threads.append(t)

    print(f"Started {THREADS} worker threads...")
    print(f"Running test for {TEST_DURATION} seconds...\n")

    # Progress indicator
    for i in range(TEST_DURATION):
        time.sleep(1)
        with metrics.lock:
            total = metrics.successful_requests + metrics.failed_requests
        print(f"Progress: {i+1}/{TEST_DURATION}s - Requests: {total}", end='\r')

    # Stop all threads
    print("\n\nStopping workers...")
    stop_event.set()
    metrics.end_time = time.time()

    # Wait for threads to finish
    for t in threads:
        t.join(timeout=2)

    # Print results
    metrics.print_summary(test_type)

    return metrics


def reset_metrics():
    """Reset server-side metrics"""
    try:
        response = requests.post(f"{API_BASE_URL}/api/load-test/metrics/reset")
        if response.status_code == 200:
            print("✓ Server metrics reset")
        else:
            print(f"⚠ Failed to reset metrics: {response.status_code}")
    except Exception as e:
        print(f"⚠ Error resetting metrics: {e}")


def get_server_metrics(test_type):
    """Get metrics from server"""
    try:
        response = requests.get(f"{API_BASE_URL}/api/load-test/metrics/{test_type}")
        if response.status_code == 200:
            return response.json()
        else:
            print(f"⚠ Failed to get server metrics: {response.status_code}")
            return None
    except Exception as e:
        print(f"⚠ Error getting server metrics: {e}")
        return None


def save_results(test_type, duration):
    """Save test results to database"""
    try:
        payload = {
            "test_type": test_type,
            "test_duration_seconds": int(duration),
            "notes": f"Python load test - {TARGET_RPS} req/sec for {TEST_DURATION}s"
        }
        response = requests.post(
            f"{API_BASE_URL}/api/load-test/results/save",
            json=payload,
            headers={"Content-Type": "application/json"}
        )
        if response.status_code == 201:
            data = response.json()
            print(f"✓ Results saved to database (ID: {data.get('result_id')})")
            return True
        else:
            print(f"⚠ Failed to save results: {response.status_code}")
            return False
    except Exception as e:
        print(f"⚠ Error saving results: {e}")
        return False


def cleanup_test_data():
    """Cleanup test data from server"""
    try:
        response = requests.delete(f"{API_BASE_URL}/api/load-test/cleanup")
        if response.status_code == 200:
            data = response.json()
            print(f"✓ Cleaned up {data.get('rows_deleted', 0)} test records")
        else:
            print(f"⚠ Failed to cleanup: {response.status_code}")
    except Exception as e:
        print(f"⚠ Error during cleanup: {e}")


def main():
    print("\n" + "="*60)
    print("MCQ API Load Testing Suite")
    print("="*60)
    print(f"Target: {TARGET_RPS} requests/second")
    print(f"Duration: {TEST_DURATION} seconds per test")
    print(f"Threads: {THREADS}")
    print(f"API: {API_BASE_URL}")
    print("="*60)

    # Test 1: Individual Insert
    print("\n\n[1/2] INDIVIDUAL INSERT TEST")
    reset_metrics()
    time.sleep(2)

    individual_metrics = run_load_test("individual", f"{API_BASE_URL}/api/load-test/individual")

    # Get server metrics
    print("\nFetching server-side metrics...")
    server_metrics = get_server_metrics("individual")
    if server_metrics:
        print("\nServer-side metrics:")
        print(json.dumps(server_metrics, indent=2))

    # Save results
    print("\nSaving results to database...")
    save_results("individual", individual_metrics.end_time - individual_metrics.start_time)

    # Cleanup between tests
    print("\nCleaning up test data...")
    cleanup_test_data()

    # Wait before next test
    print("\n\nWaiting 10 seconds before next test...")
    time.sleep(10)

    # Test 2: Batch Insert
    print("\n\n[2/2] BATCH INSERT TEST")
    reset_metrics()
    time.sleep(2)

    batch_metrics = run_load_test("batch", f"{API_BASE_URL}/api/load-test/batch")

    # Get server metrics
    print("\nFetching server-side metrics...")
    server_metrics = get_server_metrics("batch")
    if server_metrics:
        print("\nServer-side metrics:")
        print(json.dumps(server_metrics, indent=2))

    # Save results
    print("\nSaving results to database...")
    save_results("batch", batch_metrics.end_time - batch_metrics.start_time)

    # Final cleanup
    print("\nFinal cleanup...")
    cleanup_test_data()

    # Comparison Summary
    print("\n\n" + "="*60)
    print("COMPARISON SUMMARY")
    print("="*60)

    individual_summary = individual_metrics.get_summary_dict()
    batch_summary = batch_metrics.get_summary_dict()

    print(f"\n{'Metric':<30} {'Individual':<15} {'Batch':<15} {'Winner':<10}")
    print("-" * 70)
    print(f"{'Avg Response Time (ms)':<30} {individual_summary['response_time_avg']:<15.2f} {batch_summary['response_time_avg']:<15.2f} {'Batch' if batch_summary['response_time_avg'] < individual_summary['response_time_avg'] else 'Individual':<10}")
    print(f"{'P95 Response Time (ms)':<30} {individual_summary['response_time_p95']:<15.2f} {batch_summary['response_time_p95']:<15.2f} {'Batch' if batch_summary['response_time_p95'] < individual_summary['response_time_p95'] else 'Individual':<10}")
    print(f"{'P99 Response Time (ms)':<30} {individual_summary['response_time_p99']:<15.2f} {batch_summary['response_time_p99']:<15.2f} {'Batch' if batch_summary['response_time_p99'] < individual_summary['response_time_p99'] else 'Individual':<10}")
    print(f"{'Error Rate (%)':<30} {individual_summary['error_rate']:<15.2f} {batch_summary['error_rate']:<15.2f} {'Batch' if batch_summary['error_rate'] < individual_summary['error_rate'] else 'Individual':<10}")
    print(f"{'Successful Requests':<30} {individual_summary['successful_requests']:<15} {batch_summary['successful_requests']:<15} {'Batch' if batch_summary['successful_requests'] > individual_summary['successful_requests'] else 'Individual':<10}")

    print("\n" + "="*60)
    print("✓ All tests completed!")
    print("="*60 + "\n")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⚠ Test interrupted by user")
    except Exception as e:
        print(f"\n\n✗ Error: {e}")
        import traceback
        traceback.print_exc()

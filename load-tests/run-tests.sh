#!/bin/bash

# Load Test Runner Script
# Runs both individual and batch tests, compares results

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo "========================================="
echo "MCQ Load Testing Suite"
echo "========================================="
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}✗ k6 is not installed${NC}"
    echo ""
    echo "Please install k6 first:"
    echo "  macOS:   brew install k6"
    echo "  Linux:   See https://k6.io/docs/getting-started/installation/"
    echo "  Docker:  docker pull grafana/k6"
    exit 1
fi

# Check if server is running
echo -e "${YELLOW}Checking if server is running...${NC}"
if ! curl -s "$API_BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ Server is not running at $API_BASE_URL${NC}"
    echo ""
    echo "Please start the server first:"
    echo "  go run main.go"
    echo "  OR"
    echo "  docker-compose up"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"
echo ""

# Reset metrics
echo -e "${YELLOW}Resetting metrics...${NC}"
curl -s -X POST "$API_BASE_URL/api/load-test/metrics/reset" > /dev/null
echo -e "${GREEN}✓ Metrics reset${NC}"
echo ""

# Ask which test to run
echo "Which test would you like to run?"
echo "  1) Individual insert test only"
echo "  2) Batch insert test only"
echo "  3) Both tests (recommended for comparison)"
echo "  4) Exit"
echo ""
read -p "Enter your choice (1-4): " choice

case $choice in
    1)
        echo ""
        echo -e "${BLUE}=========================================${NC}"
        echo -e "${BLUE}Running Individual Insert Test${NC}"
        echo -e "${BLUE}=========================================${NC}"
        echo ""
        k6 run individual-test.js
        echo ""
        echo -e "${YELLOW}Fetching application metrics...${NC}"
        curl -s "$API_BASE_URL/api/load-test/metrics/individual" | jq
        ;;
    2)
        echo ""
        echo -e "${BLUE}=========================================${NC}"
        echo -e "${BLUE}Running Batch Insert Test${NC}"
        echo -e "${BLUE}=========================================${NC}"
        echo ""
        k6 run batch-test.js
        echo ""
        echo -e "${YELLOW}Fetching application metrics...${NC}"
        curl -s "$API_BASE_URL/api/load-test/metrics/batch" | jq
        ;;
    3)
        # Run individual test
        echo ""
        echo -e "${BLUE}=========================================${NC}"
        echo -e "${BLUE}Running Individual Insert Test (1/2)${NC}"
        echo -e "${BLUE}=========================================${NC}"
        echo ""
        k6 run individual-test.js

        # Wait a bit
        echo ""
        echo -e "${YELLOW}Waiting 10 seconds before next test...${NC}"
        sleep 10

        # Cleanup between tests
        echo -e "${YELLOW}Cleaning up test data...${NC}"
        curl -s -X DELETE "$API_BASE_URL/api/load-test/cleanup" > /dev/null
        echo -e "${GREEN}✓ Test data cleaned${NC}"
        echo ""

        # Run batch test
        echo ""
        echo -e "${BLUE}=========================================${NC}"
        echo -e "${BLUE}Running Batch Insert Test (2/2)${NC}"
        echo -e "${BLUE}=========================================${NC}"
        echo ""
        k6 run batch-test.js

        # Show comparison
        echo ""
        echo -e "${BLUE}=========================================${NC}"
        echo -e "${BLUE}Performance Comparison${NC}"
        echo -e "${BLUE}=========================================${NC}"
        echo ""

        echo -e "${YELLOW}Individual Insert Metrics:${NC}"
        curl -s "$API_BASE_URL/api/load-test/metrics/individual" | jq
        echo ""

        echo -e "${YELLOW}Batch Insert Metrics:${NC}"
        curl -s "$API_BASE_URL/api/load-test/metrics/batch" | jq
        echo ""
        ;;
    4)
        echo "Exiting..."
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Test Complete!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "Next steps:"
echo "  - Check result JSON files: *-test-results.json"
echo "  - View metrics: curl $API_BASE_URL/api/load-test/metrics/{individual|batch}"
echo "  - Cleanup: ./cleanup.sh"
echo ""

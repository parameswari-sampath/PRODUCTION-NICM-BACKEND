#!/bin/bash

# Load Test Cleanup Script
# This script cleans up all test data and resets metrics

echo "========================================="
echo "Load Test Cleanup Script"
echo "========================================="
echo ""

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Cleaning up test data...${NC}"
cleanup_response=$(curl -s -X DELETE "$API_BASE_URL/api/load-test/cleanup")
cleanup_status=$?

if [ $cleanup_status -eq 0 ]; then
    rows_deleted=$(echo $cleanup_response | grep -o '"rows_deleted":[0-9]*' | grep -o '[0-9]*')
    echo -e "${GREEN}✓ Test data cleaned up successfully${NC}"
    echo -e "  Rows deleted: $rows_deleted"
else
    echo -e "${RED}✗ Failed to cleanup test data${NC}"
    echo "  Response: $cleanup_response"
fi

echo ""
echo -e "${YELLOW}Resetting metrics...${NC}"
reset_response=$(curl -s -X POST "$API_BASE_URL/api/load-test/metrics/reset")
reset_status=$?

if [ $reset_status -eq 0 ]; then
    echo -e "${GREEN}✓ Metrics reset successfully${NC}"
else
    echo -e "${RED}✗ Failed to reset metrics${NC}"
    echo "  Response: $reset_response"
fi

echo ""
echo -e "${YELLOW}Removing result files...${NC}"
rm -f individual-test-results.json batch-test-results.json

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Result files removed${NC}"
else
    echo -e "${YELLOW}⚠ No result files found${NC}"
fi

echo ""
echo "========================================="
echo -e "${GREEN}Cleanup Complete!${NC}"
echo "========================================="

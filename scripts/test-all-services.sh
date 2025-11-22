#!/bin/bash
# Comprehensive Testing Script for All Services

set -e

FAILED_TESTS=()

echo "=== Starting Comprehensive Service Tests ==="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run tests for a service
run_service_tests() {
  local service=$1
  local test_dir=$2
  local test_cmd=$3

  echo -e "\n${YELLOW}Testing $service...${NC}"

  cd "$test_dir" || return 1

  if $test_cmd; then
    echo -e "${GREEN}✓ $service tests passed${NC}"
    return 0
  else
    echo -e "${RED}✗ $service tests failed${NC}"
    FAILED_TESTS+=("$service")
    return 1
  fi
}

# Test Auth Service
run_service_tests "Auth Service" "services/auth" "go test ./... -v -coverprofile=coverage.out" || true

# Test Session Service
run_service_tests "Session Service" "services/session" "go test ./... -v -coverprofile=coverage.out" || true

# Test Asset Service
run_service_tests "Asset Service" "services/asset" "go test ./... -v -coverprofile=coverage.out" || true

# Test Presence Service
run_service_tests "Presence Service" "services/presence" "go test ./... -v -coverprofile=coverage.out" || true

# Test Sync Service
cd services/sync
run_service_tests "Sync Service" "services/sync" "npm test" || true

# Test Analytics Service
cd ../analytics
run_service_tests "Analytics Service" "services/analytics" "pytest tests/ -v --cov=." || true

# Test Voice Service
cd ../voice
run_service_tests "Voice Service" "services/voice" "npm test" || true

# Integration Tests
echo -e "\n${YELLOW}Running Integration Tests...${NC}"
cd ../../tests/integration

if ./run-integration-tests.sh; then
  echo -e "${GREEN}✓ Integration tests passed${NC}"
else
  echo -e "${RED}✗ Integration tests failed${NC}"
  FAILED_TESTS+=("Integration Tests")
fi

# E2E Tests
echo -e "\n${YELLOW}Running E2E Tests...${NC}"
cd ../e2e

if ./run-e2e-tests.sh; then
  echo -e "${GREEN}✓ E2E tests passed${NC}"
else
  echo -e "${RED}✗ E2E tests failed${NC}"
  FAILED_TESTS+=("E2E Tests")
fi

# Summary
echo -e "\n=== Test Summary ==="

if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
  echo -e "${GREEN}All tests passed! ✓${NC}"
  exit 0
else
  echo -e "${RED}Failed tests:${NC}"
  for test in "${FAILED_TESTS[@]}"; do
    echo -e "${RED}  - $test${NC}"
  done
  exit 1
fi

#!/bin/bash

# Test runner script for DungeoGo
set -e

echo "Running DungeoGo test suite..."
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to run tests for a package
run_tests() {
    local package=$1
    local name=$2
    
    echo -e "${BLUE}Testing $name...${NC}"
    if go test "./$package" -v; then
        echo -e "${GREEN}✓ $name tests passed${NC}"
    else
        echo -e "${RED}✗ $name tests failed${NC}"
        return 1
    fi
    echo
}

# Function to run tests with coverage
run_tests_with_coverage() {
    local package=$1
    local name=$2
    
    echo -e "${BLUE}Testing $name with coverage...${NC}"
    if go test "./$package" -coverprofile="coverage_$(basename $package).out" -v; then
        echo -e "${GREEN}✓ $name tests passed${NC}"
        go tool cover -func="coverage_$(basename $package).out" | tail -1
    else
        echo -e "${RED}✗ $name tests failed${NC}"
        return 1
    fi
    echo
}

# Check if coverage flag is provided
COVERAGE=false
if [[ "$1" == "--coverage" || "$1" == "-c" ]]; then
    COVERAGE=true
    echo -e "${YELLOW}Running tests with coverage analysis${NC}"
    echo
fi

# Run unit tests
echo -e "${YELLOW}Running unit tests...${NC}"
echo

if [ "$COVERAGE" = true ]; then
    run_tests_with_coverage "pkg/game/character" "Character System"
    run_tests_with_coverage "pkg/game/items" "Item System" 
    run_tests_with_coverage "pkg/commands" "Command System"
else
    run_tests "pkg/game/character" "Character System"
    run_tests "pkg/game/items" "Item System"
    run_tests "pkg/commands" "Command System"
fi

# Build test
echo -e "${BLUE}Testing build...${NC}"
if go build ./cmd/server; then
    echo -e "${GREEN}✓ Build successful${NC}"
    rm -f server # Clean up binary
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo

# Check for race conditions (if requested)
if [[ "$1" == "--race" || "$2" == "--race" ]]; then
    echo -e "${YELLOW}Running race condition tests...${NC}"
    go test -race ./pkg/game/character ./pkg/game/items ./pkg/commands
    echo
fi

# Summary
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}All tests completed successfully!${NC}"

if [ "$COVERAGE" = true ]; then
    echo
    echo -e "${YELLOW}Coverage files generated:${NC}"
    ls -la coverage_*.out 2>/dev/null || echo "No coverage files found"
    
    echo
    echo -e "${YELLOW}To view detailed coverage:${NC}"
    echo "go tool cover -html=coverage_character.out"
    echo "go tool cover -html=coverage_items.out" 
    echo "go tool cover -html=coverage_commands.out"
fi

echo
echo -e "${BLUE}To run tests manually:${NC}"
echo "go test ./pkg/game/character -v    # Character tests"
echo "go test ./pkg/game/items -v        # Item tests" 
echo "go test ./pkg/commands -v          # Command tests (parser only, no DB)"
echo
echo -e "${BLUE}For database integration tests:${NC}"
echo "./test-with-db.sh                      # Run all tests with containerized PostgreSQL"
echo "./test-with-db.sh -c                   # Run with coverage analysis"
echo "./test-with-db.sh ./pkg/persistence... # Run specific database tests"
echo
echo -e "${BLUE}Manual database testing:${NC}"
echo "docker-compose -f docker-compose.test.yml up -d  # Start test DB"
echo "go test ./pkg/persistence/postgres -v            # Run persistence tests"
echo "go test ./pkg/integration -v                     # Run integration tests"
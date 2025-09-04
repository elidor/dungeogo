#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.test.yml"
CONTAINER_NAME="dungeogo-test-db"
MAX_WAIT_TIME=30
POSTGRES_PORT=5433

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
}

# Function to check if PostgreSQL container is already running
is_container_running() {
    docker ps --format "table {{.Names}}" | grep -q "$CONTAINER_NAME"
}

# Function to check if PostgreSQL is ready to accept connections
wait_for_postgres() {
    local wait_time=0
    print_status "Waiting for PostgreSQL to be ready..."
    
    while [ $wait_time -lt $MAX_WAIT_TIME ]; do
        if docker exec "$CONTAINER_NAME" pg_isready -U testuser -d postgres &> /dev/null; then
            print_success "PostgreSQL is ready!"
            return 0
        fi
        sleep 1
        wait_time=$((wait_time + 1))
        printf "."
    done
    
    echo
    print_error "PostgreSQL failed to start within $MAX_WAIT_TIME seconds"
    return 1
}

# Function to start the database container
start_database() {
    print_status "Starting PostgreSQL test container..."
    
    if is_container_running; then
        print_warning "PostgreSQL container is already running"
        return 0
    fi
    
    # Check if we have docker-compose or docker compose
    if command -v docker-compose &> /dev/null; then
        docker-compose -f "$COMPOSE_FILE" up -d
    else
        docker compose -f "$COMPOSE_FILE" up -d
    fi
    
    if [ $? -ne 0 ]; then
        print_error "Failed to start PostgreSQL container"
        exit 1
    fi
    
    # Wait for PostgreSQL to be ready
    if ! wait_for_postgres; then
        print_error "PostgreSQL startup failed"
        stop_database
        exit 1
    fi
}

# Function to stop the database container
stop_database() {
    print_status "Stopping PostgreSQL test container..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose -f "$COMPOSE_FILE" down -v
    else
        docker compose -f "$COMPOSE_FILE" down -v
    fi
    
    print_success "PostgreSQL container stopped"
}

# Function to run tests
run_tests() {
    print_status "Running tests with database..."
    
    # Set environment variable to indicate database is available
    export POSTGRES_TEST_URL="postgres://testuser:testpass@localhost:$POSTGRES_PORT/postgres?sslmode=disable"
    
    # Run the tests
    if [ "$#" -eq 0 ]; then
        # No specific package specified, run all tests
        go test -v ./...
    else
        # Run tests for specific packages
        go test -v "$@"
    fi
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        print_success "All tests passed!"
    else
        print_error "Some tests failed (exit code: $exit_code)"
    fi
    
    return $exit_code
}

# Function to run tests with coverage
run_tests_with_coverage() {
    print_status "Running tests with coverage analysis..."
    
    export POSTGRES_TEST_URL="postgres://testuser:testpass@localhost:$POSTGRES_PORT/postgres?sslmode=disable"
    
    # Create coverage directory if it doesn't exist
    mkdir -p coverage
    
    # Run tests with coverage
    go test -v -coverprofile=coverage/coverage.out ./...
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        print_success "Tests completed successfully!"
        
        # Generate HTML coverage report
        go tool cover -html=coverage/coverage.out -o coverage/coverage.html
        print_success "Coverage report generated: coverage/coverage.html"
        
        # Show coverage summary
        print_status "Coverage summary:"
        go tool cover -func=coverage/coverage.out | tail -1
    else
        print_error "Some tests failed (exit code: $exit_code)"
    fi
    
    return $exit_code
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] [TEST_PACKAGES...]"
    echo ""
    echo "Options:"
    echo "  -h, --help        Show this help message"
    echo "  -c, --coverage    Run tests with coverage analysis"
    echo "  -k, --keep        Keep the database container running after tests"
    echo "  -s, --stop        Stop the database container and exit"
    echo "  --start-only      Start the database container and exit"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run all tests"
    echo "  $0 ./pkg/persistence/postgres        # Run specific package tests"
    echo "  $0 -c                                 # Run all tests with coverage"
    echo "  $0 -k ./pkg/integration              # Run integration tests, keep DB running"
    echo "  $0 -s                                 # Stop database container"
    echo ""
}

# Parse command line arguments
COVERAGE=false
KEEP_RUNNING=false
STOP_ONLY=false
START_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -k|--keep)
            KEEP_RUNNING=true
            shift
            ;;
        -s|--stop)
            STOP_ONLY=true
            shift
            ;;
        --start-only)
            START_ONLY=true
            shift
            ;;
        *)
            break
            ;;
    esac
done

# Main execution
main() {
    print_status "DungeoGo Test Runner with PostgreSQL"
    echo "=================================="
    
    # Check prerequisites
    check_docker
    
    # Handle special modes
    if [ "$STOP_ONLY" = true ]; then
        stop_database
        exit 0
    fi
    
    # Start database
    start_database
    
    if [ "$START_ONLY" = true ]; then
        print_success "Database container started and ready"
        print_status "Container will keep running. Use '$0 -s' to stop it."
        exit 0
    fi
    
    # Run tests
    local test_exit_code=0
    if [ "$COVERAGE" = true ]; then
        run_tests_with_coverage "$@"
        test_exit_code=$?
    else
        run_tests "$@"
        test_exit_code=$?
    fi
    
    # Stop database unless requested to keep running
    if [ "$KEEP_RUNNING" = false ]; then
        stop_database
    else
        print_warning "Database container is still running. Use '$0 -s' to stop it."
    fi
    
    exit $test_exit_code
}

# Trap to ensure cleanup on script termination
cleanup() {
    if [ "$KEEP_RUNNING" = false ]; then
        print_status "Cleaning up..."
        stop_database
    fi
}

trap cleanup INT TERM

# Run main function
main "$@"
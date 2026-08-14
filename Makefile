# DungeoGo Makefile

.PHONY: test test-unit test-db test-coverage build clean help docker-up docker-down run apple-build apple-up apple-down apple-logs apple-status test-db-apple

# Default target
help:
	@echo "DungeoGo MUD Server - Available targets:"
	@echo ""
	@echo "Testing:"
	@echo "  test           Run unit tests (no database required)"
	@echo "  test-db        Run all tests with PostgreSQL container"
	@echo "  test-coverage  Run tests with coverage analysis"
	@echo "  test-db-only   Run only database-dependent tests"
	@echo ""
	@echo "Development:"
	@echo "  build          Build the server binary"
	@echo "  run            Build and run the server"
	@echo "  clean          Clean build artifacts and coverage files"
	@echo ""
	@echo "Database:"
	@echo "  docker-up      Start PostgreSQL test container"
	@echo "  docker-down    Stop PostgreSQL test container"
	@echo "  apple-build    Build the server image with Apple Container"
	@echo "  apple-up       Start PostgreSQL and the server with Apple Container"
	@echo "  apple-down     Stop the Apple Container stack (preserves database data)"
	@echo "  apple-logs     Show Apple Container server logs"
	@echo "  test-db-apple  Run database tests with Apple Container"
	@echo ""

# Unit tests (no database required)
test:
	@echo "Running unit tests..."
	go test -v ./pkg/game/... ./pkg/commands

# Unit tests with basic script
test-unit:
	./test.sh

# All tests with database container
test-db:
	./test-with-db.sh

# Tests with coverage analysis
test-coverage:
	./test-with-db.sh -c

# Only database-dependent tests
test-db-only:
	./test-with-db.sh ./pkg/persistence/postgres ./pkg/integration

# Build the server
build:
	@echo "Building DungeoGo server..."
	go build -o bin/dungeogo ./cmd/server

# Build and run the server
run: build
	./ensure-dev-db.sh
	./bin/dungeogo

# Clean build artifacts
clean:
	rm -f bin/dungeogo
	rm -f server
	rm -f coverage/*.out
	rm -f coverage/*.html
	rm -f coverage_*.out

# Docker management
docker-up:
	./test-with-db.sh --start-only

docker-down:
	./test-with-db.sh -s

# Apple Container management (macOS 26+ on Apple silicon)
apple-build:
	./apple-container.sh build

apple-up:
	./apple-container.sh up

apple-down:
	./apple-container.sh down

apple-logs:
	./apple-container.sh logs

apple-status:
	./apple-container.sh status

test-db-apple:
	DUNGEOGO_CONTAINER_RUNTIME=apple ./test-with-db.sh

# Continuous testing (watch for changes)
test-watch:
	@echo "Watching for changes... (requires 'entr' tool)"
	find . -name "*.go" | entr -c make test

# Check for common issues
lint:
	@echo "Running go fmt..."
	go fmt ./...
	@echo "Running go vet..."
	go vet ./...
	@if command -v golint >/dev/null 2>&1; then \
		echo "Running golint..."; \
		golint ./...; \
	fi

# Install development dependencies
dev-deps:
	go install golang.org/x/lint/golint@latest

# Full check (format, vet, test)
check: lint test

# Show test coverage in browser
coverage-html: test-coverage
	@if [ -f coverage/coverage.html ]; then \
		echo "Opening coverage report in browser..."; \
		open coverage/coverage.html || xdg-open coverage/coverage.html || echo "Please open coverage/coverage.html manually"; \
	else \
		echo "No coverage report found. Run 'make test-coverage' first."; \
	fi

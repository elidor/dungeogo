# DungeoGo Makefile

.PHONY: test test-unit test-db test-coverage build clean help docker-up docker-down

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
	docker-compose -f docker-compose.test.yml up -d

docker-down:
	docker-compose -f docker-compose.test.yml down -v

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
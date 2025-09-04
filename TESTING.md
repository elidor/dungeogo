# Testing Guide

This document describes how to run tests for the DungeoGo MUD server, including database integration tests.

## Overview

The DungeoGo test suite includes:

- **Unit tests**: Core game logic (character, items, player, commands)
- **Persistence layer tests**: Database repository operations 
- **Integration tests**: Server components and end-to-end workflows

## Prerequisites

### For Basic Tests
- Go 1.19 or later
- Basic tests will run without any external dependencies

### For Database Tests
- Docker and Docker Compose
- PostgreSQL tests use a containerized database

## Running Tests

### Quick Start - All Tests
```bash
# Run all tests (will skip database tests if no PostgreSQL available)
go test -v ./...

# Run tests with containerized PostgreSQL database
./test-with-db.sh
```

### Database Tests with Container

The `test-with-db.sh` script automatically manages a PostgreSQL test container:

```bash
# Run all tests with database
./test-with-db.sh

# Run specific package tests
./test-with-db.sh ./pkg/persistence/postgres

# Run with coverage analysis
./test-with-db.sh -c

# Keep database running after tests (useful for debugging)
./test-with-db.sh -k

# Start database container only
./test-with-db.sh --start-only

# Stop database container
./test-with-db.sh -s
```

### Manual Database Setup

If you prefer to manage PostgreSQL yourself:

```bash
# Start the test database container
docker-compose -f docker-compose.test.yml up -d

# Run tests
go test -v ./...

# Stop the container
docker-compose -f docker-compose.test.yml down -v
```

## Test Categories

### Unit Tests
- **Location**: `pkg/game/character`, `pkg/game/items`, `pkg/game/player`, `pkg/commands`
- **Dependencies**: None
- **Run with**: `go test -v ./pkg/game/... ./pkg/commands/...`

### Persistence Tests
- **Location**: `pkg/persistence/postgres`
- **Dependencies**: PostgreSQL database
- **Run with**: `./test-with-db.sh ./pkg/persistence/postgres`

### Integration Tests  
- **Location**: `pkg/integration`
- **Dependencies**: PostgreSQL database (for some tests)
- **Run with**: `./test-with-db.sh ./pkg/integration`

## Database Test Configuration

### Container Configuration
- **Image**: `postgres:15-alpine`
- **Port**: `5433` (avoids conflicts with local PostgreSQL)
- **Database**: `postgres`
- **Username**: `testuser`
- **Password**: `testpass`

### Connection Fallback
Tests automatically try:
1. Containerized PostgreSQL (`localhost:5433`)
2. Local PostgreSQL (`localhost:5432`)
3. Skip database tests if neither available

## Coverage Analysis

Generate test coverage reports:

```bash
# Run tests with coverage
./test-with-db.sh -c

# View HTML coverage report
open coverage/coverage.html
```

## Troubleshooting

### Database Connection Issues
```bash
# Check if container is running
docker ps | grep dungeogo-test-db

# Check container logs
docker logs dungeogo-test-db

# Restart container
./test-with-db.sh -s  # stop
./test-with-db.sh --start-only  # start
```

### Port Conflicts
If port 5433 is in use, modify `docker-compose.test.yml`:
```yaml
ports:
  - "5434:5432"  # Use different port
```

### Docker Issues
```bash
# Clean up all containers and volumes
docker system prune -f
docker volume prune -f

# Rebuild test container
docker-compose -f docker-compose.test.yml build --no-cache
```

## Test Structure

### Test Helper Functions
- **Location**: `pkg/persistence/postgres/test_helpers_test.go`
- **Purpose**: Database setup, test data creation, cleanup
- **Shared across**: All persistence layer tests

### Test Data
- **Players**: Test accounts with various configurations
- **Characters**: Test characters with different races/classes
- **Items**: Test item instances with enchantments
- **Database**: Isolated test databases per test run

## Continuous Integration

For CI environments, the test script provides:
- Automatic container lifecycle management
- Proper cleanup on test failure
- Exit codes for build systems
- Detailed logging for debugging

Example CI usage:
```yaml
- name: Run tests with database
  run: ./test-with-db.sh -c
```
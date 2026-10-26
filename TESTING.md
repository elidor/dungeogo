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
## Required PostgreSQL verification

Run against an existing PostgreSQL server whose test role has `CREATEDB`:

```bash
TEST_DATABASE_URL='postgres://testuser:testpass@localhost:5433/postgres?sslmode=disable' ./verify-db.sh
```

This command disables test caching and sets `DUNGEOGO_REQUIRE_DB=1`. An unavailable
server, inability to create a database, migration failure, or test failure returns
nonzero. `POSTGRES_TEST_URL` is also accepted for the existing container scripts;
`TEST_DATABASE_URL` takes precedence. Each test creates a distinct disposable
database and runs the embedded production SQL migrations, then drops that database.
The URL's database is used only for administration; its tables are never reset.
Ordinary `go test ./...` may skip database tests when PostgreSQL is unavailable.
`./test-with-db.sh` provisions PostgreSQL and also requires database tests to run.

Migration coverage includes a fresh database, unversioned upgrades from migrations
001 and 002 with existing player/character/item/room state, incompatible schema
rejection, repeat startup without reseeding, missing spawn, and broken exits.
Movement tests use the migrated starter grid and check persisted locations,
blocked exits, and failed database writes.

Session location tests (`TestSessionLocation` in `pkg/integration`) exercise the
real TCP handler: guided creation, configured spawn, movement between zones,
blocked exits, failed movement writes, abrupt disconnect, reconnect with a fresh
engine, graceful quit, valid saved locations, missing/empty/invalid locations,
and read/write failures during entry. These tests wait for session cleanup before
dropping their database. Run them with the race detector using:

```bash
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./pkg/integration -run TestSessionLocation
```

## Online presence and replacement sessions

`TestPresenceSession` uses TCP sessions and PostgreSQL to verify actual `who`
output, failed authentication, replacement before character selection, stale
socket cleanup, ordinary disconnects, new-account registration, and concurrent
successful logins. `pkg/server` registry tests also cover rejected stale actions,
in-flight action ordering, concurrent replacement, immutable presence snapshots,
and closing a socket with a stalled writer.

```bash
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./pkg/server ./pkg/integration
```

## Room presence and movement delivery

`TestRoomPresence` uses four TCP clients across three rooms to check occupant
lists, departure/arrival scope, blocked exits, failed persistence, replacement
before character selection, reconnect, and disconnect cleanup. An injected
server-side socket write failure verifies that committed movement survives a
failed delivery and is restored on reconnect.

```bash
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./pkg/server ./pkg/integration ./pkg/commands
```

World actions return addressed results in `pkg/game/world`. Legacy command APIs
still return actor-only text; the TCP handler resolves room effects through live
presence and owns socket delivery.

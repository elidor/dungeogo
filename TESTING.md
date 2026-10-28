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

## Room-local conversation

`TestLocalCommunication` places two TCP characters together and a third in a
different room. It verifies say/emote sender feedback, recipient identity and
content (including Unicode), aliases, no duplicate or cross-room delivery,
movement, disconnect, and reconnect. Failure cases cover empty messages, missing
or invalid persisted locations, inconsistent live membership, and unavailable
room storage; none may fabricate a broadcast.

```bash
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./pkg/integration -run TestLocalCommunication
```

## Private messages

`TestPrivateTell` uses real TCP sessions and PostgreSQL to verify cross-room tell,
character-name resolution rather than account usernames, case-insensitive matches,
syntax failures, offline targets, self-tell, reconnect, and bystander isolation.
A paused engine deterministically disconnects or replaces the recipient between
resolution and delivery, including replacement with the same or a different
character. A failed socket write must report failure to the sender. Domain tests
cover exact-case disambiguation and delayed success confirmation.

```bash
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./pkg/server ./pkg/integration ./pkg/commands ./pkg/game/actions
```

## Item ownership and starter seeds

The real migration sequence now includes `004_item_ownership.sql`,
`005_starter_items.sql` and `006_item_owner_fk_indexes.sql`, which indexes the
generated owner columns behind the item foreign keys so deleting a room or
character doesn't scan every item, and `007_character_deletion_destroys_items.sql`,
which makes deleting a Character destroy the Items it holds (ADR-0001) while rooms
holding items still can't be deleted. PostgreSQL tests cover valid versioned and unversioned
upgrades, preservation of legacy fields and nullable names/timestamps, atomic
failure for missing or ambiguous owners, foreign-key rejection of invalid owners,
and distinct room/character queries even when IDs overlap.

Seed tests move one starter item and delete another, reopen repository connections,
and repeat startup to prove no ownership reset, duplicate seeding, or automatic
respawn. TCP tests restart the repository and game handler, then verify room items,
inventory names and IDs, room isolation, and honest errors on unavailable storage.
Existing test fixtures now create real owners rather than placeholder strings.

```bash
./verify-db.sh
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 -timeout=120s ./...
```

## Picking up and dropping items

`TestItemActions_GetDropTargetsMessagesAndPersistence` drives `get`, `take` and
`drop` over TCP with an observer in the same room and one in another room. It
covers actor and room wording, numbered targets counted in arrival order, missing,
out-of-range and malformed targets leaving ownership unchanged, dropping after
moving rooms, and inventory after reconnecting.
`TestItemActions_CompetingPickupsExactlyOneWins` has two clients send `get gem`
for the same item simultaneously, ten times; exactly one wins each round and the
other gets a clear refusal. `TestGetReportsLostRaceWithoutRoomMessage` covers the
reply when another writer moves the item between listing and transfer, and
`TestGetStorageFailureChangesNothingAndIsLogged` a failed write: no room message,
a logged item ID. `TestItemActions_OwnershipSurvivesServerRestart` restarts the
repository and session handler and checks room and inventory ownership.

`TestGive_ExchangeMessagesRejectionsAndRestart` drives `give` over TCP with the
recipient, a bystander and someone in another room: each audience's wording,
numbered targets, every rejection (missing item, other room, unknown, self, wrong
syntax, offline recipient) leaving ownership unchanged, and inventories after a
server restart. `TestGive_RacesWithRecipientMovingOrLeaving` gives while the
recipient walks out or disconnects; whichever wins, the giver is told "You give"
exactly when the item now belongs to the recipient.
`TestOwnedItemsListInArrivalOrder` checks that an item which leaves and returns
is listed last (migration `008_item_arrival_order.sql`).

```bash
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./pkg/integration ./pkg/game/... ./pkg/persistence/postgres
```

## Session lifecycle and shutdown

`pkg/server/connection_manager_test.go` covers the connection manager with stub
handlers under the race detector: shutdown stops accepting, closes every session
and returns only after each handler has finished (including in-flight work);
an overrunning session makes `Shutdown` return an error naming it; a disconnected
session leaves the client list immediately rather than at the next sweep; an idle
session is closed while an active one survives; and a full server turns away
extra connections. `TestLifecycle_IdleExpiryRemovesPresenceAndShutdownWaits` runs
the real session handler behind the connection manager: an idle character drops
out of `who` while an active one stays, then shutdown closes the rest.
`cmd/server/main_test.go` sends the process SIGTERM and checks `serve` closes the
database and exits 0, or 1 when closing fails.

```bash
go test -race -count=1 ./pkg/server ./cmd/server
DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 -run TestLifecycle ./pkg/integration
```

# DungeoGo

A modern Multi-User Dungeon (MUD) server written in Go with comprehensive testing and PostgreSQL persistence.

## Overview

DungeoGo is a text-based multiplayer game server that implements classic MUD mechanics with modern architecture patterns. Players can create characters, explore a virtual world, interact with items, and communicate with other players through a TCP-based interface.

## Features

### Core Game Systems
- **Player Management**: Account creation, authentication, and preferences
- **Character System**: Multiple races (Human, Elf, Dwarf) and classes (Warrior, Rogue, Mage)
- **Skills & Progression**: Experience-based skill system with leveling mechanics
- **Item System**: Template-based items with enchantments, durability, and modifications
- **Command Processing**: Flexible command parser with aliases and validation
- **Real-time Communication**: Say, tell, whisper, and emote commands

### Technical Features
- **PostgreSQL Integration**: Full persistence with JSONB for complex data
- **TCP Server**: Multi-client connection handling with session management
- **Repository Pattern**: Clean separation between business logic and data access
- **Comprehensive Testing**: 97+ tests with containerized PostgreSQL for integration testing
- **Container Support**: Docker/Compose and Apple Container workflows for local PostgreSQL and server runs

## Quick Start

### Prerequisites
- Go 1.19 or later
- Docker and Docker Compose (for database tests)
- Or, on Apple silicon with macOS 26+, [Apple Container](https://github.com/apple/container)
- PostgreSQL (optional, for local development)

### Installation
```bash
git clone https://github.com/your-username/dungeogo.git
cd dungeogo
go mod download
```

### Running Tests
```bash
# Unit tests (no database required)
make test

# All tests with containerized PostgreSQL
make test-db

# Tests with coverage analysis
make test-coverage

# Manual container management
make docker-up    # Start PostgreSQL container
make docker-down  # Stop PostgreSQL container

# Apple Container: run PostgreSQL and the MUD server as OCI containers
make apple-up
make apple-logs
make apple-down
```

### Building and Running
```bash
# Build the server
make build

# Run the server (requires PostgreSQL configuration)
make run
```

### Apple Container (macOS Tahoe)

Apple Container supports OCI images but does not use Docker Compose. This project
includes a `Containerfile` and a small orchestration script that starts PostgreSQL
and the game server on an isolated Apple Container network.

```bash
# Install Apple Container, then initialize its services once.
container system start

# Build and start the complete stack.
make apple-up

# The TCP server is published only to localhost:8080.
nc 127.0.0.1 8080

# Inspect logs or stop the stack. The PostgreSQL data volume is preserved on stop.
make apple-logs
make apple-down
```

The server image is tagged `dungeogo:local` by default. You can override ports or
resource names, for example: `PORT=9090 POSTGRES_PORT=5434 make apple-up`.

To use Apple Container for the database-backed test suite while keeping Go tests
on the host:

```bash
make test-db-apple
```

## Architecture

### Project Structure
```
dungeogo/
├── cmd/server/           # Application entry point
├── pkg/
│   ├── commands/         # Command system (parser, executor)
│   ├── game/            # Core game logic
│   │   ├── character/   # Character, race, class, skills
│   │   ├── items/       # Item system with enchantments
│   │   └── player/      # Player accounts and preferences
│   ├── persistence/     # Data access layer
│   │   ├── interfaces/  # Repository interfaces
│   │   └── postgres/    # PostgreSQL implementation
│   ├── server/          # TCP server and client handling
│   └── testutil/        # Testing utilities
├── test-with-db.sh      # Database test runner
└── docker-compose.test.yml  # Test database container
```

### Key Components

**Game Engine**
- Processes player commands and manages game state
- Integrates with persistence layer for data consistency
- Handles player authentication and session management

**Command System**
- Flexible parser supporting aliases and complex arguments
- Type-safe command validation and execution
- Extensible architecture for adding new commands

**Persistence Layer**
- Repository pattern with interface-based design
- PostgreSQL with JSONB for flexible schema
- Comprehensive CRUD operations with proper error handling

**TCP Server**
- Multi-client connection pooling
- Session state tracking and timeout handling
- Graceful client disconnection and cleanup

## Testing

The project includes comprehensive testing across multiple levels:

### Test Categories
- **Unit Tests (70 tests)**: Core game logic without external dependencies
- **Persistence Tests (22 tests)**: Database operations with PostgreSQL
- **Integration Tests (5 tests)**: End-to-end server functionality

### Database Testing
Tests automatically use containerized PostgreSQL when Docker is available, with fallback to local PostgreSQL or graceful skipping:

```bash
# Full test suite with database
./test-with-db.sh

# Specific test categories  
./test-with-db.sh ./pkg/persistence/postgres
./test-with-db.sh ./pkg/integration

# Keep database running for debugging
./test-with-db.sh -k

# Coverage analysis with HTML reports
./test-with-db.sh -c
```

### Test Infrastructure Features
- **Automatic Container Management**: Start, stop, and cleanup PostgreSQL containers
- **Isolated Test Databases**: Each test run gets a unique database
- **Comprehensive Cleanup**: Proper resource management and cleanup
- **Coverage Reporting**: HTML coverage reports with detailed analysis

## Development

### Available Commands
```bash
make help           # Show all available commands
make test           # Run unit tests only
make test-db        # Run all tests with database
make test-coverage  # Generate coverage reports
make build          # Build server binary
make clean          # Clean build artifacts
make lint           # Run formatting and linting
make docker-up      # Start test database
make docker-down    # Stop test database
make test-db-apple  # Run database tests with Apple Container
make apple-up        # Run the complete server stack with Apple Container
```

### Adding New Features

1. **New Commands**: Add to `pkg/commands/` with parser and executor logic
2. **Game Systems**: Extend `pkg/game/` packages following existing patterns
3. **Persistence**: Add repository methods in `pkg/persistence/interfaces/`
4. **Tests**: Include unit tests and integration tests for new functionality

### Database Schema
The PostgreSQL schema includes:
- `players`: User accounts with preferences and subscriptions
- `characters`: Player characters with stats, skills, and location
- `item_instances`: Individual item instances with enchantments
- Proper indexes and foreign key constraints for data integrity

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add comprehensive tests for new functionality
4. Run the full test suite: `make test-db`
5. Ensure code formatting: `make lint`
6. Submit a pull request

### Testing Guidelines
- All new features require unit tests
- Database-related features need integration tests
- Maintain test coverage above 80%
- Use the containerized test environment for consistency

## Documentation

- **[TESTING.md](TESTING.md)**: Comprehensive testing guide
- **[PROJECT_STATUS.md](PROJECT_STATUS.md)**: Detailed project status and achievements
- **Code Documentation**: Inline documentation following Go conventions

## License

[Add your license here]

## Status

🎯 **Current Status**: Core systems complete with comprehensive testing infrastructure

✅ **Completed**: Game engine, persistence layer, command system, TCP server, testing suite  
🚧 **In Progress**: Additional game content and world building features  
📋 **Planned**: Web interface, advanced game mechanics, deployment configurations

The project has a solid foundation with 97+ tests, containerized database testing, and production-ready architecture patterns. Ready for additional features or deployment considerations.

### World initialization

The server embeds and automatically applies versioned SQL migrations before
accepting connections. A fresh database receives the nine-room starter grid.
`STARTING_ROOM` selects the spawn room (default `starting_room`); its zone is read
from the room definition. Startup rejects a missing spawn, malformed exits, or
any exit targeting a nonexistent room with an actionable error.

Databases initialized manually with migrations 001 or 002 are adopted after
checking their columns and constraints against the versioned schema. Incompatible
or partial schemas stop startup. Existing accounts, characters, items, and dynamic
room state are preserved. The grid migration updates starter room metadata once;
subsequent boots do not replay seeds or reset room edits.

For required database verification, run `./verify-db.sh` with `TEST_DATABASE_URL`
pointing to PostgreSQL and a role with `CREATEDB`. Unlike optional database tests,
this command fails if PostgreSQL is unavailable. See [TESTING.md](TESTING.md).

Character selection restores a saved room when both its room ID and zone match the
world definition. Missing rooms, empty locations, and mismatched zones recover to
the configured spawn and persist that recovery. Storage failures keep the player
at character selection. Movement commits its destination before reporting success;
blocked exits and failed writes preserve the saved location.

The server tracks authenticated accounts and selected characters in memory. Each
account has one active session: successful login replaces its previous connection
immediately, including before a character is selected. Failed authentication
preserves the current connection. `who` lists only characters selected in live
sessions; replacement and disconnect remove the old character from that list.
Account actions and login replacement are serialized so stale sessions cannot
change game state or unregister their replacements.

`look` lists the other live characters in your room. Movement commits the new
room and zone before updating live membership, then announces departure to the
old room and arrival to the destination. Unrelated rooms receive neither event.
Blocked exits and failed database writes leave both locations unchanged. Socket
delivery failures do not undo committed movement; reconnect restores its destination.

`say <message>` and `emote <action>` deliver to other selected live characters in
your current room, with feedback to the sender. Recipients see the sender's
character name and message; other rooms receive nothing. Moving, disconnecting,
and reconnecting change subsequent delivery through live room membership. Empty
messages and invalid or unavailable locations report failure without broadcasting.

`tell <character> <message>` sends a private message across rooms. Targets are
selected online character names, not account usernames or name prefixes. Name
matching ignores case when unique; exact capitalization disambiguates names that
only differ by case. Only the recipient receives the private content, and the
sender receives confirmation after a successful socket write. Missing or offline
targets, replacement during delivery, and failed writes report unavailability;
messages are never queued for a future login.

### Persistent starter items

Versioned setup seeds three individual items once: a Training Sword and Travel
Lantern in the starting room, and a Smooth Stone in the northern room. `look`
shows the current room's items; `inventory` shows only the selected character's
items. Names come from a custom name or known template, and each displayed item
includes its instance ID. Existing quantities are displayed without adding new
stacking behavior.

Item ownership records an explicit room or character kind. Database foreign keys
require the owner to exist, and prevent deleting a room or character while it
still owns items. Legacy ownership is migrated without resetting item state;
missing or ambiguous owners stop startup with the item ID and repair guidance.
Callers may still supply only an owner ID when it uniquely identifies an owner.

Restarts do not duplicate, relocate, or respawn seeded items. Pickup/drop commands
are tracked in #10; this change provides persistent storage and inspection.

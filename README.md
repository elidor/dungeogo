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
- **Docker Support**: Containerized test environment with automatic lifecycle management

## Quick Start

### Prerequisites
- Go 1.19 or later
- Docker and Docker Compose (for database tests)
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
```

### Building and Running
```bash
# Build the server
make build

# Run the server (requires PostgreSQL configuration)
make run
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
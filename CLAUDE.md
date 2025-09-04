# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DungeoGo is a Go-based server application that uses a configuration system with environment variables. The project follows a standard Go project layout with a cmd/server entry point.

## Architecture

The application consists of two main components:

1. **Main Server** (`cmd/server/main.go`) - Entry point that initializes configuration and starts the application
2. **Configuration System** (`config/config.go`) - Flexible configuration management using the provider pattern

### Configuration System

The config package implements a provider pattern for configuration management:
- `Config` struct holds a `ConfigProvider` interface
- `FileProvider` loads environment variables from .env files using godotenv
- `DefaultProvider` reads from system environment variables
- Predefined configuration keys: PORT, BIND_ADDRESS, DATABASE_URL, MAX_CONNECTIONS, MAX_THREADS

## Development Commands

### Building and Running
```bash
# Build the server
go build -o bin/server ./cmd/server

# Run directly with go
go run ./cmd/server

# Run with custom .env file
go run ./cmd/server
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./config
```

### Development Tools
```bash
# Format code
go fmt ./...

# Vet code for potential issues
go vet ./...

# Tidy dependencies
go mod tidy
```

## Environment Configuration

The application expects these environment variables (can be set via .env file):
- `PORT` - Server port (default from VSCode config: 8080)
- `BIND_ADDRESS` - Server bind address (default: localhost)  
- `DATABASE_URL` - Database connection string
- `MAX_CONNECTIONS` - Maximum database connections (default: 100)
- `MAX_THREADS` - Maximum threads (default: 10)

## Project Structure

```
dungeogo/
├── cmd/server/          # Main application entry point
├── config/              # Configuration management
├── .vscode/             # VSCode debug configuration
├── go.mod               # Go module definition
└── .env                 # Environment variables (gitignored)
```

The VSCode launch configuration provides example environment values for development.

## Current Implementation Status

The DungeoGo server now includes a complete foundation with the following implemented systems:

### Core Systems ✅
- **Player/Character System**: Account management with multiple character support
- **Race/Class/Skills**: Composable character creation with races (Human, Elf, Dwarf), classes (Warrior, Mage, Rogue), and skill progression
- **Item System**: Template-based items with instance modifications, enchantments, and persistence
- **TCP Server**: Multi-client connection handling with session management
- **Command System**: Extensible parser and executor for 40+ game commands
- **Database Integration**: PostgreSQL persistence layer with full CRUD operations

### Game Commands Available
- **Movement**: north, south, east, west, up, down, ne, nw, se, sw
- **Communication**: say, tell, yell, whisper, chat  
- **Information**: look, examine, who, score, time, weather
- **Inventory**: inventory, get, drop, give, wear, remove
- **Skills**: skills, practice
- **Social**: emote, smile, wave, bow
- **Combat**: kill, flee, defend (basic implementations)
- **System**: help, commands, quit, save

### Database Schema
Complete PostgreSQL schema with tables for:
- players (account data)
- characters (game avatars)  
- item_instances (owned items)
- room_states (dynamic world data)
- npc_states (NPC persistence)
- world_events (global events)

### Getting Started
1. Set up PostgreSQL database and run migrations/001_initial_schema.sql
2. Create .env file with DATABASE_URL, PORT, BIND_ADDRESS
3. `go build ./cmd/server && ./server`
4. Connect via telnet: `telnet localhost 8080`
5. Login with username 'admin', password 'admin' for testing

## Testing

The project includes comprehensive test coverage for all core systems:

### Running Tests
```bash
# Run all tests
./test.sh

# Run with coverage analysis
./test.sh --coverage

# Run individual test suites
go test ./pkg/game/character -v    # Character system tests
go test ./pkg/game/items -v        # Item system tests  
go test ./pkg/game/player -v       # Player system tests
go test ./pkg/commands -v          # Command parsing tests
```

### Test Coverage
- **Character System**: Race properties, class mechanics, skill progression, stat calculation
- **Item System**: Template creation, instance management, enchantments, stacking logic
- **Player System**: Account management, preferences, subscription handling
- **Command System**: Parsing, aliases, command types, validation

### Database Integration Tests
Some tests require PostgreSQL setup. Install PostgreSQL and set `TEST_DATABASE_URL` to run full integration tests.

## Next Development Steps
- World building system (rooms, zones, connections)
- Combat mechanics implementation  
- NPC AI and interaction system
- Magic/spell casting system
- Quest and progression systems
- Complete persistence layer tests
- Integration test suite
package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/persistence/postgres"
)

// SetupTestDatabase creates a test database with schema
func SetupTestDatabase(t *testing.T) (*sql.DB, string) {
	// Generate unique database name
	testDBName := fmt.Sprintf("dungeogo_test_%d", 
		time.Now().UnixNano())

	// Connect to postgres to create test database
	adminDB, err := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping database tests - cannot connect to postgres: %v", err)
		return nil, ""
	}

	if err := adminDB.Ping(); err != nil {
		adminDB.Close()
		t.Skipf("Skipping database tests - postgres not available: %v", err)
		return nil, ""
	}

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		adminDB.Close()
		t.Skipf("Skipping database tests - cannot create database: %v", err)
		return nil, ""
	}
	adminDB.Close()

	// Connect to test database
	testDBURL := fmt.Sprintf("postgres://localhost/%s?sslmode=disable", testDBName)
	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create schema
	err = createSchema(testDB)
	if err != nil {
		testDB.Close()
		cleanupDatabase(testDBName)
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		testDB.Close()
		cleanupDatabase(testDBName)
	})

	return testDB, testDBURL
}

// ImprovedSetupTestDB creates repository manager with proper database
func ImprovedSetupTestDB(t *testing.T) *postgres.PostgreSQLRepositoryManager {
	_, testDBURL := SetupTestDatabase(t)
	if testDBURL == "" {
		return nil
	}

	repoManager, err := postgres.NewPostgreSQLRepositoryManager(testDBURL)
	if err != nil {
		t.Fatalf("Failed to create repository manager: %v", err)
	}

	return repoManager
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";
	
	CREATE TABLE players (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		last_login TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		account_status INTEGER DEFAULT 0,
		subscription JSONB,
		preferences JSONB NOT NULL DEFAULT '{}',
		max_characters INTEGER DEFAULT 5,
		current_character_id UUID
	);

	CREATE TABLE characters (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(50) UNIQUE NOT NULL,
		race_id VARCHAR(50) NOT NULL,
		class_id VARCHAR(50) NOT NULL,
		stats JSONB NOT NULL DEFAULT '{}',
		skills JSONB NOT NULL DEFAULT '{}',
		location JSONB NOT NULL DEFAULT '{}',
		state INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		last_played TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		play_time INTERVAL DEFAULT '0 seconds',
		level INTEGER DEFAULT 1,
		experience INTEGER DEFAULT 0,
		death_count INTEGER DEFAULT 0,
		kill_count INTEGER DEFAULT 0,
		description TEXT DEFAULT '',
		appearance JSONB NOT NULL DEFAULT '{}'
	);

	CREATE TABLE item_instances (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		template_id VARCHAR(100) NOT NULL,
		owner_id UUID NOT NULL,
		quantity INTEGER DEFAULT 1,
		durability INTEGER DEFAULT 100,
		enchantments JSONB NOT NULL DEFAULT '[]',
		custom_name VARCHAR(255),
		modifications JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		last_used TIMESTAMP WITH TIME ZONE
	);

	CREATE TABLE room_states (
		room_id VARCHAR(100) PRIMARY KEY,
		items JSONB NOT NULL DEFAULT '[]',
		npcs JSONB NOT NULL DEFAULT '[]',
		players JSONB NOT NULL DEFAULT '[]',
		flags JSONB NOT NULL DEFAULT '{}',
		last_update TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE TABLE npc_states (
		npc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		template_id VARCHAR(100) NOT NULL,
		health INTEGER NOT NULL DEFAULT 100,
		location JSONB NOT NULL DEFAULT '{}',
		inventory JSONB NOT NULL DEFAULT '[]',
		state VARCHAR(50) DEFAULT 'idle',
		last_update TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE TABLE world_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		type VARCHAR(100) NOT NULL,
		description TEXT,
		start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		end_time TIMESTAMP WITH TIME ZONE,
		data JSONB NOT NULL DEFAULT '{}'
	);

	-- Create indexes
	CREATE INDEX idx_characters_player_id ON characters(player_id);
	CREATE INDEX idx_characters_name ON characters(name);
	CREATE INDEX idx_item_instances_owner ON item_instances(owner_id);
	CREATE INDEX idx_item_instances_template ON item_instances(template_id);
	`

	_, err := db.Exec(schema)
	return err
}

func cleanupDatabase(dbName string) {
	db, err := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
	if err != nil {
		return
	}
	defer db.Close()

	// Force disconnect all connections to the test database
	db.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity 
		WHERE datname = '%s' AND pid <> pg_backend_pid()`, dbName))

	// Drop the database
	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
}


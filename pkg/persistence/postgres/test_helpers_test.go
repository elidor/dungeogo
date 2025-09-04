package postgres

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// setupTestDB creates a test database with schema for testing
func setupTestDB(t *testing.T) *PostgreSQLRepositoryManager {
	// Generate unique database name
	testDBName := fmt.Sprintf("dungeogo_test_%d", time.Now().UnixNano())

	// Connect to postgres to create test database
	adminDB, err := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping database tests - cannot connect to postgres: %v", err)
		return nil
	}

	if err := adminDB.Ping(); err != nil {
		adminDB.Close()
		t.Skipf("Skipping database tests - postgres not available: %v", err)
		return nil
	}

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		adminDB.Close()
		t.Skipf("Skipping database tests - cannot create database: %v", err)
		return nil
	}
	adminDB.Close()

	// Connect to test database
	testDBURL := fmt.Sprintf("postgres://localhost/%s?sslmode=disable", testDBName)
	repoManager, err := NewPostgreSQLRepositoryManager(testDBURL)
	if err != nil {
		t.Fatalf("Failed to create repository manager: %v", err)
	}

	// Create schema
	err = createTestSchema(repoManager)
	if err != nil {
		repoManager.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		repoManager.Close()
		cleanupTestDatabase(testDBName)
	})

	return repoManager
}

func createTestSchema(repoManager *PostgreSQLRepositoryManager) error {
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

	CREATE INDEX idx_characters_player_id ON characters(player_id);
	CREATE INDEX idx_characters_name ON characters(name);
	CREATE INDEX idx_item_instances_owner ON item_instances(owner_id);
	CREATE INDEX idx_item_instances_template ON item_instances(template_id);
	`

	_, err := repoManager.GetDB().Exec(schema)
	return err
}

func cleanupTestDatabase(dbName string) {
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

func createTestPlayer() *player.Player {
	return &player.Player{
		ID:           uuid.New().String(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "testhash",
		CreatedAt:    time.Now(),
		AccountStatus: player.AccountActive,
		MaxCharacters: 5,
		Preferences: player.PlayerPrefs{
			ColorEnabled:  true,
			ScreenWidth:   80,
			AutoLoot:      false,
			CombatPrompts: true,
			Keybindings:   make(map[string]string),
		},
	}
}

func createTestCharacter(playerID string) *character.Character {
	race, _ := character.GetRaceByID("human")
	class, _ := character.GetClassByID("warrior")
	
	char := character.NewCharacter(playerID, "TestChar", race, class)
	char.ID = uuid.New().String()
	
	return char
}

func createTestItemInstance() *items.ItemInstance {
	return &items.ItemInstance{
		ID:           uuid.New().String(),
		TemplateID:   "test_template",
		OwnerID:      "test_owner",
		Quantity:     1,
		Durability:   100,
		Enchantments: []items.Enchantment{},
		Modifications: make(map[string]interface{}),
		CreatedAt:    time.Now(),
	}
}

var testCounter int

func generateUUID() string {
	testCounter++
	return fmt.Sprintf("test-uuid-%d-%s", testCounter, uuid.New().String())
}
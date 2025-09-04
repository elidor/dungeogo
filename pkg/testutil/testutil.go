package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

const (
	TestDatabaseURL = "postgres://localhost/dungeogo_test?sslmode=disable"
)

// SetupTestDB creates a test database and runs migrations
func SetupTestDB(t *testing.T) *postgres.PostgreSQLRepositoryManager {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = TestDatabaseURL
	}

	// Try to connect to postgres to check if it's available
	db, err := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping database tests - cannot connect to postgres: %v", err)
		return nil
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping database tests - postgres not available: %v", err)
		return nil
	}

	testDBName := fmt.Sprintf("dungeogo_test_%d_%d", time.Now().Unix(), os.Getpid())
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		db.Close()
		t.Skipf("Skipping database tests - cannot create test database: %v", err)
		return nil
	}
	db.Close()

	// Connect to test database
	testDBURL := fmt.Sprintf("postgres://localhost/%s?sslmode=disable", testDBName)
	repoManager, err := postgres.NewPostgreSQLRepositoryManager(testDBURL)
	if err != nil {
		// Clean up the database if we can't connect to it
		db, _ := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
		if db != nil {
			db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
			db.Close()
		}
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations (simplified schema for testing)
	err = createTestSchema(repoManager)
	if err != nil {
		repoManager.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Cleanup function
	t.Cleanup(func() {
		repoManager.Close()
		// Clean up test database
		db, err := sql.Open("postgres", "postgres://localhost/postgres?sslmode=disable")
		if err == nil {
			db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
			db.Close()
		}
	})

	return repoManager
}

func createTestSchema(repoManager *postgres.PostgreSQLRepositoryManager) error {
	// This is a simplified version of the full schema for testing
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
	`

	// Get the underlying *sql.DB from the repository manager
	// This is a simplified approach - in real implementation you'd add a method to get the DB
	db := getDBFromRepoManager(repoManager)
	_, err := db.Exec(schema)
	return err
}

// This is a hack to get the *sql.DB - in real implementation, add a proper method
func getDBFromRepoManager(repoManager *postgres.PostgreSQLRepositoryManager) *sql.DB {
	// This would need to be implemented properly in the postgres package
	// For now, we'll create a new connection
	db, _ := sql.Open("postgres", TestDatabaseURL)
	return db
}

// CreateTestPlayer creates a test player for use in tests
func CreateTestPlayer() *player.Player {
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

// CreateTestCharacter creates a test character for use in tests
func CreateTestCharacter(playerID string) *character.Character {
	race, _ := character.GetRaceByID("human")
	class, _ := character.GetClassByID("warrior")
	
	char := character.NewCharacter(playerID, "TestChar", race, class)
	char.ID = uuid.New().String()
	
	return char
}

// CreateTestItemInstance creates a test item instance
func CreateTestItemInstance(templateID, ownerID string) *items.ItemInstance {
	return &items.ItemInstance{
		ID:           uuid.New().String(),
		TemplateID:   templateID,
		OwnerID:      ownerID,
		Quantity:     1,
		Durability:   100,
		Enchantments: []items.Enchantment{},
		Modifications: make(map[string]interface{}),
		CreatedAt:    time.Now(),
	}
}

// AssertEqual is a simple equality assertion helper
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotNil checks that a value is not nil
func AssertNotNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// AssertError checks that an error occurred
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error but got none", msg)
	}
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: unexpected error: %v", msg, err)
	}
}
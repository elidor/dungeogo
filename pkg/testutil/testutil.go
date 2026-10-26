package testutil

import (
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

// SetupTestDB uses an isolated database with the production migrations.
func SetupTestDB(t *testing.T) *postgres.PostgreSQLRepositoryManager {
	t.Helper()
	return ImprovedSetupTestDB(t)
}

// CreateTestPlayer creates a test player for use in tests
func CreateTestPlayer() *player.Player {
	return &player.Player{
		ID:            uuid.New().String(),
		Username:      "testuser",
		Email:         "test@example.com",
		PasswordHash:  "testhash",
		CreatedAt:     time.Now(),
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
		ID:            uuid.New().String(),
		TemplateID:    templateID,
		OwnerID:       ownerID,
		Quantity:      1,
		Durability:    100,
		Enchantments:  []items.Enchantment{},
		Modifications: make(map[string]interface{}),
		CreatedAt:     time.Now(),
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

package postgres

import (
	"github.com/elidor/dungeogo/internal/testdb"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *PostgreSQLRepositoryManager {
	t.Helper()
	_, databaseURL := testdb.New(t, true)
	manager, err := NewPostgreSQLRepositoryManager(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Close() })
	return manager
}

func createTestPlayer() *player.Player {
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

func createTestCharacter(playerID string) *character.Character {
	race, _ := character.GetRaceByID("human")
	class, _ := character.GetClassByID("warrior")

	char := character.NewCharacter(playerID, "TestChar", race, class)
	char.ID = uuid.New().String()

	return char
}

func createTestItemInstance() *items.ItemInstance {
	return &items.ItemInstance{
		ID:            uuid.New().String(),
		TemplateID:    "test_template",
		OwnerID:       "newbie_west",
		Quantity:      1,
		Durability:    100,
		Enchantments:  []items.Enchantment{},
		Modifications: make(map[string]interface{}),
		CreatedAt:     time.Now(),
	}
}

var testCounter int

func generateUUID() string {
	testCounter++
	return uuid.New().String()
}

func existingItemCharacter(t *testing.T, manager *PostgreSQLRepositoryManager) string {
	t.Helper()
	p := createTestPlayer()
	p.Username = "owner_" + uuid.NewString()[:8]
	p.Email = p.Username + "@example.com"
	if err := manager.Players().CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	c := createTestCharacter(p.ID)
	c.Name = "Owner" + uuid.NewString()[:8]
	if err := manager.Characters().CreateCharacter(c); err != nil {
		t.Fatal(err)
	}
	return c.ID
}

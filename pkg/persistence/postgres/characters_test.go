package postgres

import (
	"testing"

	"github.com/elidor/dungeogo/pkg/game/character"
)

func TestCharacterRepository_CreateCharacter(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create a test player first
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	// Create character
	repo := repoManager.Characters()
	testChar := createTestCharacter(testPlayer.ID)

	err = repo.CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetCharacter(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve character: %v", err)
	}

	if retrieved.ID != testChar.ID {
		t.Errorf("Expected ID %s, got %s", testChar.ID, retrieved.ID)
	}

	if retrieved.Name != testChar.Name {
		t.Errorf("Expected name %s, got %s", testChar.Name, retrieved.Name)
	}

	if retrieved.PlayerID != testChar.PlayerID {
		t.Errorf("Expected player ID %s, got %s", testChar.PlayerID, retrieved.PlayerID)
	}

	if retrieved.Level != testChar.Level {
		t.Errorf("Expected level %d, got %d", testChar.Level, retrieved.Level)
	}

	// Verify race and class were restored
	if retrieved.Race == nil || retrieved.Race.ID != testChar.Race.ID {
		t.Errorf("Expected race to be restored correctly")
	}

	if retrieved.Class == nil || retrieved.Class.ID != testChar.Class.ID {
		t.Errorf("Expected class to be restored correctly")
	}

	// Verify stats were preserved
	if retrieved.Stats.Strength != testChar.Stats.Strength {
		t.Errorf("Expected strength %d, got %d", testChar.Stats.Strength, retrieved.Stats.Strength)
	}
}

func TestCharacterRepository_GetCharactersByPlayer(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()

	// Create multiple characters for the player
	char1 := createTestCharacter(testPlayer.ID)
	char1.Name = "Character1"
	char1.Level = 5

	char2 := createTestCharacter(testPlayer.ID)
	char2.Name = "Character2"
	char2.Level = 10

	err = repo.CreateCharacter(char1)
	if err != nil {
		t.Fatalf("Failed to create character 1: %v", err)
	}

	err = repo.CreateCharacter(char2)
	if err != nil {
		t.Fatalf("Failed to create character 2: %v", err)
	}

	// Get characters by player
	characters, err := repo.GetCharactersByPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to get characters by player: %v", err)
	}

	if len(characters) != 2 {
		t.Errorf("Expected 2 characters, got %d", len(characters))
	}

	// Verify character summaries
	var foundChar1, foundChar2 bool
	for _, char := range characters {
		if char.Name == "Character1" {
			foundChar1 = true
			if char.Level != 5 {
				t.Errorf("Expected Character1 level 5, got %d", char.Level)
			}
			if !char.IsAlive {
				t.Errorf("Expected Character1 to be alive")
			}
		} else if char.Name == "Character2" {
			foundChar2 = true
			if char.Level != 10 {
				t.Errorf("Expected Character2 level 10, got %d", char.Level)
			}
		}
	}

	if !foundChar1 || !foundChar2 {
		t.Errorf("Expected to find both characters in summary")
	}
}

func TestCharacterRepository_UpdateCharacter(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player and character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()
	testChar := createTestCharacter(testPlayer.ID)

	err = repo.CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update character
	testChar.Level = 20
	testChar.Experience = 50000
	testChar.Stats.Strength = 25
	testChar.Description = "Updated character description"

	err = repo.UpdateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Retrieve and verify updates
	retrieved, err := repo.GetCharacter(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated character: %v", err)
	}

	if retrieved.Level != 20 {
		t.Errorf("Expected level 20, got %d", retrieved.Level)
	}

	if retrieved.Experience != 50000 {
		t.Errorf("Expected experience 50000, got %d", retrieved.Experience)
	}

	if retrieved.Stats.Strength != 25 {
		t.Errorf("Expected strength 25, got %d", retrieved.Stats.Strength)
	}

	if retrieved.Description != "Updated character description" {
		t.Errorf("Expected updated description, got %s", retrieved.Description)
	}
}

func TestCharacterRepository_UpdateCharacterStats(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player and character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()
	testChar := createTestCharacter(testPlayer.ID)

	err = repo.CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update just stats
	newStats := &character.CharacterStats{
		Strength:     30,
		Dexterity:    25,
		Intelligence: 20,
		Constitution: 28,
		Wisdom:       22,
		Charisma:     18,
		Health:       280,
		MaxHealth:    280,
		Mana:         100,
		MaxMana:      100,
		Stamina:      140,
		MaxStamina:   140,
	}

	err = repo.UpdateCharacterStats(testChar.ID, newStats)
	if err != nil {
		t.Fatalf("Failed to update character stats: %v", err)
	}

	// Retrieve and verify stats update
	retrieved, err := repo.GetCharacter(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve character: %v", err)
	}

	if retrieved.Stats.Strength != 30 {
		t.Errorf("Expected strength 30, got %d", retrieved.Stats.Strength)
	}

	if retrieved.Stats.Health != 280 {
		t.Errorf("Expected health 280, got %d", retrieved.Stats.Health)
	}
}

func TestCharacterRepository_UpdateCharacterLocation(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player and character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()
	testChar := createTestCharacter(testPlayer.ID)

	err = repo.CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update location
	newLocation := &character.Location{
		RoomID: "tavern_main",
		ZoneID: "city_center",
		X:      100,
		Y:      50,
	}

	err = repo.UpdateCharacterLocation(testChar.ID, newLocation)
	if err != nil {
		t.Fatalf("Failed to update character location: %v", err)
	}

	// Retrieve and verify location update
	retrieved, err := repo.GetCharacter(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve character: %v", err)
	}

	if retrieved.Location.RoomID != "tavern_main" {
		t.Errorf("Expected room ID 'tavern_main', got %s", retrieved.Location.RoomID)
	}

	if retrieved.Location.X != 100 {
		t.Errorf("Expected X coordinate 100, got %d", retrieved.Location.X)
	}
}

func TestCharacterRepository_SaveCharacterSkills(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player and character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()
	testChar := createTestCharacter(testPlayer.ID)

	err = repo.CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update skills
	testChar.Skills.AddExperience(character.SkillSwords, 500)
	testChar.Skills.AddExperience(character.SkillMagic, 300)

	err = repo.SaveCharacterSkills(testChar.ID, testChar.Skills)
	if err != nil {
		t.Fatalf("Failed to save character skills: %v", err)
	}

	// Retrieve and verify skills were saved
	retrieved, err := repo.GetCharacter(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve character: %v", err)
	}

	swordsExp := retrieved.Skills.GetSkill(character.SkillSwords).Experience
	if swordsExp != 500 {
		t.Errorf("Expected swords experience 500, got %d", swordsExp)
	}

	magicExp := retrieved.Skills.GetSkill(character.SkillMagic).Experience
	if magicExp != 300 {
		t.Errorf("Expected magic experience 300, got %d", magicExp)
	}
}

func TestCharacterRepository_DeleteCharacter(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player and character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()
	testChar := createTestCharacter(testPlayer.ID)

	err = repo.CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Delete character
	err = repo.DeleteCharacter(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to delete character: %v", err)
	}

	// Verify character was deleted
	_, err = repo.GetCharacter(testChar.ID)
	if err == nil {
		t.Errorf("Expected error when retrieving deleted character")
	}
}

func TestCharacterRepository_UniqueNameConstraint(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	// Create test player
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	repo := repoManager.Characters()

	// Create first character
	char1 := createTestCharacter(testPlayer.ID)
	char1.Name = "UniqueCharacter"

	err = repo.CreateCharacter(char1)
	if err != nil {
		t.Fatalf("Failed to create first character: %v", err)
	}

	// Try to create second character with same name
	char2 := createTestCharacter(testPlayer.ID)
	char2.Name = "UniqueCharacter" // Same name

	err = repo.CreateCharacter(char2)
	if err == nil {
		t.Errorf("Expected error when creating character with duplicate name")
	}
}


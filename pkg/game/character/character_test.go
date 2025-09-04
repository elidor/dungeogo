package character

import (
	"testing"
	"time"
)

func TestNewCharacter(t *testing.T) {
	playerID := "test-player-123"
	name := "TestWarrior"
	
	race, err := GetRaceByID("human")
	if err != nil {
		t.Fatalf("Failed to get race: %v", err)
	}
	
	class, err := GetClassByID("warrior")
	if err != nil {
		t.Fatalf("Failed to get class: %v", err)
	}
	
	char := NewCharacter(playerID, name, race, class)
	
	// Test basic properties
	if char.PlayerID != playerID {
		t.Errorf("Expected PlayerID %s, got %s", playerID, char.PlayerID)
	}
	
	if char.Name != name {
		t.Errorf("Expected Name %s, got %s", name, char.Name)
	}
	
	if char.Race != race {
		t.Errorf("Expected race to be set correctly")
	}
	
	if char.Class != class {
		t.Errorf("Expected class to be set correctly")
	}
	
	// Test initial state
	if char.State != CharacterAlive {
		t.Errorf("Expected character to be alive initially")
	}
	
	if char.Level != 1 {
		t.Errorf("Expected initial level to be 1, got %d", char.Level)
	}
	
	if char.Experience != 0 {
		t.Errorf("Expected initial experience to be 0, got %d", char.Experience)
	}
	
	// Test stats calculation
	if char.Stats == nil {
		t.Errorf("Expected stats to be initialized")
	}
	
	// Human has +1 charisma, so base 10 + 1 = 11
	if char.Stats.Charisma != 11 {
		t.Errorf("Expected charisma to be 11 (base 10 + human bonus), got %d", char.Stats.Charisma)
	}
	
	// Test health calculation (constitution * 10)
	expectedHealth := char.Stats.Constitution * 10
	if char.Stats.MaxHealth != expectedHealth || char.Stats.Health != expectedHealth {
		t.Errorf("Expected health to be %d, got %d/%d", expectedHealth, char.Stats.Health, char.Stats.MaxHealth)
	}
}

func TestCharacterIsAlive(t *testing.T) {
	char := createTestCharacter()
	
	// Initially alive
	if !char.IsAlive() {
		t.Errorf("Character should be alive initially")
	}
	
	if char.IsDead() {
		t.Errorf("Character should not be dead initially")
	}
	
	// Set health to 0
	char.Stats.Health = 0
	if char.IsAlive() {
		t.Errorf("Character should not be alive with 0 health")
	}
	
	if !char.IsDead() {
		t.Errorf("Character should be dead with 0 health")
	}
	
	// Set state to dead
	char.Stats.Health = 10
	char.State = CharacterDead
	if char.IsAlive() {
		t.Errorf("Character should not be alive when state is dead")
	}
	
	if !char.IsDead() {
		t.Errorf("Character should be dead when state is dead")
	}
}

func TestCharacterUpdatePlayTime(t *testing.T) {
	char := createTestCharacter()
	
	// Set initial last played time
	initialTime := time.Now().Add(-1 * time.Hour)
	char.LastPlayed = initialTime
	
	initialPlayTime := char.PlayTime
	
	// Simulate some time passing
	time.Sleep(10 * time.Millisecond)
	
	char.UpdatePlayTime()
	
	// Check that play time was updated
	if char.PlayTime <= initialPlayTime {
		t.Errorf("PlayTime should have increased")
	}
	
	// Check that LastPlayed was updated
	if char.LastPlayed.Before(initialTime) {
		t.Errorf("LastPlayed should have been updated")
	}
}

func TestCalculateStartingStats(t *testing.T) {
	race, _ := GetRaceByID("dwarf")
	class, _ := GetClassByID("warrior")
	
	stats := calculateStartingStats(race, class)
	
	// Base stats are 10, dwarf has +1 STR, +2 CON, -1 CHA
	expectedStr := 10 + 1  // 11
	expectedCon := 10 + 2  // 12
	expectedCha := 10 - 1  // 9
	
	if stats.Strength != expectedStr {
		t.Errorf("Expected strength %d, got %d", expectedStr, stats.Strength)
	}
	
	if stats.Constitution != expectedCon {
		t.Errorf("Expected constitution %d, got %d", expectedCon, stats.Constitution)
	}
	
	if stats.Charisma != expectedCha {
		t.Errorf("Expected charisma %d, got %d", expectedCha, stats.Charisma)
	}
	
	// Test health calculation (constitution * 10)
	expectedMaxHealth := expectedCon * 10
	if stats.MaxHealth != expectedMaxHealth {
		t.Errorf("Expected max health %d, got %d", expectedMaxHealth, stats.MaxHealth)
	}
	
	if stats.Health != expectedMaxHealth {
		t.Errorf("Expected current health to equal max health")
	}
}

func TestCharacterStates(t *testing.T) {
	char := createTestCharacter()
	
	// Test all character states
	states := []CharacterState{
		CharacterAlive,
		CharacterDead,
		CharacterSleeping,
		CharacterAfk,
		CharacterInCombat,
	}
	
	for _, state := range states {
		char.State = state
		if char.State != state {
			t.Errorf("Failed to set character state to %d", state)
		}
	}
}

func createTestCharacter() *Character {
	race, _ := GetRaceByID("human")
	class, _ := GetClassByID("warrior")
	return NewCharacter("test-player", "TestChar", race, class)
}
package character

import (
	"testing"
)

func TestGetRaceByID(t *testing.T) {
	tests := []struct {
		id          string
		expectError bool
		expectedName string
	}{
		{"human", false, "Human"},
		{"elf", false, "Elf"},
		{"dwarf", false, "Dwarf"},
		{"invalid", true, ""},
	}
	
	for _, test := range tests {
		race, err := GetRaceByID(test.id)
		
		if test.expectError {
			if err == nil {
				t.Errorf("Expected error for race ID %s, but got none", test.id)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for race ID %s: %v", test.id, err)
			}
			
			if race == nil {
				t.Errorf("Expected race for ID %s, got nil", test.id)
				continue
			}
			
			if race.Name != test.expectedName {
				t.Errorf("Expected race name %s, got %s", test.expectedName, race.Name)
			}
			
			if race.ID != test.id {
				t.Errorf("Expected race ID %s, got %s", test.id, race.ID)
			}
		}
	}
}

func TestGetAllRaces(t *testing.T) {
	races := GetAllRaces()
	
	if len(races) == 0 {
		t.Errorf("Expected at least one race, got none")
	}
	
	// Check that we have the expected races
	expectedRaces := []string{"human", "elf", "dwarf"}
	for _, expectedID := range expectedRaces {
		if _, exists := races[expectedID]; !exists {
			t.Errorf("Expected race %s to exist", expectedID)
		}
	}
}

func TestHumanRaceProperties(t *testing.T) {
	race, err := GetRaceByID("human")
	if err != nil {
		t.Fatalf("Failed to get human race: %v", err)
	}
	
	// Test basic properties
	if race.Name != "Human" {
		t.Errorf("Expected name 'Human', got %s", race.Name)
	}
	
	if race.SizeCategory != SizeMedium {
		t.Errorf("Expected size category Medium, got %d", race.SizeCategory)
	}
	
	if race.Lifespan != 80 {
		t.Errorf("Expected lifespan 80, got %d", race.Lifespan)
	}
	
	// Test stat modifiers - humans have +1 charisma
	if race.StatModifiers.Charisma != 1 {
		t.Errorf("Expected charisma modifier +1, got %d", race.StatModifiers.Charisma)
	}
	
	// Other stats should be 0 for humans
	if race.StatModifiers.Strength != 0 {
		t.Errorf("Expected strength modifier 0, got %d", race.StatModifiers.Strength)
	}
}

func TestElfRaceProperties(t *testing.T) {
	race, err := GetRaceByID("elf")
	if err != nil {
		t.Fatalf("Failed to get elf race: %v", err)
	}
	
	// Test stat modifiers - elves have -1 STR, +2 DEX, +1 INT, -1 CON, +1 WIS
	expected := map[string]int{
		"strength":     -1,
		"dexterity":    2,
		"intelligence": 1,
		"constitution": -1,
		"wisdom":       1,
		"charisma":     0,
	}
	
	actual := map[string]int{
		"strength":     race.StatModifiers.Strength,
		"dexterity":    race.StatModifiers.Dexterity,
		"intelligence": race.StatModifiers.Intelligence,
		"constitution": race.StatModifiers.Constitution,
		"wisdom":       race.StatModifiers.Wisdom,
		"charisma":     race.StatModifiers.Charisma,
	}
	
	for stat, expectedValue := range expected {
		if actual[stat] != expectedValue {
			t.Errorf("Expected %s modifier %d, got %d", stat, expectedValue, actual[stat])
		}
	}
	
	// Test skill bonuses
	if archeryBonus, exists := race.SkillBonuses[SkillArchery]; !exists || archeryBonus != 10 {
		t.Errorf("Expected archery skill bonus 10, got %d", archeryBonus)
	}
	
	// Test racial abilities
	if len(race.Abilities) == 0 {
		t.Errorf("Expected elves to have racial abilities")
	}
	
	// Check for darkvision
	hasDarkvision := false
	for _, ability := range race.Abilities {
		if ability.ID == "darkvision" {
			hasDarkvision = true
			if ability.Type != AbilityVision {
				t.Errorf("Expected darkvision to be vision type ability")
			}
			if !ability.Passive {
				t.Errorf("Expected darkvision to be passive")
			}
		}
	}
	if !hasDarkvision {
		t.Errorf("Expected elves to have darkvision")
	}
}

func TestDwarfRaceProperties(t *testing.T) {
	race, err := GetRaceByID("dwarf")
	if err != nil {
		t.Fatalf("Failed to get dwarf race: %v", err)
	}
	
	// Test stat modifiers - dwarves have +1 STR, -1 DEX, +2 CON, +1 WIS, -1 CHA
	if race.StatModifiers.Strength != 1 {
		t.Errorf("Expected strength modifier +1, got %d", race.StatModifiers.Strength)
	}
	
	if race.StatModifiers.Constitution != 2 {
		t.Errorf("Expected constitution modifier +2, got %d", race.StatModifiers.Constitution)
	}
	
	// Test skill bonuses
	if axesBonus, exists := race.SkillBonuses[SkillAxes]; !exists || axesBonus != 15 {
		t.Errorf("Expected axes skill bonus 15, got %d", axesBonus)
	}
	
	if craftingBonus, exists := race.SkillBonuses[SkillCrafting]; !exists || craftingBonus != 20 {
		t.Errorf("Expected crafting skill bonus 20, got %d", craftingBonus)
	}
	
	// Test poison resistance
	hasPoisonResistance := false
	for _, ability := range race.Abilities {
		if ability.ID == "poison_resistance" {
			hasPoisonResistance = true
			if ability.Type != AbilityResistance {
				t.Errorf("Expected poison resistance to be resistance type ability")
			}
		}
	}
	if !hasPoisonResistance {
		t.Errorf("Expected dwarves to have poison resistance")
	}
}

func TestSizeCategories(t *testing.T) {
	sizes := []SizeType{SizeTiny, SizeSmall, SizeMedium, SizeLarge, SizeHuge}
	
	// Just test that the constants are defined and different
	for i, size := range sizes {
		if int(size) != i {
			t.Errorf("Expected size constant %d to have value %d, got %d", i, i, int(size))
		}
	}
}

func TestAbilityTypes(t *testing.T) {
	abilities := []AbilityType{
		AbilityVision,
		AbilityResistance,
		AbilityMovement,
		AbilityCombat,
		AbilityMagic,
	}
	
	// Just test that the constants are defined and different
	for i, ability := range abilities {
		if int(ability) != i {
			t.Errorf("Expected ability constant %d to have value %d, got %d", i, i, int(ability))
		}
	}
}
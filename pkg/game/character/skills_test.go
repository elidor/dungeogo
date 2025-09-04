package character

import (
	"testing"
	"time"
)

func TestNewSkillSet(t *testing.T) {
	skillSet := NewSkillSet()
	
	if skillSet == nil {
		t.Fatalf("NewSkillSet returned nil")
	}
	
	if skillSet.Skills == nil {
		t.Fatalf("Skills map is nil")
	}
	
	// Check that all skills are initialized
	expectedSkills := []SkillType{
		SkillSwords, SkillAxes, SkillMaces, SkillDaggers,
		SkillArchery, SkillCrossbows, SkillShields,
		SkillDodge, SkillParry, SkillMagic, SkillHealing,
		SkillEvocation, SkillDivination, SkillLockpicking,
		SkillStealth, SkillCrafting, SkillFishing, SkillMining,
	}
	
	for _, skillType := range expectedSkills {
		skill := skillSet.GetSkill(skillType)
		if skill == nil {
			t.Errorf("Expected skill %s to be initialized", GetSkillName(skillType))
		} else {
			if skill.Type != skillType {
				t.Errorf("Expected skill type %d, got %d", skillType, skill.Type)
			}
			if skill.Level != 0 {
				t.Errorf("Expected initial skill level 0, got %d", skill.Level)
			}
			if skill.Experience != 0 {
				t.Errorf("Expected initial experience 0, got %d", skill.Experience)
			}
		}
	}
}

func TestGetSkill(t *testing.T) {
	skillSet := NewSkillSet()
	
	// Test getting an existing skill
	skill := skillSet.GetSkill(SkillSwords)
	if skill == nil {
		t.Errorf("Expected to get swords skill")
	} else {
		if skill.Type != SkillSwords {
			t.Errorf("Expected skill type to be swords")
		}
	}
}

func TestGetSkillLevel(t *testing.T) {
	skillSet := NewSkillSet()
	
	// Test initial skill level
	level := skillSet.GetSkillLevel(SkillSwords)
	if level != 0 {
		t.Errorf("Expected initial skill level 0, got %d", level)
	}
	
	// Manually set a skill level and test
	swordSkill := skillSet.GetSkill(SkillSwords)
	swordSkill.Level = 15
	
	level = skillSet.GetSkillLevel(SkillSwords)
	if level != 15 {
		t.Errorf("Expected skill level 15, got %d", level)
	}
}

func TestAddExperience(t *testing.T) {
	skillSet := NewSkillSet()
	
	// Add some experience
	leveledUp := skillSet.AddExperience(SkillSwords, 50)
	
	swordSkill := skillSet.GetSkill(SkillSwords)
	if swordSkill.Experience != 50 {
		t.Errorf("Expected experience 50, got %d", swordSkill.Experience)
	}
	
	// Should not have leveled up yet (need 100 exp for level 1)
	if leveledUp {
		t.Errorf("Should not have leveled up with 50 experience")
	}
	
	if swordSkill.Level != 0 {
		t.Errorf("Expected level to still be 0, got %d", swordSkill.Level)
	}
	
	// Add more experience to trigger level up
	leveledUp = skillSet.AddExperience(SkillSwords, 50)
	
	if !leveledUp {
		t.Errorf("Should have leveled up with 100 experience")
	}
	
	if swordSkill.Level != 1 {
		t.Errorf("Expected level 1 after level up, got %d", swordSkill.Level)
	}
	
	// Check that LastUsed was updated
	if swordSkill.LastUsed.IsZero() {
		t.Errorf("Expected LastUsed to be set")
	}
}

func TestExperienceNeededForLevel(t *testing.T) {
	skillSet := NewSkillSet()
	
	tests := []struct {
		level    int
		expected int
	}{
		{0, 0},
		{1, 100},   // 1 * 1 * 100
		{2, 400},   // 2 * 2 * 100
		{3, 900},   // 3 * 3 * 100
		{10, 10000}, // 10 * 10 * 100
	}
	
	for _, test := range tests {
		actual := skillSet.experienceNeededForLevel(test.level)
		if actual != test.expected {
			t.Errorf("Expected %d experience for level %d, got %d", 
				test.expected, test.level, actual)
		}
	}
}

func TestSkillModifiers(t *testing.T) {
	skillSet := NewSkillSet()
	
	// Add a base level
	skillSet.AddExperience(SkillSwords, 100) // Level 1
	
	baseLevel := skillSet.GetSkillLevel(SkillSwords)
	effectiveLevel := skillSet.GetEffectiveSkillLevel(SkillSwords)
	
	// Should be the same initially
	if baseLevel != effectiveLevel {
		t.Errorf("Expected base and effective level to be same initially")
	}
	
	// Add a bonus modifier
	bonusModifier := SkillModifier{
		Source: "test_item",
		Value:  5,
		Type:   ModifierBonus,
	}
	skillSet.AddModifier(SkillSwords, bonusModifier)
	
	effectiveLevel = skillSet.GetEffectiveSkillLevel(SkillSwords)
	expectedEffective := baseLevel + 5
	
	if effectiveLevel != expectedEffective {
		t.Errorf("Expected effective level %d with bonus, got %d", 
			expectedEffective, effectiveLevel)
	}
	
	// Add a penalty modifier
	penaltyModifier := SkillModifier{
		Source: "curse",
		Value:  2,
		Type:   ModifierPenalty,
	}
	skillSet.AddModifier(SkillSwords, penaltyModifier)
	
	effectiveLevel = skillSet.GetEffectiveSkillLevel(SkillSwords)
	expectedEffective = baseLevel + 5 - 2 // +5 bonus, -2 penalty
	
	if effectiveLevel != expectedEffective {
		t.Errorf("Expected effective level %d with bonus and penalty, got %d", 
			expectedEffective, effectiveLevel)
	}
	
	// Test multiplier
	multiplierModifier := SkillModifier{
		Source: "blessing",
		Value:  150, // 150% = 1.5x
		Type:   ModifierMultiplier,
	}
	skillSet.AddModifier(SkillSwords, multiplierModifier)
	
	effectiveLevel = skillSet.GetEffectiveSkillLevel(SkillSwords)
	// Should be (baseLevel + 5 - 2) * 1.5 = (1 + 5 - 2) * 1.5 = 4 * 1.5 = 6
	expectedEffective = ((baseLevel + 5 - 2) * 150) / 100
	
	if effectiveLevel != expectedEffective {
		t.Errorf("Expected effective level %d with multiplier, got %d", 
			expectedEffective, effectiveLevel)
	}
}

func TestRemoveModifier(t *testing.T) {
	skillSet := NewSkillSet()
	
	// Add a modifier
	modifier := SkillModifier{
		Source: "test_item",
		Value:  5,
		Type:   ModifierBonus,
	}
	skillSet.AddModifier(SkillSwords, modifier)
	
	// Verify it's applied
	effectiveLevel := skillSet.GetEffectiveSkillLevel(SkillSwords)
	if effectiveLevel != 5 { // 0 base + 5 bonus
		t.Errorf("Expected effective level 5 with modifier, got %d", effectiveLevel)
	}
	
	// Remove the modifier
	skillSet.RemoveModifier(SkillSwords, "test_item")
	
	// Verify it's removed
	effectiveLevel = skillSet.GetEffectiveSkillLevel(SkillSwords)
	if effectiveLevel != 0 {
		t.Errorf("Expected effective level 0 after removing modifier, got %d", effectiveLevel)
	}
}

func TestGetSkillName(t *testing.T) {
	tests := []struct {
		skill    SkillType
		expected string
	}{
		{SkillSwords, "Swords"},
		{SkillArchery, "Archery"},
		{SkillMagic, "Magic"},
		{SkillStealth, "Stealth"},
		{SkillCrafting, "Crafting"},
	}
	
	for _, test := range tests {
		actual := GetSkillName(test.skill)
		if actual != test.expected {
			t.Errorf("Expected skill name %s, got %s", test.expected, actual)
		}
	}
	
	// Test unknown skill
	unknownSkill := SkillType(999)
	name := GetSkillName(unknownSkill)
	if name != "Unknown" {
		t.Errorf("Expected 'Unknown' for invalid skill, got %s", name)
	}
}

func TestSkillConstants(t *testing.T) {
	// Test that all skill constants are unique
	skills := []SkillType{
		SkillSwords, SkillAxes, SkillMaces, SkillDaggers,
		SkillArchery, SkillCrossbows, SkillShields,
		SkillDodge, SkillParry, SkillMagic, SkillHealing,
		SkillEvocation, SkillDivination, SkillLockpicking,
		SkillStealth, SkillCrafting, SkillFishing, SkillMining,
	}
	
	seen := make(map[SkillType]bool)
	for _, skill := range skills {
		if seen[skill] {
			t.Errorf("Duplicate skill constant found: %d", skill)
		}
		seen[skill] = true
	}
}

func TestLastUsedTracking(t *testing.T) {
	skillSet := NewSkillSet()
	
	// Initially LastUsed should be zero
	swordSkill := skillSet.GetSkill(SkillSwords)
	if !swordSkill.LastUsed.IsZero() {
		t.Errorf("Expected LastUsed to be zero initially")
	}
	
	// Add experience and check LastUsed is updated
	before := time.Now()
	skillSet.AddExperience(SkillSwords, 10)
	after := time.Now()
	
	if swordSkill.LastUsed.Before(before) || swordSkill.LastUsed.After(after) {
		t.Errorf("Expected LastUsed to be updated to current time")
	}
}
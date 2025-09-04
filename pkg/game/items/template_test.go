package items

import (
	"testing"
)

func TestNewItemTemplate(t *testing.T) {
	template := NewItemTemplate("test_sword", "Test Sword", ItemWeapon)
	
	// Test basic properties
	if template.ID != "test_sword" {
		t.Errorf("Expected ID 'test_sword', got %s", template.ID)
	}
	
	if template.Name != "Test Sword" {
		t.Errorf("Expected name 'Test Sword', got %s", template.Name)
	}
	
	if template.Type != ItemWeapon {
		t.Errorf("Expected type ItemWeapon, got %d", template.Type)
	}
	
	// Test default values
	if template.Rarity != RarityCommon {
		t.Errorf("Expected default rarity Common, got %d", template.Rarity)
	}
	
	if template.Weight != 1.0 {
		t.Errorf("Expected default weight 1.0, got %f", template.Weight)
	}
	
	if template.Value != 10 {
		t.Errorf("Expected default value 10, got %d", template.Value)
	}
	
	if template.Durability != 100 {
		t.Errorf("Expected default durability 100, got %d", template.Durability)
	}
	
	if !template.Enchantable {
		t.Errorf("Expected default enchantable true")
	}
	
	if template.StackSize != 1 {
		t.Errorf("Expected default stack size 1, got %d", template.StackSize)
	}
	
	// Test initialized maps
	if template.BaseStats.StatBonuses == nil {
		t.Errorf("Expected StatBonuses map to be initialized")
	}
	
	if template.Requirements.MinStats == nil {
		t.Errorf("Expected MinStats map to be initialized")
	}
}

func TestIsStackable(t *testing.T) {
	// Test non-stackable item (default)
	weapon := NewItemTemplate("sword", "Sword", ItemWeapon)
	if weapon.IsStackable() {
		t.Errorf("Expected weapon to not be stackable by default")
	}
	
	// Test stackable item
	potion := NewItemTemplate("potion", "Potion", ItemConsumable)
	potion.StackSize = 10
	if !potion.IsStackable() {
		t.Errorf("Expected potion with stack size > 1 to be stackable")
	}
}

func TestCanUse(t *testing.T) {
	template := NewItemTemplate("sword", "Sword", ItemWeapon)
	
	// Currently returns true for all - test the interface exists
	canUse := template.CanUse(nil)
	if !canUse {
		t.Errorf("Expected CanUse to return true (placeholder implementation)")
	}
}

func TestGetItemTypeName(t *testing.T) {
	tests := []struct {
		itemType ItemType
		expected string
	}{
		{ItemWeapon, "Weapon"},
		{ItemArmor, "Armor"},
		{ItemShield, "Shield"},
		{ItemConsumable, "Consumable"},
		{ItemContainer, "Container"},
		{ItemKey, "Key"},
		{ItemTreasure, "Treasure"},
		{ItemTool, "Tool"},
		{ItemMaterial, "Material"},
	}
	
	for _, test := range tests {
		actual := GetItemTypeName(test.itemType)
		if actual != test.expected {
			t.Errorf("Expected type name %s for type %d, got %s", 
				test.expected, test.itemType, actual)
		}
	}
	
	// Test unknown type
	unknownType := ItemType(999)
	name := GetItemTypeName(unknownType)
	if name != "Unknown" {
		t.Errorf("Expected 'Unknown' for invalid item type, got %s", name)
	}
}

func TestGetRarityName(t *testing.T) {
	tests := []struct {
		rarity   RarityType
		expected string
	}{
		{RarityCommon, "Common"},
		{RarityUncommon, "Uncommon"},
		{RarityRare, "Rare"},
		{RarityEpic, "Epic"},
		{RarityLegendary, "Legendary"},
	}
	
	for _, test := range tests {
		actual := GetRarityName(test.rarity)
		if actual != test.expected {
			t.Errorf("Expected rarity name %s for rarity %d, got %s", 
				test.expected, test.rarity, actual)
		}
	}
	
	// Test unknown rarity
	unknownRarity := RarityType(999)
	name := GetRarityName(unknownRarity)
	if name != "Unknown" {
		t.Errorf("Expected 'Unknown' for invalid rarity, got %s", name)
	}
}

func TestItemStats(t *testing.T) {
	template := NewItemTemplate("magic_sword", "Magic Sword", ItemWeapon)
	
	// Test setting stats
	template.BaseStats.Damage = 15
	template.BaseStats.HitBonus = 2
	template.BaseStats.StatBonuses[StatStrength] = 3
	
	if template.BaseStats.Damage != 15 {
		t.Errorf("Expected damage 15, got %d", template.BaseStats.Damage)
	}
	
	if template.BaseStats.HitBonus != 2 {
		t.Errorf("Expected hit bonus 2, got %d", template.BaseStats.HitBonus)
	}
	
	if template.BaseStats.StatBonuses[StatStrength] != 3 {
		t.Errorf("Expected strength bonus 3, got %d", 
			template.BaseStats.StatBonuses[StatStrength])
	}
}

func TestRequirements(t *testing.T) {
	template := NewItemTemplate("heavy_armor", "Heavy Armor", ItemArmor)
	
	// Set requirements
	template.Requirements.MinLevel = 10
	template.Requirements.MinStats[StatStrength] = 15
	template.Requirements.RequiredClass = []string{"warrior"}
	template.Requirements.Forbidden = []string{"mage"}
	
	if template.Requirements.MinLevel != 10 {
		t.Errorf("Expected min level 10, got %d", template.Requirements.MinLevel)
	}
	
	if template.Requirements.MinStats[StatStrength] != 15 {
		t.Errorf("Expected min strength 15, got %d", 
			template.Requirements.MinStats[StatStrength])
	}
	
	if len(template.Requirements.RequiredClass) != 1 || 
		template.Requirements.RequiredClass[0] != "warrior" {
		t.Errorf("Expected required class 'warrior'")
	}
	
	if len(template.Requirements.Forbidden) != 1 || 
		template.Requirements.Forbidden[0] != "mage" {
		t.Errorf("Expected forbidden class 'mage'")
	}
}

func TestStatTypeConstants(t *testing.T) {
	// Test that stat constants are defined and sequential
	expectedStats := []StatType{
		StatStrength,
		StatDexterity,
		StatIntelligence,
		StatConstitution,
		StatWisdom,
		StatCharisma,
	}
	
	for i, stat := range expectedStats {
		if int(stat) != i {
			t.Errorf("Expected stat constant %d to have value %d, got %d", 
				i, i, int(stat))
		}
	}
}

func TestItemTypeConstants(t *testing.T) {
	// Test that item type constants are defined and sequential
	expectedTypes := []ItemType{
		ItemWeapon,
		ItemArmor,
		ItemShield,
		ItemConsumable,
		ItemContainer,
		ItemKey,
		ItemTreasure,
		ItemTool,
		ItemMaterial,
	}
	
	for i, itemType := range expectedTypes {
		if int(itemType) != i {
			t.Errorf("Expected item type constant %d to have value %d, got %d", 
				i, i, int(itemType))
		}
	}
}

func TestRarityTypeConstants(t *testing.T) {
	// Test that rarity constants are defined and sequential
	expectedRarities := []RarityType{
		RarityCommon,
		RarityUncommon,
		RarityRare,
		RarityEpic,
		RarityLegendary,
	}
	
	for i, rarity := range expectedRarities {
		if int(rarity) != i {
			t.Errorf("Expected rarity constant %d to have value %d, got %d", 
				i, i, int(rarity))
		}
	}
}
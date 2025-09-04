package items

import (
	"testing"
	"time"
)

func TestNewItemInstance(t *testing.T) {
	templateID := "test_sword"
	ownerID := "test_player_123"
	quantity := 1
	
	instance := NewItemInstance(templateID, ownerID, quantity)
	
	// Test basic properties
	if instance.TemplateID != templateID {
		t.Errorf("Expected template ID %s, got %s", templateID, instance.TemplateID)
	}
	
	if instance.OwnerID != ownerID {
		t.Errorf("Expected owner ID %s, got %s", ownerID, instance.OwnerID)
	}
	
	if instance.Quantity != quantity {
		t.Errorf("Expected quantity %d, got %d", quantity, instance.Quantity)
	}
	
	// Test default values
	if instance.Durability != 100 {
		t.Errorf("Expected default durability 100, got %d", instance.Durability)
	}
	
	// Test initialized collections
	if instance.Enchantments == nil {
		t.Errorf("Expected enchantments slice to be initialized")
	}
	
	if len(instance.Enchantments) != 0 {
		t.Errorf("Expected empty enchantments initially, got %d", len(instance.Enchantments))
	}
	
	if instance.Modifications == nil {
		t.Errorf("Expected modifications map to be initialized")
	}
	
	// Test timestamps
	if instance.CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be set")
	}
	
	if !instance.LastUsed.IsZero() {
		t.Errorf("Expected LastUsed to be zero initially")
	}
}

func TestGetDisplayName(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	
	// Test default display name
	displayName := instance.GetDisplayName()
	if displayName != "Unknown Item" {
		t.Errorf("Expected default display name 'Unknown Item', got %s", displayName)
	}
	
	// Test custom name
	instance.CustomName = "Excalibur"
	displayName = instance.GetDisplayName()
	if displayName != "Excalibur" {
		t.Errorf("Expected custom name 'Excalibur', got %s", displayName)
	}
}

func TestIsBroken(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	
	// Should not be broken initially
	if instance.IsBroken() {
		t.Errorf("Item should not be broken initially")
	}
	
	// Set durability to 0
	instance.Durability = 0
	if !instance.IsBroken() {
		t.Errorf("Item should be broken with 0 durability")
	}
	
	// Negative durability should also be broken
	instance.Durability = -5
	if !instance.IsBroken() {
		t.Errorf("Item should be broken with negative durability")
	}
}

func TestTakeDamage(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	instance.Durability = 50
	
	// Take some damage
	instance.TakeDamage(10)
	if instance.Durability != 40 {
		t.Errorf("Expected durability 40 after taking 10 damage, got %d", instance.Durability)
	}
	
	// Take damage that would go below 0
	instance.TakeDamage(60)
	if instance.Durability != 0 {
		t.Errorf("Expected durability to be clamped at 0, got %d", instance.Durability)
	}
}

func TestRepair(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	instance.Durability = 30
	
	// Repair the item
	instance.Repair(20)
	if instance.Durability != 50 {
		t.Errorf("Expected durability 50 after repair, got %d", instance.Durability)
	}
	
	// Note: This test shows the current implementation doesn't check max durability
	// In a real implementation, you'd want to prevent over-repairing
}

func TestEnchantmentManagement(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	
	// Create test enchantment
	enchantment := Enchantment{
		ID:          "sharpness",
		Name:        "Sharpness",
		Description: "Makes the weapon sharper",
		Type:        EnchantmentDamage,
		Power:       5,
		Duration:    time.Hour,
	}
	
	// Add enchantment
	instance.AddEnchantment(enchantment)
	
	if len(instance.Enchantments) != 1 {
		t.Errorf("Expected 1 enchantment, got %d", len(instance.Enchantments))
	}
	
	// Check that AppliedAt was set
	if instance.Enchantments[0].AppliedAt.IsZero() {
		t.Errorf("Expected AppliedAt to be set when adding enchantment")
	}
	
	// Test HasEnchantment
	if !instance.HasEnchantment(EnchantmentDamage) {
		t.Errorf("Expected to have damage enchantment")
	}
	
	if instance.HasEnchantment(EnchantmentDefense) {
		t.Errorf("Should not have defense enchantment")
	}
	
	// Test GetEnchantmentBonus
	bonus := instance.GetEnchantmentBonus(EnchantmentDamage)
	if bonus != 5 {
		t.Errorf("Expected damage bonus 5, got %d", bonus)
	}
	
	// Add another enchantment of same type
	enchantment2 := Enchantment{
		ID:    "keen",
		Name:  "Keen",
		Type:  EnchantmentDamage,
		Power: 3,
	}
	instance.AddEnchantment(enchantment2)
	
	// Should stack bonuses
	bonus = instance.GetEnchantmentBonus(EnchantmentDamage)
	if bonus != 8 {
		t.Errorf("Expected combined damage bonus 8, got %d", bonus)
	}
	
	// Remove enchantment
	removed := instance.RemoveEnchantment("sharpness")
	if !removed {
		t.Errorf("Expected to successfully remove enchantment")
	}
	
	if len(instance.Enchantments) != 1 {
		t.Errorf("Expected 1 enchantment after removal, got %d", len(instance.Enchantments))
	}
	
	// Try to remove non-existent enchantment
	removed = instance.RemoveEnchantment("nonexistent")
	if removed {
		t.Errorf("Should not have removed non-existent enchantment")
	}
}

func TestUpdateLastUsed(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	
	// Initially should be zero
	if !instance.LastUsed.IsZero() {
		t.Errorf("Expected LastUsed to be zero initially")
	}
	
	// Update last used
	before := time.Now()
	instance.UpdateLastUsed()
	after := time.Now()
	
	// Should be set to current time
	if instance.LastUsed.Before(before) || instance.LastUsed.After(after) {
		t.Errorf("Expected LastUsed to be set to current time")
	}
}

func TestCanStack(t *testing.T) {
	// Create two identical basic instances
	instance1 := NewItemInstance("potion", "player1", 5)
	instance2 := NewItemInstance("potion", "player1", 3)
	
	// Should be stackable
	if !instance1.CanStack(instance2) {
		t.Errorf("Expected identical items to be stackable")
	}
	
	// Different templates
	instance3 := NewItemInstance("sword", "player1", 1)
	if instance1.CanStack(instance3) {
		t.Errorf("Different item templates should not be stackable")
	}
	
	// Different durability
	instance4 := NewItemInstance("potion", "player1", 2)
	instance4.Durability = 50
	if instance1.CanStack(instance4) {
		t.Errorf("Items with different durability should not be stackable")
	}
	
	// With enchantments
	instance1.AddEnchantment(Enchantment{ID: "test", Type: EnchantmentDamage, Power: 1})
	instance5 := NewItemInstance("potion", "player1", 2)
	if instance1.CanStack(instance5) {
		t.Errorf("Enchanted items should not be stackable with non-enchanted")
	}
	
	// Different custom names
	instance6 := NewItemInstance("potion", "player1", 2)
	instance7 := NewItemInstance("potion", "player1", 2)
	instance7.CustomName = "Special Potion"
	if instance6.CanStack(instance7) {
		t.Errorf("Items with different custom names should not be stackable")
	}
}

func TestEnchantmentTypes(t *testing.T) {
	// Test that enchantment type constants are defined
	types := []EnchantmentType{
		EnchantmentDamage,
		EnchantmentDefense,
		EnchantmentStat,
		EnchantmentResistance,
		EnchantmentSpecial,
	}
	
	for i, enchType := range types {
		if int(enchType) != i {
			t.Errorf("Expected enchantment type %d to have value %d, got %d", 
				i, i, int(enchType))
		}
	}
}

func TestEnchantmentDuration(t *testing.T) {
	instance := NewItemInstance("sword", "player1", 1)
	
	// Create temporary enchantment
	tempEnchantment := Enchantment{
		ID:       "temp_boost",
		Name:     "Temporary Boost",
		Type:     EnchantmentDamage,
		Power:    10,
		Duration: time.Millisecond * 100, // Very short for testing
	}
	
	instance.AddEnchantment(tempEnchantment)
	
	// Verify it was added
	if !instance.HasEnchantment(EnchantmentDamage) {
		t.Errorf("Expected enchantment to be added")
	}
	
	// Note: In a real game, you'd have a system to check expiration
	// This test just verifies the Duration field exists and can be set
	if instance.Enchantments[0].Duration != time.Millisecond*100 {
		t.Errorf("Expected duration to be preserved")
	}
}
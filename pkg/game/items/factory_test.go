package items

import (
	"testing"
)

func TestNewItemFactory(t *testing.T) {
	factory := NewItemFactory()
	
	if factory == nil {
		t.Fatalf("NewItemFactory returned nil")
	}
	
	if factory.registry == nil {
		t.Fatalf("Factory registry is nil")
	}
}

func TestCreateInstance(t *testing.T) {
	factory := NewItemFactory()
	
	// Test creating a valid item
	instance, err := factory.CreateInstance("rusty_sword", "player123", 1)
	if err != nil {
		t.Fatalf("Expected no error creating valid item, got: %v", err)
	}
	
	if instance == nil {
		t.Fatalf("Expected instance to be created")
	}
	
	if instance.TemplateID != "rusty_sword" {
		t.Errorf("Expected template ID 'rusty_sword', got %s", instance.TemplateID)
	}
	
	if instance.OwnerID != "player123" {
		t.Errorf("Expected owner ID 'player123', got %s", instance.OwnerID)
	}
	
	if instance.Quantity != 1 {
		t.Errorf("Expected quantity 1, got %d", instance.Quantity)
	}
	
	// Should have ID assigned
	if instance.ID == "" {
		t.Errorf("Expected instance ID to be generated")
	}
}

func TestCreateInstanceInvalidTemplate(t *testing.T) {
	factory := NewItemFactory()
	
	// Test creating item with invalid template
	instance, err := factory.CreateInstance("nonexistent_item", "player123", 1)
	if err == nil {
		t.Errorf("Expected error for nonexistent template")
	}
	
	if instance != nil {
		t.Errorf("Expected nil instance for invalid template")
	}
}

func TestCreateInstanceQuantityValidation(t *testing.T) {
	factory := NewItemFactory()
	
	// Test zero quantity (should default to 1)
	instance, err := factory.CreateInstance("rusty_sword", "player123", 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if instance.Quantity != 1 {
		t.Errorf("Expected quantity to default to 1, got %d", instance.Quantity)
	}
	
	// Test negative quantity (should default to 1)
	instance, err = factory.CreateInstance("rusty_sword", "player123", -5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if instance.Quantity != 1 {
		t.Errorf("Expected quantity to default to 1, got %d", instance.Quantity)
	}
}

func TestCreateInstanceStackableItems(t *testing.T) {
	factory := NewItemFactory()
	
	// Test creating stackable item (health potion has stack size 10)
	instance, err := factory.CreateInstance("health_potion", "player123", 5)
	if err != nil {
		t.Fatalf("Unexpected error creating stackable item: %v", err)
	}
	
	if instance.Quantity != 5 {
		t.Errorf("Expected quantity 5 for stackable item, got %d", instance.Quantity)
	}
	
	// Test creating too many of a stackable item
	instance, err = factory.CreateInstance("health_potion", "player123", 15)
	if err == nil {
		t.Errorf("Expected error for quantity exceeding stack size")
	}
	
	if instance != nil {
		t.Errorf("Expected nil instance when exceeding stack size")
	}
}

func TestCreateInstanceNonStackableItems(t *testing.T) {
	factory := NewItemFactory()
	
	// Test creating multiple non-stackable items (should fail)
	instance, err := factory.CreateInstance("rusty_sword", "player123", 2)
	if err == nil {
		t.Errorf("Expected error for multiple non-stackable items")
	}
	
	if instance != nil {
		t.Errorf("Expected nil instance for invalid quantity of non-stackable item")
	}
}

func TestCreateInstanceDurabilityFromTemplate(t *testing.T) {
	factory := NewItemFactory()
	
	// Get template to check its durability
	template, err := factory.GetTemplate("rusty_sword")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}
	
	instance, err := factory.CreateInstance("rusty_sword", "player123", 1)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	
	// Instance should have same durability as template
	if instance.Durability != template.Durability {
		t.Errorf("Expected durability %d from template, got %d", 
			template.Durability, instance.Durability)
	}
}

func TestGetTemplate(t *testing.T) {
	factory := NewItemFactory()
	
	// Test getting existing template
	template, err := factory.GetTemplate("rusty_sword")
	if err != nil {
		t.Errorf("Unexpected error getting template: %v", err)
	}
	
	if template == nil {
		t.Errorf("Expected template to be returned")
	}
	
	if template.ID != "rusty_sword" {
		t.Errorf("Expected template ID 'rusty_sword', got %s", template.ID)
	}
	
	// Test getting non-existent template
	template, err = factory.GetTemplate("nonexistent")
	if err == nil {
		t.Errorf("Expected error for non-existent template")
	}
	
	if template != nil {
		t.Errorf("Expected nil template for non-existent ID")
	}
}

func TestGetAllTemplates(t *testing.T) {
	factory := NewItemFactory()
	
	templates := factory.GetAllTemplates()
	
	if len(templates) == 0 {
		t.Errorf("Expected at least one template")
	}
	
	// Check that default templates exist
	expectedTemplates := []string{
		"rusty_sword",
		"leather_armor", 
		"health_potion",
		"magic_staff",
	}
	
	for _, expectedID := range expectedTemplates {
		if _, exists := templates[expectedID]; !exists {
			t.Errorf("Expected template %s to exist", expectedID)
		}
	}
}

func TestGetTemplatesByType(t *testing.T) {
	factory := NewItemFactory()
	
	// Test getting weapons
	weapons := factory.GetTemplatesByType(ItemWeapon)
	if len(weapons) == 0 {
		t.Errorf("Expected at least one weapon template")
	}
	
	// Verify all returned items are weapons
	for _, template := range weapons {
		if template.Type != ItemWeapon {
			t.Errorf("Expected weapon type, got %d", template.Type)
		}
	}
	
	// Test getting consumables
	consumables := factory.GetTemplatesByType(ItemConsumable)
	if len(consumables) == 0 {
		t.Errorf("Expected at least one consumable template")
	}
	
	// Test getting non-existent type
	materials := factory.GetTemplatesByType(ItemMaterial)
	if len(materials) != 0 {
		t.Errorf("Expected no material templates, got %d", len(materials))
	}
}

func TestRegisterTemplate(t *testing.T) {
	factory := NewItemFactory()
	
	// Create custom template
	customTemplate := NewItemTemplate("custom_weapon", "Custom Weapon", ItemWeapon)
	customTemplate.BaseStats.Damage = 20
	
	// Register it
	err := factory.RegisterTemplate(customTemplate)
	if err != nil {
		t.Errorf("Unexpected error registering template: %v", err)
	}
	
	// Verify it was registered
	retrieved, err := factory.GetTemplate("custom_weapon")
	if err != nil {
		t.Errorf("Failed to retrieve registered template: %v", err)
	}
	
	if retrieved.BaseStats.Damage != 20 {
		t.Errorf("Expected damage 20, got %d", retrieved.BaseStats.Damage)
	}
	
	// Test registering nil template
	err = factory.RegisterTemplate(nil)
	if err == nil {
		t.Errorf("Expected error registering nil template")
	}
}

func TestCreateEnchantedInstance(t *testing.T) {
	factory := NewItemFactory()
	
	// Create enchantments
	enchantments := []Enchantment{
		{
			ID:    "sharpness",
			Name:  "Sharpness",
			Type:  EnchantmentDamage,
			Power: 5,
		},
		{
			ID:    "durability",
			Name:  "Durability",
			Type:  EnchantmentSpecial,
			Power: 10,
		},
	}
	
	// Create enchanted instance
	instance, err := factory.CreateEnchantedInstance("rusty_sword", "player123", enchantments)
	if err != nil {
		t.Fatalf("Unexpected error creating enchanted instance: %v", err)
	}
	
	if len(instance.Enchantments) != 2 {
		t.Errorf("Expected 2 enchantments, got %d", len(instance.Enchantments))
	}
	
	// Verify enchantments were applied correctly
	if !instance.HasEnchantment(EnchantmentDamage) {
		t.Errorf("Expected damage enchantment")
	}
	
	if !instance.HasEnchantment(EnchantmentSpecial) {
		t.Errorf("Expected special enchantment")
	}
	
	// Test enchanting non-enchantable item
	nonEnchantableTemplate := NewItemTemplate("basic_material", "Basic Material", ItemMaterial)
	nonEnchantableTemplate.Enchantable = false
	factory.RegisterTemplate(nonEnchantableTemplate)
	
	instance, err = factory.CreateEnchantedInstance("basic_material", "player123", enchantments)
	if err == nil {
		t.Errorf("Expected error enchanting non-enchantable item")
	}
}

func TestCreateCustomInstance(t *testing.T) {
	factory := NewItemFactory()
	
	customName := "Legendary Blade of Testing"
	instance, err := factory.CreateCustomInstance("rusty_sword", "player123", customName)
	if err != nil {
		t.Fatalf("Unexpected error creating custom instance: %v", err)
	}
	
	if instance.CustomName != customName {
		t.Errorf("Expected custom name %s, got %s", customName, instance.CustomName)
	}
	
	// Verify display name uses custom name
	displayName := instance.GetDisplayName()
	if displayName != customName {
		t.Errorf("Expected display name %s, got %s", customName, displayName)
	}
}

func TestDefaultTemplatesExist(t *testing.T) {
	factory := NewItemFactory()
	
	// Test that default templates were loaded
	defaultTemplates := map[string]ItemType{
		"rusty_sword":    ItemWeapon,
		"leather_armor":  ItemArmor,
		"health_potion":  ItemConsumable,
		"magic_staff":    ItemWeapon,
	}
	
	for templateID, expectedType := range defaultTemplates {
		template, err := factory.GetTemplate(templateID)
		if err != nil {
			t.Errorf("Default template %s not found: %v", templateID, err)
			continue
		}
		
		if template.Type != expectedType {
			t.Errorf("Expected %s to be type %d, got %d", 
				templateID, expectedType, template.Type)
		}
	}
}

func TestMagicStaffRequirements(t *testing.T) {
	factory := NewItemFactory()
	
	template, err := factory.GetTemplate("magic_staff")
	if err != nil {
		t.Fatalf("Failed to get magic staff template: %v", err)
	}
	
	// Check requirements
	if template.Requirements.MinLevel != 3 {
		t.Errorf("Expected min level 3, got %d", template.Requirements.MinLevel)
	}
	
	if template.Requirements.MinStats[StatIntelligence] != 12 {
		t.Errorf("Expected min intelligence 12, got %d", 
			template.Requirements.MinStats[StatIntelligence])
	}
	
	if len(template.Requirements.RequiredClass) != 1 || 
		template.Requirements.RequiredClass[0] != "mage" {
		t.Errorf("Expected required class 'mage'")
	}
}
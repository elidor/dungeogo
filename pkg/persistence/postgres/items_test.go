package postgres

import (
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/items"
)

func TestItemRepository_CreateItemInstance(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	testItem := createTestItemInstance()

	err := repo.CreateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to create item instance: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetItemInstance(testItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve item instance: %v", err)
	}

	if retrieved.ID != testItem.ID {
		t.Errorf("Expected ID %s, got %s", testItem.ID, retrieved.ID)
	}

	if retrieved.TemplateID != testItem.TemplateID {
		t.Errorf("Expected template ID %s, got %s", testItem.TemplateID, retrieved.TemplateID)
	}

	if retrieved.OwnerID != testItem.OwnerID {
		t.Errorf("Expected owner ID %s, got %s", testItem.OwnerID, retrieved.OwnerID)
	}

	if retrieved.Quantity != testItem.Quantity {
		t.Errorf("Expected quantity %d, got %d", testItem.Quantity, retrieved.Quantity)
	}

	if retrieved.Durability != testItem.Durability {
		t.Errorf("Expected durability %d, got %d", testItem.Durability, retrieved.Durability)
	}
}

func TestItemRepository_UpdateItemInstance(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	testItem := createTestItemInstance()

	// Create item first
	err := repo.CreateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to create item instance: %v", err)
	}

	// Update item
	testItem.Durability = 50
	testItem.CustomName = "Epic Sword of Testing"
	testItem.LastUsed = time.Now()
	testItem.Modifications["upgraded"] = true

	err = repo.UpdateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to update item instance: %v", err)
	}

	// Retrieve and verify updates
	retrieved, err := repo.GetItemInstance(testItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated item: %v", err)
	}

	if retrieved.Durability != 50 {
		t.Errorf("Expected durability 50, got %d", retrieved.Durability)
	}

	if retrieved.CustomName != "Epic Sword of Testing" {
		t.Errorf("Expected custom name 'Epic Sword of Testing', got %s", retrieved.CustomName)
	}

	if retrieved.LastUsed.IsZero() {
		t.Errorf("Expected LastUsed to be set")
	}

	if upgraded, exists := retrieved.Modifications["upgraded"]; !exists || upgraded != true {
		t.Errorf("Expected modification 'upgraded' to be true")
	}
}

func TestItemRepository_GetPlayerItems(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	playerID := "test-player-" + generateUUID()

	// Create multiple items for the player
	item1 := createTestItemInstance()
	item1.OwnerID = playerID
	item1.TemplateID = "sword"

	item2 := createTestItemInstance()
	item2.OwnerID = playerID
	item2.TemplateID = "potion"
	item2.Quantity = 5

	item3 := createTestItemInstance()
	item3.OwnerID = "different-player"
	item3.TemplateID = "armor"

	// Create all items
	err := repo.CreateItemInstance(item1)
	if err != nil {
		t.Fatalf("Failed to create item 1: %v", err)
	}

	err = repo.CreateItemInstance(item2)
	if err != nil {
		t.Fatalf("Failed to create item 2: %v", err)
	}

	err = repo.CreateItemInstance(item3)
	if err != nil {
		t.Fatalf("Failed to create item 3: %v", err)
	}

	// Get items for specific player
	playerItems, err := repo.GetPlayerItems(playerID)
	if err != nil {
		t.Fatalf("Failed to get player items: %v", err)
	}

	if len(playerItems) != 2 {
		t.Errorf("Expected 2 items for player, got %d", len(playerItems))
	}

	// Verify we got the right items
	var foundSword, foundPotion bool
	for _, item := range playerItems {
		if item.TemplateID == "sword" {
			foundSword = true
		} else if item.TemplateID == "potion" {
			foundPotion = true
			if item.Quantity != 5 {
				t.Errorf("Expected potion quantity 5, got %d", item.Quantity)
			}
		}
	}

	if !foundSword || !foundPotion {
		t.Errorf("Expected to find both sword and potion items")
	}
}

func TestItemRepository_TransferItem(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	originalOwner := "player-1-" + generateUUID()
	newOwner := "player-2-" + generateUUID()

	// Create item for original owner
	testItem := createTestItemInstance()
	testItem.OwnerID = originalOwner

	err := repo.CreateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// Transfer item
	err = repo.TransferItem(testItem.ID, newOwner)
	if err != nil {
		t.Fatalf("Failed to transfer item: %v", err)
	}

	// Verify transfer
	retrieved, err := repo.GetItemInstance(testItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transferred item: %v", err)
	}

	if retrieved.OwnerID != newOwner {
		t.Errorf("Expected new owner %s, got %s", newOwner, retrieved.OwnerID)
	}

	// Verify original owner no longer has the item
	originalItems, err := repo.GetPlayerItems(originalOwner)
	if err != nil {
		t.Fatalf("Failed to get original owner items: %v", err)
	}

	if len(originalItems) != 0 {
		t.Errorf("Expected original owner to have 0 items after transfer, got %d", len(originalItems))
	}

	// Verify new owner has the item
	newOwnerItems, err := repo.GetPlayerItems(newOwner)
	if err != nil {
		t.Fatalf("Failed to get new owner items: %v", err)
	}

	if len(newOwnerItems) != 1 {
		t.Errorf("Expected new owner to have 1 item after transfer, got %d", len(newOwnerItems))
	}
}

func TestItemRepository_DeleteItemInstance(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	testItem := createTestItemInstance()

	// Create item first
	err := repo.CreateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to create item instance: %v", err)
	}

	// Delete item
	err = repo.DeleteItemInstance(testItem.ID)
	if err != nil {
		t.Fatalf("Failed to delete item instance: %v", err)
	}

	// Verify item was deleted
	_, err = repo.GetItemInstance(testItem.ID)
	if err == nil {
		t.Errorf("Expected error when retrieving deleted item")
	}
}

func TestItemRepository_EnchantmentSerialization(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	testItem := createTestItemInstance()

	// Add enchantments
	enchantment1 := items.Enchantment{
		ID:          "sharpness",
		Name:        "Sharpness",
		Description: "Makes weapon sharper",
		Type:        items.EnchantmentDamage,
		Power:       5,
		Duration:    time.Hour,
		AppliedAt:   time.Now(),
	}

	enchantment2 := items.Enchantment{
		ID:          "durability",
		Name:        "Durability",
		Description: "Increases item durability",
		Type:        items.EnchantmentSpecial,
		Power:       10,
		Duration:    0, // Permanent
		AppliedAt:   time.Now(),
	}

	testItem.Enchantments = []items.Enchantment{enchantment1, enchantment2}

	// Create item
	err := repo.CreateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to create enchanted item: %v", err)
	}

	// Retrieve and verify enchantments
	retrieved, err := repo.GetItemInstance(testItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve enchanted item: %v", err)
	}

	if len(retrieved.Enchantments) != 2 {
		t.Errorf("Expected 2 enchantments, got %d", len(retrieved.Enchantments))
	}

	// Verify first enchantment
	var foundSharpness, foundDurability bool
	for _, ench := range retrieved.Enchantments {
		if ench.ID == "sharpness" {
			foundSharpness = true
			if ench.Power != 5 {
				t.Errorf("Expected sharpness power 5, got %d", ench.Power)
			}
			if ench.Type != items.EnchantmentDamage {
				t.Errorf("Expected damage enchantment type")
			}
		} else if ench.ID == "durability" {
			foundDurability = true
			if ench.Power != 10 {
				t.Errorf("Expected durability power 10, got %d", ench.Power)
			}
		}
	}

	if !foundSharpness || !foundDurability {
		t.Errorf("Expected to find both enchantments after deserialization")
	}
}

func TestItemRepository_GetRoomItems(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Items()
	roomID := "tavern_main_room"

	// Create items in room
	item1 := createTestItemInstance()
	item1.OwnerID = roomID
	item1.TemplateID = "dropped_gold"

	item2 := createTestItemInstance()
	item2.OwnerID = roomID
	item2.TemplateID = "abandoned_sword"

	err := repo.CreateItemInstance(item1)
	if err != nil {
		t.Fatalf("Failed to create room item 1: %v", err)
	}

	err = repo.CreateItemInstance(item2)
	if err != nil {
		t.Fatalf("Failed to create room item 2: %v", err)
	}

	// Get room items
	roomItems, err := repo.GetRoomItems(roomID)
	if err != nil {
		t.Fatalf("Failed to get room items: %v", err)
	}

	if len(roomItems) != 2 {
		t.Errorf("Expected 2 items in room, got %d", len(roomItems))
	}

	// Verify items belong to room
	for _, item := range roomItems {
		if item.OwnerID != roomID {
			t.Errorf("Expected item to belong to room %s, got %s", roomID, item.OwnerID)
		}
	}
}


package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
	"github.com/lib/pq"
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

	if retrieved.Owner.ID != testItem.Owner.ID {
		t.Errorf("Expected owner ID %s, got %s", testItem.Owner.ID, retrieved.Owner.ID)
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
	playerID := existingItemCharacter(t, repoManager)

	// Create multiple items for the player
	item1 := createTestItemInstance()
	item1.Owner = items.CharacterOwner(playerID)
	item1.TemplateID = "sword"

	item2 := createTestItemInstance()
	item2.Owner = items.CharacterOwner(playerID)
	item2.TemplateID = "potion"
	item2.Quantity = 5

	item3 := createTestItemInstance()
	item3.Owner = items.RoomOwner("newbie_west")
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
	originalOwner := existingItemCharacter(t, repoManager)
	newOwner := existingItemCharacter(t, repoManager)

	// Create item for original owner
	testItem := createTestItemInstance()
	testItem.Owner = items.CharacterOwner(originalOwner)

	err := repo.CreateItemInstance(testItem)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// Transfer item
	moved, err := repo.TransferItem(testItem.ID, items.CharacterOwner(originalOwner), items.CharacterOwner(newOwner))
	if err != nil || !moved {
		t.Fatalf("Failed to transfer item: moved=%v err=%v", moved, err)
	}

	// Verify transfer
	retrieved, err := repo.GetItemInstance(testItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve transferred item: %v", err)
	}

	if retrieved.Owner.ID != newOwner {
		t.Errorf("Expected new owner %s, got %s", newOwner, retrieved.Owner.ID)
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
	roomID := "newbie_east"

	// Create items in room
	item1 := createTestItemInstance()
	item1.Owner = items.RoomOwner(roomID)
	item1.TemplateID = "dropped_gold"

	item2 := createTestItemInstance()
	item2.Owner = items.RoomOwner(roomID)
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
		if item.Owner.ID != roomID {
			t.Errorf("Expected item to belong to room %s, got %s", roomID, item.Owner.ID)
		}
	}
}

// TestItemRepository_CreateAndUpdateShareSaveSteps checks the fields both
// writes prepare the same way: JSON columns, nullable last_used, and owner.
func TestItemRepository_CreateAndUpdateShareSaveSteps(t *testing.T) {
	_, url := testdb.New(t, true)
	manager, err := NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	repo := manager.Items()

	item := createTestItemInstance()
	item.LastUsed = time.Now().UTC().Truncate(time.Microsecond)
	item.Modifications["sharpened"] = true
	item.Enchantments = []items.Enchantment{{ID: "e1", Name: "Keen", Type: items.EnchantmentDamage, Power: 2}}
	if err := repo.CreateItemInstance(item); err != nil {
		t.Fatal(err)
	}
	created, err := repo.GetItemInstance(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !created.LastUsed.Equal(item.LastUsed) || created.Modifications["sharpened"] != true || len(created.Enchantments) != 1 || created.Owner != items.RoomOwner("newbie_west") {
		t.Fatalf("create did not store prepared fields: %+v", created)
	}

	created.LastUsed = time.Time{}
	created.Modifications = map[string]interface{}{"dulled": true}
	created.Enchantments = []items.Enchantment{}
	created.Owner = items.RoomOwner("newbie_east") // a stale or edited copy must not move the Item
	if err := repo.UpdateItemInstance(created); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.GetItemInstance(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	var lastUsedIsNull bool
	if err := manager.GetDB().QueryRow(`SELECT last_used IS NULL FROM item_instances WHERE id=$1`, item.ID).Scan(&lastUsedIsNull); err != nil {
		t.Fatal(err)
	}
	if !lastUsedIsNull || !updated.LastUsed.IsZero() || updated.Modifications["dulled"] != true || len(updated.Enchantments) != 0 || updated.Owner != items.RoomOwner("newbie_west") {
		t.Fatalf("update did not store prepared fields: %+v (last_used null=%v)", updated, lastUsedIsNull)
	}
}

func TestItemStorageErrorsNameTheItem(t *testing.T) {
	manager := setupTestDB(t)
	repo := manager.Items()
	missing := "90000000-0000-4000-8000-0000000000ee"

	_, getErr := repo.GetItemInstance(missing)
	ghost := createTestItemInstance()
	ghost.ID = missing
	for name, err := range map[string]error{
		"get":    getErr,
		"update": repo.UpdateItemInstance(ghost),
		"delete": repo.DeleteItemInstance(missing),
	} {
		if !errors.Is(err, interfaces.ErrItemNotFound) {
			t.Errorf("%s missing item: error %v is not ErrItemNotFound", name, err)
		}
		if err == nil || !strings.Contains(err.Error(), missing) {
			t.Errorf("%s missing item: error %v does not name the item", name, err)
		}
	}

	// A real storage failure keeps its cause and names the Item.
	item := createTestItemInstance()
	if err := repo.CreateItemInstance(item); err != nil {
		t.Fatal(err)
	}
	err := repo.CreateItemInstance(item)
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		t.Fatalf("duplicate create: want wrapped unique violation, got %v", err)
	}
	if !strings.Contains(err.Error(), item.ID) || !strings.HasPrefix(err.Error(), "failed to create item") {
		t.Fatalf("duplicate create error %q should start with 'failed to create item' and name %s", err, item.ID)
	}
	if errors.Is(err, interfaces.ErrItemNotFound) {
		t.Fatalf("storage failure misreported as not found: %v", err)
	}
}

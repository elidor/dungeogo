package postgres

import (
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/player"
)

func TestPlayerRepository_CreatePlayer(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()
	testPlayer := createTestPlayer()

	err := repo.CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	// Verify player was created by retrieving it
	retrieved, err := repo.GetPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve created player: %v", err)
	}

	// Check basic fields
	if retrieved.ID != testPlayer.ID {
		t.Errorf("Expected ID %s, got %s", testPlayer.ID, retrieved.ID)
	}

	if retrieved.Username != testPlayer.Username {
		t.Errorf("Expected username %s, got %s", testPlayer.Username, retrieved.Username)
	}

	if retrieved.Email != testPlayer.Email {
		t.Errorf("Expected email %s, got %s", testPlayer.Email, retrieved.Email)
	}

	if retrieved.AccountStatus != testPlayer.AccountStatus {
		t.Errorf("Expected account status %d, got %d", testPlayer.AccountStatus, retrieved.AccountStatus)
	}

	// Check preferences were preserved
	if retrieved.Preferences.ColorEnabled != testPlayer.Preferences.ColorEnabled {
		t.Errorf("Expected ColorEnabled %v, got %v", testPlayer.Preferences.ColorEnabled, retrieved.Preferences.ColorEnabled)
	}

	if retrieved.Preferences.ScreenWidth != testPlayer.Preferences.ScreenWidth {
		t.Errorf("Expected ScreenWidth %d, got %d", testPlayer.Preferences.ScreenWidth, retrieved.Preferences.ScreenWidth)
	}
}

func TestPlayerRepository_GetPlayerByUsername(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()
	testPlayer := createTestPlayer()

	// Create player first
	err := repo.CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	// Retrieve by username
	retrieved, err := repo.GetPlayerByUsername(testPlayer.Username)
	if err != nil {
		t.Fatalf("Failed to retrieve player by username: %v", err)
	}

	if retrieved.ID != testPlayer.ID {
		t.Errorf("Expected ID %s, got %s", testPlayer.ID, retrieved.ID)
	}

	if retrieved.Username != testPlayer.Username {
		t.Errorf("Expected username %s, got %s", testPlayer.Username, retrieved.Username)
	}

	// Test non-existent username
	_, err = repo.GetPlayerByUsername("nonexistent")
	if err == nil {
		t.Errorf("Expected error for non-existent username")
	}
}

func TestPlayerRepository_UpdatePlayer(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()
	testPlayer := createTestPlayer()

	// Create player first
	err := repo.CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	// Update player
	testPlayer.Email = "updated@example.com"
	testPlayer.MaxCharacters = 10
	testPlayer.Preferences.ScreenWidth = 120
	testPlayer.Preferences.AutoLoot = true

	err = repo.UpdatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to update player: %v", err)
	}

	// Retrieve and verify updates
	retrieved, err := repo.GetPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated player: %v", err)
	}

	if retrieved.Email != "updated@example.com" {
		t.Errorf("Expected updated email, got %s", retrieved.Email)
	}

	if retrieved.MaxCharacters != 10 {
		t.Errorf("Expected max characters 10, got %d", retrieved.MaxCharacters)
	}

	if retrieved.Preferences.ScreenWidth != 120 {
		t.Errorf("Expected screen width 120, got %d", retrieved.Preferences.ScreenWidth)
	}

	if !retrieved.Preferences.AutoLoot {
		t.Errorf("Expected auto loot to be enabled")
	}
}

func TestPlayerRepository_UpdatePlayerLogin(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()
	testPlayer := createTestPlayer()

	// Set an old last login time
	oldTime := time.Now().Add(-24 * time.Hour)
	testPlayer.LastLogin = oldTime

	err := repo.CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	// Update login time
	err = repo.UpdatePlayerLogin(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to update player login: %v", err)
	}

	// Retrieve and verify login time was updated
	retrieved, err := repo.GetPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve player: %v", err)
	}

	if !retrieved.LastLogin.After(oldTime) {
		t.Errorf("Expected last login to be updated from %v to after %v, got %v", 
			oldTime, oldTime, retrieved.LastLogin)
	}
}

func TestPlayerRepository_DeletePlayer(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()
	testPlayer := createTestPlayer()

	// Create player first
	err := repo.CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	// Delete player
	err = repo.DeletePlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to delete player: %v", err)
	}

	// Verify player was deleted
	_, err = repo.GetPlayer(testPlayer.ID)
	if err == nil {
		t.Errorf("Expected error when retrieving deleted player")
	}
}

func TestPlayerRepository_UniqueConstraints(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()

	// Create first player
	player1 := createTestPlayer()
	player1.Username = "testuser"
	player1.Email = "test@example.com"

	err := repo.CreatePlayer(player1)
	if err != nil {
		t.Fatalf("Failed to create first player: %v", err)
	}

	// Try to create second player with same username
	player2 := createTestPlayer()
	player2.Username = "testuser" // Same username
	player2.Email = "different@example.com"

	err = repo.CreatePlayer(player2)
	if err == nil {
		t.Errorf("Expected error when creating player with duplicate username")
	}

	// Try to create player with same email
	player3 := createTestPlayer()
	player3.Username = "differentuser"
	player3.Email = "test@example.com" // Same email

	err = repo.CreatePlayer(player3)
	if err == nil {
		t.Errorf("Expected error when creating player with duplicate email")
	}
}

func TestPlayerRepository_Subscription(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	repo := repoManager.Players()
	testPlayer := createTestPlayer()

	// Add subscription
	testPlayer.Subscription = &player.Subscription{
		Type:      player.SubscriptionPremium,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		Active:    true,
	}

	err := repo.CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create player with subscription: %v", err)
	}

	// Retrieve and verify subscription
	retrieved, err := repo.GetPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve player: %v", err)
	}

	if retrieved.Subscription == nil {
		t.Fatalf("Expected subscription to be preserved")
	}

	if retrieved.Subscription.Type != player.SubscriptionPremium {
		t.Errorf("Expected premium subscription, got %d", retrieved.Subscription.Type)
	}

	if !retrieved.Subscription.Active {
		t.Errorf("Expected subscription to be active")
	}

	// Test player without subscription
	player2 := createTestPlayer()
	player2.Username = "nosub"
	player2.Email = "nosub@example.com"
	player2.Subscription = nil

	err = repo.CreatePlayer(player2)
	if err != nil {
		t.Fatalf("Failed to create player without subscription: %v", err)
	}

	retrieved2, err := repo.GetPlayer(player2.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve player without subscription: %v", err)
	}

	if retrieved2.Subscription != nil {
		t.Errorf("Expected no subscription, got %v", retrieved2.Subscription)
	}
}


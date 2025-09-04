package player

import (
	"testing"
	"time"
)

func TestNewPlayer(t *testing.T) {
	username := "testuser"
	email := "test@example.com"
	passwordHash := "hashed_password"
	
	player := NewPlayer(username, email, passwordHash)
	
	// Test basic properties
	if player.Username != username {
		t.Errorf("Expected username %s, got %s", username, player.Username)
	}
	
	if player.Email != email {
		t.Errorf("Expected email %s, got %s", email, player.Email)
	}
	
	if player.PasswordHash != passwordHash {
		t.Errorf("Expected password hash %s, got %s", passwordHash, player.PasswordHash)
	}
	
	// Test default values
	if player.AccountStatus != AccountActive {
		t.Errorf("Expected account to be active by default")
	}
	
	if player.MaxCharacters != 5 {
		t.Errorf("Expected max characters 5, got %d", player.MaxCharacters)
	}
	
	// Test that CreatedAt was set
	if player.CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be set")
	}
	
	// Test preferences initialization
	if !player.Preferences.ColorEnabled {
		t.Errorf("Expected color to be enabled by default")
	}
	
	if player.Preferences.ScreenWidth != 80 {
		t.Errorf("Expected screen width 80, got %d", player.Preferences.ScreenWidth)
	}
	
	if player.Preferences.Keybindings == nil {
		t.Errorf("Expected keybindings map to be initialized")
	}
}

func TestIsActive(t *testing.T) {
	player := NewPlayer("test", "test@test.com", "hash")
	
	// Should be active by default
	if !player.IsActive() {
		t.Errorf("Expected player to be active")
	}
	
	// Test suspended account
	player.AccountStatus = AccountSuspended
	if player.IsActive() {
		t.Errorf("Expected suspended player to not be active")
	}
	
	// Test banned account
	player.AccountStatus = AccountBanned
	if player.IsActive() {
		t.Errorf("Expected banned player to not be active")
	}
}

func TestHasPremium(t *testing.T) {
	player := NewPlayer("test", "test@test.com", "hash")
	
	// No subscription by default
	if player.HasPremium() {
		t.Errorf("Expected player without subscription to not have premium")
	}
	
	// Add expired subscription
	player.Subscription = &Subscription{
		Type:      SubscriptionPremium,
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
		Active:    true,
	}
	
	if player.HasPremium() {
		t.Errorf("Expected player with expired subscription to not have premium")
	}
	
	// Add valid subscription
	player.Subscription = &Subscription{
		Type:      SubscriptionPremium,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Valid
		Active:    true,
	}
	
	if !player.HasPremium() {
		t.Errorf("Expected player with valid subscription to have premium")
	}
	
	// Test inactive subscription
	player.Subscription.Active = false
	if player.HasPremium() {
		t.Errorf("Expected player with inactive subscription to not have premium")
	}
	
	// Test free subscription
	player.Subscription.Type = SubscriptionFree
	player.Subscription.Active = true
	if player.HasPremium() {
		t.Errorf("Expected player with free subscription to not have premium")
	}
}

func TestUpdateLastLogin(t *testing.T) {
	player := NewPlayer("test", "test@test.com", "hash")
	
	initialLogin := player.LastLogin
	
	// Wait a bit and update
	time.Sleep(10 * time.Millisecond)
	player.UpdateLastLogin()
	
	if !player.LastLogin.After(initialLogin) {
		t.Errorf("Expected LastLogin to be updated")
	}
}

func TestAccountStatus(t *testing.T) {
	player := NewPlayer("test", "test@test.com", "hash")
	
	// Test all account statuses
	statuses := []AccountStatus{
		AccountActive,
		AccountSuspended,
		AccountBanned,
	}
	
	for _, status := range statuses {
		player.AccountStatus = status
		if player.AccountStatus != status {
			t.Errorf("Failed to set account status to %d", status)
		}
	}
}

func TestSubscriptionTypes(t *testing.T) {
	// Test that subscription type constants are defined
	types := []SubscriptionType{
		SubscriptionFree,
		SubscriptionPremium,
	}
	
	for i, subType := range types {
		if int(subType) != i {
			t.Errorf("Expected subscription type %d to have value %d, got %d", 
				i, i, int(subType))
		}
	}
}

func TestPlayerPreferences(t *testing.T) {
	player := NewPlayer("test", "test@test.com", "hash")
	
	// Test modifying preferences
	player.Preferences.ColorEnabled = false
	player.Preferences.ScreenWidth = 120
	player.Preferences.AutoLoot = true
	player.Preferences.CombatPrompts = false
	player.Preferences.Keybindings["north"] = "n"
	
	if player.Preferences.ColorEnabled {
		t.Errorf("Expected color to be disabled")
	}
	
	if player.Preferences.ScreenWidth != 120 {
		t.Errorf("Expected screen width 120, got %d", player.Preferences.ScreenWidth)
	}
	
	if !player.Preferences.AutoLoot {
		t.Errorf("Expected auto loot to be enabled")
	}
	
	if player.Preferences.CombatPrompts {
		t.Errorf("Expected combat prompts to be disabled")
	}
	
	if player.Preferences.Keybindings["north"] != "n" {
		t.Errorf("Expected keybinding for north to be 'n'")
	}
}

func TestSubscriptionEdgeCases(t *testing.T) {
	player := NewPlayer("test", "test@test.com", "hash")
	
	// Test nil subscription
	player.Subscription = nil
	if player.HasPremium() {
		t.Errorf("Expected nil subscription to not have premium")
	}
	
	// Test subscription expiring exactly now
	player.Subscription = &Subscription{
		Type:      SubscriptionPremium,
		ExpiresAt: time.Now(),
		Active:    true,
	}
	
	// Should not have premium (expired or expiring now)
	if player.HasPremium() {
		t.Errorf("Expected subscription expiring now to not have premium")
	}
}
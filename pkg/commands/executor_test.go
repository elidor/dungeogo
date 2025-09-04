package commands

import (
	"strings"
	"testing"

	"github.com/elidor/dungeogo/pkg/testutil"
)

func TestNewExecutor(t *testing.T) {
	// Skip database tests if no database available
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	if executor == nil {
		t.Fatalf("NewExecutor returned nil")
	}
	
	if executor.repoManager == nil {
		t.Fatalf("Executor repository manager is nil")
	}
	
	if executor.handlers == nil {
		t.Fatalf("Executor handlers map is nil")
	}
	
	// Check that handlers were initialized
	if len(executor.handlers) == 0 {
		t.Errorf("Expected handlers to be initialized")
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	cmd := &Command{
		Type:        CommandUnknown,
		Verb:        "nonexistent",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	if !strings.Contains(responses[0], "Unknown command") {
		t.Errorf("Expected unknown command message, got: %s", responses[0])
	}
}

func TestExecuteMovementCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	cmd := &Command{
		Type:        CommandMovement,
		Verb:        "north",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	if !strings.Contains(responses[0], "move north") {
		t.Errorf("Expected movement message, got: %s", responses[0])
	}
}

func TestExecuteCommunicationCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	cmd := &Command{
		Type:        CommandCommunication,
		Verb:        "say",
		Args:        []string{"hello", "world"},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	if !strings.Contains(responses[0], "You say: hello world") {
		t.Errorf("Expected say message, got: %s", responses[0])
	}
}

func TestExecuteTellCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	// Test valid tell command
	cmd := &Command{
		Type:        CommandCommunication,
		Verb:        "tell",
		Args:        []string{"bob", "hello", "there"},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	expected := "You tell bob: hello there"
	if responses[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, responses[0])
	}
	
	// Test tell with insufficient args
	cmd.Args = []string{"bob"} // Missing message
	responses, err = executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if !strings.Contains(responses[0], "Usage:") {
		t.Errorf("Expected usage message for insufficient args")
	}
}

func TestExecuteLookCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	// Test look at room (no args)
	cmd := &Command{
		Type:        CommandInformation,
		Verb:        "look",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) < 1 {
		t.Errorf("Expected at least 1 response for room description")
	}
	
	if !strings.Contains(responses[0], "Simple Room") {
		t.Errorf("Expected room name in response")
	}
	
	// Test look at target
	cmd.Args = []string{"sword"}
	responses, err = executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if !strings.Contains(responses[0], "look at sword") {
		t.Errorf("Expected target look message")
	}
}

func TestExecuteWhoCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	cmd := &Command{
		Type:        CommandInformation,
		Verb:        "who",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) < 3 {
		t.Errorf("Expected at least 3 responses for who command")
	}
	
	// Should show players online
	found := false
	for _, response := range responses {
		if strings.Contains(response, "online") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'online' to appear in who command output")
	}
}

func TestExecuteHelpCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	// Test general help
	cmd := &Command{
		Type:        CommandSystem,
		Verb:        "help",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) < 3 {
		t.Errorf("Expected multiple help responses")
	}
	
	// Should mention command categories
	found := false
	for _, response := range responses {
		if strings.Contains(response, "categories") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected help to mention command categories")
	}
	
	// Test help with topic
	cmd.Args = []string{"movement"}
	responses, err = executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) < 1 {
		t.Errorf("Expected response for movement help")
	}
	
	if !strings.Contains(strings.Join(responses, " "), "north") {
		t.Errorf("Expected movement help to mention north command")
	}
}

func TestExecuteCommandsCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	cmd := &Command{
		Type:        CommandSystem,
		Verb:        "commands",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) < 3 {
		t.Errorf("Expected multiple command category responses")
	}
	
	// Should list various command types
	categories := []string{"Movement", "Communication", "Information", "Inventory"}
	for _, category := range categories {
		found := false
		for _, response := range responses {
			if strings.Contains(response, category) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected commands output to mention %s", category)
		}
	}
}

func TestExecuteEmoteCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	cmd := &Command{
		Type:        CommandSocial,
		Verb:        "emote",
		Args:        []string{"dances", "around", "happily"},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	expected := "You dances around happily"
	if responses[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, responses[0])
	}
}

func TestExecuteSocialCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	// Test social command without target
	cmd := &Command{
		Type:        CommandSocial,
		Verb:        "smile",
		Args:        []string{},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if !strings.Contains(responses[0], "You smile.") {
		t.Errorf("Expected general smile message")
	}
	
	// Test social command with target
	cmd.Args = []string{"bob"}
	responses, err = executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if !strings.Contains(responses[0], "You smile at bob.") {
		t.Errorf("Expected targeted smile message")
	}
}

func TestExecuteUnimplementedCommand(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	// Create a command that exists in parser but not in executor
	cmd := &Command{
		Type:        CommandMagic,
		Verb:        "cast",
		Args:        []string{"fireball"},
		PlayerID:    "player1",
		CharacterID: "char1",
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	if !strings.Contains(responses[0], "not implemented") {
		t.Errorf("Expected 'not implemented' message, got: %s", responses[0])
	}
}

func TestExecuteWithDatabaseCharacter(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	// Create test player and character
	testPlayer := testutil.CreateTestPlayer()
	testPlayer.ID = "test-player-123"
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}
	
	testChar := testutil.CreateTestCharacter(testPlayer.ID)
	testChar.ID = "test-char-456"
	err = repoManager.Characters().CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create test character: %v", err)
	}
	
	executor := NewExecutor(repoManager)
	
	// Test score command with real character
	cmd := &Command{
		Type:        CommandInformation,
		Verb:        "score",
		Args:        []string{},
		PlayerID:    testPlayer.ID,
		CharacterID: testChar.ID,
	}
	
	responses, err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(responses) < 3 {
		t.Errorf("Expected multiple lines for score output")
	}
	
	// Should show character name
	found := false
	for _, response := range responses {
		if strings.Contains(response, testChar.Name) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected score to show character name")
	}
}

func TestHandlerInitialization(t *testing.T) {
	repoManager := testutil.SetupTestDB(t)
	if repoManager == nil {
		t.Skip("No database available for testing")
	}
	
	executor := NewExecutor(repoManager)
	
	// Test that key handlers are initialized
	expectedHandlers := []string{
		"north", "south", "east", "west",
		"say", "tell", "look", "inventory",
		"help", "commands", "emote", "smile",
	}
	
	for _, handlerName := range expectedHandlers {
		if _, exists := executor.handlers[handlerName]; !exists {
			t.Errorf("Expected handler '%s' to be initialized", handlerName)
		}
	}
}
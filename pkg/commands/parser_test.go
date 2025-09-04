package commands

import (
	"testing"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	
	if parser == nil {
		t.Fatalf("NewParser returned nil")
	}
	
	if parser.aliases == nil {
		t.Fatalf("Parser aliases map is nil")
	}
	
	if parser.commands == nil {
		t.Fatalf("Parser commands map is nil")
	}
	
	// Check that default commands were initialized
	if len(parser.commands) == 0 {
		t.Errorf("Expected commands to be initialized")
	}
	
	if len(parser.aliases) == 0 {
		t.Errorf("Expected aliases to be initialized")
	}
}

func TestParseBasicCommand(t *testing.T) {
	parser := NewParser()
	playerID := "player123"
	characterID := "char456"
	
	// Test simple command
	cmd := parser.Parse("look", playerID, characterID)
	
	if cmd.Type != CommandInformation {
		t.Errorf("Expected command type %d, got %d", CommandInformation, cmd.Type)
	}
	
	if cmd.Verb != "look" {
		t.Errorf("Expected verb 'look', got %s", cmd.Verb)
	}
	
	if len(cmd.Args) != 0 {
		t.Errorf("Expected no args, got %d", len(cmd.Args))
	}
	
	if cmd.RawInput != "look" {
		t.Errorf("Expected raw input 'look', got %s", cmd.RawInput)
	}
	
	if cmd.PlayerID != playerID {
		t.Errorf("Expected player ID %s, got %s", playerID, cmd.PlayerID)
	}
	
	if cmd.CharacterID != characterID {
		t.Errorf("Expected character ID %s, got %s", characterID, cmd.CharacterID)
	}
}

func TestParseCommandWithArgs(t *testing.T) {
	parser := NewParser()
	
	cmd := parser.Parse("say hello world", "player1", "char1")
	
	if cmd.Verb != "say" {
		t.Errorf("Expected verb 'say', got %s", cmd.Verb)
	}
	
	if len(cmd.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(cmd.Args))
	}
	
	if cmd.Args[0] != "hello" {
		t.Errorf("Expected first arg 'hello', got %s", cmd.Args[0])
	}
	
	if cmd.Args[1] != "world" {
		t.Errorf("Expected second arg 'world', got %s", cmd.Args[1])
	}
}

func TestParseEmptyInput(t *testing.T) {
	parser := NewParser()
	
	cmd := parser.Parse("", "player1", "char1")
	
	if cmd.Type != CommandUnknown {
		t.Errorf("Expected unknown command type for empty input")
	}
	
	if cmd.RawInput != "" {
		t.Errorf("Expected empty raw input")
	}
}

func TestParseWhitespaceInput(t *testing.T) {
	parser := NewParser()
	
	tests := []string{"   ", "\t\t", "  \n  "}
	
	for _, input := range tests {
		cmd := parser.Parse(input, "player1", "char1")
		
		if cmd.Type != CommandUnknown {
			t.Errorf("Expected unknown command type for whitespace input '%s'", input)
		}
	}
}

func TestParseAliases(t *testing.T) {
	parser := NewParser()
	
	// Test direction aliases
	tests := []struct {
		input    string
		expected string
	}{
		{"n", "north"},
		{"s", "south"},
		{"e", "east"},
		{"w", "west"},
		{"l", "look"},
		{"'", "say"},
		{"i", "inventory"},
	}
	
	for _, test := range tests {
		cmd := parser.Parse(test.input, "player1", "char1")
		if cmd.Verb != test.expected {
			t.Errorf("Expected alias '%s' to resolve to '%s', got '%s'", 
				test.input, test.expected, cmd.Verb)
		}
	}
}

func TestParseCommandTypes(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		input        string
		expectedType CommandType
	}{
		{"north", CommandMovement},
		{"south", CommandMovement},
		{"say hello", CommandCommunication},
		{"tell player hello", CommandCommunication},
		{"look", CommandInformation},
		{"inventory", CommandInventory},
		{"get sword", CommandInventory},
		{"kill monster", CommandCombat},
		{"cast fireball", CommandMagic},
		{"skills", CommandSkill},
		{"emote dances", CommandSocial},
		{"help", CommandSystem},
		{"nonexistent", CommandUnknown},
	}
	
	for _, test := range tests {
		cmd := parser.Parse(test.input, "player1", "char1")
		if cmd.Type != test.expectedType {
			t.Errorf("Expected command '%s' to have type %d, got %d", 
				test.input, test.expectedType, cmd.Type)
		}
	}
}

func TestGetCommandInfo(t *testing.T) {
	parser := NewParser()
	
	// Test getting info for existing command
	info, exists := parser.GetCommandInfo("look")
	if !exists {
		t.Errorf("Expected 'look' command to exist")
	}
	
	if info.Type != CommandInformation {
		t.Errorf("Expected look to be information type")
	}
	
	if info.Description == "" {
		t.Errorf("Expected look to have description")
	}
	
	// Test alias resolution
	info, exists = parser.GetCommandInfo("l")
	if !exists {
		t.Errorf("Expected alias 'l' to resolve")
	}
	
	if info.Type != CommandInformation {
		t.Errorf("Expected alias 'l' to resolve to look command")
	}
	
	// Test non-existent command
	_, exists = parser.GetCommandInfo("nonexistent")
	if exists {
		t.Errorf("Expected non-existent command to not exist")
	}
}

func TestGetCommandsByType(t *testing.T) {
	parser := NewParser()
	
	// Test getting movement commands
	movementCommands := parser.GetCommandsByType(CommandMovement)
	if len(movementCommands) == 0 {
		t.Errorf("Expected at least one movement command")
	}
	
	expectedMovement := []string{"north", "south", "east", "west", "up", "down"}
	for _, expected := range expectedMovement {
		found := false
		for _, cmd := range movementCommands {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected movement command '%s' to be found", expected)
		}
	}
	
	// Test getting communication commands
	commCommands := parser.GetCommandsByType(CommandCommunication)
	if len(commCommands) == 0 {
		t.Errorf("Expected at least one communication command")
	}
}

func TestCaseInsensitive(t *testing.T) {
	parser := NewParser()
	
	// Test that commands are case insensitive
	tests := []string{"LOOK", "Look", "lOoK", "NORTH", "North"}
	
	for _, input := range tests {
		cmd := parser.Parse(input, "player1", "char1")
		if cmd.Type == CommandUnknown {
			t.Errorf("Expected '%s' to be recognized (case insensitive)", input)
		}
	}
}

func TestCommandValidation(t *testing.T) {
	parser := NewParser()
	
	// Test command with minimum args requirement
	cmd := parser.Parse("tell", "player1", "char1")
	
	// This should create the command but validation should fail
	if cmd.Verb != "tell" {
		t.Errorf("Expected verb 'tell' even with insufficient args")
	}
	
	// The actual validation would happen in ValidateArgs method
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		cmdType  CommandType
		expected string
	}{
		{CommandMovement, "Movement"},
		{CommandCommunication, "Communication"},
		{CommandInventory, "Inventory"},
		{CommandCombat, "Combat"},
		{CommandMagic, "Magic"},
		{CommandSkill, "Skill"},
		{CommandInformation, "Information"},
		{CommandSystem, "System"},
		{CommandSocial, "Social"},
		{CommandAdmin, "Admin"},
		{CommandUnknown, "Unknown"},
	}
	
	for _, test := range tests {
		cmd := &Command{Type: test.cmdType}
		actual := cmd.GetTypeName()
		if actual != test.expected {
			t.Errorf("Expected type name '%s' for type %d, got '%s'", 
				test.expected, test.cmdType, actual)
		}
	}
}

func TestCommandConstants(t *testing.T) {
	// Test that command type constants are sequential
	types := []CommandType{
		CommandMovement,
		CommandCommunication,
		CommandInventory,
		CommandCombat,
		CommandMagic,
		CommandSkill,
		CommandInformation,
		CommandSystem,
		CommandSocial,
		CommandAdmin,
		CommandUnknown,
	}
	
	for i, cmdType := range types {
		if int(cmdType) != i {
			t.Errorf("Expected command type %d to have value %d, got %d", 
				i, i, int(cmdType))
		}
	}
}

func TestSpecificCommandDetails(t *testing.T) {
	parser := NewParser()
	
	// Test specific command configurations
	tests := []struct {
		command  string
		minArgs  int
		maxArgs  int
		hasUsage bool
	}{
		{"look", 0, 1, true},
		{"say", 1, -1, true}, // -1 means unlimited
		{"tell", 2, -1, true},
		{"get", 1, 1, true},
		{"north", 0, 0, true},
	}
	
	for _, test := range tests {
		info, exists := parser.GetCommandInfo(test.command)
		if !exists {
			t.Errorf("Expected command '%s' to exist", test.command)
			continue
		}
		
		if info.MinArgs != test.minArgs {
			t.Errorf("Expected '%s' to have min args %d, got %d", 
				test.command, test.minArgs, info.MinArgs)
		}
		
		if info.MaxArgs != test.maxArgs {
			t.Errorf("Expected '%s' to have max args %d, got %d", 
				test.command, test.maxArgs, info.MaxArgs)
		}
		
		if test.hasUsage && info.Usage == "" {
			t.Errorf("Expected '%s' to have usage information", test.command)
		}
	}
}

func TestParseComplexCommands(t *testing.T) {
	parser := NewParser()
	
	// Test command with many arguments
	cmd := parser.Parse("tell player this is a long message", "player1", "char1")
	
	if cmd.Verb != "tell" {
		t.Errorf("Expected verb 'tell'")
	}
	
	if len(cmd.Args) != 6 { // player, this, is, a, long, message
		t.Errorf("Expected 6 args, got %d", len(cmd.Args))
	}
	
	if cmd.Args[0] != "player" {
		t.Errorf("Expected first arg 'player', got '%s'", cmd.Args[0])
	}
	
	// Test command with quoted-like content (but we don't handle quotes specially yet)
	cmd = parser.Parse("emote says \"hello world\"", "player1", "char1")
	
	if cmd.Verb != "emote" {
		t.Errorf("Expected verb 'emote'")
	}
	
	// Should split on spaces, not handle quotes
	if len(cmd.Args) != 3 { // says, "hello, world"
		t.Errorf("Expected 3 args for quoted content, got %d", len(cmd.Args))
	}
}
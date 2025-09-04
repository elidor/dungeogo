package integration

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game"
	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/server"
	"github.com/elidor/dungeogo/pkg/testutil"
)

func TestServerIntegration_BasicConnection(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	// Start test server
	gameEngine := game.NewEngine(repoManager)
	sessionHandler := server.NewSessionHandler(repoManager, gameEngine)
	connectionManager := server.NewConnectionManager(10, 5*time.Minute)
	connectionManager.SetHandler(sessionHandler)

	// Use dynamic port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().String()

	// Start server in goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Server shutting down
			}
			client := server.NewClient("test-client", conn)
			go sessionHandler.HandleClient(client)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test connection
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Should receive welcome message
	welcome, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read welcome message: %v", err)
	}

	if !strings.Contains(welcome, "Welcome to DungeoGo") {
		t.Errorf("Expected welcome message, got: %s", welcome)
	}

	// Should receive username prompt
	prompt, err := reader.ReadString('>')
	if err != nil {
		t.Fatalf("Failed to read username prompt: %v", err)
	}

	if !strings.Contains(prompt, "username") {
		t.Errorf("Expected username prompt, got: %s", prompt)
	}
}

func TestServerIntegration_ConnectionManager(t *testing.T) {
	connectionManager := server.NewConnectionManager(5, time.Minute)

	// Test initial state
	stats := connectionManager.GetStats()
	if stats.TotalClients != 0 {
		t.Errorf("Expected 0 initial clients, got %d", stats.TotalClients)
	}

	// Create mock connections
	server1, client1 := net.Pipe()
	defer server1.Close()
	defer client1.Close()

	// Add client
	testClient := server.NewClient("test-1", server1)

	// Test client properties
	if testClient.GetID() != "test-1" {
		t.Errorf("Expected client ID 'test-1', got %s", testClient.GetID())
	}

	if !testClient.IsConnected() {
		t.Errorf("Expected client to be connected")
	}

	if testClient.GetState() != server.StateConnected {
		t.Errorf("Expected client state Connected, got %d", testClient.GetState())
	}

	// Test client state transitions
	testClient.SetState(server.StateAuthenticating)
	if testClient.GetState() != server.StateAuthenticating {
		t.Errorf("Expected client state Authenticating")
	}

	testClient.SetPlayerID("player-123")
	if testClient.GetPlayerID() != "player-123" {
		t.Errorf("Expected player ID 'player-123', got %s", testClient.GetPlayerID())
	}

	testClient.SetCharacterID("char-456")
	if testClient.GetCharacterID() != "char-456" {
		t.Errorf("Expected character ID 'char-456', got %s", testClient.GetCharacterID())
	}
}

func TestServerIntegration_GameEngineCommands(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	// Create test player and character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	testChar := createTestCharacter(testPlayer.ID)
	err = repoManager.Characters().CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create test character: %v", err)
	}

	// Create game engine
	gameEngine := game.NewEngine(repoManager)

	// Test basic commands
	testCommands := []struct {
		input    string
		expected string
	}{
		{"look", "Simple Room"},
		{"score", testChar.Name},
		{"help", "Available command categories"},
		{"who", "Players currently online"},
		{"inventory", "You are carrying"},
		{"say hello", "You say: hello"},
	}

	for _, test := range testCommands {
		responses, err := gameEngine.ProcessCommand(testChar.ID, test.input)
		if err != nil {
			t.Errorf("Command '%s' failed: %v", test.input, err)
			continue
		}

		if len(responses) == 0 {
			t.Errorf("Command '%s' returned no responses", test.input)
			continue
		}

		found := false
		for _, response := range responses {
			if strings.Contains(response, test.expected) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Command '%s' expected to contain '%s', got: %v", 
				test.input, test.expected, responses)
		}
	}
}

func TestServerIntegration_CharacterManagement(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	// Test character creation and retrieval workflow
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	// Test getting empty character list
	characters, err := repoManager.Characters().GetCharactersByPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to get empty character list: %v", err)
	}

	if len(characters) != 0 {
		t.Errorf("Expected 0 characters initially, got %d", len(characters))
	}

	// Create characters
	char1 := createTestCharacter(testPlayer.ID)
	char1.Name = "TestWarrior"
	char1.Level = 5

	char2 := createTestCharacter(testPlayer.ID)
	char2.Name = "TestMage"
	char2.Level = 3

	err = repoManager.Characters().CreateCharacter(char1)
	if err != nil {
		t.Fatalf("Failed to create character 1: %v", err)
	}

	err = repoManager.Characters().CreateCharacter(char2)
	if err != nil {
		t.Fatalf("Failed to create character 2: %v", err)
	}

	// Test getting character list
	characters, err = repoManager.Characters().GetCharactersByPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to get character list: %v", err)
	}

	if len(characters) != 2 {
		t.Errorf("Expected 2 characters, got %d", len(characters))
	}

	// Verify character data integrity
	for _, char := range characters {
		if char.Name == "TestWarrior" {
			if char.Level != 5 {
				t.Errorf("Expected TestWarrior level 5, got %d", char.Level)
			}
			if char.Race != "Human" {
				t.Errorf("Expected TestWarrior race Human, got %s", char.Race)
			}
		} else if char.Name == "TestMage" {
			if char.Level != 3 {
				t.Errorf("Expected TestMage level 3, got %d", char.Level)
			}
		}
	}
}

func TestServerIntegration_ItemManagement(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	// Create test character
	testPlayer := createTestPlayer()
	err := repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	testChar := createTestCharacter(testPlayer.ID)
	err = repoManager.Characters().CreateCharacter(testChar)
	if err != nil {
		t.Fatalf("Failed to create test character: %v", err)
	}

	// Create test items
	item1 := createTestItemInstance("sword", testChar.ID)
	item2 := createTestItemInstance("potion", testChar.ID)
	item2.Quantity = 10

	err = repoManager.Items().CreateItemInstance(item1)
	if err != nil {
		t.Fatalf("Failed to create item 1: %v", err)
	}

	err = repoManager.Items().CreateItemInstance(item2)
	if err != nil {
		t.Fatalf("Failed to create item 2: %v", err)
	}

	// Test item retrieval
	items, err := repoManager.Items().GetPlayerItems(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to get player items: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	// Test item transfer
	roomID := "test_room_123"
	err = repoManager.Items().TransferItem(item1.ID, roomID)
	if err != nil {
		t.Fatalf("Failed to transfer item: %v", err)
	}

	// Verify transfer
	playerItems, err := repoManager.Items().GetPlayerItems(testChar.ID)
	if err != nil {
		t.Fatalf("Failed to get player items after transfer: %v", err)
	}

	if len(playerItems) != 1 {
		t.Errorf("Expected 1 item after transfer, got %d", len(playerItems))
	}

	roomItems, err := repoManager.Items().GetRoomItems(roomID)
	if err != nil {
		t.Fatalf("Failed to get room items: %v", err)
	}

	if len(roomItems) != 1 {
		t.Errorf("Expected 1 room item after transfer, got %d", len(roomItems))
	}
}

func TestServerIntegration_ConcurrentClients(t *testing.T) {
	// Test multiple concurrent connections
	clients := make([]*server.Client, 5)
	serverConns := make([]net.Conn, 5)
	clientConns := make([]net.Conn, 5)

	// Create multiple client connections
	for i := 0; i < 5; i++ {
		serverConn, clientConn := net.Pipe()
		serverConns[i] = serverConn
		clientConns[i] = clientConn

		clientID := generateTestID(i)
		clients[i] = server.NewClient(clientID, serverConn)

		if !clients[i].IsConnected() {
			t.Errorf("Expected client %d to be connected", i)
		}
	}

	// Clean up
	for i := 0; i < 5; i++ {
		clients[i].Close()
		serverConns[i].Close()
		clientConns[i].Close()
	}

	// Verify all clients are disconnected
	for i := 0; i < 5; i++ {
		if clients[i].IsConnected() {
			t.Errorf("Expected client %d to be disconnected after close", i)
		}
	}
}

// Helper functions

func createTestPlayer() *player.Player {
	player := testutil.CreateTestPlayer()
	player.ID = "test-player-" + generateTestUUID()
	return player
}

func createTestCharacter(playerID string) *character.Character {
	char := testutil.CreateTestCharacter(playerID)
	char.ID = "test-char-" + generateTestUUID()
	return char
}

func createTestItemInstance(templateID, ownerID string) *items.ItemInstance {
	item := testutil.CreateTestItemInstance(templateID, ownerID)
	item.ID = "test-item-" + generateTestUUID()
	return item
}

var testIDCounter int

func generateTestID(i int) string {
	testIDCounter++
	return fmt.Sprintf("test-client-%d-%d", i, testIDCounter)
}

func generateTestUUID() string {
	testIDCounter++
	return fmt.Sprintf("uuid-%d-%d", time.Now().UnixNano(), testIDCounter)
}
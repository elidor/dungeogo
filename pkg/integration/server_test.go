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
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
	"github.com/elidor/dungeogo/pkg/server"
	"github.com/elidor/dungeogo/pkg/testutil"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

func TestServerIntegration_GuidedCharacterCreation(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	gameEngine := game.NewEngine(repoManager)
	sessionHandler := server.NewSessionHandler(repoManager, gameEngine)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			client := server.NewClient("guided-create-test-client", conn)
			go sessionHandler.HandleClient(client)
		}
	}()

	password := "testpass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash test password: %v", err)
	}

	testPlayer := player.NewPlayer("guidedtester", "guidedtester@example.com", string(hash))
	err = repoManager.Players().CreatePlayer(testPlayer)
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	var transcript strings.Builder

	out, err := readUntilContains(conn, reader, "Please enter your username:")
	if err != nil {
		t.Fatalf("Failed waiting for username prompt: %v", err)
	}
	transcript.WriteString(out)

	if _, err := fmt.Fprintf(conn, "%s\n", testPlayer.Username); err != nil {
		t.Fatalf("Failed to send username: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Please enter your password:")
	if err != nil {
		t.Fatalf("Failed waiting for password prompt: %v", err)
	}
	transcript.WriteString(out)

	if _, err := fmt.Fprintf(conn, "%s\n", password); err != nil {
		t.Fatalf("Failed to send password: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Character> ")
	if err != nil {
		t.Fatalf("Failed waiting for character menu prompt: %v", err)
	}
	transcript.WriteString(out)

	if _, err := fmt.Fprintf(conn, "create\n"); err != nil {
		t.Fatalf("Failed to send create command: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Name: ")
	if err != nil {
		t.Fatalf("Failed waiting for name prompt: %v", err)
	}
	transcript.WriteString(out)

	newCharName := "GuidedHero123"
	if _, err := fmt.Fprintf(conn, "%s\n", newCharName); err != nil {
		t.Fatalf("Failed to send character name: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Race: ")
	if err != nil {
		t.Fatalf("Failed waiting for race prompt: %v", err)
	}
	transcript.WriteString(out)

	if _, err := fmt.Fprintf(conn, "elf\n"); err != nil {
		t.Fatalf("Failed to send race: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Class: ")
	if err != nil {
		t.Fatalf("Failed waiting for class prompt: %v", err)
	}
	transcript.WriteString(out)

	if _, err := fmt.Fprintf(conn, "mage\n"); err != nil {
		t.Fatalf("Failed to send class: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Confirm: ")
	if err != nil {
		t.Fatalf("Failed waiting for confirmation prompt: %v", err)
	}
	transcript.WriteString(out)

	if _, err := fmt.Fprintf(conn, "yes\n"); err != nil {
		t.Fatalf("Failed to send confirmation: %v", err)
	}

	out, err = readUntilContains(conn, reader, "Character> ")
	if err != nil {
		t.Fatalf("Failed waiting for character menu after creation: %v", err)
	}
	transcript.WriteString(out)

	var characters []*interfaces.CharacterSummary
	for i := 0; i < 10; i++ {
		characters, err = repoManager.Characters().GetCharactersByPlayer(testPlayer.ID)
		if err != nil {
			t.Fatalf("Failed to fetch characters after guided creation: %v", err)
		}
		if len(characters) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	found := false
	for _, c := range characters {
		if c.Name == newCharName {
			found = true
			if c.Race != "Elf" {
				t.Fatalf("Expected created character race Elf, got %s", c.Race)
			}
			if c.Class != "Mage" {
				t.Fatalf("Expected created character class Mage, got %s", c.Class)
			}
		}
	}

	if !found {
		var names []string
		for _, c := range characters {
			names = append(names, c.Name)
		}
		t.Fatalf("Expected guided-created character %q to exist; got names=%v; transcript=%q",
			newCharName, names, transcript.String())
	}
}

func TestServerIntegration_GuidedSelectAndDeleteByNumber(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	gameEngine := game.NewEngine(repoManager)
	sessionHandler := server.NewSessionHandler(repoManager, gameEngine)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			client := server.NewClient("guided-select-delete-test-client", conn)
			go sessionHandler.HandleClient(client)
		}
	}()

	password := "testpass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash test password: %v", err)
	}

	testPlayer := player.NewPlayer("numguidetester", "numguidetester@example.com", string(hash))
	if err := repoManager.Players().CreatePlayer(testPlayer); err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	alpha := createTestCharacter(testPlayer.ID)
	alpha.Name = "Alpha"
	if err := repoManager.Characters().CreateCharacter(alpha); err != nil {
		t.Fatalf("Failed to create Alpha character: %v", err)
	}

	zulu := createTestCharacter(testPlayer.ID)
	zulu.Name = "Zulu"
	if err := repoManager.Characters().CreateCharacter(zulu); err != nil {
		t.Fatalf("Failed to create Zulu character: %v", err)
	}

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	if _, err := readUntilContains(conn, reader, "Please enter your username:"); err != nil {
		t.Fatalf("Failed waiting for username prompt: %v", err)
	}
	fmt.Fprintf(conn, "%s\n", testPlayer.Username)
	if _, err := readUntilContains(conn, reader, "Please enter your password:"); err != nil {
		t.Fatalf("Failed waiting for password prompt: %v", err)
	}
	fmt.Fprintf(conn, "%s\n", password)
	if _, err := readUntilContains(conn, reader, "Character> "); err != nil {
		t.Fatalf("Failed waiting for character menu prompt: %v", err)
	}

	fmt.Fprintf(conn, "select\n")
	selectPrompt, err := readUntilContains(conn, reader, "Select #: ")
	if err != nil {
		t.Fatalf("Failed waiting for select prompt: %v", err)
	}
	if !strings.Contains(selectPrompt, "Alpha") || !strings.Contains(selectPrompt, "Zulu") {
		t.Fatalf("Expected numbered select list with Alpha and Zulu, got: %q", selectPrompt)
	}

	// Alphabetical ordering means Alpha is #1 and should be selected.
	fmt.Fprintf(conn, "1\n")
	selectedOut, err := readUntilContains(conn, reader, "You enter the game world...")
	if err != nil {
		t.Fatalf("Failed waiting for game entry after select: %v", err)
	}
	if !strings.Contains(selectedOut, "Welcome, Alpha!") {
		t.Fatalf("Expected to select Alpha by #1, got: %q", selectedOut)
	}

	// New session for deletion flow.
	conn2, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect second session: %v", err)
	}
	defer conn2.Close()
	reader2 := bufio.NewReader(conn2)

	if _, err := readUntilContains(conn2, reader2, "Please enter your username:"); err != nil {
		t.Fatalf("Failed waiting for username prompt (delete flow): %v", err)
	}
	fmt.Fprintf(conn2, "%s\n", testPlayer.Username)
	if _, err := readUntilContains(conn2, reader2, "Please enter your password:"); err != nil {
		t.Fatalf("Failed waiting for password prompt (delete flow): %v", err)
	}
	fmt.Fprintf(conn2, "%s\n", password)
	if _, err := readUntilContains(conn2, reader2, "Character> "); err != nil {
		t.Fatalf("Failed waiting for character menu prompt (delete flow): %v", err)
	}

	fmt.Fprintf(conn2, "delete\n")
	deletePrompt, err := readUntilContains(conn2, reader2, "Delete #: ")
	if err != nil {
		t.Fatalf("Failed waiting for delete prompt: %v", err)
	}
	if !strings.Contains(deletePrompt, "Alpha") || !strings.Contains(deletePrompt, "Zulu") {
		t.Fatalf("Expected numbered delete list with Alpha and Zulu, got: %q", deletePrompt)
	}

	// Zulu should be #2 alphabetically.
	fmt.Fprintf(conn2, "2\n")
	if _, err := readUntilContains(conn2, reader2, "Confirm Name: "); err != nil {
		t.Fatalf("Failed waiting for confirmation-name prompt: %v", err)
	}

	charsBeforeConfirm, err := repoManager.Characters().GetCharactersByPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to fetch characters before confirmation: %v", err)
	}
	if len(charsBeforeConfirm) != 2 {
		t.Fatalf("Expected 2 characters before confirmation, got %d", len(charsBeforeConfirm))
	}

	fmt.Fprintf(conn2, "Zulu\n")
	if _, err := readUntilContains(conn2, reader2, "Character 'Zulu' deleted."); err != nil {
		t.Fatalf("Failed waiting for deletion confirmation: %v", err)
	}

	charsAfterConfirm, err := repoManager.Characters().GetCharactersByPlayer(testPlayer.ID)
	if err != nil {
		t.Fatalf("Failed to fetch characters after confirmation: %v", err)
	}
	if len(charsAfterConfirm) != 1 {
		t.Fatalf("Expected 1 character after deletion, got %d", len(charsAfterConfirm))
	}
	if charsAfterConfirm[0].Name != "Alpha" {
		t.Fatalf("Expected remaining character to be Alpha, got %s", charsAfterConfirm[0].Name)
	}
}

func TestServerIntegration_InGameQuitDisconnects(t *testing.T) {
	repoManager := testutil.ImprovedSetupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for integration testing")
	}

	gameEngine := game.NewEngine(repoManager)
	sessionHandler := server.NewSessionHandler(repoManager, gameEngine)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			client := server.NewClient("quit-disconnect-test-client", conn)
			go sessionHandler.HandleClient(client)
		}
	}()

	password := "testpass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash test password: %v", err)
	}

	testPlayer := player.NewPlayer("quittester", "quittester@example.com", string(hash))
	if err := repoManager.Players().CreatePlayer(testPlayer); err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}

	alpha := createTestCharacter(testPlayer.ID)
	alpha.Name = "Alpha"
	if err := repoManager.Characters().CreateCharacter(alpha); err != nil {
		t.Fatalf("Failed to create Alpha character: %v", err)
	}

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	if _, err := readUntilContains(conn, reader, "Please enter your username:"); err != nil {
		t.Fatalf("Failed waiting for username prompt: %v", err)
	}
	fmt.Fprintf(conn, "%s\n", testPlayer.Username)
	if _, err := readUntilContains(conn, reader, "Please enter your password:"); err != nil {
		t.Fatalf("Failed waiting for password prompt: %v", err)
	}
	fmt.Fprintf(conn, "%s\n", password)
	if _, err := readUntilContains(conn, reader, "Character> "); err != nil {
		t.Fatalf("Failed waiting for character menu prompt: %v", err)
	}

	fmt.Fprintf(conn, "select\n")
	if _, err := readUntilContains(conn, reader, "Select #: "); err != nil {
		t.Fatalf("Failed waiting for select prompt: %v", err)
	}
	fmt.Fprintf(conn, "1\n")
	if _, err := readUntilContains(conn, reader, "> "); err != nil {
		t.Fatalf("Failed waiting for game entry: %v", err)
	}

	fmt.Fprintf(conn, "quit\n")
	if _, err := readUntilContains(conn, reader, "Saving character and disconnecting..."); err != nil {
		t.Fatalf("Failed waiting for quit message: %v", err)
	}

	// After quit response, no new prompt should appear and the connection should close.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			t.Fatalf("Failed setting read deadline: %v", err)
		}
		b, err := reader.ReadByte()
		if err != nil {
			return // EOF/timeout after close is acceptable.
		}
		if b == '>' {
			t.Fatalf("Expected no prompt after quit, but received one")
		}
	}

	t.Fatalf("Expected connection to close after quit")
}

// Helper functions

func createTestPlayer() *player.Player {
	player := testutil.CreateTestPlayer()
	player.ID = generateTestUUID()
	return player
}

func createTestCharacter(playerID string) *character.Character {
	char := testutil.CreateTestCharacter(playerID)
	char.ID = generateTestUUID()
	return char
}

func createTestItemInstance(templateID, ownerID string) *items.ItemInstance {
	item := testutil.CreateTestItemInstance(templateID, ownerID)
	item.ID = generateTestUUID()
	return item
}

var testIDCounter int

func generateTestID(i int) string {
	testIDCounter++
	return fmt.Sprintf("test-client-%d-%d", i, testIDCounter)
}

func generateTestUUID() string {
	return uuid.New().String()
}

func readUntilContains(conn net.Conn, reader *bufio.Reader, expected string) (string, error) {
	var b strings.Builder
	for i := 0; i < 8192; i++ {
		if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			return b.String(), err
		}

		ch, err := reader.ReadByte()
		if err != nil {
			return b.String(), err
		}
		b.WriteByte(ch)

		if strings.Contains(b.String(), expected) {
			return b.String(), nil
		}
	}

	return b.String(), fmt.Errorf("did not receive expected text %q", expected)
}

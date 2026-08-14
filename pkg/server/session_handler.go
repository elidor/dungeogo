package server

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
	"golang.org/x/crypto/bcrypt"
)

type SessionHandler struct {
	repoManager interfaces.RepositoryManager
	gameEngine  GameEngine
}

type GameEngine interface {
	ProcessCommand(characterID string, command string) ([]string, error)
	GetCharacterState(characterID string) (interface{}, error)
}

func NewSessionHandler(repoManager interfaces.RepositoryManager, gameEngine GameEngine) *SessionHandler {
	return &SessionHandler{
		repoManager: repoManager,
		gameEngine:  gameEngine,
	}
}

func (sh *SessionHandler) HandleClient(client *Client) {
	defer func() {
		fmt.Printf("Client disconnected: id=%s playerID=%s characterID=%s state=%d\n",
			client.GetID(), client.GetPlayerID(), client.GetCharacterID(), client.GetState())
		client.Close()
	}()

	// Welcome message
	client.Send("Welcome to DungeoGo!")
	client.Send("Please enter your username:")
	client.SendPrompt("> ")

	for client.IsConnected() {
		var line string
		var err error

		// Use password reading for sensitive input
		if client.GetState() == StateAuthenticating || client.GetState() == StateConfirmingPassword {
			line, err = client.ReadPassword()
		} else {
			line, err = client.ReadLine()
		}

		if err != nil {
			fmt.Printf("Error reading from client %s: %v\n", client.GetID(), err)
			break
		}

		switch client.GetState() {
		case StateConnected:
			sh.handleLogin(client, line)
		case StateAuthenticating:
			sh.handlePasswordAuth(client, line)
		case StateCreatingAccount:
			sh.handleAccountCreation(client, line)
		case StateConfirmingPassword:
			sh.handlePasswordConfirmation(client, line)
		case StateCharacterSelection:
			sh.handleCharacterSelection(client, line)
		case StateCharacterCreation:
			sh.handleCharacterCreation(client, line)
		case StateCharacterSelecting:
			sh.handleCharacterSelectByNumber(client, line)
		case StateCharacterDeleting:
			sh.handleCharacterDeleteByNumber(client, line)
		case StateInGame:
			sh.handleGameCommand(client, line)
		}
	}
}

func (sh *SessionHandler) handleLogin(client *Client, username string) {
	username = strings.TrimSpace(username)
	if username == "" {
		client.Send("Username cannot be empty. Please enter your username:")
		client.SendPrompt("> ")
		return
	}

	fmt.Printf("Login attempt for client %s: username='%s'\n", client.GetID(), username)

	// Check if player exists
	existingPlayer, err := sh.repoManager.Players().GetPlayerByUsername(username)
	if err != nil {
		fmt.Printf("Player lookup failed for client %s, username='%s': %v\n", client.GetID(), username, err)
		// New player - create account
		client.SetTempUsername(username)
		client.Send("New player! Creating account for: " + username)
		client.Send("Please enter your email address:")
		client.SendPrompt("Email: ")
		client.SetState(StateCreatingAccount)
		return
	}

	fmt.Printf("Found existing player for client %s: username='%s', ID='%s'\n",
		client.GetID(), username, existingPlayer.ID)

	if !existingPlayer.IsActive() {
		client.Send("Your account has been suspended. Please contact an administrator.")
		client.Close()
		return
	}

	client.Send("Please enter your password:")
	client.SetState(StateAuthenticating)
	// Store player ID temporarily
	client.SetPlayerID(existingPlayer.ID)
}

func (sh *SessionHandler) handlePasswordAuth(client *Client, password string) {
	password = strings.TrimSpace(password)
	if password == "" {
		client.Send("Password cannot be empty. Please enter your password:")
		client.SendPrompt("Password: ")
		return
	}

	playerID := client.GetPlayerID()
	if playerID == "" {
		// New player creation - simplified for demo
		client.Send("Account creation not fully implemented yet.")
		client.Close()
		return
	}

	// Get player and verify password (simplified - use proper password hashing)
	existingPlayer, err := sh.repoManager.Players().GetPlayer(playerID)
	if err != nil {
		client.Send("Authentication failed.")
		client.Close()
		return
	}

	// Verify password using bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(existingPlayer.PasswordHash), []byte(password))
	if err != nil {
		client.Send("Invalid password.")
		client.Close()
		return
	}

	// Authentication successful
	existingPlayer.UpdateLastLogin()
	sh.repoManager.Players().UpdatePlayerLogin(playerID)

	client.Send(fmt.Sprintf("Welcome back, %s!", existingPlayer.Username))
	client.SetState(StateCharacterSelection)
	sh.showCharacterMenu(client)
}

func (sh *SessionHandler) handleCharacterSelection(client *Client, input string) {
	input = strings.TrimSpace(input)
	parts := strings.Fields(input)

	if len(parts) == 0 || isMenuHelpInput(input) {
		sh.showCharacterMenu(client)
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "list", "l":
		sh.listCharacters(client)
	case "select", "s":
		if len(parts) != 1 {
			client.Send("Usage: select")
			client.Send("This command starts guided character selection by number.")
		} else {
			sh.startCharacterSelectByNumber(client)
		}
	case "create", "c":
		if len(parts) != 1 {
			client.Send("Usage: create")
			client.Send("This command starts guided character creation.")
		} else {
			sh.startCharacterCreation(client)
		}
	case "delete", "d":
		if len(parts) != 1 {
			client.Send("Usage: delete")
			client.Send("This command starts guided character deletion by number.")
		} else {
			sh.startCharacterDeleteByNumber(client)
		}
	case "quit", "q":
		client.Send("Goodbye!")
		client.Close()
	default:
		client.Send("Unknown command. Type 'list' to see your characters.")
	}

	if client.GetState() == StateCharacterSelection {
		client.SendPrompt("Character> ")
	}
}

func (sh *SessionHandler) handleGameCommand(client *Client, input string) {
	characterID := client.GetCharacterID()
	if characterID == "" {
		client.Send("Error: No character selected.")
		client.SetState(StateCharacterSelection)
		sh.showCharacterMenu(client)
		return
	}

	command := strings.ToLower(strings.TrimSpace(input))
	if command == "quit" || command == "q" {
		sh.handleInGameQuit(client, characterID)
		return
	}

	// Process command through game engine
	responses, err := sh.gameEngine.ProcessCommand(characterID, input)
	if err != nil {
		client.Send(fmt.Sprintf("Error: %v", err))
	} else {
		for _, response := range responses {
			client.Send(response)
		}
	}

	client.SendPrompt("> ")
}

func (sh *SessionHandler) handleInGameQuit(client *Client, characterID string) {
	// Save the character before disconnecting.
	char, err := sh.repoManager.Characters().GetCharacter(characterID)
	if err == nil {
		char.UpdatePlayTime()
		_ = sh.repoManager.Characters().UpdateCharacter(char)
	}

	client.Send("Saving character and disconnecting...")
	client.Close()
}

func (sh *SessionHandler) showCharacterMenu(client *Client) {
	client.Send("\n--- Character Selection ---")
	client.Send("Commands:")
	client.Send("  list (l)                 - List your characters")
	client.Send("  select (s)               - Select character by number")
	client.Send("  create (c)               - Guided character creation")
	client.Send("  delete (d)               - Delete character by number")
	client.Send("  quit (q)                 - Disconnect")
	client.Send("")
	client.SendPrompt("Character> ")
}

func (sh *SessionHandler) listCharacters(client *Client) {
	characters, err := sh.repoManager.Characters().GetCharactersByPlayer(client.GetPlayerID())
	if err != nil {
		client.Send("Error retrieving characters.")
		return
	}

	if len(characters) == 0 {
		client.Send("You have no characters. Use 'create' to start guided character creation.")
		return
	}

	client.Send("\nYour Characters:")
	client.Send("Name           Race      Class     Level  Status    Last Played")
	client.Send("--------------------------------------------------------------")
	for _, char := range sortCharacterSummariesByName(characters) {
		status := "Alive"
		if !char.IsAlive {
			status = "Dead"
		}

		lastPlayed := formatLastPlayed(char.LastPlayed)
		client.Send(fmt.Sprintf("%-14s %-9s %-9s %-6d %-9s %s",
			char.Name, char.Race, char.Class, char.Level, status, lastPlayed))
	}
	client.Send("")
}

func formatLastPlayed(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0001-01-01T00:00:00Z" {
		return "Never"
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Local().Format("2006-01-02 15:04")
		}
	}

	return raw
}

func isMenuHelpInput(input string) bool {
	v := strings.TrimSpace(strings.ToLower(input))
	return v == "?" || v == "help"
}

func sortCharacterSummariesByName(characters []*interfaces.CharacterSummary) []*interfaces.CharacterSummary {
	sorted := make([]*interfaces.CharacterSummary, len(characters))
	copy(sorted, characters)

	sort.Slice(sorted, func(i, j int) bool {
		a := strings.ToLower(sorted[i].Name)
		b := strings.ToLower(sorted[j].Name)
		if a == b {
			return sorted[i].Name < sorted[j].Name
		}
		return a < b
	})

	return sorted
}

func (sh *SessionHandler) getSortedCharactersForPlayer(client *Client) ([]*interfaces.CharacterSummary, error) {
	characters, err := sh.repoManager.Characters().GetCharactersByPlayer(client.GetPlayerID())
	if err != nil {
		return nil, err
	}
	return sortCharacterSummariesByName(characters), nil
}

func (sh *SessionHandler) showNumberedCharacterList(client *Client, characters []*interfaces.CharacterSummary) {
	client.Send("No. Name           Race      Class     Level  Status    Last Played")
	client.Send("--------------------------------------------------------------------")
	for i, char := range characters {
		status := "Alive"
		if !char.IsAlive {
			status = "Dead"
		}

		lastPlayed := formatLastPlayed(char.LastPlayed)
		client.Send(fmt.Sprintf("%-3d %-14s %-9s %-9s %-6d %-9s %s",
			i+1, char.Name, char.Race, char.Class, char.Level, status, lastPlayed))
	}
}

func (sh *SessionHandler) startCharacterSelectByNumber(client *Client) {
	characters, err := sh.getSortedCharactersForPlayer(client)
	if err != nil {
		client.Send("Error retrieving characters.")
		return
	}
	if len(characters) == 0 {
		client.Send("You have no characters to select. Use 'create' first.")
		return
	}

	client.SetState(StateCharacterSelecting)
	client.ClearTempMenuData()
	client.Send("\n--- Character Select ---")
	client.Send("Type a character number, or 'cancel' to return.")
	sh.showNumberedCharacterList(client, characters)
	client.SendPrompt("Select #: ")
}

func (sh *SessionHandler) handleCharacterSelectByNumber(client *Client, input string) {
	input = strings.TrimSpace(input)
	if isMenuHelpInput(input) {
		characters, err := sh.getSortedCharactersForPlayer(client)
		if err != nil {
			client.Send("Error retrieving characters.")
			client.SetState(StateCharacterSelection)
			sh.showCharacterMenu(client)
			return
		}
		if len(characters) == 0 {
			client.Send("You have no characters to select. Use 'create' first.")
			client.SetState(StateCharacterSelection)
			sh.showCharacterMenu(client)
			return
		}
		client.Send("\n--- Character Select ---")
		client.Send("Type a character number, or 'cancel' to return.")
		sh.showNumberedCharacterList(client, characters)
		client.SendPrompt("Select #: ")
		return
	}

	if strings.EqualFold(input, "cancel") {
		client.SetState(StateCharacterSelection)
		client.ClearTempMenuData()
		client.Send("Selection canceled.")
		sh.showCharacterMenu(client)
		return
	}

	index, err := strconv.Atoi(input)
	if err != nil {
		client.Send("Please enter a valid number.")
		client.SendPrompt("Select #: ")
		return
	}

	characters, err := sh.getSortedCharactersForPlayer(client)
	if err != nil {
		client.Send("Error retrieving characters.")
		client.SetState(StateCharacterSelection)
		sh.showCharacterMenu(client)
		return
	}
	if index < 1 || index > len(characters) {
		client.Send(fmt.Sprintf("Please enter a number between 1 and %d.", len(characters)))
		client.SendPrompt("Select #: ")
		return
	}

	selected := characters[index-1]
	client.ClearTempMenuData()
	client.SetCharacterID(selected.ID)
	client.SetState(StateInGame)
	client.Send(fmt.Sprintf("Welcome, %s!", selected.Name))
	client.Send("You enter the game world...")

	// Show the character's surroundings immediately, just as if they had run
	// the look command after entering the game.
	responses, err := sh.gameEngine.ProcessCommand(selected.ID, "look")
	if err != nil {
		client.Send(fmt.Sprintf("Error displaying current room: %v", err))
	} else {
		for _, response := range responses {
			client.Send(response)
		}
	}

	client.SendPrompt("> ")
}

func (sh *SessionHandler) createCharacter(client *Client, name, raceStr, classStr string) {
	name = strings.TrimSpace(name)
	if name == "" {
		client.Send("Character name cannot be empty.")
		return
	}

	// Validate race
	race, err := character.GetRaceByID(strings.ToLower(raceStr))
	if err != nil {
		client.Send(fmt.Sprintf("Invalid race: %s", raceStr))
		client.Send("Available races: human, elf, dwarf")
		return
	}

	// Validate class
	class, err := character.GetClassByID(strings.ToLower(classStr))
	if err != nil {
		client.Send(fmt.Sprintf("Invalid class: %s", classStr))
		client.Send("Available classes: warrior, mage, rogue")
		return
	}

	// Create character
	newChar := character.NewCharacter(client.GetPlayerID(), name, race, class)
	err = sh.repoManager.Characters().CreateCharacter(newChar)
	if err != nil {
		client.Send("Error creating character. Name might already be taken.")
		return
	}

	client.Send(fmt.Sprintf("Character '%s' created successfully!", name))
}

func (sh *SessionHandler) startCharacterCreation(client *Client) {
	client.ClearTempCharacterData()
	client.SetTempCreationStep(0)
	client.SetState(StateCharacterCreation)
	client.Send("\n--- Character Creation ---")
	client.Send("Type 'cancel' at any time to return to the character menu.")
	client.Send("Enter character name (3-20 letters/numbers):")
	client.SendPrompt("Name: ")
}

func (sh *SessionHandler) handleCharacterCreation(client *Client, input string) {
	input = strings.TrimSpace(input)
	if isMenuHelpInput(input) {
		switch client.GetTempCreationStep() {
		case 0:
			client.Send("Enter character name (3-20 letters/numbers):")
			client.SendPrompt("Name: ")
		case 1:
			client.Send("Choose race: human, elf, dwarf")
			client.SendPrompt("Race: ")
		case 2:
			client.Send("Choose class: warrior, mage, rogue")
			client.SendPrompt("Class: ")
		case 3:
			client.Send(fmt.Sprintf("Create character '%s' as %s %s? (yes/no)",
				client.GetTempCharacterName(), client.GetTempCharacterRace(), client.GetTempCharacterClass()))
			client.SendPrompt("Confirm: ")
		}
		return
	}

	if strings.EqualFold(input, "cancel") {
		client.ClearTempCharacterData()
		client.SetState(StateCharacterSelection)
		client.Send("Character creation canceled.")
		sh.showCharacterMenu(client)
		return
	}

	switch client.GetTempCreationStep() {
	case 0:
		if !isValidCharacterName(input) {
			client.Send("Invalid name. Use 3-20 letters or numbers.")
			client.SendPrompt("Name: ")
			return
		}
		client.SetTempCharacterName(input)
		client.SetTempCreationStep(1)
		client.Send("Choose race: human, elf, dwarf")
		client.SendPrompt("Race: ")
	case 1:
		raceID := strings.ToLower(input)
		if _, err := character.GetRaceByID(raceID); err != nil {
			client.Send("Invalid race. Available: human, elf, dwarf")
			client.SendPrompt("Race: ")
			return
		}
		client.SetTempCharacterRace(raceID)
		client.SetTempCreationStep(2)
		client.Send("Choose class: warrior, mage, rogue")
		client.SendPrompt("Class: ")
	case 2:
		classID := strings.ToLower(input)
		if _, err := character.GetClassByID(classID); err != nil {
			client.Send("Invalid class. Available: warrior, mage, rogue")
			client.SendPrompt("Class: ")
			return
		}
		client.SetTempCharacterClass(classID)
		client.SetTempCreationStep(3)
		client.Send(fmt.Sprintf("Create character '%s' as %s %s? (yes/no)",
			client.GetTempCharacterName(), client.GetTempCharacterRace(), client.GetTempCharacterClass()))
		client.SendPrompt("Confirm: ")
	case 3:
		confirm := strings.ToLower(input)
		if confirm != "yes" && confirm != "y" && confirm != "no" && confirm != "n" {
			client.Send("Please answer yes or no.")
			client.SendPrompt("Confirm: ")
			return
		}
		if confirm == "no" || confirm == "n" {
			client.ClearTempCharacterData()
			client.SetState(StateCharacterSelection)
			client.Send("Character creation canceled.")
			sh.showCharacterMenu(client)
			return
		}

		name := client.GetTempCharacterName()
		race := client.GetTempCharacterRace()
		class := client.GetTempCharacterClass()

		client.ClearTempCharacterData()
		client.SetState(StateCharacterSelection)
		sh.createCharacter(client, name, race, class)
		sh.showCharacterMenu(client)
	default:
		client.ClearTempCharacterData()
		client.SetState(StateCharacterSelection)
		sh.showCharacterMenu(client)
	}
}

func isValidCharacterName(name string) bool {
	if len(name) < 3 || len(name) > 20 {
		return false
	}

	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}

	return true
}

func (sh *SessionHandler) startCharacterDeleteByNumber(client *Client) {
	characters, err := sh.getSortedCharactersForPlayer(client)
	if err != nil {
		client.Send("Error retrieving characters.")
		return
	}
	if len(characters) == 0 {
		client.Send("You have no characters to delete.")
		return
	}

	client.SetState(StateCharacterDeleting)
	client.ClearTempMenuData()
	client.SetTempMenuStep(0)
	client.Send("\n--- Character Delete ---")
	client.Send("Choose a character number to delete, or 'cancel' to return.")
	sh.showNumberedCharacterList(client, characters)
	client.SendPrompt("Delete #: ")
}

func (sh *SessionHandler) handleCharacterDeleteByNumber(client *Client, input string) {
	input = strings.TrimSpace(input)
	if isMenuHelpInput(input) {
		if client.GetTempMenuStep() == 0 {
			characters, err := sh.getSortedCharactersForPlayer(client)
			if err != nil {
				client.Send("Error retrieving characters.")
				client.ClearTempMenuData()
				client.SetState(StateCharacterSelection)
				sh.showCharacterMenu(client)
				return
			}
			client.Send("\n--- Character Delete ---")
			client.Send("Choose a character number to delete, or 'cancel' to return.")
			sh.showNumberedCharacterList(client, characters)
			client.SendPrompt("Delete #: ")
			return
		}

		client.Send(fmt.Sprintf("To confirm deletion, type the character name exactly: %s",
			client.GetTempTargetCharacterName()))
		client.SendPrompt("Confirm Name: ")
		return
	}

	if strings.EqualFold(input, "cancel") {
		client.ClearTempMenuData()
		client.SetState(StateCharacterSelection)
		client.Send("Deletion canceled.")
		sh.showCharacterMenu(client)
		return
	}

	switch client.GetTempMenuStep() {
	case 0:
		index, err := strconv.Atoi(input)
		if err != nil {
			client.Send("Please enter a valid number.")
			client.SendPrompt("Delete #: ")
			return
		}

		characters, err := sh.getSortedCharactersForPlayer(client)
		if err != nil {
			client.Send("Error retrieving characters.")
			client.ClearTempMenuData()
			client.SetState(StateCharacterSelection)
			sh.showCharacterMenu(client)
			return
		}
		if index < 1 || index > len(characters) {
			client.Send(fmt.Sprintf("Please enter a number between 1 and %d.", len(characters)))
			client.SendPrompt("Delete #: ")
			return
		}

		target := characters[index-1]
		client.SetTempTargetCharacterID(target.ID)
		client.SetTempTargetCharacterName(target.Name)
		client.SetTempMenuStep(1)
		client.Send(fmt.Sprintf("To confirm deletion, type the character name exactly: %s", target.Name))
		client.SendPrompt("Confirm Name: ")
	case 1:
		expectedName := client.GetTempTargetCharacterName()
		targetID := client.GetTempTargetCharacterID()
		if expectedName == "" || targetID == "" {
			client.Send("Deletion context lost. Please try again.")
			client.ClearTempMenuData()
			client.SetState(StateCharacterSelection)
			sh.showCharacterMenu(client)
			return
		}

		if input != expectedName {
			client.Send("Name confirmation did not match. Deletion canceled.")
			client.ClearTempMenuData()
			client.SetState(StateCharacterSelection)
			sh.showCharacterMenu(client)
			return
		}

		if err := sh.repoManager.Characters().DeleteCharacter(targetID); err != nil {
			client.Send("Failed to delete character.")
			client.ClearTempMenuData()
			client.SetState(StateCharacterSelection)
			sh.showCharacterMenu(client)
			return
		}

		if client.GetCharacterID() == targetID {
			client.SetCharacterID("")
		}

		client.Send(fmt.Sprintf("Character '%s' deleted.", expectedName))
		client.ClearTempMenuData()
		client.SetState(StateCharacterSelection)
		sh.showCharacterMenu(client)
	default:
		client.ClearTempMenuData()
		client.SetState(StateCharacterSelection)
		sh.showCharacterMenu(client)
	}
}

// handleAccountCreation handles the account creation process
func (sh *SessionHandler) handleAccountCreation(client *Client, input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		client.Send("Email cannot be empty. Please enter your email address:")
		client.SendPrompt("Email: ")
		return
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(input) {
		client.Send("Invalid email format. Please enter a valid email address:")
		client.SendPrompt("Email: ")
		return
	}

	// Check if email is already in use
	existingPlayer, err := sh.repoManager.Players().GetPlayerByEmail(input)
	if err == nil && existingPlayer != nil {
		client.Send("An account with this email already exists.")
		client.Send("Please try logging in or use a different email address.")
		client.Close()
		return
	}

	client.SetTempEmail(input)
	client.Send("Please choose a password (minimum 6 characters):")
	client.SetState(StateConfirmingPassword)
}

// handlePasswordConfirmation handles password input and confirmation
func (sh *SessionHandler) handlePasswordConfirmation(client *Client, password string) {
	password = strings.TrimSpace(password)

	fmt.Printf("Password confirmation debug - Client %s: received password length=%d\n", client.GetID(), len(password))

	if client.GetTempPassword() == "" {
		// First password entry
		fmt.Printf("First password entry for client %s\n", client.GetID())
		if len(password) < 6 {
			client.Send("Password must be at least 6 characters long.")
			client.Send("Please choose a password (minimum 6 characters):")
			return
		}

		client.SetTempPassword(password)
		fmt.Printf("Stored password for client %s, length=%d\n", client.GetID(), len(password))
		client.Send("Please confirm your password:")
		return
	}

	// Password confirmation
	storedPassword := client.GetTempPassword()
	fmt.Printf("Password confirmation for client %s: stored=%d chars, entered=%d chars, match=%v\n",
		client.GetID(), len(storedPassword), len(password), storedPassword == password)

	if storedPassword != password {
		client.Send("Passwords do not match.")
		client.SetTempPassword("") // Clear stored password
		client.Send("Please choose a password (minimum 6 characters):")
		return
	}

	// Create the account
	sh.createAccount(client)
}

// createAccount creates a new player account
func (sh *SessionHandler) createAccount(client *Client) {
	username := client.GetTempUsername()
	email := client.GetTempEmail()
	password := client.GetTempPassword()

	fmt.Printf("Creating account for client %s: username=%s, email=%s, password_len=%d\n",
		client.GetID(), username, email, len(password))

	// Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Failed to hash password for client %s: %v\n", client.GetID(), err)
		client.Send("Failed to create account due to internal error.")
		client.Close()
		return
	}
	passwordHash := string(hashedPassword)

	// Create new player
	newPlayer := player.NewPlayer(username, email, passwordHash)
	fmt.Printf("Created player object for client %s: ID=%s\n", client.GetID(), newPlayer.ID)

	err = sh.repoManager.Players().CreatePlayer(newPlayer)
	if err != nil {
		fmt.Printf("Failed to create player in database for client %s: %v\n", client.GetID(), err)
		client.Send("Failed to create account. Username might already be taken.")
		client.Close()
		return
	}

	fmt.Printf("Successfully created account for client %s: %s\n", client.GetID(), username)

	// Clear temporary data
	client.ClearTempData()

	// Set player ID and continue to character selection
	client.SetPlayerID(newPlayer.ID)
	client.Send(fmt.Sprintf("Account created successfully! Welcome to DungeoGo, %s!", username))
	client.SetState(StateCharacterSelection)
	sh.showCharacterMenu(client)
}

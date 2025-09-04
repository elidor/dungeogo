package server

import (
	"fmt"
	"strings"
	
	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
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
	defer client.Close()
	
	// Welcome message
	client.Send("Welcome to DungeoGo!")
	client.Send("Please enter your username:")
	client.SendPrompt("> ")
	
	for client.IsConnected() {
		line, err := client.ReadLine()
		if err != nil {
			fmt.Printf("Error reading from client %s: %v\n", client.GetID(), err)
			break
		}
		
		switch client.GetState() {
		case StateConnected:
			sh.handleLogin(client, line)
		case StateAuthenticating:
			sh.handlePasswordAuth(client, line)
		case StateCharacterSelection:
			sh.handleCharacterSelection(client, line)
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
	
	// Check if player exists
	existingPlayer, err := sh.repoManager.Players().GetPlayerByUsername(username)
	if err != nil {
		// New player - create account
		client.Send("New player! Please choose a password:")
		client.SendPrompt("Password: ")
		client.SetState(StateAuthenticating)
		// Store username temporarily - in real implementation, use session data
		return
	}
	
	if !existingPlayer.IsActive() {
		client.Send("Your account has been suspended. Please contact an administrator.")
		client.Close()
		return
	}
	
	client.Send("Please enter your password:")
	client.SendPrompt("Password: ")
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
	
	// In real implementation, use bcrypt or similar
	if existingPlayer.PasswordHash != password {
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
	
	if len(parts) == 0 {
		sh.showCharacterMenu(client)
		return
	}
	
	command := strings.ToLower(parts[0])
	
	switch command {
	case "list", "l":
		sh.listCharacters(client)
	case "select", "s":
		if len(parts) < 2 {
			client.Send("Usage: select <character_name>")
		} else {
			sh.selectCharacter(client, parts[1])
		}
	case "create", "c":
		if len(parts) < 4 {
			client.Send("Usage: create <name> <race> <class>")
		} else {
			sh.createCharacter(client, parts[1], parts[2], parts[3])
		}
	case "delete", "d":
		if len(parts) < 2 {
			client.Send("Usage: delete <character_name>")
		} else {
			sh.deleteCharacter(client, parts[1])
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

func (sh *SessionHandler) showCharacterMenu(client *Client) {
	client.Send("\n--- Character Selection ---")
	client.Send("Commands:")
	client.Send("  list (l)                 - List your characters")
	client.Send("  select (s) <name>        - Enter game with character")
	client.Send("  create (c) <name> <race> <class> - Create new character")
	client.Send("  delete (d) <name>        - Delete character")
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
		client.Send("You have no characters. Use 'create <name> <race> <class>' to create one.")
		return
	}
	
	client.Send("\nYour Characters:")
	client.Send("Name           Race      Class     Level  Status    Last Played")
	client.Send("--------------------------------------------------------------")
	for _, char := range characters {
		status := "Alive"
		if !char.IsAlive {
			status = "Dead"
		}
		client.Send(fmt.Sprintf("%-14s %-9s %-9s %-6d %-9s %s",
			char.Name, char.Race, char.Class, char.Level, status, char.LastPlayed))
	}
	client.Send("")
}

func (sh *SessionHandler) selectCharacter(client *Client, name string) {
	// Get characters and find by name
	characters, err := sh.repoManager.Characters().GetCharactersByPlayer(client.GetPlayerID())
	if err != nil {
		client.Send("Error retrieving characters.")
		return
	}
	
	for _, char := range characters {
		if strings.EqualFold(char.Name, name) {
			client.SetCharacterID(char.ID)
			client.SetState(StateInGame)
			client.Send(fmt.Sprintf("Welcome, %s!", char.Name))
			client.Send("You enter the game world...")
			client.SendPrompt("> ")
			return
		}
	}
	
	client.Send(fmt.Sprintf("Character '%s' not found.", name))
}

func (sh *SessionHandler) createCharacter(client *Client, name, raceStr, classStr string) {
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

func (sh *SessionHandler) deleteCharacter(client *Client, name string) {
	client.Send("Character deletion not implemented yet.")
}
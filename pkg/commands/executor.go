package commands

import (
	"fmt"
	"strings"

	"github.com/elidor/dungeogo/pkg/game/actions"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/world"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

type Executor struct {
	repoManager  interfaces.RepositoryManager
	handlers     map[string]CommandHandler
	worldActions *actions.World
	templates    *items.ItemRegistry // the game's one item registry
}

type CommandHandler interface {
	Execute(cmd *Command) ([]string, error)
}

type CommandResponse struct {
	Messages []string
	Error    error
}

func NewExecutor(repoManager interfaces.RepositoryManager) *Executor {
	e := &Executor{
		repoManager: repoManager,
		handlers:    make(map[string]CommandHandler),
	}

	e.templates = items.NewItemRegistry()
	e.worldActions = actions.NewWorld(repoManager, e.templates)
	e.initializeHandlers()
	return e
}

// SetOnlineCharacters wires the process-owned presence source before sessions start.
func (e *Executor) SetOnlineCharacters(source func() []interfaces.OnlineCharacter) {
	e.handlers["who"] = &WhoHandler{onlineCharacters: source}
}

// SetPresence configures the shared world-action boundary before sessions start.
func (e *Executor) SetPresence(presence world.Presence) { e.worldActions.SetPresence(presence) }

// ExecuteAction returns addressed delivery effects; Execute preserves actor-only callers.
func (e *Executor) ExecuteAction(cmd *Command) (world.Result, error) {
	// give takes exactly an item and a character; anything else gets its usage
	// line rather than the generic syntax error.
	if cmd.Verb == "give" {
		if len(cmd.Args) != 2 {
			return world.Reply(cmd.CharacterID, actions.GiveUsage), nil
		}
		return e.worldActions.Give(cmd.CharacterID, cmd.Args[0], cmd.Args[1]), nil
	}
	if cmd.Type != CommandUnknown && cmd.ValidateArgs() {
		if cmd.Type == CommandMovement {
			return e.worldActions.Move(cmd.CharacterID, cmd.Verb), nil
		}
		if cmd.Verb == "tell" {
			if len(cmd.Args) < 2 {
				return world.Reply(cmd.CharacterID, "Usage: tell <character> <message>"), nil
			}
			return e.worldActions.Tell(cmd.CharacterID, cmd.Args[0], strings.Join(cmd.Args[1:], " ")), nil
		}
		if cmd.Verb == "say" {
			return e.worldActions.Say(cmd.CharacterID, strings.Join(cmd.Args, " ")), nil
		}
		if cmd.Verb == "emote" {
			return e.worldActions.Emote(cmd.CharacterID, strings.Join(cmd.Args, " ")), nil
		}
		if cmd.Verb == "look" && len(cmd.Args) == 0 {
			return e.worldActions.Look(cmd.CharacterID), nil
		}
		if cmd.Verb == "get" {
			return e.worldActions.Get(cmd.CharacterID, strings.Join(cmd.Args, " ")), nil
		}
		if cmd.Verb == "drop" {
			return e.worldActions.Drop(cmd.CharacterID, strings.Join(cmd.Args, " ")), nil
		}
	}
	lines, err := e.executeText(cmd)
	return world.Reply(cmd.CharacterID, lines...), err
}

func (e *Executor) Execute(cmd *Command) ([]string, error) {
	result, err := e.ExecuteAction(cmd)
	return result.Replies(cmd.CharacterID), err
}

func (e *Executor) executeText(cmd *Command) ([]string, error) {
	if notYetAvailable[cmd.Verb] {
		return []string{fmt.Sprintf("'%s' isn't available yet. Type 'commands' to see what you can do.", cmd.Verb)}, nil
	}
	if cmd.Type == CommandUnknown {
		return []string{fmt.Sprintf("Unknown command: %s", cmd.Verb)}, nil
	}

	if !cmd.ValidateArgs() {
		return []string{"Invalid command syntax. Type 'help' for usage information."}, nil
	}

	handler, exists := e.handlers[cmd.Verb]
	if !exists {
		return []string{fmt.Sprintf("Command '%s' is not implemented yet.", cmd.Verb)}, nil
	}

	return handler.Execute(cmd)
}

func (e *Executor) initializeHandlers() {
	// Movement handlers
	e.handlers["north"] = &MovementHandler{direction: "north", world: e.worldActions}
	e.handlers["south"] = &MovementHandler{direction: "south", world: e.worldActions}
	e.handlers["east"] = &MovementHandler{direction: "east", world: e.worldActions}
	e.handlers["west"] = &MovementHandler{direction: "west", world: e.worldActions}
	e.handlers["up"] = &MovementHandler{direction: "up", world: e.worldActions}
	e.handlers["down"] = &MovementHandler{direction: "down", world: e.worldActions}
	e.handlers["northeast"] = &MovementHandler{direction: "northeast", world: e.worldActions}
	e.handlers["northwest"] = &MovementHandler{direction: "northwest", world: e.worldActions}
	e.handlers["southeast"] = &MovementHandler{direction: "southeast", world: e.worldActions}
	e.handlers["southwest"] = &MovementHandler{direction: "southwest", world: e.worldActions}

	// Communication handlers
	e.handlers["say"] = &SayHandler{world: e.worldActions}
	e.handlers["tell"] = &TellHandler{world: e.worldActions}

	// Information handlers
	e.handlers["look"] = &LookHandler{world: e.worldActions}
	e.handlers["who"] = &WhoHandler{}
	e.handlers["score"] = &ScoreHandler{repoManager: e.repoManager}

	// Inventory handlers
	e.handlers["inventory"] = &InventoryHandler{repoManager: e.repoManager, templates: e.templates}
	// get and drop run only through ExecuteAction, which delivers their room messages.

	// System handlers
	e.handlers["quit"] = &QuitHandler{}
	e.handlers["save"] = &SaveHandler{repoManager: e.repoManager}

	e.handlers["emote"] = &EmoteHandler{world: e.worldActions}
	e.handlers["help"] = &HelpHandler{}
	e.handlers["commands"] = &CommandsHandler{}

}

// Basic handler implementations

type MovementHandler struct {
	direction string
	world     *actions.World
}

func (h *MovementHandler) Execute(cmd *Command) ([]string, error) {
	return h.world.Move(cmd.CharacterID, h.direction).Replies(cmd.CharacterID), nil
}

type SayHandler struct{ world *actions.World }

func (h *SayHandler) Execute(cmd *Command) ([]string, error) {
	return h.world.Say(cmd.CharacterID, strings.Join(cmd.Args, " ")).Replies(cmd.CharacterID), nil
}

type TellHandler struct {
	world *actions.World
}

func (h *TellHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) < 2 {
		return []string{"Usage: tell <character> <message>"}, nil
	}
	return h.world.Tell(cmd.CharacterID, cmd.Args[0], strings.Join(cmd.Args[1:], " ")).Replies(cmd.CharacterID), nil
}

type LookHandler struct {
	world *actions.World
}

func (h *LookHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) == 0 {
		return h.world.Look(cmd.CharacterID).Replies(cmd.CharacterID), nil
	}
	return []string{"Looking at a particular thing isn't available yet; 'look' on its own shows the room."}, nil
}

type WhoHandler struct {
	onlineCharacters func() []interfaces.OnlineCharacter
}

func (h *WhoHandler) Execute(cmd *Command) ([]string, error) {
	responses := []string{"Players currently online:"}
	var online []interfaces.OnlineCharacter
	if h.onlineCharacters != nil {
		online = h.onlineCharacters()
	}
	for _, c := range online {
		responses = append(responses, fmt.Sprintf("  %s (%s %s, Level %d)", c.Name, c.Race, c.Class, c.Level))
	}
	noun := "players"
	if len(online) == 1 {
		noun = "player"
	}
	return append(responses, "", fmt.Sprintf("%d %s online.", len(online), noun)), nil
}

type ScoreHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *ScoreHandler) Execute(cmd *Command) ([]string, error) {
	// Get character information
	char, err := h.repoManager.Characters().GetCharacter(cmd.CharacterID)
	if err != nil {
		return []string{"Error retrieving character information."}, nil
	}

	return []string{
		fmt.Sprintf("Name: %s", char.Name),
		fmt.Sprintf("Race: %s, Class: %s", char.Race.Name, char.Class.Name),
		fmt.Sprintf("Level: %d, Experience: %d", char.Level, char.Experience),
		fmt.Sprintf("Health: %d/%d", char.Stats.Health, char.Stats.MaxHealth),
		fmt.Sprintf("Mana: %d/%d", char.Stats.Mana, char.Stats.MaxMana),
		fmt.Sprintf("Stamina: %d/%d", char.Stats.Stamina, char.Stats.MaxStamina),
	}, nil
}

type InventoryHandler struct {
	repoManager interfaces.RepositoryManager
	templates   *items.ItemRegistry
}

func (h *InventoryHandler) Execute(cmd *Command) ([]string, error) {
	// Get character's items
	items, err := h.repoManager.Items().GetPlayerItems(cmd.CharacterID)
	if err != nil {
		return []string{"Error retrieving inventory."}, nil
	}

	if len(items) == 0 {
		return []string{"You are carrying nothing."}, nil
	}

	response := []string{"You are carrying:"}
	for _, item := range items {
		response = append(response, "  "+h.templates.Label(item))
	}

	return response, nil
}

type QuitHandler struct{}

func (h *QuitHandler) Execute(cmd *Command) ([]string, error) {
	return []string{"Saving character and disconnecting..."}, nil
}

type SaveHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *SaveHandler) Execute(cmd *Command) ([]string, error) {
	// Save character
	char, err := h.repoManager.Characters().GetCharacter(cmd.CharacterID)
	if err != nil {
		return []string{"Error saving character."}, nil
	}

	char.UpdatePlayTime()
	err = h.repoManager.Characters().UpdateCharacter(char)
	if err != nil {
		return []string{"Error saving character."}, nil
	}

	return []string{"Character saved."}, nil
}

type EmoteHandler struct{ world *actions.World }

func (h *EmoteHandler) Execute(cmd *Command) ([]string, error) {
	return h.world.Emote(cmd.CharacterID, strings.Join(cmd.Args, " ")).Replies(cmd.CharacterID), nil
}

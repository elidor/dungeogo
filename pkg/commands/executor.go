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
	e.handlers["yell"] = &YellHandler{}
	e.handlers["whisper"] = &WhisperHandler{}
	e.handlers["chat"] = &ChatHandler{}

	// Information handlers
	e.handlers["look"] = &LookHandler{world: e.worldActions}
	e.handlers["examine"] = &ExamineHandler{repoManager: e.repoManager}
	e.handlers["who"] = &WhoHandler{}
	e.handlers["score"] = &ScoreHandler{repoManager: e.repoManager}
	e.handlers["time"] = &TimeHandler{}
	e.handlers["weather"] = &WeatherHandler{}

	// Inventory handlers
	e.handlers["inventory"] = &InventoryHandler{repoManager: e.repoManager, templates: e.templates}
	e.handlers["get"] = &GetHandler{world: e.worldActions}
	e.handlers["drop"] = &DropHandler{world: e.worldActions}
	e.handlers["give"] = &GiveHandler{repoManager: e.repoManager}
	e.handlers["wear"] = &WearHandler{repoManager: e.repoManager}
	e.handlers["remove"] = &RemoveHandler{repoManager: e.repoManager}

	// Skill handlers
	e.handlers["skills"] = &SkillsHandler{repoManager: e.repoManager}
	e.handlers["practice"] = &PracticeHandler{repoManager: e.repoManager}

	// System handlers
	e.handlers["help"] = &HelpHandler{}
	e.handlers["commands"] = &CommandsHandler{}
	e.handlers["quit"] = &QuitHandler{}
	e.handlers["save"] = &SaveHandler{repoManager: e.repoManager}

	// Social handlers
	e.handlers["emote"] = &EmoteHandler{world: e.worldActions}
	e.handlers["smile"] = &SocialHandler{action: "smile"}
	e.handlers["wave"] = &SocialHandler{action: "wave"}
	e.handlers["bow"] = &SocialHandler{action: "bow"}

	// Combat handlers (basic implementations)
	e.handlers["kill"] = &KillHandler{repoManager: e.repoManager}
	e.handlers["flee"] = &FleeHandler{}
	e.handlers["defend"] = &DefendHandler{}
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

type YellHandler struct{}

func (h *YellHandler) Execute(cmd *Command) ([]string, error) {
	message := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You yell: %s", message)}, nil
}

type WhisperHandler struct{}

func (h *WhisperHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) < 2 {
		return []string{"Usage: whisper <player> <message>"}, nil
	}

	target := cmd.Args[0]
	message := strings.Join(cmd.Args[1:], " ")

	return []string{fmt.Sprintf("You whisper to %s: %s", target, message)}, nil
}

type ChatHandler struct{}

func (h *ChatHandler) Execute(cmd *Command) ([]string, error) {
	message := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("[Chat] You: %s", message)}, nil
}

type LookHandler struct {
	world *actions.World
}

func (h *LookHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) == 0 {
		return h.world.Look(cmd.CharacterID).Replies(cmd.CharacterID), nil
	}
	return []string{fmt.Sprintf("You look at %s.", strings.Join(cmd.Args, " "))}, nil
}

type ExamineHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *ExamineHandler) Execute(cmd *Command) ([]string, error) {
	target := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You examine %s closely.", target)}, nil
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

type TimeHandler struct{}

func (h *TimeHandler) Execute(cmd *Command) ([]string, error) {
	return []string{"It is midday in the realm."}, nil
}

type WeatherHandler struct{}

func (h *WeatherHandler) Execute(cmd *Command) ([]string, error) {
	return []string{"The weather is clear and pleasant."}, nil
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

type GetHandler struct{ world *actions.World }

func (h *GetHandler) Execute(cmd *Command) ([]string, error) {
	return h.world.Get(cmd.CharacterID, strings.Join(cmd.Args, " ")).Replies(cmd.CharacterID), nil
}

type DropHandler struct{ world *actions.World }

func (h *DropHandler) Execute(cmd *Command) ([]string, error) {
	return h.world.Drop(cmd.CharacterID, strings.Join(cmd.Args, " ")).Replies(cmd.CharacterID), nil
}

type GiveHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *GiveHandler) Execute(cmd *Command) ([]string, error) {
	item := cmd.Args[0]
	target := cmd.Args[1]
	return []string{fmt.Sprintf("You give %s to %s.", item, target)}, nil
}

type WearHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *WearHandler) Execute(cmd *Command) ([]string, error) {
	item := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You wear %s.", item)}, nil
}

type RemoveHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *RemoveHandler) Execute(cmd *Command) ([]string, error) {
	item := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You remove %s.", item)}, nil
}

type SkillsHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *SkillsHandler) Execute(cmd *Command) ([]string, error) {
	// Get character's skills
	_, err := h.repoManager.Characters().GetCharacter(cmd.CharacterID)
	if err != nil {
		return []string{"Error retrieving character skills."}, nil
	}

	response := []string{"Your skills:"}
	// This would iterate through actual skills
	response = append(response, "  Swords: 15", "  Magic: 8", "  Stealth: 12")

	return response, nil
}

type PracticeHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *PracticeHandler) Execute(cmd *Command) ([]string, error) {
	skill := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You practice %s.", skill)}, nil
}

type HelpHandler struct{}

func (h *HelpHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) == 0 {
		return []string{
			"Available command categories:",
			"  movement - Movement commands (north, south, etc.)",
			"  communication - Chat commands (say, tell, etc.)",
			"  inventory - Item commands (get, drop, wear, etc.)",
			"  information - Info commands (look, examine, who, etc.)",
			"  skills - Skill commands (skills, practice)",
			"  social - Social commands (emote, smile, etc.)",
			"",
			"Type 'help <category>' for specific commands.",
			"Type 'commands' to list all available commands.",
		}, nil
	}

	topic := strings.ToLower(cmd.Args[0])
	switch topic {
	case "movement":
		return []string{
			"Movement commands:",
			"  north, south, east, west (n, s, e, w)",
			"  up, down (u, d)",
			"  northeast, northwest, southeast, southwest (ne, nw, se, sw)",
		}, nil
	case "communication":
		return []string{
			"Communication commands:",
			"  say <message> (') - Say something to everyone in the room",
			"  tell <player> <message> (t) - Send private message",
			"  yell <message> - Yell across the area",
			"  whisper <player> <message> (w) - Whisper to someone",
			"  chat <message> (.) - Talk on global chat channel",
		}, nil
	default:
		return []string{fmt.Sprintf("No help available for topic: %s", topic)}, nil
	}
}

type CommandsHandler struct{}

func (h *CommandsHandler) Execute(cmd *Command) ([]string, error) {
	return []string{
		"Available commands:",
		"Movement: north, south, east, west, up, down, ne, nw, se, sw",
		"Communication: say, tell, yell, whisper, chat",
		"Information: look, examine, who, score, time, weather",
		"Inventory: inventory, get, drop, give, wear, remove",
		"Skills: skills, practice",
		"Social: emote, smile, wave, bow",
		"System: help, commands, quit, save",
	}, nil
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

type SocialHandler struct {
	action string
}

func (h *SocialHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) == 0 {
		return []string{fmt.Sprintf("You %s.", h.action)}, nil
	}

	target := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You %s at %s.", h.action, target)}, nil
}

type KillHandler struct {
	repoManager interfaces.RepositoryManager
}

func (h *KillHandler) Execute(cmd *Command) ([]string, error) {
	target := strings.Join(cmd.Args, " ")
	return []string{fmt.Sprintf("You attack %s!", target)}, nil
}

type FleeHandler struct{}

func (h *FleeHandler) Execute(cmd *Command) ([]string, error) {
	return []string{"You attempt to flee from combat!"}, nil
}

type DefendHandler struct{}

func (h *DefendHandler) Execute(cmd *Command) ([]string, error) {
	return []string{"You focus on defending yourself."}, nil
}

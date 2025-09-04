package commands

import (
	"strings"
)

type Command struct {
	Type      CommandType
	Verb      string
	Args      []string
	RawInput  string
	PlayerID  string
	CharacterID string
}

type CommandType int

const (
	CommandMovement CommandType = iota
	CommandCommunication
	CommandInventory
	CommandCombat
	CommandMagic
	CommandSkill
	CommandInformation
	CommandSystem
	CommandSocial
	CommandAdmin
	CommandUnknown
)

type Parser struct {
	aliases map[string]string
	commands map[string]CommandInfo
}

type CommandInfo struct {
	Type        CommandType
	Description string
	Usage       string
	MinArgs     int
	MaxArgs     int
	Aliases     []string
}

func NewParser() *Parser {
	p := &Parser{
		aliases:  make(map[string]string),
		commands: make(map[string]CommandInfo),
	}
	
	p.initializeCommands()
	return p
}

func (p *Parser) Parse(input, playerID, characterID string) *Command {
	input = strings.TrimSpace(input)
	if input == "" {
		return &Command{
			Type:        CommandUnknown,
			RawInput:    input,
			PlayerID:    playerID,
			CharacterID: characterID,
		}
	}
	
	parts := strings.Fields(input)
	verb := strings.ToLower(parts[0])
	args := parts[1:]
	
	// Resolve aliases
	if alias, exists := p.aliases[verb]; exists {
		verb = alias
	}
	
	// Determine command type
	cmdType := CommandUnknown
	if cmdInfo, exists := p.commands[verb]; exists {
		cmdType = cmdInfo.Type
	}
	
	return &Command{
		Type:        cmdType,
		Verb:        verb,
		Args:        args,
		RawInput:    input,
		PlayerID:    playerID,
		CharacterID: characterID,
	}
}

func (p *Parser) GetCommandInfo(verb string) (CommandInfo, bool) {
	// Resolve aliases
	if alias, exists := p.aliases[verb]; exists {
		verb = alias
	}
	
	info, exists := p.commands[verb]
	return info, exists
}

func (p *Parser) GetCommandsByType(cmdType CommandType) []string {
	var commands []string
	for verb, info := range p.commands {
		if info.Type == cmdType {
			commands = append(commands, verb)
		}
	}
	return commands
}

func (p *Parser) initializeCommands() {
	// Movement commands
	p.addCommand("north", CommandMovement, "Move north", "north", 0, 0, []string{"n"})
	p.addCommand("south", CommandMovement, "Move south", "south", 0, 0, []string{"s"})
	p.addCommand("east", CommandMovement, "Move east", "east", 0, 0, []string{"e"})
	p.addCommand("west", CommandMovement, "Move west", "west", 0, 0, []string{"w"})
	p.addCommand("up", CommandMovement, "Move up", "up", 0, 0, []string{"u"})
	p.addCommand("down", CommandMovement, "Move down", "down", 0, 0, []string{"d"})
	p.addCommand("northeast", CommandMovement, "Move northeast", "northeast", 0, 0, []string{"ne"})
	p.addCommand("northwest", CommandMovement, "Move northwest", "northwest", 0, 0, []string{"nw"})
	p.addCommand("southeast", CommandMovement, "Move southeast", "southeast", 0, 0, []string{"se"})
	p.addCommand("southwest", CommandMovement, "Move southwest", "southwest", 0, 0, []string{"sw"})
	
	// Communication commands
	p.addCommand("say", CommandCommunication, "Say something to the room", "say <message>", 1, -1, []string{"'"})
	p.addCommand("tell", CommandCommunication, "Send a private message", "tell <player> <message>", 2, -1, []string{"t"})
	p.addCommand("yell", CommandCommunication, "Yell across the area", "yell <message>", 1, -1, []string{})
	p.addCommand("whisper", CommandCommunication, "Whisper to someone", "whisper <player> <message>", 2, -1, []string{})
	p.addCommand("chat", CommandCommunication, "Chat on global channel", "chat <message>", 1, -1, []string{"."})
	
	// Inventory commands
	p.addCommand("inventory", CommandInventory, "Show your inventory", "inventory", 0, 0, []string{"i", "inv"})
	p.addCommand("get", CommandInventory, "Pick up an item", "get <item>", 1, 1, []string{"take"})
	p.addCommand("drop", CommandInventory, "Drop an item", "drop <item>", 1, 1, []string{})
	p.addCommand("give", CommandInventory, "Give an item to someone", "give <item> <player>", 2, 2, []string{})
	p.addCommand("wear", CommandInventory, "Wear/wield an item", "wear <item>", 1, 1, []string{"wield", "equip"})
	p.addCommand("remove", CommandInventory, "Remove worn item", "remove <item>", 1, 1, []string{"unwield"})
	
	// Combat commands
	p.addCommand("kill", CommandCombat, "Attack a target", "kill <target>", 1, 1, []string{"k", "attack"})
	p.addCommand("flee", CommandCombat, "Attempt to escape combat", "flee", 0, 0, []string{})
	p.addCommand("defend", CommandCombat, "Focus on defense", "defend", 0, 0, []string{})
	
	// Magic commands
	p.addCommand("cast", CommandMagic, "Cast a spell", "cast <spell> [target]", 1, 2, []string{"c"})
	p.addCommand("prepare", CommandMagic, "Prepare a spell", "prepare <spell>", 1, 1, []string{"prep"})
	
	// Information commands
	p.addCommand("look", CommandInformation, "Look at surroundings", "look [target]", 0, 1, []string{"l"})
	p.addCommand("examine", CommandInformation, "Examine something closely", "examine <target>", 1, 1, []string{"ex", "exa"})
	p.addCommand("who", CommandInformation, "List online players", "who", 0, 0, []string{})
	p.addCommand("score", CommandInformation, "Show character stats", "score", 0, 0, []string{"sc"})
	p.addCommand("time", CommandInformation, "Show game time", "time", 0, 0, []string{})
	p.addCommand("weather", CommandInformation, "Show weather", "weather", 0, 0, []string{})
	
	// Skill commands
	p.addCommand("skills", CommandSkill, "Show skill levels", "skills", 0, 0, []string{"sk"})
	p.addCommand("practice", CommandSkill, "Practice a skill", "practice <skill>", 1, 1, []string{"prac"})
	
	// Social commands
	p.addCommand("emote", CommandSocial, "Perform an emote", "emote <action>", 1, -1, []string{"em", ":"})
	p.addCommand("smile", CommandSocial, "Smile at someone", "smile [target]", 0, 1, []string{})
	p.addCommand("wave", CommandSocial, "Wave at someone", "wave [target]", 0, 1, []string{})
	p.addCommand("bow", CommandSocial, "Bow to someone", "bow [target]", 0, 1, []string{})
	
	// System commands
	p.addCommand("quit", CommandSystem, "Quit the game", "quit", 0, 0, []string{"q"})
	p.addCommand("save", CommandSystem, "Save character", "save", 0, 0, []string{})
	p.addCommand("help", CommandSystem, "Show help", "help [topic]", 0, 1, []string{"h"})
	p.addCommand("commands", CommandSystem, "List available commands", "commands", 0, 0, []string{"cmd"})
}

func (p *Parser) addCommand(verb string, cmdType CommandType, description, usage string, minArgs, maxArgs int, aliases []string) {
	p.commands[verb] = CommandInfo{
		Type:        cmdType,
		Description: description,
		Usage:       usage,
		MinArgs:     minArgs,
		MaxArgs:     maxArgs,
		Aliases:     aliases,
	}
	
	// Add aliases
	for _, alias := range aliases {
		p.aliases[alias] = verb
	}
}

func (cmd *Command) ValidateArgs() bool {
	info, exists := cmd.getCommandInfo()
	if !exists {
		return true // Unknown commands are handled elsewhere
	}
	
	argCount := len(cmd.Args)
	
	if argCount < info.MinArgs {
		return false
	}
	
	if info.MaxArgs >= 0 && argCount > info.MaxArgs {
		return false
	}
	
	return true
}

func (cmd *Command) getCommandInfo() (CommandInfo, bool) {
	// This would need access to the parser - for now return default
	return CommandInfo{}, false
}

func (cmd *Command) GetTypeName() string {
	switch cmd.Type {
	case CommandMovement:
		return "Movement"
	case CommandCommunication:
		return "Communication"
	case CommandInventory:
		return "Inventory"
	case CommandCombat:
		return "Combat"
	case CommandMagic:
		return "Magic"
	case CommandSkill:
		return "Skill"
	case CommandInformation:
		return "Information"
	case CommandSystem:
		return "System"
	case CommandSocial:
		return "Social"
	case CommandAdmin:
		return "Admin"
	default:
		return "Unknown"
	}
}
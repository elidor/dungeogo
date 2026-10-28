package commands

import (
	"fmt"
	"sort"
	"strings"
)

// notYetAvailable lists verbs the parser recognises but the game doesn't
// implement yet. They answer that plainly instead of pretending to work.
var notYetAvailable = map[string]bool{
	"yell": true, "whisper": true, "chat": true,
	"wear": true, "remove": true,
	"kill": true, "flee": true, "defend": true,
	"cast": true, "prepare": true,
	"examine": true, "time": true, "weather": true,
	"skills": true, "practice": true,
	"smile": true, "wave": true, "bow": true,
}

// commandGroup is one help topic: the commands that work, as players type them.
type commandGroup struct {
	topic    string
	aliases  []string
	commands []string
}

// availableCommands is the single list behind both 'help' and 'commands'.
var availableCommands = []commandGroup{
	{"movement", nil, []string{
		"north, south, east, west (n, s, e, w)",
		"up, down (u, d)",
		"northeast, northwest, southeast, southwest (ne, nw, se, sw)",
	}},
	{"communication", nil, []string{
		"say <message> (') - say something to everyone in the room",
		"tell <character> <message> (t) - send a private message to someone online",
		"emote <action> (:) - describe an action to the room",
	}},
	{"items", []string{"inventory"}, []string{
		"inventory (i) - list what you carry",
		"get <item> (take) - pick something up from the room",
		"drop <item> - put something down",
		"give <item> <character> - hand something to someone in the room",
		"Items are named by a word in their name; 2.sword means the second sword.",
	}},
	{"information", nil, []string{
		"look (l) - describe the room, who's here and what's lying around",
		"who - list characters online",
		"score (sc) - show your character",
	}},
	{"system", nil, []string{
		"save - save your character",
		"quit (q) - save and disconnect",
		"help [topic] (h) - this help",
		"commands (cmd) - list every command",
	}},
}

func unavailableVerbs() string {
	verbs := make([]string, 0, len(notYetAvailable))
	for verb := range notYetAvailable {
		verbs = append(verbs, verb)
	}
	sort.Strings(verbs)
	return strings.Join(verbs, ", ")
}

type HelpHandler struct{}

func (h *HelpHandler) Execute(cmd *Command) ([]string, error) {
	if len(cmd.Args) == 0 {
		lines := []string{"Help topics:"}
		for _, g := range availableCommands {
			lines = append(lines, "  "+g.topic)
		}
		return append(lines, "", "Type 'help <topic>' for details, or 'commands' for everything."), nil
	}
	topic := strings.ToLower(cmd.Args[0])
	for _, g := range availableCommands {
		if g.topic == topic || contains(g.aliases, topic) {
			lines := []string{strings.ToUpper(g.topic[:1]) + g.topic[1:] + ":"}
			for _, c := range g.commands {
				lines = append(lines, "  "+c)
			}
			return lines, nil
		}
	}
	if notYetAvailable[topic] {
		return []string{fmt.Sprintf("'%s' isn't available yet.", topic)}, nil
	}
	return []string{fmt.Sprintf("No help available for '%s'. Type 'help' for topics.", topic)}, nil
}

type CommandsHandler struct{}

func (h *CommandsHandler) Execute(cmd *Command) ([]string, error) {
	lines := []string{"Commands you can use:"}
	for _, g := range availableCommands {
		lines = append(lines, "", strings.ToUpper(g.topic[:1])+g.topic[1:]+":")
		for _, c := range g.commands {
			lines = append(lines, "  "+c)
		}
	}
	return append(lines, "", "Not available yet: "+unavailableVerbs()+"."), nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

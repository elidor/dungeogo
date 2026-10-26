// Package world defines room vocabulary and addressed game-action results.
package world

import "github.com/elidor/dungeogo/pkg/game/character"

type Room struct {
	ID          string
	ZoneID      string
	Name        string
	Description string
	Exits       map[string]string
	Flags       map[string]interface{}
}

type OnlineCharacter struct {
	SessionID string
	ID        string
	Name      string
	Race      string
	Class     string
	Level     int
	RoomID    string
	ZoneID    string
}

// Presence supplies live character snapshots and applies committed locations.
// The transport authorizes the current session before invoking a game action.
type Presence interface {
	Characters() []OnlineCharacter
	Move(characterID string, location character.Location)
}

type Scope int

const (
	ToCharacter Scope = iota
	ToRoom
)

// DeliveryReport describes sender feedback after a transport delivery attempt.
// Successful socket writes acknowledge delivery; absent or failed recipients do not.
type DeliveryReport struct {
	CharacterID string
	Success     string
	Failure     string
}

// Message describes recipients without depending on network connections.
// SessionID pins private content to the session selected during name resolution.
type Message struct {
	SessionID          string
	Report             *DeliveryReport
	Scope              Scope
	CharacterID        string
	RoomID             string
	ExcludeCharacterID string
	Text               string
}

type Result struct{ Messages []Message }

func Reply(characterID string, lines ...string) Result {
	result := Result{}
	for _, line := range lines {
		result.Messages = append(result.Messages, Message{Scope: ToCharacter, CharacterID: characterID, Text: line})
	}
	return result
}

// Replies preserves the legacy command API, which returns only actor messages.
func (r Result) Replies(characterID string) []string {
	var lines []string
	for _, message := range r.Messages {
		if message.Scope == ToCharacter && message.CharacterID == characterID {
			lines = append(lines, message.Text)
		}
	}
	return lines
}

package actions

import (
	"fmt"
	"strings"

	"github.com/elidor/dungeogo/pkg/game/world"
)

func (s *World) Say(characterID, text string) world.Result {
	return s.localMessage(characterID, "say", text)
}

func (s *World) Emote(characterID, text string) world.Result {
	return s.localMessage(characterID, "emote", text)
}

// localMessage shares movement's action boundary. Presence selects live room
// recipients at delivery; gameplay describes the effect without touching sockets.
func (s *World) localMessage(characterID, verb, text string) world.Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	text = strings.TrimSpace(text)
	if text == "" {
		return world.Reply(characterID, "Please provide a message.")
	}
	actor, err := s.repositories.Characters().GetCharacter(characterID)
	if err != nil || actor.Location == nil {
		return world.Reply(characterID, "You cannot communicate right now.")
	}
	room, err := s.repositories.World().LoadRoomDefinition(actor.Location.RoomID)
	if err != nil || room.ZoneID != actor.Location.ZoneID {
		return world.Reply(characterID, "You cannot communicate from your current location.")
	}
	if s.presence != nil {
		current := false
		for _, c := range s.presence.Characters() {
			if c.ID == characterID && c.RoomID == room.ID && c.ZoneID == room.ZoneID {
				current = true
				break
			}
		}
		if !current {
			return world.Reply(characterID, "You cannot communicate from your current location.")
		}
	}
	sender, recipient := "You "+text, actor.Name+" "+text
	if verb == "say" {
		sender = "You say: " + text
		recipient = fmt.Sprintf("%s says: %s", actor.Name, text)
	}
	result := world.Reply(characterID, sender)
	result.Messages = append(result.Messages, world.Message{Scope: world.ToRoom, RoomID: room.ID, ExcludeCharacterID: characterID, Text: recipient})
	return result
}

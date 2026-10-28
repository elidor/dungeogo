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
	room, ok := s.presentRoom(actor)
	if !ok {
		return world.Reply(characterID, "You cannot communicate from your current location.")
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

// Tell resolves live character identity and pins the effect to that session so
// a later replacement cannot receive private content intended for its predecessor.
func (s *World) Tell(characterID, target, text string) world.Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	target = strings.TrimSpace(target)
	text = strings.TrimSpace(text)
	if target == "" || text == "" {
		return world.Reply(characterID, "Usage: tell <character> <message>")
	}
	unavailable := fmt.Sprintf("Character %q is not available for private messages.", target)
	if s.presence == nil {
		return world.Reply(characterID, unavailable)
	}
	var sender *world.OnlineCharacter
	var matches []world.OnlineCharacter
	var exact *world.OnlineCharacter
	for _, c := range s.presence.Characters() {
		if c.ID == characterID {
			copy := c
			sender = &copy
		}
		if strings.EqualFold(c.Name, target) {
			matches = append(matches, c)
		}
		if c.Name == target {
			copy := c
			exact = &copy
		}
	}
	if sender == nil {
		return world.Reply(characterID, "You cannot send a private message right now.")
	}
	var recipient world.OnlineCharacter
	if exact != nil {
		recipient = *exact
	} else if len(matches) == 1 {
		recipient = matches[0]
	} else if len(matches) > 1 {
		return world.Reply(characterID, "More than one online character matches; use the exact capitalization of the character name.")
	} else {
		return world.Reply(characterID, unavailable)
	}
	if recipient.SessionID == "" {
		return world.Reply(characterID, unavailable)
	}
	return world.Result{Messages: []world.Message{{
		Scope: world.ToCharacter, CharacterID: recipient.ID, SessionID: recipient.SessionID,
		Text:   fmt.Sprintf("%s tells you: %s", sender.Name, text),
		Report: &world.DeliveryReport{CharacterID: characterID, Success: fmt.Sprintf("You tell %s: %s", recipient.Name, text), Failure: unavailable},
	}}}
}

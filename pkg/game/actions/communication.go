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
	result.Messages = append(result.Messages, world.RoomMessage(room.ID, recipient, characterID))
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
	online := s.presence.Characters()
	var sender *world.OnlineCharacter
	for _, c := range online {
		if c.ID == characterID {
			copy := c
			sender = &copy
		}
	}
	if sender == nil {
		return world.Reply(characterID, "You cannot send a private message right now.")
	}
	recipient, found, ambiguous := matchCharacter(online, target)
	if ambiguous {
		return world.Reply(characterID, "More than one online character matches; use the exact capitalization of the character name.")
	}
	if !found {
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

// matchCharacter finds the Character a player named among candidates. Names
// match ignoring case; if several do, an exact-case match wins, and otherwise
// the name is ambiguous.
func matchCharacter(candidates []world.OnlineCharacter, name string) (match world.OnlineCharacter, found, ambiguous bool) {
	var matches []world.OnlineCharacter
	for _, c := range candidates {
		if c.Name == name {
			return c, true, false
		}
		if strings.EqualFold(c.Name, name) {
			matches = append(matches, c)
		}
	}
	switch len(matches) {
	case 0:
		return world.OnlineCharacter{}, false, false
	case 1:
		return matches[0], true, false
	default:
		return world.OnlineCharacter{}, false, true
	}
}

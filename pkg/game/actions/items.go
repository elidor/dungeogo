package actions

import (
	"fmt"
	"strings"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/world"
)

// presentRoom loads the Room a Character's saved location names and checks that
// live Presence agrees the Character is there now. Callers hold the action lock,
// so the answer stays true until they finish.
func (s *World) presentRoom(actor *character.Character) (*world.Room, bool) {
	room, err := s.repositories.World().LoadRoomDefinition(actor.Location.RoomID)
	if err != nil || room.ZoneID != actor.Location.ZoneID {
		return nil, false
	}
	if s.presence == nil {
		return room, true
	}
	for _, c := range s.presence.Characters() {
		if c.ID == actor.ID && c.RoomID == room.ID && c.ZoneID == room.ZoneID {
			return room, true
		}
	}
	return nil, false
}

// itemMove is the wording for one direction of get or drop. Get moves an Item
// from the actor's Room to the actor; drop is the mirror image.
type itemMove struct {
	fromRoom  bool
	usage     string // shown for an unreadable target
	unable    string // shown when the action can't run
	missing   string // format: no Item matches the target
	gone      string // format: another writer moved the Item first
	actorVerb string // "pick up"
	roomVerb  string // "picks up"
}

var (
	getMove = itemMove{
		fromRoom:  true,
		usage:     "Get what? Use get <item>, or get 2.<item> for the second one.",
		unable:    "You can't pick anything up right now.",
		missing:   "You don't see a '%s' here.",
		gone:      "Someone got to the %s first.",
		actorVerb: "pick up",
		roomVerb:  "picks up",
	}
	dropMove = itemMove{
		usage:     "Drop what? Use drop <item>, or drop 2.<item> for the second one.",
		unable:    "You can't drop anything right now.",
		missing:   "You aren't carrying a '%s'.",
		gone:      "You aren't carrying the %s any more.",
		actorVerb: "drop",
		roomVerb:  "drops",
	}
)

// Get moves an Item from the actor's Room into their inventory.
func (s *World) Get(characterID, input string) world.Result {
	return s.moveItem(characterID, input, getMove)
}

// Drop moves an Item from the actor's inventory into their Room.
func (s *World) Drop(characterID, input string) world.Result {
	return s.moveItem(characterID, input, dropMove)
}

// moveItem runs get or drop under the action lock shared with movement, so the
// actor's Room can't change part-way. The transfer itself is a conditional write
// (ADR-0002): success is reported, to the actor and the Room, only after it
// commits; if another writer moved the Item first, nothing changes.
func (s *World) moveItem(characterID, input string, m itemMove) world.Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	target, err := items.ParseTarget(input)
	if err != nil {
		return world.Reply(characterID, m.usage)
	}
	actor, err := s.repositories.Characters().GetCharacter(characterID)
	if err != nil || actor.Location == nil {
		return world.Reply(characterID, m.unable)
	}
	room, ok := s.presentRoom(actor)
	if !ok {
		return world.Reply(characterID, m.unable)
	}
	from, to := items.RoomOwner(room.ID), items.CharacterOwner(actor.ID)
	if !m.fromRoom {
		from, to = to, from
	}
	var candidates []*items.ItemInstance
	if m.fromRoom {
		candidates, err = s.repositories.Items().GetRoomItems(room.ID)
	} else {
		candidates, err = s.repositories.Items().GetPlayerItems(actor.ID)
	}
	if err != nil {
		s.logger.Printf("item %s unavailable: character=%s room=%s: %v", m.actorVerb, actor.ID, room.ID, err)
		return world.Reply(characterID, m.unable)
	}
	item, found := target.Pick(candidates, s.templates)
	if !found {
		return world.Reply(characterID, fmt.Sprintf(m.missing, target))
	}
	moved, err := s.repositories.Items().TransferItem(item.ID, from, to)
	if err != nil {
		s.logger.Printf("item %s failed: character=%s item=%s: %v", m.actorVerb, actor.ID, item.ID, err)
		return world.Reply(characterID, m.unable)
	}
	if !moved {
		return world.Reply(characterID, fmt.Sprintf(m.gone, s.templates.DisplayName(item)))
	}
	short := s.templates.ShortDescription(item)
	result := world.Reply(characterID, fmt.Sprintf("You %s %s.", m.actorVerb, short))
	result.Messages = append(result.Messages, world.RoomMessage(room.ID, fmt.Sprintf("%s %s %s.", actor.Name, m.roomVerb, short), actor.ID))
	return result
}

// Give moves an Item from the actor to another Character in the same Room. The
// recipient must be online and present there under the same action lock as
// movement; the transfer is the conditional write, and the giver, recipient
// and onlookers are told only after it commits.
func (s *World) Give(characterID, itemInput, recipientName string) world.Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	const unable = "You can't give anything right now."
	target, err := items.ParseTarget(itemInput)
	recipientName = strings.TrimSpace(recipientName)
	if err != nil || recipientName == "" {
		return world.Reply(characterID, GiveUsage)
	}
	actor, err := s.repositories.Characters().GetCharacter(characterID)
	if err != nil || actor.Location == nil {
		return world.Reply(characterID, unable)
	}
	room, ok := s.presentRoom(actor)
	if !ok {
		return world.Reply(characterID, unable)
	}
	var here []world.OnlineCharacter
	if s.presence != nil {
		for _, c := range s.presence.Characters() {
			if c.RoomID == room.ID && c.ZoneID == room.ZoneID {
				here = append(here, c)
			}
		}
	}
	recipient, found, ambiguous := matchCharacter(here, recipientName)
	switch {
	case ambiguous:
		return world.Reply(characterID, "More than one character here has that name; use its exact capitalization.")
	case !found:
		return world.Reply(characterID, fmt.Sprintf("There's no one called '%s' here.", recipientName))
	case recipient.ID == actor.ID:
		return world.Reply(characterID, "You can't give something to yourself.")
	}
	carried, err := s.repositories.Items().GetPlayerItems(actor.ID)
	if err != nil {
		s.logger.Printf("item give unavailable: character=%s: %v", actor.ID, err)
		return world.Reply(characterID, unable)
	}
	item, found := target.Pick(carried, s.templates)
	if !found {
		return world.Reply(characterID, fmt.Sprintf("You aren't carrying a '%s'.", target))
	}
	moved, err := s.repositories.Items().TransferItem(item.ID, items.CharacterOwner(actor.ID), items.CharacterOwner(recipient.ID))
	if err != nil {
		s.logger.Printf("item give failed: character=%s item=%s recipient=%s: %v", actor.ID, item.ID, recipient.ID, err)
		return world.Reply(characterID, unable)
	}
	if !moved {
		return world.Reply(characterID, fmt.Sprintf("You aren't carrying the %s any more.", s.templates.DisplayName(item)))
	}
	short := s.templates.ShortDescription(item)
	result := world.Reply(characterID, fmt.Sprintf("You give %s to %s.", short, recipient.Name))
	result.Messages = append(result.Messages,
		world.Message{Scope: world.ToCharacter, CharacterID: recipient.ID, SessionID: recipient.SessionID, Text: fmt.Sprintf("%s gives you %s.", actor.Name, short)},
		world.RoomMessage(room.ID, fmt.Sprintf("%s gives %s to %s.", actor.Name, short, recipient.Name), actor.ID, recipient.ID),
	)
	return result
}

// GiveUsage is the reply for anything other than give <item> <character>.
const GiveUsage = "Give what to whom? Use give <item> <character>, e.g. give sword Bo."

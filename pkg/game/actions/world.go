// Package actions coordinates durable world changes and live presence.
package actions

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/elidor/dungeogo/pkg/game/world"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

// World serializes room inspection and movement in this server process. The
// account's session gate must already be held by authenticated transport callers.
// No delivery occurs here; failed delivery cannot roll back committed game state.
type World struct {
	mutex        sync.Mutex
	repositories interfaces.RepositoryManager
	presence     world.Presence
	logger       *log.Logger // operator diagnostics; players never see these
}

func NewWorld(repositories interfaces.RepositoryManager) *World {
	return &World{repositories: repositories, logger: log.Default()}
}

// SetPresence is configured once, before accepting sessions.
func (s *World) SetPresence(presence world.Presence) { s.presence = presence }

func (s *World) Move(characterID, direction string) world.Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	actor, err := s.repositories.Characters().GetCharacter(characterID)
	if err != nil || actor.Location == nil {
		return world.Reply(characterID, "You cannot move right now.")
	}
	room, err := s.repositories.World().LoadRoomDefinition(actor.Location.RoomID)
	if err != nil {
		return world.Reply(characterID, "You cannot determine which way to go.")
	}
	destinationID, exists := room.Exits[direction]
	if !exists {
		return world.Reply(characterID, fmt.Sprintf("You cannot go %s from here.", direction))
	}
	destination, err := s.repositories.World().LoadRoomDefinition(destinationID)
	if err != nil {
		return world.Reply(characterID, fmt.Sprintf("The way %s is blocked.", direction))
	}
	location := *actor.Location
	location.RoomID = destination.ID
	location.ZoneID = destination.ZoneID
	if err := s.repositories.Characters().UpdateCharacterLocation(actor.ID, &location); err != nil {
		return world.Reply(characterID, "You cannot move right now.")
	}
	if s.presence != nil {
		s.presence.Move(actor.ID, location)
	}
	result := world.Reply(characterID, append([]string{fmt.Sprintf("You move %s.", direction)}, s.describe(destination, actor.ID)...)...)
	result.Messages = append(result.Messages,
		world.Message{Scope: world.ToRoom, RoomID: room.ID, ExcludeCharacterID: actor.ID, Text: fmt.Sprintf("%s leaves %s.", actor.Name, direction)},
		world.Message{Scope: world.ToRoom, RoomID: destination.ID, ExcludeCharacterID: actor.ID, Text: fmt.Sprintf("%s arrives.", actor.Name)},
	)
	return result
}

func (s *World) Look(characterID string) world.Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	actor, err := s.repositories.Characters().GetCharacter(characterID)
	if err != nil || actor.Location == nil {
		return world.Reply(characterID, "You are in an undefined room.")
	}
	room, err := s.repositories.World().LoadRoomDefinition(actor.Location.RoomID)
	if err != nil {
		return world.Reply(characterID, "You are in an undefined room.")
	}
	return world.Reply(characterID, s.describe(room, characterID)...)
}

func (s *World) describe(room *world.Room, actorID string) []string {
	directions := make([]string, 0, len(room.Exits))
	for direction := range room.Exits {
		directions = append(directions, direction)
	}
	sort.Strings(directions)
	lines := []string{room.Name, room.Description}
	if len(directions) == 0 {
		lines = append(lines, "There are no obvious exits.")
	} else {
		lines = append(lines, "Exits: "+strings.Join(directions, ", ")+".")
	}
	var occupants []string
	if s.presence != nil {
		for _, c := range s.presence.Characters() {
			if c.ID != actorID && c.RoomID == room.ID && c.ZoneID == room.ZoneID {
				occupants = append(occupants, c.Name)
			}
		}
	}
	sort.Strings(occupants)
	if len(occupants) > 0 {
		lines = append(lines, "Also here: "+strings.Join(occupants, ", ")+".")
	}
	roomItems, err := s.repositories.Items().GetRoomItems(room.ID)
	if err != nil {
		s.logger.Printf("room items unavailable: room=%s: %v", room.ID, err)
		lines = append(lines, "Room contents are unavailable.")
	} else if len(roomItems) > 0 {
		lines = append(lines, "Items here:")
		for _, item := range roomItems {
			lines = append(lines, "  "+item.DisplayLabel())
		}
	}
	return lines
}

package actions

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/world"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

// failingRoomItems is an item repository whose room listing always fails.
type failingRoomItems struct {
	interfaces.ItemRepository
	err error
}

func (f failingRoomItems) GetRoomItems(string) ([]*items.ItemInstance, error) { return nil, f.err }

type itemsOnly struct {
	interfaces.RepositoryManager
	items interfaces.ItemRepository
}

func (r itemsOnly) Items() interfaces.ItemRepository { return r.items }

func TestFailedRoomItemLookupIsLoggedNotShown(t *testing.T) {
	cause := errors.New("connection reset by peer")
	service := NewWorld(itemsOnly{items: failingRoomItems{err: cause}}, items.NewItemRegistry())
	var logged bytes.Buffer
	service.logger = log.New(&logged, "", 0)

	lines := service.describe(&world.Room{ID: "newbie_north", ZoneID: "newbie", Name: "North Path", Description: "A path."}, "actor")

	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "Room contents are unavailable.") {
		t.Fatalf("player message changed: %q", out)
	}
	if strings.Contains(out, cause.Error()) {
		t.Fatalf("storage error leaked to the player: %q", out)
	}
	entry := logged.String()
	if !strings.Contains(entry, "newbie_north") || !strings.Contains(entry, cause.Error()) {
		t.Fatalf("log entry %q should name the room and the error", entry)
	}
}

// raceLostRepos models another writer moving the Item between listing and
// transfer: the room still lists it, but the conditional transfer misses.
type raceLostRepos struct {
	interfaces.RepositoryManager
	actor *character.Character
	room  *world.Room
	item  *items.ItemInstance
}

func (r raceLostRepos) Characters() interfaces.CharacterRepository {
	return raceLostCharacters{actor: r.actor}
}
func (r raceLostRepos) World() interfaces.WorldRepository { return raceLostWorld{room: r.room} }
func (r raceLostRepos) Items() interfaces.ItemRepository  { return raceLostItems{item: r.item} }

type raceLostCharacters struct {
	interfaces.CharacterRepository
	actor *character.Character
}

func (r raceLostCharacters) GetCharacter(string) (*character.Character, error) { return r.actor, nil }

type raceLostWorld struct {
	interfaces.WorldRepository
	room *world.Room
}

func (r raceLostWorld) LoadRoomDefinition(string) (*interfaces.RoomDefinition, error) {
	return r.room, nil
}

type raceLostItems struct {
	interfaces.ItemRepository
	item *items.ItemInstance
}

func (r raceLostItems) GetRoomItems(string) ([]*items.ItemInstance, error) {
	return []*items.ItemInstance{r.item}, nil
}
func (r raceLostItems) TransferItem(string, items.Owner, items.Owner) (bool, error) {
	return false, nil
}

func TestGetReportsLostRaceWithoutRoomMessage(t *testing.T) {
	repos := raceLostRepos{
		actor: &character.Character{ID: "c1", Name: "Ana", Location: &character.Location{RoomID: "r1", ZoneID: "z"}},
		room:  &world.Room{ID: "r1", ZoneID: "z"},
		item:  &items.ItemInstance{ID: "i1", TemplateID: "smooth_stone", CustomName: "Contested Gem"},
	}
	service := NewWorld(repos, items.NewItemRegistry())
	result := service.Get("c1", "gem")
	if got := strings.Join(result.Replies("c1"), "\n"); got != "Someone got to the Contested Gem first." {
		t.Fatalf("loser reply = %q", got)
	}
	for _, m := range result.Messages {
		if m.Scope == world.ToRoom {
			t.Fatalf("lost race announced to the room: %+v", m)
		}
	}
}

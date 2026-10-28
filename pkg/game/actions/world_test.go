package actions

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

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

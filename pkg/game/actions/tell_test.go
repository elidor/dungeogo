package actions

import (
	"strings"
	"testing"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/world"
)

type tellPresence []world.OnlineCharacter

func (p tellPresence) Characters() []world.OnlineCharacter {
	return append([]world.OnlineCharacter(nil), p...)
}
func (p tellPresence) Move(string, character.Location) {}

func TestTellNameResolution(t *testing.T) {
	service := NewWorld(nil, items.NewItemRegistry())
	service.SetPresence(tellPresence{
		{ID: "sender", SessionID: "s", Name: "Sender"},
		{ID: "upper", SessionID: "u", Name: "Moonbeam"},
		{ID: "lower", SessionID: "l", Name: "moonbeam"},
	})
	for _, tc := range []struct{ target, id, session string }{
		{"Moonbeam", "upper", "u"}, {"moonbeam", "lower", "l"},
	} {
		result := service.Tell("sender", tc.target, "message")
		if len(result.Messages) != 1 {
			t.Fatal(result)
		}
		message := result.Messages[0]
		if message.CharacterID != tc.id || message.SessionID != tc.session || message.Report == nil {
			t.Fatalf("incorrect private identity: %+v", message)
		}
		if len(result.Replies("sender")) != 0 {
			t.Fatal("success confirmation must wait for delivery")
		}
	}
	result := service.Tell("sender", "MOONBEAM", "message")
	if out := strings.Join(result.Replies("sender"), " "); !strings.Contains(out, "exact capitalization") {
		t.Fatalf("ambiguous target: %q", out)
	}
	for _, message := range result.Messages {
		if message.Report != nil || message.CharacterID != "sender" {
			t.Fatal("ambiguous target created a delivery")
		}
	}
}

func TestMatchCharacter(t *testing.T) {
	here := []world.OnlineCharacter{{ID: "1", Name: "Bo"}, {ID: "2", Name: "Moonbeam"}, {ID: "3", Name: "moonbeam"}}
	for name, want := range map[string]string{"Bo": "1", "bo": "1", "BO": "1", "Moonbeam": "2", "moonbeam": "3"} {
		got, found, ambiguous := matchCharacter(here, name)
		if !found || ambiguous || got.ID != want {
			t.Errorf("matchCharacter(%q) = %+v found=%v ambiguous=%v, want ID %s", name, got, found, ambiguous, want)
		}
	}
	if _, found, ambiguous := matchCharacter(here, "MOONBEAM"); found || !ambiguous {
		t.Errorf("MOONBEAM should be ambiguous: found=%v ambiguous=%v", found, ambiguous)
	}
	if _, found, ambiguous := matchCharacter(here, "Zed"); found || ambiguous {
		t.Errorf("Zed should be not found: found=%v ambiguous=%v", found, ambiguous)
	}
}

func TestRoomMessageExcludesEveryListedCharacter(t *testing.T) {
	m := world.RoomMessage("r1", "Ana gives a Gem to Bo.", "ana", "bo")
	for id, want := range map[string]bool{"ana": true, "bo": true, "cy": false} {
		if got := m.Excludes(id); got != want {
			t.Errorf("Excludes(%q) = %v, want %v", id, got, want)
		}
	}
	if m.Scope != world.ToRoom || m.RoomID != "r1" {
		t.Errorf("RoomMessage = %+v", m)
	}
}

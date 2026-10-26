package actions

import (
	"strings"
	"testing"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/world"
)

type tellPresence []world.OnlineCharacter

func (p tellPresence) Characters() []world.OnlineCharacter {
	return append([]world.OnlineCharacter(nil), p...)
}
func (p tellPresence) Move(string, character.Location) {}

func TestTellNameResolution(t *testing.T) {
	service := NewWorld(nil)
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

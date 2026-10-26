package integration

import (
	"strings"
	"testing"

	"github.com/elidor/dungeogo/pkg/game/character"
)

func assertNoLocalMessage(t *testing.T, s *locationSession, fragments ...string) {
	t.Helper()
	out := roomLook(s)
	for _, fragment := range fragments {
		if strings.Contains(out, fragment) {
			t.Fatalf("unexpected local message %q: %q", fragment, out)
		}
	}
}

func TestLocalCommunication_SayEmoteMovementAndDisconnect(t *testing.T) {
	manager, p, handler := locationFixture(t)
	createPresenceCharacter(t, manager, p, "Traveler")
	actor := openLocationSession(t, handler)
	actor.login(p)
	actor.selectCharacter()
	home := roomObserver(t, manager, handler, "HomePeer", "starting_room")
	away := roomObserver(t, manager, handler, "NorthPeer", "newbie_north")
	for _, tc := range []struct{ input, sender, recipient string }{
		{"say Hello, realm! 🌿", "You say: Hello, realm! 🌿", "Traveler says: Hello, realm! 🌿"},
		{"emote waves to the room.", "You waves to the room.", "Traveler waves to the room."},
		{"' Another greeting!", "You say: Another greeting!", "Traveler says: Another greeting!"},
		{": nods politely.", "You nods politely.", "Traveler nods politely."},
	} {
		actor.send(tc.input)
		out := actor.until("> ")
		if strings.Count(out, tc.sender) != 1 || strings.Contains(out, tc.recipient) {
			t.Fatalf("sender feedback: %q", out)
		}
		home.until(tc.recipient)
		assertNoLocalMessage(t, home, tc.recipient) // No duplicate delivery.
		assertNoLocalMessage(t, away, tc.recipient)
	}
	actor.send("north")
	actor.until("> ")
	home.until("Traveler leaves north.")
	away.until("Traveler arrives.")
	actor.send("say We moved rooms.")
	if out := actor.until("> "); !strings.Contains(out, "You say: We moved rooms.") {
		t.Fatalf("moved sender: %q", out)
	}
	away.until("Traveler says: We moved rooms.")
	assertNoLocalMessage(t, home, "Traveler says: We moved rooms.")
	away.send("emote greets the traveler.")
	if out := away.until("> "); !strings.Contains(out, "You greets the traveler.") {
		t.Fatalf("emote sender: %q", out)
	}
	actor.until("NorthPeer greets the traveler.")
	assertNoLocalMessage(t, home, "NorthPeer greets the traveler.")

	actor.disconnect()
	away.send("say Still in the north.")
	if out := away.until("> "); !strings.Contains(out, "You say: Still in the north.") {
		t.Fatalf("solo feedback: %q", out)
	}
	assertNoLocalMessage(t, home, "NorthPeer says: Still in the north.")
	resumed := openLocationSession(t, handler)
	resumed.login(p)
	resumed.selectCharacter()
	away.send("say Welcome back.")
	away.until("> ")
	resumed.until("NorthPeer says: Welcome back.")
	assertNoLocalMessage(t, home, "NorthPeer says: Welcome back.")
}

func TestLocalCommunication_FailedActionsPublishNothing(t *testing.T) {
	manager, p, handler := locationFixture(t)
	id := createPresenceCharacter(t, manager, p, "Traveler")
	actor := openLocationSession(t, handler)
	actor.login(p)
	actor.selectCharacter()
	home := roomObserver(t, manager, handler, "HomePeer", "starting_room")
	away := roomObserver(t, manager, handler, "NorthPeer", "newbie_north")
	for _, input := range []string{"say", "emote"} {
		actor.send(input)
		if out := actor.until("> "); !strings.Contains(out, "Please provide a message.") {
			t.Fatalf("empty command: %q", out)
		}
		assertNoLocalMessage(t, home, "Traveler ")
		assertNoLocalMessage(t, away, "Traveler ")
	}
	for _, tc := range []struct {
		name     string
		location *character.Location
	}{
		{"missing", nil},
		{"invalid", &character.Location{RoomID: "missing_room", ZoneID: "newbie_zone"}},
		{"inconsistent live membership", &character.Location{RoomID: "newbie_north", ZoneID: "newbie_zone"}},
	} {
		if err := manager.Characters().UpdateCharacterLocation(id, tc.location); err != nil {
			t.Fatal(err)
		}
		for _, input := range []string{"say failed greeting", "emote makes a failed gesture"} {
			actor.send(input)
			out := actor.until("> ")
			if !strings.Contains(out, "You cannot communicate") {
				t.Fatalf("%s failure: %q", tc.name, out)
			}
			assertNoLocalMessage(t, home, "Traveler says: failed greeting", "Traveler makes a failed gesture")
			assertNoLocalMessage(t, away, "Traveler says: failed greeting", "Traveler makes a failed gesture")
		}
	}
	if err := manager.Characters().UpdateCharacterLocation(id, &character.Location{RoomID: "starting_room", ZoneID: "newbie_zone"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.GetDB().Exec(`ALTER TABLE room_definitions RENAME TO unavailable_rooms`); err != nil {
		t.Fatal(err)
	}
	actor.send("say database failure")
	if out := actor.until("> "); !strings.Contains(out, "You cannot communicate") {
		t.Fatalf("database failure: %q", out)
	}
	if _, err := manager.GetDB().Exec(`ALTER TABLE unavailable_rooms RENAME TO room_definitions`); err != nil {
		t.Fatal(err)
	}
	assertNoLocalMessage(t, home, "Traveler says: database failure")
	assertNoLocalMessage(t, away, "Traveler says: database failure")
	// Recovery from the injected failure restores normal delivery without phantom messages.
	actor.send("say recovered")
	actor.until("> ")
	home.until("Traveler says: recovered")
	assertNoLocalMessage(t, away, "Traveler says: recovered")
}

package integration

import (
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/server"
	"golang.org/x/crypto/bcrypt"
)

func roomObserver(t *testing.T, manager *postgres.PostgreSQLRepositoryManager, handler *server.SessionHandler, name, roomID string) *locationSession {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	p := player.NewPlayer(strings.ToLower(name), strings.ToLower(name)+"@example.com", string(hash))
	if err := manager.Players().CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	id := createPresenceCharacter(t, manager, p, name)
	if err := manager.Characters().UpdateCharacterLocation(id, &character.Location{RoomID: roomID, ZoneID: "newbie_zone"}); err != nil {
		t.Fatal(err)
	}
	s := openLocationSession(t, handler)
	s.login(p)
	s.selectCharacter()
	return s
}

func roomLook(s *locationSession) string { s.t.Helper(); s.send("look"); return s.until("> ") }

func TestRoomPresence_LookMovementScopeAndReplacement(t *testing.T) {
	manager, p, handler := locationFixture(t)
	id := createPresenceCharacter(t, manager, p, "Traveler")
	actor := openLocationSession(t, handler)
	actor.login(p)
	actor.selectCharacter()
	home := roomObserver(t, manager, handler, "HomeWitness", "starting_room")
	north := roomObserver(t, manager, handler, "NorthWitness", "newbie_north")
	east := roomObserver(t, manager, handler, "EastWitness", "newbie_east")
	out := roomLook(actor)
	if !strings.Contains(out, "Also here: HomeWitness.") || strings.Contains(out, "NorthWitness") || strings.Contains(out, "EastWitness") {
		t.Fatalf("room occupants: %q", out)
	}
	actor.send("north")
	out = actor.until("> ")
	if !strings.Contains(out, "Northern Walk") || !strings.Contains(out, "Also here: NorthWitness.") || strings.Contains(out, "HomeWitness") || strings.Contains(out, "Traveler arrives") {
		t.Fatalf("movement rendering: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")
	if out := home.until("Traveler leaves north."); strings.Contains(out, "Traveler arrives.") {
		t.Fatalf("home event: %q", out)
	}
	if out := north.until("Traveler arrives."); strings.Contains(out, "Traveler leaves") {
		t.Fatalf("destination event: %q", out)
	}
	if out := roomLook(home); strings.Contains(out, "Traveler") {
		t.Fatalf("old membership: %q", out)
	}
	if out := roomLook(north); !strings.Contains(out, "Also here: Traveler.") {
		t.Fatalf("new membership: %q", out)
	}
	if out := roomLook(east); strings.Contains(out, "Traveler") {
		t.Fatalf("unrelated room received event: %q", out)
	}

	// Invalid exits and rejected writes publish neither membership changes nor success events.
	actor.send("north")
	out = actor.until("> ")
	if !strings.Contains(out, "cannot go north") || strings.Contains(out, "You move") {
		t.Fatalf("invalid move: %q", out)
	}
	if _, err := manager.GetDB().Exec(`CREATE FUNCTION reject_room_movement() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test persistence failure'; END $$; CREATE TRIGGER reject_room_movement BEFORE UPDATE ON characters FOR EACH ROW EXECUTE FUNCTION reject_room_movement()`); err != nil {
		t.Fatal(err)
	}
	actor.send("south")
	out = actor.until("> ")
	if !strings.Contains(out, "cannot move") || strings.Contains(out, "You move") {
		t.Fatalf("failed move: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")
	for _, observer := range []*locationSession{home, north, east} {
		out := roomLook(observer)
		if strings.Contains(out, "Traveler leaves") || strings.Contains(out, "Traveler arrives") {
			t.Fatalf("failure published event: %q", out)
		}
		if observer == north && !strings.Contains(out, "Also here: Traveler.") {
			t.Fatalf("failed write changed membership: %q", out)
		}
		if observer != north && strings.Contains(out, "Also here: Traveler.") {
			t.Fatalf("failed write added membership: %q", out)
		}
	}
	if _, err := manager.GetDB().Exec(`DROP TRIGGER reject_room_movement ON characters`); err != nil {
		t.Fatal(err)
	}

	replacement := openLocationSession(t, handler)
	replacement.login(p)
	expectClosedSession(t, actor)
	if out := roomLook(north); strings.Contains(out, "Traveler") {
		t.Fatalf("replacement before selection retained membership: %q", out)
	}
	if out := replacement.selectCharacter(); !strings.Contains(out, "Also here: NorthWitness.") {
		t.Fatalf("replacement location: %q", out)
	}
	if out := roomLook(north); strings.Count(out, "Traveler") != 1 {
		t.Fatalf("stale cleanup lost or duplicated replacement: %q", out)
	}
	replacement.disconnect()
	if out := roomLook(north); strings.Contains(out, "Traveler") {
		t.Fatalf("disconnect retained membership: %q", out)
	}
}

type failingRoomWriteConn struct {
	net.Conn
	fail *atomic.Bool
}

func (c *failingRoomWriteConn) Write(p []byte) (int, error) {
	if c.fail.Load() {
		return 0, fmt.Errorf("injected delivery failure")
	}
	return c.Conn.Write(p)
}

func TestRoomPresence_DeliveryFailureDoesNotUndoCommittedMove(t *testing.T) {
	manager, p, handler := locationFixture(t)
	id := createPresenceCharacter(t, manager, p, "Traveler")
	fail := &atomic.Bool{}
	actor := openLocationSessionWithConn(t, handler, func(conn net.Conn) net.Conn { return &failingRoomWriteConn{Conn: conn, fail: fail} })
	actor.login(p)
	actor.selectCharacter()
	home := roomObserver(t, manager, handler, "HomeWitness", "starting_room")
	north := roomObserver(t, manager, handler, "NorthWitness", "newbie_north")
	fail.Store(true)
	actor.send("north")
	expectClosedSession(t, actor)
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")
	home.until("Traveler leaves north.")
	north.until("Traveler arrives.")
	resumed := openLocationSession(t, handler)
	resumed.login(p)
	if out := resumed.selectCharacter(); !strings.Contains(out, "Northern Walk") || !strings.Contains(out, "Also here: NorthWitness.") {
		t.Fatalf("reconnect after failed delivery: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")
	if out := roomLook(north); strings.Count(out, "Traveler") != 1 {
		t.Fatalf("reconnected membership: %q", out)
	}
}

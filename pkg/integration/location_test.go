package integration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game"
	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/server"
	"github.com/elidor/dungeogo/pkg/testutil"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Each session uses the real TCP handler; cleanup waits for its database work to finish.
type locationSession struct {
	t      *testing.T
	conn   net.Conn
	reader *bufio.Reader
	done   chan struct{}
}

func openLocationSession(t *testing.T, handler *server.SessionHandler) *locationSession {
	t.Helper()
	return openLocationSessionWithConn(t, handler, nil)
}

func openLocationSessionWithConn(t *testing.T, handler *server.SessionHandler, wrap func(net.Conn) net.Conn) *locationSession {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			if wrap != nil {
				conn = wrap(conn)
			}
			handler.HandleClient(server.NewClient(uuid.NewString(), conn))
		}
	}()
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		listener.Close()
		t.Fatal(err)
	}
	s := &locationSession{t: t, conn: conn, reader: bufio.NewReader(conn), done: done}
	t.Cleanup(func() { listener.Close(); s.disconnect() })
	s.until("Please enter your username:")
	return s
}

func (s *locationSession) send(input string) {
	s.t.Helper()
	if err := s.conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		s.t.Fatal(err)
	}
	if _, err := fmt.Fprintln(s.conn, input); err != nil {
		s.t.Fatal(err)
	}
}

func (s *locationSession) until(text string) string {
	s.t.Helper()
	// Usage strings contain angle brackets; only a prompt at a new line completes a command.
	if text == "> " {
		text = "\r\n> "
	}
	out, err := readUntilContains(s.conn, s.reader, text)
	if err != nil {
		s.t.Fatalf("waiting for %q: %v; output=%q", text, err, out)
	}
	return out
}

func (s *locationSession) login(p *player.Player) {
	s.t.Helper()
	s.send(p.Username)
	s.until("Please enter your password:")
	s.send("testpass123")
	s.until("Character> ")
}

func (s *locationSession) selectCharacter() string {
	s.t.Helper()
	s.send("select")
	s.until("Select #: ")
	s.send("1")
	return s.until("> ")
}

func (s *locationSession) disconnect() {
	s.t.Helper()
	s.conn.Close()
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
		s.t.Error("session did not stop after disconnect")
	}
}

func locationFixture(t *testing.T) (*postgres.PostgreSQLRepositoryManager, *player.Player, *server.SessionHandler) {
	t.Helper()
	manager := testutil.ImprovedSetupTestDB(t)
	if err := manager.Bootstrap(context.Background(), "starting_room"); err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	p := player.NewPlayer("traveler", "traveler@example.com", string(hash))
	if err := manager.Players().CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	return manager, p, server.NewSessionHandler(manager, game.NewEngine(manager))
}

func assertSavedLocation(t *testing.T, manager *postgres.PostgreSQLRepositoryManager, id, room, zone string) {
	t.Helper()
	c, err := manager.Characters().GetCharacter(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Location == nil || c.Location.RoomID != room || c.Location.ZoneID != zone {
		t.Fatalf("saved location=%+v; want %s/%s", c.Location, zone, room)
	}
}

func TestSessionLocation_CreateMoveAndReconnect(t *testing.T) {
	manager, p, handler := locationFixture(t)
	s := openLocationSession(t, handler)
	s.login(p)
	for _, step := range []struct{ input, prompt string }{
		{"create", "Name: "}, {"Traveler", "Race: "}, {"human", "Class: "}, {"warrior", "Confirm: "}, {"yes", "Character> "},
	} {
		s.send(step.input)
		s.until(step.prompt)
	}
	chars, err := manager.Characters().GetCharactersByPlayer(p.ID)
	if err != nil || len(chars) != 1 {
		t.Fatalf("created characters=%v err=%v", chars, err)
	}
	id := chars[0].ID
	assertSavedLocation(t, manager, id, "starting_room", "newbie_zone")
	if out := s.selectCharacter(); !strings.Contains(out, "A Simple Room") {
		t.Fatalf("spawn: %q", out)
	}
	// The destination definition determines its zone, including travel between zones.
	if _, err := manager.GetDB().Exec(`UPDATE room_definitions SET zone_id='north_zone' WHERE room_id='newbie_north'`); err != nil {
		t.Fatal(err)
	}
	s.send("north")
	out := s.until("> ")
	if !strings.Contains(out, "You move north.") || !strings.Contains(out, "Northern Walk") {
		t.Fatalf("movement: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "north_zone")
	s.send("north")
	out = s.until("> ")
	if !strings.Contains(out, "cannot go north") || strings.Contains(out, "You move") {
		t.Fatalf("blocked exit: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "north_zone")
	// A failed durable move must produce no success response or location change.
	if _, err := manager.GetDB().Exec(`CREATE FUNCTION reject_movement() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test write failure'; END $$; CREATE TRIGGER reject_movement BEFORE UPDATE ON characters FOR EACH ROW EXECUTE FUNCTION reject_movement()`); err != nil {
		t.Fatal(err)
	}
	s.send("south")
	out = s.until("> ")
	if !strings.Contains(out, "cannot move") || strings.Contains(out, "You move") {
		t.Fatalf("failed movement: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "north_zone")
	if _, err := manager.GetDB().Exec(`DROP TRIGGER reject_movement ON characters`); err != nil {
		t.Fatal(err)
	}
	s.disconnect() // Abrupt disconnect must preserve the already committed movement.

	// A new engine/handler has no in-memory knowledge of the previous location.
	resumed := openLocationSession(t, server.NewSessionHandler(manager, game.NewEngine(manager)))
	resumed.login(p)
	if out := resumed.selectCharacter(); !strings.Contains(out, "Northern Walk") || strings.Contains(out, "A Simple Room") {
		t.Fatalf("reconnect: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "north_zone")
	resumed.send("quit")
	resumed.until("Saving character and disconnecting...")
	resumed.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadAll(resumed.reader); err != nil {
		t.Fatalf("quit did not close connection: %v", err)
	}
	resumed.disconnect()
	assertSavedLocation(t, manager, id, "newbie_north", "north_zone")
}

func TestSessionLocation_Recovery(t *testing.T) {
	for _, tc := range []struct {
		name       string
		location   *character.Location
		room, zone string
	}{
		{"valid", &character.Location{RoomID: "newbie_west", ZoneID: "newbie_zone", X: 7, Y: 9}, "newbie_west", "newbie_zone"},
		{"missing", nil, "starting_room", "newbie_zone"},
		{"empty", &character.Location{}, "starting_room", "newbie_zone"},
		{"unknown room", &character.Location{RoomID: "deleted", ZoneID: "newbie_zone"}, "starting_room", "newbie_zone"},
		{"wrong zone", &character.Location{RoomID: "newbie_west", ZoneID: "wrong_zone"}, "starting_room", "newbie_zone"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, p, handler := locationFixture(t)
			c := testutil.CreateTestCharacter(p.ID)
			c.Location = tc.location
			if err := manager.Characters().CreateCharacter(c); err != nil {
				t.Fatal(err)
			}
			s := openLocationSession(t, handler)
			s.login(p)
			out := s.selectCharacter()
			room, err := manager.World().LoadRoomDefinition(tc.room)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "Welcome, "+c.Name) || !strings.Contains(out, room.Name) {
				t.Fatalf("entry: %q", out)
			}
			assertSavedLocation(t, manager, c.ID, tc.room, tc.zone)
			if tc.name == "valid" {
				saved, err := manager.Characters().GetCharacter(c.ID)
				if err != nil {
					t.Fatal(err)
				}
				if *saved.Location != *tc.location {
					t.Fatalf("valid location changed: %+v", saved.Location)
				}
			}
			s.disconnect()
			again := openLocationSession(t, handler)
			again.login(p)
			again.selectCharacter()
			assertSavedLocation(t, manager, c.ID, tc.room, tc.zone)
		})
	}
}

func TestSessionLocation_ConfiguredSpawn(t *testing.T) {
	manager, p, handler := locationFixture(t)
	if err := manager.World().SaveRoomDefinition(&interfaces.RoomDefinition{ID: "custom_spawn", ZoneID: "custom_zone", Name: "Custom Spawn", Description: "A different beginning.", Exits: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if err := manager.ValidateWorld(context.Background(), "custom_spawn"); err != nil {
		t.Fatal(err)
	}
	handler.SetStartingRoom("custom_spawn")
	s := openLocationSession(t, handler)
	s.login(p)
	for _, step := range []struct{ input, prompt string }{
		{"create", "Name: "}, {"CustomTraveler", "Race: "}, {"elf", "Class: "}, {"mage", "Confirm: "}, {"yes", "Character> "},
	} {
		s.send(step.input)
		s.until(step.prompt)
	}
	chars, err := manager.Characters().GetCharactersByPlayer(p.ID)
	if err != nil || len(chars) != 1 {
		t.Fatalf("characters=%v err=%v", chars, err)
	}
	id := chars[0].ID
	assertSavedLocation(t, manager, id, "custom_spawn", "custom_zone")
	if out := s.selectCharacter(); !strings.Contains(out, "Custom Spawn") {
		t.Fatalf("configured spawn: %q", out)
	}
	s.disconnect()
	if err := manager.Characters().UpdateCharacterLocation(id, nil); err != nil {
		t.Fatal(err)
	}
	again := openLocationSession(t, handler)
	again.login(p)
	if out := again.selectCharacter(); !strings.Contains(out, "Custom Spawn") {
		t.Fatalf("configured recovery: %q", out)
	}
	assertSavedLocation(t, manager, id, "custom_spawn", "custom_zone")
}

func TestSessionLocation_StorageFailuresPreventEntry(t *testing.T) {
	for _, failure := range []string{"read", "recovery write"} {
		t.Run(failure, func(t *testing.T) {
			manager, p, handler := locationFixture(t)
			c := testutil.CreateTestCharacter(p.ID)
			if failure == "recovery write" {
				c.Location = &character.Location{RoomID: "deleted", ZoneID: "newbie_zone"}
			}
			if err := manager.Characters().CreateCharacter(c); err != nil {
				t.Fatal(err)
			}
			s := openLocationSession(t, handler)
			s.login(p)
			if failure == "read" {
				if _, err := manager.GetDB().Exec(`ALTER TABLE room_definitions RENAME TO unavailable_rooms`); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := manager.GetDB().Exec(`CREATE FUNCTION reject_location_update() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test write failure'; END $$; CREATE TRIGGER reject_location BEFORE UPDATE ON characters FOR EACH ROW EXECUTE FUNCTION reject_location_update()`); err != nil {
					t.Fatal(err)
				}
			}
			s.send("select")
			s.until("Select #: ")
			s.send("1")
			out := s.until("Character> ")
			if !strings.Contains(out, "Error restoring character location") || strings.Contains(out, "You enter the game world") {
				t.Fatalf("failed entry: %q", out)
			}
			assertSavedLocation(t, manager, c.ID, c.Location.RoomID, c.Location.ZoneID)
		})
	}
}

package integration

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

func createPresenceCharacter(t *testing.T, m *postgres.PostgreSQLRepositoryManager, p *player.Player, name string) string {
	t.Helper()
	c := testutil.CreateTestCharacter(p.ID)
	c.Name = name
	if err := m.Characters().CreateCharacter(c); err != nil {
		t.Fatal(err)
	}
	return c.ID
}

func whoOutput(s *locationSession) string { s.t.Helper(); s.send("who"); return s.until("> ") }

func expectClosedSession(t *testing.T, s *locationSession) {
	t.Helper()
	s.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadAll(s.reader); err != nil {
		t.Fatalf("expected closed replaced connection: %v", err)
	}
	select {
	case <-s.done:
	case <-time.After(time.Second):
		t.Fatal("old handler did not clean up")
	}
}

func TestPresenceSession_AuthenticationReplacementAndWho(t *testing.T) {
	manager, p, handler := locationFixture(t)
	id := createPresenceCharacter(t, manager, p, "Traveler")
	first := openLocationSession(t, handler)
	first.login(p)
	first.selectCharacter()
	out := whoOutput(first)
	if !strings.Contains(out, "Traveler (Human Warrior, Level 1)") || !strings.Contains(out, "1 player online.") || strings.Contains(out, "TestPlayer") {
		t.Fatalf("who: %q", out)
	}

	// Failed authentication must not change the live account or selected character.
	failed := openLocationSession(t, handler)
	failed.send(p.Username)
	failed.until("Please enter your password:")
	failed.send("wrong password")
	failed.until("Invalid password.")
	expectClosedSession(t, failed)
	if out := whoOutput(first); !strings.Contains(out, "Traveler") {
		t.Fatalf("failed auth removed active character: %q", out)
	}
	first.send("north")
	first.until("> ")
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")

	// Successful authentication revokes the old connection before character selection.
	replacement := openLocationSession(t, handler)
	replacement.login(p)
	expectClosedSession(t, first)
	// Even buffered commands from an obsolete socket must not move the character.
	first.conn.SetWriteDeadline(time.Now().Add(time.Second))
	first.conn.Write([]byte("south\n"))
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")

	// A second account observes that the replaced character is already offline.
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	observerPlayer := player.NewPlayer("observer", "observer@example.com", string(hash))
	if err := manager.Players().CreatePlayer(observerPlayer); err != nil {
		t.Fatal(err)
	}
	createPresenceCharacter(t, manager, observerPlayer, "Observer")
	observer := openLocationSession(t, handler)
	observer.login(observerPlayer)
	observer.selectCharacter()
	if out := whoOutput(observer); strings.Contains(out, "Traveler") || !strings.Contains(out, "1 player online.") {
		t.Fatalf("pre-selection presence: %q", out)
	}
	replacement.selectCharacter()
	out = whoOutput(observer)
	if !strings.Contains(out, "Traveler") || !strings.Contains(out, "Observer") || !strings.Contains(out, "2 players online.") {
		t.Fatalf("selected presence after stale cleanup: %q", out)
	}
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")
	replacement.disconnect()
	if out := whoOutput(observer); strings.Contains(out, "Traveler") || !strings.Contains(out, "1 player online.") {
		t.Fatalf("disconnect presence: %q", out)
	}
}

func TestPresenceSession_NewAccountIsRegistered(t *testing.T) {
	_, _, handler := locationFixture(t)
	s := openLocationSession(t, handler)
	s.send("newaccount")
	s.until("Email: ")
	s.send("newaccount@example.com")
	s.until("Please choose a password")
	s.send("testpass123")
	s.until("Please confirm your password:")
	s.send("testpass123")
	s.until("Character> ")
	for _, step := range []struct{ input, prompt string }{
		{"create", "Name: "}, {"NewHero", "Race: "}, {"dwarf", "Class: "}, {"rogue", "Confirm: "}, {"yes", "Character> "},
	} {
		s.send(step.input)
		s.until(step.prompt)
	}
	s.selectCharacter()
	if out := whoOutput(s); !strings.Contains(out, "NewHero (Dwarf Rogue, Level 1)") {
		t.Fatalf("new-account presence: %q", out)
	}
}

func TestPresenceSession_ConcurrentAuthenticationHasOneActiveConnection(t *testing.T) {
	manager, p, handler := locationFixture(t)
	id := createPresenceCharacter(t, manager, p, "ConcurrentHero")
	first := openLocationSession(t, handler)
	second := openLocationSession(t, handler)
	for _, s := range []*locationSession{first, second} {
		s.send(p.Username)
		s.until("Please enter your password:")
	}
	// Both handlers verify the same account's password concurrently.
	first.send("testpass123")
	second.send("testpass123")
	for _, s := range []*locationSession{first, second} {
		out, err := readUntilContains(s.conn, s.reader, "Character> ")
		if err != nil && err != io.EOF {
			t.Fatalf("login: %v output=%q", err, out)
		}
	}
	var winner, loser *locationSession
	select {
	case <-first.done:
		winner, loser = second, first
	case <-second.done:
		winner, loser = first, second
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent authentication left both handlers active")
	}
	winner.selectCharacter()
	if out := whoOutput(winner); strings.Count(out, "ConcurrentHero (") != 1 || !strings.Contains(out, "1 player online.") {
		t.Fatalf("concurrent who: %q", out)
	}
	loser.conn.Write([]byte("north\n"))
	assertSavedLocation(t, manager, id, "starting_room", "newbie_zone")
	winner.send("north")
	winner.until("> ")
	assertSavedLocation(t, manager, id, "newbie_north", "newbie_zone")
}

package integration

import (
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game"
	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/game/world"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/server"
	"golang.org/x/crypto/bcrypt"
)

func privateTarget(t *testing.T, m *postgres.PostgreSQLRepositoryManager, h *server.SessionHandler) (*player.Player, *locationSession) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	p := player.NewPlayer("receiveraccount", "receiver@example.com", string(hash))
	if err := m.Players().CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	id := createPresenceCharacter(t, m, p, "Moonbeam")
	if err := m.Characters().UpdateCharacterLocation(id, &character.Location{RoomID: "newbie_north", ZoneID: "newbie_zone"}); err != nil {
		t.Fatal(err)
	}
	s := openLocationSession(t, h)
	s.login(p)
	s.selectCharacter()
	return p, s
}

func TestPrivateTell_DeliveryIdentityIsolationAndDisconnect(t *testing.T) {
	m, p, h := locationFixture(t)
	createPresenceCharacter(t, m, p, "Traveler")
	actor := openLocationSession(t, h)
	actor.login(p)
	actor.selectCharacter()
	targetPlayer, target := privateTarget(t, m, h)
	bystander := roomObserver(t, m, h, "Bystander", "starting_room")
	actor.send("tell mOoNbEaM Private hello! 🌙")
	out := actor.until("> ")
	if !strings.Contains(out, "You tell Moonbeam: Private hello! 🌙") || strings.Contains(out, "tells you:") {
		t.Fatalf("sender confirmation: %q", out)
	}
	target.until("Traveler tells you: Private hello! 🌙")
	assertNoLocalMessage(t, bystander, "Private hello!", "You tell Moonbeam")
	target.send("t Traveler A private reply.")
	out = target.until("> ")
	if !strings.Contains(out, "You tell Traveler: A private reply.") {
		t.Fatalf("reply confirmation: %q", out)
	}
	actor.until("Moonbeam tells you: A private reply.")
	assertNoLocalMessage(t, bystander, "A private reply.")
	createPresenceCharacter(t, m, p, "Sleeper") // Persisted, but never selected online.
	for _, input := range []string{"tell receiveraccount secret", "tell Moon secret", "tell Nobody secret", "tell Sleeper secret", "tell", "tell Moonbeam"} {
		actor.send(input)
		out = actor.until("> ")
		if strings.Contains(out, "You tell ") || (!strings.Contains(out, "not available") && !strings.Contains(out, "Usage:")) {
			t.Fatalf("unavailable/invalid target %q: %q", input, out)
		}
		assertNoLocalMessage(t, target, "secret", "Traveler tells you:")
		assertNoLocalMessage(t, bystander, "secret", "Traveler tells you:")
	}
	actor.send("tell Traveler Only me.")
	out = actor.until("> ")
	if !strings.Contains(out, "Traveler tells you: Only me.") || !strings.Contains(out, "You tell Traveler: Only me.") {
		t.Fatalf("self tell: %q", out)
	}
	assertNoLocalMessage(t, bystander, "Only me.")
	assertNoLocalMessage(t, target, "Only me.")
	target.disconnect()
	actor.send("tell Moonbeam No offline queue.")
	out = actor.until("> ")
	if !strings.Contains(out, "not available") || strings.Contains(out, "You tell") {
		t.Fatalf("offline target: %q", out)
	}
	resumed := openLocationSession(t, h)
	resumed.login(targetPlayer)
	resumed.selectCharacter()
	assertNoLocalMessage(t, resumed, "No offline queue.")
	actor.send("tell Moonbeam Welcome back.")
	actor.until("> ")
	resumed.until("Traveler tells you: Welcome back.")
	assertNoLocalMessage(t, bystander, "Welcome back.", "No offline queue.")
}

type pausedTellEngine struct {
	*game.Engine
	resolved chan struct{}
	release  chan struct{}
}

func (e *pausedTellEngine) ProcessAction(id, input string) (world.Result, error) {
	result, err := e.Engine.ProcessAction(id, input)
	if strings.HasPrefix(input, "tell ") {
		close(e.resolved)
		<-e.release
	}
	return result, err
}

func TestPrivateTell_RecipientChangesBetweenResolutionAndDelivery(t *testing.T) {
	for _, mode := range []string{"disconnect", "replacement same character", "replacement different character"} {
		t.Run(mode, func(t *testing.T) {
			m, p, _ := locationFixture(t)
			engine := &pausedTellEngine{Engine: game.NewEngine(m), resolved: make(chan struct{}), release: make(chan struct{})}
			// Always release the sender before session cleanup, including on assertion failure.
			released := false
			h := server.NewSessionHandler(m, engine)
			createPresenceCharacter(t, m, p, "Traveler")
			actor := openLocationSession(t, h)
			actor.login(p)
			actor.selectCharacter()
			targetPlayer, target := privateTarget(t, m, h)
			bystander := roomObserver(t, m, h, "Bystander", "starting_room")
			t.Cleanup(func() {
				if !released {
					close(engine.release)
				}
			})
			actor.send("tell Moonbeam private race payload")
			select {
			case <-engine.resolved:
			case <-time.After(3 * time.Second):
				t.Fatal("tell did not reach delivery boundary")
			}
			var replacement *locationSession
			if mode == "disconnect" {
				target.disconnect()
			} else {
				if mode == "replacement different character" {
					createPresenceCharacter(t, m, targetPlayer, "Alternative")
				}
				replacement = openLocationSession(t, h)
				replacement.login(targetPlayer)
				expectClosedSession(t, target)
				replacement.selectCharacter()
			}
			close(engine.release)
			released = true
			out := actor.until("> ")
			if !strings.Contains(out, "not available") || strings.Contains(out, "You tell") {
				t.Fatalf("stale recipient confirmation: %q", out)
			}
			if replacement == nil {
				replacement = openLocationSession(t, h)
				replacement.login(targetPlayer)
				replacement.selectCharacter()
			}
			assertNoLocalMessage(t, replacement, "private race payload")
			assertNoLocalMessage(t, bystander, "private race payload")
		})
	}
}

func TestPrivateTell_FailedSocketWriteReportsFailure(t *testing.T) {
	m, p, h := locationFixture(t)
	createPresenceCharacter(t, m, p, "Traveler")
	actor := openLocationSession(t, h)
	actor.login(p)
	actor.selectCharacter()
	targetPlayer, target := privateTarget(t, m, h)
	target.disconnect()
	fail := &atomic.Bool{}
	target = openLocationSessionWithConn(t, h, func(conn net.Conn) net.Conn { return &failingRoomWriteConn{Conn: conn, fail: fail} })
	target.login(targetPlayer)
	target.selectCharacter()
	bystander := roomObserver(t, m, h, "Bystander", "starting_room")
	fail.Store(true)
	actor.send("tell Moonbeam undeliverable private content")
	out := actor.until("> ")
	if !strings.Contains(out, "not available") || strings.Contains(out, "You tell") {
		t.Fatalf("failed delivery acknowledged: %q", out)
	}
	expectClosedSession(t, target)
	assertNoLocalMessage(t, bystander, "undeliverable private content")
}

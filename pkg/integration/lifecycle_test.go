package integration

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/server"
	"golang.org/x/crypto/bcrypt"
)

// dialManaged connects to a ConnectionManager-served session, unlike
// openLocationSession, which hands a socket straight to the handler.
func dialManaged(t *testing.T, cm *server.ConnectionManager) *locationSession {
	t.Helper()
	conn, err := net.DialTimeout("tcp", cm.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	close(done) // the manager owns the handler goroutine; Shutdown waits for it
	s := &locationSession{t: t, conn: conn, reader: bufio.NewReader(conn), done: done}
	t.Cleanup(func() { conn.Close() })
	s.until("Please enter your username:")
	return s
}

func TestLifecycle_IdleExpiryRemovesPresenceAndShutdownWaits(t *testing.T) {
	w := newGiveWorld(t)
	cm := server.NewConnectionManager(10, 300*time.Millisecond)
	cm.SetHandler(w.handler)
	served := make(chan error, 1)
	go func() { served <- cm.Start("127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for cm.Addr() == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	login := func(name string) *locationSession {
		t.Helper()
		hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		p := player.NewPlayer(strings.ToLower(name), strings.ToLower(name)+"@example.com", string(hash))
		if err := w.manager.Players().CreatePlayer(p); err != nil {
			t.Fatal(err)
		}
		id := createPresenceCharacter(t, w.manager, p, name)
		if err := w.manager.Characters().UpdateCharacterLocation(id, &character.Location{RoomID: "starting_room", ZoneID: "newbie_zone"}); err != nil {
			t.Fatal(err)
		}
		s := dialManaged(t, cm)
		s.login(p)
		s.selectCharacter()
		return s
	}
	idle := login("Sleepy")
	awake := login("Awake")
	if out := command(awake, "who"); !strings.Contains(out, "Sleepy") || !strings.Contains(out, "Awake") {
		t.Fatalf("who before idle expiry: %q", out)
	}

	// Awake keeps talking; Sleepy says nothing and is expired by the sweep.
	expired := time.Now().Add(3 * time.Second)
	for {
		out := command(awake, "who")
		if !strings.Contains(out, "Sleepy") {
			if !strings.Contains(out, "Awake") {
				t.Fatalf("idle expiry removed the active session too: %q", out)
			}
			break
		}
		if time.Now().After(expired) {
			t.Fatalf("idle session still online after 3s: %q", out)
		}
		time.Sleep(50 * time.Millisecond)
	}
	idle.conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := idle.reader.ReadString('\n'); err == nil {
		// Drain anything already buffered, then expect the close.
		for err == nil {
			_, err = idle.reader.ReadString('\n')
		}
	}

	// Shutdown closes the remaining session, waits for it, and stops serving.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cm.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Start after shutdown: %v", err)
	}
	awake.conn.SetReadDeadline(time.Now().Add(time.Second))
	for {
		if _, err := awake.reader.ReadString('\n'); err != nil {
			break
		}
	}
	if n := cm.GetStats().TotalClients; n != 0 {
		t.Fatalf("%d clients registered after shutdown", n)
	}
}

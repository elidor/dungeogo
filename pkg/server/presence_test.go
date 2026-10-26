package server

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/game/character"
)

func registryClient(t *testing.T, id string) *Client {
	t.Helper()
	conn, peer := net.Pipe()
	c := NewClient(id, conn)
	c.SetPlayerID("account")
	t.Cleanup(func() { c.Close(); peer.Close() })
	return c
}

func TestPresenceReplacementRejectsStaleActionsAndCleanup(t *testing.T) {
	r := NewSessionRegistry()
	old := registryClient(t, "old")
	r.Register(old)
	if !r.WithSession(old, func() { r.Select(old, &character.Character{ID: "hero", Name: "Hero"}); old.SetState(StateInGame) }) {
		t.Fatal("authenticated action rejected")
	}
	if len(r.Characters()) != 1 {
		t.Fatal("selected character missing")
	}
	replacement := registryClient(t, "new")
	r.Register(replacement)
	if old.IsConnected() {
		t.Fatal("replaced connection is still live")
	}
	if len(r.Characters()) != 0 {
		t.Fatal("old character remains online before replacement selects")
	}
	called := false
	if r.WithSession(old, func() { called = true }) || called {
		t.Fatal("stale session performed an action")
	}
	r.WithSession(replacement, func() {
		r.Select(replacement, &character.Character{ID: "next", Name: "Next"})
		replacement.SetState(StateInGame)
	})
	r.Remove(old)
	online := r.Characters()
	if len(online) != 1 || online[0].ID != "next" {
		t.Fatalf("stale cleanup removed replacement: %v", online)
	}
	online[0].Name = "modified snapshot"
	if r.Characters()[0].Name != "Next" {
		t.Fatal("snapshot aliases registry state")
	}
	replacement.Close()
	if len(r.Characters()) != 0 {
		t.Fatal("disconnected character remains visible")
	}
	r.Remove(replacement)
	if r.WithSession(replacement, func() { called = true }) {
		t.Fatal("removed session authorized")
	}
}

func TestPresenceConcurrentLoginHasOneWinner(t *testing.T) {
	r := NewSessionRegistry()
	clients := make([]*Client, 20)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range clients {
		clients[i] = registryClient(t, string(rune('a'+i)))
		wg.Add(1)
		go func(c *Client) { defer wg.Done(); <-start; r.Register(c) }(clients[i])
	}
	close(start)
	wg.Wait()
	winners := 0
	for _, c := range clients {
		if r.WithSession(c, func() {}) {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("authorized sessions=%d; want 1", winners)
	}
	for _, c := range clients {
		if !c.IsConnected() {
			r.Remove(c)
		}
	}
	winners = 0
	for _, c := range clients {
		if r.WithSession(c, func() {}) {
			winners++
		}
	}
	if winners != 1 {
		t.Fatal("old-session cleanup removed winner")
	}
}

func TestPresenceReplacementSerializesWithInFlightAction(t *testing.T) {
	r := NewSessionRegistry()
	old := registryClient(t, "old")
	r.Register(old)
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() { r.WithSession(old, func() { close(entered); <-release }); close(finished) }()
	<-entered
	replacement := registryClient(t, "new")
	registered := make(chan struct{})
	go func() { r.Register(replacement); close(registered) }()
	select {
	case <-registered:
		t.Fatal("replacement overtook an authorized action")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-finished
	select {
	case <-registered:
	case <-time.After(time.Second):
		t.Fatal("replacement blocked after action finished")
	}
	if r.WithSession(old, func() { t.Error("old action ran after replacement") }) {
		t.Fatal("old session still authorized")
	}
}

type observedWriteConn struct {
	net.Conn
	started chan struct{}
}

func (c *observedWriteConn) Write(p []byte) (int, error) { close(c.started); return c.Conn.Write(p) }

func TestCloseInterruptsStalledWriter(t *testing.T) {
	conn, peer := net.Pipe()
	defer peer.Close()
	observed := &observedWriteConn{Conn: conn, started: make(chan struct{})}
	c := NewClient("stalled", observed)
	written := make(chan error, 1)
	go func() { written <- c.Send("message") }()
	<-observed.started
	closed := make(chan struct{})
	go func() { c.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close waited for stalled socket write")
	}
	select {
	case err := <-written:
		if err == nil {
			t.Fatal("stalled write unexpectedly succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("writer did not unblock")
	}
}

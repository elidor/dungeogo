package server

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// readingHandler holds each session open, reading until its connection closes,
// which is how the real session loop behaves.
type readingHandler struct {
	started, finished atomic.Int32
	linger            time.Duration // extra time a handler takes after its read fails
}

func (h *readingHandler) HandleClient(c *Client) {
	h.started.Add(1)
	defer h.finished.Add(1)
	for {
		if _, err := c.ReadLine(); err != nil {
			time.Sleep(h.linger) // e.g. a command finishing its durable write
			return
		}
	}
}

func startManager(t *testing.T, handler ClientHandler, idle time.Duration) (*ConnectionManager, chan error) {
	t.Helper()
	cm := NewConnectionManager(10, idle)
	cm.SetHandler(handler)
	served := make(chan error, 1)
	go func() { served <- cm.Start("127.0.0.1:0") }()
	eventually(t, "server to listen", func() bool { return cm.Addr() != nil })
	t.Cleanup(func() { cm.Stop() })
	return cm, served
}

func dial(t *testing.T, cm *ConnectionManager) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", cm.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestShutdownStopsAcceptingClosesSessionsAndWaits(t *testing.T) {
	handler := &readingHandler{linger: 50 * time.Millisecond}
	cm, served := startManager(t, handler, time.Minute)
	for i := 0; i < 3; i++ {
		dial(t, cm)
	}
	eventually(t, "three sessions", func() bool { return handler.started.Load() == 3 })
	addr := cm.Addr().String()

	if err := cm.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	// Shutdown returned only after every handler, including its linger, finished.
	if got := handler.finished.Load(); got != 3 {
		t.Fatalf("%d of 3 handlers finished before Shutdown returned", got)
	}
	if n := cm.GetStats().TotalClients; n != 0 {
		t.Fatalf("%d clients still registered after shutdown", n)
	}
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("Start after shutdown = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return after shutdown")
	}
	if conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
		conn.Close()
		t.Fatal("server still accepts connections after shutdown")
	}
	if err := cm.Shutdown(context.Background()); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}

func TestShutdownReportsSessionsThatOverrunTheDeadline(t *testing.T) {
	handler := &readingHandler{linger: 2 * time.Second}
	cm, _ := startManager(t, handler, time.Minute)
	dial(t, cm)
	eventually(t, "session", func() bool { return handler.started.Load() == 1 })
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := cm.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "1 session(s) still running") {
		t.Fatalf("shutdown error = %v, want deadline exceeded with 1 session", err)
	}
}

func TestDisconnectedSessionIsDeregisteredImmediately(t *testing.T) {
	handler := &readingHandler{}
	cm, _ := startManager(t, handler, time.Minute) // sweep interval is 30s
	conn := dial(t, cm)
	eventually(t, "session", func() bool { return cm.GetStats().TotalClients == 1 })
	conn.Close()
	eventually(t, "deregistration without waiting for a sweep", func() bool { return cm.GetStats().TotalClients == 0 })
}

func TestIdleSessionIsClosedAndDeregistered(t *testing.T) {
	handler := &readingHandler{}
	cm, _ := startManager(t, handler, 100*time.Millisecond)
	idle := dial(t, cm)
	active := dial(t, cm)
	eventually(t, "two sessions", func() bool { return cm.GetStats().TotalClients == 2 })
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // keep one session active
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
				active.Write([]byte("look\r\n"))
			}
		}
	}()
	defer func() { close(stop); wg.Wait() }()

	eventually(t, "idle session closed", func() bool { return handler.finished.Load() == 1 })
	idle.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := idle.Read(make([]byte, 1)); err == nil {
		t.Fatal("idle connection still open")
	}
	eventually(t, "only the active session left", func() bool { return cm.GetStats().TotalClients == 1 })
}

func TestServerFullTurnsAwayExtraConnections(t *testing.T) {
	handler := &readingHandler{}
	cm := NewConnectionManager(1, time.Minute)
	cm.SetHandler(handler)
	go cm.Start("127.0.0.1:0")
	eventually(t, "server to listen", func() bool { return cm.Addr() != nil })
	t.Cleanup(func() { cm.Stop() })
	dial(t, cm)
	eventually(t, "first session", func() bool { return handler.started.Load() == 1 })
	extra := dial(t, cm)
	extra.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 64)
	n, _ := extra.Read(buf)
	if !strings.Contains(string(buf[:n]), "Server is full") {
		t.Fatalf("extra connection got %q", buf[:n])
	}
	if handler.started.Load() != 1 {
		t.Fatal("handler ran for a connection over the limit")
	}
}

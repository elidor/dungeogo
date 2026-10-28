package main

import (
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/elidor/dungeogo/pkg/server"
)

type idleHandler struct{}

func (idleHandler) HandleClient(c *server.Client) {
	for {
		if _, err := c.ReadLine(); err != nil {
			return
		}
	}
}

type fakeRepositories struct {
	closed bool
	err    error
}

func (r *fakeRepositories) Close() error { r.closed = true; return r.err }

// runUntilSignalled serves on a free port with one open session, sends this
// process SIGTERM, and returns serve's exit code.
func runUntilSignalled(t *testing.T, repos *fakeRepositories) int {
	t.Helper()
	cm := server.NewConnectionManager(10, time.Minute)
	cm.SetHandler(idleHandler{})
	code := make(chan int, 1)
	go func() { code <- serve(cm, "127.0.0.1:0", repos) }()
	deadline := time.Now().Add(3 * time.Second)
	for cm.Addr() == nil {
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	conn, err := net.Dial("tcp", cm.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-code:
		return c
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after SIGTERM")
		return -1
	}
}

func TestServeShutsDownInOrderOnSignal(t *testing.T) {
	repos := &fakeRepositories{}
	if code := runUntilSignalled(t, repos); code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !repos.closed {
		t.Fatal("database was not closed")
	}
}

func TestServeReportsFailureToCloseDatabase(t *testing.T) {
	repos := &fakeRepositories{err: errors.New("connection reset")}
	if code := runUntilSignalled(t, repos); code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
}

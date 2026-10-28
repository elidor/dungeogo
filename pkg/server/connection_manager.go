package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ConnectionManager struct {
	clients       map[string]*Client
	playerClients map[string]*Client // playerID -> client mapping
	mutex         sync.RWMutex       // guards clients, playerClients, listener
	listener      net.Listener
	handler       ClientHandler
	maxClients    int
	idleTimeout   time.Duration

	// closing is closed once, when shutdown begins. After that no connection is
	// admitted; sessions counts admitted connections whose handler is still running.
	closing   chan struct{}
	closeOnce sync.Once
	sessions  sync.WaitGroup
}

type ClientHandler interface {
	HandleClient(client *Client)
}

func NewConnectionManager(maxClients int, idleTimeout time.Duration) *ConnectionManager {
	return &ConnectionManager{
		clients:       make(map[string]*Client),
		playerClients: make(map[string]*Client),
		maxClients:    maxClients,
		idleTimeout:   idleTimeout,
		closing:       make(chan struct{}),
	}
}

func (cm *ConnectionManager) SetHandler(handler ClientHandler) {
	cm.handler = handler
}

func (cm *ConnectionManager) shuttingDown() bool {
	select {
	case <-cm.closing:
		return true
	default:
		return false
	}
}

// Addr is the address the server is listening on, or nil before Start binds.
func (cm *ConnectionManager) Addr() net.Addr {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	if cm.listener == nil {
		return nil
	}
	return cm.listener.Addr()
}

// Start listens on address and serves connections until Shutdown. It returns
// nil once a shutdown has stopped it, or the error that prevented serving.
func (cm *ConnectionManager) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	cm.mutex.Lock()
	if cm.shuttingDown() {
		cm.mutex.Unlock()
		listener.Close()
		return nil
	}
	cm.listener = listener
	cm.mutex.Unlock()

	go cm.cleanupClients()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if cm.shuttingDown() {
				return nil
			}
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		cm.admit(conn)
	}
}

// admit registers a connection and runs its handler, unless the server is full
// or shutting down. Registration and the session count change under the same
// lock Shutdown takes, so Shutdown never misses a session it must wait for.
func (cm *ConnectionManager) admit(conn net.Conn) {
	cm.mutex.Lock()
	if cm.shuttingDown() {
		cm.mutex.Unlock()
		conn.Close()
		return
	}
	if len(cm.clients) >= cm.maxClients {
		cm.mutex.Unlock()
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		conn.Write([]byte("Server is full. Please try again later.\r\n"))
		conn.Close()
		return
	}
	client := NewClient(uuid.New().String(), conn)
	cm.clients[client.ID] = client
	cm.sessions.Add(1)
	cm.mutex.Unlock()

	fmt.Printf("New client connected: %s from %s\n", client.ID, conn.RemoteAddr())
	go func() {
		defer cm.sessions.Done()
		defer cm.RemoveClient(client.ID)
		cm.handler.HandleClient(client)
	}()
}

// Shutdown stops accepting connections, closes every session, and waits for
// their handlers to finish. A command already running completes its durable
// work first; its replies simply can't be delivered. If ctx ends first,
// Shutdown returns an error naming how many sessions are still running.
func (cm *ConnectionManager) Shutdown(ctx context.Context) error {
	cm.mutex.Lock()
	cm.closeOnce.Do(func() { close(cm.closing) })
	listener := cm.listener
	clients := make([]*Client, 0, len(cm.clients))
	for _, client := range cm.clients {
		clients = append(clients, client)
	}
	cm.mutex.Unlock()

	var listenerErr error
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			listenerErr = fmt.Errorf("close listener: %w", err)
		}
	}
	for _, client := range clients {
		client.Close()
	}

	finished := make(chan struct{})
	go func() { cm.sessions.Wait(); close(finished) }()
	select {
	case <-finished:
		return listenerErr
	case <-ctx.Done():
		return errors.Join(listenerErr, fmt.Errorf("shutdown: %d session(s) still running: %w", cm.getClientCount(), ctx.Err()))
	}
}

// Stop shuts down, allowing sessions up to ten seconds to finish.
func (cm *ConnectionManager) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return cm.Shutdown(ctx)
}

func (cm *ConnectionManager) RemoveClient(clientID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	client, exists := cm.clients[clientID]
	if !exists {
		return
	}

	// Remove from player mapping if exists
	if client.GetPlayerID() != "" {
		delete(cm.playerClients, client.GetPlayerID())
	}

	// Close and remove client
	client.Close()
	delete(cm.clients, clientID)

	fmt.Printf("Client disconnected: %s\n", clientID)
}

func (cm *ConnectionManager) GetClient(clientID string) (*Client, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	client, exists := cm.clients[clientID]
	return client, exists
}

func (cm *ConnectionManager) GetPlayerClient(playerID string) (*Client, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	client, exists := cm.playerClients[playerID]
	return client, exists
}

func (cm *ConnectionManager) RegisterPlayerClient(playerID string, client *Client) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Remove any existing mapping for this player
	if existingClient, exists := cm.playerClients[playerID]; exists {
		existingClient.Close()
	}

	cm.playerClients[playerID] = client
	client.SetPlayerID(playerID)
}

func (cm *ConnectionManager) UnregisterPlayerClient(playerID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.playerClients, playerID)
}

func (cm *ConnectionManager) BroadcastToAll(message string) {
	cm.mutex.RLock()
	clients := make([]*Client, 0, len(cm.clients))
	for _, client := range cm.clients {
		if client.IsConnected() {
			clients = append(clients, client)
		}
	}
	cm.mutex.RUnlock()

	for _, client := range clients {
		client.Send(message)
	}
}

func (cm *ConnectionManager) BroadcastToRoom(roomID, message string) {
	if scoped, ok := cm.handler.(interface{ BroadcastToRoom(string, string) }); ok {
		scoped.BroadcastToRoom(roomID, message)
	}
}

func (cm *ConnectionManager) getClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.clients)
}

func (cm *ConnectionManager) GetStats() ConnectionStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := ConnectionStats{
		TotalClients:         len(cm.clients),
		AuthenticatedClients: 0,
		InGameClients:        0,
	}

	for _, client := range cm.clients {
		switch client.GetState() {
		case StateCharacterSelection, StateCharacterCreation, StateCharacterSelecting, StateCharacterDeleting, StateInGame:
			stats.AuthenticatedClients++
			if client.GetState() == StateInGame {
				stats.InGameClients++
			}
		}
	}

	return stats
}

// cleanupClients closes idle and dead sessions until shutdown. It sweeps at
// half the idle timeout, at most every 30 seconds, so a session is closed no
// later than about 1.5x the idle timeout after its last input.
func (cm *ConnectionManager) cleanupClients() {
	interval := cm.idleTimeout / 2
	if interval <= 0 || interval > 30*time.Second {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-cm.closing:
			return
		case <-ticker.C:
			cm.performCleanup()
		}
	}
}

func (cm *ConnectionManager) performCleanup() {
	cm.mutex.RLock()
	toRemove := make([]string, 0)
	for clientID, client := range cm.clients {
		if !client.IsConnected() || client.IsIdle(cm.idleTimeout) {
			toRemove = append(toRemove, clientID)
		}
	}
	cm.mutex.RUnlock()

	for _, clientID := range toRemove {
		cm.RemoveClient(clientID)
	}
}

type ConnectionStats struct {
	TotalClients         int
	AuthenticatedClients int
	InGameClients        int
}

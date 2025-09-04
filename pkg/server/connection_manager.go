package server

import (
	"fmt"
	"net"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

type ConnectionManager struct {
	clients       map[string]*Client
	playerClients map[string]*Client // playerID -> client mapping
	mutex         sync.RWMutex
	listener      net.Listener
	handler       ClientHandler
	running       bool
	maxClients    int
	idleTimeout   time.Duration
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
	}
}

func (cm *ConnectionManager) SetHandler(handler ClientHandler) {
	cm.handler = handler
}

func (cm *ConnectionManager) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	
	cm.listener = listener
	cm.running = true
	
	// Start cleanup goroutine
	go cm.cleanupClients()
	
	// Accept connections
	for cm.running {
		conn, err := listener.Accept()
		if err != nil {
			if !cm.running {
				break // Server is shutting down
			}
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		
		if cm.getClientCount() >= cm.maxClients {
			conn.Write([]byte("Server is full. Please try again later.\r\n"))
			conn.Close()
			continue
		}
		
		client := cm.createClient(conn)
		go cm.handler.HandleClient(client)
	}
	
	return nil
}

func (cm *ConnectionManager) Stop() error {
	cm.running = false
	
	if cm.listener != nil {
		cm.listener.Close()
	}
	
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	for _, client := range cm.clients {
		client.Close()
	}
	
	return nil
}

func (cm *ConnectionManager) createClient(conn net.Conn) *Client {
	clientID := uuid.New().String()
	client := NewClient(clientID, conn)
	
	cm.mutex.Lock()
	cm.clients[clientID] = client
	cm.mutex.Unlock()
	
	fmt.Printf("New client connected: %s from %s\n", clientID, conn.RemoteAddr())
	return client
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
	cm.mutex.RLock()
	clients := make([]*Client, 0)
	for _, client := range cm.clients {
		if client.IsConnected() && client.GetState() == StateInGame {
			// TODO: Check if client's character is in the specified room
			clients = append(clients, client)
		}
	}
	cm.mutex.RUnlock()
	
	for _, client := range clients {
		client.Send(message)
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
		TotalClients:     len(cm.clients),
		AuthenticatedClients: 0,
		InGameClients:    0,
	}
	
	for _, client := range cm.clients {
		switch client.GetState() {
		case StateCharacterSelection, StateInGame:
			stats.AuthenticatedClients++
			if client.GetState() == StateInGame {
				stats.InGameClients++
			}
		}
	}
	
	return stats
}

func (cm *ConnectionManager) cleanupClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if !cm.running {
				return
			}
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
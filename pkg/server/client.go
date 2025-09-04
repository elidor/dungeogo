package server

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type Client struct {
	ID         string
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	connected  bool
	playerID   string
	characterID string
	state      ClientState
	lastActive time.Time
	mutex      sync.RWMutex
}

type ClientState int

const (
	StateConnected ClientState = iota
	StateAuthenticating
	StateCharacterSelection
	StateInGame
	StateDisconnecting
)

func NewClient(id string, conn net.Conn) *Client {
	return &Client{
		ID:         id,
		conn:       conn,
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
		connected:  true,
		state:      StateConnected,
		lastActive: time.Now(),
	}
}

func (c *Client) Send(message string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return ErrClientDisconnected
	}
	
	_, err := c.writer.WriteString(message + "\r\n")
	if err != nil {
		return err
	}
	
	return c.writer.Flush()
}

func (c *Client) SendPrompt(prompt string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return ErrClientDisconnected
	}
	
	_, err := c.writer.WriteString(prompt)
	if err != nil {
		return err
	}
	
	return c.writer.Flush()
}

func (c *Client) ReadLine() (string, error) {
	c.updateLastActive()
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	// Remove trailing newline and carriage return
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	
	return line, nil
}

func (c *Client) GetID() string {
	return c.ID
}

func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return nil
	}
	
	c.connected = false
	c.state = StateDisconnecting
	return c.conn.Close()
}

func (c *Client) GetPlayerID() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.playerID
}

func (c *Client) SetPlayerID(playerID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.playerID = playerID
}

func (c *Client) GetCharacterID() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.characterID
}

func (c *Client) SetCharacterID(characterID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.characterID = characterID
}

func (c *Client) GetState() ClientState {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.state
}

func (c *Client) SetState(state ClientState) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.state = state
}

func (c *Client) GetLastActive() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastActive
}

func (c *Client) updateLastActive() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastActive = time.Now()
}

func (c *Client) IsIdle(timeout time.Duration) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Since(c.lastActive) > timeout
}

func (c *Client) GetRemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
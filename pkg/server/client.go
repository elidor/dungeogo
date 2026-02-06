package server

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type Client struct {
	ID                 string
	conn               net.Conn
	reader             *bufio.Reader
	writer             *bufio.Writer
	connected          bool
	playerID           string
	characterID        string
	state              ClientState
	lastActive         time.Time
	tempUsername       string // For storing username during account creation
	tempPassword       string // For storing password during confirmation
	tempEmail          string // For storing email during account creation
	tempCharacterName  string // For interactive character creation
	tempCharacterRace  string // For interactive character creation
	tempCharacterClass string // For interactive character creation
	tempCreationStep   int    // 0=name, 1=race, 2=class, 3=confirm
	mutex              sync.RWMutex
}

type ClientState int

const (
	StateConnected ClientState = iota
	StateAuthenticating
	StateCreatingAccount
	StateConfirmingPassword
	StateCharacterSelection
	StateCharacterCreation
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

// ReadPassword reads a password from the client with echo disabled
func (c *Client) ReadPassword() (string, error) {
	c.updateLastActive()

	// Send telnet command to disable echo
	// IAC WILL ECHO tells the client we (server) will handle echoing
	_, err := c.conn.Write([]byte{255, 251, 1}) // IAC WILL ECHO
	if err != nil {
		return "", err
	}

	// Read the password, handling potential telnet control sequences
	var line string
	for {
		char, err := c.reader.ReadByte()
		if err != nil {
			// Re-enable echo before returning error
			c.conn.Write([]byte{255, 252, 1}) // IAC WONT ECHO
			return "", err
		}

		// Handle telnet IAC (Interpret As Command) sequences
		if char == 255 { // IAC
			// Read the next two bytes to complete the telnet sequence
			c.reader.ReadByte() // command
			c.reader.ReadByte() // option
			continue            // Skip telnet control sequences
		}

		// End of line
		if char == '\n' {
			break
		}

		// Skip carriage return
		if char == '\r' {
			continue
		}

		// Add normal character to password
		line += string(char)
	}

	// Re-enable echo - tell client we won't handle echoing anymore
	_, err = c.conn.Write([]byte{255, 252, 1}) // IAC WONT ECHO
	if err != nil {
		return "", err
	}

	// Send a newline to the client since they won't see the echo
	c.writer.WriteString("\r\n")
	c.writer.Flush()

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

// Temporary data getters/setters for account creation
func (c *Client) GetTempUsername() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempUsername
}

func (c *Client) SetTempUsername(username string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempUsername = username
}

func (c *Client) GetTempPassword() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempPassword
}

func (c *Client) SetTempPassword(password string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempPassword = password
}

func (c *Client) GetTempEmail() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempEmail
}

func (c *Client) SetTempEmail(email string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempEmail = email
}

func (c *Client) ClearTempData() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempUsername = ""
	c.tempPassword = ""
	c.tempEmail = ""
}

// Temporary character creation data getters/setters
func (c *Client) GetTempCharacterName() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempCharacterName
}

func (c *Client) SetTempCharacterName(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempCharacterName = name
}

func (c *Client) GetTempCharacterRace() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempCharacterRace
}

func (c *Client) SetTempCharacterRace(race string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempCharacterRace = race
}

func (c *Client) GetTempCharacterClass() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempCharacterClass
}

func (c *Client) SetTempCharacterClass(class string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempCharacterClass = class
}

func (c *Client) GetTempCreationStep() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tempCreationStep
}

func (c *Client) SetTempCreationStep(step int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempCreationStep = step
}

func (c *Client) ClearTempCharacterData() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tempCharacterName = ""
	c.tempCharacterRace = ""
	c.tempCharacterClass = ""
	c.tempCreationStep = 0
}

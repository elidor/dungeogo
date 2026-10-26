package server

import (
	"sort"
	"sync"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

// SessionRegistry owns authenticated accounts and selected characters for one
// server process. Account gates serialize replacement with authorized actions.
// Gates remain stable across disconnects so queued actions can never use a new lock.
type SessionRegistry struct {
	mutex    sync.RWMutex
	accounts map[string]*accountSession
}

type accountSession struct {
	action    sync.Mutex
	client    *Client
	character *interfaces.OnlineCharacter
}

func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{accounts: make(map[string]*accountSession)}
}

// Register is called only after authentication. An action already holding the
// account gate finishes before replacement; queued stale actions are rejected.
func (r *SessionRegistry) Register(client *Client) {
	id := client.GetPlayerID()
	r.mutex.Lock()
	account := r.accounts[id]
	if account == nil {
		account = &accountSession{}
		r.accounts[id] = account
	}
	r.mutex.Unlock()
	account.action.Lock()
	r.mutex.Lock()
	old := account.client
	account.client = client
	account.character = nil
	r.mutex.Unlock()
	account.action.Unlock()
	if old != nil && old != client {
		old.Close()
	}
}

func (r *SessionRegistry) WithSession(client *Client, action func()) bool {
	r.mutex.RLock()
	account := r.accounts[client.GetPlayerID()]
	r.mutex.RUnlock()
	if account == nil {
		return false
	}
	account.action.Lock()
	defer account.action.Unlock()
	r.mutex.RLock()
	current := account.client == client
	r.mutex.RUnlock()
	if !current || !client.IsConnected() {
		return false
	}
	action()
	return true
}

// Select is called inside WithSession after durable location recovery succeeds.
func (r *SessionRegistry) Select(client *Client, selected *character.Character) {
	entry := &interfaces.OnlineCharacter{ID: selected.ID, Name: selected.Name, Level: selected.Level}
	if selected.Race != nil {
		entry.Race = selected.Race.Name
	}
	if selected.Class != nil {
		entry.Class = selected.Class.Name
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if account := r.accounts[client.GetPlayerID()]; account != nil && account.client == client {
		account.character = entry
	}
}

func (r *SessionRegistry) Remove(client *Client) {
	r.mutex.RLock()
	account := r.accounts[client.GetPlayerID()]
	r.mutex.RUnlock()
	if account == nil {
		return
	}
	account.action.Lock()
	defer account.action.Unlock()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if account.client == client {
		account.client = nil
		account.character = nil
	}
}

// Characters returns value snapshots, never pointers to mutable session state.
func (r *SessionRegistry) Characters() []interfaces.OnlineCharacter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	online := make([]interfaces.OnlineCharacter, 0)
	for _, account := range r.accounts {
		if account.client != nil && account.character != nil && account.client.IsConnected() && account.client.GetState() == StateInGame {
			online = append(online, *account.character)
		}
	}
	sort.Slice(online, func(i, j int) bool { return online[i].Name < online[j].Name })
	return online
}

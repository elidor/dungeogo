package player

import (
	"time"
	
	"github.com/google/uuid"
)

type Player struct {
	ID                 string
	Username           string
	Email              string
	PasswordHash       string
	CreatedAt          time.Time
	LastLogin          time.Time
	AccountStatus      AccountStatus
	Subscription       *Subscription
	Preferences        PlayerPrefs
	MaxCharacters      int
	CurrentCharacterID string
}

type AccountStatus int

const (
	AccountActive AccountStatus = iota
	AccountSuspended
	AccountBanned
)

type Subscription struct {
	Type      SubscriptionType
	ExpiresAt time.Time
	Active    bool
}

type SubscriptionType int

const (
	SubscriptionFree SubscriptionType = iota
	SubscriptionPremium
)

type PlayerPrefs struct {
	ColorEnabled    bool
	ScreenWidth     int
	AutoLoot        bool
	CombatPrompts   bool
	Keybindings     map[string]string
}

func NewPlayer(username, email, passwordHash string) *Player {
	return &Player{
		ID:            uuid.New().String(),
		Username:      username,
		Email:         email,
		PasswordHash:  passwordHash,
		CreatedAt:     time.Now(),
		LastLogin:     time.Now(),
		AccountStatus: AccountActive,
		MaxCharacters: 5,
		Preferences: PlayerPrefs{
			ColorEnabled:  true,
			ScreenWidth:   80,
			AutoLoot:      false,
			CombatPrompts: true,
			Keybindings:   make(map[string]string),
		},
	}
}

func (p *Player) IsActive() bool {
	return p.AccountStatus == AccountActive
}

func (p *Player) HasPremium() bool {
	return p.Subscription != nil && 
		   p.Subscription.Active && 
		   p.Subscription.Type == SubscriptionPremium &&
		   p.Subscription.ExpiresAt.After(time.Now())
}

func (p *Player) UpdateLastLogin() {
	p.LastLogin = time.Now()
}
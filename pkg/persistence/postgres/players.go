package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/elidor/dungeogo/pkg/game/player"
)

type PlayerRepository struct {
	db *sql.DB
}

func NewPlayerRepository(db *sql.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) CreatePlayer(p *player.Player) error {
	prefsJSON, err := json.Marshal(p.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}
	
	var subscriptionJSON interface{}
	if p.Subscription != nil {
		subscBytes, err := json.Marshal(p.Subscription)
		if err != nil {
			return fmt.Errorf("failed to marshal subscription: %w", err)
		}
		subscriptionJSON = subscBytes
	} else {
		subscriptionJSON = nil
	}
	
	query := `
		INSERT INTO players (id, username, email, password_hash, created_at, last_login, 
			account_status, subscription, preferences, max_characters, current_character_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	
	var currentCharacterID interface{}
	if p.CurrentCharacterID == "" {
		currentCharacterID = nil
	} else {
		currentCharacterID = p.CurrentCharacterID
	}
	
	_, err = r.db.Exec(query, p.ID, p.Username, p.Email, p.PasswordHash, 
		p.CreatedAt, p.LastLogin, int(p.AccountStatus), subscriptionJSON, 
		prefsJSON, p.MaxCharacters, currentCharacterID)
	
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}
	
	return nil
}

func (r *PlayerRepository) GetPlayer(playerID string) (*player.Player, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, last_login,
			account_status, subscription, preferences, max_characters, current_character_id
		FROM players WHERE id = $1`
	
	p := &player.Player{}
	var subscriptionJSON, prefsJSON []byte
	var currentCharacterID sql.NullString
	var accountStatus int
	
	err := r.db.QueryRow(query, playerID).Scan(
		&p.ID, &p.Username, &p.Email, &p.PasswordHash, &p.CreatedAt,
		&p.LastLogin, &accountStatus, &subscriptionJSON, &prefsJSON,
		&p.MaxCharacters, &currentCharacterID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found: %s", playerID)
		}
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	
	p.AccountStatus = player.AccountStatus(accountStatus)
	
	if currentCharacterID.Valid {
		p.CurrentCharacterID = currentCharacterID.String
	} else {
		p.CurrentCharacterID = ""
	}
	
	if subscriptionJSON != nil {
		p.Subscription = &player.Subscription{}
		if err := json.Unmarshal(subscriptionJSON, p.Subscription); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subscription: %w", err)
		}
	}
	
	if err := json.Unmarshal(prefsJSON, &p.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}
	
	return p, nil
}

func (r *PlayerRepository) GetPlayerByUsername(username string) (*player.Player, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, last_login,
			account_status, subscription, preferences, max_characters, current_character_id
		FROM players WHERE username = $1`
	
	p := &player.Player{}
	var subscriptionJSON, prefsJSON []byte
	var currentCharacterID sql.NullString
	var accountStatus int
	
	err := r.db.QueryRow(query, username).Scan(
		&p.ID, &p.Username, &p.Email, &p.PasswordHash, &p.CreatedAt,
		&p.LastLogin, &accountStatus, &subscriptionJSON, &prefsJSON,
		&p.MaxCharacters, &currentCharacterID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get player by username: %w", err)
	}
	
	p.AccountStatus = player.AccountStatus(accountStatus)
	
	if currentCharacterID.Valid {
		p.CurrentCharacterID = currentCharacterID.String
	} else {
		p.CurrentCharacterID = ""
	}
	
	if subscriptionJSON != nil {
		p.Subscription = &player.Subscription{}
		if err := json.Unmarshal(subscriptionJSON, p.Subscription); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subscription: %w", err)
		}
	}
	
	if err := json.Unmarshal(prefsJSON, &p.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}
	
	return p, nil
}

func (r *PlayerRepository) GetPlayerByEmail(email string) (*player.Player, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, last_login,
			account_status, subscription, preferences, max_characters, current_character_id
		FROM players WHERE email = $1`
	
	p := &player.Player{}
	var subscriptionJSON, prefsJSON []byte
	var currentCharacterID sql.NullString
	var accountStatus int
	
	err := r.db.QueryRow(query, email).Scan(
		&p.ID, &p.Username, &p.Email, &p.PasswordHash, &p.CreatedAt,
		&p.LastLogin, &accountStatus, &subscriptionJSON, &prefsJSON,
		&p.MaxCharacters, &currentCharacterID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found: %s", email)
		}
		return nil, fmt.Errorf("failed to get player by email: %w", err)
	}
	
	p.AccountStatus = player.AccountStatus(accountStatus)
	
	if currentCharacterID.Valid {
		p.CurrentCharacterID = currentCharacterID.String
	} else {
		p.CurrentCharacterID = ""
	}
	
	if subscriptionJSON != nil {
		p.Subscription = &player.Subscription{}
		if err := json.Unmarshal(subscriptionJSON, p.Subscription); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subscription: %w", err)
		}
	}
	
	if err := json.Unmarshal(prefsJSON, &p.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}
	
	return p, nil
}

func (r *PlayerRepository) UpdatePlayer(p *player.Player) error {
	prefsJSON, err := json.Marshal(p.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}
	
	var subscriptionJSON []byte
	if p.Subscription != nil {
		subscriptionJSON, err = json.Marshal(p.Subscription)
		if err != nil {
			return fmt.Errorf("failed to marshal subscription: %w", err)
		}
	}
	
	query := `
		UPDATE players SET username = $2, email = $3, password_hash = $4, 
			last_login = $5, account_status = $6, subscription = $7, 
			preferences = $8, max_characters = $9, current_character_id = $10
		WHERE id = $1`
	
	_, err = r.db.Exec(query, p.ID, p.Username, p.Email, p.PasswordHash,
		p.LastLogin, int(p.AccountStatus), subscriptionJSON, prefsJSON,
		p.MaxCharacters, p.CurrentCharacterID)
	
	if err != nil {
		return fmt.Errorf("failed to update player: %w", err)
	}
	
	return nil
}

func (r *PlayerRepository) UpdatePlayerLogin(playerID string) error {
	query := `UPDATE players SET last_login = $1 WHERE id = $2`
	_, err := r.db.Exec(query, time.Now(), playerID)
	if err != nil {
		return fmt.Errorf("failed to update player login: %w", err)
	}
	return nil
}

func (r *PlayerRepository) DeletePlayer(playerID string) error {
	query := `DELETE FROM players WHERE id = $1`
	_, err := r.db.Exec(query, playerID)
	if err != nil {
		return fmt.Errorf("failed to delete player: %w", err)
	}
	return nil
}
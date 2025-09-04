package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

type CharacterRepository struct {
	db *sql.DB
}

func NewCharacterRepository(db *sql.DB) *CharacterRepository {
	return &CharacterRepository{db: db}
}

func (r *CharacterRepository) CreateCharacter(c *character.Character) error {
	statsJSON, err := json.Marshal(c.Stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}
	
	skillsJSON, err := json.Marshal(c.Skills)
	if err != nil {
		return fmt.Errorf("failed to marshal skills: %w", err)
	}
	
	locationJSON, err := json.Marshal(c.Location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}
	
	appearanceJSON, err := json.Marshal(c.Appearance)
	if err != nil {
		return fmt.Errorf("failed to marshal appearance: %w", err)
	}
	
	var raceID, classID string
	if c.Race != nil {
		raceID = c.Race.ID
	}
	if c.Class != nil {
		classID = c.Class.ID
	}
	
	query := `
		INSERT INTO characters (id, player_id, name, race_id, class_id, stats, 
			skills, location, state, created_at, last_played, play_time, level, 
			experience, death_count, kill_count, description, appearance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	
	_, err = r.db.Exec(query, c.ID, c.PlayerID, c.Name, raceID, classID,
		statsJSON, skillsJSON, locationJSON, int(c.State), c.CreatedAt,
		c.LastPlayed, c.PlayTime, c.Level, c.Experience, c.DeathCount,
		c.KillCount, c.Description, appearanceJSON)
	
	if err != nil {
		return fmt.Errorf("failed to create character: %w", err)
	}
	
	return nil
}

func (r *CharacterRepository) GetCharacter(characterID string) (*character.Character, error) {
	query := `
		SELECT id, player_id, name, race_id, class_id, stats, skills, location,
			state, created_at, last_played, play_time, level, experience,
			death_count, kill_count, description, appearance
		FROM characters WHERE id = $1`
	
	c := &character.Character{}
	var raceID, classID string
	var statsJSON, skillsJSON, locationJSON, appearanceJSON []byte
	var state int
	
	err := r.db.QueryRow(query, characterID).Scan(
		&c.ID, &c.PlayerID, &c.Name, &raceID, &classID, &statsJSON,
		&skillsJSON, &locationJSON, &state, &c.CreatedAt, &c.LastPlayed,
		&c.PlayTime, &c.Level, &c.Experience, &c.DeathCount, &c.KillCount,
		&c.Description, &appearanceJSON)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("character not found: %s", characterID)
		}
		return nil, fmt.Errorf("failed to get character: %w", err)
	}
	
	c.State = character.CharacterState(state)
	
	// Load race and class
	if raceID != "" {
		c.Race, _ = character.GetRaceByID(raceID)
	}
	if classID != "" {
		c.Class, _ = character.GetClassByID(classID)
	}
	
	// Unmarshal JSON fields
	if err := json.Unmarshal(statsJSON, &c.Stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}
	
	if err := json.Unmarshal(skillsJSON, &c.Skills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skills: %w", err)
	}
	
	if err := json.Unmarshal(locationJSON, &c.Location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %w", err)
	}
	
	if err := json.Unmarshal(appearanceJSON, &c.Appearance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal appearance: %w", err)
	}
	
	return c, nil
}

func (r *CharacterRepository) GetCharactersByPlayer(playerID string) ([]*interfaces.CharacterSummary, error) {
	query := `
		SELECT id, name, race_id, class_id, level, location, last_played, state
		FROM characters WHERE player_id = $1 ORDER BY last_played DESC`
	
	rows, err := r.db.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get characters: %w", err)
	}
	defer rows.Close()
	
	var characters []*interfaces.CharacterSummary
	for rows.Next() {
		var summary interfaces.CharacterSummary
		var raceID, classID string
		var locationJSON []byte
		var state int
		
		err := rows.Scan(&summary.ID, &summary.Name, &raceID, &classID,
			&summary.Level, &locationJSON, &summary.LastPlayed, &state)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character: %w", err)
		}
		
		// Set race and class names
		if race, err := character.GetRaceByID(raceID); err == nil {
			summary.Race = race.Name
		}
		if class, err := character.GetClassByID(classID); err == nil {
			summary.Class = class.Name
		}
		
		// Parse location for display
		var location character.Location
		if err := json.Unmarshal(locationJSON, &location); err == nil {
			summary.Location = location.RoomID
		}
		
		summary.IsAlive = character.CharacterState(state) == character.CharacterAlive
		
		characters = append(characters, &summary)
	}
	
	return characters, nil
}

func (r *CharacterRepository) UpdateCharacter(c *character.Character) error {
	statsJSON, err := json.Marshal(c.Stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}
	
	skillsJSON, err := json.Marshal(c.Skills)
	if err != nil {
		return fmt.Errorf("failed to marshal skills: %w", err)
	}
	
	locationJSON, err := json.Marshal(c.Location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}
	
	appearanceJSON, err := json.Marshal(c.Appearance)
	if err != nil {
		return fmt.Errorf("failed to marshal appearance: %w", err)
	}
	
	query := `
		UPDATE characters SET stats = $2, skills = $3, location = $4, state = $5,
			last_played = $6, play_time = $7, level = $8, experience = $9,
			death_count = $10, kill_count = $11, description = $12, appearance = $13
		WHERE id = $1`
	
	_, err = r.db.Exec(query, c.ID, statsJSON, skillsJSON, locationJSON,
		int(c.State), c.LastPlayed, c.PlayTime, c.Level, c.Experience,
		c.DeathCount, c.KillCount, c.Description, appearanceJSON)
	
	if err != nil {
		return fmt.Errorf("failed to update character: %w", err)
	}
	
	return nil
}

func (r *CharacterRepository) DeleteCharacter(characterID string) error {
	query := `DELETE FROM characters WHERE id = $1`
	_, err := r.db.Exec(query, characterID)
	if err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}
	return nil
}

func (r *CharacterRepository) UpdateCharacterStats(characterID string, stats *character.CharacterStats) error {
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}
	
	query := `UPDATE characters SET stats = $2 WHERE id = $1`
	_, err = r.db.Exec(query, characterID, statsJSON)
	if err != nil {
		return fmt.Errorf("failed to update character stats: %w", err)
	}
	return nil
}

func (r *CharacterRepository) UpdateCharacterLocation(characterID string, location *character.Location) error {
	locationJSON, err := json.Marshal(location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}
	
	query := `UPDATE characters SET location = $2 WHERE id = $1`
	_, err = r.db.Exec(query, characterID, locationJSON)
	if err != nil {
		return fmt.Errorf("failed to update character location: %w", err)
	}
	return nil
}

func (r *CharacterRepository) SaveCharacterSkills(characterID string, skills *character.SkillSet) error {
	skillsJSON, err := json.Marshal(skills)
	if err != nil {
		return fmt.Errorf("failed to marshal skills: %w", err)
	}
	
	query := `UPDATE characters SET skills = $2, last_played = $3 WHERE id = $1`
	_, err = r.db.Exec(query, characterID, skillsJSON, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save character skills: %w", err)
	}
	return nil
}
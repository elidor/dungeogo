package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

type WorldRepository struct {
	db *sql.DB
}

func NewWorldRepository(db *sql.DB) *WorldRepository {
	return &WorldRepository{db: db}
}

func (r *WorldRepository) SaveRoomState(roomID string, state *interfaces.RoomState) error {
	flagsJSON, err := json.Marshal(state.Flags)
	if err != nil {
		return fmt.Errorf("failed to marshal room flags: %w", err)
	}
	
	itemsJSON, err := json.Marshal(state.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal room items: %w", err)
	}
	
	npcsJSON, err := json.Marshal(state.NPCs)
	if err != nil {
		return fmt.Errorf("failed to marshal room npcs: %w", err)
	}
	
	playersJSON, err := json.Marshal(state.Players)
	if err != nil {
		return fmt.Errorf("failed to marshal room players: %w", err)
	}
	
	query := `
		INSERT INTO room_states (room_id, items, npcs, players, flags, last_update)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (room_id) DO UPDATE SET
			items = $2, npcs = $3, players = $4, flags = $5, last_update = $6`
	
	_, err = r.db.Exec(query, roomID, itemsJSON, npcsJSON, playersJSON, 
		flagsJSON, state.LastUpdate)
	
	if err != nil {
		return fmt.Errorf("failed to save room state: %w", err)
	}
	
	return nil
}

func (r *WorldRepository) LoadRoomState(roomID string) (*interfaces.RoomState, error) {
	query := `
		SELECT room_id, items, npcs, players, flags, last_update
		FROM room_states WHERE room_id = $1`
	
	state := &interfaces.RoomState{}
	var itemsJSON, npcsJSON, playersJSON, flagsJSON []byte
	
	err := r.db.QueryRow(query, roomID).Scan(
		&state.ID, &itemsJSON, &npcsJSON, &playersJSON, &flagsJSON, &state.LastUpdate)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty room state if not found
			return &interfaces.RoomState{
				ID:      roomID,
				Items:   []string{},
				NPCs:    []string{},
				Players: []string{},
				Flags:   make(map[string]interface{}),
			}, nil
		}
		return nil, fmt.Errorf("failed to load room state: %w", err)
	}
	
	if err := json.Unmarshal(itemsJSON, &state.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room items: %w", err)
	}
	
	if err := json.Unmarshal(npcsJSON, &state.NPCs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room npcs: %w", err)
	}
	
	if err := json.Unmarshal(playersJSON, &state.Players); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room players: %w", err)
	}
	
	if err := json.Unmarshal(flagsJSON, &state.Flags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room flags: %w", err)
	}
	
	return state, nil
}

func (r *WorldRepository) SaveNPCState(npcID string, state *interfaces.NPCState) error {
	locationJSON, err := json.Marshal(state.Location)
	if err != nil {
		return fmt.Errorf("failed to marshal npc location: %w", err)
	}
	
	inventoryJSON, err := json.Marshal(state.Inventory)
	if err != nil {
		return fmt.Errorf("failed to marshal npc inventory: %w", err)
	}
	
	query := `
		INSERT INTO npc_states (npc_id, template_id, health, location, inventory, state, last_update)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (npc_id) DO UPDATE SET
			template_id = $2, health = $3, location = $4, inventory = $5, state = $6, last_update = $7`
	
	_, err = r.db.Exec(query, npcID, state.TemplateID, state.Health, 
		locationJSON, inventoryJSON, state.State, state.LastUpdate)
	
	if err != nil {
		return fmt.Errorf("failed to save npc state: %w", err)
	}
	
	return nil
}

func (r *WorldRepository) LoadNPCState(npcID string) (*interfaces.NPCState, error) {
	query := `
		SELECT npc_id, template_id, health, location, inventory, state, last_update
		FROM npc_states WHERE npc_id = $1`
	
	state := &interfaces.NPCState{}
	var locationJSON, inventoryJSON []byte
	
	err := r.db.QueryRow(query, npcID).Scan(
		&state.ID, &state.TemplateID, &state.Health, &locationJSON, 
		&inventoryJSON, &state.State, &state.LastUpdate)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("npc state not found: %s", npcID)
		}
		return nil, fmt.Errorf("failed to load npc state: %w", err)
	}
	
	if err := json.Unmarshal(locationJSON, &state.Location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal npc location: %w", err)
	}
	
	if err := json.Unmarshal(inventoryJSON, &state.Inventory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal npc inventory: %w", err)
	}
	
	return state, nil
}

func (r *WorldRepository) SaveWorldEvent(event *interfaces.WorldEvent) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal world event data: %w", err)
	}
	
	query := `
		INSERT INTO world_events (id, type, description, start_time, end_time, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			type = $2, description = $3, start_time = $4, end_time = $5, data = $6`
	
	_, err = r.db.Exec(query, event.ID, event.Type, event.Description, 
		event.StartTime, event.EndTime, dataJSON)
	
	if err != nil {
		return fmt.Errorf("failed to save world event: %w", err)
	}
	
	return nil
}

func (r *WorldRepository) GetActiveWorldEvents() ([]*interfaces.WorldEvent, error) {
	query := `
		SELECT id, type, description, start_time, end_time, data
		FROM world_events 
		WHERE start_time <= NOW() AND (end_time IS NULL OR end_time > NOW())
		ORDER BY start_time`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active world events: %w", err)
	}
	defer rows.Close()
	
	var events []*interfaces.WorldEvent
	for rows.Next() {
		event := &interfaces.WorldEvent{}
		var dataJSON []byte
		
		err := rows.Scan(&event.ID, &event.Type, &event.Description,
			&event.StartTime, &event.EndTime, &dataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan world event: %w", err)
		}
		
		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal world event data: %w", err)
		}
		
		events = append(events, event)
	}
	
	return events, nil
}
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elidor/dungeogo/migrations"
)

const DefaultStartingRoom = "starting_room"

// Bootstrap upgrades the database and validates the world before sessions start.
func (m *PostgreSQLRepositoryManager) Bootstrap(ctx context.Context, startingRoom string) error {
	if err := migrations.Apply(ctx, m.db); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return m.ValidateWorld(ctx, startingRoom)
}

func (m *PostgreSQLRepositoryManager) ValidateWorld(ctx context.Context, startingRoom string) error {
	rows, err := m.db.QueryContext(ctx, `SELECT room_id, exits FROM room_definitions ORDER BY room_id`)
	if err != nil {
		return fmt.Errorf("load world for validation: %w", err)
	}
	defer rows.Close()
	rooms := make(map[string]map[string]string)
	var order []string
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		var exits map[string]string
		if err := json.Unmarshal(raw, &exits); err != nil || exits == nil {
			return fmt.Errorf("room %q has invalid exits; expected an object of direction-to-room strings", id)
		}
		rooms[id] = exits
		order = append(order, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, ok := rooms[startingRoom]; !ok {
		return fmt.Errorf("starting room %q does not exist; check STARTING_ROOM and room_definitions", startingRoom)
	}
	for _, id := range order {
		for direction, target := range rooms[id] {
			if _, ok := rooms[target]; !ok || target == "" {
				return fmt.Errorf("room %q exit %q targets missing room %q; repair room_definitions", id, direction, target)
			}
		}
	}
	return nil
}

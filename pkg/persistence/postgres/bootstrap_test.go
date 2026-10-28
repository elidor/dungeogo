package postgres

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
)

func TestBootstrapFreshAndRepeated(t *testing.T) {
	db, databaseURL := testdb.New(t, false)
	manager, err := NewPostgreSQLRepositoryManager(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	ctx := context.Background()
	if err := manager.Bootstrap(ctx, DefaultStartingRoom); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE room_definitions SET description='Operator edit' WHERE room_id='starting_room'; INSERT INTO item_instances(template_id,owner_kind,owner_id) VALUES ('keepsake','room','newbie_west')`); err != nil {
		t.Fatal(err)
	}
	if err := manager.Bootstrap(ctx, DefaultStartingRoom); err != nil {
		t.Fatal(err)
	}
	var description string
	if err := db.QueryRow(`SELECT description FROM room_definitions WHERE room_id='starting_room'`).Scan(&description); err != nil || description != "Operator edit" {
		t.Fatalf("repeat boot reset world: %q %v", description, err)
	}
	scripts, err := filepath.Glob(filepath.Join("..", "..", "..", "migrations", "*.sql"))
	if err != nil || len(scripts) == 0 {
		t.Fatalf("list migrations: %v %v", scripts, err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil || count != len(scripts) {
		t.Fatalf("migration count %d, want %d (each applied exactly once): %v", count, len(scripts), err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM item_instances WHERE template_id='keepsake' AND owner_kind='room' AND owner_id='newbie_west'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("item count %d: %v", count, err)
	}
}

func TestBootstrapRejectsInvalidWorld(t *testing.T) {
	for _, tc := range []struct{ name, sql, spawn, want string }{
		{"missing spawn", `DELETE FROM item_instances WHERE owner_kind='room' AND owner_id='starting_room'; DELETE FROM room_definitions WHERE room_id='starting_room'`, "starting_room", `starting room "starting_room"`},
		{"bad exit", `UPDATE room_definitions SET exits='{"north":"nowhere"}' WHERE room_id='starting_room'`, "starting_room", `exit "north" targets missing room "nowhere"`},
		{"malformed exits", `UPDATE room_definitions SET exits='[]' WHERE room_id='newbie_north'`, "starting_room", `room "newbie_north" has invalid exits`},
		{"configured spawn", `SELECT 1`, "missing_custom_spawn", `starting room "missing_custom_spawn"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, databaseURL := testdb.New(t, true)
			manager, err := NewPostgreSQLRepositoryManager(databaseURL)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close()
			if _, err := db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			err = manager.Bootstrap(context.Background(), tc.spawn)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

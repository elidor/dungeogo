package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/migrations"
)

func TestUpgradePreservesData(t *testing.T) {
	for _, legacy := range []string{"001_initial_schema.sql", "002_room_definitions.sql"} {
		t.Run(legacy, func(t *testing.T) {
			db, _ := testdb.New(t, false)
			for _, name := range []string{"001_initial_schema.sql", "002_room_definitions.sql"} {
				script, err := os.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(string(script)); err != nil {
					t.Fatal(err)
				}
				if name == legacy {
					break
				}
			}
			_, err := db.Exec(`
    INSERT INTO players(id, username,email,password_hash) VALUES ('00000000-0000-0000-0000-000000000001','legacy','legacy@example.com','hash');
    INSERT INTO characters(id,player_id,name,race_id,class_id,location) VALUES ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','Legacy','human','warrior','{"room_id":"newbie_north"}');
    INSERT INTO item_instances(template_id,owner_id) VALUES ('sword','00000000-0000-0000-0000-000000000002');
    INSERT INTO room_states(room_id,flags) VALUES ('starting_room','{"visited":true}');`)
			if err != nil {
				t.Fatal(err)
			}
			if err := migrations.Apply(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM room_definitions`).Scan(&count); err != nil || count != 9 {
				t.Fatalf("rooms=%d err=%v", count, err)
			}
			var preserved bool
			if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM players p JOIN characters c ON c.player_id=p.id JOIN item_instances i ON i.owner_id=c.id::text WHERE p.username='legacy' AND c.location->>'room_id'='newbie_north') AND EXISTS(SELECT 1 FROM room_states WHERE flags->>'visited'='true')`).Scan(&preserved); err != nil || !preserved {
				t.Fatalf("legacy data lost: %v", err)
			}
		})
	}
}

func TestIncompatibleLegacySchemaIsNotAdopted(t *testing.T) {
	db, _ := testdb.New(t, false)
	script, err := os.ReadFile("001_initial_schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(script)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE characters DROP COLUMN location`); err != nil {
		t.Fatal(err)
	}
	err = migrations.Apply(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "characters") {
		t.Fatalf("expected actionable incompatible schema error, got %v", err)
	}
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatalf("failed adoption should roll back history: exists=%v err=%v", exists, err)
	}
}

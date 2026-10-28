package migrations_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/migrations"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
)

const legacyCharacter = "00000000-0000-0000-0000-000000000002"
const legacyItem = "00000000-0000-0000-0000-000000000003"

func beforeItemOwnership(t *testing.T, tracked bool) (*sql.DB, string) {
	t.Helper()
	db, url := testdb.New(t, false)
	for _, name := range []string{"001_initial_schema.sql", "002_room_definitions.sql", "003_newbie_zone_grid.sql"} {
		script, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO players(id,username,email,password_hash) VALUES ('00000000-0000-0000-0000-000000000001','legacy','legacy@example.com','hash'); INSERT INTO characters(id,player_id,name,race_id,class_id) VALUES ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','Legacy','human','warrior')`); err != nil {
		t.Fatal(err)
	}
	if tracked {
		if _, err := db.Exec(`CREATE TABLE schema_migrations(version text PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now()); INSERT INTO schema_migrations(version) VALUES ('001_initial_schema.sql'),('002_room_definitions.sql'),('003_newbie_zone_grid.sql')`); err != nil {
			t.Fatal(err)
		}
	}
	return db, url
}

func TestItemOwnershipUpgradePreservesLegacyRows(t *testing.T) {
	for _, tracked := range []bool{false, true} {
		name := "unversioned"
		if tracked {
			name = "versioned"
		}
		t.Run(name, func(t *testing.T) {
			db, url := beforeItemOwnership(t, tracked)
			if _, err := db.Exec(`INSERT INTO item_instances(id,template_id,owner_id,quantity,durability,modifications) VALUES ($1,'rusty_sword',$2,3,47,'{"legacy":true}'),('00000000-0000-0000-0000-000000000004','leather_armor','newbie_east',1,33,'{}')`, legacyItem, legacyCharacter); err != nil {
				t.Fatal(err)
			}
			if err := migrations.Apply(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			manager, err := postgres.NewPostgreSQLRepositoryManager(url)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close()
			item, err := manager.Items().GetItemInstance(legacyItem)
			if err != nil {
				t.Fatal(err)
			}
			if item.OwnerKind != items.OwnerCharacter || item.OwnerID != legacyCharacter || item.Quantity != 3 || item.Durability != 47 || item.Modifications["legacy"] != true || !item.LastUsed.IsZero() || item.GetDisplayName() != "Rusty Sword" {
				t.Fatalf("legacy item altered: %+v", item)
			}
			roomItems, err := manager.Items().GetRoomItems("newbie_east")
			if err != nil || len(roomItems) != 1 || roomItems[0].OwnerKind != items.OwnerRoom {
				t.Fatalf("legacy room ownership: %v %v", roomItems, err)
			}
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM item_instances`).Scan(&count); err != nil || count != 5 {
				t.Fatalf("expected two existing and three seeded items: %d %v", count, err)
			}
		})
	}
}

func TestItemOwnershipUpgradeRejectsInvalidOwnersAtomically(t *testing.T) {
	for _, mode := range []string{"missing", "ambiguous"} {
		t.Run(mode, func(t *testing.T) {
			db, _ := beforeItemOwnership(t, true)
			owner := "missing_owner"
			if mode == "ambiguous" {
				owner = legacyCharacter
				if _, err := db.Exec(`INSERT INTO room_definitions(room_id,zone_id,name,description) VALUES ($1,'legacy','Ambiguous owner','A room sharing a character ID.')`, owner); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.Exec(`INSERT INTO item_instances(id,template_id,owner_id,durability) VALUES ($1,'keepsake',$2,37)`, legacyItem, owner); err != nil {
				t.Fatal(err)
			}
			err := migrations.Apply(context.Background(), db)
			if err == nil || !strings.Contains(err.Error(), legacyItem) || !strings.Contains(err.Error(), owner) || !strings.Contains(err.Error(), "Repair") {
				t.Fatalf("expected actionable ownership failure: %v", err)
			}
			var count, durability int
			if err := db.QueryRow(`SELECT count(*) FROM item_instances`).Scan(&count); err != nil || count != 1 {
				t.Fatalf("migration changed items: %d %v", count, err)
			}
			if err := db.QueryRow(`SELECT durability FROM item_instances WHERE id=$1`, legacyItem).Scan(&durability); err != nil || durability != 37 {
				t.Fatalf("migration altered legacy row: %d %v", durability, err)
			}
			if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil || count != 3 {
				t.Fatalf("failed migration recorded progress: %d %v", count, err)
			}
			var column bool
			if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='item_instances' AND column_name='owner_kind')`).Scan(&column); err != nil || column {
				t.Fatalf("partial schema escaped rollback: %v %v", column, err)
			}
		})
	}
}

package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/migrations"
)

// ownerColumnIndexed reports whether item_instances has an index whose
// leading key column is the given foreign-key column.
func ownerColumnIndexed(t *testing.T, db *sql.DB, column string) bool {
	t.Helper()
	var indexed bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_index x
			JOIN pg_attribute a ON a.attrelid = x.indrelid AND a.attnum = x.indkey[0]
			WHERE x.indrelid = 'item_instances'::regclass AND a.attname = $1)`, column).Scan(&indexed)
	if err != nil {
		t.Fatal(err)
	}
	return indexed
}

func TestItemOwnerForeignKeysAreIndexed(t *testing.T) {
	db, _ := testdb.New(t, true)
	for i := 0; i < 2; i++ { // repeat startup must be a no-op
		if err := migrations.Apply(context.Background(), db); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}
	for _, column := range []string{"owner_room_id", "owner_character_id"} {
		if !ownerColumnIndexed(t, db, column) {
			t.Errorf("no index leads with foreign-key column %s", column)
		}
	}
}

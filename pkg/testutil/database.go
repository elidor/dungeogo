package testutil

import (
	"database/sql"
	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"testing"
)

func SetupTestDatabase(t *testing.T) (*sql.DB, string) {
	t.Helper()
	return testdb.New(t, true)
}

func ImprovedSetupTestDB(t *testing.T) *postgres.PostgreSQLRepositoryManager {
	t.Helper()
	_, databaseURL := SetupTestDatabase(t)
	manager, err := postgres.NewPostgreSQLRepositoryManager(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Close() })
	return manager
}

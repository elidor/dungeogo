// Package testdb provisions isolated databases for PostgreSQL behavioral tests.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/elidor/dungeogo/migrations"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// New creates an empty database, registers cleanup immediately, and optionally migrates it.
// Only connection unavailability may skip optional tests; setup errors always fail.
func New(t *testing.T, migrate bool) (*sql.DB, string) {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		raw = os.Getenv("POSTGRES_TEST_URL")
	}
	if raw == "" {
		raw = "postgres://testuser:testpass@localhost:5433/postgres?sslmode=disable"
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	q := u.Query()
	q.Set("connect_timeout", "2")
	u.RawQuery = q.Encode()
	admin, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { admin.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		if os.Getenv("DUNGEOGO_REQUIRE_DB") == "1" {
			t.Fatalf("PostgreSQL required but unavailable: %v", err)
		}
		t.Skipf("PostgreSQL unavailable (set TEST_DATABASE_URL; DUNGEOGO_REQUIRE_DB=1 forbids skipping): %v", err)
	}
	name := "dungeogo_test_" + uuid.New().String()[:8]
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		t.Fatalf("create isolated test database: %v", err)
	}
	u.Path = "/" + name
	db, err := sql.Open("postgres", u.String())
	t.Cleanup(func() {
		if db != nil {
			db.Close()
		}
		if _, err := admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1 AND pid<>pg_backend_pid()`, name); err != nil {
			t.Errorf("disconnect test database: %v", err)
		}
		if _, err := admin.Exec(`DROP DATABASE "` + name + `"`); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if migrate {
		if err := migrations.Apply(context.Background(), db); err != nil {
			t.Fatal(fmt.Errorf("migrate test database: %w", err))
		}
	}
	return db, u.String()
}

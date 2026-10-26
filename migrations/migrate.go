// Package migrations owns the versioned schema and starter-world seeds.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var files embed.FS

// Apply applies each migration once, atomically, and serializes concurrent boots.
// Existing installations without history are adopted only after schema validation.
func Apply(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SET LOCAL search_path TO public`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(426739001)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	var tracked int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&tracked); err != nil {
		return err
	}
	if tracked == 0 {
		if err = adoptLegacy(ctx, tx); err != nil {
			return fmt.Errorf("cannot adopt unversioned database: %w", err)
		}
	}
	entries, err := files.ReadDir(".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		var applied bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		script, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(script)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func adoptLegacy(ctx context.Context, tx *sql.Tx) error {
	tables := []string{"players", "characters", "item_instances", "room_states", "npc_states", "world_events"}
	count := 0
	for _, table := range tables {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			return err
		}
		if exists {
			count++
		}
	}
	var rooms bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.room_definitions') IS NOT NULL`).Scan(&rooms); err != nil {
		return err
	}
	if count == 0 && !rooms {
		return nil
	}
	if count != len(tables) {
		return fmt.Errorf("partial initial schema; expected all six initial tables; restore or repair the schema before starting")
	}
	// Build the expected schema from the actual SQL, without touching existing data.
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA dungeogo_baseline_check; SET LOCAL search_path TO dungeogo_baseline_check, public`); err != nil {
		return err
	}
	versions := []string{"001_initial_schema.sql"}
	if rooms {
		versions = append(versions, "002_room_definitions.sql")
		tables = append(tables, "room_definitions")
	}
	for _, version := range versions {
		script, _ := files.ReadFile(version)
		if _, err := tx.ExecContext(ctx, string(script)); err != nil {
			return err
		}
	}
	for _, table := range tables {
		// Compare column names, types, nullability and defaults, plus constraints.
		var compatible bool
		err := tx.QueryRowContext(ctx, `
   WITH signatures AS (
    SELECT n.nspname AS schema, 'column:' || a.attname || ':' || format_type(a.atttypid,a.atttypmod) || ':' || a.attnotnull::text || ':' || coalesce(pg_get_expr(d.adbin,d.adrelid),'') AS signature
    FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace
    LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
    WHERE c.relname=$1 AND n.nspname IN ('public','dungeogo_baseline_check') AND a.attnum>0 AND NOT a.attisdropped
    UNION ALL
    SELECT n.nspname, 'constraint:' || replace(replace(pg_get_constraintdef(k.oid), 'dungeogo_baseline_check.', ''), 'public.', '')
    FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace
    WHERE c.relname=$1 AND n.nspname IN ('public','dungeogo_baseline_check')
   ) SELECT
    (SELECT array_agg(signature ORDER BY signature) FROM signatures WHERE schema='public') =
    (SELECT array_agg(signature ORDER BY signature) FROM signatures WHERE schema='dungeogo_baseline_check')`, table).Scan(&compatible)
		if err != nil {
			return err
		}
		if !compatible {
			return fmt.Errorf("table %s differs from the versioned schema; restore or repair it before starting", table)
		}
	}
	if _, err := tx.ExecContext(ctx, `SET LOCAL search_path TO public; DROP SCHEMA dungeogo_baseline_check CASCADE`); err != nil {
		return err
	}
	for _, version := range versions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO public.schema_migrations(version) VALUES ($1)`, version); err != nil {
			return err
		}
	}
	return nil
}

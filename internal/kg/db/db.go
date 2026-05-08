package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"

	"aikits/internal/storage/sqlite"
)

//go:embed schema.sql
var schemaDDL string

const schemaVersion = 3

// Open opens (or creates) the SQLite DB at path, applies WAL pragmas, and initializes the schema.
// Returns error with exit-code hint on schema mismatch.
func Open(path string) (*sql.DB, error) {
	return sqlite.Open(path, initSchema)
}

// SchemaVersionError is returned when the on-disk schema version doesn't match the binary.
type SchemaVersionError struct {
	Got  int
	Want int
}

func (e *SchemaVersionError) Error() string {
	return fmt.Sprintf("schema version mismatch: DB has version %d, binary expects %d; run 'aikits kg init --reinit' or delete %s",
		e.Got, e.Want, filepath.Join(".kg", "kg.sqlite"))
}

func initSchema(db *sql.DB) error {
	// check if schema_version table exists
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'")
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("check schema_version: %w", err)
	}

	if count > 0 {
		// table exists; check version
		var ver int
		if err := db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&ver); err != nil {
			return fmt.Errorf("read schema version: %w", err)
		}
		if ver == schemaVersion {
			return nil
		}
		// v2 → v3: add FTS5 virtual tables, triggers, and backfill.
		if ver == 2 {
			if err := migrateV2ToV3(db); err != nil {
				return err
			}
			return nil
		}
		return &SchemaVersionError{Got: ver, Want: schemaVersion}
	}

	// fresh DB: apply DDL
	if _, err := db.Exec(schemaDDL); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (?)", schemaVersion); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	return nil
}

// migrateV2ToV3 creates FTS5 virtual tables and triggers, backfills them,
// and updates schema_version to 3 — all inside a single transaction.
func migrateV2ToV3(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin v2→v3 migration: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	stmts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
			name, fqn, doc,
			content='symbols', content_rowid='id'
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS callsites_fts USING fts5(
			callee_text,
			content='callsites', content_rowid='id'
		)`,
		`CREATE TRIGGER IF NOT EXISTS symbols_fts_ai AFTER INSERT ON symbols BEGIN
			INSERT INTO symbols_fts(rowid, name, fqn, doc)
			VALUES (new.id, new.name, COALESCE(new.fqn,''), COALESCE(new.doc,''));
		END`,
		`CREATE TRIGGER IF NOT EXISTS symbols_fts_ad AFTER DELETE ON symbols BEGIN
			INSERT INTO symbols_fts(symbols_fts, rowid, name, fqn, doc)
			VALUES ('delete', old.id, old.name, COALESCE(old.fqn,''), COALESCE(old.doc,''));
		END`,
		`CREATE TRIGGER IF NOT EXISTS symbols_fts_au AFTER UPDATE ON symbols BEGIN
			INSERT INTO symbols_fts(symbols_fts, rowid, name, fqn, doc)
			VALUES ('delete', old.id, old.name, COALESCE(old.fqn,''), COALESCE(old.doc,''));
			INSERT INTO symbols_fts(rowid, name, fqn, doc)
			VALUES (new.id, new.name, COALESCE(new.fqn,''), COALESCE(new.doc,''));
		END`,
		`CREATE TRIGGER IF NOT EXISTS callsites_fts_ai AFTER INSERT ON callsites BEGIN
			INSERT INTO callsites_fts(rowid, callee_text) VALUES (new.id, new.callee_text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS callsites_fts_ad AFTER DELETE ON callsites BEGIN
			INSERT INTO callsites_fts(callsites_fts, rowid, callee_text)
			VALUES ('delete', old.id, old.callee_text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS callsites_fts_au AFTER UPDATE ON callsites BEGIN
			INSERT INTO callsites_fts(callsites_fts, rowid, callee_text)
			VALUES ('delete', old.id, old.callee_text);
			INSERT INTO callsites_fts(rowid, callee_text) VALUES (new.id, new.callee_text);
		END`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return fmt.Errorf("v2→v3 migration DDL: %w", err)
		}
	}

	// Backfill FTS indexes from existing rows.
	if _, err := tx.Exec(`INSERT INTO symbols_fts(symbols_fts) VALUES('rebuild')`); err != nil {
		return fmt.Errorf("symbols_fts rebuild: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO callsites_fts(callsites_fts) VALUES('rebuild')`); err != nil {
		return fmt.Errorf("callsites_fts rebuild: %w", err)
	}

	if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, schemaVersion); err != nil {
		return fmt.Errorf("update schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit v2→v3 migration: %w", err)
	}
	tx = nil
	return nil
}

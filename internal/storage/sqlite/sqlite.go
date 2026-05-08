// Package sqlite provides a shared SQLite connection opener used by all
// db packages in the application. Centralizing the setup here ensures
// consistent WAL configuration, single-writer enforcement, and pragma
// application across kg and memory databases.
package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DefaultPragmas are applied to every SQLite connection opened via Open.
// They enable WAL journal mode for concurrent reads, enforce foreign-key
// constraints, set a busy timeout to avoid immediate lock errors, and
// keep temp tables in memory.
var DefaultPragmas = []string{
	"PRAGMA journal_mode=WAL",
	"PRAGMA synchronous=NORMAL",
	"PRAGMA foreign_keys=ON",
	"PRAGMA busy_timeout=5000",
	"PRAGMA temp_store=MEMORY",
}

// Open opens (or creates) a SQLite db at the given filesystem path.
// It enforces single-writer mode, applies DefaultPragmas, and calls migrate
// to initialise or upgrade the schema. The migrate function receives the
// raw *sql.DB so it can execute driver-specific DDL.
//
// path must be a filesystem path. In-memory DSNs such as ":memory:" will
// cause MkdirAll to operate on the current directory, which is harmless but
// unnecessary.
func Open(path string, migrate func(*sql.DB) error) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite supports only one concurrent writer; a single connection avoids
	// SQLITE_BUSY contention inside the same process.
	conn.SetMaxOpenConns(1)

	for _, p := range DefaultPragmas {
		if _, err := conn.Exec(p); err != nil {
			conn.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	if migrate != nil {
		if err := migrate(conn); err != nil {
			conn.Close()
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	return conn, nil
}

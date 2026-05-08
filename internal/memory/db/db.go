package db

import (
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"
	"sync"

	"aikits/internal/storage/sqlite"
)

//go:embed migrations/001_initial.sql
var initialSchema string

const (
	defaultDirName  = ".aikits"
	defaultFileName = "memory.db"
)

// DB wraps a *sql.DB with helper methods.
type DB struct {
	conn *sql.DB
	Path string
}

var (
	singleton *DB
	once      sync.Once
	initErr   error
)

// Get returns the singleton DB instance.
// The optional dbPath overrides the default (~/.aikits/memory.db).
func Get(dbPath ...string) (*DB, error) {
	once.Do(func() {
		path := defaultPath()
		if len(dbPath) > 0 && dbPath[0] != "" {
			path = dbPath[0]
		}
		singleton, initErr = open(path)
	})
	return singleton, initErr
}

// ResetForTesting closes and clears the singleton (test use only).
func ResetForTesting() {
	if singleton != nil {
		singleton.conn.Close()
		singleton = nil
	}
	once = sync.Once{}
	initErr = nil
}

func defaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultDirName, defaultFileName)
}

func open(path string) (*DB, error) {
	conn, err := sqlite.Open(path, func(conn *sql.DB) error {
		_, err := conn.Exec(initialSchema)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &DB{conn: conn, Path: path}, nil
}

// QueryRow executes a query expected to return one row.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// Query executes a multi-row query.
func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// Exec executes a statement.
func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// Transaction runs fn inside a db transaction, rolling back on error.
func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Close closes the underlying connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

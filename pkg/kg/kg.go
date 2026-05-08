// Package kg provides a programmatic API over the aikits Knowledge Graph.
// It is the canonical implementation of KG logic; the CLI commands are thin
// adapters that call into this package.
package kg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
	"aikits/internal/kg/pathutil"
)

// KG is a handle to an open knowledge graph database for a single repository.
// All methods are safe to call concurrently as long as the underlying SQLite
// driver supports it (modernc/sqlite does with WAL mode).
type KG struct {
	db   *sql.DB
	repo *kgdb.RepoRow
	root string
	log  *zap.Logger
}

// FindRepoRoot walks up from dir (or the current working directory when dir is
// empty) until it finds a directory containing a .git entry.
func FindRepoRoot(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git directory found; are you inside a git repository?")
		}
		dir = parent
	}
}

// Init creates the .kg directory and initialises (or reinitialises) the
// knowledge graph database for repoRoot, then returns an open KG handle.
func Init(_ context.Context, repoRoot string, reinit bool, log *zap.Logger) (*KG, error) {
	kgDir := filepath.Join(repoRoot, ".kg")
	dbPath := filepath.Join(kgDir, "kg.sqlite")

	if reinit {
		_ = os.Remove(dbPath)
	}

	if err := os.MkdirAll(kgDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .kg directory: %w", err)
	}

	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(repoRoot)
	repoID, err := kgdb.UpsertRepo(db, pathutil.ToSlash(repoRoot), name)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("upsert repo: %w", err)
	}

	repo, err := kgdb.GetRepoByPath(db, pathutil.ToSlash(repoRoot))
	if err != nil || repo == nil {
		db.Close()
		return nil, ErrNotInitialized
	}
	repo.ID = repoID

	if log != nil {
		log.Info("knowledge graph initialized", zap.String("repo", name), zap.String("db", dbPath))
	}
	return &KG{db: db, repo: repo, root: repoRoot, log: log}, nil
}

// Open opens an existing knowledge graph database for repoRoot.
// Returns ErrNotInitialized if .kg/kg.sqlite does not exist.
// Returns *ErrSchemaMismatch if the schema version is out of date.
func Open(repoRoot string, log *zap.Logger) (*KG, error) {
	kgDir := filepath.Join(repoRoot, ".kg")
	if _, err := os.Stat(kgDir); os.IsNotExist(err) {
		return nil, ErrNotInitialized
	}
	dbPath := filepath.Join(kgDir, "kg.sqlite")

	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	repo, err := kgdb.GetRepoByPath(db, pathutil.ToSlash(repoRoot))
	if err != nil || repo == nil {
		db.Close()
		return nil, ErrNotInitialized
	}

	if log == nil {
		log = zap.NewNop()
	}
	return &KG{db: db, repo: repo, root: repoRoot, log: log}, nil
}

// Close releases the database connection.
func (kg *KG) Close() error {
	return kg.db.Close()
}

// RepoRoot returns the absolute path of the repository root.
func (kg *KG) RepoRoot() string { return kg.root }

// openDB opens the SQLite file, translating kgdb errors into package-level
// typed errors.
func openDB(dbPath string) (*sql.DB, error) {
	db, err := kgdb.Open(dbPath)
	if err != nil {
		var sve *kgdb.SchemaVersionError
		if errors.As(err, &sve) {
			return nil, &ErrSchemaMismatch{Got: sve.Got, Want: sve.Want}
		}
		return nil, fmt.Errorf("open db: %w", err)
	}
	return db, nil
}

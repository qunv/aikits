package lang

import (
	"database/sql"
	"strings"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
)

// Indexer is the Strategy interface for per-language source file indexing.
// Implementations must be safe for concurrent use from multiple goroutines.
type Indexer interface {
	// Extract parses src and returns all symbols, callsites, and refs for the file.
	// A non-nil error signals a non-fatal parse warning: callers should log the error
	// and store the file row with an empty FileExtract (no symbols). The error must
	// never be treated as fatal or counted as an index error.
	Extract(src []byte, absPath, relPath string, repoID int64) (FileExtract, error)

	// StoreRefs persists language-specific ref tables (import_refs, extends_refs,
	// type_refs) for the given file. Must be called only after a successful BatchWrite
	// for that file, since it depends on symbols already being indexed.
	StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error
}

// FileExtract holds all extraction results for a single source file.
type FileExtract struct {
	Symbols        []kgdb.SymbolRow
	Callsites      []kgdb.CallsiteRow
	ImportPaths    []string
	SrcPkgFQN      string
	ExtendsRefs    []kgdb.ExtendsRef
	ImplementsRefs []kgdb.ImplementsRef
	TypeRefs       []kgdb.TypeRef
}

// Resolver is the Strategy interface for per-language semantic resolution.
type Resolver interface {
	// Resolve runs the LSP-based semantic upgrade pass for the language.
	Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error
}

// ParseLangs splits a comma-separated language filter string into a slice.
// Returns nil if s is empty (meaning all languages).
func ParseLangs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var langs []string
	for _, p := range parts {
		if t := strings.TrimSpace(strings.ToLower(p)); t != "" {
			langs = append(langs, t)
		}
	}
	return langs
}

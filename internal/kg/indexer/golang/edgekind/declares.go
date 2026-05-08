package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertDeclaresEdges inserts DECLARES edges for Go symbols in the given repo.
// A package DECLARES its direct top-level named symbols (functions, types,
// interfaces, consts, vars) — anything that is not a package, method, field,
// or constructor.
// Safe to call multiple times (INSERT OR IGNORE).
func InsertDeclaresEdges(sqlDB *sql.DB, repoID int64) error {
	q := `INSERT OR IGNORE INTO edges
		(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
	SELECT pkg.repo_id, 'DECLARES', pkg.id, sym.id, 1.0, 'extractor', datetime('now')
	FROM symbols pkg
	JOIN symbols sym ON sym.repo_id = pkg.repo_id AND sym.id != pkg.id
	WHERE pkg.repo_id = ?
	  AND pkg.lang = 'go'
	  AND pkg.kind = 'package'
	  AND sym.kind NOT IN ('package', 'method', 'field', 'constructor')
	  AND sym.fqn = pkg.fqn || '.' || sym.name`

	if _, err := sqlDB.Exec(q, repoID); err != nil {
		return fmt.Errorf("go InsertDeclaresEdges: %w", err)
	}
	return nil
}

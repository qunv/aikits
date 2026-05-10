package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertContainsEdges inserts CONTAINS edges for TypeScript symbols in the given repo.
// Covers:
//   - class → method/function/arrow_function: child.fqn = parent.fqn + '.' + child.name
//   - enum → enum_member: child.fqn = parent.fqn + '.' + child.name
//
// Safe to call multiple times (INSERT OR IGNORE).
func InsertContainsEdges(sqlDB *sql.DB, repoID int64) error {
	_, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT child.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols child
		JOIN symbols parent ON parent.repo_id = child.repo_id
		WHERE child.repo_id = ?
		  AND child.lang = 'typescript'
		  AND (
		      (child.kind IN ('method', 'function', 'arrow_function') AND parent.kind IN ('class', 'function', 'arrow_function'))
		   OR (child.kind = 'enum_member' AND parent.kind = 'enum')
		  )
		  AND child.fqn = parent.fqn || '.' || child.name
	`, repoID)
	if err != nil {
		return fmt.Errorf("ts InsertContainsEdges: %w", err)
	}
	return nil
}

package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertImplementsEdges inserts heuristic IMPLEMENTS edges for Go types in the given repo.
// A struct type T IMPLEMENTS interface I when every method declared by I is also
// present in T's method set (matched by name via CONTAINS edges).
// Confidence is 0.5 (structural duck-typing heuristic). Safe to call multiple times
// (INSERT OR IGNORE). Must be called after InsertContainsEdges.
func InsertImplementsEdges(sqlDB *sql.DB, repoID int64) error {
	q := `INSERT OR IGNORE INTO edges
		(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
	SELECT DISTINCT t.repo_id, 'IMPLEMENTS', t.id, i.id, 0.5, 'heuristic', datetime('now')
	FROM symbols t
	JOIN symbols i ON i.repo_id = t.repo_id AND i.kind = 'interface' AND i.id != t.id
	WHERE t.repo_id = ?
	  AND t.lang = 'go'
	  AND t.kind = 'type'
	  -- interface must have at least one method (empty interfaces match everything — skip)
	  AND EXISTS (
	      SELECT 1 FROM edges ce
	      JOIN symbols im ON ce.dst_symbol_id = im.id
	      WHERE ce.repo_id = t.repo_id AND ce.kind = 'CONTAINS' AND ce.src_symbol_id = i.id
	        AND im.kind = 'method'
	  )
	  -- every method of the interface must be present in T's method set
	  AND NOT EXISTS (
	      SELECT 1 FROM edges ce
	      JOIN symbols im ON ce.dst_symbol_id = im.id
	      WHERE ce.repo_id = t.repo_id AND ce.kind = 'CONTAINS' AND ce.src_symbol_id = i.id
	        AND im.kind = 'method'
	        AND NOT EXISTS (
	            SELECT 1 FROM edges te
	            JOIN symbols tm ON te.dst_symbol_id = tm.id
	            WHERE te.repo_id = t.repo_id AND te.kind = 'CONTAINS' AND te.src_symbol_id = t.id
	              AND tm.kind = 'method'
	              AND tm.name = im.name
	        )
	  )`

	if _, err := sqlDB.Exec(q, repoID); err != nil {
		return fmt.Errorf("go InsertImplementsEdges: %w", err)
	}
	return nil
}

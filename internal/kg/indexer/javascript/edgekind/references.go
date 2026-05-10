package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertReferencesEdges rebuilds heuristic REFERENCES edges for JavaScript symbols.
//
// JavaScript has no static type system, so type references are detected from
// runtime patterns: `instanceof MyClass` and `new MyClass(...)`. These are stored
// as bare names in the refs table. The SQL matches using a LIKE suffix on the FQN,
// targeting class symbols only.
//
// Confidence is 0.4 / provenance=heuristic to reflect the heuristic nature.
func InsertReferencesEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE kind = 'REFERENCES'
		  AND repo_id = ?
		  AND provenance = 'heuristic'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'javascript'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("delete heuristic javascript REFERENCES edges: %w", err)
	}

	_, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges (repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT r.repo_id, 'REFERENCES', r.src_symbol_id, dst.id, 0.4, 'heuristic', datetime('now')
		FROM refs r
		JOIN symbols src ON src.id = r.src_symbol_id AND src.lang = 'javascript'
		JOIN symbols dst ON dst.repo_id = r.repo_id
		    AND dst.lang = 'javascript'
		    AND dst.kind IN ('class')
		    AND (dst.fqn = r.ref_text OR dst.fqn LIKE '%.' || r.ref_text)
		WHERE r.repo_id = ?
		  AND r.provenance = 'type-ref'
		  AND r.src_symbol_id != dst.id
	`, repoID)
	if err != nil {
		return fmt.Errorf("insert javascript REFERENCES edges: %w", err)
	}
	return nil
}

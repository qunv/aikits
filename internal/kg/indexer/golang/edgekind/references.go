package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertReferencesEdges rebuilds heuristic REFERENCES edges for Go symbols.
// Deletes existing heuristic edges, then re-inserts from the refs table using
// exact FQN matching (dst.fqn = r.ref_text) to avoid false positives.
// Confidence=0.4, provenance=heuristic; jdtls/gopls can upgrade to 1.0.
func InsertReferencesEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE kind = 'REFERENCES'
		  AND repo_id = ?
		  AND provenance = 'heuristic'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'go'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("delete heuristic go REFERENCES edges: %w", err)
	}

	_, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges (repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance)
		SELECT r.repo_id, 'REFERENCES', r.src_symbol_id, dst.id, 0.4, 'heuristic'
		FROM refs r
		JOIN symbols src ON src.id = r.src_symbol_id AND src.lang = 'go'
		JOIN symbols dst ON dst.repo_id = r.repo_id AND dst.fqn = r.ref_text
		WHERE r.repo_id = ?
		  AND r.provenance = 'type-ref'
		  AND r.src_symbol_id != dst.id
	`, repoID)
	if err != nil {
		return fmt.Errorf("insert go REFERENCES edges: %w", err)
	}
	return nil
}

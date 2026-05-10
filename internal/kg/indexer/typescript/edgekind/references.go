package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertReferencesEdges rebuilds heuristic REFERENCES edges for TypeScript symbols.
//
// TypeScript type references are stored as bare names (e.g. "KGEdge") rather than
// fully-qualified names because the extractor does not resolve imports. The SQL
// therefore matches using a LIKE suffix: `dst.fqn LIKE '%.' || r.ref_text`, which
// handles the FQN format "<dir>/<file>.<TypeName>" used by the TypeScript extractor.
//
// Cross-file references with the same bare name may produce false positives, but
// the confidence=0.4 / provenance=heuristic labels make this explicit.
func InsertReferencesEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE kind = 'REFERENCES'
		  AND repo_id = ?
		  AND provenance = 'heuristic'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'typescript'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("delete heuristic typescript REFERENCES edges: %w", err)
	}

	_, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges (repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT r.repo_id, 'REFERENCES', r.src_symbol_id, dst.id, 0.4, 'heuristic', datetime('now')
		FROM refs r
		JOIN symbols src ON src.id = r.src_symbol_id AND src.lang = 'typescript'
		JOIN symbols dst ON dst.repo_id = r.repo_id
		    AND dst.lang = 'typescript'
		    AND dst.kind IN ('interface', 'class', 'type_alias', 'enum')
		    AND (dst.fqn = r.ref_text OR dst.fqn LIKE '%.' || r.ref_text)
		WHERE r.repo_id = ?
		  AND r.provenance = 'type-ref'
		  AND r.src_symbol_id != dst.id
	`, repoID)
	if err != nil {
		return fmt.Errorf("insert typescript REFERENCES edges: %w", err)
	}
	return nil
}

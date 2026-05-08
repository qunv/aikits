package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertExtendsEdges rebuilds EXTENDS edges for Java classes/interfaces in the given repo.
// Deletes existing heuristic EXTENDS edges, then re-derives them from the refs table
// (provenance='extends-heuristic'). Confidence=0.5; jdtls can upgrade to 1.0.
func InsertExtendsEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE repo_id = ? AND kind = 'EXTENDS' AND provenance = 'heuristic'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'java'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("java delete old extends edges: %w", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT DISTINCT r.repo_id, 'EXTENDS', r.src_symbol_id, sup.id, 0.5, 'heuristic', datetime('now')
		FROM refs r
		JOIN symbols sup ON sup.repo_id = r.repo_id
		                AND sup.lang = 'java'
		                AND sup.name = r.ref_text
		                AND sup.kind IN ('class', 'interface')
		WHERE r.repo_id = ?
		  AND r.src_symbol_id IS NOT NULL
		  AND r.provenance = 'extends-heuristic'
	`, repoID); err != nil {
		return fmt.Errorf("java insert extends edges: %w", err)
	}
	return nil
}

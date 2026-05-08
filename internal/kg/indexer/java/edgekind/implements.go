package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertImplementsEdges rebuilds heuristic IMPLEMENTS edges for Java classes in the given repo.
// It uses explicit `implements` keyword refs (provenance='implements-heuristic') stored during
// extraction, matching each declared interface by simple name against known interface symbols.
//
// This replaces the old structural name-matching heuristic which produced false positives when
// multiple interfaces declared the same method name (e.g. two interfaces both declaring handle()).
//
// Confidence=0.5 (simple-name match; jdtls upgrades these to 1.0 with full type resolution).
// Note: only classes with an explicit `implements` clause in their own declaration get edges.
// Indirect interface satisfaction via superclasses is not covered here; use
// `aikits kg resolve --lang java` for complete results.
//
// Safe to call multiple times (deletes old heuristic edges before inserting).
// Must be called after symbols and implements refs have been written.
func InsertImplementsEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE repo_id = ? AND kind = 'IMPLEMENTS' AND provenance = 'heuristic'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'java'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("java delete old implements edges: %w", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT DISTINCT r.repo_id, 'IMPLEMENTS', r.src_symbol_id, iface.id, 0.5, 'heuristic', datetime('now')
		FROM refs r
		JOIN symbols iface ON iface.repo_id = r.repo_id
		                  AND iface.lang = 'java'
		                  AND iface.name = r.ref_text
		                  AND iface.kind = 'interface'
		WHERE r.repo_id = ?
		  AND r.src_symbol_id IS NOT NULL
		  AND r.provenance = 'implements-heuristic'
	`, repoID); err != nil {
		return fmt.Errorf("java insert implements edges: %w", err)
	}
	return nil
}

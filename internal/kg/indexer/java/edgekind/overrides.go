package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertOverridesEdges rebuilds heuristic OVERRIDES edges for Java methods.
// An OVERRIDES edge is inserted from method m1 to method m2 when:
//   - m1 is CONTAINed by class A
//   - m2 is CONTAINed by class B
//   - A IMPLEMENTS or EXTENDS B (heuristic or semantic edge)
//   - m1 and m2 share the same name
//
// Confidence=0.5 (name-only match; ignores overloading). jdtls upgrades to 1.0.
// Must be called after GenerateStructuralEdges (CONTAINS + IMPLEMENTS) and
// GenerateExtendsEdges (EXTENDS) have been populated.
func InsertOverridesEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE repo_id = ? AND kind = 'OVERRIDES' AND provenance = 'heuristic'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'java'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("java delete old overrides edges: %w", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT DISTINCT
			s1.repo_id, 'OVERRIDES', s1.id, s2.id, 0.5, 'heuristic', datetime('now')
		FROM symbols s1
		JOIN edges ce1 ON ce1.kind = 'CONTAINS' AND ce1.dst_symbol_id = s1.id
		JOIN symbols clsA ON clsA.id = ce1.src_symbol_id AND clsA.kind IN ('class', 'interface')
		JOIN edges ie ON (ie.kind = 'IMPLEMENTS' OR ie.kind = 'EXTENDS') AND ie.src_symbol_id = clsA.id
		JOIN symbols clsB ON clsB.id = ie.dst_symbol_id AND clsB.kind IN ('class', 'interface')
		JOIN edges ce2 ON ce2.kind = 'CONTAINS' AND ce2.src_symbol_id = clsB.id
		JOIN symbols s2 ON s2.id = ce2.dst_symbol_id AND s2.kind = 'method' AND s2.name = s1.name
		WHERE s1.repo_id = ?
		  AND s1.kind = 'method'
		  AND s1.lang = 'java'
		  AND s1.id != s2.id
	`, repoID); err != nil {
		return fmt.Errorf("java insert overrides edges: %w", err)
	}
	return nil
}

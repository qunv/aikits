package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertImportsEdges rebuilds IMPORTS edges for Java packages in the given repo.
// Deletes existing extractor-generated IMPORTS edges whose source is a Java symbol,
// then re-derives them from the refs table.
func InsertImportsEdges(sqlDB *sql.DB, repoID int64) error {
	if _, err := sqlDB.Exec(`
		DELETE FROM edges
		WHERE repo_id = ? AND kind = 'IMPORTS' AND provenance = 'extractor'
		  AND src_symbol_id IN (
		      SELECT id FROM symbols WHERE repo_id = ? AND lang = 'java'
		  )
	`, repoID, repoID); err != nil {
		return fmt.Errorf("java delete old import edges: %w", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT DISTINCT r.repo_id, 'IMPORTS', r.src_symbol_id, pkg.id, 0.8, 'extractor', datetime('now')
		FROM refs r
		JOIN symbols pkg ON pkg.repo_id = r.repo_id AND pkg.kind = 'package' AND pkg.fqn = r.ref_text
		WHERE r.repo_id = ?
		  AND r.src_symbol_id IS NOT NULL
		  AND r.provenance = 'extractor'
		  AND pkg.lang = 'java'
	`, repoID); err != nil {
		return fmt.Errorf("java insert import edges: %w", err)
	}
	return nil
}

package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertContainsEdges inserts CONTAINS edges for Go symbols in the given repo.
// Covers: concrete methods (receiver syntax), interface methods, struct fields,
// and package → type/interface containment.
// Safe to call multiple times (INSERT OR IGNORE).
func InsertContainsEdges(sqlDB *sql.DB, repoID int64) error {
	stmts := []string{
		// CONTAINS: Go concrete methods — FQN = base + '.(' + TypeName + ').' + method
		`INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT child.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols child
		JOIN symbols parent ON parent.repo_id = child.repo_id
		WHERE child.repo_id = ?
		  AND child.lang = 'go'
		  AND child.kind = 'method'
		  AND instr(child.fqn, '.(') > 0
		  AND parent.kind IN ('type', 'interface')
		  AND parent.fqn =
		      substr(child.fqn, 1, instr(child.fqn, '.(') - 1)
		      || '.'
		      || substr(
		           child.fqn,
		           instr(child.fqn, '.(') + 2,
		           instr(substr(child.fqn, instr(child.fqn, '.(') + 2), ').') - 1
		         )`,

		// CONTAINS: Go interface methods and struct fields — FQN = parent.fqn + '.' + name
		`INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT child.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols child
		JOIN symbols parent ON parent.repo_id = child.repo_id
		WHERE child.repo_id = ?
		  AND child.lang = 'go'
		  AND child.kind IN ('method', 'field')
		  AND instr(child.fqn, '.(') = 0
		  AND instr(child.fqn, '#') = 0
		  AND parent.kind IN ('type', 'interface')
		  AND parent.fqn = substr(child.fqn, 1, length(child.fqn) - length(child.name) - 1)`,

		// CONTAINS: package → type/interface (direct children only)
		`INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT parent.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols parent
		JOIN symbols child ON child.repo_id = parent.repo_id AND child.id != parent.id
		WHERE parent.repo_id = ?
		  AND parent.lang = 'go'
		  AND parent.kind = 'package'
		  AND child.kind IN ('type', 'interface')
		  AND child.fqn = parent.fqn || '.' || child.name`,
	}

	for _, q := range stmts {
		if _, err := sqlDB.Exec(q, repoID); err != nil {
			return fmt.Errorf("go InsertContainsEdges (%s...): %w", q[:20], err)
		}
	}
	return nil
}

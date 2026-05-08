package edgekind

import (
	"database/sql"
	"fmt"
)

// InsertContainsEdges inserts CONTAINS edges for Java symbols in the given repo.
// Covers: class/interface/enum fields (dot pattern), methods and constructors
// (hash pattern), and package → class/interface/enum containment.
// Safe to call multiple times (INSERT OR IGNORE).
func InsertContainsEdges(sqlDB *sql.DB, repoID int64) error {
	stmts := []string{
		// CONTAINS: Java fields — FQN = parent.fqn + '.' + name (no # delimiter)
		`INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT child.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols child
		JOIN symbols parent ON parent.repo_id = child.repo_id
		WHERE child.repo_id = ?
		  AND child.lang = 'java'
		  AND child.kind = 'field'
		  AND instr(child.fqn, '#') = 0
		  AND parent.kind IN ('class', 'interface', 'enum')
		  AND parent.fqn = substr(child.fqn, 1, length(child.fqn) - length(child.name) - 1)`,

		// CONTAINS: Java methods and constructors — FQN = parent.fqn + '#' + rest
		`INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT child.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols child
		JOIN symbols parent ON parent.repo_id = child.repo_id
		WHERE child.repo_id = ?
		  AND child.lang = 'java'
		  AND child.kind IN ('method', 'constructor')
		  AND instr(child.fqn, '#') > 0
		  AND parent.kind IN ('class', 'interface', 'enum')
		  AND parent.fqn = substr(child.fqn, 1, instr(child.fqn, '#') - 1)`,

		// CONTAINS: package → class/interface/enum (direct children only)
		`INSERT OR IGNORE INTO edges
			(repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT parent.repo_id, 'CONTAINS', parent.id, child.id, 1.0, 'extractor', datetime('now')
		FROM symbols parent
		JOIN symbols child ON child.repo_id = parent.repo_id AND child.id != parent.id
		WHERE parent.repo_id = ?
		  AND parent.lang = 'java'
		  AND parent.kind = 'package'
		  AND child.kind IN ('class', 'interface', 'enum')
		  AND child.fqn = parent.fqn || '.' || child.name`,
	}

	for _, q := range stmts {
		if _, err := sqlDB.Exec(q, repoID); err != nil {
			return fmt.Errorf("java InsertContainsEdges (%s...): %w", q[:20], err)
		}
	}
	return nil
}

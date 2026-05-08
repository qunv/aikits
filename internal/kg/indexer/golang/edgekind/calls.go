package edgekind

import (
	"aikits/internal/storage"
)

// InsertCallsEdges inserts heuristic CALLS edges for Go callsites in the given file.
// Matches callee_text to Go symbols by last component (handles "pkg.Func", "recv.Method", "Func").
// Confidence is 0.4 (syntax-only heuristic). Safe to call multiple times (INSERT OR IGNORE).
func InsertCallsEdges(q storage.Querier, repoID, fileID int64) error {
	_, err := q.Exec(`
		INSERT OR IGNORE INTO edges (repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		SELECT
			cs.repo_id,
			'CALLS',
			cs.caller_symbol_id,
			s.id,
			0.4,
			'treesitter',
			datetime('now')
		FROM callsites cs
		JOIN symbols s ON s.repo_id = cs.repo_id
		WHERE cs.file_id = ?
		  AND cs.caller_symbol_id IS NOT NULL
		  AND s.lang = 'go'
		  AND s.name = CASE
				WHEN instr(cs.callee_text, '.') > 0
					THEN substr(cs.callee_text, instr(cs.callee_text, '.') + 1)
				ELSE cs.callee_text
			END
		  AND s.kind IN ('function', 'method')
	`, fileID)
	return err
}

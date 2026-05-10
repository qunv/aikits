package edgekind

import (
	"database/sql"
	"fmt"

	"aikits/internal/storage"
)

// InsertCallsEdges inserts heuristic CALLS edges for JavaScript callsites in the given file.
// Matches callee_text to JavaScript symbols by last component (handles "obj.method", "fn").
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
		  AND s.lang = 'javascript'
		  AND s.name = CASE
				WHEN instr(cs.callee_text, '.') > 0
					THEN substr(cs.callee_text, instr(cs.callee_text, '.') + 1)
				ELSE cs.callee_text
			END
		  AND s.kind IN ('function', 'method')
	`, fileID)
	return err
}

// InsertBulkCallsEdges rebuilds all heuristic CALLS edges for JavaScript across the entire repo.
// Must be called after all file symbols and callsites have been written, so that cross-file
// callees are already in the symbols table regardless of file-processing order.
// Safe to call multiple times (INSERT OR IGNORE).
func InsertBulkCallsEdges(db *sql.DB, repoID int64) error {
	_, err := db.Exec(`
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
		JOIN symbols caller ON caller.id = cs.caller_symbol_id AND caller.lang = 'javascript'
		JOIN symbols s ON s.repo_id = cs.repo_id
		WHERE cs.repo_id = ?
		  AND cs.caller_symbol_id IS NOT NULL
		  AND s.lang = 'javascript'
		  AND s.name = CASE
				WHEN instr(cs.callee_text, '.') > 0
					THEN substr(cs.callee_text, instr(cs.callee_text, '.') + 1)
				ELSE cs.callee_text
			END
		  AND s.kind IN ('function', 'method')
	`, repoID)
	if err != nil {
		return fmt.Errorf("js InsertBulkCallsEdges: %w", err)
	}
	return nil
}

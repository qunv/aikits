package resolver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	kgdb "aikits/internal/kg/db"
	"aikits/internal/kg/query"
)

type locationResult struct {
	URI   string `json:"uri"`
	Range struct {
		Start struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"start"`
	} `json:"range"`
}

// ResolveCallsite calls textDocument/definition at the callsite position and
// returns the target symbol ID from the DB, or 0 if not resolvable.
func ResolveCallsite(client *LSPClient, sqlDB *sql.DB, repoID int64, rootPath string, cs *kgdb.CallsiteRow) (int64, error) {
	// Get the file path for this callsite
	var filePath string
	err := sqlDB.QueryRow("SELECT path FROM files WHERE id=?", cs.FileID).Scan(&filePath)
	if err != nil {
		return 0, fmt.Errorf("get file path: %w", err)
	}

	absPath := filepath.Join(rootPath, filepath.FromSlash(filePath))
	fileURI := "file://" + filepath.ToSlash(absPath)

	params := map[string]any{
		"textDocument": map[string]any{"uri": fileURI},
		"position": map[string]any{
			"line":      cs.StartLine - 1, // LSP is 0-indexed
			"character": cs.StartCol - 1,
		},
	}

	result, err := client.Call("textDocument/definition", params)
	if err != nil {
		return 0, nil // not resolvable
	}

	// Result can be Location, []Location, or null
	if string(result) == "null" || len(result) == 0 {
		return 0, nil
	}

	var locations []locationResult
	// Try array form first
	if err := json.Unmarshal(result, &locations); err != nil || len(locations) == 0 {
		// Try single location
		var loc locationResult
		if err2 := json.Unmarshal(result, &loc); err2 != nil {
			return 0, nil
		}
		locations = []locationResult{loc}
	}

	if len(locations) == 0 {
		return 0, nil
	}

	loc := locations[0]
	defURI := loc.URI
	defLine := loc.Range.Start.Line + 1 // convert back to 1-indexed

	// Convert URI to repo-relative path
	defPath := strings.TrimPrefix(defURI, "file://")
	defPath = filepath.ToSlash(defPath)
	rootSlash := filepath.ToSlash(rootPath)
	if !strings.HasPrefix(rootSlash, "/") {
		rootSlash = "/" + rootSlash
	}
	relPath := strings.TrimPrefix(defPath, rootSlash+"/")

	// Look up the symbol in our DB by file path + line
	var symbolID int64
	err = sqlDB.QueryRow(`
		SELECT s.id FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE f.repo_id=? AND f.path=? AND s.start_line <= ? AND s.end_line >= ?
		ORDER BY (s.end_line - s.start_line) ASC
		LIMIT 1
	`, repoID, relPath, defLine, defLine).Scan(&symbolID)

	if err == sql.ErrNoRows {
		// Try looking up by callee text (function name) as fallback
		parts := strings.Split(cs.CalleeText, ".")
		name := parts[len(parts)-1]
		syms, lookupErr := query.LookupByName(sqlDB, repoID, name)
		if lookupErr == nil && len(syms) > 0 {
			return syms[0].ID, nil
		}
		return 0, nil
	}
	if err != nil {
		return 0, nil
	}
	return symbolID, nil
}

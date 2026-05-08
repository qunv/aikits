package resolver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	kgdb "aikits/internal/kg/db"
)

// StartJdtls starts an Eclipse jdt.ls LSP server for the given workspace.
//
// Detection order:
//  1. JDTLS_LAUNCHER_JAR env var — path to org.eclipse.equinox.launcher_*.jar
//  2. JDTLS_HOME env var — jdtls install dir; launcher jar is globbed from plugins/
//  3. "jdtls" binary on PATH — wrapper script installed by package managers
//
// dataDir is the workspace-specific data directory (typically <repo>/.kg/jdtls-data).
// extraJvmArgs are prepended to the java invocation when using jar-form startup.
// logDir is the directory for stderr capture; empty string disables logging.
//
// After initialization the server is given JDTLS_INIT_WAIT seconds (default 5) to
// complete its initial workspace import before returning.
func StartJdtls(rootPath, dataDir string, extraJvmArgs []string, logDir string) (*LSPClient, error) {
	exe, args, err := jdtlsCommand(dataDir, extraJvmArgs)
	if err != nil {
		return nil, err
	}

	c, err := startProcess(exe, args, logDir)
	if err != nil {
		return nil, fmt.Errorf("start jdtls: %w", err)
	}

	rootURI := "file://" + filepath.ToSlash(rootPath)
	initParams := map[string]any{
		"processId": nil,
		"rootUri":   rootURI,
		"workspaceFolders": []map[string]any{
			{"uri": rootURI, "name": filepath.Base(rootPath)},
		},
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"implementation": map[string]any{"dynamicRegistration": false},
				"typeHierarchy":  map[string]any{"dynamicRegistration": false},
			},
		},
		"initializationOptions": map[string]any{
			"bundles": []string{},
			"settings": map[string]any{
				"java": map[string]any{
					"autobuild": map[string]any{"enabled": false},
				},
			},
		},
	}
	if _, err := c.Call("initialize", initParams); err != nil {
		_ = c.Shutdown()
		return nil, fmt.Errorf("jdtls initialize: %w", err)
	}
	if err := c.Notify("initialized", map[string]any{}); err != nil {
		_ = c.Shutdown()
		return nil, fmt.Errorf("jdtls initialized notify: %w", err)
	}

	// Allow jdtls to complete its workspace import before we start querying.
	// Users with large projects can override via JDTLS_INIT_WAIT (seconds).
	wait := 5 * time.Second
	if s := os.Getenv("JDTLS_INIT_WAIT"); s != "" {
		var secs int
		if _, scanErr := fmt.Sscan(s, &secs); scanErr == nil && secs >= 0 {
			wait = time.Duration(secs) * time.Second
		}
	}
	if wait > 0 {
		time.Sleep(wait)
	}

	return c, nil
}

// jdtlsCommand resolves the executable and arguments to start jdtls.
func jdtlsCommand(dataDir string, extraJvmArgs []string) (exe string, args []string, err error) {
	// 1. JDTLS_LAUNCHER_JAR — direct path to the equinox launcher jar.
	if jar := os.Getenv("JDTLS_LAUNCHER_JAR"); jar != "" {
		configDir := jdtlsConfigDir(filepath.Dir(jar))
		return javaExe(), append(jdtlsJVMFlags(jar, configDir, dataDir, extraJvmArgs)), nil
	}

	// 2. JDTLS_HOME — installation directory; find launcher jar inside plugins/.
	if home := os.Getenv("JDTLS_HOME"); home != "" {
		jar, jarErr := jdtlsFindLauncherJar(filepath.Join(home, "plugins"))
		if jarErr != nil {
			return "", nil, fmt.Errorf("JDTLS_HOME set but no launcher jar found: %w", jarErr)
		}
		configDir := jdtlsConfigDir(home)
		return javaExe(), jdtlsJVMFlags(jar, configDir, dataDir, extraJvmArgs), nil
	}

	// 3. "jdtls" wrapper script on PATH.
	path, lookErr := exec.LookPath("jdtls")
	if lookErr != nil {
		return "", nil, fmt.Errorf(
			"jdtls not found; set JDTLS_LAUNCHER_JAR, JDTLS_HOME, or install the jdtls wrapper script",
		)
	}
	return path, []string{"--data", dataDir}, nil
}

// javaExe returns the java executable name (uses JAVA_HOME if set).
func javaExe() string {
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		return filepath.Join(jh, "bin", "java")
	}
	return "java"
}

// jdtlsFindLauncherJar finds the first equinox launcher jar in pluginsDir.
func jdtlsFindLauncherJar(pluginsDir string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(pluginsDir, "org.eclipse.equinox.launcher_*.jar"))
	if err != nil || len(entries) == 0 {
		return "", fmt.Errorf("no org.eclipse.equinox.launcher_*.jar in %s", pluginsDir)
	}
	return entries[0], nil
}

// jdtlsConfigDir returns the OS-specific configuration directory.
func jdtlsConfigDir(base string) string {
	var suffix string
	switch runtime.GOOS {
	case "darwin":
		suffix = "config_mac"
	case "windows":
		suffix = "config_win"
	default:
		suffix = "config_linux"
	}
	return filepath.Join(base, suffix)
}

// jdtlsJVMFlags builds the argument list for a java jar invocation of jdtls.
func jdtlsJVMFlags(launcherJar, configDir, dataDir string, extraJvmArgs []string) []string {
	args := make([]string, 0, len(extraJvmArgs)+12)
	args = append(args, extraJvmArgs...)
	args = append(args,
		"-Declipse.application=org.eclipse.jdt.ls.core.id1",
		"-Dosgi.bundles.defaultStartLevel=4",
		"-Declipse.product=org.eclipse.jdt.ls.core.product",
		"-Dfile.encoding=UTF-8",
		"--add-modules=ALL-SYSTEM",
		"--add-opens", "java.base/java.util=ALL-UNNAMED",
		"--add-opens", "java.base/java.lang=ALL-UNNAMED",
		"-jar", launcherJar,
		"-configuration", configDir,
		"-data", dataDir,
	)
	return args
}

// ─── File open/close ─────────────────────────────────────────────────────────

// OpenFile sends textDocument/didOpen for the given file so jdtls can answer
// semantic queries about it. The file content is read from disk.
func OpenFile(client *LSPClient, absPath, lang string) error {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("open file read %s: %w", absPath, err)
	}
	uri := "file://" + filepath.ToSlash(absPath)
	return client.Notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": lang,
			"version":    1,
			"text":       string(content),
		},
	})
}

// CloseFile sends textDocument/didClose.
func CloseFile(client *LSPClient, absPath string) error {
	uri := "file://" + filepath.ToSlash(absPath)
	return client.Notify("textDocument/didClose", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
}

// ─── Semantic edge discovery ─────────────────────────────────────────────────

// symRecord holds the minimal columns needed to make LSP position requests.
type symRecord struct {
	id       int64
	fileID   int64
	startLine int
	startCol  int
	filePath  string
	kind      string
}

// FindImplementationEdges queries Java method symbols and uses
// textDocument/implementation to build OVERRIDES edges (implementor → interface method).
// Only symbols that have at least one implementation are included.
func FindImplementationEdges(client *LSPClient, sqlDB *sql.DB, repoID int64, rootPath string, budget int) ([]kgdb.EdgeRow, error) {
	syms, err := queryJavaSymbols(sqlDB, repoID, []string{"method"}, budget)
	if err != nil {
		return nil, err
	}

	var edges []kgdb.EdgeRow
	opened := make(map[string]bool)

	for i := range syms {
		sym := &syms[i]
		absPath := filepath.Join(rootPath, filepath.FromSlash(sym.filePath))

		if !opened[absPath] {
			if openErr := OpenFile(client, absPath, "java"); openErr == nil {
				opened[absPath] = true
			}
		}

		uri := "file://" + filepath.ToSlash(absPath)
		params := map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position": map[string]any{
				"line":      sym.startLine - 1,
				"character": sym.startCol - 1,
			},
		}
		result, callErr := client.Call("textDocument/implementation", params)
		if callErr != nil || string(result) == "null" || len(result) == 0 {
			continue
		}

		locs, parseErr := parseLocations(result)
		if parseErr != nil || len(locs) == 0 {
			continue
		}

		for _, loc := range locs {
			dstID, lookupErr := symbolIDAtLocation(sqlDB, repoID, rootPath, loc)
			if lookupErr != nil || dstID == 0 {
				continue
			}
			// src is the implementing method; dst is the interface/abstract method.
			edges = append(edges, kgdb.EdgeRow{
				RepoID:      repoID,
				Kind:        "OVERRIDES",
				SrcSymbolID: dstID,
				DstSymbolID: sym.id,
				Confidence:  1.0,
				Provenance:  "jdtls",
			})
		}
	}

	// Close opened files.
	for absPath := range opened {
		_ = CloseFile(client, absPath)
	}

	return edges, nil
}

// typeHierarchyItem mirrors a subset of the LSP TypeHierarchyItem.
type typeHierarchyItem struct {
	Name          string          `json:"name"`
	Kind          int             `json:"kind"` // LSP SymbolKind: 5=Class, 11=Interface
	URI           string          `json:"uri"`
	Range         lspRange        `json:"range"`
	SelectionRange lspRange       `json:"selectionRange"`
	Data          json.RawMessage `json:"data,omitempty"`
}

// FindTypeHierarchyEdges uses textDocument/prepareTypeHierarchy +
// typeHierarchy/supertypes to build EXTENDS and IMPLEMENTS edges.
func FindTypeHierarchyEdges(client *LSPClient, sqlDB *sql.DB, repoID int64, rootPath string, budget int) ([]kgdb.EdgeRow, error) {
	syms, err := queryJavaSymbols(sqlDB, repoID, []string{"class", "interface"}, budget)
	if err != nil {
		return nil, err
	}

	var edges []kgdb.EdgeRow
	opened := make(map[string]bool)

	for i := range syms {
		sym := &syms[i]
		absPath := filepath.Join(rootPath, filepath.FromSlash(sym.filePath))

		if !opened[absPath] {
			if openErr := OpenFile(client, absPath, "java"); openErr == nil {
				opened[absPath] = true
			}
		}

		uri := "file://" + filepath.ToSlash(absPath)
		prepParams := map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position": map[string]any{
				"line":      sym.startLine - 1,
				"character": sym.startCol - 1,
			},
		}
		prepResult, callErr := client.Call("textDocument/prepareTypeHierarchy", prepParams)
		if callErr != nil || string(prepResult) == "null" || len(prepResult) == 0 {
			continue
		}

		var items []typeHierarchyItem
		if err := json.Unmarshal(prepResult, &items); err != nil || len(items) == 0 {
			continue
		}

		// For each type hierarchy item, query its supertypes.
		for _, item := range items {
			superResult, superErr := client.Call("typeHierarchy/supertypes", map[string]any{"item": item})
			if superErr != nil || string(superResult) == "null" || len(superResult) == 0 {
				continue
			}

			var parents []typeHierarchyItem
			if err := json.Unmarshal(superResult, &parents); err != nil {
				continue
			}

			for _, parent := range parents {
				parentLoc := locationResult{
					URI: parent.URI,
					Range: struct {
						Start struct {
							Line      int `json:"line"`
							Character int `json:"character"`
						} `json:"start"`
					}{
						Start: struct {
							Line      int `json:"line"`
							Character int `json:"character"`
						}{
							Line:      parent.SelectionRange.Start.Line,
							Character: parent.SelectionRange.Start.Character,
						},
					},
				}
				parentID, lookupErr := symbolIDAtLocation(sqlDB, repoID, rootPath, parentLoc)
				if lookupErr != nil || parentID == 0 {
					continue
				}

				// Classify edge kind by parent type:
				//   child (class/interface) → parent class/interface:
				//   - class  → class      = EXTENDS
				//   - class  → interface  = IMPLEMENTS
				//   - interface → interface = EXTENDS
				// LSP SymbolKind: 5 = Class, 11 = Interface
				edgeKind := "EXTENDS"
				if sym.kind == "class" && parent.Kind == 11 {
					edgeKind = "IMPLEMENTS"
				}

				edges = append(edges, kgdb.EdgeRow{
					RepoID:      repoID,
					Kind:        edgeKind,
					SrcSymbolID: sym.id,
					DstSymbolID: parentID,
					Confidence:  1.0,
					Provenance:  "jdtls",
				})
			}
		}
	}

	for absPath := range opened {
		_ = CloseFile(client, absPath)
	}

	return edges, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// lspRange mirrors the LSP Range type.
type lspRange struct {
	Start struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"start"`
	End struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"end"`
}

// queryJavaSymbols fetches Java symbols of the given kinds from the DB.
func queryJavaSymbols(sqlDB *sql.DB, repoID int64, kinds []string, limit int) ([]symRecord, error) {
	placeholders := make([]string, len(kinds))
	args := make([]any, 0, len(kinds)+2)
	args = append(args, repoID)
	for i, k := range kinds {
		placeholders[i] = "?"
		args = append(args, k)
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT s.id, s.file_id, s.start_line, s.start_col, f.path, s.kind
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE f.repo_id = ? AND f.lang = 'java' AND s.kind IN (%s)
		  AND s.start_col > 0
		ORDER BY f.path, s.start_line
		LIMIT ?
	`, strings.Join(placeholders, ", "))

	rows, err := sqlDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query java symbols: %w", err)
	}
	defer rows.Close()

	var syms []symRecord
	for rows.Next() {
		var s symRecord
		if err := rows.Scan(&s.id, &s.fileID, &s.startLine, &s.startCol, &s.filePath, &s.kind); err != nil {
			return nil, err
		}
		syms = append(syms, s)
	}
	return syms, rows.Err()
}

// parseLocations unmarshals a `textDocument/definition` or `textDocument/implementation`
// result, which may be Location, []Location, or []LocationLink.
func parseLocations(raw json.RawMessage) ([]locationResult, error) {
	// Try array of locations.
	var locs []locationResult
	if err := json.Unmarshal(raw, &locs); err == nil && len(locs) > 0 {
		return locs, nil
	}
	// Try single location.
	var loc locationResult
	if err := json.Unmarshal(raw, &loc); err == nil && loc.URI != "" {
		return []locationResult{loc}, nil
	}
	// Try LocationLink array: {"targetUri":..., "targetSelectionRange":...}
	type locationLink struct {
		TargetURI            string   `json:"targetUri"`
		TargetSelectionRange lspRange `json:"targetSelectionRange"`
	}
	var links []locationLink
	if err := json.Unmarshal(raw, &links); err == nil {
		for _, l := range links {
			locs = append(locs, locationResult{
				URI: l.TargetURI,
				Range: struct {
					Start struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					} `json:"start"`
				}{
					Start: struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					}{
						Line:      l.TargetSelectionRange.Start.Line,
						Character: l.TargetSelectionRange.Start.Character,
					},
				},
			})
		}
		return locs, nil
	}
	return nil, fmt.Errorf("cannot parse location result")
}

// symbolIDAtLocation maps a Location (URI + line) back to a symbol ID in the DB.
func symbolIDAtLocation(sqlDB *sql.DB, repoID int64, rootPath string, loc locationResult) (int64, error) {
	defURI := loc.URI
	defLine := loc.Range.Start.Line + 1 // LSP is 0-indexed; DB is 1-indexed

	defPath := strings.TrimPrefix(defURI, "file://")
	defPath = filepath.ToSlash(defPath)
	rootSlash := filepath.ToSlash(rootPath)
	if !strings.HasPrefix(rootSlash, "/") {
		rootSlash = "/" + rootSlash
	}
	relPath := strings.TrimPrefix(defPath, rootSlash+"/")

	var symbolID int64
	err := sqlDB.QueryRow(`
		SELECT s.id FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE f.repo_id = ? AND f.path = ? AND s.start_line <= ? AND s.end_line >= ?
		ORDER BY (s.end_line - s.start_line) ASC
		LIMIT 1
	`, repoID, relPath, defLine, defLine).Scan(&symbolID)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	return symbolID, err
}

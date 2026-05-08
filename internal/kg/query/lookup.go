package query

import (
	"database/sql"
	"fmt"
	"strings"
)

const symbolSelectCols = `
	id, repo_id, file_id, lang, kind, name,
	COALESCE(fqn,''), COALESCE(signature,''), COALESCE(visibility,''),
	start_line, start_col, end_line, end_col, start_byte, end_byte
`

func scanSymbol(row *sql.Row) (Symbol, error) {
	var s Symbol
	err := row.Scan(
		&s.ID, &s.RepoID, &s.FileID, &s.Lang, &s.Kind, &s.Name,
		&s.FQN, &s.Signature, &s.Visibility,
		&s.StartLine, &s.StartCol, &s.EndLine, &s.EndCol, &s.StartByte, &s.EndByte,
	)
	return s, err
}

func scanSymbols(rows *sql.Rows) ([]Symbol, error) {
	var result []Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(
			&s.ID, &s.RepoID, &s.FileID, &s.Lang, &s.Kind, &s.Name,
			&s.FQN, &s.Signature, &s.Visibility,
			&s.StartLine, &s.StartCol, &s.EndLine, &s.EndCol, &s.StartByte, &s.EndByte,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// LookupByFQN returns symbols matching an exact FQN.
func LookupByFQN(db *sql.DB, repoID int64, fqn string) ([]Symbol, error) {
	rows, err := db.Query(
		"SELECT "+symbolSelectCols+" FROM symbols WHERE repo_id=? AND fqn=?",
		repoID, fqn,
	)
	if err != nil {
		return nil, fmt.Errorf("lookup by fqn: %w", err)
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// LookupByName returns symbols matching a name (partial or exact).
func LookupByName(db *sql.DB, repoID int64, name string) ([]Symbol, error) {
	// Escape SQL LIKE wildcards so user input is treated literally.
	escaped := strings.ReplaceAll(name, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "%", `\%`)
	escaped = strings.ReplaceAll(escaped, "_", `\_`)
	rows, err := db.Query(
		`SELECT `+symbolSelectCols+` FROM symbols WHERE repo_id=? AND name LIKE ? ESCAPE '\'`,
		repoID, "%"+escaped+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("lookup by name: %w", err)
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// GetSymbolByID returns a single symbol by ID.
func GetSymbolByID(db *sql.DB, id int64) (Symbol, error) {
	row := db.QueryRow("SELECT "+symbolSelectCols+" FROM symbols WHERE id=?", id)
	s, err := scanSymbol(row)
	if err != nil {
		return Symbol{}, fmt.Errorf("get symbol by id: %w", err)
	}
	return s, nil
}

// GetSymbolIDByFQN returns the ID of a symbol given its FQN, or 0 if not found.
func GetSymbolIDByFQN(db *sql.DB, repoID int64, fqn string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM symbols WHERE repo_id=? AND fqn=? LIMIT 1", repoID, fqn).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// resolveSymbolID resolves a name-or-FQN to a single symbol ID.
// It first tries exact FQN match, then falls back to name-contains match.
// Returns an error if ambiguous or not found.
func resolveSymbolID(db *sql.DB, repoID int64, nameOrFQN string) (int64, error) {
	id, err := GetSymbolIDByFQN(db, repoID, nameOrFQN)
	if err != nil {
		return 0, err
	}
	if id != 0 {
		return id, nil
	}
	// Fall back to name lookup
	syms, err := LookupByName(db, repoID, nameOrFQN)
	if err != nil {
		return 0, err
	}
	if len(syms) == 0 {
		return 0, fmt.Errorf("symbol not found: %s", nameOrFQN)
	}
	if len(syms) == 1 {
		return syms[0].ID, nil
	}
	// Prefer exact name match over partial
	var exact []Symbol
	for _, s := range syms {
		if s.Name == nameOrFQN {
			exact = append(exact, s)
		}
	}
	if len(exact) == 1 {
		return exact[0].ID, nil
	}
	if len(exact) > 1 {
		// Prefer callable kinds over packages/types
		var callable []Symbol
		for _, s := range exact {
			if s.Kind == "function" || s.Kind == "method" {
				callable = append(callable, s)
			}
		}
		if len(callable) == 1 {
			return callable[0].ID, nil
		}
		exact = callable // if still multiple, fall through to ambiguous error
		if len(exact) == 0 {
			exact = syms // restore if no callable found
		}
	}
	// Ambiguous — list options
	var fqns []string
	for _, s := range syms {
		fqns = append(fqns, s.FQN)
	}
	return 0, fmt.Errorf("ambiguous name %q; use full FQN: %s", nameOrFQN, strings.Join(fqns, ", "))
}

// Callers returns symbols that have outbound CALLS edges to the given FQN or name.
func Callers(db *sql.DB, repoID int64, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	startID, err := resolveSymbolID(db, repoID, fqn)
	if err != nil {
		return nil, err
	}
	ids, err := BFS(db, repoID, startID, depth, maxNodes, Inbound, []string{"CALLS"})
	if err != nil {
		return nil, err
	}
	return idsToNodes(db, ids)
}

// Callees returns symbols that the given FQN or name calls (outbound CALLS edges).
func Callees(db *sql.DB, repoID int64, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	startID, err := resolveSymbolID(db, repoID, fqn)
	if err != nil {
		return nil, err
	}
	ids, err := BFS(db, repoID, startID, depth, maxNodes, Outbound, []string{"CALLS"})
	if err != nil {
		return nil, err
	}
	return idsToNodes(db, ids)
}

// Impls returns symbols that implement the given interface FQN or name
// (inbound IMPLEMENTS edges → "who implements this interface?").
func Impls(db *sql.DB, repoID int64, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	startID, err := resolveSymbolID(db, repoID, fqn)
	if err != nil {
		return nil, err
	}
	ids, err := BFS(db, repoID, startID, depth, maxNodes, Inbound, []string{"IMPLEMENTS"})
	if err != nil {
		return nil, err
	}
	return idsToNodes(db, ids)
}

// Overrides returns symbols that override the given method FQN or name
// (inbound OVERRIDES edges → "who overrides this method?").
func Overrides(db *sql.DB, repoID int64, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	startID, err := resolveSymbolID(db, repoID, fqn)
	if err != nil {
		return nil, err
	}
	ids, err := BFS(db, repoID, startID, depth, maxNodes, Inbound, []string{"OVERRIDES"})
	if err != nil {
		return nil, err
	}
	return idsToNodes(db, ids)
}

func idsToNodes(db *sql.DB, ids []IDAtDepth) ([]SymbolNode, error) {
	var nodes []SymbolNode
	for _, item := range ids {
		s, err := GetSymbolByID(db, item.ID)
		if err != nil {
			continue
		}
		nodes = append(nodes, SymbolNode{Symbol: s, Depth: item.Depth})
	}
	return nodes, nil
}

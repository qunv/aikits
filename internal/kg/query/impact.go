package query

import "database/sql"

// Impact returns symbols that depend on the given FQN or name (inbound CALLS + REFERENCES edges).
func Impact(db *sql.DB, repoID int64, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	startID, err := resolveSymbolID(db, repoID, fqn)
	if err != nil {
		return nil, err
	}
	ids, err := BFS(db, repoID, startID, depth, maxNodes, Inbound, []string{"CALLS", "REFERENCES"})
	if err != nil {
		return nil, err
	}
	return idsToNodes(db, ids)
}

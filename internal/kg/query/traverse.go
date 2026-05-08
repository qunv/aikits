package query

import (
	"database/sql"
	"fmt"
)

// Direction controls BFS traversal direction over edges.
type Direction int

const (
	Inbound  Direction = iota // traverse edges where dst_symbol_id = node
	Outbound                  // traverse edges where src_symbol_id = node
)

// BFS performs a bounded breadth-first traversal over edges.
// edgeKinds: filter to only these edge kinds (empty = all).
// direction: Inbound or Outbound.
// Returns list of reachable symbol IDs with their traversal depth (excluding the start node).
func BFS(db *sql.DB, repoID, startID int64, depth, maxNodes int, direction Direction, edgeKinds []string) ([]IDAtDepth, error) {
	if depth <= 0 {
		depth = 3
	}
	if maxNodes <= 0 {
		maxNodes = 100
	}

	visited := map[int64]bool{startID: true}
	queue := []int64{startID}
	var result []IDAtDepth

	for d := 0; d < depth && len(queue) > 0 && len(result) < maxNodes; d++ {
		var nextQueue []int64
		for _, nodeID := range queue {
			neighbors, err := edgeNeighbors(db, repoID, nodeID, direction, edgeKinds)
			if err != nil {
				return nil, fmt.Errorf("BFS neighbors for %d: %w", nodeID, err)
			}
			for _, nid := range neighbors {
				if visited[nid] {
					continue
				}
				visited[nid] = true
				result = append(result, IDAtDepth{ID: nid, Depth: d + 1})
				nextQueue = append(nextQueue, nid)
				if len(result) >= maxNodes {
					break
				}
			}
			if len(result) >= maxNodes {
				break
			}
		}
		queue = nextQueue
	}
	return result, nil
}

func edgeNeighbors(db *sql.DB, repoID, nodeID int64, direction Direction, edgeKinds []string) ([]int64, error) {
	var q string
	var args []any

	if direction == Outbound {
		q = "SELECT dst_symbol_id FROM edges WHERE repo_id=? AND src_symbol_id=?"
	} else {
		q = "SELECT src_symbol_id FROM edges WHERE repo_id=? AND dst_symbol_id=?"
	}
	args = append(args, repoID, nodeID)

	if len(edgeKinds) > 0 {
		q += " AND kind IN (?" + fmt.Sprintf("%s", repeatComma(len(edgeKinds)-1)) + ")"
		for _, k := range edgeKinds {
			args = append(args, k)
		}
	}

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func repeatComma(n int) string {
	if n <= 0 {
		return ""
	}
	s := ""
	for i := 0; i < n; i++ {
		s += ",?"
	}
	return s
}

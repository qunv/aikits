package kg

import (
	"context"
	"time"

	kgdb "aikits/internal/kg/db"
)

// Status returns current statistics for the knowledge graph.
func (kg *KG) Status(_ context.Context) (*StatusResult, error) {
	files, symbols, callsites, resolved, lastIndexedStr, err := kgdb.GetRepoCounts(kg.db, kg.repo.ID)
	if err != nil {
		return nil, err
	}

	var lastIndexed *time.Time
	if lastIndexedStr != "" && lastIndexedStr != "never" {
		if t, parseErr := time.Parse(time.RFC3339, lastIndexedStr); parseErr == nil {
			lastIndexed = &t
		}
	}

	return &StatusResult{
		RepoName:    kg.repo.Name,
		RootPath:    kg.repo.RootPath,
		Files:       files,
		Symbols:     symbols,
		Callsites:   callsites,
		Resolved:    resolved,
		LastIndexed: lastIndexed,
	}, nil
}

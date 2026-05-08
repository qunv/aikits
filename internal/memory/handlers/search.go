package handlers

import (
	"aikits/internal/memory/db"
	merrors "aikits/internal/memory/errors"
	"aikits/internal/memory/services"
	"aikits/internal/memory/types"
)

const (
	defaultLimit = 5
	maxLimit     = 20
	minQueryLen  = 3
	maxQueryLen  = 500
)

// Search performs a full-text knowledge search and returns ranked results.
func Search(input *types.SearchInput) (*types.SearchResult, error) {
	if err := validateSearchInput(input); err != nil {
		return nil, err
	}

	db, err := db.Get()
	if err != nil {
		return nil, &merrors.StorageError{Msg: "failed to open db", Cause: err.Error()}
	}

	limit := clamp(input.Limit, 1, maxLimit, defaultLimit)
	ftsQuery := services.BuildFtsQuery(input.Query)

	var rows []types.RawSearchRow

	if ftsQuery == "" {
		q := services.BuildSimpleQuery(input.Scope, limit*2)
		rows, err = queryRows(db, q)
	} else {
		q := services.BuildSearchQuery(ftsQuery, input.Scope, limit*2)
		rows, err = queryRows(db, q)
		if err != nil {
			// FTS syntax error – fall back to recency sort.
			q = services.BuildSimpleQuery(input.Scope, limit*2)
			rows, err = queryRows(db, q)
		}
	}
	if err != nil {
		return nil, &merrors.StorageError{Msg: "search query failed", Cause: err.Error()}
	}

	ranked := services.RankResults(rows, services.RankContext{
		ContextTags: input.ContextTags,
		QueryScope:  input.Scope,
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return &types.SearchResult{
		Results:      ranked,
		TotalMatches: len(ranked),
		Query:        input.Query,
	}, nil
}

func queryRows(db *db.DB, q services.SearchQuery) ([]types.RawSearchRow, error) {
	sqlRows, err := db.Query(q.SQL, q.Params...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	var rows []types.RawSearchRow
	for sqlRows.Next() {
		var r types.RawSearchRow
		if err := sqlRows.Scan(&r.ID, &r.Title, &r.Content, &r.Tags, &r.Scope, &r.BM25Score); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, sqlRows.Err()
}

func validateSearchInput(in *types.SearchInput) error {
	var errs []string
	q := in.Query
	if len(q) < minQueryLen {
		errs = append(errs, "query must be at least 3 characters")
	}
	if len(q) > maxQueryLen {
		errs = append(errs, "query must be at most 500 characters")
	}
	if in.Limit < 0 {
		errs = append(errs, "limit must be a positive number")
	}
	if len(errs) > 0 {
		return &merrors.ValidationError{Msg: errs[0], Errors: errs}
	}
	return nil
}

func clamp(v, min, max, def int) int {
	if v <= 0 {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

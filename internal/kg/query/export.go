package query

import (
	"database/sql"
	"fmt"
)

// EdgeResult holds a single edge for export purposes.
type EdgeResult struct {
	ID          int64
	Kind        string
	SrcSymbolID int64
	DstSymbolID int64
	Confidence  float64
	Provenance  string
}

// IterateSymbols calls fn for each symbol in the repo, streaming rows one at a time.
// If lang is non-empty, only symbols of that language are included.
func IterateSymbols(db *sql.DB, repoID int64, lang string, fn func(Symbol) error) error {
	var rows *sql.Rows
	var err error
	if lang != "" {
		rows, err = db.Query(
			"SELECT "+symbolSelectCols+" FROM symbols WHERE repo_id=? AND lang=? ORDER BY id",
			repoID, lang,
		)
	} else {
		rows, err = db.Query(
			"SELECT "+symbolSelectCols+" FROM symbols WHERE repo_id=? ORDER BY id",
			repoID,
		)
	}
	if err != nil {
		return fmt.Errorf("iterate symbols: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Symbol
		if err := rows.Scan(
			&s.ID, &s.RepoID, &s.FileID, &s.Lang, &s.Kind, &s.Name,
			&s.FQN, &s.Signature, &s.Visibility,
			&s.StartLine, &s.StartCol, &s.EndLine, &s.EndCol, &s.StartByte, &s.EndByte,
		); err != nil {
			return fmt.Errorf("scan symbol: %w", err)
		}
		if err := fn(s); err != nil {
			return err
		}
	}
	return rows.Err()
}

// IterateEdges calls fn for each edge in the repo, streaming rows one at a time.
func IterateEdges(db *sql.DB, repoID int64, fn func(EdgeResult) error) error {
	rows, err := db.Query(
		"SELECT id, kind, src_symbol_id, dst_symbol_id, confidence, provenance FROM edges WHERE repo_id=? ORDER BY id",
		repoID,
	)
	if err != nil {
		return fmt.Errorf("iterate edges: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e EdgeResult
		if err := rows.Scan(&e.ID, &e.Kind, &e.SrcSymbolID, &e.DstSymbolID, &e.Confidence, &e.Provenance); err != nil {
			return fmt.Errorf("scan edge: %w", err)
		}
		if err := fn(e); err != nil {
			return err
		}
	}
	return rows.Err()
}

package lang

import (
	"database/sql"

	kgdb "aikits/internal/kg/db"
	goedge "aikits/internal/kg/indexer/golang/edgekind"
	javaedge "aikits/internal/kg/indexer/java/edgekind"
	jsedge "aikits/internal/kg/indexer/javascript/edgekind"
	tsedge "aikits/internal/kg/indexer/typescript/edgekind"
	"aikits/internal/storage"
)

// EdgeGenerator generates repository-wide structural and semantic edges for a single language.
// Methods that do not apply to a language must be implemented as no-ops returning nil.
type EdgeGenerator interface {
	InsertStructuralEdges(db *sql.DB, repoID int64) error
	InsertImportEdges(db *sql.DB, repoID int64) error
	InsertExtendsEdges(db *sql.DB, repoID int64) error   // no-op for Go
	InsertOverridesEdges(db *sql.DB, repoID int64) error // no-op for Go
	InsertReferencesEdges(db *sql.DB, repoID int64) error
}

// registeredEdgeGens is the ordered list of all language edge generators.
var registeredEdgeGens = []EdgeGenerator{
	goEdgeGen{},
	javaEdgeGen{},
	jsEdgeGen{},
	tsEdgeGen{},
}

// --- Go edge generator ---

type goEdgeGen struct{}

func (goEdgeGen) InsertStructuralEdges(db *sql.DB, repoID int64) error {
	if err := goedge.InsertDeclaresEdges(db, repoID); err != nil {
		return err
	}
	if err := goedge.InsertContainsEdges(db, repoID); err != nil {
		return err
	}
	return goedge.InsertImplementsEdges(db, repoID)
}

func (goEdgeGen) InsertImportEdges(db *sql.DB, repoID int64) error {
	return goedge.InsertImportsEdges(db, repoID)
}

func (goEdgeGen) InsertExtendsEdges(_ *sql.DB, _ int64) error   { return nil }
func (goEdgeGen) InsertOverridesEdges(_ *sql.DB, _ int64) error { return nil }

func (goEdgeGen) InsertReferencesEdges(db *sql.DB, repoID int64) error {
	return goedge.InsertReferencesEdges(db, repoID)
}

// --- Java edge generator ---

type javaEdgeGen struct{}

func (javaEdgeGen) InsertStructuralEdges(db *sql.DB, repoID int64) error {
	if err := javaedge.InsertDeclaresEdges(db, repoID); err != nil {
		return err
	}
	if err := javaedge.InsertContainsEdges(db, repoID); err != nil {
		return err
	}
	return javaedge.InsertImplementsEdges(db, repoID)
}

func (javaEdgeGen) InsertImportEdges(db *sql.DB, repoID int64) error {
	return javaedge.InsertImportsEdges(db, repoID)
}

func (javaEdgeGen) InsertExtendsEdges(db *sql.DB, repoID int64) error {
	return javaedge.InsertExtendsEdges(db, repoID)
}

func (javaEdgeGen) InsertOverridesEdges(db *sql.DB, repoID int64) error {
	return javaedge.InsertOverridesEdges(db, repoID)
}

func (javaEdgeGen) InsertReferencesEdges(db *sql.DB, repoID int64) error {
	return javaedge.InsertReferencesEdges(db, repoID)
}

// --- JavaScript edge generator ---

type jsEdgeGen struct{}

func (jsEdgeGen) InsertStructuralEdges(db *sql.DB, repoID int64) error {
	return jsedge.InsertContainsEdges(db, repoID)
}

func (jsEdgeGen) InsertImportEdges(_ *sql.DB, _ int64) error    { return nil }
func (jsEdgeGen) InsertExtendsEdges(_ *sql.DB, _ int64) error   { return nil }
func (jsEdgeGen) InsertOverridesEdges(_ *sql.DB, _ int64) error { return nil }
func (jsEdgeGen) InsertReferencesEdges(db *sql.DB, repoID int64) error {
	return jsedge.InsertReferencesEdges(db, repoID)
}

// --- TypeScript edge generator ---

type tsEdgeGen struct{}

func (tsEdgeGen) InsertStructuralEdges(db *sql.DB, repoID int64) error {
	return tsedge.InsertContainsEdges(db, repoID)
}

func (tsEdgeGen) InsertImportEdges(_ *sql.DB, _ int64) error    { return nil }
func (tsEdgeGen) InsertExtendsEdges(_ *sql.DB, _ int64) error   { return nil }
func (tsEdgeGen) InsertOverridesEdges(_ *sql.DB, _ int64) error { return nil }
func (tsEdgeGen) InsertReferencesEdges(db *sql.DB, repoID int64) error {
	return tsedge.InsertReferencesEdges(db, repoID)
}

// --- Per-file CALLS edge inserters ---

// goCallsEdge implements kgdb.CallsEdgeInserter for Go.
type goCallsEdge struct{}

func (goCallsEdge) InsertCallsEdges(q storage.Querier, repoID, fileID int64) error {
	return goedge.InsertCallsEdges(q, repoID, fileID)
}

// javaCallsEdge implements kgdb.CallsEdgeInserter for Java.
type javaCallsEdge struct{}

func (javaCallsEdge) InsertCallsEdges(q storage.Querier, repoID, fileID int64) error {
	return javaedge.InsertCallsEdges(q, repoID, fileID)
}

// jsCallsEdge implements kgdb.CallsEdgeInserter for JavaScript.
type jsCallsEdge struct{}

func (jsCallsEdge) InsertCallsEdges(q storage.Querier, repoID, fileID int64) error {
	return jsedge.InsertCallsEdges(q, repoID, fileID)
}

// tsCallsEdge implements kgdb.CallsEdgeInserter for TypeScript.
type tsCallsEdge struct{}

func (tsCallsEdge) InsertCallsEdges(q storage.Querier, repoID, fileID int64) error {
	return tsedge.InsertCallsEdges(q, repoID, fileID)
}

// DefaultCallsEdgeInserters returns CALLS edge inserters for all supported languages.
// Pass the result to kgdb.BatchWrite; each inserter's SQL is scoped to its own language.
func DefaultCallsEdgeInserters() []kgdb.CallsEdgeInserter {
	return []kgdb.CallsEdgeInserter{goCallsEdge{}, javaCallsEdge{}, jsCallsEdge{}, tsCallsEdge{}}
}

// --- Exported Generate* functions (replace the former kgdb.Generate* set) ---

// GenerateCallsEdges rebuilds all heuristic CALLS edges for languages that require a
// post-processing pass (JS/TS) to handle cross-file callees. Must be called after all
// BatchWrite calls so every callee symbol is already in the symbols table.
func GenerateCallsEdges(db *sql.DB, repoID int64) error {
	if err := jsedge.InsertBulkCallsEdges(db, repoID); err != nil {
		return err
	}
	return tsedge.InsertBulkCallsEdges(db, repoID)
}

// GenerateStructuralEdges inserts DECLARES, CONTAINS, and IMPLEMENTS edges for all languages.
// IMPLEMENTS depends on CONTAINS being populated first within each language pass.
func GenerateStructuralEdges(db *sql.DB, repoID int64) error {
	for _, g := range registeredEdgeGens {
		if err := g.InsertStructuralEdges(db, repoID); err != nil {
			return err
		}
	}
	return nil
}

// GenerateImportEdges rebuilds IMPORTS edges for all languages from stored import refs.
func GenerateImportEdges(db *sql.DB, repoID int64) error {
	for _, g := range registeredEdgeGens {
		if err := g.InsertImportEdges(db, repoID); err != nil {
			return err
		}
	}
	return nil
}

// GenerateExtendsEdges rebuilds EXTENDS edges (Java-only; no-op for other languages).
func GenerateExtendsEdges(db *sql.DB, repoID int64) error {
	for _, g := range registeredEdgeGens {
		if err := g.InsertExtendsEdges(db, repoID); err != nil {
			return err
		}
	}
	return nil
}

// GenerateOverridesEdges rebuilds OVERRIDES edges (Java-only; no-op for other languages).
// Must be called after GenerateStructuralEdges and GenerateExtendsEdges.
func GenerateOverridesEdges(db *sql.DB, repoID int64) error {
	for _, g := range registeredEdgeGens {
		if err := g.InsertOverridesEdges(db, repoID); err != nil {
			return err
		}
	}
	return nil
}

// GenerateReferencesEdges rebuilds REFERENCES edges for all languages from stored type refs.
func GenerateReferencesEdges(db *sql.DB, repoID int64) error {
	for _, g := range registeredEdgeGens {
		if err := g.InsertReferencesEdges(db, repoID); err != nil {
			return err
		}
	}
	return nil
}

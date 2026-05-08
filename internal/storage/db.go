package storage

import "database/sql"

// Querier abstracts the common query methods shared by *sql.DB and *sql.Tx.
// Functions that do not need to start or manage transactions should accept
// Querier so they work transparently in both contexts, making it easy to call
// them inside or outside of a transaction without duplicating code.
type Querier interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
}

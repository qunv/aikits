package kg

import (
	"errors"
	"fmt"
)

// ErrNotInitialized is returned when the .kg directory or database is missing.
var ErrNotInitialized = errors.New("kg: repository not initialized; run 'aikits kg init' first")

// ErrSchemaMismatch is returned when the database schema version does not match
// the expected version. The database must be re-initialized.
type ErrSchemaMismatch struct {
	Got  int
	Want int
}

func (e *ErrSchemaMismatch) Error() string {
	return fmt.Sprintf("kg: schema version mismatch (got %d, want %d); run 'aikits kg init --reinit'", e.Got, e.Want)
}

// ErrToolNotFound is returned when a required external tool is not found in PATH.
type ErrToolNotFound struct {
	Tool string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("kg: required tool %q not found in PATH", e.Tool)
}

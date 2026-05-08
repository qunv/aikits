package errors

import "fmt"

// DuplicateError is returned when a knowledge item with the same title or
// content already exists in the same scope.
type DuplicateError struct {
	Msg        string `json:"message"`
	ExistingID string `json:"existingId"`
	Field      string `json:"field"` // "title" or "content"
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("DUPLICATE_KNOWLEDGE: %s (existing id: %s, field: %s)", e.Msg, e.ExistingID, e.Field)
}

// NotFoundError is returned when a knowledge item is not found by ID.
type NotFoundError struct {
	Msg string `json:"message"`
	ID  string `json:"id"`
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("NOT_FOUND: %s (id: %s)", e.Msg, e.ID)
}

// ValidationError is returned when input fails validation.
type ValidationError struct {
	Msg    string   `json:"message"`
	Errors []string `json:"errors"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("VALIDATION_ERROR: %s", e.Msg)
}

// StorageError is returned on unexpected db failures.
type StorageError struct {
	Msg   string `json:"message"`
	Cause string `json:"cause,omitempty"`
}

func (e *StorageError) Error() string {
	if e.Cause != "" {
		return fmt.Sprintf("STORAGE_ERROR: %s: %s", e.Msg, e.Cause)
	}
	return fmt.Sprintf("STORAGE_ERROR: %s", e.Msg)
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"aikits/internal/memory/db"
	merrors "aikits/internal/memory/errors"
	"aikits/internal/memory/services"
	"aikits/internal/memory/types"
)

// Store validates and persists a new knowledge item.
func Store(input *types.StoreInput) (*types.StoreResult, error) {
	if err := services.ValidateStoreInput(input); err != nil {
		return nil, err
	}

	db, err := db.Get()
	if err != nil {
		return nil, &merrors.StorageError{Msg: "failed to open db", Cause: err.Error()}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	normalizedTitle := services.NormalizeTitle(input.Title)
	scope := services.NormalizeScope(input.Scope)
	tags := services.NormalizeTags(input.Tags)
	contentHash := services.HashContent(input.Content)
	id := uuid.New().String()

	tagsJSON, _ := json.Marshal(tags)

	var resultID string

	err = db.Transaction(func(tx *sql.Tx) error {
		// Dedup by normalised title within the same scope.
		var existingID string
		err := tx.QueryRow(
			`SELECT id FROM knowledge WHERE normalized_title = ? AND scope = ?`,
			normalizedTitle, scope,
		).Scan(&existingID)
		if err == nil {
			return &merrors.DuplicateError{
				Msg:        "knowledge with similar title already exists in this scope",
				ExistingID: existingID,
				Field:      "title",
			}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check title duplicate: %w", err)
		}

		// Dedup by content hash within the same scope.
		err = tx.QueryRow(
			`SELECT id FROM knowledge WHERE content_hash = ? AND scope = ?`,
			contentHash, scope,
		).Scan(&existingID)
		if err == nil {
			return &merrors.DuplicateError{
				Msg:        "knowledge with identical content already exists in this scope",
				ExistingID: existingID,
				Field:      "content",
			}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check content duplicate: %w", err)
		}

		_, err = tx.Exec(
			`INSERT INTO knowledge
				(id, title, content, tags, scope, normalized_title, content_hash, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id,
			strings.TrimSpace(input.Title),
			strings.TrimSpace(input.Content),
			string(tagsJSON),
			scope,
			normalizedTitle,
			contentHash,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert knowledge: %w", err)
		}

		resultID = id
		return nil
	})

	if err != nil {
		var dupErr *merrors.DuplicateError
		if ok := isType(err, &dupErr); ok {
			return nil, dupErr
		}
		return nil, &merrors.StorageError{Msg: "failed to store knowledge", Cause: err.Error()}
	}

	return &types.StoreResult{Success: true, ID: resultID, Message: "knowledge stored successfully"}, nil
}

// isType performs a type assertion and sets target if err matches.
func isType[T error](err error, target *T) bool {
	if t, ok := err.(T); ok {
		*target = t
		return true
	}
	return false
}

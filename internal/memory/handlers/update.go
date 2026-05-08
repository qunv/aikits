package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aikits/internal/memory/db"
	merrors "aikits/internal/memory/errors"
	"aikits/internal/memory/services"
	"aikits/internal/memory/types"
)

type knowledgeRow struct {
	ID              string
	Title           string
	Content         string
	Tags            string
	Scope           string
	NormalizedTitle string
	ContentHash     string
}

// Update modifies an existing knowledge item identified by its ID.
func Update(input *types.UpdateInput) (*types.UpdateResult, error) {
	if err := services.ValidateUpdateInput(input); err != nil {
		return nil, err
	}

	db, err := db.Get()
	if err != nil {
		return nil, &merrors.StorageError{Msg: "failed to open db", Cause: err.Error()}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	err = db.Transaction(func(tx *sql.Tx) error {
		var existing knowledgeRow
		err := tx.QueryRow(
			`SELECT id, title, content, tags, scope, normalized_title, content_hash
			 FROM knowledge WHERE id = ?`,
			input.ID,
		).Scan(
			&existing.ID, &existing.Title, &existing.Content,
			&existing.Tags, &existing.Scope,
			&existing.NormalizedTitle, &existing.ContentHash,
		)
		if err == sql.ErrNoRows {
			return &merrors.NotFoundError{Msg: "knowledge item not found", ID: input.ID}
		} else if err != nil {
			return fmt.Errorf("fetch existing: %w", err)
		}

		// Resolve final values (use existing when field not provided).
		title := existing.Title
		if input.Title != nil {
			title = strings.TrimSpace(*input.Title)
		}

		content := existing.Content
		if input.Content != nil {
			content = strings.TrimSpace(*input.Content)
		}

		scope := existing.Scope
		if input.Scope != nil {
			scope = services.NormalizeScope(*input.Scope)
		}

		var tags []string
		if input.TagsProvided {
			tags = services.NormalizeTags(input.Tags)
		} else {
			_ = json.Unmarshal([]byte(existing.Tags), &tags)
		}

		normalizedTitle := services.NormalizeTitle(title)
		contentHash := services.HashContent(content)
		tagsJSON, _ := json.Marshal(tags)

		// Dedup checks (exclude self).
		var conflictID string
		err = tx.QueryRow(
			`SELECT id FROM knowledge WHERE normalized_title = ? AND scope = ? AND id != ?`,
			normalizedTitle, scope, input.ID,
		).Scan(&conflictID)
		if err == nil {
			return &merrors.DuplicateError{
				Msg:        "knowledge with similar title already exists in this scope",
				ExistingID: conflictID,
				Field:      "title",
			}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check title duplicate: %w", err)
		}

		err = tx.QueryRow(
			`SELECT id FROM knowledge WHERE content_hash = ? AND scope = ? AND id != ?`,
			contentHash, scope, input.ID,
		).Scan(&conflictID)
		if err == nil {
			return &merrors.DuplicateError{
				Msg:        "knowledge with identical content already exists in this scope",
				ExistingID: conflictID,
				Field:      "content",
			}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check content duplicate: %w", err)
		}

		_, err = tx.Exec(
			`UPDATE knowledge
			 SET title = ?, content = ?, tags = ?, scope = ?,
			     normalized_title = ?, content_hash = ?, updated_at = ?
			 WHERE id = ?`,
			title, content, string(tagsJSON), scope,
			normalizedTitle, contentHash, now,
			input.ID,
		)
		return err
	})

	if err != nil {
		var dupErr *merrors.DuplicateError
		if isType(err, &dupErr) {
			return nil, dupErr
		}
		var nfErr *merrors.NotFoundError
		if isType(err, &nfErr) {
			return nil, nfErr
		}
		return nil, &merrors.StorageError{Msg: "failed to update knowledge", Cause: err.Error()}
	}

	return &types.UpdateResult{Success: true, ID: input.ID, Message: "knowledge updated successfully"}, nil
}

package services

import (
	"regexp"
	"strings"

	merrors "aikits/internal/memory/errors"
	"aikits/internal/memory/types"
)

const (
	titleMin   = 10
	titleMax   = 100
	contentMin = 50
	contentMax = 5000
	tagsMax    = 10
)

var (
	scopePattern = regexp.MustCompile(`(?i)^(global|project:[a-z0-9_-]+|repo:[a-z0-9_-]+)$`)
	tagPattern   = regexp.MustCompile(`(?i)^[a-z0-9][a-z0-9-]*$`)

	genericPhrases = []string{
		"this is important",
		"remember this",
		"note to self",
		"todo",
		"fix this",
		"do this",
		"always do",
		"never do",
	}
)

// ValidateStoreInput validates input before storing a knowledge item.
func ValidateStoreInput(in *types.StoreInput) error {
	var errs []string
	errs = append(errs, validateTitle(in.Title)...)
	errs = append(errs, validateContent(in.Content)...)
	errs = append(errs, validateTags(in.Tags)...)
	errs = append(errs, validateScope(in.Scope)...)
	if len(errs) > 0 {
		return &merrors.ValidationError{Msg: strings.Join(errs, "; "), Errors: errs}
	}
	return nil
}

// ValidateUpdateInput validates input before updating a knowledge item.
func ValidateUpdateInput(in *types.UpdateInput) error {
	var errs []string

	if strings.TrimSpace(in.ID) == "" {
		errs = append(errs, "id is required")
	}

	hasField := in.Title != nil || in.Content != nil || in.TagsProvided || in.Scope != nil
	if !hasField {
		errs = append(errs, "at least one of title, content, tags, or scope must be provided")
	}

	if in.Title != nil {
		errs = append(errs, validateTitle(*in.Title)...)
	}
	if in.Content != nil {
		errs = append(errs, validateContent(*in.Content)...)
	}
	if in.TagsProvided {
		errs = append(errs, validateTags(in.Tags)...)
	}
	if in.Scope != nil {
		errs = append(errs, validateScope(*in.Scope)...)
	}

	if len(errs) > 0 {
		return &merrors.ValidationError{Msg: strings.Join(errs, "; "), Errors: errs}
	}
	return nil
}

func validateTitle(title string) []string {
	var errs []string
	t := strings.TrimSpace(title)
	if t == "" {
		return []string{"title is required"}
	}
	if len(t) < titleMin {
		errs = append(errs, "title must be at least 10 characters")
	}
	if len(t) > titleMax {
		errs = append(errs, "title must be at most 100 characters")
	}
	return errs
}

func validateContent(content string) []string {
	var errs []string
	c := strings.TrimSpace(content)
	if c == "" {
		return []string{"content is required"}
	}
	if len(c) < contentMin {
		errs = append(errs, "content must be at least 50 characters")
	}
	if len(c) > contentMax {
		errs = append(errs, "content must be at most 5000 characters")
	}
	lower := strings.ToLower(c)
	for _, phrase := range genericPhrases {
		if lower == phrase || strings.HasPrefix(lower, phrase+" ") {
			errs = append(errs, "content appears too generic; provide specific, actionable knowledge")
			break
		}
	}
	return errs
}

func validateTags(tags []string) []string {
	var errs []string
	if len(tags) > tagsMax {
		errs = append(errs, "maximum 10 tags allowed")
	}
	for _, t := range tags {
		if !tagPattern.MatchString(t) {
			errs = append(errs, "invalid tag \""+t+"\": tags must be alphanumeric with hyphens")
		}
	}
	return errs
}

func validateScope(scope string) []string {
	if scope == "" || scope == "global" {
		return nil
	}
	if !scopePattern.MatchString(scope) {
		return []string{`invalid scope: must be "global", "project:<name>", or "repo:<name>"`}
	}
	return nil
}

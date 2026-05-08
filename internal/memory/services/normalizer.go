package services

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// NormalizeTitle lowercases and collapses whitespace for deduplication.
func NormalizeTitle(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(title))), " ")
}

// NormalizeContent trims and normalises line endings for consistent hashing.
func NormalizeContent(content string) string {
	s := strings.TrimSpace(content)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Collapse runs of 3+ blank lines to two.
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

// HashContent returns a SHA-256 hex digest of the normalised content.
func HashContent(content string) string {
	normalised := NormalizeContent(content)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(normalised)))
}

// NormalizeTags lowercases, trims, deduplicates, and drops empty tags.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}

// NormalizeScope returns a canonical scope string, defaulting to "global".
func NormalizeScope(scope string) string {
	s := strings.ToLower(strings.TrimSpace(scope))
	if s == "" {
		return "global"
	}
	return s
}

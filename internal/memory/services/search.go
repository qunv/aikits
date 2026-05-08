package services

import (
	"regexp"
	"strings"
)

var ftsSpecial = regexp.MustCompile(`[*^():\-]`)
var booleanOp = regexp.MustCompile(`(?i)\b(AND|OR|NOT)\b`)

// BuildFtsQuery converts a natural-language query string into an FTS5
// MATCH expression using prefix matching.
func BuildFtsQuery(query string) string {
	// Escape double quotes by doubling them, then remove other FTS operators.
	escaped := strings.ReplaceAll(query, `"`, `""`)
	escaped = ftsSpecial.ReplaceAllString(escaped, " ")
	escaped = booleanOp.ReplaceAllString(escaped, "")
	escaped = strings.TrimSpace(escaped)
	// Collapse multiple spaces.
	escaped = strings.Join(strings.Fields(escaped), " ")

	words := strings.Fields(escaped)
	if len(words) == 0 {
		return ""
	}
	// Prefix-match each word for partial matching.
	prefixed := make([]string, len(words))
	for i, w := range words {
		prefixed[i] = w + "*"
	}
	return strings.Join(prefixed, " ")
}

// SearchQuery holds a parameterised SQL query and its bind values.
type SearchQuery struct {
	SQL    string
	Params []any
}

// BuildSearchQuery returns a full-text search query with BM25 column weights.
// title=10, content=5, tags=1
func BuildSearchQuery(ftsQuery, scope string, limit int) SearchQuery {
	var sb strings.Builder
	params := make([]any, 0, 3)

	sb.WriteString(`
SELECT k.id, k.title, k.content, k.tags, k.scope,
       bm25(knowledge_fts, 10.0, 5.0, 1.0) AS bm25_score
FROM   knowledge k
JOIN   knowledge_fts fts ON k.rowid = fts.rowid
WHERE  knowledge_fts MATCH ?`)
	params = append(params, ftsQuery)

	if scope != "" {
		sb.WriteString(` AND (k.scope = ? OR k.scope = 'global')`)
		params = append(params, scope)
	}

	sb.WriteString(` ORDER BY bm25_score LIMIT ?`)
	params = append(params, limit)

	return SearchQuery{SQL: sb.String(), Params: params}
}

// BuildSimpleQuery returns a recency-ordered fallback query (no FTS).
func BuildSimpleQuery(scope string, limit int) SearchQuery {
	var sb strings.Builder
	params := make([]any, 0, 2)

	sb.WriteString(`
SELECT id, title, content, tags, scope,
       0 AS bm25_score
FROM   knowledge`)

	if scope != "" {
		sb.WriteString(` WHERE scope = ? OR scope = 'global'`)
		params = append(params, scope)
	}

	sb.WriteString(` ORDER BY created_at DESC LIMIT ?`)
	params = append(params, limit)

	return SearchQuery{SQL: sb.String(), Params: params}
}

package types

// KnowledgeItem is the full representation of a stored knowledge entry.
type KnowledgeItem struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	Tags            []string `json:"tags"`
	Scope           string   `json:"scope"`
	NormalizedTitle string   `json:"normalizedTitle"`
	ContentHash     string   `json:"contentHash"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

// StoreInput is the input for storing a new knowledge item.
type StoreInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
	Scope   string   `json:"scope,omitempty"`
}

// StoreResult is the result of a store operation.
type StoreResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message"`
}

// UpdateInput is the input for updating an existing knowledge item.
// Pointer fields distinguish "not provided" from "empty string".
type UpdateInput struct {
	ID           string
	Title        *string
	Content      *string
	Tags         []string
	TagsProvided bool // true when caller explicitly passed tags (even empty slice)
	Scope        *string
}

// UpdateResult is the result of an update operation.
type UpdateResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// SearchInput is the input for searching knowledge.
type SearchInput struct {
	Query       string   `json:"query"`
	ContextTags []string `json:"contextTags,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}

// SearchResultItem is a single ranked search result.
type SearchResultItem struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Scope   string   `json:"scope"`
	Score   float64  `json:"score"`
}

// SearchResult is the result of a search operation.
type SearchResult struct {
	Results      []SearchResultItem `json:"results"`
	TotalMatches int                `json:"totalMatches"`
	Query        string             `json:"query"`
}

// RawSearchRow is the raw row returned from an FTS query.
type RawSearchRow struct {
	ID        string
	Title     string
	Content   string
	Tags      string // JSON-encoded array
	Scope     string
	BM25Score float64
}

package query

// Symbol represents a symbol from the knowledge graph.
type Symbol struct {
	ID         int64
	RepoID     int64
	FileID     int64
	Lang       string
	Kind       string
	Name       string
	FQN        string
	Signature  string
	Visibility string
	StartLine  int
	StartCol   int
	EndLine    int
	EndCol     int
	StartByte  int
	EndByte    int
}

// IDAtDepth is a symbol ID paired with its BFS traversal depth.
type IDAtDepth struct {
	ID    int64
	Depth int
}

// SymbolNode wraps a Symbol with its traversal depth.
type SymbolNode struct {
	Symbol Symbol
	Depth  int
}

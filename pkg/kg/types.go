package kg

import "time"

// Lang represents a supported programming language for indexing/resolution.
type Lang string

const (
	LangAll        Lang = ""
	LangGo         Lang = "go"
	LangJava       Lang = "java"
	LangJavaScript Lang = "javascript"
	LangHTML       Lang = "html"
	LangCSS        Lang = "css"
)

// ExportFormat is the output format for graph export.
type ExportFormat string

const (
	FormatJSON    ExportFormat = "json"
	FormatGraphML ExportFormat = "graphml"
)

// Symbol is a code symbol in the knowledge graph.
type Symbol struct {
	ID         int64
	Lang       string
	Kind       string
	Name       string
	FQN        string
	Signature  string
	Visibility string
	StartLine  int
}

// SymbolNode is a Symbol with its BFS traversal depth.
type SymbolNode struct {
	Symbol Symbol
	Depth  int
}

// StatusResult holds the current state of the knowledge graph.
type StatusResult struct {
	RepoName    string
	RootPath    string
	Files       int64
	Symbols     int64
	Callsites   int64
	Resolved    int64
	LastIndexed *time.Time
}

// IndexOptions controls the behavior of Index.
type IndexOptions struct {
	// Full ignores the file cache and reindexes everything.
	Full bool
	// Lang restricts indexing to specific languages. Zero value means all.
	Lang []Lang
	// Jobs is the number of parallel workers. 0 means use NumCPU.
	Jobs int
}

// IndexResult summarizes the outcome of an Index call.
type IndexResult struct {
	Indexed   int
	Unchanged int
	Symbols   int
	Callsites int
	Errors    int
}

// ResolveOptions controls the behavior of Resolve.
type ResolveOptions struct {
	Lang              Lang
	Budget            int
	MavenDownloadDeps bool
}

// TimeFormat is the format used for LastIndexed timestamps in StatusResult.
const TimeFormat = time.RFC3339

// GraphNode is a node in the knowledge graph for UI display.
type GraphNode struct {
	ID         int64  `json:"id"`
	Lang       string `json:"lang"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	FQN        string `json:"fqn"`
	Signature  string `json:"signature"`
	Visibility string `json:"visibility"`
	StartLine  int    `json:"startLine"`
}

// GraphEdge is a directed edge in the knowledge graph for UI display.
type GraphEdge struct {
	ID         int64   `json:"id"`
	Kind       string  `json:"kind"`
	Src        int64   `json:"src"`
	Dst        int64   `json:"dst"`
	Confidence float64 `json:"confidence"`
	Provenance string  `json:"provenance"`
}

// GraphData holds all nodes and edges of the knowledge graph.
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ExportOptions controls the behavior of Export.
type ExportOptions struct {
	Format ExportFormat
	// Output is the file path to write to. Empty string uses the default
	// location: <repoRoot>/.kg/kg.<format>.
	Output string
	// Lang restricts export to a specific language. Zero value means all.
	Lang Lang
}

package graph

// Node represents a generic node in FalkorDB.
type Node struct {
	ID         string
	Label      string            // e.g. "Person", "Commit", "File", "Decision"
	Properties map[string]string // Key-value attributes
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	FromLabel string
	FromID    string
	Relation  string // e.g. "TOUCHED", "OWNS", "APPLIES_TO", "SUPERSEDES"
	ToLabel   string
	ToID      string
}

// QueryResult holds targeted query output returned to the user or agent.
type QueryResult struct {
	Headers []string
	Rows    [][]string
}

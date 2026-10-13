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

// DualOverviewStats holds repository statistics across both git.lddb and sage.lddb.
type DualOverviewStats struct {
	// Git Graph (git.lddb)
	GitCommits    int
	GitPeople     int
	GitFiles      int
	GitLastSynced string

	// AI Mind Graph (sage.lddb)
	MindFiles      int
	MindDecisions  int
	MindRoles      int
	MindLastSynced string
}


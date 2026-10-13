package graph

// QueryResult holds targeted query output returned to the user or agent.
type QueryResult struct {
	Headers []string
	Rows    [][]string
}

// DualOverviewStats holds repository statistics across both Git History and AI Mind in SQLite.
type DualOverviewStats struct {
	// Git History
	GitCommits    int
	GitPeople     int
	GitFiles      int
	GitLastSynced string

	// AI Mind Knowledge
	MindFiles      int
	MindDecisions  int
	MindRoles      int
	MindLastSynced string
}

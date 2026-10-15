package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
)

func TestSQLiteEngineLifecycleAndOperations(t *testing.T) {
	tmpDir := t.TempDir()
	eng, err := NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create sqlite engine: %v", err)
	}
	defer eng.Close()

	// 1. Verify single file exists
	dbFile := filepath.Join(tmpDir, "sage.db")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		t.Fatalf("expected single database file at %s, but not found", dbFile)
	}

	// 2. Metadata
	if err := eng.SetGitLastSyncedCommit("abc1234"); err != nil {
		t.Fatalf("set git last synced commit: %v", err)
	}
	sha, err := eng.GetGitLastSyncedCommit()
	if err != nil || sha != "abc1234" {
		t.Fatalf("expected abc1234, got %s (err: %v)", sha, err)
	}

	if err := eng.SetMindLastSyncedCommit("def5678"); err != nil {
		t.Fatalf("set mind last synced commit: %v", err)
	}
	sha, err = eng.GetMindLastSyncedCommit()
	if err != nil || sha != "def5678" {
		t.Fatalf("expected def5678, got %s (err: %v)", sha, err)
	}

	// 3. Git Ingestion
	commits := []git.Commit{
		{
			SHA:         "c123456789",
			Message:     "Initial commit",
			AuthorName:  "Alice",
			AuthorEmail: "alice@example.com",
			Timestamp:   time.Now(),
			Files:       []string{"main.go", "pkg/util.go"},
		},
	}
	expertise := map[string][]git.ContributorExpertise{
		"main.go": {
			{
				AuthorName:  "Alice",
				AuthorEmail: "alice@example.com",
				Score:       10.5,
				CommitCount: 1,
			},
		},
	}
	if err := eng.IngestGitHistory(commits, expertise); err != nil {
		t.Fatalf("ingest git history: %v", err)
	}

	// 4. CODEOWNERS Ingestion
	codeowners := []git.OwnershipRule{
		{Pattern: "main.go", Owners: []string{"@alice"}},
		{Pattern: "pkg/*", Owners: []string{"@bob"}},
	}
	if err := eng.IngestCodeowners(codeowners); err != nil {
		t.Fatalf("ingest codeowners: %v", err)
	}

	owner, experts, err := eng.QueryOwner("main.go")
	if err != nil {
		t.Fatalf("query owner: %v", err)
	}
	if owner != "@alice" {
		t.Errorf("expected owner @alice, got %s", owner)
	}
	if len(experts) == 0 {
		t.Errorf("expected at least 1 expert, got 0")
	}

	// 5. AI File Analysis
	analyses := []ai.FileAnalysis{
		{
			Path:         "main.go",
			Summary:      "Application entry point",
			Role:         "CLI Handler",
			Exports:      []string{"main"},
			Dependencies: []string{"fmt", "os"},
			Hash:         "hash123",
			UpdatedAt:    time.Now(),
		},
	}
	if err := eng.IngestFileAnalysis(analyses); err != nil {
		t.Fatalf("ingest file analysis: %v", err)
	}

	hash, err := eng.GetStoredFileHash("main.go")
	if err != nil || hash != "hash123" {
		t.Errorf("expected hash123, got %s (err: %v)", hash, err)
	}

	analysis, err := eng.GetFileAnalysis("main.go")
	if err != nil || analysis == nil || analysis.Role != "CLI Handler" {
		t.Errorf("unexpected analysis: %+v (err: %v)", analysis, err)
	}

	// 6. Decisions Ingestion
	prs := []github.PullRequest{
		{Number: 101, Title: "Refactor storage", URL: "http://github.com/pull/101", Author: "alice"},
	}
	decisions := []ai.Decision{
		{
			ID:        "DEC-101",
			Title:     "Adopt SQLite",
			Rationale: "Single file and zero external shared library friction",
			AppliesTo: []string{"main.go"},
		},
	}
	if err := eng.IngestDecisions(prs, decisions); err != nil {
		t.Fatalf("ingest decisions: %v", err)
	}

	// 7. QueryWhy
	whyResults, err := eng.QueryWhy("main.go")
	if err != nil {
		t.Fatalf("query why: %v", err)
	}
	if len(whyResults) == 0 {
		t.Errorf("expected QueryWhy results, got none")
	}

	// 8. Overview Stats
	overview, err := eng.QueryOverview()
	if err != nil {
		t.Fatalf("query overview: %v", err)
	}
	if overview["Commits"] != 1 || overview["People"] != 1 {
		t.Errorf("unexpected overview stats: %+v", overview)
	}

	// 9. Raw SQL Execution
	rawRows, err := eng.ExecuteRawSQL("SELECT count(*) FROM commits;")
	if err != nil {
		t.Fatalf("execute raw sql: %v", err)
	}
	if len(rawRows) < 2 || rawRows[1][0] != "1" {
		t.Errorf("unexpected raw sql result: %+v", rawRows)
	}

	// 10. Verify invalid SQL returns error and does NOT panic
	_, err = eng.ExecuteRawSQL("SELECT * FROM non_existent_table;")
	if err == nil {
		t.Errorf("expected error executing invalid query, got nil")
	}
}

func TestEngine_QueryArchitectureGraph(t *testing.T) {
	tmpDir := t.TempDir()
	eng, err := NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create sqlite engine: %v", err)
	}
	defer eng.Close()

	// 1. Empty database returns valid empty graph
	emptyGraph, err := eng.QueryArchitectureGraph()
	if err != nil {
		t.Fatalf("query architecture graph on empty DB failed: %v", err)
	}
	if emptyGraph == nil || emptyGraph.TotalFiles != 0 || len(emptyGraph.Clusters) != 0 {
		t.Errorf("expected empty graph for fresh DB, got: %+v", emptyGraph)
	}

	// 2. Ingest file analyses across domains
	analyses := []ai.FileAnalysis{
		{
			Path:         "cmd/sage/main.go",
			Role:         "CLI Command",
			Summary:      "CLI entry point",
			Exports:      []string{"main"},
			Dependencies: []string{"sage/internal/graph"},
			Hash:         "h1",
			UpdatedAt:    time.Now(),
		},
		{
			Path:         "cmd/sage/cli/root.go",
			Role:         "CLI Command",
			Summary:      "Root command setup",
			Exports:      []string{"Execute"},
			Dependencies: []string{"sage/cmd/sage", "sage/internal/graph"},
			Hash:         "h1b",
			UpdatedAt:    time.Now(),
		},
		{
			Path:         "internal/graph/sqlite.go",
			Role:         "Knowledge Graph Storage",
			Summary:      "SQLite backend",
			Exports:      []string{"NewEngine"},
			Dependencies: []string{"sage/internal/git"},
			Hash:         "h2",
			UpdatedAt:    time.Now(),
		},
		{
			Path:         "internal/git/git.go",
			Role:         "Git Extractor",
			Summary:      "Git parsing",
			Exports:      []string{"ExtractCommits"},
			Dependencies: []string{},
			Hash:         "h3",
			UpdatedAt:    time.Now(),
		},
	}
	if err := eng.IngestFileAnalysis(analyses); err != nil {
		t.Fatalf("ingest file analysis: %v", err)
	}

	// 3. Query Architecture Graph
	archGraph, err := eng.QueryArchitectureGraph()
	if err != nil {
		t.Fatalf("failed to query architecture graph: %v", err)
	}
	if archGraph == nil {
		t.Fatalf("architecture graph is nil")
	}

	if archGraph.TotalFiles != 4 {
		t.Errorf("expected 4 total files, got %d", archGraph.TotalFiles)
	}

	if len(archGraph.Clusters) != 3 {
		t.Errorf("expected 3 clusters, got %d", len(archGraph.Clusters))
	}

	// Verify node details are preserved
	var cliCluster *ArchitectureCluster
	for i := range archGraph.Clusters {
		if archGraph.Clusters[i].ID == ClusterCLI {
			cliCluster = &archGraph.Clusters[i]
			break
		}
	}
	if cliCluster == nil {
		t.Fatalf("CLI cluster not found in graph")
	}
	if cliCluster.FileCount != 2 {
		t.Errorf("expected 2 files in CLI cluster, got %d", cliCluster.FileCount)
	}
	firstNode := cliCluster.Nodes[0]
	if firstNode.Summary == "" || firstNode.Role == "" || len(firstNode.Exports) == 0 {
		t.Errorf("expected populated node details, got: %+v", firstNode)
	}

	// Verify edges: CLI->Graph has weight 2, Graph->Git has weight 1; no self-loops
	if len(archGraph.Edges) != 2 {
		t.Errorf("expected 2 inter-cluster edges, got %d", len(archGraph.Edges))
	}

	edgeMap := make(map[string]int)
	for _, edge := range archGraph.Edges {
		if edge.Source == edge.Target {
			t.Errorf("unexpected self-loop edge: %+v", edge)
		}
		edgeMap[string(edge.Source)+"->"+string(edge.Target)] = edge.Weight
	}
	if edgeMap["cli->graph"] != 2 {
		t.Errorf("expected edge cli->graph with weight 2, got %d", edgeMap["cli->graph"])
	}
	if edgeMap["graph->git"] != 1 {
		t.Errorf("expected edge graph->git with weight 1, got %d", edgeMap["graph->git"])
	}

	// Verify JSON serialization round-trip
	graphJSON, err := json.Marshal(archGraph)
	if err != nil {
		t.Fatalf("failed to marshal ArchitectureGraph to JSON: %v", err)
	}
	if len(graphJSON) == 0 {
		t.Errorf("expected non-empty JSON serialization")
	}
}



package graph

import (
	"testing"
)

func TestBuildArchitectureGraph_ClusteringAndEdges(t *testing.T) {
	files := []StoredFileRecord{
		{
			Path:         "cmd/sage/main.go",
			Role:         "CLI Command",
			Summary:      "Entry point for sage CLI",
			Exports:      []string{"main"},
			Dependencies: []string{"sage/cmd/sage/cli", "sage/internal/graph"},
		},
		{
			Path:         "cmd/sage/cli/root.go",
			Role:         "CLI Command",
			Summary:      "Root command setup",
			Exports:      []string{"Execute"},
			Dependencies: []string{"sage/internal/graph", "sage/internal/config"},
		},
		{
			Path:         "internal/graph/sqlite.go",
			Role:         "Knowledge Graph Storage",
			Summary:      "SQLite graph database engine",
			Exports:      []string{"NewEngine", "QueryArchitectureGraph"},
			Dependencies: []string{"sage/internal/git", "sage/internal/ai"},
		},
		{
			Path:         "internal/git/git.go",
			Role:         "Git Extractor",
			Summary:      "Git repository history extractor",
			Exports:      []string{"ExtractCommits"},
			Dependencies: []string{},
		},
		{
			Path:         "internal/ai/ai.go",
			Role:         "AI Synthesis Engine",
			Summary:      "AI reasoning and synthesis",
			Exports:      []string{"Synthesize"},
			Dependencies: []string{"sage/internal/graph"},
		},
		{
			Path:         "internal/config/config.go",
			Role:         "Configuration",
			Summary:      "App configuration",
			Exports:      []string{"Load"},
			Dependencies: []string{},
		},
		{
			Path:         "internal/graph/sqlite_test.go",
			Role:         "Automated Test Suite",
			Summary:      "Tests for sqlite engine",
			Exports:      []string{},
			Dependencies: []string{"sage/internal/graph"},
		},
	}

	graph := BuildArchitectureGraph(files)
	if graph == nil {
		t.Fatalf("expected non-nil architecture graph")
	}

	if graph.TotalFiles != 7 {
		t.Errorf("expected 7 total files, got %d", graph.TotalFiles)
	}

	// Verify expected clusters exist
	clusterMap := make(map[ClusterID]ArchitectureCluster)
	for _, c := range graph.Clusters {
		clusterMap[c.ID] = c
	}

	expectedClusterIDs := []ClusterID{ClusterCLI, ClusterGraph, ClusterGit, ClusterAI, ClusterConfig, ClusterTesting}
	for _, id := range expectedClusterIDs {
		c, ok := clusterMap[id]
		if !ok {
			t.Errorf("expected cluster %q in graph, but was not found", id)
			continue
		}
		if len(c.Nodes) == 0 {
			t.Errorf("expected cluster %q to have nodes, got 0", id)
		}
		if c.FileCount != len(c.Nodes) {
			t.Errorf("expected cluster %q FileCount=%d to match len(Nodes)=%d", id, c.FileCount, len(c.Nodes))
		}
	}

	// Verify CLI cluster has cmd/sage/main.go and cmd/sage/cli/root.go
	cliCluster := clusterMap[ClusterCLI]
	if cliCluster.FileCount != 2 {
		t.Errorf("expected 2 files in CLI cluster, got %d", cliCluster.FileCount)
	}

	// Verify Edges
	edgeMap := make(map[string]int)
	for _, edge := range graph.Edges {
		if edge.Source == edge.Target {
			t.Errorf("unexpected self-loop edge: %+v", edge)
		}
		key := string(edge.Source) + "->" + string(edge.Target)
		edgeMap[key] = edge.Weight
	}

	if edgeMap[string(ClusterCLI)+"->"+string(ClusterGraph)] != 2 {
		t.Errorf("expected CLI->Graph weight 2, got %d", edgeMap[string(ClusterCLI)+"->"+string(ClusterGraph)])
	}
	if edgeMap[string(ClusterCLI)+"->"+string(ClusterConfig)] != 1 {
		t.Errorf("expected CLI->Config weight 1, got %d", edgeMap[string(ClusterCLI)+"->"+string(ClusterConfig)])
	}
	if edgeMap[string(ClusterGraph)+"->"+string(ClusterGit)] != 1 {
		t.Errorf("expected Graph->Git weight 1, got %d", edgeMap[string(ClusterGraph)+"->"+string(ClusterGit)])
	}
	if edgeMap[string(ClusterGraph)+"->"+string(ClusterAI)] != 1 {
		t.Errorf("expected Graph->AI weight 1, got %d", edgeMap[string(ClusterGraph)+"->"+string(ClusterAI)])
	}
	if edgeMap[string(ClusterAI)+"->"+string(ClusterGraph)] != 1 {
		t.Errorf("expected AI->Graph weight 1, got %d", edgeMap[string(ClusterAI)+"->"+string(ClusterGraph)])
	}
	if edgeMap[string(ClusterTesting)+"->"+string(ClusterGraph)] != 1 {
		t.Errorf("expected Testing->Graph weight 1, got %d", edgeMap[string(ClusterTesting)+"->"+string(ClusterGraph)])
	}

	// Ensure no self-loop edge for CLI
	if _, ok := edgeMap[string(ClusterCLI)+"->"+string(ClusterCLI)]; ok {
		t.Errorf("found unwanted self-loop CLI->CLI")
	}
}

func TestClassifyCluster_NoGreedySubstringFalsePositives(t *testing.T) {
	tests := []struct {
		path     string
		role     string
		expected ClusterID
	}{
		{"pkg/query/builder.go", "Query Builder", ClusterCore},
		{"pkg/client/client.go", "API Client Handler", ClusterCore},
		{"internal/ui/server.go", "Web Server", ClusterUI},
		{"cmd/sage/main.go", "CLI Tool", ClusterCLI},
		{"docs/adr/001.md", "Architecture Decision", ClusterDocs},
	}

	for _, tt := range tests {
		got := ClassifyCluster(tt.path, tt.role)
		if got != tt.expected {
			t.Errorf("ClassifyCluster(%q, %q) = %q, expected %q", tt.path, tt.role, got, tt.expected)
		}
	}
}

package graph

import (
	"path/filepath"
	"testing"
	"time"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
)

func TestEngineLifecycleAndOperations(t *testing.T) {
	tmpDir := t.TempDir()
	eng, err := NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// 1. Metadata
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

	// 2. Git Ingestion
	commits := []git.Commit{
		{
			SHA:         "c1",
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

	// 3. CODEOWNERS
	rules := []git.OwnershipRule{
		{
			Pattern: "pkg/*",
			Owners:  []string{"@alice"},
		},
	}
	if err := eng.IngestCodeowners(rules); err != nil {
		t.Fatalf("ingest codeowners: %v", err)
	}

	owner, experts, err := eng.QueryOwner("pkg/util.go")
	if err != nil {
		t.Fatalf("query owner: %v", err)
	}
	if owner != "@alice" {
		t.Fatalf("expected owner @alice, got %s", owner)
	}

	owner, experts, err = eng.QueryOwner("main.go")
	if err != nil {
		t.Fatalf("query owner main.go: %v", err)
	}
	if len(experts) == 0 {
		t.Fatalf("expected at least 1 expert for main.go")
	}

	// 4. AI File Analysis
	analyses := []ai.FileAnalysis{
		{
			Path:         "main.go",
			Summary:      "Application entry point",
			Role:         "CLI Handler",
			Exports:      []string{"main"},
			Dependencies: []string{"fmt"},
			Hash:         "hash123",
			UpdatedAt:    time.Now(),
		},
	}
	if err := eng.IngestFileAnalysis(analyses); err != nil {
		t.Fatalf("ingest file analysis: %v", err)
	}

	storedHash, err := eng.GetStoredFileHash("main.go")
	if err != nil || storedHash != "hash123" {
		t.Fatalf("expected hash123, got %s (err: %v)", storedHash, err)
	}

	analysis, err := eng.GetFileAnalysis("main.go")
	if err != nil || analysis == nil || analysis.Role != "CLI Handler" {
		t.Fatalf("expected CLI Handler analysis, got %+v (err: %v)", analysis, err)
	}

	// 5. Ingest Decisions
	prs := []github.PullRequest{
		{Number: 42, Title: "Refactor storage", URL: "https://github.com/pr/42", Author: "alice"},
	}
	decisions := []ai.Decision{
		{ID: "DEC-42", Title: "Use LadybugDB", Rationale: "Faster embedded graphs", ApprovedBy: "alice", PRNumber: 42},
	}
	if err := eng.IngestDecisions(prs, decisions); err != nil {
		t.Fatalf("ingest decisions: %v", err)
	}

	// 6. Query Why
	whyLines, err := eng.QueryWhy("main.go")
	if err != nil {
		t.Fatalf("query why: %v", err)
	}
	if len(whyLines) == 0 {
		t.Fatalf("expected why results, got empty")
	}

	// 7. Query Overview
	dual, err := eng.QueryDualOverview()
	if err != nil {
		t.Fatalf("dual overview: %v", err)
	}
	if dual.GitCommits == 0 || dual.GitFiles == 0 || dual.MindFiles == 0 {
		t.Fatalf("unexpected dual stats: %+v", dual)
	}

	// 8. Execute Raw Cypher / Prefix
	rows, err := eng.ExecuteRawCypher("person:")
	if err != nil || len(rows) == 0 {
		t.Fatalf("execute raw cypher prefix failed: %v, rows: %v", err, rows)
	}

	cypherRows, err := eng.ExecuteRawCypher("MATCH (n:KV) RETURN n.key LIMIT 1")
	if err != nil || len(cypherRows) == 0 {
		t.Fatalf("execute direct cypher query failed: %v, rows: %v", err, cypherRows)
	}

	// 9. Delete Analysis
	if err := eng.DeleteFileAnalysis([]string{"main.go"}); err != nil {
		t.Fatalf("delete file analysis: %v", err)
	}
	delAnalysis, _ := eng.GetFileAnalysis("main.go")
	if delAnalysis != nil {
		t.Fatalf("expected nil analysis after delete, got %+v", delAnalysis)
	}
}

func TestLddbFilesCreated(t *testing.T) {
	tmpDir := t.TempDir()
	eng, err := NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	_ = eng.Close()

	// Verify .lddb files exist
	gitLddb := filepath.Join(tmpDir, "git.lddb")
	mindLddb := filepath.Join(tmpDir, "sage.lddb")

	gitStat, err := filepath.Glob(gitLddb + "*")
	if err != nil || len(gitStat) == 0 {
		t.Fatalf("git.lddb not created")
	}
	mindStat, err := filepath.Glob(mindLddb + "*")
	if err != nil || len(mindStat) == 0 {
		t.Fatalf("sage.lddb not created")
	}
}

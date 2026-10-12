package graph

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
)

func TestBadgerEngineLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sage-badger-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "data")
	engine, err := NewEngine(dbPath)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	defer engine.Close()

	// 1. Ingest Git History
	commits := []git.Commit{
		{
			SHA:         "abcdef1234567890",
			AuthorName:  "Jane Developer",
			AuthorEmail: "jane@company.com",
			Timestamp:   time.Now(),
			Message:     "feat: implement auth service (#10)",
			Files:       []string{"services/auth.go", "config/auth.json"},
		},
	}
	expertise := map[string][]git.ContributorExpertise{
		"services/auth.go": {
			{
				AuthorName:  "Jane Developer",
				AuthorEmail: "jane@company.com",
				Score:       42.5,
				CommitCount: 5,
			},
		},
	}

	if err := engine.IngestGitHistory(commits, expertise); err != nil {
		t.Fatalf("IngestGitHistory failed: %v", err)
	}

	// 2. Ingest Codeowners
	codeowners := []git.OwnershipRule{
		{
			Pattern: "services/*",
			Owners:  []string{"@auth-team"},
		},
	}
	if err := engine.IngestCodeowners(codeowners); err != nil {
		t.Fatalf("IngestCodeowners failed: %v", err)
	}

	// 3. Ingest PR and Decisions
	prs := []github.PullRequest{
		{
			Number: 10,
			Title:  "Implement JWT auth",
			URL:    "https://github.com/org/repo/pull/10",
			Author: "jane",
		},
	}
	decisions := []ai.Decision{
		{
			ID:        "DEC-010",
			Title:     "Use asymmetric JWT keys",
			Rationale: "Enables independent verification across microservices",
		},
	}
	if err := engine.IngestDecisions(prs, decisions); err != nil {
		t.Fatalf("IngestDecisions failed: %v", err)
	}

	// 4. Test QueryOwner
	owner, experts, err := engine.QueryOwner("services/auth.go")
	if err != nil {
		t.Fatalf("QueryOwner failed: %v", err)
	}
	if owner != "@auth-team" {
		t.Errorf("expected owner '@auth-team', got %q", owner)
	}
	if len(experts) == 0 {
		t.Errorf("expected experts for services/auth.go, got 0")
	}

	// 5. Test QueryWhy
	lineage, err := engine.QueryWhy("services/auth.go")
	if err != nil {
		t.Fatalf("QueryWhy failed: %v", err)
	}
	if len(lineage) == 0 {
		t.Fatalf("expected lineage trace results, got 0")
	}

	// 6. Test QueryOverview
	stats, err := engine.QueryOverview()
	if err != nil {
		t.Fatalf("QueryOverview failed: %v", err)
	}
	if stats["Files"] < 2 {
		t.Errorf("expected >= 2 files, got %d", stats["Files"])
	}
	if stats["Commits"] < 1 {
		t.Errorf("expected >= 1 commit, got %d", stats["Commits"])
	}
}

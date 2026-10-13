package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCalculateExpertise(t *testing.T) {
	now := time.Now()
	extractor := NewExtractor(".", 100, 60*24*time.Hour)

	commits := []Commit{
		{
			SHA:         "aaa111",
			AuthorName:  "Alice",
			AuthorEmail: "alice@example.com",
			Timestamp:   now.Add(-24 * time.Hour),
			Files:       []string{"main.go", "utils.go"},
		},
		{
			SHA:         "bbb222",
			AuthorName:  "Bob",
			AuthorEmail: "bob@example.com",
			Timestamp:   now.Add(-90 * 24 * time.Hour), // Older commit
			Files:       []string{"main.go"},
		},
	}

	expertise := extractor.CalculateExpertise(commits)
	if len(expertise["main.go"]) != 2 {
		t.Fatalf("expected 2 contributors for main.go, got %d", len(expertise["main.go"]))
	}

	// Alice's score should be higher than Bob's due to recency decay
	var aliceScore, bobScore float64
	for _, c := range expertise["main.go"] {
		if c.AuthorEmail == "alice@example.com" {
			aliceScore = c.Score
		} else if c.AuthorEmail == "bob@example.com" {
			bobScore = c.Score
		}
	}

	if aliceScore <= bobScore {
		t.Errorf("expected Alice score (%.4f) to be > Bob score (%.4f) due to recency", aliceScore, bobScore)
	}
}

func TestParseCodeownersFile(t *testing.T) {
	tmpDir := t.TempDir()
	codeownersPath := filepath.Join(tmpDir, "CODEOWNERS")

	content := `# Global ownership
* @global-owner
/internal/auth/* @security-team @alice
# Empty line

/docs/ @docs-team
`
	if err := os.WriteFile(codeownersPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test CODEOWNERS: %v", err)
	}

	rules, err := parseCodeownersFile(codeownersPath)
	if err != nil {
		t.Fatalf("parseCodeownersFile failed: %v", err)
	}

	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	if rules[0].Pattern != "*" || rules[0].Owners[0] != "@global-owner" {
		t.Errorf("unexpected first rule: %+v", rules[0])
	}

	if rules[1].Pattern != "/internal/auth/*" || len(rules[1].Owners) != 2 {
		t.Errorf("unexpected second rule: %+v", rules[1])
	}
}

func TestIncrementalGitExtraction(t *testing.T) {
	// Test on current repository
	extractor := NewExtractor(".", 10, 60*24*time.Hour)
	headSHA, err := extractor.GetHeadSHA()
	if err != nil {
		t.Fatalf("GetHeadSHA failed: %v", err)
	}
	if len(headSHA) < 40 {
		t.Errorf("expected 40-char SHA, got %s", headSHA)
	}

	// Commits since HEAD should be empty
	commits, err := extractor.ExtractCommitsSince(headSHA)
	if err != nil {
		t.Fatalf("ExtractCommitsSince(HEAD) failed: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits since HEAD, got %d", len(commits))
	}

	// Tracked files should include main files
	tracked, err := extractor.ListTrackedFiles()
	if err != nil {
		t.Fatalf("ListTrackedFiles failed: %v", err)
	}
	if len(tracked) == 0 {
		t.Errorf("expected tracked files, got 0")
	}
}


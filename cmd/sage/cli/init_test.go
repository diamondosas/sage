package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sage-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(origWd)

	// Set custom token
	initToken = "ghp_test_token_12345"
	initForce = true
	defer func() {
		initToken = ""
		initForce = false
	}()

	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// 1. Check workflow file created
	workflowPath := filepath.Join(".github", "workflows", "sage-sync.yml")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("expected workflow file at %s, got error: %v", workflowPath, err)
	}
	if !strings.Contains(string(content), "Sage Knowledge Graph Sync") {
		t.Errorf("expected workflow content to contain 'Sage Knowledge Graph Sync'")
	}
	if !strings.Contains(string(content), "permissions:\n  contents: write\n  pull-requests: read") {
		t.Errorf("expected workflow to specify contents write and pull-requests read permissions")
	}

	// 2. Check AGENTS.md created
	if _, err := os.Stat("AGENTS.md"); os.IsNotExist(err) {
		t.Errorf("expected AGENTS.md to be created")
	}

	// 3. Check .env and .gitignore created
	envContent, err := os.ReadFile(".env")
	if err != nil {
		t.Fatalf("expected .env file to be created: %v", err)
	}
	if !strings.Contains(string(envContent), "GITHUB_TOKEN=ghp_test_token_12345") {
		t.Errorf("expected .env to contain token, got %s", string(envContent))
	}

	gitIgnoreContent, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("expected .gitignore file to be created: %v", err)
	}
	if !strings.Contains(string(gitIgnoreContent), ".env") {
		t.Errorf("expected .gitignore to ignore .env")
	}
}

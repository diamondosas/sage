package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sage/cmd/sage/cli"
)

func TestInitCmd_CreatesStarterTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "SAGE.md")

	cmd := cli.NewInitCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", targetFile})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	text := string(content)
	// Verify header comment, sample group, shapes (rect, cylinder, diamond), and connections
	if !strings.Contains(text, "SAGE Architecture Diagram") && !strings.Contains(text, "// group:") {
		t.Errorf("expected starter template to contain architecture header / groups, got:\n%s", text)
	}
	if !strings.Contains(text, "rect") || !strings.Contains(text, "cylinder") || !strings.Contains(text, "diamond") {
		t.Errorf("expected starter template to contain rect, cylinder, and diamond shapes, got:\n%s", text)
	}
	if !strings.Contains(text, "->") {
		t.Errorf("expected starter template to contain connection arrows, got:\n%s", text)
	}
}

func TestInitCmd_FailsWithoutForceIfFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "SAGE.md")
	initialContent := "existing content"
	if err := os.WriteFile(targetFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	cmd := cli.NewInitCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", targetFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when file exists without --force, got nil")
	}

	// Verify original file was preserved
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != initialContent {
		t.Errorf("expected file content to remain unchanged, got: %s", string(content))
	}
}

func TestInitCmd_OverwritesWithForce(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "SAGE.md")
	if err := os.WriteFile(targetFile, []byte("old content"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	cmd := cli.NewInitCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", targetFile, "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success with --force, got: %v", err)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) == "old content" {
		t.Error("expected file content to be overwritten")
	}
}

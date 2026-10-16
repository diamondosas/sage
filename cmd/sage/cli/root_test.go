package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sage/cmd/sage/cli"
)

func TestRootCmd_SubcommandsRegistered(t *testing.T) {
	rootCmd := cli.NewRootCmd()

	var hasUI, hasInit bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "ui" {
			hasUI = true
		}
		if cmd.Name() == "init" {
			hasInit = true
		}
	}

	if !hasUI {
		t.Error("expected 'ui' subcommand to be registered on root command")
	}
	if !hasInit {
		t.Error("expected 'init' subcommand to be registered on root command")
	}
}

func TestRootCmd_CreatesStarterFileIfNotPresent(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "SAGE.md")

	cli.SetBrowserOpenerForTest(func(url string) error {
		return nil
	})
	defer cli.ResetBrowserOpenerForTest()

	var capturedFile string
	cli.SetServerRunnerForTest(func(ctx context.Context, host string, port int, filePath string) error {
		capturedFile = filePath
		return nil
	})
	defer cli.ResetServerRunnerForTest()

	rootCmd := cli.NewRootCmd()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--file", targetFile, "--no-browser"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected root command execution to succeed, got: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("expected starter file %s to be created, but got err: %v", targetFile, err)
	}
	if !strings.Contains(string(content), "SAGE Architecture Diagram") {
		t.Errorf("expected starter content in %s, got:\n%s", targetFile, string(content))
	}
	if capturedFile != targetFile {
		t.Errorf("expected server to be started with file %q, got %q", targetFile, capturedFile)
	}
}

func TestRootCmd_PreservesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "SAGE.md")
	customContent := "// Custom Existing Architecture Diagram\nrect existing \"Existing Service\"\n"
	if err := os.WriteFile(targetFile, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cli.SetBrowserOpenerForTest(func(url string) error {
		return nil
	})
	defer cli.ResetBrowserOpenerForTest()

	cli.SetServerRunnerForTest(func(ctx context.Context, host string, port int, filePath string) error {
		return nil
	})
	defer cli.ResetServerRunnerForTest()

	rootCmd := cli.NewRootCmd()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--file", targetFile, "--no-browser"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected root command execution to succeed, got: %v", err)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != customContent {
		t.Errorf("expected file to be preserved verbatim, but got:\n%s", string(content))
	}
}

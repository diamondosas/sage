package cli

import (
	"bytes"
	"testing"
)

func TestRootCommandRegistration(t *testing.T) {
	expectedCommands := []string{"init", "sync", "who", "why", "overview", "query", "mcp"}

	for _, name := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Errorf("expected subcommand %q to be registered on rootCmd", name)
		}
	}
}

func TestCommandArgsValidation(t *testing.T) {
	// 'who' requires exactly 1 argument
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"who"})

	err := rootCmd.Execute()
	if err == nil {
		t.Errorf("expected 'sage who' without args to fail with validation error")
	}

	// 'why' requires exactly 1 argument
	rootCmd.SetArgs([]string{"why"})
	err = rootCmd.Execute()
	if err == nil {
		t.Errorf("expected 'sage why' without args to fail with validation error")
	}

	// 'query' requires exactly 1 argument
	rootCmd.SetArgs([]string{"query"})
	err = rootCmd.Execute()
	if err == nil {
		t.Errorf("expected 'sage query' without args to fail with validation error")
	}
}

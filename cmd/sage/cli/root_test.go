package cli

import (
	"bytes"
	"testing"
)

func TestRootCommandRegistration(t *testing.T) {
	expectedCommands := []string{"init", "setup", "sync", "overview", "query", "ui"}

	for _, name := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Errorf("expected subcommand %q to be registered on rootCmd", name)
		}
	}

	purgedCommands := []string{"who", "why", "mcp"}
	for _, name := range purgedCommands {
		cmd, _, _ := rootCmd.Find([]string{name})
		if cmd != nil && cmd.Name() == name {
			t.Errorf("deprecated command %q should NOT be registered on rootCmd", name)
		}
	}
}

func TestCommandArgsValidation(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// 'query' requires exactly 1 argument
	rootCmd.SetArgs([]string{"query"})
	err := rootCmd.Execute()
	if err == nil {
		t.Errorf("expected 'sage query' without args to fail with validation error")
	}
}


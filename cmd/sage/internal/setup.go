package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var setupInstall bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install Sage skill into AI coding assistants (Cursor, Claude, Antigravity)",
	Long:  "Discovers installed AI coding agents and installs the Sage SKILL.md into their respective configuration folders.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If Node.js is present and installer/setup.js exists, delegate to installer/setup.js
		setupScript := filepath.Join("installer", "setup.js")
		if _, err := os.Stat(setupScript); err == nil {
			nodeCmd := exec.Command("node", setupScript)
			nodeCmd.Stdout = os.Stdout
			nodeCmd.Stderr = os.Stderr
			nodeCmd.Stdin = os.Stdin
			if err := nodeCmd.Run(); err == nil {
				return nil
			}
		}

		// Fallback native installation if node is not installed
		fmt.Println("==> Running native agent skill installation...")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		// Common agent paths
		// CHANGE HERE: Add more custom editor paths if needed
		agentPaths := []string{
			filepath.Join(homeDir, ".gemini", "config", "skills", "sage"),
			filepath.Join(homeDir, ".antigravity", "skills", "sage"),
			filepath.Join(homeDir, ".cursor", "skills", "sage"),
			filepath.Join(homeDir, ".claude", "skills", "sage"),
			filepath.Join(homeDir, ".config", "claude", "skills", "sage"),
			filepath.Join(homeDir, ".copilot", "skills", "sage"),
			filepath.Join(homeDir, ".codex", "skills", "sage"),
			filepath.Join(homeDir, ".codeium", "windsurf", "skills", "sage"),
		}

		skillSource := filepath.Join("skills", "sage", "SKILL.md")
		skillContent, err := os.ReadFile(skillSource)
		if err != nil {
			return fmt.Errorf("unable to read %s: %w", skillSource, err)
		}

		installed := 0
		for _, targetDir := range agentPaths {
			parent := filepath.Dir(targetDir)
			if _, err := os.Stat(parent); err == nil {
				_ = os.MkdirAll(targetDir, 0755)
				dest := filepath.Join(targetDir, "SKILL.md")
				if err := os.WriteFile(dest, skillContent, 0644); err == nil {
					fmt.Printf("✓ Installed skill -> %s\n", dest)
					installed++
				}
			}
		}

		if installed == 0 {
			fallback := filepath.Join(homeDir, ".sage", "skills", "sage")
			_ = os.MkdirAll(fallback, 0755)
			_ = os.WriteFile(filepath.Join(fallback, "SKILL.md"), skillContent, 0644)
			fmt.Printf("✓ Installed fallback skill -> %s\n", filepath.Join(fallback, "SKILL.md"))
		}

		fmt.Println("✓ Setup complete!")
		return nil
	},
}

func init() {
	setupCmd.Flags().BoolVarP(&setupInstall, "install", "i", false, "Install skill to all detected AI assistants")
}

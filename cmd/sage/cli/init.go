package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initForce bool
	initToken string
)

const workflowTemplate = `name: Sage Knowledge Graph Sync

# Automatically syncs on push to main or when pull requests are merged
on:
  push:
    # CHANGE HERE: Add your default branches
    branches: [ main, master ]
  pull_request:
    types: [ closed ]
  workflow_dispatch:

# Permissions required for reading PRs and persisting the BadgerDB knowledge base
permissions:
  contents: write
  pull-requests: read

jobs:
  sync:
    name: Sync Sage Knowledge Graph
    if: github.event_name != 'pull_request' || github.event.pull_request.merged == true
    runs-on: ubuntu-latest

    steps:
      - name: Check out repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      # CHANGE HERE: Install Sage binary using install.sh (no Go compiler required)
      - name: Install Sage CLI
        run: |
          if [ -f "./install/install.sh" ]; then
            bash ./install/install.sh
          else
            curl -fsSL https://raw.githubusercontent.com/diamondosas/sage/main/install/install.sh | bash
          fi

      - name: Run Sage Sync
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GROQ_API_KEY: ${{ secrets.GROQ_API_KEY }}
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
        run: |
          sage sync

      # CHANGE HERE: Persist updated BadgerDB knowledge graph into the repository
      - name: Commit & Push Updated Graph
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "Sage Bot"
          if [ -d ".github/sage" ]; then
            git add .github/sage
            git diff --staged --quiet || git commit -m "chore(sage): update knowledge graph [skip ci]"
            git push || true
          fi
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Sage repository context, workflow, and AGENTS.md instructions",
	Long:  "Initializes the local repository context directory, prompts for GitHub token, generates GitHub Actions sync workflow, and writes AGENTS.md.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Create .github/sage directory
		// CHANGE HERE: Modify Sage directory path in config if needed
		if err := os.MkdirAll(cfg.SageDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", cfg.SageDir, err)
		}

		// 2. Handle GitHub Token prompt & local config
		token := initToken
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
		if token == "" && isTerminal(os.Stdin) {
			fmt.Print("🔑 Enter GitHub Token for PR indexing (press Enter to skip / use default GITHUB_TOKEN): ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			token = strings.TrimSpace(line)
		}
		if token != "" {
			saveEnvToken(token)
			fmt.Println("✓ Saved GITHUB_TOKEN to .env (added to .gitignore)")
		} else {
			fmt.Println("ℹ  No GitHub token entered. In GitHub Actions, secrets.GITHUB_TOKEN will be used automatically.")
		}

		// 3. Generate GitHub Actions sync workflow
		workflowDir := filepath.Join(".github", "workflows")
		workflowPath := filepath.Join(workflowDir, "sage-sync.yml")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			return fmt.Errorf("failed to create workflow dir: %w", err)
		}

		if _, err := os.Stat(workflowPath); err == nil && !initForce {
			fmt.Println("ℹ  " + workflowPath + " already exists (use --force to overwrite).")
		} else {
			if err := os.WriteFile(workflowPath, []byte(workflowTemplate), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", workflowPath, err)
			}
			fmt.Println("✓ Generated GitHub Actions auto-sync workflow in " + workflowPath)
		}

		// 4. Generate AGENTS.md instructions for Cursor / Claude Code / Antigravity
		agentsContent := `# Sage Agent Instructions

This repository is indexed by Sage (BadgerDB). When working in this codebase:
- Run 'sage who <file>' to find formal owners and top recent expert contributors.
- Run 'sage why <target>' to trace why code, dependencies, or patterns exist.
- Run 'sage overview' to see high-level repository graph statistics.
- Run 'sage query "<prefix>"' to inspect raw stored graph keys.
`
		if _, err := os.Stat("AGENTS.md"); err == nil && !initForce {
			fmt.Println("ℹ  AGENTS.md already exists (use --force to overwrite).")
		} else {
			if err := os.WriteFile("AGENTS.md", []byte(agentsContent), 0644); err != nil {
				return fmt.Errorf("failed to write AGENTS.md: %w", err)
			}
			fmt.Println("✓ Generated AGENTS.md for AI coding assistants.")
		}

		fmt.Printf("✓ Initialized Sage repository context in ./%s/\n", cfg.SageDir)
		fmt.Println("\nNext step: Run 'sage sync' to index Git history into local BadgerDB.")
		return nil
	},
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func ensureGitignore(entry string) {
	content, err := os.ReadFile(".gitignore")
	if err == nil && strings.Contains(string(content), entry) {
		return
	}
	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString("\n" + entry + "\n")
	}
}

func saveEnvToken(token string) {
	ensureGitignore(".env")
	// CHANGE HERE: Change default env file path or format
	f, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("GITHUB_TOKEN=%s\n", token))
	}
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing AGENTS.md and workflow if present")
	initCmd.Flags().StringVarP(&initToken, "token", "t", "", "GitHub Personal Access Token for PR discussions & decision graph")
}

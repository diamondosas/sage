package internal

import (
	"fmt"

	"github.com/spf13/cobra"
	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
)

var (
	syncMaxCommits int
	syncSkipAI     bool
	syncSkipPRs    bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Scan Git history, PRs, and update BadgerDB knowledge graph",
	Long:  "Extracts commit history, CODEOWNERS, GitHub PR discussions, synthesizes decisions, and indexes them into BadgerDB.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("==> Sage: Starting Repository Graph Sync...")

		maxCommits := cfg.MaxCommits
		if syncMaxCommits > 0 {
			maxCommits = syncMaxCommits
		}

		// 1. Extract Git History
		gitExtractor := git.NewExtractor(".", maxCommits, cfg.DecayHalfLife)
		commits, err := gitExtractor.ExtractCommits()
		if err != nil {
			fmt.Printf("⚠️  Warning: Unable to read Git history: %v\n", err)
		} else {
			fmt.Printf("✓ Extracted %d commits from Git\n", len(commits))
		}

		expertise := gitExtractor.CalculateExpertise(commits)
		codeowners, _ := gitExtractor.ParseCodeowners()
		if len(codeowners) > 0 {
			fmt.Printf("✓ Found %d CODEOWNERS rules\n", len(codeowners))
		}

		// 2. Extract GitHub PRs (if available and not skipped)
		var prs []github.PullRequest
		var decisions []ai.Decision

		if !syncSkipPRs {
			slug, err := github.DetectRepoSlug(".")
			if err == nil {
				ghClient := github.NewClient(cfg.GitHubToken)
				fetchedPRs, err := ghClient.FetchMergedPRs(slug, 30)
				if err == nil {
					prs = fetchedPRs
					fmt.Printf("✓ Fetched %d merged PRs from GitHub (%s)\n", len(prs), slug)

					// 3. Synthesize Decisions with GoAI (or heuristic fallback)
					if !syncSkipAI {
						synthesizer := ai.NewSynthesizer(cfg.LLMProvider, cfg.LLMModel, cfg.LLMKey, cfg.OllamaHost)
						for _, pr := range prs {
							dec := synthesizer.SynthesizeDecision(pr)
							decisions = append(decisions, dec)
						}
						fmt.Printf("✓ Synthesized %d decision records\n", len(decisions))
					}
				} else {
					fmt.Printf("ℹ  Note: Unable to fetch PRs from GitHub: %v\n", err)
				}
			}
		}

		// 4. Ingest into BadgerDB
		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		_ = engine.InitSchema()

		if err := engine.IngestGitHistory(commits, expertise); err != nil {
			fmt.Printf("⚠️  Warning during Git ingestion: %v\n", err)
		}
		if err := engine.IngestCodeowners(codeowners); err != nil {
			fmt.Printf("⚠️  Warning during CODEOWNERS ingestion: %v\n", err)
		}
		if len(prs) > 0 {
			if err := engine.IngestDecisions(prs, decisions); err != nil {
				fmt.Printf("⚠️  Warning during Decisions ingestion: %v\n", err)
			}
		}

		fmt.Println("==> Sage: Sync Complete! Knowledge graph is updated in BadgerDB.")
		return nil
	},
}

func init() {
	syncCmd.Flags().IntVar(&syncMaxCommits, "max-commits", 0, "Override max commits to scan (default: from config/env)")
	syncCmd.Flags().BoolVar(&syncSkipAI, "skip-ai", false, "Skip AI decision summarization")
	syncCmd.Flags().BoolVar(&syncSkipPRs, "skip-prs", false, "Skip fetching GitHub PRs")
}

package cli

import (
	"fmt"
	"os"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"

	"github.com/spf13/cobra"
)

var (
	syncMaxCommits int
	syncSkipAI     bool
	syncSkipGit    bool
	syncSkipPRs    bool
	syncForce      bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Incrementally update Git (git.lddb) and AI Mind (sage.lddb) knowledge graphs",
	Long: `Incrementally scans new Git commits and analyzes changed codebase files.
Updates git.lddb with commits, CODEOWNERS, and recency-decay expertise.
Updates sage.lddb with AI-synthesized architectural roles, file purposes, and decisions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("==> Sage: Starting Repository Graph Sync...")

		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		_ = engine.InitSchema()

		// Detect Git HEAD and sync state
		maxCommits := cfg.MaxCommits
		if syncMaxCommits > 0 {
			maxCommits = syncMaxCommits
		}
		gitExtractor := git.NewExtractor(".", maxCommits, cfg.DecayHalfLife)
		headSHA, _ := gitExtractor.GetHeadSHA()

		gitLastSHA, _ := engine.GetGitLastSyncedCommit()
		mindLastSHA, _ := engine.GetMindLastSyncedCommit()

		// Fast-path: check if repository is already fully up to date
		if !syncForce && headSHA != "" && gitLastSHA == headSHA && mindLastSHA == headSHA {
			shortSHA := headSHA
			if len(shortSHA) > 7 {
				shortSHA = shortSHA[:7]
			}
			fmt.Printf("✓ Git graph (git.lddb) is already up to date (HEAD: %s)\n", shortSHA)
			fmt.Printf("✓ AI Mind graph (sage.lddb) is already up to date (HEAD: %s)\n", shortSHA)
			fmt.Println("==> Sage: Repository is already up to date.")
			return nil
		}

		// --- Track 1: Git Graph Sync (git.lddb) ---
		if !syncSkipGit {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("⚠️  Recovered from Git sync warning: %v (continuing)\n", r)
					}
				}()

				var commits []git.Commit
				var extractErr error
				if gitLastSHA != "" && !syncForce {
					commits, extractErr = gitExtractor.ExtractCommitsSince(gitLastSHA)
				} else {
					commits, extractErr = gitExtractor.ExtractCommits()
				}

				if extractErr != nil {
					fmt.Printf("⚠️  Warning: Unable to extract Git history: %v (continuing...)\n", extractErr)
					return
				}

				if len(commits) == 0 && gitLastSHA == headSHA {
					fmt.Println("✓ Git graph (git.lddb) already up to date")
				} else if len(commits) > 0 {
					expertise := gitExtractor.CalculateExpertise(commits)
					if err := engine.IngestGitHistory(commits, expertise); err != nil {
						fmt.Printf("⚠️  Warning: Git history ingestion error: %v\n", err)
					} else {
						fmt.Printf("✓ Ingested %d new commits into git.lddb\n", len(commits))
					}
				}

				codeowners, _ := gitExtractor.ParseCodeowners()
				if len(codeowners) > 0 {
					_ = engine.IngestCodeowners(codeowners)
				}

				if headSHA != "" {
					_ = engine.SetGitLastSyncedCommit(headSHA)
				}
			}()
		}

		// --- Track 2: AI Mind Graph Sync (sage.lddb) ---
		if !syncSkipAI {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("⚠️  Recovered from AI Mind sync warning: %v (continuing)\n", r)
					}
				}()

				// 1. Detect candidate changed or added files
				var candidateFiles []string
				var deletedFiles []string
				var diffErr error

				if mindLastSHA != "" && !syncForce {
					candidateFiles, deletedFiles, diffErr = gitExtractor.GetChangedFilesSince(mindLastSHA)
				} else {
					candidateFiles, diffErr = gitExtractor.ListTrackedFiles()
				}

				if diffErr != nil {
					fmt.Printf("⚠️  Warning: Unable to detect changed files: %v (continuing...)\n", diffErr)
					return
				}

				// Clean up deleted files from AI Mind graph
				if len(deletedFiles) > 0 {
					_ = engine.DeleteFileAnalysis(deletedFiles)
				}

				// 2. Filter indexable source files and check SHA-256 hashes
				synthesizer := ai.NewSynthesizer(cfg.LLMProvider, cfg.LLMModel, cfg.LLMKey, cfg.OllamaHost)
				var filesToAnalyze []string

				for _, f := range candidateFiles {
					if !ai.IsIndexableSourceFile(f) {
						continue
					}
					content, readErr := os.ReadFile(f)
					if readErr != nil {
						continue
					}
					currentHash := ai.ComputeFileHash(content)
					storedHash, _ := engine.GetStoredFileHash(f)
					if currentHash != storedHash || syncForce {
						filesToAnalyze = append(filesToAnalyze, f)
					}
				}

				if len(filesToAnalyze) == 0 && mindLastSHA == headSHA {
					fmt.Println("✓ AI Mind graph (sage.lddb) already up to date")
				} else if len(filesToAnalyze) > 0 {
					fmt.Printf("🧠 AI Mind: Reading and analyzing %d new/modified files...\n", len(filesToAnalyze))
					var analyses []ai.FileAnalysis

					// CHANGE HERE: Maximum files to analyze in a single sync batch (default 50)
					maxBatch := 50
					for i, f := range filesToAnalyze {
						if i >= maxBatch {
							fmt.Printf("ℹ  Reached batch limit of %d files. Remaining will sync on next run.\n", maxBatch)
							break
						}
						content, readErr := os.ReadFile(f)
						if readErr != nil {
							continue
						}
						analysis := synthesizer.AnalyzeFile(f, content)
						analyses = append(analyses, analysis)
					}

					if err := engine.IngestFileAnalysis(analyses); err != nil {
						fmt.Printf("⚠️  Warning: File analysis ingestion error: %v\n", err)
					} else {
						fmt.Printf("✓ Ingested %d AI file analyses into sage.lddb\n", len(analyses))
					}
				}

				// 3. PR Discussions & Decisions (if available)
				if !syncSkipPRs && cfg.GitHubToken != "" {
					slug, err := github.DetectRepoSlug(".")
					if err == nil {
						ghClient := github.NewClient(cfg.GitHubToken)
						prs, err := ghClient.FetchMergedPRs(slug, 20)
						if err == nil && len(prs) > 0 {
							var decisions []ai.Decision
							for _, pr := range prs {
								dec := synthesizer.SynthesizeDecision(pr)
								decisions = append(decisions, dec)
							}
							_ = engine.IngestDecisions(prs, decisions)
							fmt.Printf("✓ Synthesized %d PR decisions into sage.lddb\n", len(decisions))
						}
					}
				}

				if headSHA != "" {
					_ = engine.SetMindLastSyncedCommit(headSHA)
				}
			}()
		}

		fmt.Println("==> Sage: Sync Complete! Knowledge graphs updated in .github/sage/ (git.lddb & sage.lddb).")
		return nil
	},
}

func init() {
	// CHANGE HERE: Flags to control sync behavior and batch sizes
	syncCmd.Flags().IntVar(&syncMaxCommits, "max-commits", 0, "Override max commits to scan (default: from config/env)")
	syncCmd.Flags().BoolVar(&syncSkipAI, "skip-ai", false, "Skip AI Mind file indexing and PR summarization")
	syncCmd.Flags().BoolVar(&syncSkipGit, "skip-git", false, "Skip Git history and CODEOWNERS ingestion")
	syncCmd.Flags().BoolVar(&syncSkipPRs, "skip-prs", false, "Skip fetching GitHub PRs")
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Force full re-sync even if already up to date")
}

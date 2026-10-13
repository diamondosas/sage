package cli

import (
	"fmt"
	"os"

	"sage/internal/ai"
	"sage/internal/git"
	"sage/internal/github"
	"sage/internal/graph"

	"github.com/spf13/cobra"
)

var (
	syncMaxCommits int
	syncSkipAI     bool
	syncSkipGit    bool
	syncSkipPRs    bool
	syncForce      bool
)

const maxFileAnalysisBatch = 50

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Incrementally update Git history and AI Mind knowledge in SQLite (sage.db)",
	Long: `Incrementally scans new Git commits and analyzes changed codebase files.
Updates the unified SQLite knowledge database (.github/sage/sage.db) with commits,
CODEOWNERS, recency-decay expertise, architectural roles, file purposes, and decisions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("==> Sage: Starting Repository Knowledge Sync...")

		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

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
			fmt.Printf("✓ Repository knowledge is already up to date (HEAD: %s)\n", shortSHA)
			fmt.Println("==> Sage: Repository is already up to date.")
			return nil
		}

		// 1. Git History Track
		if !syncSkipGit {
			if err := syncGitTrack(engine, gitExtractor, headSHA, gitLastSHA, syncForce); err != nil {
				fmt.Printf("⚠️  Warning: Git sync error: %v (continuing...)\n", err)
			}
		}

		// 2. AI Mind Knowledge Track
		if !syncSkipAI {
			if err := syncAIMindTrack(engine, gitExtractor, headSHA, mindLastSHA, syncForce); err != nil {
				fmt.Printf("⚠️  Warning: AI Mind sync error: %v (continuing...)\n", err)
			}
		}

		fmt.Println("==> Sage: Sync Complete! Knowledge stored in .github/sage/sage.db.")
		return nil
	},
}

func syncGitTrack(engine *graph.Engine, extractor *git.Extractor, headSHA, lastSHA string, force bool) error {
	var commits []git.Commit
	var err error

	if lastSHA != "" && !force {
		commits, err = extractor.ExtractCommitsSince(lastSHA)
	} else {
		commits, err = extractor.ExtractCommits()
	}
	if err != nil {
		return fmt.Errorf("unable to extract git commits: %w", err)
	}

	if len(commits) == 0 && lastSHA == headSHA {
		fmt.Println("✓ Git history already up to date")
	} else if len(commits) > 0 {
		expertise := extractor.CalculateExpertise(commits)
		if err := engine.IngestGitHistory(commits, expertise); err != nil {
			return fmt.Errorf("git history ingestion failed: %w", err)
		}
		fmt.Printf("✓ Ingested %d new commits into sage.db\n", len(commits))
	}

	codeowners, _ := extractor.ParseCodeowners()
	if len(codeowners) > 0 {
		if err := engine.IngestCodeowners(codeowners); err != nil {
			fmt.Printf("⚠️  Warning: CODEOWNERS ingestion error: %v\n", err)
		}
	}

	if headSHA != "" {
		_ = engine.SetGitLastSyncedCommit(headSHA)
	}
	return nil
}

func syncAIMindTrack(engine *graph.Engine, extractor *git.Extractor, headSHA, lastSHA string, force bool) error {
	var candidateFiles []string
	var deletedFiles []string
	var err error

	if lastSHA != "" && !force {
		candidateFiles, deletedFiles, err = extractor.GetChangedFilesSince(lastSHA)
	} else {
		candidateFiles, err = extractor.ListTrackedFiles()
	}
	if err != nil {
		return fmt.Errorf("unable to detect changed files: %w", err)
	}

	// Remove deleted files from knowledge store
	if len(deletedFiles) > 0 {
		_ = engine.DeleteFileAnalysis(deletedFiles)
	}

	// Filter changed and indexable source files by content hash
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
		if currentHash != storedHash || force {
			filesToAnalyze = append(filesToAnalyze, f)
		}
	}

	if len(filesToAnalyze) == 0 && lastSHA == headSHA {
		fmt.Println("✓ AI Mind knowledge already up to date")
	} else if len(filesToAnalyze) > 0 {
		totalFiles := len(filesToAnalyze)
		batchLimit := maxFileAnalysisBatch
		if totalFiles < batchLimit {
			batchLimit = totalFiles
		}

		fmt.Printf("🧠 AI Mind: Reading and analyzing %d new/modified files (batch limit: %d)...\n", totalFiles, maxFileAnalysisBatch)
		var analyses []ai.FileAnalysis

		for i := 0; i < batchLimit; i++ {
			f := filesToAnalyze[i]
			content, readErr := os.ReadFile(f)
			if readErr != nil {
				continue
			}
			analyses = append(analyses, synthesizer.AnalyzeFile(f, content))
		}

		if err := engine.IngestFileAnalysis(analyses); err != nil {
			return fmt.Errorf("file analysis ingestion failed: %w", err)
		}
		fmt.Printf("✓ Ingested %d AI file analyses into sage.db\n", len(analyses))

		if totalFiles > batchLimit {
			fmt.Printf("ℹ  %d files remaining; will analyze on next sync run.\n", totalFiles-batchLimit)
		}
	}

	// Sync PR decisions if enabled and GitHub token is present
	if !syncSkipPRs && cfg.GitHubToken != "" {
		if err := syncPRTrack(engine, synthesizer); err != nil {
			fmt.Printf("⚠️  Warning: PR decision sync warning: %v\n", err)
		}
	}

	if headSHA != "" {
		_ = engine.SetMindLastSyncedCommit(headSHA)
	}
	return nil
}

func syncPRTrack(engine *graph.Engine, synthesizer *ai.Synthesizer) error {
	slug, err := github.DetectRepoSlug(".")
	if err != nil {
		return err
	}

	ghClient := github.NewClient(cfg.GitHubToken)
	prs, err := ghClient.FetchMergedPRs(slug, 20)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		return nil
	}

	var decisions []ai.Decision
	for _, pr := range prs {
		dec := synthesizer.SynthesizeDecision(pr)
		decisions = append(decisions, dec)
	}

	if err := engine.IngestDecisions(prs, decisions); err != nil {
		return err
	}
	fmt.Printf("✓ Synthesized %d PR decisions into sage.db\n", len(decisions))
	return nil
}

func init() {
	syncCmd.Flags().IntVar(&syncMaxCommits, "max-commits", 0, "Override max commits to scan (default: from config/env)")
	syncCmd.Flags().BoolVar(&syncSkipAI, "skip-ai", false, "Skip AI Mind file indexing and PR summarization")
	syncCmd.Flags().BoolVar(&syncSkipGit, "skip-git", false, "Skip Git history and CODEOWNERS ingestion")
	syncCmd.Flags().BoolVar(&syncSkipPRs, "skip-prs", false, "Skip fetching GitHub PRs")
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Force full re-sync even if already up to date")
}

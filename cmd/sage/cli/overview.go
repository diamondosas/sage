package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Display compact repository knowledge statistics",
	Long:  "Queries high-level counts from the SQLite knowledge store (.github/sage/sage.db) without bloating AI context windows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		dual, err := engine.QueryDualOverview()
		if err != nil {
			return fmt.Errorf("failed to retrieve graph overview: %w", err)
		}

		gitSHA := dual.GitLastSynced
		if len(gitSHA) > 7 {
			gitSHA = gitSHA[:7]
		}
		if gitSHA == "" {
			gitSHA = "Not synced"
		}

		mindSHA := dual.MindLastSynced
		if len(mindSHA) > 7 {
			mindSHA = mindSHA[:7]
		}
		if mindSHA == "" {
			mindSHA = "Not synced"
		}

		fmt.Println("\n📊 Sage Knowledge Overview (.github/sage/sage.db):")
		fmt.Println(strings.Repeat("═", 50))
		fmt.Println("📦 Git History Knowledge:")
		fmt.Printf("  • Commits Tracked      : %d\n", dual.GitCommits)
		fmt.Printf("  • Active Contributors  : %d\n", dual.GitPeople)
		fmt.Printf("  • Touched Files        : %d\n", dual.GitFiles)
		fmt.Printf("  • Last Synced Commit   : %s\n", gitSHA)

		fmt.Println("\n🧠 AI Mind Knowledge:")
		fmt.Printf("  • Files Analyzed by AI : %d\n", dual.MindFiles)
		fmt.Printf("  • Architectural Roles  : %d\n", dual.MindRoles)
		fmt.Printf("  • Synthesized Decisions: %d\n", dual.MindDecisions)
		fmt.Printf("  • Last Synced Commit   : %s\n", mindSHA)
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println()
		return nil
	},
}

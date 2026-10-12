package internal

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Display compact repository graph statistics",
	Long:  "Queries high-level node counts (Files, Commits, People, Decisions) from BadgerDB without bloating AI context windows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		stats, err := engine.QueryOverview()
		if err != nil {
			return fmt.Errorf("failed to retrieve graph overview: %w", err)
		}

		fmt.Println("\n📊 Sage Knowledge Graph Overview:")
		fmt.Println(strings.Repeat("─", 36))
		for _, key := range []string{"Files", "Commits", "People", "Decisions"} {
			val := stats[key]
			fmt.Printf("  • %-12s : %d\n", key, val)
		}
		fmt.Println()
		return nil
	},
}

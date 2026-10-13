package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var whyCmd = &cobra.Command{
	Use:   "why <target>",
	Short: "Trace why code was written back to PRs and decisions",
	Long:  "Traces the multi-hop lineage of a file or commit SHA through commits, pull requests, and architectural decisions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		lineage, err := engine.QueryWhy(target)
		if err != nil {
			return fmt.Errorf("failed to trace lineage: %w", err)
		}

		fmt.Printf("\n🔍 Lineage Trace for: %s\n", target)
		fmt.Println(strings.Repeat("═", 55))
		if len(lineage) == 0 {
			fmt.Println("ℹ  No recorded commit or decision history found for this target.")
		} else {
			for _, entry := range lineage {
				fmt.Println(entry)
				fmt.Println(strings.Repeat("─", 40))
			}
		}
		fmt.Println()
		return nil
	},
}

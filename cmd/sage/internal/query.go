package internal

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <key_prefix>",
	Short: "Inspect stored entries in BadgerDB",
	Long:  "Inspects keys and stored records directly in the local BadgerDB embedded store matching the given key prefix.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyPrefix := args[0]

		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		rows, err := engine.ExecuteRawCypher(keyPrefix)
		if err != nil {
			return fmt.Errorf("query execution failed: %w", err)
		}

		fmt.Println("\n📋 BadgerDB Stored Results:")
		fmt.Println(strings.Repeat("─", 45))
		if len(rows) == 0 {
			fmt.Println("  (No results returned)")
		} else {
			for _, row := range rows {
				fmt.Println("  | " + strings.Join(row, " | ") + " |")
			}
		}
		fmt.Printf("\nTotal: %d row(s)\n\n", len(rows))
		return nil
	},
}

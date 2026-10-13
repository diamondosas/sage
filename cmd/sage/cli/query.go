package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <sql_query|search_term>",
	Short: "Inspect stored entries or execute SQL queries in SQLite",
	Long:  "Executes raw SQL queries (e.g. 'SELECT * FROM commits') or searches matching tables in the local SQLite database (.github/sage/sage.db).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		queryArg := args[0]

		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		rows, err := engine.ExecuteRawSQL(queryArg)
		if err != nil {
			return fmt.Errorf("query execution failed: %w", err)
		}

		fmt.Println("\n📋 SQLite Query Results:")
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

package internal

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var whoCmd = &cobra.Command{
	Use:   "who <path>",
	Short: "Find formal owner and top recent expert contributors",
	Long:  "Queries CODEOWNERS for the formal owner and BadgerDB for top recent active contributors based on recency-decay scoring.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		engine, err := getEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		owner, experts, err := engine.QueryOwner(path)
		if err != nil {
			return fmt.Errorf("failed to query ownership: %w", err)
		}

		fmt.Printf("\n📂 Path: %s\n", path)
		fmt.Println(strings.Repeat("─", 45))
		fmt.Printf("👑 Formal Owner:  %s\n", owner)
		fmt.Println("🧠 Recent Experts:")
		if len(experts) == 0 {
			fmt.Println("   (No recorded commits or activity for this path)")
		} else {
			for _, exp := range experts {
				fmt.Printf("   • %s\n", exp)
			}
		}
		fmt.Println()
		return nil
	},
}

package cli

import (
	"fmt"
	"os"

	"sage/internal/mcp"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run Model Context Protocol (MCP) stdio server for coding agents",
	Long:  "Starts a Model Context Protocol server over standard I/O for Cursor, Claude Code, Antigravity, and other AI pair programmers.",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "MCP initialization error: %v\n", err)
			return err
		}
		defer engine.Close()

		srv := mcp.NewServer(engine)
		if err := srv.StartStdio(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server runtime error: %v\n", err)
			return err
		}
		return nil
	},
}

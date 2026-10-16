package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// NewRootCmd constructs the primary 'sage' CLI command.
// When invoked without subcommands, it automatically ensures SAGE.md exists
// and launches the interactive web visualizer.
func NewRootCmd() *cobra.Command {
	var opts UIOptions

	rootCmd := &cobra.Command{
		Use:   "sage",
		Short: "Sage architecture mapping and visualizer tool",
		Long: `Sage maps codebase architecture into SAGE.md and launches a local
sketchy Excalidraw-style web visualizer for human inspection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := opts.FilePath
			if filePath == "" {
				filePath = DefaultDiagramFile
			}

			// Automatically initialize starter SAGE.md if file does not exist
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				cmd.Printf("Diagram %q not found; initializing starter template...\n", filePath)
				if err := CreateStarterFile(filePath, false); err != nil {
					return fmt.Errorf("failed to create starter diagram: %w", err)
				}
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return RunUIWorkflow(ctx, opts, cmd.OutOrStdout())
		},
	}

	// Persistent and local flags for the root command.
	// ACTIONABLE: Change default flag assignments (DefaultDiagramFile, DefaultPort, DefaultHost)
	// or add new persistent CLI flags below.
	rootCmd.PersistentFlags().StringVarP(&opts.FilePath, "file", "f", DefaultDiagramFile, "Path to diagram markdown file")
	rootCmd.Flags().IntVarP(&opts.Port, "port", "p", DefaultPort, "Port to run the UI server on")
	rootCmd.Flags().StringVarP(&opts.Host, "host", "H", DefaultHost, "Bind host address")
	rootCmd.Flags().BoolVar(&opts.NoBrowser, "no-browser", false, "Do not automatically launch the web browser")

	// Register subcommands
	rootCmd.AddCommand(NewUICmd())
	rootCmd.AddCommand(NewInitCmd())

	return rootCmd
}

// Execute is the main entry point called from cmd/sage/main.go.
func Execute() error {
	return NewRootCmd().Execute()
}

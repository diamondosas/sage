package cli

import (
	"fmt"

	"sage/internal/config"
	"sage/internal/graph"

	"github.com/spf13/cobra"
)

var (
	cfg    *config.Config
	dbPath string
)

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "Sage — Git-Native Local Company Brain powered by SQLite",
	Long: `Sage is a Git-native company knowledge brain.
It connects codebase structure with Git history, human decisions, and CODEOWNERS
so developers and AI agents can query tribal context and ownership on demand.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cfg = config.LoadFromEnv()

	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", cfg.DBPath, "SQLite embedded database path (default: .github/sage/sage.db)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(overviewCmd)
	rootCmd.AddCommand(queryCmd)
}

// Execute runs the Sage root CLI command.
func Execute() error {
	return rootCmd.Execute()
}

// getEngine creates an initialized SQLite knowledge engine using the local database path.
func getEngine() (*graph.Engine, error) {
	path := dbPath
	if path == "" {
		path = cfg.DBPath
	}
	if path == "" {
		path = ".github/sage"
	}

	engine, err := graph.NewEngine(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open embedded SQLite database at %s: %w", path, err)
	}
	return engine, nil
}

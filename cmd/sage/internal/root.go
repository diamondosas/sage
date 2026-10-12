package internal

import (
	"fmt"

	"github.com/spf13/cobra"
	"sage/internal/config"
	"sage/internal/graph"
)

var (
	cfg    *config.Config
	dbPath string
)

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "Sage — Git-Native Local Company Brain powered by BadgerDB",
	Long: `Sage is a Git-native company knowledge brain.
It connects codebase structure with Git history, human decisions, and CODEOWNERS
so developers and AI agents can query tribal context and ownership on demand.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cfg = config.LoadFromEnv()

	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", cfg.BadgerPath, "BadgerDB embedded storage directory path")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(whoCmd)
	rootCmd.AddCommand(whyCmd)
	rootCmd.AddCommand(overviewCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(mcpCmd)
}

// Execute runs the Sage root CLI command.
func Execute() error {
	return rootCmd.Execute()
}

// getEngine creates an initialized BadgerDB graph engine using the local database path.
func getEngine() (*graph.Engine, error) {
	path := dbPath
	if path == "" {
		path = cfg.BadgerPath
	}
	if path == "" {
		path = ".github/sage"
	}

	engine, err := graph.NewEngine(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open embedded Badger database at %s: %w", path, err)
	}
	return engine, nil
}

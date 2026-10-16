package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// StarterTemplate defines the initial SAGE.md architecture diagram skeleton.
// ACTIONABLE: Customize this template string to change the default starter diagram,
// default shapes, or domain clusters created for new repositories.
const StarterTemplate = `// SAGE Architecture Diagram
// AI Agent Mission: Map codebase architecture here so humans can inspect it in the web visualizer.
// The AI agent must never assume anything: verify facts via git diff and file inspections.

// group: Web Interface
// path: ui/
circle user "Browser Client" [green]

// group: CLI & Service
// path: cmd/sage
rect cli "Cobra CLI" [blue]
diamond router "Command Router" [yellow]

// group: Storage
// path: internal/graph
cylinder db "Database Store" [purple]

// Connections
user -> cli "Runs 'sage' in terminal"
cli -> router "Dispatches action"
router -> db "Reads/writes state"
`

// DefaultDiagramFile is the default filename used for architecture diagrams.
// ACTIONABLE: Change this constant if a different default diagram file name is desired.
const DefaultDiagramFile = "SAGE.md"

// CreateStarterFile writes the starter architecture template to the given path.
// If force is false and the file already exists, it returns an error.
func CreateStarterFile(path string, force bool) error {
	if path == "" {
		path = DefaultDiagramFile
	}

	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file %q already exists (use --force to overwrite)", path)
		}
	}

	if err := os.WriteFile(path, []byte(StarterTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write %q: %w", path, err)
	}
	return nil
}

// NewInitCmd returns a new Cobra command for "sage init".
func NewInitCmd() *cobra.Command {
	var filePath string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new starter SAGE.md diagram",
		Long:  "Create a starter SAGE.md architecture diagram template with sample groups, shapes, and connections.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := CreateStarterFile(filePath, force); err != nil {
				return err
			}
			cmd.Printf("Initialized starter architecture diagram in %s\n", filePath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", DefaultDiagramFile, "Path to diagram file")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing diagram file")

	return cmd
}

package commands

import (
	"github.com/harness/helm-optimize/pkg/cleanup"
	"github.com/spf13/cobra"
)

var (
	// Cleanup command flags
	cleanupDryRun      bool
	cleanupShowDeleted bool
)

// NewCleanupCmd creates the cleanup subcommand
func NewCleanupCmd() *cobra.Command {
	var cleanupCmd = &cobra.Command{
		Use:   "cleanup [CHART_PATH]",
		Short: "Remove unnecessary chart directories",
		Long: `Remove unnecessary directories created during 'helm dep up'.
		
This command performs a depth-first search on Helm charts, runs 'helm dep up'
at the bottom-most level, and removes original directories for dependencies 
with 'repository: file:' format after the dependency charts are created.`,
		Args: cobra.ExactArgs(1),
		RunE: runCleanup,
	}

	// Add flags specific to cleanup command
	f := cleanupCmd.Flags()
	f.BoolVar(&cleanupDryRun, "dry-run", false, "Simulate cleanup without making changes")
	f.BoolVar(&cleanupShowDeleted, "show-deleted", false, "Show paths that would be deleted")

	return cleanupCmd
}

// runCleanup implements the cleanup command logic
func runCleanup(cmd *cobra.Command, args []string) error {
	chartPath := args[0]

	// Create cleanup options
	opts := cleanup.Options{
		ChartPath:   chartPath,
		DryRun:      cleanupDryRun,
		ShowDeleted: cleanupShowDeleted,
		Verbose:     IsVerbose(),
	}

	// Run the cleanup
	return cleanup.Run(opts)
}

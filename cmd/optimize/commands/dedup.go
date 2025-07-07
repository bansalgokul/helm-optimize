package commands

import (
	"github.com/spf13/cobra"
	"github.com/harness/helm-optimize/pkg/dedup"
)

var (
	// Dedup command flags
	outputDir   string
	package_    bool
	dryRun      bool
	showDeleted bool
)

// NewDedupCmd creates the dedup subcommand
func NewDedupCmd() *cobra.Command {
	var dedupCmd = &cobra.Command{
		Use:   "dedup [CHART_PATH]",
		Short: "Deduplicate dependency charts",
		Long: `Deduplicate dependency charts to reduce the size of Helm packages.

This command analyzes a Helm chart and its dependencies, identifying and removing
duplicate subchart references while maintaining all required functionality.`,
		Args: cobra.ExactArgs(1),
		RunE: runDedup,
	}

	// Add flags specific to dedup command
	f := dedupCmd.Flags()
	f.StringVarP(&outputDir, "output", "o", "", "Output directory for deduplicated chart (default: input directory)")
	f.BoolVarP(&package_, "package", "p", false, "Package chart after deduplication")
	f.BoolVar(&dryRun, "dry-run", false, "Simulate deduplication without making changes")
	f.BoolVar(&showDeleted, "show-deleted", false, "Show paths that would be deleted")

	return dedupCmd
}

// runDedup implements the dedup command logic
func runDedup(cmd *cobra.Command, args []string) error {
	chartPath := args[0]
	
	// Create deduplicator options
	opts := dedup.Options{
		ChartPath:   chartPath,
		OutputDir:   outputDir,
		Package:     package_,
		DryRun:      dryRun,
		ShowDeleted: showDeleted,
		Verbose:     IsVerbose(),
	}
	
	// Run the deduplication
	return dedup.Run(opts)
}

package commands

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "optimize",
		Short: "Optimize Helm charts",
		Long: `A Helm plugin that optimizes Helm charts in various ways.

This plugin provides multiple optimization features for Helm charts,
helping to improve performance, reduce size, and enhance usability.`,
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(NewDedupCmd())

	return rootCmd
}

// IsVerbose returns the global verbose flag value
func IsVerbose() bool {
	return verbose
}

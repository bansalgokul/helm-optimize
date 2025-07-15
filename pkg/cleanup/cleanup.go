package cleanup

import (
	"fmt"
)

// Options represents the configuration options for the cleanup operation
type Options struct {
	ChartPath   string
	DryRun      bool
	ShowDeleted bool
	Verbose     bool
}

// Run executes the cleanup operation with the given options
func Run(opts Options) error {
	if opts.Verbose {
		fmt.Printf("Starting cleanup of chart at %s\n", opts.ChartPath)
	}

	cleaner := NewCleaner(opts)
	return cleaner.Cleanup()
}

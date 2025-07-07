package main

import (
	"fmt"
	"os"

	"github.com/harness/helm-optimize/cmd/optimize/commands"
)

func main() {
	// Create the root command
	rootCmd := commands.NewRootCmd()

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

# helm-optimize

A Helm plugin to optimize Helm charts, including deduplication of dependencies to reduce package size.

## Overview

When packaging Helm charts with many modules and subcharts, duplicate dependency charts can increase the size of the chart zip significantly. This plugin addresses that issue by implementing an algorithm to detect and eliminate duplicated dependencies.

## Installation

```bash
helm plugin install https://github.com/bansalgokul/helm-optimize
```

## Usage

The plugin provides multiple optimization commands:

```bash
# Show available commands
helm optimize --help

# Use deduplication feature
helm optimize dedup CHART_PATH

# Package after deduplication
helm optimize dedup CHART_PATH --package

# Specify output directory
helm optimize dedup CHART_PATH --output OUTPUT_DIR

# View detailed information (global flag available to all commands)
helm optimize --verbose dedup CHART_PATH

# Dry run mode (show what would be done without making changes)
helm optimize dedup CHART_PATH --dry-run --show-deleted
```

## Features

### Deduplication

The `dedup` command analyzes the chart and its dependencies, identifying duplicate subchart references. It then restructures the chart to eliminate these duplications while maintaining all required functionality.

## Extending the Plugin

This plugin uses a subcommand architecture for extensibility. To add a new optimization feature:

1. Create a new subcommand function in `main.go`
2. Add the command to the root command
3. Implement the optimization logic

Example structure for adding a new feature:

```go
// In main.go
func main() {
    // ... existing code ...
    
    // Add subcommands
    rootCmd.AddCommand(newDedupCmd())
    rootCmd.AddCommand(newYourFeatureCmd())  // Add your new feature command
}

// Create a new subcommand
func newYourFeatureCmd() *cobra.Command {
    // Define the command and flags
    // Implement the command logic
}
```

## License

MIT

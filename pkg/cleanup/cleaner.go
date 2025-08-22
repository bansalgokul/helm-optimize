package cleanup

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Cleaner is responsible for performing the cleanup operation
type Cleaner struct {
	opts         Options
	deletedPaths []string
}

// NewCleaner creates a new Cleaner instance
func NewCleaner(opts Options) *Cleaner {
	return &Cleaner{
		opts:         opts,
		deletedPaths: []string{},
	}
}

// Cleanup performs the cleanup operation
func (c *Cleaner) Cleanup() error {
	chartPath, err := filepath.Abs(c.opts.ChartPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Start DFS traversal from the root chart
	if err := c.processChart(chartPath); err != nil {
		return err
	}

	// Show deleted paths if requested
	if c.opts.ShowDeleted && len(c.deletedPaths) > 0 {
		fmt.Println("Deleted directories:")
		for _, path := range c.deletedPaths {
			fmt.Printf("  %s\n", path)
		}
	}

	if c.opts.DryRun {
		fmt.Println("Dry run completed. No changes were made.")
	} else if len(c.deletedPaths) > 0 {
		fmt.Printf("Cleanup completed. Removed %d unnecessary directories.\n", len(c.deletedPaths))
	} else {
		fmt.Println("Cleanup completed. No unnecessary directories were found.")
	}

	return nil
}

// processChart recursively processes a chart and its dependencies
func (c *Cleaner) processChart(chartPath string) error {
	// Check if this is a valid chart directory
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	if _, err := os.Stat(chartFile); os.IsNotExist(err) {
		return nil // Not a chart, skip
	}

	// First, process this chart
	if c.opts.Verbose {
		fmt.Printf("Processing chart: %s\n", chartPath)
	}

	// Cleanup any file dependencies in the current Chart.yaml
	if err := c.cleanupFileDependencies(chartPath); err != nil {
		return err
	}

	// Then process subdirectories (DFS)
	chartsDir := filepath.Join(chartPath, "charts")
	if _, err := os.Stat(chartsDir); err == nil {
		// List all subdirectories in charts/
		files, err := ioutil.ReadDir(chartsDir)
		if err != nil {
			return fmt.Errorf("failed to read charts directory: %w", err)
		}

		for _, file := range files {
			if file.IsDir() {
				subchartPath := filepath.Join(chartsDir, file.Name())
				if err := c.processChart(subchartPath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// cleanupFileDependencies identifies and removes original directories for file: dependencies
func (c *Cleaner) cleanupFileDependencies(chartPath string) error {
	// Read and parse Chart.yaml to find dependencies
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	data, err := ioutil.ReadFile(chartFile)
	if err != nil {
		return fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	var chart struct {
		Dependencies []struct {
			Name       string `yaml:"name"`
			Version    string `yaml:"version"`
			Repository string `yaml:"repository"`
		} `yaml:"dependencies"`
	}

	if err := yaml.Unmarshal(data, &chart); err != nil {
		return fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}

	// Process each dependency
	for _, dep := range chart.Dependencies {
		// Check if this is a file: repository
		if !strings.HasPrefix(dep.Repository, "file:") {
			continue
		}

		// Extract the path from the repository field
		relPath := strings.TrimPrefix(dep.Repository, "file://")
		relPath = strings.TrimPrefix(relPath, "file:")

		// Normalize the path (remove ./ if present)
		if strings.HasPrefix(relPath, "./") {
			relPath = relPath[2:]
		}

		// Construct the full path to the original directory
		// For paths like "./database", we want chartPath/database
		originalDirPath := filepath.Join(chartPath, relPath)

		// Make sure the path is absolute
		originalDirPath, err := filepath.Abs(originalDirPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		// Check if the directory exists
		if _, err := os.Stat(originalDirPath); os.IsNotExist(err) {
			if c.opts.Verbose {
				fmt.Printf("Directory %s does not exist, skipping\n", originalDirPath)
			}
			continue
		}

		// Check if the immediate parent directory is named "charts"
		// Skip deletion of directories that are direct children of a "charts" directory
		// as they are part of the chart structure and should be preserved
		parentDir := filepath.Base(filepath.Dir(originalDirPath))
		if parentDir == "charts" {
			if c.opts.Verbose {
				fmt.Printf("Skipping directory %s - it's under charts/ directory (preserving chart structure)\n", originalDirPath)
			}
			continue
		}

		// Directory exists and does NOT have 'charts' as immediate parent, remove it
		if c.opts.Verbose || c.opts.ShowDeleted {
			fmt.Printf("Found file dependency directory: %s\n", originalDirPath)
		}

		if !c.opts.DryRun {
			if err := os.RemoveAll(originalDirPath); err != nil {
				return fmt.Errorf("failed to remove directory %s: %w", originalDirPath, err)
			}
			c.deletedPaths = append(c.deletedPaths, originalDirPath)
		} else {
			c.deletedPaths = append(c.deletedPaths, originalDirPath)
		}
	}

	return nil
}

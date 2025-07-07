package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// ChartYaml represents the structure of a Chart.yaml file
type ChartYaml struct {
	Dependencies []struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		// Other fields omitted for simplicity
	} `yaml:"dependencies"`
}

// Dependency represents a chart dependency with name and version
type Dependency struct {
	Name    string
	Version string
}

// DependencyKey generates a unique key for a dependency
func (d Dependency) Key() string {
	return fmt.Sprintf("%s-%s", d.Name, d.Version)
}

// ChartPath represents a chart location in the filesystem
type ChartPath struct {
	Path string
	ParentPath string
}

var (
	// Global flags
	verbose bool
	
	// Dedup command flags
	outputDir   string
	package_    bool
	dryRun      bool
	showDeleted bool
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "optimize",
		Short: "Optimize Helm charts",
		Long: `A Helm plugin that optimizes Helm charts in various ways.

This plugin provides multiple optimization features for Helm charts,
helping to improve performance, reduce size, and enhance usability.`,
	}
	
	// Add global flags to root command
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	
	// Add subcommands
	rootCmd.AddCommand(newDedupCmd())
	
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// newDedupCmd creates the dedup subcommand
func newDedupCmd() *cobra.Command {
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

func runDedup(cmd *cobra.Command, args []string) error {
	chartPath := args[0]
	
	// Validate chart path
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		return fmt.Errorf("chart path '%s' does not exist", chartPath)
	}

	// Set default output directory if not specified
	if outputDir == "" {
		outputDir = chartPath
	}
	
	fmt.Printf("Starting deduplication for chart at '%s'...\n", chartPath)
	
	// Create the deduplicator
	deduplicator := NewDeduplicator(verbose, dryRun, showDeleted)
	
	// Run the deduplication algorithm
	deletedPaths, err := deduplicator.DeduplicateChart(chartPath)
	if err != nil {
		return fmt.Errorf("deduplication failed: %v", err)
	}
	
	// Report results
	fmt.Printf("Deduplication completed. %d duplicate dependencies removed.\n", len(deletedPaths))
	
	// Package chart if requested
	if package_ && !dryRun {
		fmt.Println("Packaging deduplicated chart...")
		packageChart(chartPath)
	}
	
	return nil
}

// Deduplicator manages the dependency deduplication process
type Deduplicator struct {
	// Maps dependency name-version to the chart paths where it was found
	overallDependencies map[string][]ChartPath
	// List of paths to dependencies that will be kept
	currentDependencies []string
	// List of paths to dependencies that will be deleted
	deleteDependencies []string
	// Mutex for thread safety
	mu        sync.Mutex
	verbose   bool
	dryRun    bool
	showDeleted bool
}

// NewDeduplicator creates a new Deduplicator
func NewDeduplicator(verbose, dryRun, showDeleted bool) *Deduplicator {
	return &Deduplicator{
		overallDependencies: make(map[string][]ChartPath),
		currentDependencies: []string{},
		deleteDependencies:  []string{},
		verbose:             verbose,
		dryRun:              dryRun,
		showDeleted:         showDeleted,
	}
}

// DeduplicateChart performs dependency deduplication on a chart
func (d *Deduplicator) DeduplicateChart(chartPath string) ([]string, error) {
	// Start the deduplication process from the root chart path
	err := d.processDependencies(chartPath)
	if err != nil {
		return nil, err
	}
	
	// Delete duplicate dependencies
	if !d.dryRun {
		for _, path := range d.deleteDependencies {
			if d.verbose || d.showDeleted {
				fmt.Printf("Removing duplicate dependency: %s\n", path)
			}
			if err := os.RemoveAll(path); err != nil {
				return nil, fmt.Errorf("failed to remove %s: %v", path, err)
			}
		}
	} else if d.showDeleted {
		fmt.Println("Dry run - would delete these directories:")
		for _, path := range d.deleteDependencies {
			fmt.Printf("  %s\n", path)
		}
	}
	
	return d.deleteDependencies, nil
}

// processDependencies processes dependencies for the chart at the given path
func (d *Deduplicator) processDependencies(chartPath string) error {
	// Look for Chart.yaml file
	chartYamlPath := filepath.Join(chartPath, "Chart.yaml")
	
	// Check if Chart.yaml exists
	if _, err := os.Stat(chartYamlPath); err == nil {
		// Read and parse Chart.yaml
		chartYaml, err := readChartYaml(chartYamlPath)
		if err != nil {
			return err
		}
		
		if d.verbose {
			fmt.Printf("Processing dependencies in %s\n", chartYamlPath)
		}
		
		// Process dependencies in Chart.yaml
		for _, dep := range chartYaml.Dependencies {
			dependency := Dependency{
				Name:    dep.Name,
				Version: dep.Version,
			}
			
			depKey := dependency.Key()
			depPath := filepath.Join(chartPath, "charts", dependency.Name)
			
			d.mu.Lock()
			// Get parent directory path to check context
			parentPath := filepath.Dir(chartPath)
			
			// Check if dependency already exists in overall dependencies
			if paths, found := d.overallDependencies[depKey]; found {
				// Check if this is a true duplicate or a specialized version
				isTrueDuplicate := false
				
				// Basic rule: charts at the same level of hierarchy with the same parent are considered duplicates
				// Charts in different parts of hierarchy should be preserved
				for _, existingChartPath := range paths {
					if existingChartPath.ParentPath == parentPath {
						isTrueDuplicate = true
						break
					}
				}
				
				if isTrueDuplicate {
					// Duplicate found - add to delete dependencies
					d.deleteDependencies = append(d.deleteDependencies, depPath)
					if d.verbose {
						fmt.Printf("  Found duplicate dependency %s at %s (original at %s)\n", 
							depKey, depPath, paths[0].Path)
					}
				} else {
					// Similar dependency in different context - keep it
					newChartPath := ChartPath{Path: depPath, ParentPath: parentPath}
					d.overallDependencies[depKey] = append(paths, newChartPath)
					d.currentDependencies = append(d.currentDependencies, depPath)
					if d.verbose {
						fmt.Printf("  Found contextual dependency %s at %s (keeping)\n", depKey, depPath)
					}
				}
			} else {
				// New dependency - add to overall dependencies and current dependencies
				newChartPath := ChartPath{Path: depPath, ParentPath: parentPath}
				d.overallDependencies[depKey] = []ChartPath{newChartPath}
				d.currentDependencies = append(d.currentDependencies, depPath)
				if d.verbose {
					fmt.Printf("  Found new dependency %s at %s\n", depKey, depPath)
				}
			}
			d.mu.Unlock()
		}
	}
	
	// Check for charts directory
	chartsDir := filepath.Join(chartPath, "charts")
	if _, err := os.Stat(chartsDir); err == nil {
		// Read chart directory entries
		entries, err := os.ReadDir(chartsDir)
		if err != nil {
			return err
		}
		
		// For each entry in the charts directory
		for _, entry := range entries {
			if entry.IsDir() {
				subChartPath := filepath.Join(chartsDir, entry.Name())
				
				// Skip directories that are marked for deletion
				skipDir := false
				for _, deletePath := range d.deleteDependencies {
					if deletePath == subChartPath {
						skipDir = true
						break
					}
				}
				
				if !skipDir {
					// Recursively process the subchart
					if err := d.processDependencies(subChartPath); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// readChartYaml reads and parses a Chart.yaml file
func readChartYaml(path string) (*ChartYaml, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var chartYaml ChartYaml
	if err := yaml.Unmarshal(data, &chartYaml); err != nil {
		return nil, err
	}
	
	return &chartYaml, nil
}

// packageChart packages a chart
func packageChart(chartPath string) error {
	fmt.Println("Packaging functionality not implemented yet.")
	return nil
}

// loadChart loads a chart from the specified path
func loadChart(path string) (*chart.Chart, error) {
	chartRequested, err := loader.Load(path)
	if err != nil {
		return nil, err
	}
	return chartRequested, nil
}

// analyzeChartDependencies analyzes the dependencies of a chart
func analyzeChartDependencies(c *chart.Chart) ([]*chart.Dependency, error) {
	return c.Metadata.Dependencies, nil
}

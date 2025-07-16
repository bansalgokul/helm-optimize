package dedup

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

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
	Path       string
	ParentPath string
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
	mu          sync.Mutex
	opts        Options
}

// NewDeduplicator creates a new Deduplicator
func NewDeduplicator(opts Options) *Deduplicator {
	return &Deduplicator{
		overallDependencies: make(map[string][]ChartPath),
		currentDependencies: []string{},
		deleteDependencies:  []string{},
		opts:                opts,
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
	if !d.opts.DryRun {
		for _, path := range d.deleteDependencies {
			if d.opts.Verbose || d.opts.ShowDeleted {
				fmt.Printf("Removing duplicate dependency: %s\n", path)
			}
			if err := os.RemoveAll(path); err != nil {
				return nil, fmt.Errorf("failed to remove %s: %v", path, err)
			}
		}
	} else if d.opts.ShowDeleted {
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
		
		if d.opts.Verbose {
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
				var originalPath string
				
				// Basic rule: charts at the same level of hierarchy with the same parent are considered duplicates
				// Charts in different parts of hierarchy should be preserved
				for _, existingChartPath := range paths {
					// Store original path for reference
					if originalPath == "" {
						originalPath = existingChartPath.Path
					}
					
					// If exact same path, skip it entirely - this is not a duplicate but the same dependency
					if existingChartPath.Path == depPath {
						// This is the exact same path, not a duplicate
						isTrueDuplicate = false
						break
					}
					
					// Check for duplicate at same parent level
					if existingChartPath.ParentPath == parentPath {
						isTrueDuplicate = true
					}
				}
				
				if isTrueDuplicate {
					// Duplicate found - but double check that we're not deleting the original path
					if depPath != originalPath {
						d.deleteDependencies = append(d.deleteDependencies, depPath)
						if d.opts.Verbose {
							fmt.Printf("  Found duplicate dependency %s at %s (original at %s)\n", 
								depKey, depPath, originalPath)
						}
					} else {
						// This is the original path, we shouldn't delete it
						newChartPath := ChartPath{Path: depPath, ParentPath: parentPath}
						d.overallDependencies[depKey] = append(paths, newChartPath)
						d.currentDependencies = append(d.currentDependencies, depPath)
						if d.opts.Verbose {
							fmt.Printf("  Found original dependency %s at %s (keeping)\n", depKey, depPath)
						}
					}
				} else {
					// Similar dependency in different context - keep it
					newChartPath := ChartPath{Path: depPath, ParentPath: parentPath}
					d.overallDependencies[depKey] = append(paths, newChartPath)
					d.currentDependencies = append(d.currentDependencies, depPath)
					if d.opts.Verbose {
						fmt.Printf("  Found contextual dependency %s at %s (keeping)\n", depKey, depPath)
					}
				}
			} else {
				// New dependency - add to overall dependencies and current dependencies
				newChartPath := ChartPath{Path: depPath, ParentPath: parentPath}
				d.overallDependencies[depKey] = []ChartPath{newChartPath}
				d.currentDependencies = append(d.currentDependencies, depPath)
				if d.opts.Verbose {
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

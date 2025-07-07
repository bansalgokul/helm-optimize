package dedup

import (
	"os"
	"gopkg.in/yaml.v3"
)

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

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WrapperProfile defines the structure for a single wrapper configuration profile.
// Corresponds to an entry under the 'wrappers' key in the YAML file.
type WrapperProfile struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"` // Placeholders like {{env:VAR}}, {{keyring:svc:acc}}, {{file:path}}
	Alias   string            `yaml:"alias,omitempty"`
}

// WrapperConfig defines the top-level structure of the YAML configuration file.
// It contains a map of profile names to their definitions.
type WrapperConfig struct {
	Wrappers map[string]WrapperProfile `yaml:"wrappers"`
}

// LoadWrapperConfig reads the specified YAML file and parses it into WrapperConfig struct.
func LoadWrapperConfig(filePath string) (*WrapperConfig, error) {
	// Read the YAML file content
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wrapper config file '%s': %w", filePath, err)
	}

	// Parse the YAML content
	var config WrapperConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wrapper config file '%s': %w", filePath, err)
	}

	if config.Wrappers == nil {
		// Handle case where the file is valid YAML but the 'wrappers' key is missing or null
		// Initialize an empty map to avoid nil pointer issues later
		config.Wrappers = make(map[string]WrapperProfile)
		// Optionally log a warning or return an error if the structure is mandatory
		// log.Printf("Warning: Wrapper config file '%s' is missing 'wrappers' map.", filePath)
	}

	return &config, nil
} 
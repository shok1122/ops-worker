package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type CheckConfig struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Schedule string                 `yaml:"schedule"`
	Options  map[string]interface{} `yaml:"options"`
}

type ChecksConfig struct {
	Checks []CheckConfig `yaml:"checks"`
}

func LoadChecks(path string) (*ChecksConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading checks file: %w", err)
	}

	cfg := &ChecksConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing checks file: %w", err)
	}

	return cfg, nil
}

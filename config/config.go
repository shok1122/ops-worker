package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	TLS      bool   `yaml:"tls"`
	Password string `yaml:"password"`
	Timeout  int    `yaml:"timeout"`
}

type HealthcheckConfig struct {
	Schedule string `yaml:"schedule"`
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Healthcheck HealthcheckConfig `yaml:"healthcheck"`
	ChecksFile  string            `yaml:"checks_file"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults
	if cfg.Server.Timeout == 0 {
		cfg.Server.Timeout = 10
	}
	if cfg.ChecksFile == "" {
		cfg.ChecksFile = "/etc/ops-worker/checks.yaml"
	}

	return cfg, nil
}

func (c *Config) ReportURL() string {
	scheme := "http"
	if c.Server.TLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/api/v1/report", scheme, c.Server.Host, c.Server.Port)
}

func (c *Config) HealthURL() string {
	scheme := "http"
	if c.Server.TLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/api/v1/health", scheme, c.Server.Host, c.Server.Port)
}

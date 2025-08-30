package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Discovery GitDiscovery `yaml:"discovery"`
}

// GitDiscovery configuration for git branch-based file discovery
type GitDiscovery struct {
	Enabled  bool   `yaml:"enabled"`
	Template string `yaml:"template"`
}

var (
	// configCache stores the loaded configuration for the session
	configCache *Config
	// configOnce ensures configuration is loaded only once
	configOnce sync.Once
	// configError stores any error from loading configuration
	configError error
)

// LoadConfig loads configuration from available sources in precedence order
// Configuration is cached after first load for performance
func LoadConfig() (*Config, error) {
	configOnce.Do(func() {
		configCache, configError = loadConfigUncached()
	})
	return configCache, configError
}

// loadConfigUncached loads configuration without caching
func loadConfigUncached() (*Config, error) {
	// Check for config in order of precedence
	paths := []string{
		"./.go-tasks.yml",
		expandHome("~/.config/go-tasks/config.yml"),
	}

	for _, path := range paths {
		if cfg, err := loadConfigFile(path); err == nil {
			return cfg, nil
		}
	}

	// Return default config if no file found
	return defaultConfig(), nil
}

// loadConfigFile loads configuration from a specific file
func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	// Apply defaults for missing values
	if cfg.Discovery.Template == "" {
		cfg.Discovery.Template = "specs/{branch}/tasks.md"
	}

	return &cfg, nil
}

// defaultConfig returns the default configuration
func defaultConfig() *Config {
	return &Config{
		Discovery: GitDiscovery{
			Enabled:  false,
			Template: "specs/{branch}/tasks.md",
		},
	}
}

// expandHome expands the ~ character to the user's home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// resetConfigCache resets the configuration cache for testing
func resetConfigCache() {
	configCache = nil
	configOnce = sync.Once{}
	configError = nil
}

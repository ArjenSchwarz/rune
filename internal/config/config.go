package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	// Build config search paths in precedence order.
	// The repo-root .rune.yml is checked first so the tool works from
	// any subdirectory of the repository.
	paths := []string{}

	if root, err := getRepoRoot(); err == nil {
		paths = append(paths, filepath.Join(root, ".rune.yml"))
	}
	// CWD-relative path as fallback (covers non-git usage)
	paths = append(paths, "./.rune.yml")
	paths = append(paths, expandHome("~/.config/rune/config.yml"))

	for _, path := range paths {
		cfg, err := loadConfigFile(path)
		if err == nil {
			return cfg, nil
		}
		// File not found is expected — try the next path in the search order.
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		// File exists but is invalid — surface the error immediately.
		return nil, err
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
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		// Empty file — treat as zero-value config with defaults applied below
		if errors.Is(err, io.EOF) {
			cfg = Config{}
		} else {
			return nil, fmt.Errorf("parsing config file %s: %w", path, err)
		}
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
			Enabled:  true,
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

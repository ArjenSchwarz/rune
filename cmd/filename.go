package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/go-tasks/internal/config"
)

// resolveFilename resolves the task filename from args or git discovery
func resolveFilename(args []string) (string, error) {
	// Try explicit argument first
	if len(args) > 0 {
		return args[0], nil
	}

	// Try git discovery if enabled
	cfg, err := config.LoadConfig()
	if err != nil {
		// Log warning but continue with manual file requirement
		if verbose {
			fmt.Printf("Warning: failed to load config: %v\n", err)
		}
	} else if cfg.Discovery.Enabled {
		if path, err := config.DiscoverFileFromBranch(cfg.Discovery.Template); err == nil {
			return path, nil
		} else if verbose {
			fmt.Printf("Git discovery failed: %v\n", err)
		}
		// Fall through to require explicit file
	}

	return "", fmt.Errorf("no filename specified and git discovery failed or disabled")
}

package cmd

import (
	"fmt"
	"os"

	"github.com/arjenschwarz/rune/internal/config"
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
		return "", fmt.Errorf("configuration error: %w", err)
	}
	if cfg.Discovery.Enabled {
		if path, err := config.DiscoverFileFromBranch(cfg.Discovery.Template); err == nil {
			// The discovered path is relative to the repo root. Run the command
			// from there so it opens the right file and the file passes the
			// working-directory containment check on write, even when rune is
			// invoked from a subdirectory.
			root, err := config.RepoRoot()
			if err != nil {
				return "", fmt.Errorf("finding repo root: %w", err)
			}
			if err := os.Chdir(root); err != nil {
				return "", fmt.Errorf("changing to repo root: %w", err)
			}
			return path, nil
		} else if verbose {
			fmt.Printf("Git discovery failed: %v\n", err)
		}
		// Fall through to require explicit file
	}

	return "", fmt.Errorf("no filename specified and git discovery failed or disabled")
}

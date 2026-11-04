package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
			return discoveredPath(path)
		} else {
			// stderr keeps stdout clean for JSON output (T-2127)
			verboseStderr("Git discovery failed: %v", err)
		}
		// Fall through to require explicit file
	}

	return "", fmt.Errorf("no filename specified and git discovery failed or disabled")
}

// discoveredPath turns a repo-relative path from git discovery into one the
// command can use. From the repo root the path is returned unchanged. From a
// subdirectory, rune switches to the repo root, so the file passes the
// working-directory containment check on write, and returns the absolute path,
// so the path rune reports is still valid in the caller's directory.
func discoveredPath(path string) (string, error) {
	root, err := config.RepoRoot()
	if err != nil {
		return "", fmt.Errorf("finding repo root: %w", err)
	}
	if isCurrentDir(root) {
		return path, nil
	}
	if err := os.Chdir(root); err != nil {
		return "", fmt.Errorf("changing to repo root: %w", err)
	}
	return filepath.Join(root, path), nil
}

// isCurrentDir reports whether dir is the current working directory.
func isCurrentDir(dir string) bool {
	cwdInfo, err := os.Stat(".")
	if err != nil {
		return false
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return os.SameFile(cwdInfo, dirInfo)
}

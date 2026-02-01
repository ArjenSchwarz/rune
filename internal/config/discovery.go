package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// DiscoverFileFromBranch discovers a task file based on the current git branch
// and the configured template pattern. It strips the branch prefix (everything
// before and including the first /) to extract the spec name:
//   - "feature/my-feature" -> "my-feature"
//   - "feature/sub/deep" -> "sub/deep"
//   - "main" -> "main" (no change)
//
// It tries multiple candidate paths: first the stripped branch name, then the
// full branch name. If both paths exist, the stripped path takes precedence.
func DiscoverFileFromBranch(template string) (string, error) {
	branch, err := getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("getting git branch: %w", err)
	}

	if isSpecialGitState(branch) {
		return "", fmt.Errorf("special git state detected: %s (please specify file explicitly)", branch)
	}

	// Strip everything before and including first slash (branch prefix)
	// e.g., "feature/my-feature" -> "my-feature"
	// e.g., "feature/sub/deep" -> "sub/deep"
	strippedBranch := branch
	if _, after, found := strings.Cut(branch, "/"); found {
		strippedBranch = after
	}

	// Try stripped name first, then full name
	candidates := []string{
		strings.ReplaceAll(template, "{branch}", strippedBranch),
	}
	if strippedBranch != branch {
		candidates = append(candidates, strings.ReplaceAll(template, "{branch}", branch))
	}

	for _, path := range candidates {
		if fileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("task file not found for branch %q (tried: %s)",
		branch, strings.Join(candidates, ", "))
}

// getCurrentBranch is a variable that points to the function for getting current git branch
// This allows for easy mocking in tests
var getCurrentBranch = getCurrentBranchImpl

// getCurrentBranchImpl gets the current git branch name
func getCurrentBranchImpl() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	// Set timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd.Stdin = nil // Ensure no stdin

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git command timed out")
		}
		return "", fmt.Errorf("git command failed: %w (stderr: %s)", err, errOut.String())
	}

	branch := strings.TrimSpace(out.String())

	// Sanitize branch name to prevent injection
	if strings.ContainsAny(branch, ";&|<>$`\"'\\") {
		return "", fmt.Errorf("invalid characters in branch name: %s", branch)
	}

	if branch == "" {
		return "", fmt.Errorf("git branch name is empty")
	}

	return branch, nil
}

// isSpecialGitState checks if the git repository is in a special state
// that should require explicit file specification
func isSpecialGitState(branch string) bool {
	specialStates := []string{
		"HEAD",        // detached HEAD
		"(no branch)", // also detached HEAD in some git versions
	}

	if slices.Contains(specialStates, branch) {
		return true
	}

	// Check for rebase/merge states (branches typically contain these strings)
	if strings.Contains(branch, "rebase") || strings.Contains(branch, "merge") {
		return true
	}

	return false
}

// fileExists checks if a file exists and is accessible
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

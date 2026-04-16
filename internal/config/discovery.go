package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// Resolve the repo root so file checks work from any subdirectory
	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("finding repo root: %w", err)
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
		if fileExists(filepath.Join(repoRoot, path)) {
			return path, nil
		}
	}

	return "", fmt.Errorf("task file not found for branch %q (tried: %s)",
		branch, strings.Join(candidates, ", "))
}

// getCurrentBranch is a variable that points to the function for getting current git branch
// This allows for easy mocking in tests
var getCurrentBranch = getCurrentBranchImpl

// getRepoRoot is a variable that points to the function for getting the repo root.
// This allows for easy mocking in tests.
var getRepoRoot = getRepoRootImpl

// gitCommandTimeout controls how long to wait for git commands before timing out.
// Exposed as a variable to allow tests to use shorter durations.
var gitCommandTimeout = 5 * time.Second

// getCurrentBranchImpl gets the current git branch name
func getCurrentBranchImpl() (string, error) {
	// Create context with timeout to prevent hanging on slow/unresponsive git
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	// WaitDelay closes pipes after the process is killed, preventing hangs when
	// child processes (e.g., shell spawning sleep) inherit the pipes. This adds
	// to the total wait time: effective max = gitCommandTimeout + WaitDelay.
	cmd.WaitDelay = 500 * time.Millisecond
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
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

// getRepoRootImpl returns the root directory of the current git repository
// by running "git rev-parse --show-toplevel".
func getRepoRootImpl() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.WaitDelay = 500 * time.Millisecond
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git command timed out")
		}
		return "", fmt.Errorf("git command failed: %w (stderr: %s)", err, errOut.String())
	}

	root := strings.TrimSpace(out.String())
	if root == "" {
		return "", fmt.Errorf("git repo root is empty")
	}

	return root, nil
}

// isSpecialGitState checks if the git repository is in a special state
// that should require explicit file specification. Only actual detached or
// unnamed states are rejected — normal branch names that happen to contain
// words like "merge" or "rebase" are allowed.
func isSpecialGitState(branch string) bool {
	specialStates := []string{
		"HEAD",        // detached HEAD
		"(no branch)", // also detached HEAD in some git versions
	}

	return slices.Contains(specialStates, branch)
}

// fileExists checks if a file exists and is accessible
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

package config

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFileFromBranch(t *testing.T) {
	tests := map[string]struct {
		template      string
		branch        string
		createPaths   []string // paths to create for testing
		expectError   bool
		expectedPath  string
		errorContains string
		gitError      error
		specialState  bool
	}{
		"prefixed branch finds stripped path": {
			template:     "specs/{branch}/tasks.md",
			branch:       "specs/my-feature",
			createPaths:  []string{"specs/my-feature/tasks.md"},
			expectError:  false,
			expectedPath: "specs/my-feature/tasks.md",
		},
		"prefixed branch falls back to full path": {
			template:     "specs/{branch}/tasks.md",
			branch:       "feature/auth",
			createPaths:  []string{"specs/feature/auth/tasks.md"},
			expectError:  false,
			expectedPath: "specs/feature/auth/tasks.md",
		},
		"stripped path takes precedence over full path": {
			template:     "specs/{branch}/tasks.md",
			branch:       "specs/my-feature",
			createPaths:  []string{"specs/my-feature/tasks.md", "specs/specs/my-feature/tasks.md"},
			expectError:  false,
			expectedPath: "specs/my-feature/tasks.md",
		},
		"file not found shows all tried paths": {
			template:      "specs/{branch}/tasks.md",
			branch:        "feature/nonexistent",
			createPaths:   nil,
			expectError:   true,
			errorContains: "tried: specs/nonexistent/tasks.md, specs/feature/nonexistent/tasks.md", // Prefix stripped gives 'nonexistent'
		},
		"single component branch - file not found shows single path": {
			template:      "specs/{branch}/tasks.md",
			branch:        "main",
			createPaths:   nil,
			expectError:   true,
			errorContains: "tried: specs/main/tasks.md",
		},
		"git error": {
			template:    "specs/{branch}/tasks.md",
			gitError:    context.DeadlineExceeded,
			expectError: true,
		},
		"special git state - detached HEAD": {
			template:     "specs/{branch}/tasks.md",
			branch:       "HEAD",
			specialState: true,
			expectError:  true,
		},
		"special git state - no branch": {
			template:     "specs/{branch}/tasks.md",
			branch:       "(no branch)",
			specialState: true,
			expectError:  true,
		},
		"template with complex path": {
			template:     "projects/{branch}/docs/tasks.md",
			branch:       "feature/complex-name",
			createPaths:  []string{"projects/complex-name/docs/tasks.md"},
			expectError:  false,
			expectedPath: "projects/complex-name/docs/tasks.md",
		},
		"branch with single component": {
			template:     "specs/{branch}/tasks.md",
			branch:       "main",
			createPaths:  []string{"specs/main/tasks.md"},
			expectError:  false,
			expectedPath: "specs/main/tasks.md",
		},
		"branch with multiple slashes": {
			template:     "specs/{branch}/tasks.md",
			branch:       "feature/auth/oauth",
			createPaths:  []string{"specs/auth/oauth/tasks.md"},
			expectError:  false,
			expectedPath: "specs/auth/oauth/tasks.md",
		},
		"branch with multiple slashes falls back to full": {
			template:     "specs/{branch}/tasks.md",
			branch:       "feature/auth/oauth",
			createPaths:  []string{"specs/feature/auth/oauth/tasks.md"},
			expectError:  false,
			expectedPath: "specs/feature/auth/oauth/tasks.md",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a temp directory for testing file existence
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			defer func() {
				os.Chdir(originalDir)
			}()
			os.Chdir(tempDir)

			// Create the test files if specified
			for _, path := range tc.createPaths {
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}
				if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Mock the git command by temporarily replacing getCurrentBranch
			originalGetCurrentBranch := getCurrentBranch
			defer func() {
				getCurrentBranch = originalGetCurrentBranch
			}()

			if tc.gitError != nil {
				getCurrentBranch = func() (string, error) {
					return "", tc.gitError
				}
			} else {
				getCurrentBranch = func() (string, error) {
					return tc.branch, nil
				}
			}

			// Test the function
			result, err := DiscoverFileFromBranch(tc.template)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tc.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tc.expectedPath {
				t.Errorf("Expected path %s, got %s", tc.expectedPath, result)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tests := map[string]struct {
		name         string
		setupFunc    func(*testing.T) (cleanup func())
		expectedErr  bool
		expectedText string // for successful cases
		errorText    string // for error cases
	}{
		"valid branch name": {
			setupFunc: func(t *testing.T) func() {
				return setupMockGitCommand(t, "feature/auth\n", "", 0)
			},
			expectedText: "feature/auth",
		},
		"branch with special characters in name": {
			setupFunc: func(t *testing.T) func() {
				return setupMockGitCommand(t, "feature-auth_123\n", "", 0)
			},
			expectedText: "feature-auth_123",
		},
		"empty branch name": {
			setupFunc: func(t *testing.T) func() {
				return setupMockGitCommand(t, "\n", "", 0)
			},
			expectedErr: true,
			errorText:   "git branch name is empty",
		},
		"git command fails": {
			setupFunc: func(t *testing.T) func() {
				return setupMockGitCommand(t, "", "fatal: not a git repository", 1)
			},
			expectedErr: true,
			errorText:   "git command failed",
		},
		"branch with dangerous characters": {
			setupFunc: func(t *testing.T) func() {
				return setupMockGitCommand(t, "branch;rm -rf /\n", "", 0)
			},
			expectedErr: true,
			errorText:   "invalid characters in branch name",
		},
		"detached HEAD": {
			setupFunc: func(t *testing.T) func() {
				return setupMockGitCommand(t, "HEAD\n", "", 0)
			},
			expectedText: "HEAD", // This is valid for getCurrentBranch, but will be caught by isSpecialGitState
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cleanup := tc.setupFunc(t)
			defer cleanup()

			result, err := getCurrentBranch()

			if tc.expectedErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tc.errorText != "" && !strings.Contains(err.Error(), tc.errorText) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorText, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tc.expectedText {
				t.Errorf("Expected '%s', got '%s'", tc.expectedText, result)
			}
		})
	}
}

func TestIsSpecialGitState(t *testing.T) {
	tests := map[string]struct {
		branch   string
		expected bool
	}{
		"normal branch":              {"main", false},
		"feature branch":             {"feature/auth", false},
		"detached HEAD":              {"HEAD", true},
		"no branch":                  {"(no branch)", true},
		"rebase state":               {"main-rebase", true},
		"merge state":                {"feature-merge", true},
		"branch with rebase in name": {"rebase-feature", true}, // This might be overly cautious but safer
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := isSpecialGitState(tc.branch)
			if result != tc.expected {
				t.Errorf("For branch '%s', expected %t, got %t", tc.branch, tc.expected, result)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := map[string]struct {
		path     string
		expected bool
	}{
		"existing file":    {testFile, true},
		"nonexistent file": {filepath.Join(tempDir, "nonexistent.md"), false},
		"directory":        {testDir, false},
		"nonexistent path": {filepath.Join(tempDir, "nonexistent/file.md"), false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := fileExists(tc.path)
			if result != tc.expected {
				t.Errorf("For path '%s', expected %t, got %t", tc.path, tc.expected, result)
			}
		})
	}
}

// Helper function to mock git commands for testing
func setupMockGitCommand(t *testing.T, stdout, stderr string, exitCode int) func() {
	if testing.Short() {
		t.Skip("Skipping test that modifies global state in short mode")
	}

	// Create a mock git script
	tempDir := t.TempDir()
	mockGitPath := filepath.Join(tempDir, "git")

	scriptContent := `#!/bin/sh
if [ "$1" = "rev-parse" ] && [ "$2" = "--abbrev-ref" ] && [ "$3" = "HEAD" ]; then
    printf "%s" "` + stdout + `"
    printf "%s" "` + stderr + `" >&2
    exit ` + string(rune(exitCode+'0')) + `
fi
echo "Unexpected git command: $*" >&2
exit 1
`

	if err := os.WriteFile(mockGitPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create mock git script: %v", err)
	}

	// Modify PATH to use our mock git
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+":"+originalPath)

	return func() {
		os.Setenv("PATH", originalPath)
	}
}

// Integration test that requires real git (only runs with INTEGRATION=1)
func TestDiscoverFileFromBranchIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test (set INTEGRATION=1 to run)")
	}

	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		t.Skip("Not in a git repository, skipping integration test")
	}

	// Create a temporary task file in the current branch structure
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repo for testing
	runCommand(t, "git", "init")
	runCommand(t, "git", "config", "user.email", "test@example.com")
	runCommand(t, "git", "config", "user.name", "Test User")

	// Create initial commit so we can create branches
	if err := os.WriteFile("README.md", []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	runCommand(t, "git", "add", "README.md")
	runCommand(t, "git", "commit", "-m", "Initial commit")
	runCommand(t, "git", "checkout", "-b", "test-branch")

	// Create the expected file structure
	expectedPath := "specs/test-branch/tasks.md"
	if err := os.MkdirAll(filepath.Dir(expectedPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(expectedPath, []byte("# Test Tasks"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test discovery
	result, err := DiscoverFileFromBranch("specs/{branch}/tasks.md")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, result)
	}
}

func runCommand(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Command %s %v failed: %v\nStderr: %s", name, args, err, stderr.String())
	}
}

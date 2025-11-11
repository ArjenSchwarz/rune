package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// originalWorkingDir stores the working directory when tests start, before changing to temp directories
var originalWorkingDir string

// runeBinaryPath stores the path to the compiled rune binary for integration tests
var runeBinaryPath string

func init() {
	// Capture the original working directory when the test package loads
	wd, err := os.Getwd()
	if err == nil {
		originalWorkingDir = wd
	}
}

// TestMain builds the rune binary before running integration tests
func TestMain(m *testing.M) {
	// Only build binary if running integration tests
	if os.Getenv("INTEGRATION") != "" {
		// Build the binary
		tmpDir, err := os.MkdirTemp("", "rune-integration-test")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
			os.Exit(1)
		}
		defer os.RemoveAll(tmpDir)

		runeBinaryPath = filepath.Join(tmpDir, "rune")
		buildCmd := exec.Command("go", "build", "-o", runeBinaryPath, "../")
		buildCmd.Dir = originalWorkingDir
		if output, err := buildCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to build rune binary: %v\n%s\n", err, output)
			os.Exit(1)
		}
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// getExamplePath resolves the absolute path to an example file from the project root
func getExamplePath(relativePath string) (string, error) {
	if originalWorkingDir == "" {
		return "", fmt.Errorf("original working directory not captured")
	}

	// Go up one level from cmd/ to project root
	projectRoot := filepath.Dir(originalWorkingDir)
	examplePath := filepath.Join(projectRoot, relativePath)

	// Verify the file exists
	if _, err := os.Stat(examplePath); err != nil {
		return "", fmt.Errorf("example file not found at %s: %w", examplePath, err)
	}

	return examplePath, nil
}

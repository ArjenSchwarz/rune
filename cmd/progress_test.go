package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

func TestRunProgressDryRunFormats(t *testing.T) {
	tempDir := filepath.Join(".", "test-tmp-progress-dryrun")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		format       string
		validateJSON bool
		expectPhrase string
	}{
		"dry run with json format": {
			format:       "json",
			validateJSON: true,
		},
		"dry run with markdown format": {
			format:       "markdown",
			expectPhrase: "**",
		},
		"dry run with table format": {
			format:       "table",
			expectPhrase: "Would mark task",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test-"+strings.ReplaceAll(name, " ", "-")+".md")

			// Create a task file with a pending task
			tl := task.NewTaskList("Test Tasks")
			tl.AddTask("", "Setup environment", "")
			if err := tl.WriteFile(filename); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Read initial content
			initialContent, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read initial file: %v", err)
			}

			// Set flags
			oldDryRun := dryRun
			oldFormat := format
			dryRun = true
			format = tt.format
			defer func() {
				dryRun = oldDryRun
				format = oldFormat
			}()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			cmd := &cobra.Command{}
			err = runProgress(cmd, []string{filename, "1"})

			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("Unexpected error in dry run: %v", err)
			}

			// Verify file was NOT modified
			finalContent, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read final file: %v", err)
			}
			if !bytes.Equal(initialContent, finalContent) {
				t.Fatal("File was modified during dry run")
			}

			// Format-specific validation
			if tt.validateJSON {
				var resp ProgressResponse
				if err := json.Unmarshal([]byte(output), &resp); err != nil {
					t.Fatalf("Expected valid JSON output, got parse error: %v\nOutput was: %s", err, output)
				}
				if resp.TaskID != "1" {
					t.Errorf("Expected task_id '1', got '%s'", resp.TaskID)
				}
				if resp.Title != "Setup environment" {
					t.Errorf("Expected title 'Setup environment', got '%s'", resp.Title)
				}
				if !resp.DryRun {
					t.Error("Expected dry_run to be true in JSON response")
				}
				if resp.CurrentStatus != "pending" {
					t.Errorf("Expected current_status 'pending', got '%s'", resp.CurrentStatus)
				}
			}

			if tt.expectPhrase != "" {
				if !strings.Contains(output, tt.expectPhrase) {
					t.Errorf("Expected output to contain '%s', got: %s", tt.expectPhrase, output)
				}
			}
		})
	}
}

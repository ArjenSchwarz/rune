package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
)

func TestCreateCommand(t *testing.T) {
	tests := map[string]struct {
		title       string
		filename    string
		wantErr     bool
		wantContent []string
	}{
		"create basic task file": {
			title:    "My Project Tasks",
			filename: "test-tasks.md",
			wantErr:  false,
			wantContent: []string{
				"# My Project Tasks",
				"",
			},
		},
		"create with spaces in title": {
			title:    "Project with Spaces",
			filename: "spaces-tasks.md",
			wantErr:  false,
			wantContent: []string{
				"# Project with Spaces",
				"",
			},
		},
		"empty title": {
			title:    "",
			filename: "empty-tasks.md",
			wantErr:  false,
			wantContent: []string{
				"# ",
				"",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "go-tasks-create-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Create new task list and write to file
			tl := task.NewTaskList(tc.title)
			err = tl.WriteFile(tc.filename)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check file was created
			if _, err := os.Stat(tc.filename); os.IsNotExist(err) {
				t.Errorf("file %s was not created", tc.filename)
				return
			}

			// Read and verify content
			content, err := os.ReadFile(tc.filename)
			if err != nil {
				t.Errorf("failed to read created file: %v", err)
				return
			}

			lines := strings.Split(string(content), "\n")
			for i, wantLine := range tc.wantContent {
				if i >= len(lines) {
					t.Errorf("expected line %d to be %q, but file has only %d lines", i, wantLine, len(lines))
					continue
				}
				if lines[i] != wantLine {
					t.Errorf("line %d: expected %q, got %q", i, wantLine, lines[i])
				}
			}
		})
	}
}

func TestCreateCommandPathValidation(t *testing.T) {
	tests := map[string]struct {
		filename string
		wantErr  bool
		errMsg   string
	}{
		"valid relative path": {
			filename: "tasks.md",
			wantErr:  false,
		},
		"valid nested path": {
			filename: "project/tasks.md",
			wantErr:  false,
		},
		"path traversal attempt": {
			filename: "../../../etc/passwd",
			wantErr:  true,
			errMsg:   "path traversal",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "go-tasks-security-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Ensure parent directory exists for nested paths
			if strings.Contains(tc.filename, "/") {
				dir := filepath.Dir(tc.filename)
				os.MkdirAll(dir, 0755)
			}

			tl := task.NewTaskList("Test")
			err = tl.WriteFile(tc.filename)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q but got none", tc.errMsg)
				} else if !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

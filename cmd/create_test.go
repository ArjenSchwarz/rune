package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
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
			tempDir, err := os.MkdirTemp("", "rune-create-test")
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
			tempDir, err := os.MkdirTemp("", "rune-security-test")
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

func TestCreateCommandWithFrontMatter(t *testing.T) {
	tests := map[string]struct {
		title           string
		references      []string
		metadata        []string
		filename        string
		wantErr         bool
		wantFrontMatter bool
		checkContent    func(t *testing.T, content string)
	}{
		"single reference flag": {
			title:           "My Project",
			references:      []string{"README.md"},
			filename:        "tasks.md",
			wantFrontMatter: true,
			checkContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "---\n") {
					t.Error("expected front matter delimiter")
				}
				if !strings.Contains(content, "references:") {
					t.Error("expected references section")
				}
				if !strings.Contains(content, "- README.md") {
					t.Error("expected reference in front matter")
				}
			},
		},
		"multiple reference flags": {
			title:           "My Project",
			references:      []string{"README.md", "docs/guide.md", "CONTRIBUTING.md"},
			filename:        "tasks.md",
			wantFrontMatter: true,
			checkContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "references:\n") {
					t.Error("expected references section")
				}
				if !strings.Contains(content, "- README.md") {
					t.Error("expected first reference")
				}
				if !strings.Contains(content, "- docs/guide.md") {
					t.Error("expected second reference")
				}
				if !strings.Contains(content, "- CONTRIBUTING.md") {
					t.Error("expected third reference")
				}
			},
		},
		"single meta flag": {
			title:           "My Project",
			metadata:        []string{"author:John Doe"},
			filename:        "tasks.md",
			wantFrontMatter: true,
			checkContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "metadata:\n") {
					t.Error("expected metadata section")
				}
				if !strings.Contains(content, "author: John Doe") {
					t.Error("expected author metadata")
				}
			},
		},
		"multiple meta flags": {
			title:           "My Project",
			metadata:        []string{"author:John Doe", "version:1.0.0", "status:active"},
			filename:        "tasks.md",
			wantFrontMatter: true,
			checkContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "metadata:\n") {
					t.Error("expected metadata section")
				}
				if !strings.Contains(content, "author: John Doe") {
					t.Error("expected author metadata")
				}
				if !strings.Contains(content, "version: 1.0.0") {
					t.Error("expected version metadata")
				}
				if !strings.Contains(content, "status: active") {
					t.Error("expected status metadata")
				}
			},
		},
		"combined references and metadata": {
			title:           "My Project",
			references:      []string{"README.md", "LICENSE"},
			metadata:        []string{"priority:high", "team:backend"},
			filename:        "tasks.md",
			wantFrontMatter: true,
			checkContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "references:\n") {
					t.Error("expected references section")
				}
				if !strings.Contains(content, "- README.md") {
					t.Error("expected README reference")
				}
				if !strings.Contains(content, "- LICENSE") {
					t.Error("expected LICENSE reference")
				}
				if !strings.Contains(content, "metadata:\n") {
					t.Error("expected metadata section")
				}
				if !strings.Contains(content, "priority: high") {
					t.Error("expected priority metadata")
				}
				if !strings.Contains(content, "team: backend") {
					t.Error("expected team metadata")
				}
			},
		},
		"invalid metadata format": {
			title:    "My Project",
			metadata: []string{"invalid_format"},
			filename: "tasks.md",
			wantErr:  true,
		},
		"metadata with nested keys not supported": {
			title:    "My Project",
			metadata: []string{"author.name:John Doe"},
			filename: "tasks.md",
			wantErr:  true,
		},
		"metadata with concatenated values": {
			title:           "My Project",
			metadata:        []string{"tags:feature", "tags:enhancement", "tags:v2"},
			filename:        "tasks.md",
			wantFrontMatter: true,
			checkContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "metadata:\n") {
					t.Error("expected metadata section")
				}
				if !strings.Contains(content, "tags: feature,enhancement,v2") {
					t.Error("expected concatenated tags value")
				}
			},
		},
		"no front matter flags": {
			title:           "My Project",
			filename:        "tasks.md",
			wantFrontMatter: false,
			checkContent: func(t *testing.T, content string) {
				if strings.Contains(content, "---\n") {
					t.Error("unexpected front matter delimiter")
				}
				if !strings.Contains(content, "# My Project") {
					t.Error("expected title")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "rune-frontmatter-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Build FrontMatter if needed
			var fm *task.FrontMatter
			if len(tc.references) > 0 || len(tc.metadata) > 0 {
				fm = &task.FrontMatter{
					References: tc.references,
				}

				if len(tc.metadata) > 0 {
					parsedMeta, err := task.ParseMetadataFlags(tc.metadata)
					if err != nil {
						if tc.wantErr {
							// Expected error
							return
						}
						t.Fatalf("failed to parse metadata: %v", err)
					}
					fm.Metadata = parsedMeta
				}
			}

			// Create task list with front matter
			tl := task.NewTaskList(tc.title, fm)
			err = tl.WriteFile(tc.filename)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Read and verify content
			content, err := os.ReadFile(tc.filename)
			if err != nil {
				t.Errorf("failed to read created file: %v", err)
				return
			}

			// Check content using test-specific validator
			if tc.checkContent != nil {
				tc.checkContent(t, string(content))
			}
		})
	}
}

// setCreateFlags points the package-level create flags at the given values for
// the duration of the test and restores the previous values afterwards. The cmd
// package shares these globals across tests (see docs/agent-notes/testing.md).
func setCreateFlags(t *testing.T, title string, dry bool) {
	t.Helper()

	oldTitle, oldRefs, oldMeta, oldDryRun := createTitle, createReferences, createMetadata, dryRun
	t.Cleanup(func() {
		createTitle, createReferences, createMetadata, dryRun = oldTitle, oldRefs, oldMeta, oldDryRun
	})

	createTitle = title
	createReferences = nil
	createMetadata = nil
	dryRun = dry
}

// TestCreateCommandRejectsInvalidTitles verifies that `rune create` rejects
// titles that would produce an H1 heading the parser cannot read back — empty
// or whitespace-only titles (which render as a bare "# ") and titles containing
// newlines or other control characters (which split the heading) — as well as
// titles longer than the documented limit. Regression test for T-1500.
func TestCreateCommandRejectsInvalidTitles(t *testing.T) {
	tests := map[string]struct {
		title   string
		dryRun  bool
		wantErr string
	}{
		"newline":         {title: "Bad\nTitle", wantErr: "control characters"},
		"carriage return": {title: "Bad\rTitle", wantErr: "control characters"},
		"crlf":            {title: "Bad\r\nTitle", wantErr: "control characters"},
		"null byte":       {title: "Bad\x00Title", wantErr: "control characters"},
		"empty":           {title: "", wantErr: "cannot be empty"},
		"whitespace only": {title: "   ", wantErr: "cannot be empty"},
		"newline dry run": {title: "Bad\nTitle", dryRun: true, wantErr: "control characters"},
		"too long": {
			title:   strings.Repeat("a", task.MaxTitleLength+1),
			wantErr: fmt.Sprintf("title exceeds %d characters", task.MaxTitleLength),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setCreateFlags(t, tt.title, tt.dryRun)

			filename := "tasks.md"
			err := runCreate(&cobra.Command{}, []string{filename})

			if err == nil {
				t.Fatalf("expected error for title %q, got nil", tt.title)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}

			if _, statErr := os.Stat(filename); statErr == nil {
				t.Errorf("file %s should not have been created for invalid title %q", filename, tt.title)
			}
		})
	}
}

// TestCreateCommandWritesParseableFile is the positive half of the T-1500
// regression: a title accepted by `rune create` must produce a file that parses
// back with the same title. Without this, tightening title validation could
// pass its own tests while breaking the round trip it exists to protect.
func TestCreateCommandWritesParseableFile(t *testing.T) {
	titles := map[string]string{
		"plain":               "My Project Tasks",
		"tab is allowed":      "My\tTasks",
		"title at max length": strings.Repeat("a", task.MaxTitleLength),
	}

	for name, title := range titles {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setCreateFlags(t, title, false)

			filename := "tasks.md"
			if err := runCreate(&cobra.Command{}, []string{filename}); err != nil {
				t.Fatalf("runCreate() unexpected error: %v", err)
			}

			tl, err := task.ParseFile(filename)
			if err != nil {
				t.Fatalf("created file does not parse back: %v", err)
			}
			if tl.Title != title {
				t.Errorf("round-tripped title = %q, want %q", tl.Title, title)
			}
		})
	}
}

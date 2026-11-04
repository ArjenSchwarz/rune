package task

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Regression tests for T-2079: unrelated mutations rewrote the front matter
// from decoded values, dropping YAML comments. Comments must survive any
// mutation that does not change the commented key.

const commentedFrontMatterFile = `---
# project note
references:
  - "docs/guide.md" # keep this explanation
metadata:
  z: 'last'
  a: >-
    folded value
owner: arjen # unknown field comment
---
# Demo

- [ ] 1. Task
`

const commentedFrontMatterPhasesFile = `---
# project note
references:
  - "docs/guide.md" # keep this explanation
---
# Demo

## Phase One

- [ ] 1. Task
`

const commentOnlyFrontMatterFile = "---\n# Project handoff notes\n---\n# Tasks\n- [ ] 1. First\n"

const emptyCommentedFieldsFile = "---\nreferences: [] # filled in later\n# metadata note\nmetadata: {}\n---\n# Tasks\n- [ ] 1. First\n"

func updateAndWrite(t *testing.T, tl *TaskList, path string) string {
	t.Helper()
	if err := tl.UpdateStatus("1", Completed); err != nil {
		t.Fatal(err)
	}
	return writeAndRead(t, tl, path)
}

func TestMutationsPreserveFrontMatterComments(t *testing.T) {
	completed := Completed
	tests := map[string]struct {
		content string
		mutate  func(t *testing.T, tl *TaskList, path string) string
		want    []string
	}{
		"UpdateStatus + WriteFile": {
			content: commentedFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				if err := tl.UpdateStatus("1", Completed); err != nil {
					t.Fatal(err)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"# project note", "# keep this explanation", "# unknown field comment"},
		},
		"comments separated by blank lines": {
			content: "---\n# detached head\n\nowner: me\n\n# detached foot\n---\n# Demo\n\n- [ ] 1. Task\n",
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				if err := tl.UpdateStatus("1", Completed); err != nil {
					t.Fatal(err)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"# detached head", "owner: me", "# detached foot"},
		},
		"comment-only front matter": {
			content: commentOnlyFrontMatterFile,
			mutate:  updateAndWrite,
			want:    []string{"---\n# Project handoff notes\n---\n"},
		},
		"comment-only front matter, batch": {
			content: commentOnlyFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				resp, err := tl.ExecuteBatch([]Operation{{Type: "update", ID: "1", Status: &completed}}, false)
				if err != nil || !resp.Success {
					t.Fatalf("batch failed: %v %v", err, resp.Errors)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"---\n# Project handoff notes\n---\n"},
		},
		"comment-only front matter, adding metadata": {
			content: commentOnlyFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				if err := tl.AddFrontMatterContent(nil, map[string]string{"owner": "me"}); err != nil {
					t.Fatal(err)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"---\n# Project handoff notes\n", "owner: me"},
		},
		"commented empty references and metadata": {
			content: emptyCommentedFieldsFile,
			mutate:  updateAndWrite,
			want:    []string{"references: [] # filled in later", "# metadata note", "metadata: {}"},
		},
		"commented empty references and metadata, batch dry-run preview": {
			content: emptyCommentedFieldsFile,
			mutate: func(t *testing.T, tl *TaskList, _ string) string {
				t.Helper()
				resp, err := tl.ExecuteBatch([]Operation{{Type: "update", ID: "1", Status: &completed}}, true)
				if err != nil || !resp.Success {
					t.Fatalf("batch failed: %v %v", err, resp.Errors)
				}
				return resp.Preview
			},
			want: []string{"references: [] # filled in later", "# metadata note", "metadata: {}"},
		},
		"phase file, WriteFileWithPhases path": {
			content: commentedFrontMatterPhasesFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				if err := tl.UpdateStatus("1", Completed); err != nil {
					t.Fatal(err)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"# project note", "# keep this explanation"},
		},
		"batch operation (deep copy)": {
			content: commentedFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				resp, err := tl.ExecuteBatch([]Operation{{Type: "update", ID: "1", Status: &completed}}, false)
				if err != nil || !resp.Success {
					t.Fatalf("batch failed: %v %v", err, resp.Errors)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"# project note", "# keep this explanation", "# unknown field comment"},
		},
		"batch dry-run preview": {
			content: commentedFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, _ string) string {
				t.Helper()
				resp, err := tl.ExecuteBatch([]Operation{{Type: "update", ID: "1", Status: &completed}}, true)
				if err != nil || !resp.Success {
					t.Fatalf("batch failed: %v %v", err, resp.Errors)
				}
				return resp.Preview
			},
			want: []string{"# project note", "# keep this explanation", "# unknown field comment"},
		},
		"adding metadata keeps comments on other keys": {
			content: commentedFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				if err := tl.AddFrontMatterContent(nil, map[string]string{"new": "value"}); err != nil {
					t.Fatal(err)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"# project note", "# keep this explanation", "# unknown field comment", "new: value", "z: last"},
		},
		"adding a reference keeps comments on other keys": {
			content: commentedFrontMatterFile,
			mutate: func(t *testing.T, tl *TaskList, path string) string {
				t.Helper()
				if err := tl.AddFrontMatterContent([]string{"docs/extra.md"}, nil); err != nil {
					t.Fatal(err)
				}
				return writeAndRead(t, tl, path)
			},
			want: []string{"# project note", "docs/extra.md", "docs/guide.md", "# unknown field comment"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			path := "tasks.md"
			if err := os.WriteFile(filepath.Join(dir, path), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			tl, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}

			out := tc.mutate(t, tl, path)
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}

			// The result must parse back to the same modelled values.
			reparsed, err := ParseMarkdown([]byte(out))
			if err != nil {
				t.Fatalf("re-parse: %v\n%s", err, out)
			}
			if got, want := reparsed.FrontMatter.References, tl.FrontMatter.References; !slices.Equal(got, want) {
				t.Errorf("references = %q, want %q", got, want)
			}
			for k, v := range tl.FrontMatter.Metadata {
				if reparsed.FrontMatter.Metadata[k] != v {
					t.Errorf("metadata[%q] = %q, want %q", k, reparsed.FrontMatter.Metadata[k], v)
				}
			}
		})
	}
}

func writeAndRead(t *testing.T, tl *TaskList, path string) string {
	t.Helper()
	if err := tl.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

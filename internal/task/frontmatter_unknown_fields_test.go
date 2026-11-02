package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for T-1845: front-matter keys other than `references` and
// `metadata` were dropped when the file was parsed, so the next mutating
// command rewrote the file without them (or without any front matter at all).
// Expected: unsupported top-level keys survive a parse/write round trip.

const unknownFieldsFile = `---
owner: arjen
tags: [important]
---
# Tasks

- [ ] 1. First task
- [ ] 2. Second task
`

const unknownFieldsWithReferencesFile = `---
references:
  - ./docs/design.md
owner: arjen
tags: [important]
---
# Tasks

- [ ] 1. First task
`

const unknownFieldsWithPhasesFile = `---
owner: arjen
---
# Tasks

## Phase One

- [ ] 1. First task
`

func TestWriteFilePreservesUnknownFrontMatterFields(t *testing.T) {
	tests := map[string]struct {
		content string
		mutate  func(t *testing.T, tl *TaskList, path string)
	}{
		"only unknown fields, UpdateStatus + WriteFile": {
			content: unknownFieldsFile,
			mutate: func(t *testing.T, tl *TaskList, path string) {
				t.Helper()
				if err := tl.UpdateStatus("1", Completed); err != nil {
					t.Fatal(err)
				}
				if err := tl.WriteFile(path); err != nil {
					t.Fatal(err)
				}
			},
		},
		"unknown fields alongside references": {
			content: unknownFieldsWithReferencesFile,
			mutate: func(t *testing.T, tl *TaskList, path string) {
				t.Helper()
				if err := tl.UpdateStatus("1", Completed); err != nil {
					t.Fatal(err)
				}
				if err := tl.WriteFile(path); err != nil {
					t.Fatal(err)
				}
			},
		},
		"phase file, WriteFileWithPhases path": {
			content: unknownFieldsWithPhasesFile,
			mutate: func(t *testing.T, tl *TaskList, path string) {
				t.Helper()
				if err := tl.UpdateStatus("1", Completed); err != nil {
					t.Fatal(err)
				}
				// WriteFile delegates to WriteFileWithPhases when the file has phases
				if err := tl.WriteFile(path); err != nil {
					t.Fatal(err)
				}
			},
		},
		"batch operation": {
			content: unknownFieldsFile,
			mutate: func(t *testing.T, tl *TaskList, path string) {
				t.Helper()
				completed := Completed
				resp, err := tl.ExecuteBatch([]Operation{{Type: "update", ID: "1", Status: &completed}}, false)
				if err != nil {
					t.Fatal(err)
				}
				if !resp.Success {
					t.Fatalf("batch failed: %v", resp.Errors)
				}
				if err := tl.WriteFile(path); err != nil {
					t.Fatal(err)
				}
			},
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
			tc.mutate(t, tl, path)

			got, err := os.ReadFile(filepath.Join(dir, path))
			if err != nil {
				t.Fatal(err)
			}
			out := string(got)
			if !strings.HasPrefix(out, "---\n") {
				t.Fatalf("front matter was removed; file now:\n%s", out)
			}
			if !strings.Contains(out, "owner: arjen") {
				t.Errorf("owner field was dropped; file now:\n%s", out)
			}
			if strings.Contains(tc.content, "tags:") && !strings.Contains(out, "important") {
				t.Errorf("tags field was dropped; file now:\n%s", out)
			}
			if strings.Contains(tc.content, "references:") && !strings.Contains(out, "./docs/design.md") {
				t.Errorf("references were dropped; file now:\n%s", out)
			}

			// The rewritten file must still parse and keep the fields on a second round trip.
			reparsed, err := ParseFile(path)
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}
			if reparsed.FrontMatter == nil || reparsed.FrontMatter.Extra["owner"] != "arjen" {
				t.Errorf("owner not preserved after re-parse: %+v", reparsed.FrontMatter)
			}
		})
	}
}

func TestSerializeWithFrontMatterOnlyUnknownFields(t *testing.T) {
	fm, rest, err := ParseFrontMatter(unknownFieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	out := SerializeWithFrontMatter(fm, rest)
	if !strings.Contains(out, "owner: arjen") || !strings.Contains(out, "- important") {
		t.Errorf("unknown fields not serialized:\n%s", out)
	}
}

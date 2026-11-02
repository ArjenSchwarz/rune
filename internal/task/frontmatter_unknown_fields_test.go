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
			if again := SerializeWithFrontMatter(reparsed.FrontMatter, ""); !strings.Contains(again, "owner: arjen") {
				t.Errorf("owner not preserved after re-parse:\n%s", again)
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
	if !strings.Contains(out, "owner: arjen") || !strings.Contains(out, "important") {
		t.Errorf("unknown fields not serialized:\n%s", out)
	}
}

// Review follow-up on T-1845: unknown values were decoded into plain Go
// values, so YAML tags were dropped on write (`owner: !secret value` became
// `owner: value`), changing what downstream tools read. Unknown values must be
// written back with their tags, and the rewritten front matter must still
// parse (an alias must not be written before its anchor).
func TestWriteFilePreservesTaggedAndAliasedFrontMatterValues(t *testing.T) {
	tests := map[string]struct {
		frontMatter string
		want        []string
	}{
		"custom scalar tag": {
			frontMatter: "owner: !secret placeholder-value\n",
			want:        []string{"owner: !secret placeholder-value"},
		},
		"custom tag nested in mapping and sequence": {
			frontMatter: "settings:\n  key: !secret placeholder-value\nitems:\n  - !custom item-one\n",
			want:        []string{"key: !secret placeholder-value", "- !custom item-one"},
		},
		"explicit core tag": {
			frontMatter: "version: !!str 1.0\n",
			want:        []string{"version: !!str 1.0"},
		},
		"tagged value alongside references": {
			frontMatter: "references:\n  - ./docs/design.md\nowner: !secret placeholder-value\n",
			want:        []string{"./docs/design.md", "owner: !secret placeholder-value"},
		},
		"alias whose key sorts before its anchor": {
			frontMatter: "zz_base: &shared {level: 1}\naa_copy: *shared\n",
			want:        []string{"zz_base:", "aa_copy:"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			path := "tasks.md"
			content := "---\n" + tc.frontMatter + "---\n# Tasks\n\n- [ ] 1. First task\n"
			if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			tl, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			if err := tl.UpdateStatus("1", Completed); err != nil {
				t.Fatal(err)
			}
			if err := tl.WriteFile(path); err != nil {
				t.Fatal(err)
			}

			got, err := os.ReadFile(filepath.Join(dir, path))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(got), want) {
					t.Errorf("rewritten file is missing %q:\n%s", want, got)
				}
			}
			reparsed, err := ParseFile(path)
			if err != nil {
				t.Fatalf("rewritten file no longer parses: %v\n%s", err, got)
			}
			// A second write must be stable: the tags survive repeated round trips.
			if err := reparsed.WriteFile(path); err != nil {
				t.Fatal(err)
			}
			again, err := os.ReadFile(filepath.Join(dir, path))
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != string(got) {
				t.Errorf("second write changed the file:\nfirst:\n%s\nsecond:\n%s", got, again)
			}
		})
	}
}

// Front matter rune cannot write back faithfully must be rejected at parse
// time, before any command rewrites the file. Here an unknown key aliases an
// anchor on a modelled field; rune re-encodes modelled fields without their
// anchors, so the alias would dangle.
func TestParseFrontMatterRejectsUnpreservableAlias(t *testing.T) {
	content := "---\nreferences: &refs\n  - ./docs/design.md\nmirror: *refs\n---\n# Tasks\n\n- [ ] 1. First task\n"
	if _, err := ParseMarkdown([]byte(content)); err == nil {
		t.Fatal("expected an error for front matter that cannot be written back, got none")
	}
}

package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/config"
)

// writeContextTaskFile writes content to tasks.md in a fresh temp dir, changes into
// that dir, and returns the file name.
func writeContextTaskFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return "tasks.md"
}

// useFormat sets the global output format for the test and restores it after.
func useFormat(t *testing.T, f string) {
	t.Helper()
	old := format
	format = f
	t.Cleanup(func() { format = old })
}

// runFindWith runs the find command with the given flag values and returns
// stdout. Flag-backed globals are reset before and after, since other tests
// leave them set.
func runFindWith(t *testing.T, file string, flags map[string]string) string {
	t.Helper()
	reset := func() {
		includeParent, statusFilter, maxDepth = false, "", 0
		searchDetails, searchRefs, caseSensitive = false, false, false
		for _, name := range []string{"pattern", "include-parent", "status", "max-depth", "parent"} {
			if f := findCmd.Flags().Lookup(name); f != nil {
				f.Value.Set(f.DefValue)
				f.Changed = false
			}
		}
	}
	reset()
	t.Cleanup(reset)
	for name, value := range flags {
		if err := findCmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	return captureStdout(t, func() {
		if err := runFind(findCmd, []string{file}); err != nil {
			t.Errorf("find failed: %v", err)
		}
	})
}

func resetListFlags(t *testing.T) {
	t.Helper()
	listFilter, showAll, listStreamFilter, listOwnerFilter = "", false, 0, ""
	t.Cleanup(func() { listFilter, showAll, listStreamFilter, listOwnerFilter = "", false, 0, "" })
}

// T-2070: --include-parent context must only be returned for matches that
// survive --max-depth / --status filtering.
func TestFind_IncludeParentContextOnlyForSurvivingMatches(t *testing.T) {
	content := "# T\n\n" +
		"- [ ] 1. Root\n" +
		"  - [ ] 1.1. Context\n" +
		"    - [x] 1.1.1. needle\n"

	tests := map[string]map[string]string{
		"max-depth removes the match": {"pattern": "needle", "include-parent": "true", "max-depth": "2"},
		"status removes the match":    {"pattern": "needle", "include-parent": "true", "status": "pending"},
	}

	for name, flags := range tests {
		t.Run(name, func(t *testing.T) {
			file := writeContextTaskFile(t, content)
			useFormat(t, formatJSON)

			out := runFindWith(t, file, flags)

			var resp struct {
				Count int `json:"count"`
			}
			if err := json.Unmarshal([]byte(out), &resp); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, out)
			}
			if resp.Count != 0 || strings.Contains(out, "Context") {
				t.Errorf("expected no results once the match is filtered out, got:\n%s", out)
			}
		})
	}

	t.Run("context kept for surviving match", func(t *testing.T) {
		file := writeContextTaskFile(t, content)
		useFormat(t, formatJSON)

		out := runFindWith(t, file, map[string]string{"pattern": "needle", "include-parent": "true", "status": "completed"})

		if !strings.Contains(out, `"Context"`) || !strings.Contains(out, `"needle"`) {
			t.Errorf("expected match and its parent context, got:\n%s", out)
		}
	})

	// The parent matches the pattern too, but the status filter removes it.
	// It must come back as context for the surviving child.
	t.Run("filtered-out matching parent restored as context", func(t *testing.T) {
		file := writeContextTaskFile(t, "# T\n\n- [x] 1. Feature parent\n  - [ ] 1.1. Feature child\n")
		useFormat(t, formatJSON)

		out := runFindWith(t, file, map[string]string{"pattern": "Feature", "include-parent": "true", "status": "pending"})

		var resp struct {
			Count int `json:"count"`
			Tasks []struct {
				ID       string `json:"ID"`
				ParentID string `json:"ParentID"`
			} `json:"Tasks"`
		}
		if err := json.Unmarshal([]byte(out), &resp); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if resp.Count != 2 || len(resp.Tasks) != 2 {
			t.Fatalf("want parent context and child, got:\n%s", out)
		}
		if resp.Tasks[0].ID != "1" || resp.Tasks[1].ID != "1.1" || resp.Tasks[1].ParentID != "1" {
			t.Errorf("want [1, 1.1 (parent 1)], got:\n%s", out)
		}
	})
}

// T-2100: an empty list result must keep the file-level title and front matter.
func TestList_EmptyResultKeepsDocumentContext(t *testing.T) {
	content := "---\nreferences:\n  - spec.md\nmetadata:\n  owner: team\n---\n# Empty\n"

	t.Run("json", func(t *testing.T) {
		file := writeContextTaskFile(t, content)
		useFormat(t, formatJSON)
		resetListFlags(t)
		showAll = true

		out := captureStdout(t, func() {
			if err := runList(listCmd, []string{file}); err != nil {
				t.Errorf("list failed: %v", err)
			}
		})

		var resp map[string]any
		if err := json.Unmarshal([]byte(out), &resp); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if resp["count"] != float64(0) || resp["success"] != true {
			t.Errorf("empty envelope changed: %s", out)
		}
		if resp["Title"] != "Empty" {
			t.Errorf("Title missing from empty JSON: %s", out)
		}
		if !strings.Contains(out, "spec.md") || !strings.Contains(out, "team") {
			t.Errorf("front matter missing from empty JSON: %s", out)
		}
	})

	t.Run("markdown", func(t *testing.T) {
		file := writeContextTaskFile(t, content)
		useFormat(t, formatMarkdown)
		resetListFlags(t)
		showAll = true

		out := captureStdout(t, func() {
			if err := runList(listCmd, []string{file}); err != nil {
				t.Errorf("list failed: %v", err)
			}
		})

		for _, want := range []string{"## Document References", "- spec.md", "# Empty", "No tasks found"} {
			if !strings.Contains(out, want) {
				t.Errorf("markdown missing %q:\n%s", want, out)
			}
		}
	})
}

// T-2182: markdown list output includes front-matter references without --all.
func TestList_MarkdownIncludesDocumentReferencesWithoutAll(t *testing.T) {
	file := writeContextTaskFile(t, "---\nreferences:\n  - docs/spec.md\n---\n# Tasks\n\n- [ ] 1. First\n")
	useFormat(t, formatMarkdown)
	resetListFlags(t)

	out := captureStdout(t, func() {
		if err := runList(listCmd, []string{file}); err != nil {
			t.Errorf("list failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Document References") || !strings.Contains(out, "- docs/spec.md") {
		t.Errorf("document references missing without --all:\n%s", out)
	}
}

// T-2127: a failed git discovery diagnostic must not be written to stdout.
func TestResolveFilename_DiscoveryFailureVerboseGoesToStderr(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := exec.Command("git", "-C", dir, "init", "-q", "-b", "main").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	t.Chdir(dir)
	config.ResetConfigCache()
	t.Cleanup(config.ResetConfigCache)
	useFormat(t, formatJSON)
	oldVerbose := verbose
	verbose = true
	t.Cleanup(func() { verbose = oldVerbose })

	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			if _, err := resolveFilename(nil); err == nil {
				t.Error("expected discovery to fail")
			}
		})
	})

	if stdout != "" {
		t.Errorf("stdout should be empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "Git discovery failed") {
		t.Errorf("stderr should carry the discovery diagnostic, got %q", stderr)
	}
}

const metadataTaskFile = "# Demo\n\n" +
	"- [x] 1. Finished blocker <!-- id:abc0001 -->\n\n" +
	"- [ ] 2. Stream task <!-- id:abc0002 -->\n" +
	"  - Blocked-by: abc0001 (Finished blocker)\n" +
	"  - Stream: 2\n" +
	"  - Owner: bot\n" +
	"  - Requirements: [3.6](requirements.md#3.6)\n"

// T-2242: human find output shows stream, owner, dependencies and requirements.
func TestFind_HumanOutputShowsTaskMetadata(t *testing.T) {
	tests := map[string]struct {
		format string
		want   []string
	}{
		"table":    {format: formatTable, want: []string{"STREAM", "BLOCKEDBY", "OWNER", "REQUIREMENTS", "bot", "3.6"}},
		"markdown": {format: formatMarkdown, want: []string{"Stream: 2", "Blocked-by: 1", "Owner: bot", "Requirements: [3.6](requirements.md#3.6)"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file := writeContextTaskFile(t, metadataTaskFile)
			useFormat(t, tc.format)

			out := runFindWith(t, file, map[string]string{"pattern": "Stream task"})

			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
		})
	}
}

// T-2274: human next --stream output shows stream and dependency IDs.
func TestNextStream_HumanOutputShowsStreamAndBlockedBy(t *testing.T) {
	// Without an owner so the task is ready for `next --stream`.
	content := strings.Replace(metadataTaskFile, "  - Owner: bot\n", "", 1)

	tests := map[string]struct {
		format string
		want   []string
	}{
		"table":    {format: formatTable, want: []string{"STREAM", "BLOCKEDBY", "| 2      | 1         |"}},
		"markdown": {format: formatMarkdown, want: []string{"Stream: 2", "Blocked-by: 1"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file := writeContextTaskFile(t, content)
			useFormat(t, tc.format)
			phaseFlag, streamFlag, claimFlag, oneFlag = false, 2, "", false
			t.Cleanup(func() { streamFlag = 0 })

			out := captureStdout(t, func() {
				if err := runNextWithStream(file); err != nil {
					t.Errorf("next --stream failed: %v", err)
				}
			})

			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
		})
	}
}

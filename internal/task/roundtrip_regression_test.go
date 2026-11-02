package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for T-2062: a valid task file without an H1 title was
// rendered with a bare "# " line, which the parser rejects, so any write made
// the file unparsable. Expected: no H1 is written when the title is empty.
func TestTitlelessFileRoundTrip(t *testing.T) {
	tests := map[string]struct {
		content string
	}{
		"unphased": {
			content: "- [ ] 1. Untitled task\n",
		},
		"phased": {
			content: "## Phase One\n\n- [ ] 1. Untitled task\n\n## Phase Two\n\n- [ ] 2. Second task\n",
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
				t.Fatalf("initial parse: %v", err)
			}
			if tl.Title != "" {
				t.Fatalf("expected empty title, got %q", tl.Title)
			}
			if err := tl.UpdateStatus("1", InProgress); err != nil {
				t.Fatal(err)
			}
			// WriteFile renders through RenderMarkdownWithPhases when the file has phases.
			if err := tl.WriteFile(path); err != nil {
				t.Fatal(err)
			}

			got, err := os.ReadFile(filepath.Join(dir, path))
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(string(got), "#") && !strings.HasPrefix(string(got), "## ") {
				t.Errorf("rendered file starts with an H1 for an empty title:\n%s", got)
			}

			reparsed, phases, err := ParseFileWithPhases(path)
			if err != nil {
				t.Fatalf("re-parse after write failed: %v\nfile:\n%s", err, got)
			}
			if reparsed.Tasks[0].Status != InProgress {
				t.Errorf("task 1 status = %v, want InProgress", reparsed.Tasks[0].Status)
			}
			if strings.Contains(tc.content, "## ") && len(phases) != 2 {
				t.Errorf("got %d phase markers, want 2", len(phases))
			}
		})
	}
}

func TestRenderEmptyTitleOmitsHeading(t *testing.T) {
	tl := &TaskList{Tasks: []Task{{ID: "1", Title: "Task", Status: Pending}}}
	renderers := map[string]func() []byte{
		"RenderMarkdown":           func() []byte { return RenderMarkdown(tl) },
		"RenderMarkdownWithPhases": func() []byte { return RenderMarkdownWithPhases(tl, nil, nil) },
	}
	for name, render := range renderers {
		t.Run(name, func(t *testing.T) {
			got := string(render())
			if !strings.HasPrefix(got, "- [ ] 1. Task") {
				t.Errorf("expected output to start with the task line, got:\n%q", got)
			}
			if _, err := ParseMarkdown([]byte(got)); err != nil {
				t.Errorf("rendered output does not parse: %v", err)
			}
		})
	}
}

// Regression tests for T-1973: requirement links pointing at different files
// were collapsed to the first file's path, so the next write silently
// retargeted the later links. A task list has a single requirements file
// (task-requirements-linking Decision 3), so mixed paths must be rejected.
func TestParseRejectsMixedRequirementsFiles(t *testing.T) {
	tests := map[string]struct {
		content string
		wantErr bool
	}{
		"different files across tasks": {
			content: `# Mixed Requirements Repro

- [ ] 1. First task
  - Requirements: [1.1](requirements-a.md#1.1)
- [ ] 2. Second task
  - Requirements: [2.1](requirements-b.md#2.1)
`,
			wantErr: true,
		},
		"different files within one line": {
			content: `# Mixed

- [ ] 1. First task
  - Requirements: [1.1](requirements-a.md#1.1), [2.1](requirements-b.md#2.1)
`,
			wantErr: true,
		},
		"different files in subtask": {
			content: `# Mixed

- [ ] 1. First task
  - Requirements: [1.1](requirements-a.md#1.1)
  - [ ] 1.1. Child
    - Requirements: [2.1](requirements-b.md#2.1)
`,
			wantErr: true,
		},
		"same file across tasks": {
			content: `# Same

- [ ] 1. First task
  - Requirements: [1.1](specs/req.md#1.1)
- [ ] 2. Second task
  - Requirements: [2.1](specs/req.md#2.1), [2.2](specs/req.md#2.2)
`,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error for mixed requirements files, got none; re-rendered:\n%s", RenderMarkdown(tl))
				}
				if !strings.Contains(err.Error(), "requirements-b.md") || !strings.Contains(err.Error(), "requirements-a.md") {
					t.Errorf("error should name both files, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := string(RenderMarkdown(tl))
			for _, link := range []string{"[1.1](specs/req.md#1.1)", "[2.1](specs/req.md#2.1), [2.2](specs/req.md#2.2)"} {
				if !strings.Contains(got, link) {
					t.Errorf("round trip lost link %q:\n%s", link, got)
				}
			}
		})
	}
}

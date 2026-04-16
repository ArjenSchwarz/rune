package task

import (
	"strings"
	"testing"
)

// Regression tests for T-674: Parser silently ignores invalid indented
// non-task lines at task level.
//
// parseTasksAtLevel used to silently skip non-task lines whose indentation
// was deeper than the current level (the default branch did `continue`).
// The parser must return an error for such lines instead.

func TestParseRejectsIndentedNonTaskLines(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content     string
		errContains string
	}{
		// Exercises the new default branch in parseTasksAtLevel:
		// 2-space indent (expectedIndent+2) non-task line at root level.
		"indented_plain_text_before_task": {
			content: `# Tasks
  not-a-task line
- [ ] 1. Real task`,
			errContains: "unexpected content",
		},
		"indented_plain_text_only": {
			content: `# Tasks
  just some indented text`,
			errContains: "unexpected content",
		},
		// Pre-existing validation: 4-space indent at root is caught by the
		// indent > expectedIndent && indent != expectedIndent+2 guard, not
		// by the default branch fixed in this change.
		"double_indented_plain_text_at_root": {
			content: `# Tasks
    deeply indented line
- [ ] 1. Real task`,
			errContains: "unexpected indentation",
		},
		// Exercises parseDetailsAndChildren (different code path), not the
		// default branch in parseTasksAtLevel. Kept as a regression guard.
		"indented_non_task_at_child_level": {
			content: `# Tasks
- [ ] 1. Parent task
    not-a-task at grandchild level
  - [ ] 1.1. Child task`,
			errContains: "unexpected",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseMarkdown([]byte(tc.content))
			if err == nil {
				t.Fatal("expected error but got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseAllowsValidIndentedContent(t *testing.T) {
	t.Parallel()

	// Ensure we don't break valid indented content (detail lines, subtasks).
	tests := map[string]struct {
		content      string
		wantTasks    int
		wantChildren int // expected children of first task (0 = don't check)
	}{
		"subtasks_are_valid": {
			content: `# Tasks
- [ ] 1. Parent
  - [ ] 1.1. Child`,
			wantTasks:    1,
			wantChildren: 1,
		},
		"detail_lines_are_valid": {
			content: `# Tasks
- [ ] 1. Parent
  - Some detail`,
			wantTasks: 1,
		},
		"phase_headers_at_root_are_valid": {
			content: `# Tasks
- [ ] 1. First task
## Phase 2
- [ ] 2. Second task`,
			wantTasks: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tl.Tasks) != tc.wantTasks {
				t.Errorf("got %d tasks, want %d", len(tl.Tasks), tc.wantTasks)
			}
			if tc.wantChildren > 0 && len(tl.Tasks) > 0 {
				if got := len(tl.Tasks[0].Children); got != tc.wantChildren {
					t.Errorf("got %d children, want %d", got, tc.wantChildren)
				}
			}
		})
	}
}

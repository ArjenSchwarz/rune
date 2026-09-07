package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for T-2041: parser drops non-bullet task body lines on
// mutation.
//
// parseDetailsAndChildren's indent == expectedIndent branch called
// parseDetailLine and silently dropped the line whenever it returned "" —
// i.e. whenever the line wasn't a valid Markdown bullet — instead of
// erroring. That let malformed indented content parse successfully while
// silently discarding it: `list --format json` would omit the content, and
// a later mutation (e.g. `add`) would rewrite the file without it,
// permanently losing user data. The parser must return an error for such
// lines instead, matching the pattern already fixed for parseTasksAtLevel's
// default branch in T-674 (see parse_invalid_indent_test.go).

func TestParseRejectsNonBulletDetailLines(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content     string
		errContains string
	}{
		// Exact repro from T-2041: a non-bullet line at the exact expected
		// detail indentation directly under a task.
		"non_bullet_line_at_detail_indent": {
			content: `# Tasks

- [ ] 1. Keep me
  this malformed body line is not a Markdown bullet
- [ ] 2. Keep me too`,
			// The hint is asserted here because this is the first error
			// users of previously-tolerated files will hit after upgrading.
			errContains: "unexpected content at this indentation level (missing '- ' bullet?)",
		},
		"non_bullet_line_after_valid_detail": {
			content: `# Tasks

- [ ] 1. Keep me
  - Valid detail
  another malformed line`,
			errContains: "unexpected content",
		},
		// A bullet marker with no content is not a valid detail line either.
		"empty_bullet_at_detail_indent": {
			content: `# Tasks

- [ ] 1. Keep me
  -`,
			errContains: "unexpected content",
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

func TestParseAllowsValidDetailLines(t *testing.T) {
	t.Parallel()

	// Ensure the fix doesn't break legitimate detail lines.
	tl, err := ParseMarkdown([]byte(`# Tasks

- [ ] 1. Keep me
  - Valid detail line
- [ ] 2. Keep me too`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tl.Tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(tl.Tasks))
	}
	if len(tl.Tasks[0].Details) != 1 || tl.Tasks[0].Details[0] != "Valid detail line" {
		t.Errorf("got details %v, want [\"Valid detail line\"]", tl.Tasks[0].Details)
	}
}

// TestMutationDoesNotDropNonBulletLines guards the user-facing symptom from
// T-2041 rather than only its root cause: `rune add` (and every other command
// that rewrites the file) used to parse successfully, drop the malformed line
// from the in-memory TaskList, and then write the file back without it —
// destroying the content with a zero exit code. The parse error must abort the
// read-modify-write cycle before anything is written, leaving the file intact.
func TestMutationDoesNotDropNonBulletLines(t *testing.T) {
	t.Parallel()

	const content = `# Tasks

- [ ] 1. Keep me
  this malformed body line is not a Markdown bullet
- [ ] 2. Keep me too
`

	path := filepath.Join(t.TempDir(), "tasks.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	// The read step of the read-modify-write cycle must fail...
	tl, err := ParseFile(path)
	if err == nil {
		t.Fatal("ParseFile() succeeded on a non-bullet body line; the mutation " +
			"would proceed and drop the line")
	}
	if tl != nil {
		t.Errorf("ParseFile() returned a task list alongside the error: %+v", tl)
	}
	if !strings.Contains(err.Error(), "unexpected content") {
		t.Errorf("error %q does not mention unexpected content", err.Error())
	}

	// ...so the file is never rewritten and the line survives.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file back: %v", err)
	}
	if string(after) != content {
		t.Errorf("file was modified.\ngot:\n%s\nwant:\n%s", after, content)
	}
}

// TestAddTaskToPhaseRejectsNonBulletLines covers the same guarantee for the
// filepath-taking mutation helpers, which do their own parsing internally
// rather than going through ParseFile.
//
// The error message is asserted rather than just its presence: writing to a
// path outside the working directory is separately rejected as path traversal,
// so a bare non-nil check would pass even with the parser bug reintroduced.
func TestAddTaskToPhaseRejectsNonBulletLines(t *testing.T) {
	t.Parallel()

	const content = `# Tasks

## Phase One

- [ ] 1. Keep me
  this malformed body line is not a Markdown bullet
`

	path := filepath.Join(t.TempDir(), "tasks.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	_, err := AddTaskToPhase(path, "", "Added", "Phase One")
	if err == nil {
		t.Fatal("AddTaskToPhase() succeeded on a file with a non-bullet body line")
	}
	if !strings.Contains(err.Error(), "unexpected content") {
		t.Errorf("AddTaskToPhase() error = %q, want it to report the parse failure", err)
	}
}

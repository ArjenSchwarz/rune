package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const linkedTaskFile = `# Tasks

- [ ] 1. Linked
  - Requirements: [1.1](specs/custom-requirements.md#1.1)
  - [ ] 1.1. Linked child
    - Requirements: [2.1](specs/custom-requirements.md#2.1)
`

// resetOutputContextFlags resets the command flag variables these tests rely
// on, before and after the test, so values set by other tests do not leak in
// and values set here do not leak out.
func resetOutputContextFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		addTitle, addParent, addPosition, addPhase = "", "", "", ""
		addRequirements, addRequirementsFile, addBlockedBy, addOwner = "", "", "", ""
		addStream = 0
		resetUpdateFlags()
		createTitle = ""
		addFMReferences, addFMMetadata = []string{}, []string{}
		listFilter, listOwnerFilter, listStreamFilter, showAll = "", "", 0, false
		findPattern, statusFilter, parentIDFilter, maxDepth = "", "", "", 0
		searchDetails, searchRefs, caseSensitive, includeParent = false, false, false, false
		format, versionOutputFormat = formatTable, formatTable
	}
	reset()
	t.Cleanup(reset)
}

// decodeJSON unmarshals out into a generic map, failing the test when out is
// not valid JSON.
func decodeJSON(t *testing.T, out string) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	return got
}

// T-1129: list JSON for a phased file includes requirements_file, as it does
// for a file without phases.
func TestListJSONPhasedIncludesRequirementsFile(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Tasks

## Phase A

- [ ] 1. Linked
  - Requirements: [1.1](custom-reqs.md#1.1)
`)

	got := decodeJSON(t, mustRun(t, dir, "list", "tasks.md", "--format", "json"))
	if got["requirements_file"] != "custom-reqs.md" {
		t.Errorf("requirements_file = %v, want custom-reqs.md", got["requirements_file"])
	}
}

// T-1995: find JSON keeps the source file's requirements_file.
func TestFindJSONIncludesRequirementsFile(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, linkedTaskFile)

	got := decodeJSON(t, mustRun(t, dir, "find", "tasks.md", "--pattern", "Linked", "--format", "json"))
	if got["requirements_file"] != "specs/custom-requirements.md" {
		t.Errorf("requirements_file = %v, want specs/custom-requirements.md", got["requirements_file"])
	}
}

// T-1994: next JSON includes task requirements and the requirements file, in
// the plain, --stream and --phase variants.
func TestNextJSONIncludesRequirements(t *testing.T) {
	tests := map[string][]string{
		"plain":  {"next", "tasks.md", "--format", "json"},
		"stream": {"next", "tasks.md", "--stream", "1", "--format", "json"},
		"phase":  {"next", "tasks.md", "--phase", "--format", "json"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			resetOutputContextFlags(t)
			dir := t.TempDir()
			writeTaskFile(t, dir, linkedTaskFile)

			out := mustRun(t, dir, args...)
			got := decodeJSON(t, out)
			if got["requirements_file"] != "specs/custom-requirements.md" {
				t.Errorf("requirements_file = %v, want specs/custom-requirements.md\n%s", got["requirements_file"], out)
			}
			for _, want := range []string{`"1.1"`, `"2.1"`} {
				if !strings.Contains(out, `"requirements": [`+"\n") || !strings.Contains(out, want) {
					t.Errorf("output is missing requirement %s:\n%s", want, out)
				}
			}
		})
	}
}

// T-1994: next markdown includes requirement links for the task and its
// subtasks, in the plain and --phase variants.
func TestNextMarkdownIncludesRequirements(t *testing.T) {
	tests := map[string][]string{
		"plain": {"next", "tasks.md", "--format", "markdown"},
		"phase": {"next", "tasks.md", "--phase", "--format", "markdown"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			resetOutputContextFlags(t)
			dir := t.TempDir()
			writeTaskFile(t, dir, linkedTaskFile)

			out := mustRun(t, dir, args...)
			for _, want := range []string{
				"Requirements: [1.1](specs/custom-requirements.md#1.1)",
				"Requirements: [2.1](specs/custom-requirements.md#2.1)",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("markdown output is missing %q:\n%s", want, out)
				}
			}
		})
	}
}

// T-2022: next table output includes the details and task-level references of
// subtasks, not only of the root task.
func TestNextTableIncludesSubtaskMetadata(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Next Child Metadata

- [ ] 1. Parent task
  - Parent note
  - [ ] 1.1. Child task
    - Child implementation note
    - References: child-spec.md
    - [ ] 1.1.1. Grandchild task
      - Grandchild note
      - References: grandchild-spec.md
`)

	out := mustRun(t, dir, "next", "tasks.md", "--format", "table")
	for _, want := range []string{
		"Parent note",
		"Child implementation note", "child-spec.md",
		"Grandchild note", "grandchild-spec.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table output is missing %q:\n%s", want, out)
		}
	}
}

// T-1409: streams and version honour the global --format json.
func TestStreamsAndVersionHonourGlobalJSONFormat(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, "# Demo\n\n- [ ] 1. Ready task\n")

	streams := decodeJSON(t, mustRun(t, dir, "streams", "tasks.md", "--format", "json"))
	if _, ok := streams["streams"]; !ok {
		t.Errorf("streams JSON has no streams field: %v", streams)
	}

	// Other tests point versionCmd at their own buffers; write to the root output
	versionCmd.SetOut(nil)
	version := decodeJSON(t, mustRun(t, dir, "version", "--format", "json"))
	if _, ok := version["version"]; !ok {
		t.Errorf("version JSON has no version field: %v", version)
	}
}

// T-1159: dry-run paths of mutating commands honour --format json and do not
// change files.
func TestDryRunHonoursJSONFormat(t *testing.T) {
	const existing = "# Tasks\n\n- [ ] 1. Existing\n"
	tests := map[string]struct {
		args        []string
		wantMessage string
	}{
		"add":             {[]string{"add", "tasks.md", "--title", "New task"}, "Would add task 2"},
		"update":          {[]string{"update", "tasks.md", "1", "--title", "Changed"}, "Would update task 1"},
		"remove":          {[]string{"remove", "tasks.md", "1"}, "Would remove task 1"},
		"create":          {[]string{"create", "new.md", "--title", "Created"}, "Would create new.md"},
		"add-phase":       {[]string{"add-phase", "tasks.md", "Phase X"}, "Would add phase 'Phase X'"},
		"add-frontmatter": {[]string{"add-frontmatter", "tasks.md", "--reference", "doc.md"}, "Would update tasks.md"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resetOutputContextFlags(t)
			dir := t.TempDir()
			writeTaskFile(t, dir, existing)

			args := append(tc.args, "--dry-run", "--format", "json")
			got := decodeJSON(t, mustRun(t, dir, args...))
			if got["dry_run"] != true || got["message"] != tc.wantMessage {
				t.Errorf("got dry_run=%v message=%v, want true and %q", got["dry_run"], got["message"], tc.wantMessage)
			}

			content, err := os.ReadFile(filepath.Join(dir, "tasks.md"))
			if err != nil || string(content) != existing {
				t.Errorf("dry run changed tasks.md: %q (err %v)", content, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "new.md")); err == nil {
				t.Error("dry run created new.md")
			}
		})
	}
}

// T-1159: dry-run paths honour --format markdown.
func TestDryRunHonoursMarkdownFormat(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, "# Tasks\n\n- [ ] 1. Existing\n")

	out := mustRun(t, dir, "remove", "tasks.md", "1", "--dry-run", "--format", "markdown")
	if !strings.HasPrefix(out, "**Dry run:** Would remove 1 - Existing") {
		t.Errorf("unexpected markdown dry-run output: %q", out)
	}
}

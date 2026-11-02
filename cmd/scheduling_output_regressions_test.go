package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runRuneCLIError runs rune with args inside dir and returns the command
// error, for commands that are expected to fail.
func runRuneCLIError(t *testing.T, dir string, args ...string) error {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	resetFlags := func() {
		streamFlag, claimFlag, phaseFlag, oneFlag = 0, "", false, false
		dryRun = false
	}
	resetFlags()
	t.Cleanup(resetFlags)

	rootCmd.SetArgs(args)
	runErr := rootCmd.Execute()
	rootCmd.SetArgs([]string{})
	return runErr
}

// findJSONTask returns the task with the given ID from list/find JSON output.
func findJSONTask(t *testing.T, out, id string) map[string]any {
	t.Helper()
	var got struct {
		Tasks []map[string]any `json:"Tasks"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, out)
	}
	for _, task := range got.Tasks {
		if task["ID"] == id {
			return task
		}
	}
	t.Fatalf("task %s not in output:\n%s", id, out)
	return nil
}

// T-1206: filtered list and find JSON translate Blocked-by references to
// tasks that are not part of the result.
func TestFilteredJSONKeepsExternalBlockedBy(t *testing.T) {
	const content = `# Tasks

- [x] 1. Done blocker <!-- id:aaa0001 -->
- [ ] 2. Pending blocked <!-- id:bbb0002 -->
  - Blocked-by: aaa0001 (Done blocker)
`
	tests := map[string][]string{
		"list filter":      {"list", "tasks.md", "--filter", "pending", "--format", "json"},
		"find":             {"find", "tasks.md", "--pattern", "Pending", "--format", "json"},
		"find status only": {"find", "tasks.md", "--pattern", "blocked", "--status", "pending", "--format", "json"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			resetOutputContextFlags(t)
			dir := t.TempDir()
			writeTaskFile(t, dir, content)

			out := mustRun(t, dir, args...)
			got := findJSONTask(t, out, "2")
			blockedBy, _ := got["blockedBy"].([]any)
			if len(blockedBy) != 1 || blockedBy[0] != "1" {
				t.Errorf("task 2 blockedBy = %v, want [1]\n%s", got["blockedBy"], out)
			}
		})
	}
}

// T-1626: streams keeps a stream whose only pending task is claimed but not
// started, reports it as claimed, and does not mark it available.
func TestStreamsReportsClaimedPendingTasks(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, `# Streams Owned Pending

- [ ] 1. Claimed but not started <!-- id:own0001 -->
  - Stream: 2
  - Owner: agent-1
`)

	out := mustRun(t, dir, "streams", "tasks.md", "--json")
	var got struct {
		Streams []struct {
			ID      int      `json:"id"`
			Claimed []string `json:"claimed"`
		} `json:"streams"`
		Available []int `json:"available"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse streams JSON: %v\n%s", err, out)
	}
	if len(got.Streams) != 1 || got.Streams[0].ID != 2 || strings.Join(got.Streams[0].Claimed, ",") != "1" {
		t.Errorf("want stream 2 with claimed [1], got:\n%s", out)
	}
	if len(got.Available) != 0 {
		t.Errorf("claimed-only stream must not be available, got:\n%s", out)
	}

	table := mustRun(t, dir, "streams", "tasks.md")
	if strings.Contains(table, "No streams with pending tasks") || !strings.Contains(table, "CLAIMED") {
		t.Errorf("table should list stream 2 with a Claimed column, got:\n%s", table)
	}
}

const unresolvedBlockerFile = `# Tasks

- [ ] 1. Task <!-- id:aaaaaaa -->
  - Blocked-by: bbbbbbb (Missing task)
- [ ] 2. Free <!-- id:ccccccc -->
`

// T-1809: list output reports blocker IDs that match no task, instead of
// dropping them while the task stays blocked.
func TestListReportsUnresolvedBlockers(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, unresolvedBlockerFile)

	out := mustRun(t, dir, "list", "tasks.md", "--format", "json")
	got := findJSONTask(t, out, "1")
	unresolved, _ := got["unresolvedBlockedBy"].([]any)
	if len(unresolved) != 1 || unresolved[0] != "bbbbbbb" {
		t.Errorf("task 1 unresolvedBlockedBy = %v, want [bbbbbbb]\n%s", got["unresolvedBlockedBy"], out)
	}

	table := mustRun(t, dir, "list", "tasks.md")
	if !strings.Contains(table, "bbbbbbb (unresolved)") {
		t.Errorf("table output is missing the unresolved blocker:\n%s", table)
	}
}

// T-1809: an unresolved blocker still blocks scheduling (conservative T-337
// behaviour is kept).
func TestUnresolvedBlockerStillBlocks(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, unresolvedBlockerFile)

	out := mustRun(t, dir, "next", "tasks.md", "--format", "json")
	if !strings.Contains(out, `"title": "Free"`) || strings.Contains(out, `"title": "Task"`) {
		t.Errorf("next should skip the task with an unresolved blocker, got:\n%s", out)
	}
}

const phaseMetadataFile = `# Demo

## Work

- [ ] 1. Blocker <!-- id:abc0001 -->
  - Stream: 2

- [ ] 2. Dependent <!-- id:abc0002 -->
  - Blocked-by: abc0001 (Blocker)
  - Stream: 3
  - [ ] 2.1. Child <!-- id:abc0003 -->
    - Blocked-by: abc0001 (Blocker)
    - Stream: 3
`

// T-2176: next --phase table output shows stream and blocked-by for every
// task, subtasks included.
func TestNextPhaseTableShowsStreamAndBlockedBy(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, phaseMetadataFile)

	out := mustRun(t, dir, "next", "tasks.md", "--phase", "--format", "table")
	if !strings.Contains(out, "STREAM") || !strings.Contains(out, "BLOCKEDBY") {
		t.Fatalf("table has no Stream/BlockedBy columns:\n%s", out)
	}
	rows := map[string][]string{
		"| 1 ":   {"| 2 "},
		"| 2 ":   {"| 3 ", "| 1 "},
		"| 2.1 ": {"| 3 ", "| 1 "},
	}
	for prefix, cells := range rows {
		line := tableRow(out, prefix)
		if line == "" {
			t.Errorf("no table row for %q:\n%s", prefix, out)
			continue
		}
		for _, cell := range cells {
			if !strings.Contains(line[len(prefix):], cell) {
				t.Errorf("row %q is missing %q", line, cell)
			}
		}
	}
}

func tableRow(out, prefix string) string {
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

// T-2176: next --phase markdown output shows stream for every task and
// blocked-by whenever a task has dependencies.
func TestNextPhaseMarkdownShowsStreamAndBlockedBy(t *testing.T) {
	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, phaseMetadataFile)

	out := mustRun(t, dir, "next", "tasks.md", "--phase", "--format", "markdown")
	for _, want := range []string{
		"- [ ] 1. Blocker\n  Stream: 2\n",
		"- [ ] 2. Dependent (blocked by: 1)\n  Stream: 3\n  Blocked-by: 1\n",
		"  - [ ] 2.1. Child (blocked by: 1)\n    Stream: 3\n    Blocked-by: 1\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown output is missing %q:\n%s", want, out)
		}
	}
}

const claimContextFile = `---
references:
  - docs/spec.md
---
# Tasks

- [ ] 1. Implement <!-- id:abc0001 -->
  - Important detail
  - References: src/impl.go
`

// T-2184: next --claim output includes the claimed task's details, task
// references and front matter references, in every format.
func TestNextClaimIncludesTaskContext(t *testing.T) {
	for _, f := range []string{"json", "markdown", "table"} {
		t.Run(f, func(t *testing.T) {
			resetOutputContextFlags(t)
			dir := t.TempDir()
			writeTaskFile(t, dir, claimContextFile)

			out := mustRun(t, dir, "next", "tasks.md", "--claim", "agent", "--format", f)
			for _, want := range []string{"Important detail", "src/impl.go", "docs/spec.md"} {
				if !strings.Contains(out, want) {
					t.Errorf("%s claim output is missing %q:\n%s", f, want, out)
				}
			}
		})
	}

	resetOutputContextFlags(t)
	dir := t.TempDir()
	writeTaskFile(t, dir, claimContextFile)
	got := decodeJSON(t, mustRun(t, dir, "next", "tasks.md", "--claim", "agent", "--format", "json", "--dry-run"))
	refs, _ := got["front_matter_references"].([]any)
	if len(refs) != 1 || refs[0] != "docs/spec.md" {
		t.Errorf("front_matter_references = %v, want [docs/spec.md]", got["front_matter_references"])
	}
}

// T-2010: an explicitly empty --claim is rejected and leaves the file
// unchanged, instead of running the read-only next path.
func TestNextEmptyClaimIsRejected(t *testing.T) {
	for name, value := range map[string]string{"empty": "", "whitespace": "  "} {
		t.Run(name, func(t *testing.T) {
			resetOutputContextFlags(t)
			dir := t.TempDir()
			const content = "# Repro\n\n- [ ] 1. Ready\n"
			writeTaskFile(t, dir, content)

			err := runRuneCLIError(t, dir, "next", "tasks.md", "--claim", value, "--format", "json")
			if err == nil || !strings.Contains(err.Error(), "--claim requires a non-empty agent ID") {
				t.Errorf("want empty-claim error, got %v", err)
			}
			if got, _ := os.ReadFile(filepath.Join(dir, "tasks.md")); string(got) != content {
				t.Errorf("file changed: %q", got)
			}

			// A later next without --claim still runs the read-only path
			out := mustRun(t, dir, "next", "tasks.md", "--format", "json")
			if !strings.Contains(out, `"next_task"`) {
				t.Errorf("plain next after a rejected claim failed:\n%s", out)
			}
		})
	}
}

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

// runRuneCLI runs rune with args inside dir and returns stdout and stderr.
// Command flags that persist between Execute calls are reset first.
func runRuneCLI(t *testing.T, dir string, args ...string) (string, string) {
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
		streamsAvailable, streamsJSON = false, false
		dryRun = false
		resetBatchFlags()
	}
	resetFlags()
	t.Cleanup(resetFlags)

	oldStdout, oldStderr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW

	var cobraOut bytes.Buffer
	rootCmd.SetOut(&cobraOut)
	rootCmd.SetArgs(args)
	runErr := rootCmd.Execute()
	rootCmd.SetArgs([]string{})
	rootCmd.SetOut(nil)

	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	stdout, _ := io.ReadAll(outR)
	stderr, _ := io.ReadAll(errR)

	if runErr != nil {
		t.Fatalf("rune %v: %v\nstderr: %s", args, runErr, stderr)
	}
	return string(stdout) + cobraOut.String(), string(stderr)
}

func writeTaskFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(dir+"/tasks.md", []byte(content), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
}

type streamsJSONOutput struct {
	Streams []struct {
		ID      int      `json:"id"`
		Ready   []string `json:"ready"`
		Blocked []string `json:"blocked"`
	} `json:"streams"`
}

func parseStreams(t *testing.T, out string) streamsJSONOutput {
	t.Helper()
	var got streamsJSONOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse streams JSON: %v\n%s", err, out)
	}
	return got
}

// T-1762: a duplicated stable ID must not rebind task 3's dependency to the
// later, pending task, and the duplicate must be reported.
func TestDuplicateStableIDDoesNotRebindDependency(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [x] 1. First <!-- id:abc1234 -->
- [ ] 2. Second <!-- id:abc1234 -->
- [ ] 3. Third
  - Blocked-by: abc1234 (First)
`)

	out, stderr := runRuneCLI(t, dir, "streams", "tasks.md", "--json")
	streams := parseStreams(t, out)
	if len(streams.Streams) != 1 || len(streams.Streams[0].Blocked) != 0 {
		t.Errorf("task 3 reported blocked by the duplicate: %s", out)
	}
	if !strings.Contains(stderr, "duplicate stable ID abc1234") {
		t.Errorf("expected duplicate stable ID warning on stderr, got %q", stderr)
	}

	out, _ = runRuneCLI(t, dir, "list", "tasks.md", "--format", "json")
	var list struct {
		Tasks []struct {
			BlockedBy []string `json:"blockedBy"`
		} `json:"Tasks"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatalf("parse list JSON: %v\n%s", err, out)
	}
	if got := list.Tasks[2].BlockedBy; len(got) != 1 || got[0] != "1" {
		t.Errorf("task 3 blockedBy = %v, want [1]", got)
	}
}

// T-1772: removing a parent through the CLI must drop references to its
// deleted child's stable ID.
func TestRemoveParentCleansChildDependencyCLI(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [ ] 1. Parent <!-- id:aaa0001 -->
  - [ ] 1.1. Child <!-- id:bbb0002 -->
- [ ] 2. Other <!-- id:ccc0003 -->
  - Blocked-by: bbb0002 (Child)
`)

	runRuneCLI(t, dir, "remove", "tasks.md", "1")

	content, _ := os.ReadFile(dir + "/tasks.md")
	if strings.Contains(string(content), "bbb0002") {
		t.Errorf("stale reference to deleted child remains:\n%s", content)
	}
	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--stream", "1", "--format", "json")
	if !strings.Contains(out, `"title": "Other"`) {
		t.Errorf("surviving task should be ready, got: %s", out)
	}
}

// T-1772: the same cleanup applies to batch remove.
func TestRemoveParentCleansChildDependencyBatch(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [ ] 1. Parent <!-- id:aaa0001 -->
  - [ ] 1.1. Child <!-- id:bbb0002 -->
- [ ] 2. Other <!-- id:ccc0003 -->
  - Blocked-by: bbb0002 (Child)
`)

	req, _ := json.Marshal(task.BatchRequest{
		File:       "tasks.md",
		Operations: []task.Operation{{Type: "remove", ID: "1"}},
	})
	runRuneCLI(t, dir, "batch", "--input", string(req), "--format", "json")

	content, _ := os.ReadFile(dir + "/tasks.md")
	if strings.Contains(string(content), "bbb0002") {
		t.Errorf("stale reference to deleted child remains:\n%s", content)
	}
}

// T-1777: plain next (no --stream/--claim) must not return a blocked task.
func TestPlainNextSkipsBlockedTask(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [ ] 1. Blocked <!-- id:aaa0001 -->
  - Blocked-by: bbb0002 (Blocker)
- [ ] 2. Blocker <!-- id:bbb0002 -->
`)

	for _, args := range [][]string{
		{"next", "tasks.md", "--format", "json"},
		{"next", "tasks.md", "--one", "--format", "json"},
	} {
		out, _ := runRuneCLI(t, dir, args...)
		if !strings.Contains(out, `"title": "Blocker"`) || strings.Contains(out, `"title": "Blocked"`) {
			t.Errorf("rune %v should return task 2 (Blocker), got: %s", args, out)
		}
	}
}

// T-1777: when work remains but none of it is ready, plain next says so
// rather than claiming everything is complete.
func TestPlainNextNoReadyWork(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [ ] 1. Blocked <!-- id:aaa0001 -->
  - Blocked-by: bbb0002 (Claimed)
- [-] 2. Claimed <!-- id:bbb0002 -->
  - Owner: agent-1
`)

	out, _ := runRuneCLI(t, dir, "next", "tasks.md", "--format", "json")
	if strings.Contains(out, "All tasks are complete") || !strings.Contains(out, "No ready tasks") {
		t.Errorf("expected a no-ready-tasks message, got: %s", out)
	}
}

// T-2126: completing the last child of a blocked parent must not
// auto-complete the parent or unblock its dependents.
func TestCompleteDoesNotAutoCompleteBlockedParent(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [ ] 1. Blocker <!-- id:aaa0001 -->
- [ ] 2. Parent <!-- id:bbb0002 -->
  - Blocked-by: aaa0001 (Blocker)
  - [ ] 2.1. Child
- [ ] 3. Downstream <!-- id:ccc0003 -->
  - Blocked-by: bbb0002 (Parent)
`)

	out, _ := runRuneCLI(t, dir, "complete", "tasks.md", "2.1", "--format", "json")
	if strings.Contains(out, "auto_completed") {
		t.Errorf("blocked parent was auto-completed: %s", out)
	}

	streams := parseStreams(t, mustRun(t, dir, "streams", "tasks.md", "--json"))
	if got := streams.Streams[0].Ready; len(got) != 1 || got[0] != "1" {
		t.Errorf("ready = %v, want only [1]", got)
	}
}

// T-2161: a completed blocker with a pending child must not unlock dependents.
func TestCompletedBlockerWithPendingChildKeepsDependentBlocked(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, `# T
- [x] 1. Blocker parent <!-- id:aaa0001 -->
  - [ ] 1.1. Child
- [ ] 2. Dependent <!-- id:bbb0002 -->
  - Blocked-by: aaa0001 (Blocker parent)
  - Stream: 2
`)

	streams := parseStreams(t, mustRun(t, dir, "streams", "tasks.md", "--json"))
	for _, s := range streams.Streams {
		if s.ID == 2 && (len(s.Ready) != 0 || len(s.Blocked) != 1) {
			t.Errorf("stream 2 ready=%v blocked=%v, want task 2 blocked", s.Ready, s.Blocked)
		}
	}

	out := mustRun(t, dir, "next", "tasks.md", "--stream", "2", "--format", "json")
	if strings.Contains(out, `"title": "Dependent"`) {
		t.Errorf("next --stream 2 returned the blocked dependent: %s", out)
	}
}

func mustRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, _ := runRuneCLI(t, dir, args...)
	return out
}

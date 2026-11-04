package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

const blockerCleanupWarning = "removed dependency references from 1 task(s)"

var wantBlockerWarning = task.Warning{Code: task.WarnDependentsRemoved, Message: blockerCleanupWarning, TaskID: "1"}

// writeBlockerFile writes tasks.md in the current directory with task 2
// blocked by task 1.
func writeBlockerFile(t *testing.T) {
	t.Helper()
	tl := task.NewTaskList("Tasks")
	if _, err := tl.AddTask("", "Blocker", ""); err != nil {
		t.Fatalf("AddTask: %v", err)
	}
	if _, err := tl.AddTaskWithOptions("", "Dependent", task.AddOptions{BlockedBy: []string{"1"}}); err != nil {
		t.Fatalf("AddTaskWithOptions: %v", err)
	}
	if err := tl.WriteFile("tasks.md"); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// T-1860: removing a blocker reports the dependency cleanup, in a "warnings"
// array for JSON and on stderr for table and markdown output.
func TestRemoveBlockerReportsWarning(t *testing.T) {
	for _, outFormat := range []string{formatJSON, formatTable, formatMarkdown} {
		t.Run(outFormat, func(t *testing.T) {
			t.Chdir(t.TempDir())
			writeBlockerFile(t)

			oldFormat, oldDryRun := format, dryRun
			format, dryRun = outFormat, false
			t.Cleanup(func() { format, dryRun = oldFormat, oldDryRun })

			var runErr error
			var stdout string
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					runErr = runRemove(&cobra.Command{}, []string{"tasks.md", "1"})
				})
			})
			if runErr != nil {
				t.Fatalf("runRemove: %v", runErr)
			}

			if outFormat == formatJSON {
				var resp RemoveResponse
				if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
					t.Fatalf("invalid JSON %q: %v", stdout, err)
				}
				if len(resp.Warnings) != 1 || resp.Warnings[0] != wantBlockerWarning {
					t.Errorf("warnings = %+v, want [%+v]", resp.Warnings, wantBlockerWarning)
				}
				if stderr != "" {
					t.Errorf("JSON output should not write warnings to stderr, got %q", stderr)
				}
				return
			}
			if want := "Warning: " + blockerCleanupWarning + "\n"; stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
			if strings.Contains(stdout, "Warning") {
				t.Errorf("warning should not be on stdout: %q", stdout)
			}
		})
	}
}

func TestBatchRemoveBlockerReportsWarningOnCommandOutput(t *testing.T) {
	for _, outFormat := range []string{formatJSON, formatTable} {
		t.Run(outFormat, func(t *testing.T) {
			t.Cleanup(resetBatchFlags)
			t.Chdir(t.TempDir())
			writeBlockerFile(t)

			input := `{"file":"tasks.md","operations":[{"type":"remove","id":"1"}]}`
			var stdout bytes.Buffer
			rootCmd.SetOut(&stdout)
			t.Cleanup(func() { rootCmd.SetOut(nil) })
			rootCmd.SetArgs([]string{"batch", "--input", input, "--format", outFormat})

			var runErr error
			stderr := captureStderr(t, func() { runErr = rootCmd.Execute() })
			if runErr != nil {
				t.Fatalf("batch: %v", runErr)
			}

			if outFormat == formatJSON {
				var resp task.BatchResponse
				if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
					t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
				}
				if len(resp.Warnings) != 1 || resp.Warnings[0] != wantBlockerWarning {
					t.Errorf("warnings = %+v, want [%+v]", resp.Warnings, wantBlockerWarning)
				}
				return
			}
			if !strings.Contains(stderr, "Warning: "+blockerCleanupWarning+"\n") {
				t.Errorf("stderr = %q, want cleanup warning", stderr)
			}
		})
	}
}

package cmd

import (
	"os"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("tasks.md", []byte(content), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
	return "tasks.md"
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}

// T-1456: uncomplete/progress reopen completed ancestors.
func TestStatusRollbackReopensCompletedAncestors(t *testing.T) {
	tests := map[string]struct {
		run        func(*cobra.Command, []string) error
		wantStatus task.Status
	}{
		"uncomplete": {run: runUncomplete, wantStatus: task.Pending},
		"progress":   {run: runProgress, wantStatus: task.InProgress},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file := writeTemp(t, "# Demo\n\n- [x] 1. Root\n  - [x] 1.1. Parent\n    - [x] 1.1.1. Child\n    - [x] 1.1.2. Sibling\n  - [x] 1.2. Other\n- [x] 2. Unrelated\n")
			dryRun = false
			format = formatTable
			t.Cleanup(func() { dryRun = false })

			captureStdout(t, func() {
				if err := tc.run(&cobra.Command{}, []string{file, "1.1.1"}); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})

			tl, err := task.ParseFile(file)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			want := map[string]task.Status{
				"1.1.1": tc.wantStatus,
				"1.1":   task.Pending,
				"1":     task.Pending,
				"1.1.2": task.Completed,
				"1.2":   task.Completed,
				"2":     task.Completed,
			}
			for id, status := range want {
				if got := tl.FindTask(id).Status; got != status {
					t.Errorf("task %s status = %v, want %v", id, got, status)
				}
			}
		})
	}
}

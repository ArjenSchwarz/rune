package task

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// batchRunner runs ops against tl either through the plain or the phase-aware
// batch path. The phase-aware path is given one phase marker so it does not
// fall back to ExecuteBatch, and writes to a file in the current directory.
type batchRunner func(t *testing.T, tl *TaskList, ops []Operation, dryRun bool) *BatchResponse

func runPlainBatch(t *testing.T, tl *TaskList, ops []Operation, dryRun bool) *BatchResponse {
	t.Helper()
	resp, err := tl.ExecuteBatch(ops, dryRun)
	if err != nil {
		t.Fatalf("ExecuteBatch: %v", err)
	}
	return resp
}

func runPhasedBatch(t *testing.T, tl *TaskList, ops []Operation, dryRun bool) *BatchResponse {
	t.Helper()
	t.Chdir(t.TempDir())
	markers := []PhaseMarker{{Name: "Work", AfterTaskID: ""}}
	resp, err := tl.ExecuteBatchWithPhases(ops, dryRun, markers, "tasks.md")
	if err != nil {
		t.Fatalf("ExecuteBatchWithPhases: %v", err)
	}
	return resp
}

var batchRunners = map[string]batchRunner{
	"plain":  runPlainBatch,
	"phased": runPhasedBatch,
}

// T-1776: auto_completed must describe the final state of the batch, not
// every auto-completion event that happened along the way.
func TestBatchAutoCompletedReflectsFinalState(t *testing.T) {
	newList := func() *TaskList {
		tl := NewTaskList("Tasks")
		tl.AddTask("", "Parent", "")
		tl.AddTask("1", "Child", "")
		return tl
	}

	tests := map[string]struct {
		setup func() *TaskList
		ops   []Operation
		want  []string
	}{
		"later update reverses the auto-completed parent": {
			setup: newList,
			ops: []Operation{
				{Type: updateOperation, ID: "1.1", Status: StatusPtr(Completed)},
				{Type: updateOperation, ID: "1", Status: StatusPtr(Pending)},
			},
			want: []string{},
		},
		"later remove deletes the auto-completed parent": {
			setup: newList,
			ops: []Operation{
				{Type: updateOperation, ID: "1.1", Status: StatusPtr(Completed)},
				{Type: removeOperation, ID: "1"},
			},
			want: []string{},
		},
		"later add gives the auto-completed parent an incomplete child": {
			setup: newList,
			ops: []Operation{
				{Type: updateOperation, ID: "1.1", Status: StatusPtr(Completed)},
				{Type: addOperation, Parent: "1", Title: "New child"},
			},
			want: []string{},
		},
		"later explicit completion is not reported as automatic": {
			setup: newList,
			ops: []Operation{
				{Type: updateOperation, ID: "1.1", Status: StatusPtr(Completed)},
				{Type: updateOperation, ID: "1", Status: StatusPtr(Pending)},
				{Type: updateOperation, ID: "1", Status: StatusPtr(Completed)},
			},
			want: []string{},
		},
		"auto-completed parent is reported under its final ID after renumbering": {
			setup: func() *TaskList {
				tl := NewTaskList("Tasks")
				tl.AddTask("", "Removed", "")
				tl.AddTask("", "Parent", "")
				tl.AddTask("2", "Child", "")
				return tl
			},
			ops: []Operation{
				{Type: updateOperation, ID: "2.1", Status: StatusPtr(Completed)},
				{Type: removeOperation, ID: "1"},
			},
			want: []string{"1"},
		},
		"unreversed auto-completion is still reported": {
			setup: newList,
			ops: []Operation{
				{Type: updateOperation, ID: "1.1", Status: StatusPtr(Completed)},
			},
			want: []string{"1"},
		},
	}

	for runnerName, run := range batchRunners {
		for name, tc := range tests {
			for _, dryRun := range []bool{true, false} {
				mode := "live"
				if dryRun {
					mode = "dry-run"
				}
				t.Run(runnerName+"/"+mode+"/"+name, func(t *testing.T) {
					tl := tc.setup()
					resp := run(t, tl, tc.ops, dryRun)
					if !resp.Success {
						t.Fatalf("batch failed: %v", resp.Errors)
					}
					if !slices.Equal(resp.AutoCompleted, tc.want) {
						t.Errorf("auto_completed = %v, want %v", resp.AutoCompleted, tc.want)
					}
					if dryRun {
						return
					}
					for _, id := range resp.AutoCompleted {
						task := tl.FindTask(id)
						if task == nil || task.Status != Completed || !allChildrenComplete(task) {
							t.Errorf("reported task %s is not complete in the final list", id)
						}
					}
				})
			}
		}
	}
}

// T-1780: errors name the operation's position in the request, not its
// position after sortOperationsForExecution reordered a run.
func TestBatchErrorsUseInputOperationIndex(t *testing.T) {
	tests := map[string]struct {
		ops     []Operation
		wantErr string
	}{
		"invalid positioned add after a higher positioned add": {
			ops: []Operation{
				{Type: addOperation, Position: "1"},
				{Type: addOperation, Title: "Valid", Position: "2"},
			},
			wantErr: "operation 1: add operation requires title",
		},
		"missing remove after a lower remove": {
			ops: []Operation{
				{Type: removeOperation, ID: "1"},
				{Type: removeOperation, ID: "9"},
			},
			wantErr: "operation 2: task 9 not found",
		},
		"missing update before a positioned add": {
			ops: []Operation{
				{Type: updateOperation, ID: "9", Title: "x"},
				{Type: addOperation, Title: "Valid", Position: "1"},
			},
			wantErr: "operation 1: task 9 not found",
		},
	}

	for runnerName, run := range batchRunners {
		for name, tc := range tests {
			t.Run(runnerName+"/"+name, func(t *testing.T) {
				tl := NewTaskList("Tasks")
				tl.AddTask("", "One", "")
				tl.AddTask("", "Two", "")
				resp := run(t, tl, tc.ops, true)
				if resp.Success {
					t.Fatal("expected the batch to fail")
				}
				if len(resp.Errors) != 1 || resp.Errors[0] != tc.wantErr {
					t.Errorf("errors = %q, want [%q]", resp.Errors, tc.wantErr)
				}
			})
		}
	}
}

// T-1933: positioned adds that target the same slot keep their request order.
func TestBatchEqualPositionAddsKeepRequestOrder(t *testing.T) {
	tests := map[string]struct {
		parent string
		pos    string
		want   []string
	}{
		"root level": {
			pos:  "2",
			want: []string{"One", "First requested", "Second requested", "Two"},
		},
		"same parent subtasks": {
			parent: "1",
			pos:    "1.2",
			want:   []string{"Child A", "First requested", "Second requested", "Child B"},
		},
	}

	for runnerName, run := range batchRunners {
		for name, tc := range tests {
			t.Run(runnerName+"/"+name, func(t *testing.T) {
				tl := NewTaskList("Tasks")
				tl.AddTask("", "One", "")
				tl.AddTask("", "Two", "")
				tl.AddTask("1", "Child A", "")
				tl.AddTask("1", "Child B", "")
				ops := []Operation{
					{Type: addOperation, Parent: tc.parent, Title: "First requested", Position: tc.pos},
					{Type: addOperation, Parent: tc.parent, Title: "Second requested", Position: tc.pos},
				}
				resp := run(t, tl, ops, false)
				if !resp.Success {
					t.Fatalf("batch failed: %v", resp.Errors)
				}
				tasks := tl.Tasks
				if tc.parent != "" {
					tasks = tl.FindTask(tc.parent).Children
				}
				var got []string
				for _, task := range tasks {
					got = append(got, task.Title)
				}
				if !slices.Equal(got, tc.want) {
					t.Errorf("titles = %v, want %v", got, tc.want)
				}
			})
		}
	}
}

// T-1860: removing a blocker reports the dependency cleanup as a warning.
func TestBatchRemoveBlockerReportsWarning(t *testing.T) {
	for runnerName, run := range batchRunners {
		for _, dryRun := range []bool{true, false} {
			t.Run(runnerName+"/dryRun="+map[bool]string{true: "true", false: "false"}[dryRun], func(t *testing.T) {
				tl := NewTaskList("Tasks")
				tl.AddTask("", "Blocker", "")
				if _, err := tl.AddTaskWithOptions("", "Dependent", AddOptions{BlockedBy: []string{"1"}}); err != nil {
					t.Fatalf("AddTaskWithOptions: %v", err)
				}
				resp := run(t, tl, []Operation{{Type: removeOperation, ID: "1"}}, dryRun)
				if !resp.Success {
					t.Fatalf("batch failed: %v", resp.Errors)
				}
				want := []Warning{{Code: WarnDependentsRemoved, Message: "removed dependency references from 1 task(s)", TaskID: "1"}}
				if !slices.Equal(resp.Warnings, want) {
					t.Errorf("warnings = %q, want %q", resp.Warnings, want)
				}
			})
		}
	}
}

func TestRemoveTaskWithPhasesReturnsDependencyWarning(t *testing.T) {
	tests := map[string]string{
		"without phases": "# Tasks\n\n- [ ] 1. Blocker\n- [ ] 2. Dependent\n",
		"with phases":    "# Tasks\n\n## Work\n\n- [ ] 1. Blocker\n- [ ] 2. Dependent\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tl, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatalf("ParseMarkdown: %v", err)
			}
			if err := tl.UpdateTaskWithOptions("2", UpdateOptions{BlockedBy: []string{"1"}}); err != nil {
				t.Fatalf("UpdateTaskWithOptions: %v", err)
			}
			tl.FilePath = "tasks.md"
			warnings, err := tl.RemoveTaskWithPhases("1", []byte(content))
			if err != nil {
				t.Fatalf("RemoveTaskWithPhases: %v", err)
			}
			want := []Warning{{Code: WarnDependentsRemoved, Message: "removed dependency references from 1 task(s)", TaskID: "1"}}
			if !slices.Equal(warnings, want) {
				t.Errorf("warnings = %q, want %q", warnings, want)
			}
		})
	}
}

// T-1888: the dry-run preview is the resulting file, including front matter.
func TestBatchDryRunPreviewIncludesFrontMatter(t *testing.T) {
	content := "---\nreferences:\n  - docs/design.md\nmetadata:\n  owner: team\n---\n# Tasks\n\n- [ ] 1. One\n"
	for runnerName, run := range batchRunners {
		t.Run(runnerName, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatalf("ParseMarkdown: %v", err)
			}
			resp := run(t, tl, []Operation{{Type: addOperation, Title: "Two"}}, true)
			if !resp.Success {
				t.Fatalf("batch failed: %v", resp.Errors)
			}
			for _, want := range []string{"---\nreferences:\n", "docs/design.md", "owner: team", "- [ ] 2. Two"} {
				if !strings.Contains(resp.Preview, want) {
					t.Errorf("preview missing %q:\n%s", want, resp.Preview)
				}
			}
			if !strings.HasPrefix(resp.Preview, "---\n") {
				t.Errorf("preview should start with front matter:\n%s", resp.Preview)
			}

			// The preview should match what a live run writes.
			t.Chdir(t.TempDir())
			live, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatalf("ParseMarkdown: %v", err)
			}
			if resp := run(t, live, []Operation{{Type: addOperation, Title: "Two"}}, false); !resp.Success {
				t.Fatalf("live batch failed: %v", resp.Errors)
			}
			if runnerName == "plain" {
				if err := live.WriteFile("tasks.md"); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			}
			written, err := os.ReadFile("tasks.md")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(written) != resp.Preview {
				t.Errorf("preview differs from written file\npreview:\n%s\nwritten:\n%s", resp.Preview, written)
			}
		})
	}
}

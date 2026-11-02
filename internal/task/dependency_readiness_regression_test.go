package task

import (
	"slices"
	"testing"
)

// T-1762: parsing a file with a duplicated stable ID must not let the later
// task take over the ID. The first occurrence keeps it, the later occurrence
// loses it, and a duplicate_stable_id warning is recorded for the later task.
func TestParseDuplicateStableIDKeepsFirstOccurrence(t *testing.T) {
	content := `# T
- [x] 1. First <!-- id:abc1234 -->
- [ ] 2. Second <!-- id:abc1234 -->
- [ ] 3. Third
  - Blocked-by: abc1234 (First)
`
	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}

	if got := tl.Tasks[0].StableID; got != "abc1234" {
		t.Errorf("first occurrence StableID = %q, want abc1234", got)
	}
	if got := tl.Tasks[1].StableID; got != "" {
		t.Errorf("duplicate occurrence StableID = %q, want it dropped", got)
	}

	if len(tl.Warnings) != 1 {
		t.Fatalf("got %d warnings, want 1: %+v", len(tl.Warnings), tl.Warnings)
	}
	if w := tl.Warnings[0]; w.Code != WarnDuplicateStableID || w.TaskID != "2" {
		t.Errorf("warning = %+v, want code %q on task 2", w, WarnDuplicateStableID)
	}

	idx := BuildDependencyIndex(tl.Tasks)
	if idx.IsBlocked(&tl.Tasks[2]) {
		t.Error("task 3 is blocked, but its blocker (task 1) is completed")
	}
	if got := idx.TranslateToHierarchical(tl.Tasks[2].BlockedBy); !slices.Equal(got, []string{"1"}) {
		t.Errorf("blockedBy translates to %v, want [1]", got)
	}
}

// T-1762: an index built from tasks that share a stable ID (for example
// constructed in code rather than parsed) keeps binding to the first task.
func TestBuildDependencyIndexDuplicateStableIDFirstWins(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "First", Status: Completed, StableID: "abc1234"},
		{ID: "2", Title: "Second", Status: Pending, StableID: "abc1234"},
	}
	idx := BuildDependencyIndex(tasks)
	if got := idx.GetTask("abc1234"); got == nil || got.ID != "1" {
		t.Errorf("GetTask(abc1234) = %+v, want task 1", got)
	}
}

// T-1772: removing a parent must also clean up references to the stable IDs
// of its deleted descendants.
func TestRemoveTaskWithDependentsCleansDescendantReferences(t *testing.T) {
	tl := &TaskList{
		Tasks: []Task{
			{
				ID: "1", Title: "Parent", Status: Pending, StableID: "aaa0001",
				Children: []Task{
					{ID: "1.1", Title: "Child", Status: Pending, StableID: "bbb0002", ParentID: "1"},
				},
			},
			{ID: "2", Title: "Other", Status: Pending, StableID: "ccc0003", BlockedBy: []string{"bbb0002"}},
			{ID: "3", Title: "Both", Status: Pending, StableID: "ddd0004", BlockedBy: []string{"aaa0001", "bbb0002"}},
			{ID: "4", Title: "Unrelated", Status: Pending, StableID: "eee0005", BlockedBy: []string{"ccc0003"}},
		},
	}

	warnings, err := tl.RemoveTaskWithDependents("1")
	if err != nil {
		t.Fatalf("RemoveTaskWithDependents: %v", err)
	}

	for _, task := range tl.Tasks[:2] {
		if len(task.BlockedBy) != 0 {
			t.Errorf("task %q still blocked by %v after its blockers were removed", task.Title, task.BlockedBy)
		}
	}
	if got := tl.Tasks[2].BlockedBy; !slices.Equal(got, []string{"ccc0003"}) {
		t.Errorf("unrelated dependency changed: %v", got)
	}
	if want := []string{"removed dependency references from 2 task(s)"}; !slices.Equal(warnings, want) {
		t.Errorf("warnings = %v, want %v", warnings, want)
	}
}

// T-1777: plain next must skip tasks that are blocked or claimed and return
// the next task with ready work.
func TestFindNextReadyTask(t *testing.T) {
	tests := map[string]struct {
		tasks  []Task
		wantID string
	}{
		"blocked first task is skipped for its ready blocker": {
			tasks: []Task{
				{ID: "1", Title: "Blocked", Status: Pending, StableID: "aaa0001", BlockedBy: []string{"bbb0002"}},
				{ID: "2", Title: "Blocker", Status: Pending, StableID: "bbb0002"},
			},
			wantID: "2",
		},
		"claimed task is skipped": {
			tasks: []Task{
				{ID: "1", Title: "Claimed", Status: InProgress, Owner: "agent-1"},
				{ID: "2", Title: "Free", Status: Pending},
			},
			wantID: "2",
		},
		"unclaimed in-progress task is still returned": {
			tasks: []Task{
				{ID: "1", Title: "Current", Status: InProgress},
				{ID: "2", Title: "Later", Status: Pending},
			},
			wantID: "1",
		},
		"completed parent with blocked child only is skipped": {
			tasks: []Task{
				{ID: "1", Title: "Parent", Status: Completed, Children: []Task{
					{ID: "1.1", Title: "Blocked child", Status: Pending, ParentID: "1", BlockedBy: []string{"bbb0002"}},
				}},
				{ID: "2", Title: "Blocker", Status: Pending, StableID: "bbb0002"},
			},
			wantID: "2",
		},
		"everything blocked returns nil": {
			tasks: []Task{
				{ID: "1", Title: "A", Status: Pending, StableID: "aaa0001", BlockedBy: []string{"bbb0002"}},
				{ID: "2", Title: "B", Status: InProgress, StableID: "bbb0002", Owner: "agent-1"},
			},
			wantID: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FindNextReadyTask(tc.tasks, BuildDependencyIndex(tc.tasks))
			gotID := ""
			if got != nil {
				gotID = got.ID
			}
			if gotID != tc.wantID {
				t.Errorf("FindNextReadyTask = %q, want %q", gotID, tc.wantID)
			}
		})
	}
}

// T-1777: --one must follow the first child with ready work, not the first
// incomplete child.
func TestFilterToFirstReadyPathSkipsBlockedChild(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Parent", Status: Pending, Children: []Task{
			{ID: "1.1", Title: "Blocked", Status: Pending, ParentID: "1", BlockedBy: []string{"bbb0002"}},
			{ID: "1.2", Title: "Blocker", Status: Pending, ParentID: "1", StableID: "bbb0002"},
		}},
	}
	idx := BuildDependencyIndex(tasks)
	next := FindNextReadyTask(tasks, idx)
	if next == nil {
		t.Fatal("FindNextReadyTask returned nil")
	}
	FilterToFirstReadyPath(next, idx)
	if len(next.IncompleteChildren) != 1 || next.IncompleteChildren[0].ID != "1.2" {
		t.Errorf("--one path = %+v, want only 1.2", next.IncompleteChildren)
	}
}

// T-2126: completing the final child must not auto-complete a parent whose
// own dependencies are unresolved.
func TestAutoCompleteParentsSkipsBlockedParent(t *testing.T) {
	tl := &TaskList{
		Tasks: []Task{
			{ID: "1", Title: "Blocker", Status: Pending, StableID: "aaa0001"},
			{ID: "2", Title: "Parent", Status: Pending, StableID: "bbb0002", BlockedBy: []string{"aaa0001"}, Children: []Task{
				{ID: "2.1", Title: "Child", Status: Completed, ParentID: "2"},
			}},
			{ID: "3", Title: "Downstream", Status: Pending, StableID: "ccc0003", BlockedBy: []string{"bbb0002"}},
		},
	}

	completed, err := tl.AutoCompleteParents("2.1")
	if err != nil {
		t.Fatalf("AutoCompleteParents: %v", err)
	}
	if len(completed) != 0 {
		t.Errorf("auto-completed %v, want none while parent is blocked", completed)
	}
	if tl.Tasks[1].Status == Completed {
		t.Error("blocked parent was marked completed")
	}
	if !BuildDependencyIndex(tl.Tasks).IsBlocked(&tl.Tasks[2]) {
		t.Error("downstream task became ready")
	}
}

// T-2161: a completed blocker with incomplete descendants does not satisfy
// a dependency.
func TestIsReadyRequiresBlockerSubtreeComplete(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Blocker parent", Status: Completed, StableID: "aaa0001", Children: []Task{
			{ID: "1.1", Title: "Child", Status: Pending, ParentID: "1"},
		}},
		{ID: "2", Title: "Dependent", Status: Pending, StableID: "bbb0002", BlockedBy: []string{"aaa0001"}},
	}
	idx := BuildDependencyIndex(tasks)
	if idx.IsReady(&tasks[1]) {
		t.Error("dependent is ready while blocker 1.1 is still pending")
	}

	tasks[0].Children[0].Status = Completed
	idx = BuildDependencyIndex(tasks)
	if !idx.IsReady(&tasks[1]) {
		t.Error("dependent is not ready after the blocker subtree completed")
	}
}

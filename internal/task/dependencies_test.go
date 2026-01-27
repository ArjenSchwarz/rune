package task

import (
	"testing"
)

func TestBuildDependencyIndex(t *testing.T) {
	tests := map[string]struct {
		tasks          []Task
		wantByStableID map[string]string   // stableID -> hierarchicalID
		wantDependents map[string][]string // stableID -> list of dependent stableIDs
	}{
		"empty_task_list": {
			tasks:          []Task{},
			wantByStableID: map[string]string{},
			wantDependents: map[string][]string{},
		},
		"single_task_no_dependencies": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", StableID: "abc0001"},
			},
			wantByStableID: map[string]string{"abc0001": "1"},
			wantDependents: map[string][]string{},
		},
		"two_tasks_with_dependency": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", StableID: "abc0001"},
				{ID: "2", Title: "Task 2", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
			},
			wantByStableID: map[string]string{"abc0001": "1", "abc0002": "2"},
			wantDependents: map[string][]string{"abc0001": {"abc0002"}},
		},
		"nested_tasks_with_dependencies": {
			tasks: []Task{
				{
					ID:       "1",
					Title:    "Phase 1",
					StableID: "abc0001",
					Children: []Task{
						{ID: "1.1", Title: "Subtask 1.1", StableID: "abc0011"},
						{ID: "1.2", Title: "Subtask 1.2", StableID: "abc0012", BlockedBy: []string{"abc0011"}},
					},
				},
				{ID: "2", Title: "Task 2", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
			},
			wantByStableID: map[string]string{
				"abc0001": "1",
				"abc0011": "1.1",
				"abc0012": "1.2",
				"abc0002": "2",
			},
			wantDependents: map[string][]string{
				"abc0001": {"abc0002"},
				"abc0011": {"abc0012"},
			},
		},
		"multiple_dependents": {
			tasks: []Task{
				{ID: "1", Title: "Setup", StableID: "abc0001"},
				{ID: "2", Title: "Task A", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
				{ID: "3", Title: "Task B", StableID: "abc0003", BlockedBy: []string{"abc0001"}},
				{ID: "4", Title: "Task C", StableID: "abc0004", BlockedBy: []string{"abc0001"}},
			},
			wantByStableID: map[string]string{
				"abc0001": "1",
				"abc0002": "2",
				"abc0003": "3",
				"abc0004": "4",
			},
			wantDependents: map[string][]string{
				"abc0001": {"abc0002", "abc0003", "abc0004"},
			},
		},
		"task_with_multiple_blockers": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", StableID: "abc0001"},
				{ID: "2", Title: "Task 2", StableID: "abc0002"},
				{ID: "3", Title: "Task 3", StableID: "abc0003", BlockedBy: []string{"abc0001", "abc0002"}},
			},
			wantByStableID: map[string]string{
				"abc0001": "1",
				"abc0002": "2",
				"abc0003": "3",
			},
			wantDependents: map[string][]string{
				"abc0001": {"abc0003"},
				"abc0002": {"abc0003"},
			},
		},
		"legacy_task_without_stable_id": {
			tasks: []Task{
				{ID: "1", Title: "Legacy Task"},
				{ID: "2", Title: "Task 2", StableID: "abc0002"},
			},
			wantByStableID: map[string]string{"abc0002": "2"},
			wantDependents: map[string][]string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			idx := BuildDependencyIndex(tc.tasks)
			if idx == nil {
				t.Fatal("BuildDependencyIndex returned nil")
			}

			// Verify byStableID mappings
			for stableID, wantHierarchicalID := range tc.wantByStableID {
				task := idx.GetTask(stableID)
				if task == nil {
					t.Errorf("GetTask(%q) returned nil, expected task with ID %q", stableID, wantHierarchicalID)
					continue
				}
				if task.ID != wantHierarchicalID {
					t.Errorf("GetTask(%q).ID = %q, want %q", stableID, task.ID, wantHierarchicalID)
				}
			}

			// Verify dependents
			for stableID, wantDependents := range tc.wantDependents {
				gotDependents := idx.GetDependents(stableID)
				if len(gotDependents) != len(wantDependents) {
					t.Errorf("GetDependents(%q) = %v (len %d), want %v (len %d)",
						stableID, gotDependents, len(gotDependents), wantDependents, len(wantDependents))
					continue
				}
				// Check all expected dependents are present (order may vary)
				gotSet := make(map[string]bool)
				for _, d := range gotDependents {
					gotSet[d] = true
				}
				for _, want := range wantDependents {
					if !gotSet[want] {
						t.Errorf("GetDependents(%q) missing expected dependent %q", stableID, want)
					}
				}
			}
		})
	}
}

func TestDependencyIndex_GetTask(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Task 1", StableID: "abc0001"},
		{ID: "2", Title: "Task 2", StableID: "abc0002"},
	}
	idx := BuildDependencyIndex(tasks)

	tests := map[string]struct {
		stableID string
		wantNil  bool
		wantID   string
	}{
		"existing_stable_id": {
			stableID: "abc0001",
			wantNil:  false,
			wantID:   "1",
		},
		"non_existing_stable_id": {
			stableID: "xyz9999",
			wantNil:  true,
		},
		"empty_stable_id": {
			stableID: "",
			wantNil:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			task := idx.GetTask(tc.stableID)
			if tc.wantNil {
				if task != nil {
					t.Errorf("GetTask(%q) = %+v, want nil", tc.stableID, task)
				}
			} else {
				if task == nil {
					t.Fatalf("GetTask(%q) = nil, want task", tc.stableID)
				}
				if task.ID != tc.wantID {
					t.Errorf("GetTask(%q).ID = %q, want %q", tc.stableID, task.ID, tc.wantID)
				}
			}
		})
	}
}

func TestDependencyIndex_GetTaskByHierarchicalID(t *testing.T) {
	tasks := []Task{
		{
			ID:       "1",
			Title:    "Task 1",
			StableID: "abc0001",
			Children: []Task{
				{ID: "1.1", Title: "Subtask 1.1", StableID: "abc0011"},
			},
		},
		{ID: "2", Title: "Task 2", StableID: "abc0002"},
	}
	idx := BuildDependencyIndex(tasks)

	tests := map[string]struct {
		hierarchicalID string
		wantNil        bool
		wantTitle      string
	}{
		"top_level_task": {
			hierarchicalID: "1",
			wantNil:        false,
			wantTitle:      "Task 1",
		},
		"nested_task": {
			hierarchicalID: "1.1",
			wantNil:        false,
			wantTitle:      "Subtask 1.1",
		},
		"non_existing_id": {
			hierarchicalID: "99",
			wantNil:        true,
		},
		"empty_id": {
			hierarchicalID: "",
			wantNil:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			task := idx.GetTaskByHierarchicalID(tc.hierarchicalID)
			if tc.wantNil {
				if task != nil {
					t.Errorf("GetTaskByHierarchicalID(%q) = %+v, want nil", tc.hierarchicalID, task)
				}
			} else {
				if task == nil {
					t.Fatalf("GetTaskByHierarchicalID(%q) = nil, want task", tc.hierarchicalID)
				}
				if task.Title != tc.wantTitle {
					t.Errorf("GetTaskByHierarchicalID(%q).Title = %q, want %q",
						tc.hierarchicalID, task.Title, tc.wantTitle)
				}
			}
		})
	}
}

func TestDependencyIndex_GetDependents(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Setup", StableID: "abc0001"},
		{ID: "2", Title: "Task A", StableID: "abc0002", BlockedBy: []string{"abc0001"}},
		{ID: "3", Title: "Task B", StableID: "abc0003", BlockedBy: []string{"abc0001"}},
		{ID: "4", Title: "Standalone", StableID: "abc0004"},
	}
	idx := BuildDependencyIndex(tasks)

	tests := map[string]struct {
		stableID       string
		wantDependents []string
	}{
		"task_with_dependents": {
			stableID:       "abc0001",
			wantDependents: []string{"abc0002", "abc0003"},
		},
		"task_without_dependents": {
			stableID:       "abc0004",
			wantDependents: []string{},
		},
		"non_existing_task": {
			stableID:       "xyz9999",
			wantDependents: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := idx.GetDependents(tc.stableID)
			if len(got) != len(tc.wantDependents) {
				t.Errorf("GetDependents(%q) = %v (len %d), want len %d",
					tc.stableID, got, len(got), len(tc.wantDependents))
				return
			}
			gotSet := make(map[string]bool)
			for _, d := range got {
				gotSet[d] = true
			}
			for _, want := range tc.wantDependents {
				if !gotSet[want] {
					t.Errorf("GetDependents(%q) missing %q", tc.stableID, want)
				}
			}
		})
	}
}

func TestDependencyIndex_IsReady(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Setup", StableID: "abc0001", Status: Completed},
		{ID: "2", Title: "In Progress", StableID: "abc0002", Status: InProgress},
		{ID: "3", Title: "Pending", StableID: "abc0003", Status: Pending},
		{ID: "4", Title: "Blocked by completed", StableID: "abc0004", BlockedBy: []string{"abc0001"}},
		{ID: "5", Title: "Blocked by in-progress", StableID: "abc0005", BlockedBy: []string{"abc0002"}},
		{ID: "6", Title: "Blocked by pending", StableID: "abc0006", BlockedBy: []string{"abc0003"}},
		{ID: "7", Title: "Blocked by multiple all completed", StableID: "abc0007", BlockedBy: []string{"abc0001"}},
		{ID: "8", Title: "Blocked by multiple mixed", StableID: "abc0008", BlockedBy: []string{"abc0001", "abc0003"}},
	}
	idx := BuildDependencyIndex(tasks)

	tests := map[string]struct {
		stableID  string
		wantReady bool
	}{
		"task_without_blockers": {
			stableID:  "abc0001",
			wantReady: true,
		},
		"blocked_by_completed_task": {
			stableID:  "abc0004",
			wantReady: true,
		},
		"blocked_by_in_progress_task": {
			stableID:  "abc0005",
			wantReady: false,
		},
		"blocked_by_pending_task": {
			stableID:  "abc0006",
			wantReady: false,
		},
		"blocked_by_multiple_all_completed": {
			stableID:  "abc0007",
			wantReady: true,
		},
		"blocked_by_multiple_mixed_status": {
			stableID:  "abc0008",
			wantReady: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			task := idx.GetTask(tc.stableID)
			if task == nil {
				t.Fatalf("GetTask(%q) returned nil", tc.stableID)
			}
			got := idx.IsReady(task)
			if got != tc.wantReady {
				t.Errorf("IsReady(task %q) = %v, want %v", tc.stableID, got, tc.wantReady)
			}
		})
	}
}

func TestDependencyIndex_IsBlocked(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Completed", StableID: "abc0001", Status: Completed},
		{ID: "2", Title: "Pending", StableID: "abc0002", Status: Pending},
		{ID: "3", Title: "Blocked by completed", StableID: "abc0003", BlockedBy: []string{"abc0001"}},
		{ID: "4", Title: "Blocked by pending", StableID: "abc0004", BlockedBy: []string{"abc0002"}},
		{ID: "5", Title: "No blockers", StableID: "abc0005"},
	}
	idx := BuildDependencyIndex(tasks)

	tests := map[string]struct {
		stableID    string
		wantBlocked bool
	}{
		"task_without_blockers": {
			stableID:    "abc0005",
			wantBlocked: false,
		},
		"blocked_by_completed": {
			stableID:    "abc0003",
			wantBlocked: false,
		},
		"blocked_by_pending": {
			stableID:    "abc0004",
			wantBlocked: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			task := idx.GetTask(tc.stableID)
			if task == nil {
				t.Fatalf("GetTask(%q) returned nil", tc.stableID)
			}
			got := idx.IsBlocked(task)
			if got != tc.wantBlocked {
				t.Errorf("IsBlocked(task %q) = %v, want %v", tc.stableID, got, tc.wantBlocked)
			}
		})
	}
}

func TestDependencyIndex_TranslateToHierarchical(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Task 1", StableID: "abc0001"},
		{ID: "2", Title: "Task 2", StableID: "abc0002"},
		{
			ID:       "3",
			Title:    "Phase",
			StableID: "abc0003",
			Children: []Task{
				{ID: "3.1", Title: "Subtask", StableID: "abc0031"},
			},
		},
	}
	idx := BuildDependencyIndex(tasks)

	tests := map[string]struct {
		stableIDs  []string
		wantResult []string
	}{
		"empty_list": {
			stableIDs:  []string{},
			wantResult: []string{},
		},
		"single_id": {
			stableIDs:  []string{"abc0001"},
			wantResult: []string{"1"},
		},
		"multiple_ids": {
			stableIDs:  []string{"abc0001", "abc0002"},
			wantResult: []string{"1", "2"},
		},
		"nested_task_id": {
			stableIDs:  []string{"abc0031"},
			wantResult: []string{"3.1"},
		},
		"unknown_id_excluded": {
			stableIDs:  []string{"abc0001", "xyz9999"},
			wantResult: []string{"1"},
		},
		"all_unknown_ids": {
			stableIDs:  []string{"xyz9999"},
			wantResult: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := idx.TranslateToHierarchical(tc.stableIDs)
			if len(got) != len(tc.wantResult) {
				t.Errorf("TranslateToHierarchical(%v) = %v (len %d), want %v (len %d)",
					tc.stableIDs, got, len(got), tc.wantResult, len(tc.wantResult))
				return
			}
			for i, want := range tc.wantResult {
				if got[i] != want {
					t.Errorf("TranslateToHierarchical(%v)[%d] = %q, want %q",
						tc.stableIDs, i, got[i], want)
				}
			}
		})
	}
}

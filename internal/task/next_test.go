package task

import (
	"os"
	"strings"
	"testing"
)

func TestFindNextIncompleteTask(t *testing.T) {
	tests := map[string]struct {
		tasks    []Task
		wantNil  bool
		wantID   string
		wantDesc string
	}{
		"flat list with first task incomplete": {
			tasks: []Task{
				{ID: "1", Title: "First task", Status: Pending},
				{ID: "2", Title: "Second task", Status: Completed},
				{ID: "3", Title: "Third task", Status: Pending},
			},
			wantNil:  false,
			wantID:   "1",
			wantDesc: "First task",
		},
		"flat list with middle task incomplete": {
			tasks: []Task{
				{ID: "1", Title: "First task", Status: Completed},
				{ID: "2", Title: "Second task", Status: InProgress},
				{ID: "3", Title: "Third task", Status: Pending},
			},
			wantNil:  false,
			wantID:   "2",
			wantDesc: "Second task",
		},
		"flat list all complete": {
			tasks: []Task{
				{ID: "1", Title: "First task", Status: Completed},
				{ID: "2", Title: "Second task", Status: Completed},
				{ID: "3", Title: "Third task", Status: Completed},
			},
			wantNil: true,
		},
		"nested hierarchy with incomplete parent": {
			tasks: []Task{
				{
					ID:     "1",
					Title:  "Parent task",
					Status: Pending,
					Children: []Task{
						{ID: "1.1", Title: "Child 1", Status: Completed},
						{ID: "1.2", Title: "Child 2", Status: Pending},
					},
				},
				{ID: "2", Title: "Second task", Status: Completed},
			},
			wantNil:  false,
			wantID:   "1",
			wantDesc: "Parent task",
		},
		"nested hierarchy with complete parent but incomplete child": {
			tasks: []Task{
				{
					ID:     "1",
					Title:  "Parent task",
					Status: Completed,
					Children: []Task{
						{ID: "1.1", Title: "Child 1", Status: Completed},
						{ID: "1.2", Title: "Child 2", Status: Pending},
					},
				},
				{ID: "2", Title: "Second task", Status: Completed},
			},
			wantNil:  false,
			wantID:   "1",
			wantDesc: "Parent task",
		},
		"deeply nested with incomplete grandchild": {
			tasks: []Task{
				{
					ID:     "1",
					Title:  "Parent task",
					Status: Completed,
					Children: []Task{
						{
							ID:     "1.1",
							Title:  "Child 1",
							Status: Completed,
							Children: []Task{
								{ID: "1.1.1", Title: "Grandchild 1", Status: Completed},
								{ID: "1.1.2", Title: "Grandchild 2", Status: InProgress},
							},
						},
						{ID: "1.2", Title: "Child 2", Status: Completed},
					},
				},
				{ID: "2", Title: "Second task", Status: Completed},
			},
			wantNil:  false,
			wantID:   "1",
			wantDesc: "Parent task",
		},
		"in-progress task treated as incomplete": {
			tasks: []Task{
				{ID: "1", Title: "First task", Status: Completed},
				{ID: "2", Title: "Second task", Status: InProgress},
				{ID: "3", Title: "Third task", Status: Pending},
			},
			wantNil:  false,
			wantID:   "2",
			wantDesc: "Second task",
		},
		"mixed completion states": {
			tasks: []Task{
				{
					ID:     "1",
					Title:  "First parent",
					Status: Completed,
					Children: []Task{
						{ID: "1.1", Title: "Child 1", Status: Completed},
						{ID: "1.2", Title: "Child 2", Status: Completed},
					},
				},
				{
					ID:     "2",
					Title:  "Second parent",
					Status: InProgress,
					Children: []Task{
						{ID: "2.1", Title: "Child 1", Status: Completed},
						{ID: "2.2", Title: "Child 2", Status: Pending},
						{ID: "2.3", Title: "Child 3", Status: InProgress},
					},
				},
				{ID: "3", Title: "Third task", Status: Pending},
			},
			wantNil:  false,
			wantID:   "2",
			wantDesc: "Second parent",
		},
		"empty task list": {
			tasks:   []Task{},
			wantNil: true,
		},
		"first complete parent second incomplete": {
			tasks: []Task{
				{
					ID:     "1",
					Title:  "First parent",
					Status: Completed,
					Children: []Task{
						{ID: "1.1", Title: "Child 1", Status: Completed},
						{ID: "1.2", Title: "Child 2", Status: Completed},
					},
				},
				{
					ID:     "2",
					Title:  "Second parent",
					Status: Pending,
					Children: []Task{
						{ID: "2.1", Title: "Child 1", Status: Pending},
					},
				},
			},
			wantNil:  false,
			wantID:   "2",
			wantDesc: "Second parent",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := FindNextIncompleteTask(tc.tasks)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got task with ID %s", result.ID)
				}
				return
			}

			if result == nil {
				t.Errorf("expected task with ID %s, got nil", tc.wantID)
				return
			}

			if result.ID != tc.wantID {
				t.Errorf("expected task ID %s, got %s", tc.wantID, result.ID)
			}

			if result.Title != tc.wantDesc {
				t.Errorf("expected task title %q, got %q", tc.wantDesc, result.Title)
			}
		})
	}
}

func TestFilterIncompleteChildren(t *testing.T) {
	tests := map[string]struct {
		children []Task
		wantIDs  []string
	}{
		"all children incomplete": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Pending},
				{ID: "1.2", Title: "Child 2", Status: InProgress},
				{ID: "1.3", Title: "Child 3", Status: Pending},
			},
			wantIDs: []string{"1.1", "1.2", "1.3"},
		},
		"some children complete": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Completed},
				{ID: "1.2", Title: "Child 2", Status: InProgress},
				{ID: "1.3", Title: "Child 3", Status: Pending},
			},
			wantIDs: []string{"1.2", "1.3"},
		},
		"all children complete": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Completed},
				{ID: "1.2", Title: "Child 2", Status: Completed},
			},
			wantIDs: []string{},
		},
		"children with grandchildren": {
			children: []Task{
				{
					ID:     "1.1",
					Title:  "Child 1",
					Status: Completed,
					Children: []Task{
						{ID: "1.1.1", Title: "Grandchild 1", Status: Pending},
					},
				},
				{ID: "1.2", Title: "Child 2", Status: Completed},
			},
			wantIDs: []string{"1.1"}, // Child 1 has incomplete grandchildren
		},
		"empty children list": {
			children: []Task{},
			wantIDs:  []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := filterIncompleteChildren(tc.children)

			if len(result) != len(tc.wantIDs) {
				t.Errorf("expected %d incomplete children, got %d", len(tc.wantIDs), len(result))
				return
			}

			for i, task := range result {
				if task.ID != tc.wantIDs[i] {
					t.Errorf("expected task ID %s at position %d, got %s", tc.wantIDs[i], i, task.ID)
				}
			}
		})
	}
}

func TestHasIncompleteWork(t *testing.T) {
	tests := map[string]struct {
		task *Task
		want bool
	}{
		"pending task": {
			task: &Task{ID: "1", Title: "Task", Status: Pending},
			want: true,
		},
		"in-progress task": {
			task: &Task{ID: "1", Title: "Task", Status: InProgress},
			want: true,
		},
		"completed task with no children": {
			task: &Task{ID: "1", Title: "Task", Status: Completed},
			want: false,
		},
		"completed task with incomplete child": {
			task: &Task{
				ID:     "1",
				Title:  "Task",
				Status: Completed,
				Children: []Task{
					{ID: "1.1", Title: "Child", Status: Pending},
				},
			},
			want: true,
		},
		"completed task with all children complete": {
			task: &Task{
				ID:     "1",
				Title:  "Task",
				Status: Completed,
				Children: []Task{
					{ID: "1.1", Title: "Child 1", Status: Completed},
					{ID: "1.2", Title: "Child 2", Status: Completed},
				},
			},
			want: false,
		},
		"completed task with incomplete grandchild": {
			task: &Task{
				ID:     "1",
				Title:  "Task",
				Status: Completed,
				Children: []Task{
					{
						ID:     "1.1",
						Title:  "Child",
						Status: Completed,
						Children: []Task{
							{ID: "1.1.1", Title: "Grandchild", Status: InProgress},
						},
					},
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := hasIncompleteWork(tc.task)
			if result != tc.want {
				t.Errorf("expected hasIncompleteWork to return %v, got %v", tc.want, result)
			}
		})
	}
}

func TestDepthProtection(t *testing.T) {
	// Create a task with depth exceeding the max depth limit
	task := &Task{
		ID:     "1",
		Title:  "Root",
		Status: Pending,
	}

	// This test ensures the function doesn't crash with deep nesting
	// The max depth is set to 100 in the implementation
	current := task
	for range 150 {
		child := Task{
			ID:     current.ID + ".1",
			Title:  "Child",
			Status: Pending,
		}
		current.Children = []Task{child}
		current = &current.Children[0]
	}

	// Should still return true for the root task (it's pending)
	// but won't traverse beyond depth 100
	result := hasIncompleteWork(task)
	if !result {
		t.Errorf("expected hasIncompleteWork to return true for pending root task")
	}
}

func TestNextTaskWithIncompleteChildren(t *testing.T) {
	tasks := []Task{
		{
			ID:     "1",
			Title:  "Parent",
			Status: InProgress,
			Children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Completed},
				{ID: "1.2", Title: "Child 2", Status: Pending},
				{ID: "1.3", Title: "Child 3", Status: InProgress},
				{ID: "1.4", Title: "Child 4", Status: Completed},
			},
		},
	}

	result := FindNextIncompleteTask(tasks)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ID != "1" {
		t.Errorf("expected parent task ID 1, got %s", result.ID)
	}

	// Should only include incomplete children
	if len(result.IncompleteChildren) != 2 {
		t.Errorf("expected 2 incomplete children, got %d", len(result.IncompleteChildren))
	}

	expectedIDs := map[string]bool{"1.2": true, "1.3": true}
	for _, child := range result.IncompleteChildren {
		if !expectedIDs[child.ID] {
			t.Errorf("unexpected child ID %s in incomplete children", child.ID)
		}
		delete(expectedIDs, child.ID)
	}

	if len(expectedIDs) > 0 {
		t.Errorf("missing expected children: %v", expectedIDs)
	}
}

func TestHasReadyTaskInStream(t *testing.T) {
	tests := map[string]struct {
		tasks       []Task
		stream      int
		index       *DependencyIndex
		wantReady   bool
		description string
	}{
		"has ready task in stream": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 2},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 2}}),
			wantReady:   true,
			description: "pending task with no owner and not blocked",
		},
		"task has owner - not ready": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "agent-1", Stream: 2},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Pending, Owner: "agent-1", Stream: 2}}),
			wantReady:   false,
			description: "task with owner is not ready",
		},
		"task is blocked - not ready": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 1, StableID: "abc123"},
				{ID: "2", Title: "Task 2", Status: Pending, Owner: "", Stream: 2, StableID: "def456", BlockedBy: []string{"abc123"}},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 1, StableID: "abc123"}, {ID: "2", Title: "Task 2", Status: Pending, Owner: "", Stream: 2, StableID: "def456", BlockedBy: []string{"abc123"}}}),
			wantReady:   false,
			description: "blocked task is not ready",
		},
		"task is in-progress - not ready": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: InProgress, Owner: "agent-1", Stream: 2},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: InProgress, Owner: "agent-1", Stream: 2}}),
			wantReady:   false,
			description: "in-progress task is not ready",
		},
		"task is completed - not ready": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Completed, Owner: "", Stream: 2},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Completed, Owner: "", Stream: 2}}),
			wantReady:   false,
			description: "completed task is not ready",
		},
		"no tasks in stream": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 1},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 1}}),
			wantReady:   false,
			description: "no tasks in requested stream",
		},
		"multiple tasks - one ready": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "agent-1", Stream: 2},
				{ID: "2", Title: "Task 2", Status: Pending, Owner: "", Stream: 2},
				{ID: "3", Title: "Task 3", Status: InProgress, Owner: "agent-2", Stream: 2},
			},
			stream:      2,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Pending, Owner: "agent-1", Stream: 2}, {ID: "2", Title: "Task 2", Status: Pending, Owner: "", Stream: 2}, {ID: "3", Title: "Task 3", Status: InProgress, Owner: "agent-2", Stream: 2}}),
			wantReady:   true,
			description: "at least one ready task exists",
		},
		"nil index returns false": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 2},
			},
			stream:      2,
			index:       nil,
			wantReady:   false,
			description: "nil index is handled safely",
		},
		"empty task list": {
			tasks:       []Task{},
			stream:      2,
			index:       mustBuildIndex([]Task{}),
			wantReady:   false,
			description: "empty task list returns false",
		},
		"task with default stream (1)": {
			tasks: []Task{
				{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 0}, // Stream 0 defaults to 1
			},
			stream:      1,
			index:       mustBuildIndex([]Task{{ID: "1", Title: "Task 1", Status: Pending, Owner: "", Stream: 0}}),
			wantReady:   true,
			description: "task with stream 0 defaults to stream 1",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := hasReadyTaskInStream(tc.tasks, tc.stream, tc.index)
			if got != tc.wantReady {
				t.Errorf("%s: hasReadyTaskInStream() = %v, want %v", tc.description, got, tc.wantReady)
			}
		})
	}
}

// mustBuildIndex is a test helper that builds a dependency index or panics
func mustBuildIndex(tasks []Task) *DependencyIndex {
	index := BuildDependencyIndex(tasks)
	return index
}

func TestFindNextPhaseTasksForStream(t *testing.T) {
	tests := map[string]struct {
		content     string
		stream      int
		wantNil     bool
		wantPhase   string
		wantTaskIDs []string
		wantErr     bool
	}{
		"single phase with stream exists": {
			content: `## Phase A
- [ ] 1. Task A1
  - Stream: 1
- [ ] 2. Task A2
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"2"},
		},
		"first phase lacks stream - skip to second": {
			content: `## Phase A
- [ ] 1. Task A1
  - Stream: 1

## Phase B
- [ ] 2. Task B1
  - Stream: 1
- [ ] 3. Task B2
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"3"},
		},
		"all phases lack stream": {
			content: `## Phase A
- [ ] 1. Task A1
  - Stream: 1

## Phase B
- [ ] 2. Task B1
  - Stream: 1
`,
			stream:  2,
			wantNil: true,
		},
		"stream tasks all blocked": {
			content: `## Phase A
- [ ] 1. Task A1 <!-- id:abc1234 -->
  - Stream: 1

## Phase B
- [ ] 2. Task B1
  - Stream: 2
  - Blocked-by: abc1234 (Task A1)
`,
			stream:  2,
			wantNil: true,
		},
		"no phases in document": {
			content: `- [ ] 1. Task 1
  - Stream: 2
- [ ] 2. Task 2
  - Stream: 2
`,
			stream:  2,
			wantNil: true,
		},
		"tasks before first phase excluded": {
			content: `- [ ] 1. Orphan task
  - Stream: 2

## Phase A
- [ ] 2. Task A1
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"2"},
		},
		"empty phases skipped": {
			content: `## Phase A

## Phase B
- [ ] 1. Task B1
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"1"},
		},
		"mixed ready and blocked in stream - returns all": {
			content: `## Phase A
- [ ] 1. Task A1 <!-- id:abc1234 -->
  - Stream: 2
- [ ] 2. Task A2
  - Stream: 2
  - Blocked-by: abc1234 (Task A1)
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1", "2"},
		},
		"task with owner not ready - skip phase": {
			content: `## Phase A
- [ ] 1. Task A1
  - Stream: 2
  - Owner: agent-1

## Phase B
- [ ] 2. Task B1
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"2"},
		},
		"in-progress task not ready - skip phase": {
			content: `## Phase A
- [-] 1. Task A1
  - Stream: 2

## Phase B
- [ ] 2. Task B1
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"2"},
		},
		"completed task not ready - skip phase": {
			content: `## Phase A
- [x] 1. Task A1
  - Stream: 2

## Phase B
- [ ] 2. Task B1
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"2"},
		},
		"task with default stream (1)": {
			content: `## Phase A
- [ ] 1. Task A1
`,
			stream:      1,
			wantNil:     false,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1"},
		},
		"multiple tasks in stream - one ready": {
			content: `## Phase A
- [ ] 1. Task A1
  - Stream: 2
  - Owner: agent-1
- [ ] 2. Task A2
  - Stream: 2
- [-] 3. Task A3
  - Stream: 2
`,
			stream:      2,
			wantNil:     false,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1", "2", "3"},
		},
		"invalid stream returns error": {
			content: `## Phase A
- [ ] 1. Task A1
`,
			stream:  0,
			wantErr: true,
		},
		"negative stream returns error": {
			content: `## Phase A
- [ ] 1. Task A1
`,
			stream:  -1,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary file
			tmpfile, err := os.CreateTemp("", "test-*.md")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tc.content)); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatalf("failed to close temp file: %v", err)
			}

			// Call the function
			result, err := FindNextPhaseTasksForStream(tmpfile.Name(), tc.stream)

			// Check error expectation
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check nil expectation
			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got phase %q with %d tasks", result.PhaseName, len(result.Tasks))
				}
				return
			}

			// Check result
			if result == nil {
				t.Fatalf("expected result, got nil")
			}

			if result.PhaseName != tc.wantPhase {
				t.Errorf("phase name: got %q, want %q", result.PhaseName, tc.wantPhase)
			}

			if len(result.Tasks) != len(tc.wantTaskIDs) {
				t.Errorf("task count: got %d, want %d", len(result.Tasks), len(tc.wantTaskIDs))
			}

			// Check task IDs
			gotIDs := make([]string, len(result.Tasks))
			for i, task := range result.Tasks {
				gotIDs[i] = task.ID
			}

			for i, wantID := range tc.wantTaskIDs {
				if i >= len(gotIDs) {
					t.Errorf("missing task ID %q at index %d", wantID, i)
					continue
				}
				if gotIDs[i] != wantID {
					t.Errorf("task ID at index %d: got %q, want %q", i, gotIDs[i], wantID)
				}
			}
		})
	}
}

func TestHasReadyTaskInStream_NestedTasks(t *testing.T) {
	// Regression test for T-170: hasReadyTaskInStream should find ready tasks
	// nested inside parent tasks with a different stream.
	tests := map[string]struct {
		tasks     []Task
		stream    int
		wantReady bool
	}{
		"ready_nested_task_in_different_stream_parent": {
			tasks: []Task{
				{
					ID:       "1",
					Title:    "Parent stream 1",
					Status:   Pending,
					Stream:   1,
					StableID: "abc0001",
					Children: []Task{
						{ID: "1.1", Title: "Child stream 2", Status: Pending, Stream: 2, StableID: "abc0011"},
					},
				},
			},
			stream:    2,
			wantReady: true,
		},
		"no_ready_nested_tasks_all_owned": {
			tasks: []Task{
				{
					ID:       "1",
					Title:    "Parent stream 1",
					Status:   Pending,
					Stream:   1,
					StableID: "abc0001",
					Children: []Task{
						{ID: "1.1", Title: "Child stream 2", Status: Pending, Stream: 2, StableID: "abc0011", Owner: "agent-1"},
					},
				},
			},
			stream:    2,
			wantReady: false,
		},
		"deeply_nested_ready_task": {
			tasks: []Task{
				{
					ID:       "1",
					Title:    "Parent stream 1",
					Status:   Pending,
					Stream:   1,
					StableID: "abc0001",
					Children: []Task{
						{
							ID:       "1.1",
							Title:    "Child stream 1",
							Status:   Pending,
							Stream:   1,
							StableID: "abc0011",
							Children: []Task{
								{ID: "1.1.1", Title: "Grandchild stream 2", Status: Pending, Stream: 2, StableID: "abc0111"},
							},
						},
					},
				},
			},
			stream:    2,
			wantReady: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			index := BuildDependencyIndex(tc.tasks)
			got := hasReadyTaskInStream(tc.tasks, tc.stream, index)
			if got != tc.wantReady {
				t.Errorf("hasReadyTaskInStream() = %v, want %v", got, tc.wantReady)
			}
		})
	}
}

func TestFindNextPhaseTasksForStream_NestedTasks(t *testing.T) {
	// Regression test for T-170: FindNextPhaseTasksForStream should find
	// nested tasks whose effective stream matches, even when their parent
	// is in a different stream.
	tests := map[string]struct {
		content     string
		stream      int
		wantNil     bool
		wantPhase   string
		wantTaskIDs []string
	}{
		"nested_subtask_in_target_stream": {
			content: `## Phase A
- [ ] 1. Parent task
  - Stream: 1
  - [ ] 1.1. Child in stream 2
    - Stream: 2
`,
			stream:      2,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1.1"},
		},
		"skip_phase_without_ready_nested_tasks": {
			content: `## Phase A
- [ ] 1. Parent task
  - Stream: 1

## Phase B
- [ ] 2. Parent task B
  - Stream: 1
  - [ ] 2.1. Child in stream 2
    - Stream: 2
`,
			stream:      2,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"2.1"},
		},
		"deeply_nested_stream_task": {
			content: `## Phase A
- [ ] 1. Top-level stream 1
  - Stream: 1
  - [ ] 1.1. Child stream 1
    - Stream: 1
    - [ ] 1.1.1. Grandchild stream 3
      - Stream: 3
`,
			stream:      3,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1.1.1"},
		},
		"mix_of_top_level_and_nested_stream_tasks": {
			content: `## Phase A
- [ ] 1. Top-level stream 2
  - Stream: 2
- [ ] 2. Parent stream 1
  - Stream: 1
  - [ ] 2.1. Nested stream 2
    - Stream: 2
`,
			stream:      2,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1", "2.1"},
		},
		"nested_completed_stream_task_excluded": {
			content: `## Phase A
- [ ] 1. Parent stream 1
  - Stream: 1
  - [x] 1.1. Completed child stream 2
    - Stream: 2
  - [ ] 1.2. Pending child stream 2
    - Stream: 2
`,
			stream:      2,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"1.2"},
		},
		"all_nested_stream_tasks_owned_returns_nil": {
			content: `## Phase A
- [ ] 1. Parent stream 1
  - Stream: 1
  - [ ] 1.1. Child stream 2
    - Stream: 2
    - Owner: agent-1
`,
			stream:  2,
			wantNil: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test-*.md")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tc.content)); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatalf("failed to close temp file: %v", err)
			}

			result, err := FindNextPhaseTasksForStream(tmpfile.Name(), tc.stream)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got phase %q with %d tasks", result.PhaseName, len(result.Tasks))
				}
				return
			}

			if result == nil {
				t.Fatalf("expected result, got nil")
			}

			if result.PhaseName != tc.wantPhase {
				t.Errorf("phase name: got %q, want %q", result.PhaseName, tc.wantPhase)
			}

			gotIDs := make([]string, len(result.Tasks))
			for i, task := range result.Tasks {
				gotIDs[i] = task.ID
			}

			if len(result.Tasks) != len(tc.wantTaskIDs) {
				t.Fatalf("task count: got %d %v, want %d %v", len(result.Tasks), gotIDs, len(tc.wantTaskIDs), tc.wantTaskIDs)
			}

			for i, wantID := range tc.wantTaskIDs {
				if gotIDs[i] != wantID {
					t.Errorf("task ID at index %d: got %q, want %q", i, gotIDs[i], wantID)
				}
			}
		})
	}
}

func TestFilterToFirstIncompletePath(t *testing.T) {
	tests := map[string]struct {
		children []Task
		wantIDs  []string // IDs of tasks in the filtered path
	}{
		"multiple incomplete children - returns first": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Pending},
				{ID: "1.2", Title: "Child 2", Status: Pending},
				{ID: "1.3", Title: "Child 3", Status: InProgress},
			},
			wantIDs: []string{"1.1"},
		},
		"first child complete, second incomplete": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Completed},
				{ID: "1.2", Title: "Child 2", Status: Pending},
				{ID: "1.3", Title: "Child 3", Status: Pending},
			},
			wantIDs: []string{"1.2"},
		},
		"all children complete": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Completed},
				{ID: "1.2", Title: "Child 2", Status: Completed},
			},
			wantIDs: []string{},
		},
		"empty children list": {
			children: []Task{},
			wantIDs:  []string{},
		},
		"deeply nested - returns single path": {
			children: []Task{
				{
					ID:     "1.1",
					Title:  "Child 1",
					Status: Pending,
					Children: []Task{
						{
							ID:     "1.1.1",
							Title:  "Grandchild 1",
							Status: Pending,
							Children: []Task{
								{ID: "1.1.1.1", Title: "Great-grandchild 1", Status: Pending},
								{ID: "1.1.1.2", Title: "Great-grandchild 2", Status: Pending},
							},
						},
						{ID: "1.1.2", Title: "Grandchild 2", Status: Pending},
					},
				},
				{ID: "1.2", Title: "Child 2", Status: Pending},
			},
			wantIDs: []string{"1.1", "1.1.1", "1.1.1.1"},
		},
		"child with multiple incomplete grandchildren - returns first grandchild": {
			children: []Task{
				{
					ID:     "1.1",
					Title:  "Child 1",
					Status: Completed,
					Children: []Task{
						{ID: "1.1.1", Title: "Grandchild 1", Status: Pending},
						{ID: "1.1.2", Title: "Grandchild 2", Status: Pending},
						{ID: "1.1.3", Title: "Grandchild 3", Status: InProgress},
					},
				},
				{ID: "1.2", Title: "Child 2", Status: Pending},
			},
			wantIDs: []string{"1.1", "1.1.1"},
		},
		"single incomplete child at each level": {
			children: []Task{
				{
					ID:     "1.1",
					Title:  "Child 1",
					Status: Pending,
					Children: []Task{
						{
							ID:     "1.1.1",
							Title:  "Grandchild 1",
							Status: Pending,
						},
					},
				},
			},
			wantIDs: []string{"1.1", "1.1.1"},
		},
		"skip completed children to find incomplete": {
			children: []Task{
				{ID: "1.1", Title: "Child 1", Status: Completed},
				{ID: "1.2", Title: "Child 2", Status: Completed},
				{
					ID:     "1.3",
					Title:  "Child 3",
					Status: Pending,
					Children: []Task{
						{ID: "1.3.1", Title: "Grandchild 1", Status: Pending},
						{ID: "1.3.2", Title: "Grandchild 2", Status: Pending},
					},
				},
			},
			wantIDs: []string{"1.3", "1.3.1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := filterToFirstIncompletePath(tc.children)

			// Collect all IDs in the filtered path
			var gotIDs []string
			var collectIDs func([]Task)
			collectIDs = func(tasks []Task) {
				for _, task := range tasks {
					gotIDs = append(gotIDs, task.ID)
					if len(task.Children) > 0 {
						collectIDs(task.Children)
					}
				}
			}
			collectIDs(result)

			if len(gotIDs) != len(tc.wantIDs) {
				t.Errorf("expected %d tasks in path, got %d\nwant: %v\ngot: %v",
					len(tc.wantIDs), len(gotIDs), tc.wantIDs, gotIDs)
				return
			}

			for i, wantID := range tc.wantIDs {
				if gotIDs[i] != wantID {
					t.Errorf("at position %d: expected ID %s, got %s", i, wantID, gotIDs[i])
				}
			}
		})
	}
}

func TestFilterToFirstIncompletePathIntegration(t *testing.T) {
	// Test the public FilterToFirstIncompletePath function
	tests := map[string]struct {
		taskCtx *TaskWithContext
		wantIDs []string
	}{
		"nil task context": {
			taskCtx: nil,
			wantIDs: nil,
		},
		"task with multiple incomplete children": {
			taskCtx: &TaskWithContext{
				Task: &Task{ID: "1", Title: "Parent", Status: Pending},
				IncompleteChildren: []Task{
					{ID: "1.1", Title: "Child 1", Status: Pending},
					{ID: "1.2", Title: "Child 2", Status: InProgress},
				},
			},
			wantIDs: []string{"1.1"},
		},
		"task with nested incomplete children": {
			taskCtx: &TaskWithContext{
				Task: &Task{ID: "1", Title: "Parent", Status: Pending},
				IncompleteChildren: []Task{
					{
						ID:     "1.1",
						Title:  "Child 1",
						Status: Pending,
						Children: []Task{
							{ID: "1.1.1", Title: "Grandchild 1", Status: Pending},
							{ID: "1.1.2", Title: "Grandchild 2", Status: Pending},
						},
					},
					{ID: "1.2", Title: "Child 2", Status: Pending},
				},
			},
			wantIDs: []string{"1.1", "1.1.1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			FilterToFirstIncompletePath(tc.taskCtx)

			if tc.taskCtx == nil {
				return
			}

			// Collect all IDs in the filtered path
			var gotIDs []string
			var collectIDs func([]Task)
			collectIDs = func(tasks []Task) {
				for _, task := range tasks {
					gotIDs = append(gotIDs, task.ID)
					if len(task.Children) > 0 {
						collectIDs(task.Children)
					}
				}
			}
			collectIDs(tc.taskCtx.IncompleteChildren)

			if len(gotIDs) != len(tc.wantIDs) {
				t.Errorf("expected %d tasks in path, got %d\nwant: %v\ngot: %v",
					len(tc.wantIDs), len(gotIDs), tc.wantIDs, gotIDs)
				return
			}

			for i, wantID := range tc.wantIDs {
				if gotIDs[i] != wantID {
					t.Errorf("at position %d: expected ID %s, got %s", i, wantID, gotIDs[i])
				}
			}
		})
	}
}

// TestFindNextPhaseTasksCRLF verifies that FindNextPhaseTasks works correctly
// with CRLF line endings. This is a regression test for T-488.
func TestFindNextPhaseTasksCRLF(t *testing.T) {
	tests := map[string]struct {
		content       string
		wantPhaseName string
		wantTaskCount int
	}{
		"crlf_returns_first_pending_phase": {
			content:       "# Project\r\n\r\n## Planning\r\n\r\n- [ ] 1. Define requirements\r\n- [ ] 2. Create design\r\n\r\n## Implementation\r\n\r\n- [ ] 3. Write code\r\n",
			wantPhaseName: "Planning",
			wantTaskCount: 2,
		},
		"crlf_skips_completed_phase": {
			content:       "# Project\r\n\r\n## Planning\r\n\r\n- [x] 1. Done\r\n\r\n## Implementation\r\n\r\n- [ ] 2. Write code\r\n- [ ] 3. Write tests\r\n",
			wantPhaseName: "Implementation",
			wantTaskCount: 2,
		},
		"crlf_no_phases_returns_all_pending": {
			content:       "# Project\r\n\r\n- [ ] 1. Task one\r\n- [ ] 2. Task two\r\n",
			wantPhaseName: "",
			wantTaskCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Write CRLF content to a temp file
			tmpFile, err := os.CreateTemp("", "crlf-phase-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			tmpFile.Close()

			result, err := FindNextPhaseTasks(tmpFile.Name())
			if err != nil {
				t.Fatalf("FindNextPhaseTasks() error = %v", err)
			}

			if tc.wantTaskCount == 0 {
				if result != nil {
					t.Errorf("FindNextPhaseTasks() with CRLF = non-nil, want nil")
				}
				return
			}

			if result == nil {
				t.Fatalf("FindNextPhaseTasks() with CRLF = nil, want phase %q with %d tasks",
					tc.wantPhaseName, tc.wantTaskCount)
			}

			if result.PhaseName != tc.wantPhaseName {
				t.Errorf("FindNextPhaseTasks() with CRLF phase = %q, want %q",
					result.PhaseName, tc.wantPhaseName)
			}

			if len(result.Tasks) != tc.wantTaskCount {
				t.Errorf("FindNextPhaseTasks() with CRLF returned %d tasks, want %d",
					len(result.Tasks), tc.wantTaskCount)
			}
		})
	}
}

// TestExtractPhasesWithTaskRangesCRLF verifies that extractPhasesWithTaskRanges
// works correctly with CRLF line endings. This is a regression test for T-488.
func TestExtractPhasesWithTaskRangesCRLF(t *testing.T) {
	content := "# Project\r\n\r\n## Planning\r\n\r\n- [ ] 1. Define requirements\r\n\r\n## Implementation\r\n\r\n- [ ] 2. Write code\r\n- [ ] 3. Write tests\r\n"

	// Parse tasks with ParseMarkdown (which handles CRLF internally)
	taskList, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}

	lines := strings.Split(content, "\n")
	phases := extractPhasesWithTaskRanges(lines, taskList.Tasks)

	if len(phases) != 2 {
		t.Fatalf("extractPhasesWithTaskRanges() with CRLF got %d phases, want 2", len(phases))
	}

	if phases[0].Name != "Planning" {
		t.Errorf("phase[0].Name = %q, want %q", phases[0].Name, "Planning")
	}
	if len(phases[0].Tasks) != 1 {
		t.Errorf("phase[0] has %d tasks, want 1", len(phases[0].Tasks))
	}

	if phases[1].Name != "Implementation" {
		t.Errorf("phase[1].Name = %q, want %q", phases[1].Name, "Implementation")
	}
	if len(phases[1].Tasks) != 2 {
		t.Errorf("phase[1] has %d tasks, want 2", len(phases[1].Tasks))
	}
}

// TestExtractPhasesWithTaskRangesIndentedLines verifies that indented lines
// are not misclassified as phase headers or top-level task references.
// Regression test for T-594.
func TestExtractPhasesWithTaskRangesIndentedLines(t *testing.T) {
	tests := map[string]struct {
		content        string
		wantPhases     int
		wantPhaseNames []string
		wantTaskCounts []int // number of tasks per phase
	}{
		"indented_task_line_not_treated_as_top_level": {
			content:        "# Project\n\n## Phase A\n\n- [ ] 1. Task One\n  - [ ] 1.1. Subtask\n- [ ] 2. Task Two\n\n## Phase B\n\n- [ ] 3. Task Three\n",
			wantPhases:     2,
			wantPhaseNames: []string{"Phase A", "Phase B"},
			wantTaskCounts: []int{2, 1},
		},
		"indented_phase_header_not_treated_as_phase": {
			content:        "# Project\n\n## Real Phase\n\n- [ ] 1. Task One\n  ## Not a phase\n- [ ] 2. Task Two\n",
			wantPhases:     1,
			wantPhaseNames: []string{"Real Phase"},
			wantTaskCounts: []int{2},
		},
		"subtask_not_counted_as_top_level_after_trim": {
			content:        "# Project\n\n## Phase A\n\n- [ ] 1. Task One\n  - [ ] 1.1. Subtask One\n  - [ ] 1.2. Subtask Two\n- [ ] 2. Task Two\n\n## Phase B\n\n- [ ] 3. Task Three\n  - [ ] 3.1. Subtask of Three\n",
			wantPhases:     2,
			wantPhaseNames: []string{"Phase A", "Phase B"},
			wantTaskCounts: []int{2, 1},
		},
		"continuation_lines_with_description": {
			content:        "# Project\n\n## Planning\n\n- [ ] 1. Define requirements\n  This task involves gathering input\n  - [ ] 1.1. Review docs\n- [ ] 2. Write spec\n\n## Implementation\n\n- [ ] 3. Build feature\n",
			wantPhases:     2,
			wantPhaseNames: []string{"Planning", "Implementation"},
			wantTaskCounts: []int{2, 1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			taskList, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error = %v", err)
			}

			lines := strings.Split(tc.content, "\n")
			phases := extractPhasesWithTaskRanges(lines, taskList.Tasks)

			if len(phases) != tc.wantPhases {
				t.Fatalf("got %d phases, want %d; phases: %+v", len(phases), tc.wantPhases, phases)
			}

			for i, wantName := range tc.wantPhaseNames {
				if phases[i].Name != wantName {
					t.Errorf("phase[%d].Name = %q, want %q", i, phases[i].Name, wantName)
				}
			}

			for i, wantCount := range tc.wantTaskCounts {
				if len(phases[i].Tasks) != wantCount {
					taskIDs := make([]string, len(phases[i].Tasks))
					for j, task := range phases[i].Tasks {
						taskIDs[j] = task.ID
					}
					t.Errorf("phase[%d] (%s) has %d tasks %v, want %d",
						i, phases[i].Name, len(phases[i].Tasks), taskIDs, wantCount)
				}
			}
		})
	}
}

// TestExtractPhasesWithTaskRanges_NonSequentialIDs verifies that
// extractPhasesWithTaskRanges correctly associates tasks with phases when
// markdown task IDs are non-sequential (e.g., due to manual editing or deletions).
// Regression test for T-604.
func TestExtractPhasesWithTaskRanges_NonSequentialIDs(t *testing.T) {
	tests := map[string]struct {
		content        string
		wantPhaseCount int
		wantPhases     []struct {
			name      string
			taskCount int
			taskIDs   []string
		}
	}{
		"non-sequential IDs across two phases": {
			content: `## Phase A
- [ ] 10. First task
- [ ] 20. Second task

## Phase B
- [ ] 30. Third task
`,
			wantPhaseCount: 2,
			wantPhases: []struct {
				name      string
				taskCount int
				taskIDs   []string
			}{
				{name: "Phase A", taskCount: 2, taskIDs: []string{"1", "2"}},
				{name: "Phase B", taskCount: 1, taskIDs: []string{"3"}},
			},
		},
		"large gap in IDs": {
			content: `## Phase A
- [ ] 100. Setup
- [ ] 200. Configure

## Phase B
- [ ] 500. Deploy
`,
			wantPhaseCount: 2,
			wantPhases: []struct {
				name      string
				taskCount int
				taskIDs   []string
			}{
				{name: "Phase A", taskCount: 2, taskIDs: []string{"1", "2"}},
				{name: "Phase B", taskCount: 1, taskIDs: []string{"3"}},
			},
		},
		"single non-sequential ID in one phase": {
			content: `## Phase A
- [ ] 5. Only task
`,
			wantPhaseCount: 1,
			wantPhases: []struct {
				name      string
				taskCount int
				taskIDs   []string
			}{
				{name: "Phase A", taskCount: 1, taskIDs: []string{"1"}},
			},
		},
		"tasks before first phase with non-sequential IDs": {
			content: `- [ ] 10. Orphan task

## Phase A
- [ ] 20. Phase task
`,
			wantPhaseCount: 1,
			wantPhases: []struct {
				name      string
				taskCount int
				taskIDs   []string
			}{
				{name: "Phase A", taskCount: 1, taskIDs: []string{"2"}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			taskList, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error = %v", err)
			}

			lines := splitLines(tc.content)
			phases := extractPhasesWithTaskRanges(lines, taskList.Tasks)

			if len(phases) != tc.wantPhaseCount {
				t.Fatalf("got %d phases, want %d", len(phases), tc.wantPhaseCount)
			}

			for i, wantPhase := range tc.wantPhases {
				if phases[i].Name != wantPhase.name {
					t.Errorf("phase[%d].Name = %q, want %q", i, phases[i].Name, wantPhase.name)
				}
				if len(phases[i].Tasks) != wantPhase.taskCount {
					t.Errorf("phase[%d] has %d tasks, want %d", i, len(phases[i].Tasks), wantPhase.taskCount)
				}
				for j, wantID := range wantPhase.taskIDs {
					if j < len(phases[i].Tasks) && phases[i].Tasks[j].ID != wantID {
						t.Errorf("phase[%d].Tasks[%d].ID = %q, want %q", i, j, phases[i].Tasks[j].ID, wantID)
					}
				}
			}
		})
	}
}

// TestFindNextPhaseTasks_NonSequentialIDs verifies that FindNextPhaseTasks
// correctly returns pending tasks when markdown IDs are non-sequential.
// Regression test for T-604.
func TestFindNextPhaseTasks_NonSequentialIDs(t *testing.T) {
	tests := map[string]struct {
		content       string
		wantNil       bool
		wantPhase     string
		wantTaskCount int
	}{
		"non-sequential IDs returns first pending phase": {
			content: `## Phase A
- [ ] 10. First task
- [ ] 20. Second task

## Phase B
- [ ] 30. Third task
`,
			wantPhase:     "Phase A",
			wantTaskCount: 2,
		},
		"completed non-sequential phase skipped": {
			content: `## Phase A
- [x] 10. Done task

## Phase B
- [ ] 20. Pending task
`,
			wantPhase:     "Phase B",
			wantTaskCount: 1,
		},
		"all non-sequential phases complete": {
			content: `## Phase A
- [x] 10. Done

## Phase B
- [x] 20. Also done
`,
			wantNil: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "nonseq-phase-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			tmpFile.Close()

			result, err := FindNextPhaseTasks(tmpFile.Name())
			if err != nil {
				t.Fatalf("FindNextPhaseTasks() error = %v", err)
			}

			if tc.wantNil {
				if result != nil {
					t.Errorf("FindNextPhaseTasks() = non-nil, want nil")
				}
				return
			}

			if result == nil {
				t.Fatalf("FindNextPhaseTasks() = nil, want phase %q with %d tasks",
					tc.wantPhase, tc.wantTaskCount)
			}

			if result.PhaseName != tc.wantPhase {
				t.Errorf("FindNextPhaseTasks() phase = %q, want %q",
					result.PhaseName, tc.wantPhase)
			}

			if len(result.Tasks) != tc.wantTaskCount {
				t.Errorf("FindNextPhaseTasks() returned %d tasks, want %d",
					len(result.Tasks), tc.wantTaskCount)
			}
		})
	}
}

// TestFindNextPhaseTasksForStream_NonSequentialIDs verifies that
// FindNextPhaseTasksForStream correctly handles non-sequential markdown IDs.
// Regression test for T-604.
func TestFindNextPhaseTasksForStream_NonSequentialIDs(t *testing.T) {
	tests := map[string]struct {
		content     string
		stream      int
		wantNil     bool
		wantPhase   string
		wantTaskIDs []string
	}{
		"non-sequential IDs with streams": {
			content: `## Phase A
- [ ] 10. Task in stream 1
  - Stream: 1
- [ ] 20. Task in stream 2
  - Stream: 2

## Phase B
- [ ] 30. Another stream 2 task
  - Stream: 2
`,
			stream:      2,
			wantPhase:   "Phase A",
			wantTaskIDs: []string{"2"},
		},
		"non-sequential skip to second phase": {
			content: `## Phase A
- [ ] 10. Task in stream 1
  - Stream: 1

## Phase B
- [ ] 20. Stream 2 task
  - Stream: 2
`,
			stream:      2,
			wantPhase:   "Phase B",
			wantTaskIDs: []string{"2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "nonseq-stream-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			tmpFile.Close()

			result, err := FindNextPhaseTasksForStream(tmpFile.Name(), tc.stream)
			if err != nil {
				t.Fatalf("FindNextPhaseTasksForStream() error = %v", err)
			}

			if tc.wantNil {
				if result != nil {
					t.Errorf("got non-nil, want nil")
				}
				return
			}

			if result == nil {
				t.Fatalf("got nil, want phase %q", tc.wantPhase)
			}

			if result.PhaseName != tc.wantPhase {
				t.Errorf("phase = %q, want %q", result.PhaseName, tc.wantPhase)
			}

			var gotIDs []string
			for _, task := range result.Tasks {
				gotIDs = append(gotIDs, task.ID)
			}

			if len(gotIDs) != len(tc.wantTaskIDs) {
				t.Fatalf("got %d tasks %v, want %d tasks %v",
					len(gotIDs), gotIDs, len(tc.wantTaskIDs), tc.wantTaskIDs)
			}

			for i, wantID := range tc.wantTaskIDs {
				if gotIDs[i] != wantID {
					t.Errorf("task[%d].ID = %q, want %q", i, gotIDs[i], wantID)
				}
			}
		})
	}
}

// TestSkipFrontMatter_HorizontalRuleAfterFrontMatter verifies that a horizontal
// rule (---) appearing after YAML front matter does not cause subsequent lines to
// be dropped. Regression test for T-763.
func TestSkipFrontMatter_HorizontalRuleAfterFrontMatter(t *testing.T) {
	tests := map[string]struct {
		content   string
		wantLines []string
	}{
		"hr after front matter preserved": {
			content: "---\ntitle: test\n---\n## Phase A\n- [ ] 1. Task A\n---\n## Phase B\n- [ ] 2. Task B\n",
			wantLines: []string{
				"## Phase A",
				"- [ ] 1. Task A",
				"---",
				"## Phase B",
				"- [ ] 2. Task B",
				"",
			},
		},
		"no front matter unchanged": {
			content: "## Phase A\n- [ ] 1. Task A\n---\n## Phase B\n- [ ] 2. Task B\n",
			wantLines: []string{
				"## Phase A",
				"- [ ] 1. Task A",
				"---",
				"## Phase B",
				"- [ ] 2. Task B",
				"",
			},
		},
		"front matter only": {
			content: "---\ntitle: test\n---\n## Phase A\n- [ ] 1. Task A\n",
			wantLines: []string{
				"## Phase A",
				"- [ ] 1. Task A",
				"",
			},
		},
		"multiple hrs after front matter preserved": {
			content: "---\ntitle: x\n---\n## A\n---\n## B\n---\n## C\n",
			wantLines: []string{
				"## A",
				"---",
				"## B",
				"---",
				"## C",
				"",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			lines := strings.Split(tc.content, "\n")
			got := skipFrontMatter(tc.content, lines)
			if len(got) != len(tc.wantLines) {
				t.Fatalf("skipFrontMatter() returned %d lines %v, want %d lines %v",
					len(got), got, len(tc.wantLines), tc.wantLines)
			}
			for i, want := range tc.wantLines {
				if got[i] != want {
					t.Errorf("line[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

// TestFindNextPhaseTasks_WithFrontMatter is an end-to-end test verifying that
// FindNextPhaseTasks correctly handles files with YAML front matter. It first
// confirms skipFrontMatter preserves phase content, then exercises the full
// FindNextPhaseTasks pipeline. Regression test for T-763.
//
// Note: ParseMarkdown does not allow bare --- at root level in a task file,
// so the end-to-end portion uses content without a horizontal rule separator.
func TestFindNextPhaseTasks_WithFrontMatter(t *testing.T) {
	// Verify skipFrontMatter with content matching FindNextPhaseTasks usage pattern.
	content := "---\ntitle: Project\n---\n## Phase A\n- [x] 1. Done task\n\n## Phase B\n- [ ] 2. Pending task\n"
	lines := strings.Split(content, "\n")
	got := skipFrontMatter(content, lines)

	// After stripping, we should have Phase A, Phase B content intact.
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "## Phase A") {
		t.Errorf("skipFrontMatter dropped Phase A")
	}
	if !strings.Contains(joined, "## Phase B") {
		t.Errorf("skipFrontMatter dropped Phase B")
	}
	if !strings.Contains(joined, "- [ ] 2. Pending task") {
		t.Errorf("skipFrontMatter dropped tasks after front matter")
	}

	// End-to-end: verify the full pipeline finds the correct next phase.
	tmpFile, err := os.CreateTemp("", "hr-frontmatter-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpFile.Close()

	result, err := FindNextPhaseTasks(tmpFile.Name())
	if err != nil {
		t.Fatalf("FindNextPhaseTasks() error = %v", err)
	}

	if result == nil {
		t.Fatal("FindNextPhaseTasks() = nil, want Phase B with 1 task")
	}

	if result.PhaseName != "Phase B" {
		t.Errorf("phase = %q, want %q", result.PhaseName, "Phase B")
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(result.Tasks))
	}

	if result.Tasks[0].Title != "Pending task" {
		t.Errorf("task title = %q, want %q", result.Tasks[0].Title, "Pending task")
	}
}

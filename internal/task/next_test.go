package task

import (
	"os"
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

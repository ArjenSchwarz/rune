package task

import (
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

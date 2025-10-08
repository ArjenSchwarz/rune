package task

import (
	"strings"
	"testing"
)

func TestAutoCompleteParents(t *testing.T) {
	tests := map[string]struct {
		taskList          *TaskList
		completedTaskID   string
		expectedCompleted []string
		expectError       bool
		errorContains     string
	}{
		"no parent to complete": {
			taskList: &TaskList{
				Tasks: []Task{
					{ID: "1", Title: "Task 1", Status: Completed},
				},
			},
			completedTaskID:   "1",
			expectedCompleted: []string{},
			expectError:       false,
		},
		"single level parent completion": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Parent", Status: Pending,
						Children: []Task{
							{ID: "1.1", Title: "Child 1", Status: Completed, ParentID: "1"},
							{ID: "1.2", Title: "Child 2", Status: Completed, ParentID: "1"},
						},
					},
				},
			},
			completedTaskID:   "1.2",
			expectedCompleted: []string{"1"},
			expectError:       false,
		},
		"multi-level parent completion": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Grandparent", Status: Pending,
						Children: []Task{
							{
								ID: "1.1", Title: "Parent", Status: Pending, ParentID: "1",
								Children: []Task{
									{ID: "1.1.1", Title: "Child 1", Status: Completed, ParentID: "1.1"},
									{ID: "1.1.2", Title: "Child 2", Status: Completed, ParentID: "1.1"},
								},
							},
						},
					},
				},
			},
			completedTaskID:   "1.1.2",
			expectedCompleted: []string{"1.1", "1"},
			expectError:       false,
		},
		"partial completion - only immediate parent": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Grandparent", Status: Pending,
						Children: []Task{
							{
								ID: "1.1", Title: "Parent 1", Status: Pending, ParentID: "1",
								Children: []Task{
									{ID: "1.1.1", Title: "Child 1", Status: Completed, ParentID: "1.1"},
									{ID: "1.1.2", Title: "Child 2", Status: Completed, ParentID: "1.1"},
								},
							},
							{ID: "1.2", Title: "Parent 2", Status: Pending, ParentID: "1"}, // Incomplete sibling
						},
					},
				},
			},
			completedTaskID:   "1.1.2",
			expectedCompleted: []string{"1.1"}, // Only parent, not grandparent
			expectError:       false,
		},
		"parent already complete": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Parent", Status: Completed,
						Children: []Task{
							{ID: "1.1", Title: "Child 1", Status: Completed, ParentID: "1"},
							{ID: "1.2", Title: "Child 2", Status: Completed, ParentID: "1"},
						},
					},
				},
			},
			completedTaskID:   "1.2",
			expectedCompleted: []string{}, // Parent already complete
			expectError:       false,
		},
		"missing parent task": {
			taskList:          &TaskList{Tasks: []Task{}},
			completedTaskID:   "1.1",
			expectedCompleted: []string{},
			expectError:       false, // Should handle gracefully
		},
		"deep hierarchy completion": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Level 1", Status: Pending,
						Children: []Task{
							{
								ID: "1.1", Title: "Level 2", Status: Pending, ParentID: "1",
								Children: []Task{
									{
										ID: "1.1.1", Title: "Level 3", Status: Pending, ParentID: "1.1",
										Children: []Task{
											{ID: "1.1.1.1", Title: "Level 4", Status: Completed, ParentID: "1.1.1"},
										},
									},
								},
							},
						},
					},
				},
			},
			completedTaskID:   "1.1.1.1",
			expectedCompleted: []string{"1.1.1", "1.1", "1"},
			expectError:       false,
		},
		"incomplete sibling prevents completion": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Parent", Status: Pending,
						Children: []Task{
							{ID: "1.1", Title: "Child 1", Status: Completed, ParentID: "1"},
							{ID: "1.2", Title: "Child 2", Status: Pending, ParentID: "1"}, // Incomplete
						},
					},
				},
			},
			completedTaskID:   "1.1",
			expectedCompleted: []string{}, // Parent cannot be completed
			expectError:       false,
		},
		"mixed subtask completion states": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Parent", Status: Pending,
						Children: []Task{
							{
								ID: "1.1", Title: "Child 1", Status: Completed, ParentID: "1",
								Children: []Task{
									{ID: "1.1.1", Title: "Grandchild 1", Status: Completed, ParentID: "1.1"},
									{ID: "1.1.2", Title: "Grandchild 2", Status: Pending, ParentID: "1.1"}, // Incomplete grandchild
								},
							},
							{ID: "1.2", Title: "Child 2", Status: Completed, ParentID: "1"},
						},
					},
				},
			},
			completedTaskID:   "1.2",
			expectedCompleted: []string{}, // Parent cannot complete due to 1.1.2 being incomplete
			expectError:       false,
		},
		"nested completion with all children done": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID: "1", Title: "Parent", Status: Pending,
						Children: []Task{
							{
								ID: "1.1", Title: "Child 1", Status: Completed, ParentID: "1",
								Children: []Task{
									{ID: "1.1.1", Title: "Grandchild 1", Status: Completed, ParentID: "1.1"},
									{ID: "1.1.2", Title: "Grandchild 2", Status: Completed, ParentID: "1.1"},
								},
							},
							{ID: "1.2", Title: "Child 2", Status: Completed, ParentID: "1"},
						},
					},
				},
			},
			completedTaskID:   "1.1.2",
			expectedCompleted: []string{"1"}, // 1.1 already complete, only 1 gets completed
			expectError:       false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			completed, err := tc.taskList.AutoCompleteParents(tc.completedTaskID)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("expected error to contain %q but got %q", tc.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(completed) != len(tc.expectedCompleted) {
				t.Errorf("expected %d completed parents, got %d: %v", len(tc.expectedCompleted), len(completed), completed)
				return
			}

			for i, expected := range tc.expectedCompleted {
				if i >= len(completed) || completed[i] != expected {
					t.Errorf("expected completed[%d] = %s, got %v", i, expected, completed)
				}
			}

			// Verify that the tasks were actually marked as completed
			for _, taskID := range completed {
				task := tc.taskList.FindTask(taskID)
				if task == nil {
					t.Errorf("completed task %s not found", taskID)
				} else if task.Status != Completed {
					t.Errorf("task %s should be completed but has status %v", taskID, task.Status)
				}
			}
		})
	}
}

func TestAllChildrenComplete(t *testing.T) {
	tests := map[string]struct {
		task     Task
		expected bool
	}{
		"no children": {
			task:     Task{ID: "1", Title: "Task", Status: Pending},
			expected: true, // No children means all children are complete
		},
		"all children complete": {
			task: Task{
				ID: "1", Title: "Parent", Status: Pending,
				Children: []Task{
					{ID: "1.1", Title: "Child 1", Status: Completed},
					{ID: "1.2", Title: "Child 2", Status: Completed},
				},
			},
			expected: true,
		},
		"one child incomplete": {
			task: Task{
				ID: "1", Title: "Parent", Status: Pending,
				Children: []Task{
					{ID: "1.1", Title: "Child 1", Status: Completed},
					{ID: "1.2", Title: "Child 2", Status: Pending},
				},
			},
			expected: false,
		},
		"child in progress": {
			task: Task{
				ID: "1", Title: "Parent", Status: Pending,
				Children: []Task{
					{ID: "1.1", Title: "Child 1", Status: Completed},
					{ID: "1.2", Title: "Child 2", Status: InProgress},
				},
			},
			expected: false,
		},
		"nested children all complete": {
			task: Task{
				ID: "1", Title: "Parent", Status: Pending,
				Children: []Task{
					{
						ID: "1.1", Title: "Child 1", Status: Completed,
						Children: []Task{
							{ID: "1.1.1", Title: "Grandchild 1", Status: Completed},
							{ID: "1.1.2", Title: "Grandchild 2", Status: Completed},
						},
					},
					{ID: "1.2", Title: "Child 2", Status: Completed},
				},
			},
			expected: true,
		},
		"nested children with incomplete grandchild": {
			task: Task{
				ID: "1", Title: "Parent", Status: Pending,
				Children: []Task{
					{
						ID: "1.1", Title: "Child 1", Status: Completed,
						Children: []Task{
							{ID: "1.1.1", Title: "Grandchild 1", Status: Completed},
							{ID: "1.1.2", Title: "Grandchild 2", Status: Pending},
						},
					},
					{ID: "1.2", Title: "Child 2", Status: Completed},
				},
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := allChildrenComplete(&tc.task)
			if result != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func TestGetParentID(t *testing.T) {
	tests := map[string]struct {
		taskID   string
		expected string
	}{
		"top level task":      {taskID: "1", expected: ""},
		"second level task":   {taskID: "1.2", expected: "1"},
		"third level task":    {taskID: "1.2.3", expected: "1.2"},
		"deep nested task":    {taskID: "1.2.3.4.5", expected: "1.2.3.4"},
		"empty string":        {taskID: "", expected: ""},
		"single digit":        {taskID: "5", expected: ""},
		"double digit":        {taskID: "10.20", expected: "10"},
		"triple level double": {taskID: "10.20.30", expected: "10.20"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := getParentID(tc.taskID)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

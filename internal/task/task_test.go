package task

import (
	"testing"
	"time"
)

func TestStatus_String(t *testing.T) {
	tests := map[string]struct {
		status Status
		want   string
	}{
		"pending": {
			status: Pending,
			want:   "[ ]",
		},
		"in_progress": {
			status: InProgress,
			want:   "[-]",
		},
		"completed": {
			status: Completed,
			want:   "[x]",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.status.String()
			if got != tc.want {
				t.Errorf("Status.String() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    Status
		wantErr bool
	}{
		"pending_brackets": {
			input: "[ ]",
			want:  Pending,
		},
		"in_progress_dash": {
			input: "[-]",
			want:  InProgress,
		},
		"completed_x": {
			input: "[x]",
			want:  Completed,
		},
		"completed_X": {
			input: "[X]",
			want:  Completed,
		},
		"invalid_status": {
			input:   "[?]",
			wantErr: true,
		},
		"empty_string": {
			input:   "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseStatus(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseStatus() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("ParseStatus() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTask_Validation(t *testing.T) {
	tests := map[string]struct {
		task    *Task
		wantErr bool
	}{
		"valid_task": {
			task: &Task{
				ID:    "1",
				Title: "Valid task",
			},
		},
		"empty_title": {
			task: &Task{
				ID:    "1",
				Title: "",
			},
			wantErr: true,
		},
		"title_too_long": {
			task: &Task{
				ID:    "1",
				Title: string(make([]byte, 501)),
			},
			wantErr: true,
		},
		"invalid_id_format": {
			task: &Task{
				ID:    "1.a",
				Title: "Task",
			},
			wantErr: true,
		},
		"valid_hierarchical_id": {
			task: &Task{
				ID:    "1.2.3",
				Title: "Subtask",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.task.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Task.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestTaskList_FindTask(t *testing.T) {
	taskList := &TaskList{
		Title: "Test List",
		Tasks: []Task{
			{
				ID:    "1",
				Title: "First task",
				Children: []Task{
					{
						ID:    "1.1",
						Title: "Subtask",
						Children: []Task{
							{
								ID:    "1.1.1",
								Title: "Deep subtask",
							},
						},
					},
					{
						ID:    "1.2",
						Title: "Another subtask",
					},
				},
			},
			{
				ID:    "2",
				Title: "Second task",
			},
		},
	}

	tests := map[string]struct {
		taskID string
		want   string
		found  bool
	}{
		"find_root_task": {
			taskID: "1",
			want:   "First task",
			found:  true,
		},
		"find_subtask": {
			taskID: "1.1",
			want:   "Subtask",
			found:  true,
		},
		"find_deep_subtask": {
			taskID: "1.1.1",
			want:   "Deep subtask",
			found:  true,
		},
		"task_not_found": {
			taskID: "3",
			found:  false,
		},
		"empty_id": {
			taskID: "",
			found:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := taskList.FindTask(tc.taskID)
			if tc.found {
				if got == nil {
					t.Errorf("FindTask() = nil, want task with title %v", tc.want)
					return
				}
				if got.Title != tc.want {
					t.Errorf("FindTask().Title = %v, want %v", got.Title, tc.want)
				}
			} else {
				if got != nil {
					t.Errorf("FindTask() = %v, want nil", got)
				}
			}
		})
	}
}

func TestTaskList_AddTask(t *testing.T) {
	tests := map[string]struct {
		setup    func() *TaskList
		parentID string
		title    string
		wantID   string
		wantErr  bool
	}{
		"add_root_task": {
			setup: func() *TaskList {
				return &TaskList{Title: "Test"}
			},
			parentID: "",
			title:    "New task",
			wantID:   "1",
		},
		"add_second_root_task": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "First"}},
				}
			},
			parentID: "",
			title:    "Second task",
			wantID:   "2",
		},
		"add_subtask": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Parent"}},
				}
			},
			parentID: "1",
			title:    "Subtask",
			wantID:   "1.1",
		},
		"add_to_nonexistent_parent": {
			setup: func() *TaskList {
				return &TaskList{Title: "Test"}
			},
			parentID: "99",
			title:    "Orphan task",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := tc.setup()
			err := tl.AddTask(tc.parentID, tc.title)
			if (err != nil) != tc.wantErr {
				t.Errorf("AddTask() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				task := tl.FindTask(tc.wantID)
				if task == nil {
					t.Errorf("Task with ID %v not found after AddTask", tc.wantID)
					return
				}
				if task.Title != tc.title {
					t.Errorf("Added task title = %v, want %v", task.Title, tc.title)
				}
			}
		})
	}
}

func TestTaskList_RemoveTask(t *testing.T) {
	tests := map[string]struct {
		setup           func() *TaskList
		removeID        string
		checkID         string
		expectNewID     string
		expectRemaining int
		wantErr         bool
	}{
		"remove_single_task": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{
						{ID: "1", Title: "First"},
						{ID: "2", Title: "Second"},
						{ID: "3", Title: "Third"},
					},
				}
			},
			removeID:        "2",
			checkID:         "3",
			expectNewID:     "2",
			expectRemaining: 2,
		},
		"remove_task_with_children": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{
						{
							ID:    "1",
							Title: "Parent",
							Children: []Task{
								{ID: "1.1", Title: "Child1"},
								{ID: "1.2", Title: "Child2"},
							},
						},
						{ID: "2", Title: "Second"},
					},
				}
			},
			removeID:        "1",
			checkID:         "2",
			expectNewID:     "1",
			expectRemaining: 1,
		},
		"remove_subtask": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{
						{
							ID:    "1",
							Title: "Parent",
							Children: []Task{
								{ID: "1.1", Title: "Child1"},
								{ID: "1.2", Title: "Child2"},
								{ID: "1.3", Title: "Child3"},
							},
						},
					},
				}
			},
			removeID:        "1.2",
			checkID:         "1.3",
			expectNewID:     "1.2",
			expectRemaining: 1,
		},
		"remove_nonexistent": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Only"}},
				}
			},
			removeID: "99",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := tc.setup()
			err := tl.RemoveTask(tc.removeID)
			if (err != nil) != tc.wantErr {
				t.Errorf("RemoveTask() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if tc.checkID != "" {
					task := tl.FindTask(tc.expectNewID)
					if task == nil {
						t.Errorf("Expected task with ID %v not found after removal", tc.expectNewID)
					}
				}
				if tc.expectRemaining > 0 {
					if len(tl.Tasks) != tc.expectRemaining {
						t.Errorf("Expected %d remaining tasks, got %d", tc.expectRemaining, len(tl.Tasks))
					}
				}
			}
		})
	}
}

func TestTaskList_UpdateStatus(t *testing.T) {
	tests := map[string]struct {
		setup     func() *TaskList
		taskID    string
		newStatus Status
		wantErr   bool
	}{
		"update_root_task_status": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Task", Status: Pending}},
				}
			},
			taskID:    "1",
			newStatus: Completed,
		},
		"update_subtask_status": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{
						{
							ID:     "1",
							Title:  "Parent",
							Status: Pending,
							Children: []Task{
								{ID: "1.1", Title: "Child", Status: Pending},
							},
						},
					},
				}
			},
			taskID:    "1.1",
			newStatus: InProgress,
		},
		"update_nonexistent_task": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Task", Status: Pending}},
				}
			},
			taskID:    "99",
			newStatus: Completed,
			wantErr:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := tc.setup()
			err := tl.UpdateStatus(tc.taskID, tc.newStatus)
			if (err != nil) != tc.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				task := tl.FindTask(tc.taskID)
				if task.Status != tc.newStatus {
					t.Errorf("Task status = %v, want %v", task.Status, tc.newStatus)
				}
			}
		})
	}
}

func TestTaskList_UpdateTask(t *testing.T) {
	tests := map[string]struct {
		setup   func() *TaskList
		taskID  string
		title   string
		details []string
		refs    []string
		wantErr bool
	}{
		"update_title": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Old title"}},
				}
			},
			taskID: "1",
			title:  "New title",
		},
		"update_details": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Task"}},
				}
			},
			taskID:  "1",
			details: []string{"Detail 1", "Detail 2"},
		},
		"update_references": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Task"}},
				}
			},
			taskID: "1",
			refs:   []string{"ref1.md", "ref2.md"},
		},
		"update_all_fields": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Old"}},
				}
			},
			taskID:  "1",
			title:   "New title",
			details: []string{"New detail"},
			refs:    []string{"new.md"},
		},
		"update_nonexistent": {
			setup: func() *TaskList {
				return &TaskList{
					Title: "Test",
					Tasks: []Task{{ID: "1", Title: "Task"}},
				}
			},
			taskID:  "99",
			title:   "New",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := tc.setup()
			err := tl.UpdateTask(tc.taskID, tc.title, tc.details, tc.refs)
			if (err != nil) != tc.wantErr {
				t.Errorf("UpdateTask() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				task := tl.FindTask(tc.taskID)
				if tc.title != "" && task.Title != tc.title {
					t.Errorf("Task title = %v, want %v", task.Title, tc.title)
				}
				if tc.details != nil && len(task.Details) != len(tc.details) {
					t.Errorf("Task details count = %v, want %v", len(task.Details), len(tc.details))
				}
				if tc.refs != nil && len(task.References) != len(tc.refs) {
					t.Errorf("Task refs count = %v, want %v", len(task.References), len(tc.refs))
				}
			}
		})
	}
}

func TestTaskList_ModifiedTime(t *testing.T) {
	tl := &TaskList{Title: "Test"}

	initialTime := tl.Modified
	time.Sleep(10 * time.Millisecond)

	err := tl.AddTask("", "New task")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if !tl.Modified.After(initialTime) {
		t.Errorf("Modified time not updated after AddTask")
	}

	time.Sleep(10 * time.Millisecond)
	beforeUpdate := tl.Modified

	err = tl.UpdateStatus("1", Completed)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	if !tl.Modified.After(beforeUpdate) {
		t.Errorf("Modified time not updated after UpdateStatus")
	}
}

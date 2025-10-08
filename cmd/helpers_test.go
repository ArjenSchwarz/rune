package cmd

import (
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

func TestFormatStatus(t *testing.T) {
	tests := map[string]struct {
		status task.Status
		want   string
	}{
		"pending":     {task.Pending, "Pending"},
		"in_progress": {task.InProgress, "In Progress"},
		"completed":   {task.Completed, "Completed"},
		"unknown":     {task.Status(99), "Unknown"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatStatus(tc.status)
			if got != tc.want {
				t.Errorf("formatStatus(%v) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestFormatStatusMarkdown(t *testing.T) {
	tests := map[string]struct {
		status task.Status
		want   string
	}{
		"pending":     {task.Pending, "[ ]"},
		"in_progress": {task.InProgress, "[-]"},
		"completed":   {task.Completed, "[x]"},
		"unknown":     {task.Status(99), "[ ]"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatStatusMarkdown(tc.status)
			if got != tc.want {
				t.Errorf("formatStatusMarkdown(%v) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestGetTaskLevel(t *testing.T) {
	tests := map[string]struct {
		id   string
		want int
	}{
		"empty":         {"", 0},
		"top_level":     {"1", 1},
		"second_level":  {"1.2", 2},
		"third_level":   {"1.2.3", 3},
		"fourth_level":  {"1.2.3.4", 4},
		"multi_digit":   {"10.20.30", 3},
		"single_dot":    {".", 2},
		"trailing_dots": {"1.2.", 3},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := getTaskLevel(tc.id)
			if got != tc.want {
				t.Errorf("getTaskLevel(%q) = %d, want %d", tc.id, got, tc.want)
			}
		})
	}
}

func TestCountAllTasks(t *testing.T) {
	tests := map[string]struct {
		tasks []task.Task
		want  int
	}{
		"empty": {
			tasks: []task.Task{},
			want:  0,
		},
		"single_task": {
			tasks: []task.Task{
				{ID: "1", Title: "Task 1"},
			},
			want: 1,
		},
		"multiple_tasks_flat": {
			tasks: []task.Task{
				{ID: "1", Title: "Task 1"},
				{ID: "2", Title: "Task 2"},
				{ID: "3", Title: "Task 3"},
			},
			want: 3,
		},
		"nested_tasks_simple": {
			tasks: []task.Task{
				{
					ID:    "1",
					Title: "Task 1",
					Children: []task.Task{
						{ID: "1.1", Title: "Task 1.1"},
						{ID: "1.2", Title: "Task 1.2"},
					},
				},
			},
			want: 3,
		},
		"nested_tasks_complex": {
			tasks: []task.Task{
				{
					ID:    "1",
					Title: "Task 1",
					Children: []task.Task{
						{ID: "1.1", Title: "Task 1.1"},
						{
							ID:    "1.2",
							Title: "Task 1.2",
							Children: []task.Task{
								{ID: "1.2.1", Title: "Task 1.2.1"},
								{ID: "1.2.2", Title: "Task 1.2.2"},
							},
						},
					},
				},
				{
					ID:    "2",
					Title: "Task 2",
					Children: []task.Task{
						{ID: "2.1", Title: "Task 2.1"},
					},
				},
			},
			want: 7,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := countAllTasks(tc.tasks)
			if got != tc.want {
				t.Errorf("countAllTasks() = %d, want %d", got, tc.want)
			}
		})
	}
}

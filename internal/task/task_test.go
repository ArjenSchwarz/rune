package task

import (
	"testing"
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

func TestTask_RequirementsValidation(t *testing.T) {
	tests := map[string]struct {
		task    *Task
		wantErr bool
	}{
		"valid_single_requirement": {
			task: &Task{
				ID:           "1",
				Title:        "Task with single requirement",
				Requirements: []string{"1.1"},
			},
		},
		"valid_multiple_requirements": {
			task: &Task{
				ID:           "1",
				Title:        "Task with multiple requirements",
				Requirements: []string{"1.1", "1.2", "2.3"},
			},
		},
		"valid_hierarchical_requirement_ids": {
			task: &Task{
				ID:           "1",
				Title:        "Task with hierarchical requirement IDs",
				Requirements: []string{"1", "1.1", "1.2.3", "2.4.5.6"},
			},
		},
		"empty_requirements_array": {
			task: &Task{
				ID:           "1",
				Title:        "Task with empty requirements",
				Requirements: []string{},
			},
		},
		"nil_requirements_array": {
			task: &Task{
				ID:           "1",
				Title:        "Task with nil requirements",
				Requirements: nil,
			},
		},
		"invalid_requirement_id_format_letters": {
			task: &Task{
				ID:           "1",
				Title:        "Task with invalid requirement",
				Requirements: []string{"abc"},
			},
			wantErr: true,
		},
		"invalid_requirement_id_format_mixed": {
			task: &Task{
				ID:           "1",
				Title:        "Task with invalid requirement",
				Requirements: []string{"1.a"},
			},
			wantErr: true,
		},
		"invalid_requirement_id_starts_with_zero": {
			task: &Task{
				ID:           "1",
				Title:        "Task with invalid requirement",
				Requirements: []string{"0.1"},
			},
			wantErr: true,
		},
		"invalid_requirement_id_has_zero_segment": {
			task: &Task{
				ID:           "1",
				Title:        "Task with invalid requirement",
				Requirements: []string{"1.0.1"},
			},
			wantErr: true,
		},
		"mixed_valid_and_invalid_requirements": {
			task: &Task{
				ID:           "1",
				Title:        "Task with mixed requirements",
				Requirements: []string{"1.1", "invalid", "2.3"},
			},
			wantErr: true,
		},
		"valid_requirements_with_large_numbers": {
			task: &Task{
				ID:           "1",
				Title:        "Task with large requirement numbers",
				Requirements: []string{"99.999.9999"},
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

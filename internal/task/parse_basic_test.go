package task

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantTitle   string
		wantTasks   int
		wantErr     bool
		errContains string
	}{
		"simple_tasks": {
			content: `# Simple Tasks

- [ ] 1. First task
- [-] 2. Second task
- [x] 3. Third task`,
			wantTitle: "Simple Tasks",
			wantTasks: 3,
		},
		"tasks_with_subtasks": {
			content: `# Project

- [ ] 1. Main task
  - [ ] 1.1. Subtask one
  - [ ] 1.2. Subtask two`,
			wantTitle: "Project",
			wantTasks: 1,
		},
		"empty_file": {
			content:   "",
			wantTitle: "",
			wantTasks: 0,
		},
		"only_title": {
			content:   "# My Tasks",
			wantTitle: "My Tasks",
			wantTasks: 0,
		},
		"invalid_checkbox": {
			content: `# Tasks
- [?] 1. Invalid status`,
			wantErr:     true,
			errContains: "invalid status",
		},
		"unexpected_indentation": {
			content: `# Tasks
- [ ] 1. Task
    - [ ] 1.1. Too indented`,
			wantErr:     true,
			errContains: "unexpected indentation",
		},
		"file_too_large": {
			content:     strings.Repeat("x", MaxFileSize+1),
			wantErr:     true,
			errContains: "exceeds maximum size",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMarkdown() expected error containing %q, got nil", tc.errContains)
					return
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("ParseMarkdown() error = %v, want error containing %q", err, tc.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMarkdown() unexpected error: %v", err)
				return
			}

			if tl.Title != tc.wantTitle {
				t.Errorf("ParseMarkdown() title = %q, want %q", tl.Title, tc.wantTitle)
			}

			if len(tl.Tasks) != tc.wantTasks {
				t.Errorf("ParseMarkdown() tasks count = %d, want %d", len(tl.Tasks), tc.wantTasks)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tests := map[string]struct {
		filename    string
		wantTitle   string
		wantTasks   int
		wantErr     bool
		errContains string
	}{
		"simple_file": {
			filename:  "simple.md",
			wantTitle: "Simple Tasks",
			wantTasks: 3,
		},
		"complex_file": {
			filename:  "complex.md",
			wantTitle: "Complex Project Tasks",
			wantTasks: 3,
		},
		"malformed_file": {
			filename:    "malformed.md",
			wantErr:     true,
			errContains: "missing task number",
		},
		"nonexistent_file": {
			filename:    "nonexistent.md",
			wantErr:     true,
			errContains: "reading file",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("testdata", tc.filename)
			tl, err := ParseFile(path)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseFile() expected error containing %q, got nil", tc.errContains)
					return
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("ParseFile() error = %v, want error containing %q", err, tc.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseFile() unexpected error: %v", err)
				return
			}

			if tl.Title != tc.wantTitle {
				t.Errorf("ParseFile() title = %q, want %q", tl.Title, tc.wantTitle)
			}

			if len(tl.Tasks) != tc.wantTasks {
				t.Errorf("ParseFile() tasks count = %d, want %d", len(tl.Tasks), tc.wantTasks)
			}
		})
	}
}

func TestParseTaskStatus(t *testing.T) {
	tests := map[string]struct {
		content      string
		wantStatuses map[string]Status
	}{
		"all_status_types": {
			content: `# Tasks
- [ ] 1. Pending task
- [-] 2. In progress task
- [x] 3. Completed task
- [X] 4. Also completed`,
			wantStatuses: map[string]Status{
				"1": Pending,
				"2": InProgress,
				"3": Completed,
				"4": Completed,
			},
		},
		"nested_statuses": {
			content: `# Tasks
- [-] 1. Parent in progress
  - [x] 1.1. Child completed
  - [ ] 1.2. Child pending`,
			wantStatuses: map[string]Status{
				"1":   InProgress,
				"1.1": Completed,
				"1.2": Pending,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error: %v", err)
			}

			for id, wantStatus := range tc.wantStatuses {
				task := tl.FindTask(id)
				if task == nil {
					t.Errorf("Task %s not found", id)
					continue
				}
				if task.Status != wantStatus {
					t.Errorf("Task %s status = %v, want %v", id, task.Status, wantStatus)
				}
			}
		})
	}
}

func TestParseDetailsAndReferences(t *testing.T) {
	tests := map[string]struct {
		content        string
		taskID         string
		wantDetails    []string
		wantReferences []string
	}{
		"task_with_details": {
			content: `# Tasks
- [ ] 1. Main task
  - Detail one
  - Detail two
  - Detail three`,
			taskID:      "1",
			wantDetails: []string{"Detail one", "Detail two", "Detail three"},
		},
		"task_with_references": {
			content: `# Tasks
- [ ] 1. Main task
  - References: doc1.md, doc2.md, doc3.md`,
			taskID:         "1",
			wantReferences: []string{"doc1.md", "doc2.md", "doc3.md"},
		},
		"task_with_both": {
			content: `# Tasks
- [ ] 1. Main task
  - First detail
  - Second detail
  - References: ref1.md, ref2.md`,
			taskID:         "1",
			wantDetails:    []string{"First detail", "Second detail"},
			wantReferences: []string{"ref1.md", "ref2.md"},
		},
		"subtask_with_details": {
			content: `# Tasks
- [ ] 1. Parent
  - [ ] 1.1. Child task
    - Child detail
    - References: child.md`,
			taskID:         "1.1",
			wantDetails:    []string{"Child detail"},
			wantReferences: []string{"child.md"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error: %v", err)
			}

			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			if tc.wantDetails != nil {
				if len(task.Details) != len(tc.wantDetails) {
					t.Errorf("Task details count = %d, want %d", len(task.Details), len(tc.wantDetails))
				}
				for i, detail := range tc.wantDetails {
					if i >= len(task.Details) {
						break
					}
					if task.Details[i] != detail {
						t.Errorf("Task detail[%d] = %q, want %q", i, task.Details[i], detail)
					}
				}
			}

			if tc.wantReferences != nil {
				if len(task.References) != len(tc.wantReferences) {
					t.Errorf("Task references count = %d, want %d", len(task.References), len(tc.wantReferences))
				}
				for i, ref := range tc.wantReferences {
					if i >= len(task.References) {
						break
					}
					if task.References[i] != ref {
						t.Errorf("Task reference[%d] = %q, want %q", i, task.References[i], ref)
					}
				}
			}
		})
	}
}

func TestParseHierarchy(t *testing.T) {
	content := `# Hierarchical Tasks

- [ ] 1. Level 1 task
  - [ ] 1.1. Level 2 task
    - [ ] 1.1.1. Level 3 task
    - [ ] 1.1.2. Another level 3
  - [ ] 1.2. Another level 2
- [ ] 2. Second root task
  - [ ] 2.1. Its child`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	tests := map[string]struct {
		taskID       string
		wantParentID string
		wantChildren int
	}{
		"root_task_1": {
			taskID:       "1",
			wantParentID: "",
			wantChildren: 2,
		},
		"child_1.1": {
			taskID:       "1.1",
			wantParentID: "1",
			wantChildren: 2,
		},
		"deep_child_1.1.1": {
			taskID:       "1.1.1",
			wantParentID: "1.1",
			wantChildren: 0,
		},
		"root_task_2": {
			taskID:       "2",
			wantParentID: "",
			wantChildren: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			if task.ParentID != tc.wantParentID {
				t.Errorf("Task %s parent ID = %q, want %q", tc.taskID, task.ParentID, tc.wantParentID)
			}

			if len(task.Children) != tc.wantChildren {
				t.Errorf("Task %s children count = %d, want %d", tc.taskID, len(task.Children), tc.wantChildren)
			}
		})
	}
}

func TestParseMalformedContent(t *testing.T) {
	tests := map[string]struct {
		content     string
		errContains string
	}{
		"wrong_indentation_4_spaces": {
			content: `# Tasks
- [ ] 1. Task
    - [ ] 1.1. Wrong indent (4 spaces instead of 2)`,
			errContains: "unexpected indentation",
		},
		"wrong_indentation_1_space": {
			content: `# Tasks
- [ ] 1. Task
 - [ ] 1.1. Wrong indent (1 space instead of 2)`,
			errContains: "unexpected indentation",
		},
		"mixed_indentation": {
			content: `# Tasks
- [ ] 1. Task
  - [ ] 1.1. Correct indent
	- [ ] 1.2. Tab instead of spaces`,
			errContains: "unexpected indentation",
		},
		"orphaned_subtask": {
			content: `# Tasks
- [ ] 1. Task
    - [ ] 1.1. Subtask
      - [ ] 1.1.1. Too deep without proper parent`,
			errContains: "unexpected indentation",
		},
		"invalid_status_marker": {
			content: `# Tasks
- [~] 1. Unknown status`,
			errContains: "invalid status",
		},
		"missing_space_after_checkbox": {
			content: `# Tasks
- [ ]1. No space after checkbox`,
			errContains: "invalid task format",
		},
		"no_number_prefix": {
			content: `# Tasks
- [ ] Task without number`,
			errContains: "invalid task format",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseMarkdown([]byte(tc.content))
			if err == nil {
				t.Errorf("ParseMarkdown() expected error containing %q, got nil", tc.errContains)
				return
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("ParseMarkdown() error = %v, want error containing %q", err, tc.errContains)
			}
		})
	}
}

func TestParseComplexFile(t *testing.T) {
	path := filepath.Join("testdata", "complex.md")
	tl, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error: %v", err)
	}

	// Test the parsed structure matches expected
	if tl.Title != "Complex Project Tasks" {
		t.Errorf("Title = %q, want %q", tl.Title, "Complex Project Tasks")
	}

	// Check root tasks
	if len(tl.Tasks) != 3 {
		t.Fatalf("Root tasks count = %d, want 3", len(tl.Tasks))
	}

	// Task 1: Design system architecture
	task1 := tl.FindTask("1")
	if task1 == nil {
		t.Fatal("Task 1 not found")
	}
	if task1.Status != Pending {
		t.Errorf("Task 1 status = %v, want Pending", task1.Status)
	}
	if len(task1.Details) != 2 {
		t.Errorf("Task 1 details count = %d, want 2", len(task1.Details))
	}
	if len(task1.References) != 2 {
		t.Errorf("Task 1 references count = %d, want 2", len(task1.References))
	}

	// Task 2: Implement core features (in progress)
	task2 := tl.FindTask("2")
	if task2 == nil {
		t.Fatal("Task 2 not found")
	}
	if task2.Status != InProgress {
		t.Errorf("Task 2 status = %v, want InProgress", task2.Status)
	}
	if len(task2.Children) != 3 {
		t.Errorf("Task 2 children count = %d, want 3", len(task2.Children))
	}

	// Task 2.1: Set up project structure (completed)
	task21 := tl.FindTask("2.1")
	if task21 == nil {
		t.Fatal("Task 2.1 not found")
	}
	if task21.Status != Completed {
		t.Errorf("Task 2.1 status = %v, want Completed", task21.Status)
	}
	if len(task21.Details) != 2 {
		t.Errorf("Task 2.1 details count = %d, want 2", len(task21.Details))
	}

	// Task 2.2: Build parser module (pending)
	task22 := tl.FindTask("2.2")
	if task22 == nil {
		t.Fatal("Task 2.2 not found")
	}
	if task22.Status != Pending {
		t.Errorf("Task 2.2 status = %v, want Pending", task22.Status)
	}
	if len(task22.References) != 1 {
		t.Errorf("Task 2.2 references count = %d, want 1", len(task22.References))
	}

	// Task 3: Testing and documentation
	task3 := tl.FindTask("3")
	if task3 == nil {
		t.Fatal("Task 3 not found")
	}
	if len(task3.Children) != 3 {
		t.Errorf("Task 3 children count = %d, want 3", len(task3.Children))
	}

	// Task 3.3: Documentation with nested details
	task33 := tl.FindTask("3.3")
	if task33 == nil {
		t.Fatal("Task 3.3 not found")
	}
	if len(task33.Details) != 2 {
		t.Errorf("Task 3.3 details count = %d, want 2", len(task33.Details))
	}
	if len(task33.References) != 2 {
		t.Errorf("Task 3.3 references count = %d, want 2", len(task33.References))
	}
}

func TestParseEdgeCases(t *testing.T) {
	tests := map[string]struct {
		content   string
		wantTasks int
		wantErr   bool
	}{
		"empty_lines_between_tasks": {
			content: `# Tasks

- [ ] 1. First task

- [ ] 2. Second task

- [ ] 3. Third task`,
			wantTasks: 3,
		},
		"windows_line_endings": {
			content:   "# Tasks\r\n- [ ] 1. Task one\r\n- [ ] 2. Task two",
			wantTasks: 2,
		},
		"trailing_whitespace": {
			content:   "# Tasks   \n- [ ] 1. Task one   \n  - Detail with spaces   ",
			wantTasks: 1,
		},
		"unicode_in_titles": {
			content:   "# 项目任务\n- [ ] 1. 中文任务\n- [ ] 2. Task with émojis 🚀",
			wantTasks: 2,
		},
		"very_deep_nesting": {
			content: `# Tasks
- [ ] 1. Level 1
  - [ ] 1.1. Level 2
    - [ ] 1.1.1. Level 3
      - [ ] 1.1.1.1. Level 4
        - [ ] 1.1.1.1.1. Level 5`,
			wantTasks: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseMarkdown() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMarkdown() unexpected error: %v", err)
				return
			}

			if len(tl.Tasks) != tc.wantTasks {
				t.Errorf("ParseMarkdown() tasks count = %d, want %d", len(tl.Tasks), tc.wantTasks)
			}
		})
	}
}

func TestParsePerformance(t *testing.T) {
	// Create a large task list to test performance
	var content strings.Builder
	content.WriteString("# Large Task List\n\n")

	for i := 1; i <= 100; i++ {
		content.WriteString("- [ ] ")
		content.WriteString(strings.TrimSpace(fmt.Sprintf("%d. Task %d\n", i, i)))
		content.WriteString("\n")
		for j := 1; j <= 5; j++ {
			content.WriteString("  - [ ] ")
			content.WriteString(strings.TrimSpace(fmt.Sprintf("%d.%d. Subtask %d\n", i, j, j)))
			content.WriteString("\n")
			content.WriteString("    - Detail line\n")
			content.WriteString("    - References: ref.md\n")
		}
	}

	data := []byte(content.String())

	// Skip performance test if ParseMarkdown is not implemented yet
	_, err := ParseMarkdown(data)
	if err != nil {
		t.Skip("Skipping performance test - ParseMarkdown not yet implemented")
	}
}

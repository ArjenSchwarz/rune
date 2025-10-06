package task

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractPhaseMarkers(t *testing.T) {
	tests := map[string]struct {
		content      string
		wantMarkers  []PhaseMarker
		wantErr      bool
		errContains  string
		validateFunc func(t *testing.T, markers []PhaseMarker)
	}{
		"no_phases": {
			content: `# Project Tasks
- [ ] 1. First task
- [ ] 2. Second task`,
			wantMarkers: []PhaseMarker{},
		},
		"single_phase_at_start": {
			content: `# Project Tasks

## Planning

- [ ] 1. Define requirements
- [ ] 2. Create design`,
			wantMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
			},
		},
		"multiple_phases": {
			content: `# Project Tasks

## Planning

- [ ] 1. Define requirements

## Implementation

- [ ] 2. Write code
- [ ] 3. Write tests

## Testing

- [ ] 4. Run tests`,
			wantMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Implementation", AfterTaskID: "1"},
				{Name: "Testing", AfterTaskID: "3"},
			},
		},
		"phase_after_task": {
			content: `# Project Tasks

- [ ] 1. Initial task

## Development

- [ ] 2. Dev task`,
			wantMarkers: []PhaseMarker{
				{Name: "Development", AfterTaskID: "1"},
			},
		},
		"mixed_content_with_phases": {
			content: `# Project Tasks

- [ ] 1. Pre-phase task

## Phase One

- [ ] 2. Phase one task
  - [ ] 2.1. Subtask

## Phase Two

- [ ] 3. Phase two task

- [ ] 4. Post-phases task`,
			wantMarkers: []PhaseMarker{
				{Name: "Phase One", AfterTaskID: "1"},
				{Name: "Phase Two", AfterTaskID: "2"},
			},
			validateFunc: func(t *testing.T, markers []PhaseMarker) {
				// Verify that subtasks don't affect phase positioning
				if markers[1].AfterTaskID != "2" {
					t.Errorf("Phase Two should come after task 2, not its subtasks")
				}
			},
		},
		"empty_phases": {
			content: `# Project Tasks

## Empty Phase One

## Empty Phase Two

- [ ] 1. First task after empty phases

## Non-Empty Phase

- [ ] 2. Task in phase`,
			wantMarkers: []PhaseMarker{
				{Name: "Empty Phase One", AfterTaskID: ""},
				{Name: "Empty Phase Two", AfterTaskID: ""},
				{Name: "Non-Empty Phase", AfterTaskID: "1"},
			},
		},
		"duplicate_phase_names": {
			content: `# Project Tasks

## Development

- [ ] 1. First dev task

## Testing

- [ ] 2. Test task

## Development

- [ ] 3. Second dev task`,
			wantMarkers: []PhaseMarker{
				{Name: "Development", AfterTaskID: ""},
				{Name: "Testing", AfterTaskID: "1"},
				{Name: "Development", AfterTaskID: "2"},
			},
			validateFunc: func(t *testing.T, markers []PhaseMarker) {
				// Both Development phases should be preserved
				if len(markers) != 3 {
					t.Errorf("Expected 3 phase markers, got %d", len(markers))
				}
				if markers[0].Name != "Development" || markers[2].Name != "Development" {
					t.Errorf("Duplicate phase names should be preserved")
				}
			},
		},
		"phase_with_special_characters": {
			content: `# Tasks

## Phase-1: Planning & Design

- [ ] 1. Task one

## Phase 2 (Implementation)

- [ ] 2. Task two`,
			wantMarkers: []PhaseMarker{
				{Name: "Phase-1: Planning & Design", AfterTaskID: ""},
				{Name: "Phase 2 (Implementation)", AfterTaskID: "1"},
			},
		},
		"phase_headers_not_h2": {
			content: `# Project Tasks

### This is H3

- [ ] 1. Task one

# This is H1

- [ ] 2. Task two

#### This is H4

- [ ] 3. Task three`,
			wantMarkers: []PhaseMarker{},
			validateFunc: func(t *testing.T, markers []PhaseMarker) {
				// Only H2 headers should be recognized as phases
				if len(markers) != 0 {
					t.Errorf("Non-H2 headers should not be recognized as phases")
				}
			},
		},
		"phase_with_trailing_spaces": {
			content: "# Tasks\n\n## Phase Name   \n\n- [ ] 1. Task",
			wantMarkers: []PhaseMarker{
				{Name: "Phase Name", AfterTaskID: ""},
			},
			validateFunc: func(t *testing.T, markers []PhaseMarker) {
				// Trailing spaces should be trimmed
				if markers[0].Name != "Phase Name" {
					t.Errorf("Phase name should have trailing spaces trimmed, got %q", markers[0].Name)
				}
			},
		},
		"phase_with_nested_tasks": {
			content: `# Project

## Phase One

- [ ] 1. Parent task
  - [ ] 1.1. Child one
    - [ ] 1.1.1. Grandchild
  - [ ] 1.2. Child two

## Phase Two

- [ ] 2. Next parent`,
			wantMarkers: []PhaseMarker{
				{Name: "Phase One", AfterTaskID: ""},
				{Name: "Phase Two", AfterTaskID: "1"},
			},
			validateFunc: func(t *testing.T, markers []PhaseMarker) {
				// Phase should be positioned after the parent task, not its children
				if markers[1].AfterTaskID != "1" {
					t.Errorf("Phase Two should be after task 1, got after %q", markers[1].AfterTaskID)
				}
			},
		},
		"phase_header_with_markdown_formatting": {
			content: `# Tasks

## **Bold Phase**

- [ ] 1. Task one

## _Italic Phase_

- [ ] 2. Task two`,
			wantMarkers: []PhaseMarker{
				{Name: "**Bold Phase**", AfterTaskID: ""},
				{Name: "_Italic Phase_", AfterTaskID: "1"},
			},
			validateFunc: func(t *testing.T, markers []PhaseMarker) {
				// Markdown formatting should be preserved in phase names
				if markers[0].Name != "**Bold Phase**" {
					t.Errorf("Markdown formatting should be preserved, got %q", markers[0].Name)
				}
			},
		},
		"phases_between_tasks_without_gaps": {
			content: `# Tasks

- [ ] 1. Task one
## Phase One
- [ ] 2. Task two
## Phase Two
- [ ] 3. Task three`,
			wantMarkers: []PhaseMarker{
				{Name: "Phase One", AfterTaskID: "1"},
				{Name: "Phase Two", AfterTaskID: "2"},
			},
		},
		"phase_only_document": {
			content: `# Project Phases

## Planning

## Development  

## Testing

## Deployment`,
			wantMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Development", AfterTaskID: ""},
				{Name: "Testing", AfterTaskID: ""},
				{Name: "Deployment", AfterTaskID: ""},
			},
		},
		"phase_with_details": {
			content: `# Tasks

## Phase One

- [ ] 1. Task with details
  - Detail one
  - Detail two

## Phase Two

- [ ] 2. Another task`,
			wantMarkers: []PhaseMarker{
				{Name: "Phase One", AfterTaskID: ""},
				{Name: "Phase Two", AfterTaskID: "1"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			lines := strings.Split(tc.content, "\n")
			markers := ExtractPhaseMarkers(lines)

			if tc.wantErr {
				// For now, ExtractPhaseMarkers doesn't return errors
				// This is here for future extension if needed
				return
			}

			if !reflect.DeepEqual(markers, tc.wantMarkers) {
				t.Errorf("ExtractPhaseMarkers() got %d markers, want %d", len(markers), len(tc.wantMarkers))
				for i, m := range markers {
					if i < len(tc.wantMarkers) {
						if m != tc.wantMarkers[i] {
							t.Errorf("  marker[%d]: got {Name: %q, AfterTaskID: %q}, want {Name: %q, AfterTaskID: %q}",
								i, m.Name, m.AfterTaskID, tc.wantMarkers[i].Name, tc.wantMarkers[i].AfterTaskID)
						}
					} else {
						t.Errorf("  extra marker[%d]: {Name: %q, AfterTaskID: %q}", i, m.Name, m.AfterTaskID)
					}
				}
				for i := len(markers); i < len(tc.wantMarkers); i++ {
					t.Errorf("  missing marker[%d]: {Name: %q, AfterTaskID: %q}",
						i, tc.wantMarkers[i].Name, tc.wantMarkers[i].AfterTaskID)
				}
			}

			if tc.validateFunc != nil {
				tc.validateFunc(t, markers)
			}
		})
	}
}

func TestPhaseDetectionWithFrontMatter(t *testing.T) {
	content := `---
feature: task-phases
---

# Project Tasks

## Planning

- [ ] 1. Define requirements

## Implementation  

- [ ] 2. Write code`

	// First parse to remove front matter
	fm, remaining, err := ParseFrontMatter(content)
	if err != nil {
		t.Fatalf("ParseFrontMatter() error = %v", err)
	}
	if fm == nil {
		t.Fatalf("Expected front matter, got nil")
	}

	lines := strings.Split(remaining, "\n")
	markers := ExtractPhaseMarkers(lines)

	expected := []PhaseMarker{
		{Name: "Planning", AfterTaskID: ""},
		{Name: "Implementation", AfterTaskID: "1"},
	}

	if !reflect.DeepEqual(markers, expected) {
		t.Errorf("ExtractPhaseMarkers() after front matter = %+v, want %+v", markers, expected)
	}
}

func TestPhasePreservationDuringParsing(t *testing.T) {
	// This test verifies that phase headers are detected but not stored in the model
	content := `# Project

## Phase One

- [ ] 1. Task one

## Phase Two

- [ ] 2. Task two`

	taskList, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}

	// Verify tasks are parsed correctly
	if len(taskList.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(taskList.Tasks))
	}

	// Verify task IDs are sequential across phases
	if taskList.Tasks[0].ID != "1" {
		t.Errorf("First task ID = %q, want %q", taskList.Tasks[0].ID, "1")
	}
	if taskList.Tasks[1].ID != "2" {
		t.Errorf("Second task ID = %q, want %q", taskList.Tasks[1].ID, "2")
	}

	// Phase information should not be stored in tasks
	// (no phase field in Task struct to check)
}

func TestMixedContentWithPhasesAndNonPhasedTasks(t *testing.T) {
	content := `# Mixed Content

- [ ] 1. Non-phased task at start

## First Phase

- [ ] 2. Task in first phase
- [ ] 3. Another task in first phase

- [ ] 4. Non-phased task after phase

## Second Phase  

- [ ] 5. Task in second phase

- [ ] 6. Non-phased task at end`

	lines := strings.Split(content, "\n")
	markers := ExtractPhaseMarkers(lines)

	expected := []PhaseMarker{
		{Name: "First Phase", AfterTaskID: "1"},
		{Name: "Second Phase", AfterTaskID: "4"},
	}

	if !reflect.DeepEqual(markers, expected) {
		t.Errorf("ExtractPhaseMarkers() = %+v, want %+v", markers, expected)
		for i, m := range markers {
			t.Logf("  markers[%d] = {Name: %q, AfterTaskID: %q}", i, m.Name, m.AfterTaskID)
		}
		for i, e := range expected {
			t.Logf("  expected[%d] = {Name: %q, AfterTaskID: %q}", i, e.Name, e.AfterTaskID)
		}
	}
}

func TestPhaseHeaderExtractionRegex(t *testing.T) {
	tests := map[string]struct {
		line      string
		isPhase   bool
		phaseName string
	}{
		"valid_h2":              {"## Planning", true, "Planning"},
		"h2_with_spaces":        {"##  Planning  ", true, "Planning"},
		"h2_with_special_chars": {"## Phase-1: Setup & Config", true, "Phase-1: Setup & Config"},
		"h2_with_numbers":       {"## Phase 123", true, "Phase 123"},
		"h3_header":             {"### Not a phase", false, ""},
		"h1_header":             {"# Not a phase", false, ""},
		"h4_header":             {"#### Not a phase", false, ""},
		"not_a_header":          {"Just text", false, ""},
		"task_line":             {"- [ ] 1. Task", false, ""},
		"h2_no_space":           {"##NoSpace", false, ""},
		"h2_with_leading_space": {" ## Planning", false, ""},
		"h2_with_tab":           {"\t## Planning", false, ""},
		"multiple_hashes":       {"### ## Planning", false, ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			lines := []string{tc.line}
			markers := ExtractPhaseMarkers(lines)

			if tc.isPhase {
				if len(markers) != 1 {
					t.Errorf("Expected 1 phase marker, got %d", len(markers))
					return
				}
				if markers[0].Name != tc.phaseName {
					t.Errorf("Phase name = %q, want %q", markers[0].Name, tc.phaseName)
				}
			} else {
				if len(markers) != 0 {
					t.Errorf("Expected no phase markers, got %d", len(markers))
				}
			}
		})
	}
}

func TestGetTaskPhasePositional(t *testing.T) {
	tests := map[string]struct {
		content     string
		taskID      string
		wantPhase   string
		description string
	}{
		"task_in_first_phase": {
			content: `# Project

## Planning

- [ ] 1. Define requirements
- [ ] 2. Create design

## Implementation

- [ ] 3. Write code`,
			taskID:      "1",
			wantPhase:   "Planning",
			description: "Task in first phase should return phase name",
		},
		"task_in_second_phase": {
			content: `# Project

## Planning

- [ ] 1. Define requirements

## Implementation

- [ ] 2. Write code
- [ ] 3. Write tests`,
			taskID:      "2",
			wantPhase:   "Implementation",
			description: "Task in second phase should return correct phase name",
		},
		"task_before_any_phase": {
			content: `# Project

- [ ] 1. Pre-phase task

## Planning

- [ ] 2. Phase task`,
			taskID:      "1",
			wantPhase:   "",
			description: "Task before any phase should return empty string",
		},
		"task_remains_in_phase_until_new_phase": {
			content: `# Project

## Planning

- [ ] 1. Phase task
- [ ] 2. Still in Planning phase`,
			taskID:      "2",
			wantPhase:   "Planning",
			description: "Task remains in phase until new phase header appears",
		},
		"subtask_in_phase": {
			content: `# Project

## Development

- [ ] 1. Parent task
  - [ ] 1.1. Subtask
  - [ ] 1.2. Another subtask`,
			taskID:      "1.1",
			wantPhase:   "Development",
			description: "Subtask should inherit parent's phase",
		},
		"deeply_nested_subtask": {
			content: `# Project

## Testing

- [ ] 1. Test suite
  - [ ] 1.1. Unit tests
    - [ ] 1.1.1. Component tests`,
			taskID:      "1.1.1",
			wantPhase:   "Testing",
			description: "Deeply nested subtask should return correct phase",
		},
		"task_in_phase_with_special_chars": {
			content: `# Project

## Phase-1: Planning & Design

- [ ] 1. Task one`,
			taskID:      "1",
			wantPhase:   "Phase-1: Planning & Design",
			description: "Phase names with special characters should be preserved",
		},
		"task_between_empty_phases": {
			content: `# Project

## Empty Phase One

## Phase Two

- [ ] 1. Task in phase two

## Empty Phase Three`,
			taskID:      "1",
			wantPhase:   "Phase Two",
			description: "Task should belong to most recent phase header",
		},
		"nonexistent_task": {
			content: `# Project

## Planning

- [ ] 1. Existing task`,
			taskID:      "999",
			wantPhase:   "",
			description: "Nonexistent task should return empty string",
		},
		"task_in_duplicate_phase_first": {
			content: `# Project

## Development

- [ ] 1. First dev task

## Testing

- [ ] 2. Test task

## Development

- [ ] 3. Second dev task`,
			taskID:      "1",
			wantPhase:   "Development",
			description: "Task in first occurrence of duplicate phase",
		},
		"task_in_duplicate_phase_second": {
			content: `# Project

## Development

- [ ] 1. First dev task

## Testing

- [ ] 2. Test task

## Development

- [ ] 3. Second dev task`,
			taskID:      "3",
			wantPhase:   "Development",
			description: "Task in second occurrence of duplicate phase should still return phase name",
		},
		"document_with_no_phases": {
			content: `# Project

- [ ] 1. Task one
- [ ] 2. Task two`,
			taskID:      "1",
			wantPhase:   "",
			description: "Document without phases should return empty string",
		},
		"task_after_multiple_phases": {
			content: `# Project

## Phase One

- [ ] 1. Task one

## Phase Two

- [ ] 2. Task two

## Phase Three

- [ ] 3. Task three`,
			taskID:      "3",
			wantPhase:   "Phase Three",
			description: "Task in last phase should be correctly identified",
		},
		"empty_task_id": {
			content: `# Project

## Planning

- [ ] 1. Task`,
			taskID:      "",
			wantPhase:   "",
			description: "Empty task ID should return empty string",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotPhase := getTaskPhase(tc.taskID, []byte(tc.content))

			if gotPhase != tc.wantPhase {
				t.Errorf("%s\ngetTaskPhase(%q) = %q, want %q",
					tc.description, tc.taskID, gotPhase, tc.wantPhase)
			}
		})
	}
}

func TestAddPhase(t *testing.T) {
	tests := map[string]struct {
		phaseName   string
		wantLine    string
		description string
	}{
		"simple_phase_name": {
			phaseName:   "Planning",
			wantLine:    "## Planning\n",
			description: "Simple phase name should be formatted as H2",
		},
		"phase_with_spaces": {
			phaseName:   "Phase One",
			wantLine:    "## Phase One\n",
			description: "Phase with spaces should preserve spaces",
		},
		"phase_with_special_chars": {
			phaseName:   "Phase-1: Planning & Design",
			wantLine:    "## Phase-1: Planning & Design\n",
			description: "Special characters should be preserved",
		},
		"phase_with_numbers": {
			phaseName:   "Phase 123",
			wantLine:    "## Phase 123\n",
			description: "Numbers in phase name should be preserved",
		},
		"phase_with_parentheses": {
			phaseName:   "Implementation (Backend)",
			wantLine:    "## Implementation (Backend)\n",
			description: "Parentheses should be preserved",
		},
		"phase_with_unicode": {
			phaseName:   "Planning 📋",
			wantLine:    "## Planning 📋\n",
			description: "Unicode characters should be preserved",
		},
		"empty_phase_name": {
			phaseName:   "",
			wantLine:    "## \n",
			description: "Empty phase name should still create valid H2 header",
		},
		"phase_with_leading_trailing_spaces": {
			phaseName:   "  Phase Name  ",
			wantLine:    "##   Phase Name  \n",
			description: "Leading/trailing spaces should be preserved (no trimming)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotLine := addPhase(tc.phaseName)

			if gotLine != tc.wantLine {
				t.Errorf("%s\naddPhase(%q) = %q, want %q",
					tc.description, tc.phaseName, gotLine, tc.wantLine)
			}
		})
	}
}

func TestGetNextPhaseTasks(t *testing.T) {
	tests := map[string]struct {
		content       string
		wantPhaseName string
		wantTaskCount int
		validateFunc  func(t *testing.T, tasks []Task, phaseName string)
		description   string
	}{
		"first_phase_with_pending": {
			content: `# Project

## Planning

- [ ] 1. Define requirements
- [ ] 2. Create design

## Implementation

- [ ] 3. Write code`,
			wantPhaseName: "Planning",
			wantTaskCount: 2,
			description:   "Should return all pending tasks from first phase",
		},
		"skip_completed_phase": {
			content: `# Project

## Planning

- [x] 1. Define requirements
- [x] 2. Create design

## Implementation

- [ ] 3. Write code
- [ ] 4. Write tests`,
			wantPhaseName: "Implementation",
			wantTaskCount: 2,
			description:   "Should skip completed phase and return next phase with pending tasks",
		},
		"phase_with_mixed_status": {
			content: `# Project

## Development

- [x] 1. Setup project
- [ ] 2. Write code
- [ ] 3. Write tests
- [x] 4. Review`,
			wantPhaseName: "Development",
			wantTaskCount: 2,
			validateFunc: func(t *testing.T, tasks []Task, phaseName string) {
				if len(tasks) != 2 {
					t.Errorf("Expected 2 pending tasks, got %d", len(tasks))
					return
				}
				if tasks[0].ID != "2" {
					t.Errorf("First pending task ID = %q, want %q", tasks[0].ID, "2")
				}
				if tasks[1].ID != "3" {
					t.Errorf("Second pending task ID = %q, want %q", tasks[1].ID, "3")
				}
			},
			description: "Should return only pending tasks from phase with mixed status",
		},
		"no_pending_tasks": {
			content: `# Project

## Planning

- [x] 1. Define requirements

## Implementation

- [x] 2. Write code`,
			wantPhaseName: "",
			wantTaskCount: 0,
			description:   "Should return empty when no pending tasks exist",
		},
		"tasks_before_phases": {
			content: `# Project

- [ ] 1. Pre-phase task

## Planning

- [x] 2. Completed task

## Implementation

- [ ] 3. Pending task`,
			wantPhaseName: "Implementation",
			wantTaskCount: 1,
			description:   "Should skip non-phased tasks and find first phase with pending",
		},
		"empty_phases_before_pending": {
			content: `# Project

## Empty Phase One

## Empty Phase Two

## Phase Three

- [ ] 1. First task`,
			wantPhaseName: "Phase Three",
			wantTaskCount: 1,
			description:   "Should skip empty phases and find first with pending tasks",
		},
		"phase_with_subtasks": {
			content: `# Project

## Development

- [ ] 1. Parent task
  - [ ] 1.1. Subtask one
  - [ ] 1.2. Subtask two`,
			wantPhaseName: "Development",
			wantTaskCount: 1,
			validateFunc: func(t *testing.T, tasks []Task, phaseName string) {
				if len(tasks) != 1 {
					t.Errorf("Expected 1 top-level task, got %d", len(tasks))
					return
				}
				if len(tasks[0].Children) != 2 {
					t.Errorf("Expected 2 subtasks, got %d", len(tasks[0].Children))
				}
			},
			description: "Should return parent task with its subtasks",
		},
		"document_without_phases": {
			content: `# Project

- [ ] 1. Task one
- [ ] 2. Task two`,
			wantPhaseName: "",
			wantTaskCount: 0,
			description:   "Should return empty for document without phases",
		},
		"phase_with_in_progress_tasks": {
			content: `# Project

## Current Sprint

- [-] 1. In progress task
- [ ] 2. Pending task
- [ ] 3. Another pending`,
			wantPhaseName: "Current Sprint",
			wantTaskCount: 3,
			validateFunc: func(t *testing.T, tasks []Task, phaseName string) {
				if len(tasks) != 3 {
					t.Errorf("Expected 3 tasks (in-progress + pending), got %d", len(tasks))
					return
				}
				// In-progress tasks should be included as they're not completed
				if tasks[0].Status != InProgress {
					t.Errorf("First task status = %v, want InProgress", tasks[0].Status)
				}
			},
			description: "Should include in-progress tasks as non-completed",
		},
		"multiple_phases_second_has_pending": {
			content: `# Project

## Phase One

- [x] 1. Completed

## Phase Two

- [x] 2. Also completed

## Phase Three

- [ ] 3. Pending task
- [ ] 4. Another pending`,
			wantPhaseName: "Phase Three",
			wantTaskCount: 2,
			description:   "Should return third phase when first two are completed",
		},
		"duplicate_phase_names": {
			content: `# Project

## Development

- [x] 1. Completed

## Testing

- [x] 2. Completed

## Development

- [ ] 3. Pending in second Development`,
			wantPhaseName: "Development",
			wantTaskCount: 1,
			description:   "Should handle duplicate phase names and find pending in second occurrence",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tasks, phaseName := getNextPhaseTasks([]byte(tc.content))

			if phaseName != tc.wantPhaseName {
				t.Errorf("%s\ngetNextPhaseTasks() phase = %q, want %q",
					tc.description, phaseName, tc.wantPhaseName)
			}

			if len(tasks) != tc.wantTaskCount {
				t.Errorf("%s\ngetNextPhaseTasks() returned %d tasks, want %d",
					tc.description, len(tasks), tc.wantTaskCount)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(t, tasks, phaseName)
			}
		})
	}
}

func TestFindPhasePosition(t *testing.T) {
	tests := map[string]struct {
		content       string
		phaseName     string
		wantFound     bool
		wantAfterTask string
		description   string
	}{
		"phase_at_start": {
			content: `# Project

## Planning

- [ ] 1. Task`,
			phaseName:     "Planning",
			wantFound:     true,
			wantAfterTask: "",
			description:   "Phase at document start should be found",
		},
		"phase_after_task": {
			content: `# Project

- [ ] 1. First task

## Development

- [ ] 2. Second task`,
			phaseName:     "Development",
			wantFound:     true,
			wantAfterTask: "1",
			description:   "Phase after task should indicate correct position",
		},
		"nonexistent_phase": {
			content: `# Project

## Planning

- [ ] 1. Task`,
			phaseName:     "Implementation",
			wantFound:     false,
			wantAfterTask: "",
			description:   "Nonexistent phase should not be found",
		},
		"first_of_duplicate_phases": {
			content: `# Project

## Development

- [ ] 1. First task

## Development

- [ ] 2. Second task`,
			phaseName:     "Development",
			wantFound:     true,
			wantAfterTask: "",
			description:   "Should find first occurrence of duplicate phase names",
		},
		"case_sensitive_match": {
			content: `# Project

## planning

- [ ] 1. Task`,
			phaseName:     "Planning",
			wantFound:     false,
			wantAfterTask: "",
			description:   "Phase names should be case-sensitive",
		},
		"phase_with_special_characters": {
			content: `# Project

## Phase-1: Setup & Config

- [ ] 1. Task`,
			phaseName:     "Phase-1: Setup & Config",
			wantFound:     true,
			wantAfterTask: "",
			description:   "Should match phase with special characters exactly",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotFound, gotAfterTask := findPhasePosition(tc.phaseName, []byte(tc.content))

			if gotFound != tc.wantFound {
				t.Errorf("%s\nfindPhasePosition(%q) found = %v, want %v",
					tc.description, tc.phaseName, gotFound, tc.wantFound)
			}

			if gotAfterTask != tc.wantAfterTask {
				t.Errorf("%s\nfindPhasePosition(%q) afterTask = %q, want %q",
					tc.description, tc.phaseName, gotAfterTask, tc.wantAfterTask)
			}
		})
	}
}

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
			markers := extractPhaseMarkers(lines)

			if tc.wantErr {
				// For now, extractPhaseMarkers doesn't return errors
				// This is here for future extension if needed
				return
			}

			if !reflect.DeepEqual(markers, tc.wantMarkers) {
				t.Errorf("extractPhaseMarkers() got %d markers, want %d", len(markers), len(tc.wantMarkers))
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
	markers := extractPhaseMarkers(lines)

	expected := []PhaseMarker{
		{Name: "Planning", AfterTaskID: ""},
		{Name: "Implementation", AfterTaskID: "1"},
	}

	if !reflect.DeepEqual(markers, expected) {
		t.Errorf("extractPhaseMarkers() after front matter = %+v, want %+v", markers, expected)
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
	markers := extractPhaseMarkers(lines)

	expected := []PhaseMarker{
		{Name: "First Phase", AfterTaskID: "1"},
		{Name: "Second Phase", AfterTaskID: "4"},
	}

	if !reflect.DeepEqual(markers, expected) {
		t.Errorf("extractPhaseMarkers() = %+v, want %+v", markers, expected)
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
			markers := extractPhaseMarkers(lines)

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

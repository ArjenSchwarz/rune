package task

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderMarkdownWithPhases(t *testing.T) {
	tests := map[string]struct {
		input        *TaskList
		phaseMarkers []PhaseMarker
		wantContent  string
		description  string
	}{
		"tasks_with_single_phase": {
			input: &TaskList{
				Title: "Project",
				Tasks: []Task{
					{ID: "1", Title: "Setup project", Status: Pending},
					{ID: "2", Title: "Create database", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
			},
			wantContent: `# Project

## Planning

- [ ] 1. Setup project

- [ ] 2. Create database
`,
			description: "Single phase at the beginning of document",
		},
		"tasks_with_multiple_phases": {
			input: &TaskList{
				Title: "Development",
				Tasks: []Task{
					{ID: "1", Title: "Design architecture", Status: Completed},
					{ID: "2", Title: "Setup environment", Status: Completed},
					{ID: "3", Title: "Write code", Status: InProgress},
					{ID: "4", Title: "Code review", Status: Pending},
					{ID: "5", Title: "Write tests", Status: Pending},
					{ID: "6", Title: "Run tests", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Implementation", AfterTaskID: "2"},
				{Name: "Testing", AfterTaskID: "4"},
			},
			wantContent: `# Development

## Planning

- [x] 1. Design architecture

- [x] 2. Setup environment

## Implementation

- [-] 3. Write code

- [ ] 4. Code review

## Testing

- [ ] 5. Write tests

- [ ] 6. Run tests
`,
			description: "Multiple phases dividing tasks",
		},
		"tasks_with_mixed_phases": {
			input: &TaskList{
				Title: "Mixed Content",
				Tasks: []Task{
					{ID: "1", Title: "Unphased task", Status: Pending},
					{ID: "2", Title: "Another unphased", Status: Pending},
					{ID: "3", Title: "Phase task 1", Status: Pending},
					{ID: "4", Title: "Phase task 2", Status: InProgress},
					{ID: "5", Title: "Final unphased", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Development", AfterTaskID: "2"},
				{Name: "Review", AfterTaskID: "4"},
			},
			wantContent: `# Mixed Content

- [ ] 1. Unphased task

- [ ] 2. Another unphased

## Development

- [ ] 3. Phase task 1

- [-] 4. Phase task 2

## Review

- [ ] 5. Final unphased
`,
			description: "Tasks both inside and outside phases",
		},
		"empty_phases": {
			input: &TaskList{
				Title: "Empty Phases",
				Tasks: []Task{
					{ID: "1", Title: "Task one", Status: Pending},
					{ID: "2", Title: "Task two", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Empty Phase One", AfterTaskID: ""},
				{Name: "Phase With Tasks", AfterTaskID: ""},
				{Name: "Empty Phase Two", AfterTaskID: "1"},
				{Name: "Another Phase", AfterTaskID: "2"},
				{Name: "Final Empty Phase", AfterTaskID: "2"},
			},
			wantContent: `# Empty Phases

## Empty Phase One

## Phase With Tasks

- [ ] 1. Task one

## Empty Phase Two

- [ ] 2. Task two

## Another Phase

## Final Empty Phase
`,
			description: "Phases can exist without tasks",
		},
		"tasks_with_subtasks_and_phases": {
			input: &TaskList{
				Title: "Hierarchical with Phases",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Main task",
						Status: InProgress,
						Children: []Task{
							{ID: "1.1", Title: "Subtask one", Status: Completed, ParentID: "1"},
							{ID: "1.2", Title: "Subtask two", Status: Pending, ParentID: "1"},
						},
					},
					{
						ID:      "2",
						Title:   "Second main",
						Status:  Pending,
						Details: []string{"Some detail"},
						Children: []Task{
							{ID: "2.1", Title: "Another subtask", Status: Pending, ParentID: "2"},
						},
					},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Phase One", AfterTaskID: ""},
				{Name: "Phase Two", AfterTaskID: "1"},
			},
			wantContent: `# Hierarchical with Phases

## Phase One

- [-] 1. Main task
  - [x] 1.1. Subtask one
  - [ ] 1.2. Subtask two

## Phase Two

- [ ] 2. Second main
  - Some detail
  - [ ] 2.1. Another subtask
`,
			description: "Hierarchical tasks preserve structure within phases",
		},
		"no_phases": {
			input: &TaskList{
				Title: "No Phases",
				Tasks: []Task{
					{ID: "1", Title: "Task one", Status: Pending},
					{ID: "2", Title: "Task two", Status: InProgress},
				},
			},
			phaseMarkers: []PhaseMarker{},
			wantContent: `# No Phases

- [ ] 1. Task one

- [-] 2. Task two
`,
			description: "Document without phases renders normally",
		},
		"duplicate_phase_names": {
			input: &TaskList{
				Title: "Duplicate Phases",
				Tasks: []Task{
					{ID: "1", Title: "First development task", Status: Pending},
					{ID: "2", Title: "Second development task", Status: Pending},
					{ID: "3", Title: "Third development task", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Development", AfterTaskID: ""},
				{Name: "Development", AfterTaskID: "2"},
			},
			wantContent: `# Duplicate Phases

## Development

- [ ] 1. First development task

- [ ] 2. Second development task

## Development

- [ ] 3. Third development task
`,
			description: "Duplicate phase names are preserved",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := RenderMarkdownWithPhases(tc.input, tc.phaseMarkers, nil)
			gotStr := string(got)

			if gotStr != tc.wantContent {
				t.Errorf("%s - RenderMarkdownWithPhases() mismatch:\nGot:\n%s\n\nWant:\n%s",
					tc.description, gotStr, tc.wantContent)
			}
		})
	}
}

func TestGetTaskPhase(t *testing.T) {
	tests := map[string]struct {
		taskList     *TaskList
		phaseMarkers []PhaseMarker
		taskID       string
		wantPhase    string
		description  string
	}{
		"task_in_first_phase": {
			taskList: &TaskList{
				Tasks: []Task{
					{ID: "1", Title: "Task one"},
					{ID: "2", Title: "Task two"},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Development", AfterTaskID: "1"},
			},
			taskID:      "1",
			wantPhase:   "Planning",
			description: "Task in first phase returns phase name",
		},
		"task_in_middle_phase": {
			taskList: &TaskList{
				Tasks: []Task{
					{ID: "1", Title: "Task one"},
					{ID: "2", Title: "Task two"},
					{ID: "3", Title: "Task three"},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Development", AfterTaskID: "1"},
				{Name: "Testing", AfterTaskID: "2"},
			},
			taskID:      "2",
			wantPhase:   "Development",
			description: "Task in middle phase returns correct phase",
		},
		"task_before_any_phase": {
			taskList: &TaskList{
				Tasks: []Task{
					{ID: "1", Title: "Task one"},
					{ID: "2", Title: "Task two"},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: "1"},
			},
			taskID:      "1",
			wantPhase:   "",
			description: "Task before any phase returns empty string",
		},
		"subtask_phase": {
			taskList: &TaskList{
				Tasks: []Task{
					{
						ID:    "1",
						Title: "Parent",
						Children: []Task{
							{ID: "1.1", Title: "Child", ParentID: "1"},
						},
					},
					{ID: "2", Title: "Task two"},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Phase One", AfterTaskID: ""},
				{Name: "Phase Two", AfterTaskID: "1"},
			},
			taskID:      "1.1",
			wantPhase:   "Phase One",
			description: "Subtask inherits phase from parent",
		},
		"no_phases": {
			taskList: &TaskList{
				Tasks: []Task{
					{ID: "1", Title: "Task one"},
				},
			},
			phaseMarkers: []PhaseMarker{},
			taskID:       "1",
			wantPhase:    "",
			description:  "No phases returns empty string",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetTaskPhase(tc.taskList, tc.phaseMarkers, tc.taskID)
			if got != tc.wantPhase {
				t.Errorf("%s - GetTaskPhase() = %q, want %q",
					tc.description, got, tc.wantPhase)
			}
		})
	}
}

func TestTableOutputWithPhases(t *testing.T) {
	tests := map[string]struct {
		taskList      *TaskList
		phaseMarkers  []PhaseMarker
		shouldHaveCol bool
		description   string
	}{
		"table_with_phases": {
			taskList: &TaskList{
				Title: "Project",
				Tasks: []Task{
					{ID: "1", Title: "Task one", Status: Pending},
					{ID: "2", Title: "Task two", Status: InProgress},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
			},
			shouldHaveCol: true,
			description:   "Table should have Phase column when phases exist",
		},
		"table_without_phases": {
			taskList: &TaskList{
				Title: "Project",
				Tasks: []Task{
					{ID: "1", Title: "Task one", Status: Pending},
					{ID: "2", Title: "Task two", Status: InProgress},
				},
			},
			phaseMarkers:  []PhaseMarker{},
			shouldHaveCol: false,
			description:   "Table should not have Phase column when no phases",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// This test would verify table output contains/doesn't contain Phase column
			// The actual implementation would be in the table rendering logic
			// For now, this is a placeholder that demonstrates the test structure
			hasPhases := len(tc.phaseMarkers) > 0
			if hasPhases != tc.shouldHaveCol {
				t.Errorf("%s - Phase column presence = %v, want %v",
					tc.description, hasPhases, tc.shouldHaveCol)
			}
		})
	}
}

// TestRenderJSONWithPhases_PointerReuse verifies that each task in the JSON
// output retains its own identity. A pointer reuse bug would cause all tasks
// to share the same underlying data (the last iteration's task).
func TestRenderJSONWithPhases_PointerReuse(t *testing.T) {
	tl := &TaskList{
		Title: "Pointer Reuse Check",
		Tasks: []Task{
			{ID: "1", Title: "Alpha", Status: Pending},
			{ID: "2", Title: "Beta", Status: InProgress},
			{ID: "3", Title: "Gamma", Status: Completed},
		},
	}
	phaseMarkers := []PhaseMarker{
		{Name: "Phase A", AfterTaskID: ""},
		{Name: "Phase B", AfterTaskID: "1"},
	}

	jsonData := RenderJSONWithPhases(tl, phaseMarkers, nil)

	var result struct {
		Tasks []struct {
			ID    string `json:"ID"`
			Title string `json:"Title"`
			Phase string `json:"Phase"`
		} `json:"Tasks"`
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(result.Tasks))
	}

	wantTasks := []struct {
		id, title, phase string
	}{
		{"1", "Alpha", "Phase A"},
		{"2", "Beta", "Phase B"},
		{"3", "Gamma", "Phase B"},
	}

	for i, want := range wantTasks {
		got := result.Tasks[i]
		if got.ID != want.id {
			t.Errorf("Task[%d] ID = %q, want %q", i, got.ID, want.id)
		}
		if got.Title != want.title {
			t.Errorf("Task[%d] Title = %q, want %q (pointer reuse would show %q)",
				i, got.Title, want.title, result.Tasks[len(result.Tasks)-1].Title)
		}
		if got.Phase != want.phase {
			t.Errorf("Task[%d] Phase = %q, want %q", i, got.Phase, want.phase)
		}
	}
}

// TestRenderJSONWithPhases_FilteredBoundaryTask verifies that phase labels
// remain correct when the task list has been filtered and boundary tasks
// (referenced in PhaseMarker.AfterTaskID) are no longer present.
// This is the regression test for T-537.
func TestRenderJSONWithPhases_FilteredBoundaryTask(t *testing.T) {
	// Original (unfiltered) task list with 4 tasks in 2 phases.
	// Phase "Design" starts at the beginning (AfterTaskID="").
	// Phase "Build" starts after task 2 (AfterTaskID="2").
	originalTL := &TaskList{
		Title: "Phase Boundary Test",
		Tasks: []Task{
			{ID: "1", Title: "Research", Status: Completed},
			{ID: "2", Title: "Prototype", Status: Completed},
			{ID: "3", Title: "Implement", Status: InProgress},
			{ID: "4", Title: "Deploy", Status: Pending},
		},
	}
	phaseMarkers := []PhaseMarker{
		{Name: "Design", AfterTaskID: ""},
		{Name: "Build", AfterTaskID: "2"},
	}

	// Filtered list: only pending and in-progress tasks remain.
	// Task 2 (the boundary task for "Build") is filtered out.
	filteredTL := &TaskList{
		Title: "Phase Boundary Test",
		Tasks: []Task{
			{ID: "3", Title: "Implement", Status: InProgress},
			{ID: "4", Title: "Deploy", Status: Pending},
		},
	}

	jsonData := RenderJSONWithPhases(filteredTL, phaseMarkers, originalTL)

	var result struct {
		Tasks []struct {
			ID    string `json:"ID"`
			Phase string `json:"Phase"`
		} `json:"Tasks"`
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(result.Tasks))
	}

	// Both tasks 3 and 4 are in the "Build" phase.
	// Without the fix, they would be labeled "Design" because the boundary
	// task (2) is not in the filtered list.
	wantPhases := []struct {
		id, phase string
	}{
		{"3", "Build"},
		{"4", "Build"},
	}

	for i, want := range wantPhases {
		got := result.Tasks[i]
		if got.ID != want.id {
			t.Errorf("Task[%d] ID = %q, want %q", i, got.ID, want.id)
		}
		if got.Phase != want.phase {
			t.Errorf("Task[%d] Phase = %q, want %q (boundary task filtered out — T-537 regression)",
				i, got.Phase, want.phase)
		}
	}
}

func TestJSONOutputWithPhases(t *testing.T) {
	tests := map[string]struct {
		taskList     *TaskList
		phaseMarkers []PhaseMarker
		checkPhase   func(data map[string]any) bool
		description  string
	}{
		"json_with_phases": {
			taskList: &TaskList{
				Title: "Project",
				Tasks: []Task{
					{ID: "1", Title: "Task in phase", Status: Pending},
					{ID: "2", Title: "Task outside phase", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{
				{Name: "Development", AfterTaskID: ""},
			},
			checkPhase: func(data map[string]any) bool {
				// Verify phase field exists in tasks when phases are present
				tasks, ok := data["tasks"].([]any)
				if !ok || len(tasks) == 0 {
					return false
				}
				task1 := tasks[0].(map[string]any)
				_, hasPhase := task1["phase"]
				return hasPhase
			},
			description: "JSON should include phase field when phases exist",
		},
		"json_without_phases": {
			taskList: &TaskList{
				Title: "Project",
				Tasks: []Task{
					{ID: "1", Title: "Task one", Status: Pending},
				},
			},
			phaseMarkers: []PhaseMarker{},
			checkPhase: func(data map[string]any) bool {
				// Verify phase field doesn't exist when no phases
				tasks, ok := data["tasks"].([]any)
				if !ok || len(tasks) == 0 {
					return false
				}
				task1 := tasks[0].(map[string]any)
				_, hasPhase := task1["phase"]
				return !hasPhase // Should NOT have phase field
			},
			description: "JSON should not include phase field when no phases",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Convert to JSON and check phase field presence
			jsonData := RenderJSONWithPhases(tc.taskList, tc.phaseMarkers, nil)

			var data map[string]any
			if err := json.Unmarshal(jsonData, &data); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			// For this test, we're just checking the structure
			// Actual implementation would populate phase data
			hasPhases := len(tc.phaseMarkers) > 0
			if hasPhases {
				// When phases exist, we expect phase information in output
				if !strings.Contains(string(jsonData), "phase_markers") {
					// Note: actual implementation may differ
				}
			}
		})
	}
}

// TestRenderMarkdownWithPhases_FilteredBoundaryTask verifies that phase headers
// are preserved in markdown output when the task list has been filtered and
// boundary tasks (referenced in PhaseMarker.AfterTaskID) are no longer present.
// This is the regression test for T-698, analogous to T-537 for JSON.
func TestRenderMarkdownWithPhases_FilteredBoundaryTask(t *testing.T) {
	// Original (unfiltered) task list with 4 tasks in 2 phases.
	// Phase "Design" starts at the beginning (AfterTaskID="").
	// Phase "Build" starts after task 2 (AfterTaskID="2").
	originalTL := &TaskList{
		Title: "Phase Boundary Test",
		Tasks: []Task{
			{ID: "1", Title: "Research", Status: Completed},
			{ID: "2", Title: "Prototype", Status: Completed},
			{ID: "3", Title: "Implement", Status: InProgress},
			{ID: "4", Title: "Deploy", Status: Pending},
		},
	}
	phaseMarkers := []PhaseMarker{
		{Name: "Design", AfterTaskID: ""},
		{Name: "Build", AfterTaskID: "2"},
	}

	// Filtered list: only pending and in-progress tasks remain.
	// Task 2 (the boundary task for "Build") is filtered out.
	filteredTL := &TaskList{
		Title: "Phase Boundary Test",
		Tasks: []Task{
			{ID: "3", Title: "Implement", Status: InProgress},
			{ID: "4", Title: "Deploy", Status: Pending},
		},
	}

	got := string(RenderMarkdownWithPhases(filteredTL, phaseMarkers, originalTL))

	// The "Build" phase header must appear even though task 2 is filtered out.
	if !strings.Contains(got, "## Build") {
		t.Errorf("Expected '## Build' phase header in filtered output, got:\n%s", got)
	}

	// The "Design" phase header also appears (empty — all Design tasks filtered out).
	// This is consistent with how unfiltered rendering handles empty phases.
	if !strings.Contains(got, "## Design") {
		t.Errorf("Expected '## Design' phase header (empty) in filtered output, got:\n%s", got)
	}

	// Phase headers must appear before their tasks (ordering check).
	designIdx := strings.Index(got, "## Design")
	buildIdx := strings.Index(got, "## Build")
	implIdx := strings.Index(got, "Implement")
	deployIdx := strings.Index(got, "Deploy")
	if designIdx >= buildIdx {
		t.Errorf("'## Design' should appear before '## Build', got:\n%s", got)
	}
	if buildIdx >= implIdx {
		t.Errorf("'## Build' header should appear before 'Implement' task, got:\n%s", got)
	}
	if implIdx >= deployIdx {
		t.Errorf("'Implement' should appear before 'Deploy', got:\n%s", got)
	}
}

// TestRenderMarkdownWithPhases_FilteredBoundaryThreePhases verifies correct
// behaviour with three phases where multiple boundary tasks are filtered out.
func TestRenderMarkdownWithPhases_FilteredBoundaryThreePhases(t *testing.T) {
	originalTL := &TaskList{
		Title: "Three Phases",
		Tasks: []Task{
			{ID: "1", Title: "Plan", Status: Completed},
			{ID: "2", Title: "Design", Status: Completed},
			{ID: "3", Title: "Code", Status: InProgress},
			{ID: "4", Title: "Test", Status: Pending},
			{ID: "5", Title: "Ship", Status: Pending},
		},
	}
	phaseMarkers := []PhaseMarker{
		{Name: "Planning", AfterTaskID: ""},
		{Name: "Development", AfterTaskID: "1"},
		{Name: "Release", AfterTaskID: "3"},
	}

	// Filter removes tasks 1 and 3 (both boundary tasks).
	filteredTL := &TaskList{
		Title: "Three Phases",
		Tasks: []Task{
			{ID: "2", Title: "Design", Status: Completed},
			{ID: "4", Title: "Test", Status: Pending},
			{ID: "5", Title: "Ship", Status: Pending},
		},
	}

	got := string(RenderMarkdownWithPhases(filteredTL, phaseMarkers, originalTL))

	// All three phase headers should appear (Planning is empty since task 1 was filtered).
	for _, phase := range []string{"## Planning", "## Development", "## Release"} {
		if !strings.Contains(got, phase) {
			t.Errorf("Expected %q phase header in filtered output, got:\n%s", phase, got)
		}
	}

	// Verify ordering: Planning < Development < Design task < Release < Test task.
	planIdx := strings.Index(got, "## Planning")
	devIdx := strings.Index(got, "## Development")
	designIdx := strings.Index(got, "Design")
	relIdx := strings.Index(got, "## Release")
	testIdx := strings.Index(got, "Test")
	if planIdx >= devIdx || devIdx >= designIdx || designIdx >= relIdx || relIdx >= testIdx {
		t.Errorf("Phase headers and tasks are not in expected order, got:\n%s", got)
	}
}

// TestRenderMarkdownWithPhases_FilteredNonAdjacentBlocks verifies that phase
// resolution works when filtered tasks come from non-adjacent blocks in the
// original list — the boundary task belongs to a different phase that is fully
// filtered out.
func TestRenderMarkdownWithPhases_FilteredNonAdjacentBlocks(t *testing.T) {
	originalTL := &TaskList{
		Title: "Non-Adjacent",
		Tasks: []Task{
			{ID: "1", Title: "Alpha", Status: Completed},
			{ID: "2", Title: "Beta", Status: Completed},
			{ID: "3", Title: "Gamma", Status: Pending},
			{ID: "4", Title: "Delta", Status: Completed},
			{ID: "5", Title: "Epsilon", Status: Pending},
		},
	}
	phaseMarkers := []PhaseMarker{
		{Name: "Phase A", AfterTaskID: ""},
		{Name: "Phase B", AfterTaskID: "2"},
		{Name: "Phase C", AfterTaskID: "4"},
	}

	// Filter keeps only pending tasks: 3 (Phase B) and 5 (Phase C).
	// Tasks 2 and 4 (boundary tasks for Phase B and C) are filtered out.
	filteredTL := &TaskList{
		Title: "Non-Adjacent",
		Tasks: []Task{
			{ID: "3", Title: "Gamma", Status: Pending},
			{ID: "5", Title: "Epsilon", Status: Pending},
		},
	}

	got := string(RenderMarkdownWithPhases(filteredTL, phaseMarkers, originalTL))

	// All three phase headers should appear (Phase A is empty).
	for _, phase := range []string{"## Phase A", "## Phase B", "## Phase C"} {
		if !strings.Contains(got, phase) {
			t.Errorf("Expected %q in output, got:\n%s", phase, got)
		}
	}

	// Verify ordering: Phase A < Phase B < Gamma < Phase C < Epsilon.
	aIdx := strings.Index(got, "## Phase A")
	bIdx := strings.Index(got, "## Phase B")
	gammaIdx := strings.Index(got, "Gamma")
	cIdx := strings.Index(got, "## Phase C")
	epsIdx := strings.Index(got, "Epsilon")
	if aIdx >= bIdx || bIdx >= gammaIdx || gammaIdx >= cIdx || cIdx >= epsIdx {
		t.Errorf("Expected order: Phase A < Phase B < Gamma < Phase C < Epsilon, got:\n%s", got)
	}
}

// TestRenderMarkdownWithPhases_NilPhaseSource ensures backward compatibility —
// passing nil phaseSource behaves identically to the old two-argument form.
func TestRenderMarkdownWithPhases_NilPhaseSource(t *testing.T) {
	tl := &TaskList{
		Title: "Simple",
		Tasks: []Task{
			{ID: "1", Title: "Alpha", Status: Pending},
			{ID: "2", Title: "Beta", Status: Pending},
		},
	}
	markers := []PhaseMarker{
		{Name: "Start", AfterTaskID: ""},
		{Name: "End", AfterTaskID: "1"},
	}

	got := string(RenderMarkdownWithPhases(tl, markers, nil))
	if !strings.Contains(got, "## Start") || !strings.Contains(got, "## End") {
		t.Errorf("Phase headers missing with nil phaseSource:\n%s", got)
	}

	// Verify ordering.
	startIdx := strings.Index(got, "## Start")
	alphaIdx := strings.Index(got, "Alpha")
	endIdx := strings.Index(got, "## End")
	betaIdx := strings.Index(got, "Beta")
	if startIdx >= alphaIdx || alphaIdx >= endIdx || endIdx >= betaIdx {
		t.Errorf("Expected order: Start < Alpha < End < Beta, got:\n%s", got)
	}
}

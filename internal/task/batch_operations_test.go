package task

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestExecuteBatch_MixedPhaseOperations(t *testing.T) {
	content := `# Test Tasks

## Planning

- [ ] 1. Plan task

## Implementation

- [ ] 2. Impl task`

	tempFile := "test_batch_mixed_phases.md"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	defer os.Remove(tempFile)

	tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Mix of phase and non-phase operations
	ops := []Operation{
		{
			Type:  "add",
			Title: "Task in Planning",
			Phase: "Planning",
		},
		{
			Type:  "add",
			Title: "Task without phase",
		},
		{
			Type:   "update",
			ID:     "1",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatchWithPhases(ops, false, phaseMarkers, tempFile)
	if err != nil {
		t.Fatalf("ExecuteBatchWithPhases failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	if response.Applied != 3 {
		t.Errorf("Expected 3 applied operations, got %d", response.Applied)
	}

	// Verify all operations succeeded
	tl, _, err = ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to re-parse file: %v", err)
	}

	phaseTask := findTaskByTitle(tl, "Task in Planning")
	if phaseTask == nil {
		t.Error("Task in Planning not found")
	}

	nonPhaseTask := findTaskByTitle(tl, "Task without phase")
	if nonPhaseTask == nil {
		t.Error("Task without phase not found")
	}

	task1 := tl.FindTask("1")
	if task1 == nil || task1.Status != Completed {
		t.Error("Task 1 should be completed")
	}
}

func TestExecuteBatch_PhaseDuplicateHandling(t *testing.T) {
	content := `# Test Tasks

## Development

- [ ] 1. First dev task

## Testing

- [ ] 2. Test task

## Development

- [ ] 3. Second dev section`

	tempFile := "test_batch_duplicate_phases.md"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	defer os.Remove(tempFile)

	tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Add task to "Development" phase - should go to first occurrence
	ops := []Operation{
		{
			Type:  "add",
			Title: "New dev task",
			Phase: "Development",
		},
	}

	response, err := tl.ExecuteBatchWithPhases(ops, false, phaseMarkers, tempFile)
	if err != nil {
		t.Fatalf("ExecuteBatchWithPhases failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Re-parse and verify task was added to first Development phase
	tl, _, err = ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to re-parse file: %v", err)
	}

	newTask := findTaskByTitle(tl, "New dev task")
	if newTask == nil {
		t.Fatal("New dev task not found")
	}

	// Task should be added after "First dev task" and before "Test task"
	if newTask.ID != "2" {
		t.Errorf("Expected task ID 2 (in first Development phase), got %s", newTask.ID)
	}
}

// TestExecuteBatch_MixedPhaseOperations tests batch with some operations having phases, some without
func TestExecuteBatch_PhaseAddOperation(t *testing.T) {
	tests := map[string]struct {
		setup       func() string
		ops         []Operation
		verify      func(*testing.T, *TaskList)
		description string
	}{
		"add task to existing phase": {
			setup: func() string {
				return `# Test Tasks

## Planning

- [ ] 1. Existing task

## Implementation

- [ ] 2. Another task`
			},
			ops: []Operation{
				{
					Type:  "add",
					Title: "New planning task",
					Phase: "Planning",
				},
			},
			verify: func(t *testing.T, tl *TaskList) {
				task := findTaskByTitle(tl, "New planning task")
				if task == nil {
					t.Error("New planning task not found")
					return
				}
				// Task should be added to Planning phase (after existing task, before Implementation phase)
				if task.ID != "2" {
					t.Errorf("Expected task ID 2, got %s", task.ID)
				}
			},
			description: "Task should be added to existing phase",
		},
		"add task to non-existent phase creates phase": {
			setup: func() string {
				return `# Test Tasks

- [ ] 1. Existing task`
			},
			ops: []Operation{
				{
					Type:  "add",
					Title: "New phase task",
					Phase: "New Phase",
				},
			},
			verify: func(t *testing.T, tl *TaskList) {
				task := findTaskByTitle(tl, "New phase task")
				if task == nil {
					t.Error("New phase task not found")
					return
				}
				// Task should be added after existing task
				if task.ID != "2" {
					t.Errorf("Expected task ID 2, got %s", task.ID)
				}
			},
			description: "Non-existent phase should be created and task added",
		},
		"add multiple tasks to same phase": {
			setup: func() string {
				return `# Test Tasks

## Development

- [ ] 1. First dev task`
			},
			ops: []Operation{
				{
					Type:  "add",
					Title: "Second dev task",
					Phase: "Development",
				},
				{
					Type:  "add",
					Title: "Third dev task",
					Phase: "Development",
				},
			},
			verify: func(t *testing.T, tl *TaskList) {
				if len(tl.Tasks) != 3 {
					t.Errorf("Expected 3 tasks, got %d", len(tl.Tasks))
				}
			},
			description: "Multiple tasks should be added to same phase",
		},
		"add task with parent in phase": {
			setup: func() string {
				return `# Test Tasks

## Planning

- [ ] 1. Parent task`
			},
			ops: []Operation{
				{
					Type:   "add",
					Title:  "Child task",
					Parent: "1",
					Phase:  "Planning",
				},
			},
			verify: func(t *testing.T, tl *TaskList) {
				parent := tl.FindTask("1")
				if parent == nil {
					t.Error("Parent task not found")
					return
				}
				if len(parent.Children) != 1 {
					t.Errorf("Expected 1 child, got %d", len(parent.Children))
					return
				}
				if parent.Children[0].Title != "Child task" {
					t.Errorf("Expected 'Child task', got '%s'", parent.Children[0].Title)
				}
			},
			description: "Task with parent should be added correctly in phase",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp file with initial content
			content := tc.setup()
			tempFile := fmt.Sprintf("test_batch_phase_%s.md", strings.ReplaceAll(name, " ", "_"))
			if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			defer os.Remove(tempFile)

			// Parse the file with phases
			tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Execute batch with phase operations
			response, err := tl.ExecuteBatchWithPhases(tc.ops, false, phaseMarkers, tempFile)
			if err != nil {
				t.Fatalf("ExecuteBatchWithPhases failed: %v", err)
			}

			if !response.Success {
				t.Fatalf("%s: Expected success, got errors: %v", tc.description, response.Errors)
			}

			// Re-parse to verify
			tl, _, err = ParseFileWithPhases(tempFile)
			if err != nil {
				t.Fatalf("Failed to re-parse file: %v", err)
			}

			// Run verification
			tc.verify(t, tl)
		})
	}
}

// TestExecuteBatch_PhaseDuplicateHandling tests duplicate phase name handling
func TestExecuteBatch_AutoCompleteComplexScenario(t *testing.T) {
	tl := NewTaskList("Complex Test")

	// Create a complex hierarchy
	// 1. Project (parent)
	//   1.1. Setup
	//     1.1.1. Install deps
	//     1.1.2. Configure DB
	//   1.2. Development
	//     1.2.1. Feature A
	//     1.2.2. Feature B
	//   1.3. Testing
	tl.AddTask("", "Project", "")
	tl.AddTask("1", "Setup", "")
	tl.AddTask("1.1", "Install deps", "")
	tl.AddTask("1.1", "Configure DB", "")
	tl.AddTask("1", "Development", "")
	tl.AddTask("1.2", "Feature A", "")
	tl.AddTask("1.2", "Feature B", "")
	tl.AddTask("1", "Testing", "")

	// Complete setup subtasks and one dev task
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.1.2",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.2.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.3",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Setup task should be auto-completed
	setup := tl.FindTask("1.1")
	if setup.Status != Completed {
		t.Errorf("Expected Setup task to be auto-completed, but status is %v", setup.Status)
	}

	// Development task should NOT be auto-completed (Feature B still pending)
	dev := tl.FindTask("1.2")
	if dev.Status == Completed {
		t.Error("Development task should not be auto-completed with pending subtask")
	}

	// Project should NOT be auto-completed (Development still has pending work)
	project := tl.FindTask("1")
	if project.Status == Completed {
		t.Error("Project should not be auto-completed with incomplete subtasks")
	}

	// Only Setup should be auto-completed
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
	if len(response.AutoCompleted) > 0 && response.AutoCompleted[0] != "1.1" {
		t.Errorf("Expected auto-completed task ID '1.1', got '%s'", response.AutoCompleted[0])
	}
}

func TestExecuteBatch_AutoCompleteDryRun(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up hierarchy
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")

	// Complete all children in dry-run mode
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, true)
	if err != nil {
		t.Fatalf("ExecuteBatch dry-run failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// In dry-run, original should be unchanged
	parent := tl.FindTask("1")
	if parent.Status == Completed {
		t.Error("Parent should not be modified in dry-run mode")
	}

	child1 := tl.FindTask("1.1")
	if child1.Status == Completed {
		t.Error("Child 1 should not be modified in dry-run mode")
	}

	// Preview should show auto-completed tasks
	if response.Preview == "" {
		t.Error("Expected preview content in dry-run")
	}

	// Auto-completed tasks should still be reported in dry-run
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task in dry-run, got %d", len(response.AutoCompleted))
	}
}

func TestExecuteBatch_AutoCompleteSameParentMultipleTimes(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up hierarchy
	tl.AddTask("", "Parent task", "")
	tl.AddTask("1", "Child 1", "")
	tl.AddTask("1", "Child 2", "")
	tl.AddTask("1", "Child 3", "")

	// Complete all children in batch - should trigger parent completion only once
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.2",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "1.3",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Parent should be completed
	parent := tl.FindTask("1")
	if parent.Status != Completed {
		t.Errorf("Expected parent to be auto-completed, but status is %v", parent.Status)
	}

	// Should only report parent as auto-completed once
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
	if len(response.AutoCompleted) > 0 && response.AutoCompleted[0] != "1" {
		t.Errorf("Expected auto-completed task ID '1', got '%s'", response.AutoCompleted[0])
	}
}

func TestExecuteBatch_AutoCompleteWithMixedOperations(t *testing.T) {
	tl := NewTaskList("Test Tasks")

	// Set up initial structure
	tl.AddTask("", "Task 1", "")
	tl.AddTask("1", "Task 1.1", "")
	tl.AddTask("", "Task 2", "")
	tl.AddTask("2", "Task 2.1", "")
	tl.AddTask("2", "Task 2.2", "")

	// Mixed operations including completions that trigger auto-complete
	ops := []Operation{
		{
			Type:   "update",
			ID:     "1.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "add",
			Parent: "2",
			Title:  "Task 2.3",
		},
		{
			Type:   "update",
			ID:     "2.1",
			Status: StatusPtr(Completed),
		},
		{
			Type:   "update",
			ID:     "2.2",
			Status: StatusPtr(Completed),
		},
	}

	response, err := tl.ExecuteBatch(ops, false)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Task 1 should be auto-completed (only child is complete)
	task1 := tl.FindTask("1")
	if task1.Status != Completed {
		t.Errorf("Expected Task 1 to be auto-completed, but status is %v", task1.Status)
	}

	// Task 2 should NOT be auto-completed (new child was added)
	task2 := tl.FindTask("2")
	if task2.Status == Completed {
		t.Error("Task 2 should not be auto-completed after adding a new child")
	}

	// Verify auto-completed count
	if len(response.AutoCompleted) != 1 {
		t.Errorf("Expected 1 auto-completed task, got %d", len(response.AutoCompleted))
	}
}

// TestExecuteBatchWithPhases_RemovePreservesPhases tests that batch remove operations
// correctly adjust phase markers to maintain phase boundaries after task removal.
func TestExecuteBatchWithPhases_RemovePreservesPhases(t *testing.T) {
	content := `# Test Tasks

## Planning

- [ ] 1. Define requirements
- [ ] 2. Create design

## Implementation

- [ ] 3. Write code
- [ ] 4. Write tests`

	tempFile := "test_batch_remove_phases.md"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	defer os.Remove(tempFile)

	tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Verify initial setup
	if len(tl.Tasks) != 4 {
		t.Fatalf("Expected 4 initial tasks, got %d", len(tl.Tasks))
	}
	if len(phaseMarkers) != 2 {
		t.Fatalf("Expected 2 phase markers, got %d", len(phaseMarkers))
	}

	// Remove task 1 (first task in Planning phase)
	ops := []Operation{
		{Type: "remove", ID: "1"},
	}

	response, err := tl.ExecuteBatchWithPhases(ops, false, phaseMarkers, tempFile)
	if err != nil {
		t.Fatalf("ExecuteBatchWithPhases failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	// Re-parse the file to verify structure
	tl, newPhaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to re-parse file: %v", err)
	}

	// Should have 3 tasks remaining, renumbered 1, 2, 3
	if len(tl.Tasks) != 3 {
		t.Errorf("Expected 3 tasks after removal, got %d", len(tl.Tasks))
	}

	// Verify task content after renumbering
	expectedTitles := []string{"Create design", "Write code", "Write tests"}
	for i, task := range tl.Tasks {
		if task.Title != expectedTitles[i] {
			t.Errorf("Task %d: expected title '%s', got '%s'", i+1, expectedTitles[i], task.Title)
		}
	}

	// Verify phases are still correct
	if len(newPhaseMarkers) != 2 {
		t.Errorf("Expected 2 phase markers after removal, got %d", len(newPhaseMarkers))
	}

	// Read file content to verify phase structure
	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	contentStr := string(fileContent)

	// Planning phase should still contain "Create design" (now task 1)
	if !strings.Contains(contentStr, "## Planning") {
		t.Error("Planning phase header missing")
	}
	if !strings.Contains(contentStr, "1. Create design") {
		t.Error("Task 1 (Create design) should be in Planning phase")
	}

	// Implementation phase should still contain "Write code" and "Write tests"
	if !strings.Contains(contentStr, "## Implementation") {
		t.Error("Implementation phase header missing")
	}
	if !strings.Contains(contentStr, "2. Write code") {
		t.Error("Task 2 (Write code) should be in Implementation phase")
	}
}

// TestExecuteBatchWithPhases_MultipleRemovesPreservesPhases tests that multiple batch removes
// correctly adjust phase markers using original task IDs.
func TestExecuteBatchWithPhases_MultipleRemovesPreservesPhases(t *testing.T) {
	content := `# Test Tasks

## Planning

- [ ] 1. Define requirements
- [ ] 2. Create design

## Implementation

- [ ] 3. Write code
- [ ] 4. Write tests

## Testing

- [ ] 5. Run unit tests
- [ ] 6. Run integration tests`

	tempFile := "test_batch_multi_remove_phases.md"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	defer os.Remove(tempFile)

	tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Verify initial setup
	if len(tl.Tasks) != 6 {
		t.Fatalf("Expected 6 initial tasks, got %d", len(tl.Tasks))
	}
	if len(phaseMarkers) != 3 {
		t.Fatalf("Expected 3 phase markers, got %d", len(phaseMarkers))
	}

	// Remove tasks 1, 3, and 5 using original IDs
	// With reverse-order processing: remove 5 first, then 3, then 1
	ops := []Operation{
		{Type: "remove", ID: "1"},
		{Type: "remove", ID: "3"},
		{Type: "remove", ID: "5"},
	}

	response, err := tl.ExecuteBatchWithPhases(ops, false, phaseMarkers, tempFile)
	if err != nil {
		t.Fatalf("ExecuteBatchWithPhases failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected success, got errors: %v", response.Errors)
	}

	if response.Applied != 3 {
		t.Errorf("Expected 3 applied operations, got %d", response.Applied)
	}

	// Re-parse the file to verify structure
	tl, newPhaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to re-parse file: %v", err)
	}

	// Should have 3 tasks remaining (2, 4, 6 → renumbered to 1, 2, 3)
	if len(tl.Tasks) != 3 {
		t.Errorf("Expected 3 tasks after removals, got %d", len(tl.Tasks))
	}

	// Verify task content after renumbering
	expectedTitles := []string{"Create design", "Write tests", "Run integration tests"}
	for i, task := range tl.Tasks {
		if task.Title != expectedTitles[i] {
			t.Errorf("Task %d: expected title '%s', got '%s'", i+1, expectedTitles[i], task.Title)
		}
	}

	// Verify phases are still correct (3 phases preserved)
	if len(newPhaseMarkers) != 3 {
		t.Errorf("Expected 3 phase markers after removals, got %d", len(newPhaseMarkers))
	}

	// Read file content to verify phase structure is intact
	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	contentStr := string(fileContent)

	// All three phases should still exist with correct tasks
	if !strings.Contains(contentStr, "## Planning") {
		t.Error("Planning phase header missing")
	}
	if !strings.Contains(contentStr, "## Implementation") {
		t.Error("Implementation phase header missing")
	}
	if !strings.Contains(contentStr, "## Testing") {
		t.Error("Testing phase header missing")
	}

	// Verify task positions in phases
	planningIdx := strings.Index(contentStr, "## Planning")
	implIdx := strings.Index(contentStr, "## Implementation")
	testingIdx := strings.Index(contentStr, "## Testing")

	task1Idx := strings.Index(contentStr, "1. Create design")
	task2Idx := strings.Index(contentStr, "2. Write tests")
	task3Idx := strings.Index(contentStr, "3. Run integration tests")

	// Task 1 should be between Planning and Implementation
	if task1Idx < planningIdx || task1Idx > implIdx {
		t.Error("Task 1 (Create design) should be in Planning phase")
	}

	// Task 2 should be between Implementation and Testing
	if task2Idx < implIdx || task2Idx > testingIdx {
		t.Error("Task 2 (Write tests) should be in Implementation phase")
	}

	// Task 3 should be after Testing
	if task3Idx < testingIdx {
		t.Error("Task 3 (Run integration tests) should be in Testing phase")
	}
}

// TestExecuteBatch_AddPhaseOperation tests the add-phase batch operation validation and execution
func TestExecuteBatch_AddPhaseOperation(t *testing.T) {
	tests := map[string]struct {
		setup       func() string
		ops         []Operation
		wantSuccess bool
		wantError   string
		verify      func(*testing.T, *TaskList, []PhaseMarker)
		description string
	}{
		"add single phase to empty file": {
			setup: func() string {
				return `# Test Tasks`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "Planning"},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				if len(markers) != 1 {
					t.Errorf("Expected 1 phase marker, got %d", len(markers))
					return
				}
				if markers[0].Name != "Planning" {
					t.Errorf("Expected phase name 'Planning', got '%s'", markers[0].Name)
				}
				if markers[0].AfterTaskID != "" {
					t.Errorf("Expected empty AfterTaskID for first phase, got '%s'", markers[0].AfterTaskID)
				}
			},
			description: "Single phase should be added to empty file",
		},
		"add phase to file with tasks": {
			setup: func() string {
				return `# Test Tasks

- [ ] 1. Existing task
- [ ] 2. Another task`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "New Phase"},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				if len(markers) != 1 {
					t.Errorf("Expected 1 phase marker, got %d", len(markers))
					return
				}
				if markers[0].Name != "New Phase" {
					t.Errorf("Expected phase name 'New Phase', got '%s'", markers[0].Name)
				}
				if markers[0].AfterTaskID != "2" {
					t.Errorf("Expected AfterTaskID '2', got '%s'", markers[0].AfterTaskID)
				}
			},
			description: "Phase should be added after last task",
		},
		"add phase then add task to it": {
			setup: func() string {
				return `# Test Tasks

- [ ] 1. Existing task`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "Development"},
				{Type: "add", Title: "New dev task", Phase: "Development"},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				if len(tl.Tasks) != 2 {
					t.Errorf("Expected 2 tasks, got %d", len(tl.Tasks))
					return
				}
				// The new task should be task 2
				task := findTaskByTitle(tl, "New dev task")
				if task == nil {
					t.Error("New dev task not found")
					return
				}
				if task.ID != "2" {
					t.Errorf("Expected task ID '2', got '%s'", task.ID)
				}
			},
			description: "Task should be added to newly created phase",
		},
		"add multiple phases in sequence": {
			setup: func() string {
				return `# Test Tasks

- [ ] 1. Initial task`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "Phase One"},
				{Type: "add-phase", Phase: "Phase Two"},
				{Type: "add-phase", Phase: "Phase Three"},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				if len(markers) != 3 {
					t.Errorf("Expected 3 phase markers, got %d", len(markers))
					return
				}
				expectedPhases := []string{"Phase One", "Phase Two", "Phase Three"}
				for i, expected := range expectedPhases {
					if markers[i].Name != expected {
						t.Errorf("Phase %d: expected '%s', got '%s'", i, expected, markers[i].Name)
					}
				}
			},
			description: "Multiple phases should be added in order",
		},
		"add phase with empty name fails": {
			setup: func() string {
				return `# Test Tasks`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: ""},
			},
			wantSuccess: false,
			wantError:   "phase name cannot be empty",
			description: "Empty phase name should fail validation",
		},
		"add phase with whitespace-only name fails": {
			setup: func() string {
				return `# Test Tasks`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "   "},
			},
			wantSuccess: false,
			wantError:   "phase name cannot be empty",
			// Note: For whitespace-only names, the error is returned directly
			// rather than through response.Errors due to early validation
			description: "Whitespace-only phase name should fail validation",
		},
		"add phase with surrounding whitespace trims name": {
			setup: func() string {
				return `# Test Tasks`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "  Trimmed Phase  "},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				if len(markers) != 1 {
					t.Errorf("Expected 1 phase marker, got %d", len(markers))
					return
				}
				// Phase name should be trimmed to match CLI behavior
				if markers[0].Name != "Trimmed Phase" {
					t.Errorf("Expected trimmed phase name 'Trimmed Phase', got '%s'", markers[0].Name)
				}
			},
			description: "Phase name with whitespace should be trimmed",
		},
		"add duplicate phase name succeeds": {
			setup: func() string {
				return `# Test Tasks

## Existing Phase

- [ ] 1. Task in existing phase`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "Existing Phase"},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				// Duplicate phases are allowed (matches CLI behavior)
				if len(markers) < 2 {
					t.Errorf("Expected at least 2 phase markers (original + new), got %d", len(markers))
				}
			},
			description: "Duplicate phase names should be allowed",
		},
		"add phase with existing phases preserves order": {
			setup: func() string {
				return `# Test Tasks

## Phase A

- [ ] 1. Task A

## Phase B

- [ ] 2. Task B`
			},
			ops: []Operation{
				{Type: "add-phase", Phase: "Phase C"},
			},
			wantSuccess: true,
			verify: func(t *testing.T, tl *TaskList, markers []PhaseMarker) {
				if len(markers) != 3 {
					t.Errorf("Expected 3 phase markers, got %d", len(markers))
					return
				}
				if markers[0].Name != "Phase A" {
					t.Errorf("Expected first phase 'Phase A', got '%s'", markers[0].Name)
				}
				if markers[1].Name != "Phase B" {
					t.Errorf("Expected second phase 'Phase B', got '%s'", markers[1].Name)
				}
				if markers[2].Name != "Phase C" {
					t.Errorf("Expected third phase 'Phase C', got '%s'", markers[2].Name)
				}
			},
			description: "New phase should be added at end, preserving existing order",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp file with initial content
			content := tc.setup()
			tempFile := fmt.Sprintf("test_batch_add_phase_%s.md", strings.ReplaceAll(name, " ", "_"))
			if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			defer os.Remove(tempFile)

			// Parse the file with phases
			tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Execute batch with phase operations
			response, err := tl.ExecuteBatchWithPhases(tc.ops, false, phaseMarkers, tempFile)

			if tc.wantSuccess {
				if err != nil {
					t.Fatalf("ExecuteBatchWithPhases failed: %v", err)
				}
				if !response.Success {
					t.Fatalf("%s: Expected success, got errors: %v", tc.description, response.Errors)
				}

				// Re-parse to verify
				tl, newMarkers, err := ParseFileWithPhases(tempFile)
				if err != nil {
					t.Fatalf("Failed to re-parse file: %v", err)
				}

				// Run verification
				if tc.verify != nil {
					tc.verify(t, tl, newMarkers)
				}
			} else {
				// Error can come back either as err or in response.Errors
				if err != nil {
					// Error returned directly
					if tc.wantError != "" && !strings.Contains(err.Error(), tc.wantError) {
						t.Errorf("Expected error containing '%s', got '%s'", tc.wantError, err.Error())
					}
					return
				}
				if response.Success {
					t.Fatalf("%s: Expected failure, but got success", tc.description)
				}
				if tc.wantError != "" {
					found := false
					for _, errMsg := range response.Errors {
						if strings.Contains(errMsg, tc.wantError) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got: %v", tc.wantError, response.Errors)
					}
				}
			}
		})
	}
}

// TestExecuteBatch_AddPhaseOperationDryRun tests dry-run mode for add-phase operations
func TestExecuteBatch_AddPhaseOperationDryRun(t *testing.T) {
	content := `# Test Tasks

- [ ] 1. Existing task`

	tempFile := "test_batch_add_phase_dry_run.md"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	defer os.Remove(tempFile)

	tl, phaseMarkers, err := ParseFileWithPhases(tempFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	ops := []Operation{
		{Type: "add-phase", Phase: "New Phase"},
		{Type: "add", Title: "Task in new phase", Phase: "New Phase"},
	}

	response, err := tl.ExecuteBatchWithPhases(ops, true, phaseMarkers, tempFile)
	if err != nil {
		t.Fatalf("ExecuteBatchWithPhases dry-run failed: %v", err)
	}

	if !response.Success {
		t.Fatalf("Expected dry-run success, got errors: %v", response.Errors)
	}

	// Preview should contain the new phase
	if !strings.Contains(response.Preview, "## New Phase") {
		t.Error("Expected preview to contain new phase header")
	}

	// Preview should contain the new task
	if !strings.Contains(response.Preview, "Task in new phase") {
		t.Error("Expected preview to contain new task")
	}

	// Original file should be unchanged
	originalContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if strings.Contains(string(originalContent), "New Phase") {
		t.Error("Original file should not be modified in dry-run mode")
	}
}

// TestValidateOperation_AddPhase tests the validateOperation function for add-phase operations
func TestValidateOperation_AddPhase(t *testing.T) {
	tests := map[string]struct {
		op        Operation
		wantError bool
		errMsg    string
	}{
		"valid phase name": {
			op:        Operation{Type: "add-phase", Phase: "Planning"},
			wantError: false,
		},
		"valid phase name with spaces": {
			op:        Operation{Type: "add-phase", Phase: "Development and Testing"},
			wantError: false,
		},
		"empty phase name": {
			op:        Operation{Type: "add-phase", Phase: ""},
			wantError: true,
			errMsg:    "phase name cannot be empty",
		},
		"whitespace-only phase name": {
			op:        Operation{Type: "add-phase", Phase: "   \t\n   "},
			wantError: true,
			errMsg:    "phase name cannot be empty",
		},
		"phase name with leading/trailing spaces": {
			op:        Operation{Type: "add-phase", Phase: "  Valid Phase  "},
			wantError: false,
		},
	}

	tl := NewTaskList("Test")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateOperation(tl, tc.op)
			if tc.wantError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.errMsg)
				} else if !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got '%s'", err.Error())
				}
			}
		})
	}
}

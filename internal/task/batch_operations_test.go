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

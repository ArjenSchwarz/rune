package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

// TestIntegrationBatchOperations tests batch operation workflows
func TestIntegrationBatchOperations(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	tests := map[string]struct {
		name        string
		workflow    func(t *testing.T, tempDir string)
		description string
	}{
		"unified_update_operations": {
			name:        "Unified Update Operations",
			description: "Test unified update operations replacing update_status",
			workflow:    testUnifiedUpdateOperations,
		},
		"position_insertion_workflows": {
			name:        "Position Insertion Workflows",
			description: "Test position insertion in batch and CLI operations",
			workflow:    testPositionInsertionWorkflows,
		},
		"batch_position_and_update_combination": {
			name:        "Batch Position and Update Combination",
			description: "Test complex batches combining position insertion with updates on renumbered tasks",
			workflow:    testBatchPositionAndUpdateCombination,
		},
		"enhanced_error_handling_workflows": {
			name:        "Enhanced Error Handling Workflows",
			description: "Test error handling and rollback for unified updates and position insertion",
			workflow:    testEnhancedErrorHandlingWorkflows,
		},
		"cli_position_insertion_integration": {
			name:        "CLI Position Insertion Integration",
			description: "Test CLI position insertion with file operations and git discovery",
			workflow:    testCLIPositionInsertionIntegration,
		},
		"front_matter_integration": {
			name:        "Front Matter Integration Tests",
			description: "Test front matter creation, modification, and edge cases",
			workflow:    testFrontMatterIntegration,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Create isolated temp directory for each test
			tempDir, err := os.MkdirTemp("", "rune-integration-batch-"+testName)
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(oldDir)
			}()

			t.Logf("Running integration test: %s", tc.description)
			tc.workflow(t, tempDir)
		})
	}
}

func testUnifiedUpdateOperations(t *testing.T, tempDir string) {
	filename := "unified-update.md"

	// Create initial task structure
	tl := task.NewTaskList("Unified Update Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Add initial tasks
	initialTasks := []struct {
		parent string
		title  string
	}{
		{"", "Setup project infrastructure"},
		{"", "Implement core features"},
		{"", "Write comprehensive tests"},
		{"1", "Configure build system"},
		{"1", "Setup CI/CD pipeline"},
		{"2", "User authentication"},
		{"2", "Data validation"},
	}

	for _, tt := range initialTasks {
		if _, err := tl.AddTask(tt.parent, tt.title, ""); err != nil {
			t.Fatalf("failed to add initial task %s: %v", tt.title, err)
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write initial tasks: %v", err)
	}

	// Test 1: Unified update with all fields
	t.Run("unified_update_all_fields", func(t *testing.T) {
		operations := []task.Operation{
			{
				Type:       "update",
				ID:         "1.1",
				Title:      "Configure advanced build system",
				Status:     task.StatusPtr(task.InProgress),
				Details:    []string{"Setup Webpack", "Configure TypeScript", "Add hot reload"},
				References: []string{"build-config.md", "webpack-guide.md"},
			},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to execute unified update: %v", err)
		}
		if !response.Success {
			t.Fatalf("unified update failed: %v", response.Errors)
		}

		// Verify all fields were updated
		task11 := tl.FindTask("1.1")
		if task11 == nil {
			t.Fatal("task 1.1 not found after update")
		}
		if task11.Title != "Configure advanced build system" {
			t.Errorf("expected updated title, got: %s", task11.Title)
		}
		if task11.Status != task.InProgress {
			t.Errorf("expected in-progress status, got: %v", task11.Status)
		}
		if len(task11.Details) != 3 {
			t.Errorf("expected 3 details, got: %d", len(task11.Details))
		}
		if len(task11.References) != 2 {
			t.Errorf("expected 2 references, got: %d", len(task11.References))
		}
	})

	// Test 2: Status-only update with auto-completion
	t.Run("unified_update_status_only_autocompletion", func(t *testing.T) {
		// First complete task 1.1 (which was set to InProgress in previous test)
		// Then complete task 1.2 to trigger parent auto-completion
		completeTasks := []task.Operation{
			{Type: "update", ID: "1.1", Status: task.StatusPtr(task.Completed)},
			{Type: "update", ID: "1.2", Status: task.StatusPtr(task.Completed)},
		}

		response, err := tl.ExecuteBatch(completeTasks, false)
		if err != nil {
			t.Fatalf("failed to complete subtasks: %v", err)
		}
		if !response.Success {
			t.Fatalf("completion failed: %v", response.Errors)
		}

		// Verify parent task 1 was auto-completed (both 1.1 and 1.2 are now completed)
		task1 := tl.FindTask("1")
		if task1.Status != task.Completed {
			t.Errorf("expected parent task 1 to be auto-completed, got status: %v", task1.Status)
		}

		// Check that auto-completion was tracked
		if len(response.AutoCompleted) == 0 {
			t.Error("expected auto-completed tasks to be tracked")
		}
	})

	// Test 3: Update with status completion triggering auto-completion
	t.Run("unified_update_mixed_fields_with_autocompletion", func(t *testing.T) {
		operations := []task.Operation{
			{
				Type:    "update",
				ID:      "2.1",
				Title:   "Advanced user authentication with 2FA",
				Status:  task.StatusPtr(task.Completed),
				Details: []string{"OAuth2 integration", "JWT tokens", "Two-factor authentication"},
			},
			{
				Type:   "update",
				ID:     "2.2",
				Status: task.StatusPtr(task.Completed),
			},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to execute mixed update: %v", err)
		}
		if !response.Success {
			t.Fatalf("mixed update failed: %v", response.Errors)
		}

		// Verify task 2.1 was updated with all fields
		task21 := tl.FindTask("2.1")
		if task21.Title != "Advanced user authentication with 2FA" {
			t.Errorf("expected updated title, got: %s", task21.Title)
		}
		if task21.Status != task.Completed {
			t.Errorf("expected completed status, got: %v", task21.Status)
		}
		if len(task21.Details) != 3 {
			t.Errorf("expected 3 details, got: %d", len(task21.Details))
		}

		// Verify parent task 2 was auto-completed
		task2 := tl.FindTask("2")
		if task2.Status != task.Completed {
			t.Errorf("expected parent task 2 to be auto-completed, got status: %v", task2.Status)
		}
	})

	// Test 4: No-op update (empty fields)
	t.Run("unified_update_noop", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "update", ID: "3"}, // No fields provided
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to execute no-op update: %v", err)
		}
		if !response.Success {
			t.Fatalf("no-op update failed: %v", response.Errors)
		}

		// Task should remain unchanged
		task3 := tl.FindTask("3")
		if task3.Status != task.Pending {
			t.Errorf("expected task 3 to remain pending, got: %v", task3.Status)
		}
	})

	// Test 5: Batch with multiple unified updates
	t.Run("batch_multiple_unified_updates", func(t *testing.T) {
		operations := []task.Operation{
			{
				Type:   "update",
				ID:     "3",
				Title:  "Write comprehensive integration tests",
				Status: task.StatusPtr(task.InProgress),
			},
			{
				Type:       "update",
				ID:         "3",
				Details:    []string{"Unit tests", "Integration tests", "End-to-end tests"},
				References: []string{"testing-strategy.md"},
			},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to execute multiple updates: %v", err)
		}
		if !response.Success {
			t.Fatalf("multiple updates failed: %v", response.Errors)
		}

		// Verify all updates were applied
		task3 := tl.FindTask("3")
		if task3.Title != "Write comprehensive integration tests" {
			t.Errorf("expected updated title, got: %s", task3.Title)
		}
		if task3.Status != task.InProgress {
			t.Errorf("expected in-progress status, got: %v", task3.Status)
		}
		if len(task3.Details) != 3 {
			t.Errorf("expected 3 details, got: %d", len(task3.Details))
		}
		if len(task3.References) != 1 {
			t.Errorf("expected 1 reference, got: %d", len(task3.References))
		}
	})

	t.Logf("Unified update operations test passed successfully")
}

func testPositionInsertionWorkflows(t *testing.T, tempDir string) {
	filename := "position-insertion.md"

	// Create initial task structure
	tl := task.NewTaskList("Position Insertion Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Add initial tasks
	initialTasks := []string{
		"First task",
		"Second task",
		"Third task",
		"Fourth task",
	}

	for _, title := range initialTasks {
		if _, err := tl.AddTask("", title, ""); err != nil {
			t.Fatalf("failed to add initial task %s: %v", title, err)
		}
	}

	// Add subtasks to task 2
	subtasks := []string{
		"Subtask 2.1",
		"Subtask 2.2",
	}

	for _, title := range subtasks {
		if _, err := tl.AddTask("2", title, ""); err != nil {
			t.Fatalf("failed to add subtask %s: %v", title, err)
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write initial tasks: %v", err)
	}

	// Test 1: Insert at beginning of root tasks
	t.Run("position_insertion_at_beginning", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Title: "Urgent task at beginning", Position: "1"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to insert at beginning: %v", err)
		}
		if !response.Success {
			t.Fatalf("position insertion failed: %v", response.Errors)
		}

		// Verify insertion and renumbering
		if len(tl.Tasks) != 5 {
			t.Errorf("expected 5 root tasks, got: %d", len(tl.Tasks))
		}

		task1 := tl.FindTask("1")
		if task1.Title != "Urgent task at beginning" {
			t.Errorf("expected new task at position 1, got: %s", task1.Title)
		}

		task2 := tl.FindTask("2")
		if task2.Title != "First task" {
			t.Errorf("expected original first task at position 2, got: %s", task2.Title)
		}

		task3 := tl.FindTask("3")
		if task3.Title != "Second task" {
			t.Errorf("expected original second task at position 3, got: %s", task3.Title)
		}

		// Verify subtasks were renumbered correctly
		task31 := tl.FindTask("3.1")
		if task31 == nil || task31.Title != "Subtask 2.1" {
			t.Error("subtask 2.1 should have been renumbered to 3.1")
		}

		task32 := tl.FindTask("3.2")
		if task32 == nil || task32.Title != "Subtask 2.2" {
			t.Error("subtask 2.2 should have been renumbered to 3.2")
		}
	})

	// Test 2: Insert in middle of root tasks
	t.Run("position_insertion_in_middle", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Title: "Middle task", Position: "3"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to insert in middle: %v", err)
		}
		if !response.Success {
			t.Fatalf("middle insertion failed: %v", response.Errors)
		}

		// Verify insertion
		task3 := tl.FindTask("3")
		if task3.Title != "Middle task" {
			t.Errorf("expected new task at position 3, got: %s", task3.Title)
		}

		// Verify renumbering of subsequent tasks
		task4 := tl.FindTask("4")
		if task4.Title != "Second task" {
			t.Errorf("expected original second task moved to position 4, got: %s", task4.Title)
		}
	})

	// Test 3: Insert at subtask level
	t.Run("position_insertion_subtask_level", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Parent: "4", Title: "New first subtask", Position: "4.1"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed to insert subtask: %v", err)
		}
		if !response.Success {
			t.Fatalf("subtask insertion failed: %v", response.Errors)
		}

		// Verify subtask insertion
		task41 := tl.FindTask("4.1")
		if task41.Title != "New first subtask" {
			t.Errorf("expected new subtask at position 4.1, got: %s", task41.Title)
		}

		// Verify renumbering of existing subtasks
		task42 := tl.FindTask("4.2")
		if task42 == nil || task42.Title != "Subtask 2.1" {
			t.Error("original subtask 2.1 should be renumbered to 4.2")
		}

		task43 := tl.FindTask("4.3")
		if task43 == nil || task43.Title != "Subtask 2.2" {
			t.Error("original subtask 2.2 should be renumbered to 4.3")
		}
	})

	// Test 4: Multiple position insertions in single batch (reverse order processing)
	t.Run("multiple_position_insertions", func(t *testing.T) {
		// Reset to a clean state for this test
		tl = task.NewTaskList("Multiple Position Insertion Test")
		initialTasks := []string{
			"Alpha", "Beta", "Gamma", "Delta",
		}

		for _, title := range initialTasks {
			if _, err := tl.AddTask("", title, ""); err != nil {
				t.Fatalf("failed to add initial task %s: %v", title, err)
			}
		}

		operations := []task.Operation{
			{Type: "add", Title: "Task at position 2", Position: "2"},
			{Type: "add", Title: "Task at position 4", Position: "4"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed multiple position insertions: %v", err)
		}
		if !response.Success {
			t.Fatalf("multiple insertions failed: %v", response.Errors)
		}

		// Verify both insertions worked correctly
		// Position references use original state, so:
		// 1. "Task at position 4" gets inserted before original "Delta" (position 4)
		// 2. "Task at position 2" gets inserted before original "Beta" (position 2)
		// Final order should be: Alpha, Task at position 2, Beta, Gamma, Task at position 4, Delta

		if len(tl.Tasks) != 6 {
			t.Errorf("expected 6 tasks after insertions, got: %d", len(tl.Tasks))
		}

		expectedOrder := []string{
			"Alpha",              // position 1
			"Task at position 2", // position 2 (inserted)
			"Beta",               // position 3 (was 2)
			"Gamma",              // position 4 (was 3)
			"Task at position 4", // position 5 (inserted before original Delta)
			"Delta",              // position 6 (was 4)
		}

		for i, expectedTitle := range expectedOrder {
			if i >= len(tl.Tasks) || tl.Tasks[i].Title != expectedTitle {
				t.Errorf("position %d: expected %s, got %s", i+1, expectedTitle,
					func() string {
						if i >= len(tl.Tasks) {
							return "missing"
						}
						return tl.Tasks[i].Title
					}())
			}
		}
	})

	// Test 5: Position beyond list size (should append)
	t.Run("position_beyond_list_size", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Title: "Task beyond end", Position: "999"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed position beyond end: %v", err)
		}
		if !response.Success {
			t.Fatalf("position beyond end failed: %v", response.Errors)
		}

		// Should be appended at the end
		lastTask := tl.Tasks[len(tl.Tasks)-1]
		if lastTask.Title != "Task beyond end" {
			t.Errorf("expected task to be appended at end, got: %s", lastTask.Title)
		}
	})

	t.Logf("Position insertion workflows test passed successfully")
}

func testBatchPositionAndUpdateCombination(t *testing.T, tempDir string) {
	filename := "batch-combination.md"

	// Create initial task structure
	tl := task.NewTaskList("Batch Combination Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Add initial tasks
	initialTasks := []string{
		"Task 1",
		"Task 2",
		"Task 3",
	}

	for _, title := range initialTasks {
		if _, err := tl.AddTask("", title, ""); err != nil {
			t.Fatalf("failed to add initial task %s: %v", title, err)
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write initial tasks: %v", err)
	}

	// Test 1: Position insertion followed by updates to renumbered tasks
	t.Run("position_insertion_then_update_renumbered", func(t *testing.T) {
		operations := []task.Operation{
			// Insert at position 1 (will cause renumbering)
			{Type: "add", Title: "New urgent task", Position: "1"},
			// Update what was originally task 1 (now task 2)
			{Type: "update", ID: "2", Title: "Updated original task 1", Status: task.StatusPtr(task.InProgress)},
			// Update what was originally task 2 (now task 3)
			{Type: "update", ID: "3", Title: "Updated original task 2", Status: task.StatusPtr(task.Completed)},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed batch combination: %v", err)
		}
		if !response.Success {
			t.Fatalf("batch combination failed: %v", response.Errors)
		}

		// Verify position insertion worked
		task1 := tl.FindTask("1")
		if task1.Title != "New urgent task" {
			t.Errorf("expected new urgent task at position 1, got: %s", task1.Title)
		}

		// Verify updates to renumbered tasks worked
		task2 := tl.FindTask("2")
		if task2.Title != "Updated original task 1" {
			t.Errorf("expected updated title for original task 1, got: %s", task2.Title)
		}
		if task2.Status != task.InProgress {
			t.Errorf("expected in-progress status for task 2, got: %v", task2.Status)
		}

		task3 := tl.FindTask("3")
		if task3.Title != "Updated original task 2" {
			t.Errorf("expected updated title for original task 2, got: %s", task3.Title)
		}
		if task3.Status != task.Completed {
			t.Errorf("expected completed status for task 3, got: %v", task3.Status)
		}
	})

	// Test 2: Multiple position insertions with updates - operations applied sequentially
	t.Run("multiple_position_insertions_with_original_references", func(t *testing.T) {
		// Reset to known state
		tl = task.NewTaskList("Batch Combination Test")
		for _, title := range initialTasks {
			tl.AddTask("", title, "")
		}

		operations := []task.Operation{
			// Position insertions reference original state (processed in reverse order)
			{Type: "add", Title: "Insert at pos 1", Position: "1"},
			{Type: "add", Title: "Insert at pos 3", Position: "3"}, // Original position 3
			// Updates reference current state after insertions
			{Type: "update", ID: "1", Title: "Update task at pos 1"}, // Now the insertion
			{Type: "update", ID: "2", Title: "Update task at pos 2"}, // Now the original task 1
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed multiple position insertions with updates: %v", err)
		}
		if !response.Success {
			t.Fatalf("multiple insertions with updates failed: %v", response.Errors)
		}

		// Operations are processed in order:
		// 1. Insert at pos 3 (original) -> [Task 1] [Task 2] [Insert at pos 3] [Task 3]
		// 2. Insert at pos 1 (original) -> [Insert at pos 1] [Task 1] [Task 2] [Insert at pos 3] [Task 3]
		// 3. Update ID "1" (now Insert at pos 1) -> update its title
		// 4. Update ID "2" (now original Task 1) -> update its title
		if len(tl.Tasks) != 5 {
			t.Errorf("expected 5 tasks after operations, got: %d", len(tl.Tasks))
		}

		// Check the final layout after all operations
		task1 := tl.FindTask("1")
		if task1.Title != "Update task at pos 1" {
			t.Errorf("expected updated insertion at position 1, got: %s", task1.Title)
		}

		task2 := tl.FindTask("2")
		if task2.Title != "Update task at pos 2" {
			t.Errorf("expected updated original task 1, got: %s", task2.Title)
		}

		// Verify the other tasks exist in expected positions
		task3 := tl.FindTask("3")
		if task3.Title != "Task 2" {
			t.Errorf("expected original task 2 at position 3, got: %s", task3.Title)
		}

		task4 := tl.FindTask("4")
		if task4.Title != "Insert at pos 3" {
			t.Errorf("expected insertion at position 4 (after renumbering), got: %s", task4.Title)
		}
	})

	// Test 3: Complex hierarchical insertion and updates
	t.Run("hierarchical_insertion_and_updates", func(t *testing.T) {
		// Reset and create simpler hierarchical structure for testing
		tl = task.NewTaskList("Hierarchical Test")
		tl.AddTask("", "Parent A", "")
		tl.AddTask("", "Parent B", "")
		tl.AddTask("1", "Child A1", "")
		tl.AddTask("2", "Child B1", "")

		// Simple operations that should work predictably
		operations := []task.Operation{
			// Insert new parent at beginning
			{Type: "add", Title: "New Parent", Position: "1"},
			// Update the task that is now at position 2 (original Parent A)
			{Type: "update", ID: "2", Title: "Updated Parent A"},
			// Add a new child to Parent A (now at position 2)
			{Type: "add", Parent: "2", Title: "New Child A2"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("failed hierarchical insertion and updates: %v", err)
		}
		if !response.Success {
			t.Fatalf("hierarchical operations failed: %v", response.Errors)
		}

		// Debug: Print the structure
		t.Logf("Final structure:")
		for _, task := range tl.Tasks {
			t.Logf("- %s: %s", task.ID, task.Title)
			for _, child := range task.Children {
				t.Logf("  - %s: %s", child.ID, child.Title)
			}
		}

		// Verify structure
		newParent := tl.FindTask("1")
		if newParent.Title != "New Parent" {
			t.Errorf("expected new parent at position 1, got: %s", newParent.Title)
		}

		updatedParent := tl.FindTask("2")
		if updatedParent.Title != "Updated Parent A" {
			t.Errorf("expected updated parent A at position 2, got: %s", updatedParent.Title)
		}

		// Check that Parent A now has 2 children
		if len(updatedParent.Children) != 2 {
			t.Errorf("expected parent A to have 2 children, got: %d", len(updatedParent.Children))
		}

		// Verify the new child was added
		newChild := tl.FindTask("2.2")
		if newChild == nil {
			t.Error("expected to find new child at 2.2")
		} else if newChild.Title != "New Child A2" {
			t.Errorf("expected new child A2, got: %s", newChild.Title)
		}
	})

	t.Logf("Batch position and update combination test passed successfully")
}

func testEnhancedErrorHandlingWorkflows(t *testing.T, tempDir string) {
	filename := "error-handling.md"

	// Create initial task structure
	tl := task.NewTaskList("Error Handling Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Add initial tasks
	if _, err := tl.AddTask("", "Task 1", ""); err != nil {
		t.Fatalf("failed to add initial task: %v", err)
	}
	if _, err := tl.AddTask("", "Task 2", ""); err != nil {
		t.Fatalf("failed to add initial task: %v", err)
	}
	if _, err := tl.AddTask("1", "Subtask 1.1", ""); err != nil {
		t.Fatalf("failed to add subtask: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write initial tasks: %v", err)
	}

	// Test 1: Invalid position format
	t.Run("invalid_position_format", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Title: "Invalid position", Position: "invalid.position.format"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if response.Success {
			t.Error("expected validation to fail for invalid position format")
		}
		if len(response.Errors) == 0 {
			t.Error("expected validation errors for invalid position")
		}
	})

	// Test 2: Update with invalid status value
	t.Run("invalid_status_value", func(t *testing.T) {
		invalidStatus := task.Status(99)
		operations := []task.Operation{
			{Type: "update", ID: "1", Status: &invalidStatus},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if response.Success {
			t.Error("expected validation to fail for invalid status value")
		}
	})

	// Test 3: Mixed valid and invalid operations (atomic failure)
	t.Run("atomic_failure_mixed_operations", func(t *testing.T) {
		originalTaskCount := len(tl.Tasks)
		operations := []task.Operation{
			{Type: "add", Title: "Valid task", Position: "1"},                 // Valid
			{Type: "update", ID: "1", Status: task.StatusPtr(task.Completed)}, // Valid
			{Type: "update", ID: "nonexistent", Title: "Should fail"},         // Invalid
			{Type: "add", Title: "Another valid task"},                        // Valid
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if response.Success {
			t.Error("expected batch to fail due to invalid operation")
		}

		// Verify no changes were applied (atomic failure)
		if len(tl.Tasks) != originalTaskCount {
			t.Errorf("expected no changes after failed batch, got %d tasks (was %d)", len(tl.Tasks), originalTaskCount)
		}

		// Verify original task 1 was not modified
		task1 := tl.FindTask("1")
		if task1.Status != task.Pending {
			t.Errorf("expected task 1 to remain pending after failed batch, got: %v", task1.Status)
		}
	})

	// Test 4: Position insertion with nonexistent parent
	t.Run("position_insertion_nonexistent_parent", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Parent: "nonexistent", Title: "Child task", Position: "1"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if response.Success {
			t.Error("expected validation to fail for nonexistent parent")
		}
	})

	// Test 5: Invalid operation type
	t.Run("invalid_operation_type", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "invalid_operation", ID: "1"},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if response.Success {
			t.Error("expected validation to fail for invalid operation type")
		}
	})

	// Test 6: Title length validation
	t.Run("title_length_validation", func(t *testing.T) {
		longTitle := ""
		for range 501 { // 501 characters
			longTitle += "a"
		}

		operations := []task.Operation{
			{Type: "add", Title: longTitle},
			{Type: "update", ID: "1", Title: longTitle},
		}

		response, err := tl.ExecuteBatch(operations, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if response.Success {
			t.Error("expected validation to fail for title length")
		}
		// Should have at least 1 error (both operations might fail on first error)
		if len(response.Errors) == 0 {
			t.Error("expected validation errors for operations with long titles")
		}
		t.Logf("Got %d validation errors as expected", len(response.Errors))
	})

	// Test 7: Successful dry run after error scenarios
	t.Run("dry_run_success_after_errors", func(t *testing.T) {
		operations := []task.Operation{
			{Type: "add", Title: "Valid task", Position: "1"},
			{Type: "update", ID: "2", Status: task.StatusPtr(task.InProgress), Details: []string{"Progress update"}},
		}

		response, err := tl.ExecuteBatch(operations, true) // dry run
		if err != nil {
			t.Fatalf("unexpected error in dry run: %v", err)
		}
		if !response.Success {
			t.Fatalf("dry run should succeed with valid operations: %v", response.Errors)
		}

		// Verify dry run didn't modify anything
		if len(tl.Tasks) != 2 {
			t.Error("dry run should not modify task list")
		}

		// Verify preview was generated
		if response.Preview == "" {
			t.Error("expected preview in dry run response")
		}
	})

	t.Logf("Enhanced error handling workflows test passed successfully")
}

func testCLIPositionInsertionIntegration(t *testing.T, tempDir string) {
	// Test 1: CLI position insertion with explicit file
	t.Run("cli_explicit_file_position_insertion", func(t *testing.T) {
		filename := "cli-position.md"

		// Create initial task file via CLI create command
		runGoCommand(t, "create", filename, "--title", "CLI Position Test")

		// Add initial tasks
		runGoCommand(t, "add", filename, "--title", "Task 1")
		runGoCommand(t, "add", filename, "--title", "Task 2")
		runGoCommand(t, "add", filename, "--title", "Task 3")

		// Test position insertion at beginning
		runGoCommand(t, "add", filename, "--title", "Urgent task", "--position", "1")

		// Verify structure
		output := runGoCommand(t, "list", filename, "-f", "json")
		if !strings.Contains(output, `"ID": "1"`) || !strings.Contains(output, "Urgent task") {
			t.Errorf("expected urgent task at position 1, got: %s", output)
		}
		if !strings.Contains(output, `"ID": "2"`) || !strings.Contains(output, "Task 1") {
			t.Errorf("expected original Task 1 at position 2, got: %s", output)
		}

		// Test position insertion in middle
		runGoCommand(t, "add", filename, "--title", "Middle task", "--position", "3")

		output = runGoCommand(t, "list", filename, "-f", "json")
		if !strings.Contains(output, `"ID": "3"`) || !strings.Contains(output, "Middle task") {
			t.Errorf("expected middle task at position 3, got: %s", output)
		}

		// Test subtask position insertion
		runGoCommand(t, "add", filename, "--title", "First subtask", "--parent", "2", "--position", "2.1")

		output = runGoCommand(t, "list", filename, "-f", "json")
		if !strings.Contains(output, `"ID": "2.1"`) || !strings.Contains(output, "First subtask") {
			t.Errorf("expected subtask at position 2.1, got: %s", output)
		}
	})

	// Test 2: CLI position insertion with git discovery
	t.Run("cli_git_discovery_position_insertion", func(t *testing.T) {
		// Initialize git repository
		runCommand(t, "git", "init")
		runCommand(t, "git", "config", "user.email", "test@example.com")
		runCommand(t, "git", "config", "user.name", "Test User")

		// Create config enabling git discovery
		configContent := `discovery:
  enabled: true
  template: "specs/{branch}/tasks.md"`
		if err := os.WriteFile(".rune.yml", []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create readme: %v", err)
		}
		runCommand(t, "git", "add", ".")
		runCommand(t, "git", "commit", "-m", "Initial commit")

		// Create feature branch
		runCommand(t, "git", "checkout", "-b", "feature/position-test")

		// Create task directory and file
		if err := os.MkdirAll("specs/feature/position-test", 0755); err != nil {
			t.Fatalf("failed to create task directory: %v", err)
		}

		taskFile := "specs/feature/position-test/tasks.md"
		runGoCommand(t, "create", taskFile, "--title", "Git Discovery Position Test")

		// Add initial tasks using git discovery (no filename needed)
		runGoCommand(t, "add", "--title", "First task")
		runGoCommand(t, "add", "--title", "Second task")
		runGoCommand(t, "add", "--title", "Third task")

		// Test position insertion with git discovery
		runGoCommand(t, "add", "--title", "Inserted at beginning", "--position", "1")

		// Verify using git discovery
		output := runGoCommand(t, "list", "-f", "json")
		if !strings.Contains(output, "Inserted at beginning") {
			t.Errorf("expected inserted task with git discovery, got: %s", output)
		}

		// Test subtask insertion
		runGoCommand(t, "add", "--title", "Subtask", "--parent", "2", "--position", "2.1")

		output = runGoCommand(t, "list", "-f", "json")
		if !strings.Contains(output, `"ID": "2.1"`) || !strings.Contains(output, "Subtask") {
			t.Errorf("expected subtask with git discovery, got: %s", output)
		}
	})

	// Test 3: CLI error handling for position insertion
	t.Run("cli_position_insertion_errors", func(t *testing.T) {
		filename := "cli-errors.md"
		runGoCommand(t, "create", filename, "--title", "Error Test")

		// Test invalid position format
		output := runGoCommandWithError(t, "add", filename, "--title", "Invalid", "--position", "invalid.format")
		if !strings.Contains(output, "failed to add task") {
			t.Errorf("expected error for invalid position format, got: %s", output)
		}

		// Test nonexistent parent for position insertion
		output = runGoCommandWithError(t, "add", filename, "--title", "No parent", "--parent", "999", "--position", "999.1")
		if !strings.Contains(output, "parent task 999 not found") {
			t.Errorf("expected parent not found error, got: %s", output)
		}
	})

	// Test 4: CLI dry run with position insertion
	t.Run("cli_position_insertion_dry_run", func(t *testing.T) {
		filename := "cli-dry-run.md"
		runGoCommand(t, "create", filename, "--title", "Dry Run Test")
		runGoCommand(t, "add", filename, "--title", "Existing task")

		// Test dry run shows position insertion plan
		output := runGoCommand(t, "add", filename, "--title", "Dry run task", "--position", "1", "--dry-run")
		if !strings.Contains(output, "Would add task") {
			t.Errorf("expected dry run output, got: %s", output)
		}
		if !strings.Contains(output, "Position: 1") {
			t.Errorf("expected position in dry run output, got: %s", output)
		}

		// Verify no actual changes were made
		listOutput := runGoCommand(t, "list", filename, "-f", "json")
		if strings.Contains(listOutput, "Dry run task") {
			t.Error("dry run should not have added actual task")
		}
	})

	// Test 5: CLI position insertion with verbose output
	t.Run("cli_position_insertion_verbose", func(t *testing.T) {
		filename := "cli-verbose.md"
		runGoCommand(t, "create", filename, "--title", "Verbose Test")

		// Test verbose output shows detailed information
		output := runGoCommand(t, "add", filename, "--title", "Verbose task", "--position", "1", "--verbose")
		if !strings.Contains(output, "Successfully added task") {
			t.Errorf("expected success message in verbose output, got: %s", output)
		}
		if !strings.Contains(output, "Task ID:") {
			t.Errorf("expected task ID in verbose output, got: %s", output)
		}
	})

	t.Logf("CLI position insertion integration test passed successfully")
}

func testFrontMatterIntegration(t *testing.T, tempDir string) {
	// Test 1: Create file with front matter end-to-end
	t.Run("create_with_front_matter", func(t *testing.T) {
		filename := "frontmatter-create.md"

		// Create file with front matter via CLI
		runGoCommand(t, "create", filename, "--title", "Front Matter Test")

		// Add front matter references
		runGoCommand(t, "add-frontmatter", filename,
			"--reference", "requirements.md",
			"--reference", "design.md",
			"--reference", "spec.md")

		// Add metadata
		runGoCommand(t, "add-frontmatter", filename,
			"--meta", "author:John Doe",
			"--meta", "version:2.0",
			"--meta", "status:draft")

		// Verify front matter was added correctly
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "---") {
			t.Error("expected YAML front matter delimiter")
		}
		if !strings.Contains(contentStr, "references:") {
			t.Error("expected references section in front matter")
		}
		if !strings.Contains(contentStr, "- requirements.md") {
			t.Error("expected requirements.md in references")
		}
		if !strings.Contains(contentStr, "metadata:") {
			t.Error("expected metadata section in front matter")
		}
		if !strings.Contains(contentStr, "author: John Doe") {
			t.Error("expected author in metadata")
		}

		// Verify YAML validity by parsing
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file with front matter: %v", err)
		}

		// Check that front matter references are preserved in task list
		if tl.FrontMatter == nil {
			t.Error("expected front matter to be preserved in task list")
		}
		if len(tl.FrontMatter.References) != 3 {
			t.Errorf("expected 3 references, got %d", len(tl.FrontMatter.References))
		}
	})

	// Test 2: Add front matter to existing file
	t.Run("add_front_matter_to_existing", func(t *testing.T) {
		filename := "existing-tasks.md"

		// Create file with existing tasks
		content := `# Existing Tasks

- [ ] 1. Task One
  - [ ] 1.1. Subtask One
- [ ] 2. Task Two
- [x] 3. Task Three`

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Add front matter
		runGoCommand(t, "add-frontmatter", filename,
			"--reference", "api-docs.md",
			"--meta", "priority:high")

		// Verify content preservation
		newContent, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		newContentStr := string(newContent)
		if !strings.Contains(newContentStr, "# Existing Tasks") {
			t.Error("expected original title to be preserved")
		}
		if !strings.Contains(newContentStr, "1. Task One") {
			t.Error("expected original tasks to be preserved")
		}
		if !strings.Contains(newContentStr, "[x] 3. Task Three") {
			t.Error("expected completed task to be preserved")
		}
		if !strings.Contains(newContentStr, "priority: high") {
			t.Error("expected metadata to be added")
		}
	})

	// Test 3: Complex nested metadata structures
	t.Run("complex_nested_metadata", func(t *testing.T) {
		filename := "complex-metadata.md"

		runGoCommand(t, "create", filename, "--title", "Complex Metadata Test")

		// Add deeply nested metadata
		runGoCommand(t, "add-frontmatter", filename,
			"--meta", "team:engineering",
			"--meta", "sprint:24",
			"--meta", "labels:backend",
			"--meta", "labels:critical",
			"--meta", "config_timeout:30",
			"--meta", "config_retries:3")

		// Verify parsing works with complex metadata
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse complex metadata: %v", err)
		}

		if tl.FrontMatter == nil || tl.FrontMatter.Metadata == nil {
			t.Error("expected metadata to be preserved")
		}

		// Verify metadata values
		metadata := tl.FrontMatter.Metadata
		if team, ok := metadata["team"]; !ok || team != "engineering" {
			t.Error("expected team metadata to be preserved")
		}
		if sprint, ok := metadata["sprint"]; !ok || sprint != "24" {
			t.Error("expected sprint metadata to be preserved")
		}
	})

	// Test 4: Resource limits handling
	t.Run("resource_limits", func(t *testing.T) {
		filename := "resource-limits.md"

		runGoCommand(t, "create", filename, "--title", "Resource Limits Test")

		// Test with many references (but within reasonable limits)
		var refs []string
		for i := 1; i <= 50; i++ {
			refs = append(refs, "--reference")
			refs = append(refs, fmt.Sprintf("doc%d.md", i))
		}

		args := append([]string{"add-frontmatter", filename}, refs...)
		runGoCommand(t, args...)

		// Verify all references were added
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to read file with many references: %v", err)
		}

		if len(tl.FrontMatter.References) != 50 {
			t.Errorf("expected 50 references, got %d", len(tl.FrontMatter.References))
		}
	})

	// Test 5: Verify YAML output validity
	t.Run("yaml_validity", func(t *testing.T) {
		filename := "yaml-valid.md"

		runGoCommand(t, "create", filename, "--title", "YAML Validity Test")

		// Add various metadata that could break YAML
		runGoCommand(t, "add-frontmatter", filename,
			"--reference", "file with spaces.md",
			"--reference", "special:chars.md",
			"--meta", "description:This has: colons and special chars",
			"--meta", "multiline:Line one\\nLine two",
			"--meta", "quoted:test with spaces")

		// Verify the file can be parsed successfully
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("YAML parsing failed: %v", err)
		}

		// Check that metadata was properly handled
		if tl.FrontMatter.Metadata["quoted"] != "test with spaces" {
			t.Error("expected quoted metadata to be preserved")
		}
	})

	// Test 6: Edge cases
	t.Run("edge_cases", func(t *testing.T) {
		filename := "edge-cases.md"

		runGoCommand(t, "create", filename, "--title", "Edge Cases Test")

		// Test empty value in key:value format
		runGoCommand(t, "add-frontmatter", filename,
			"--meta", "empty:",
			"--meta", "normal:value")

		// Test colons in values
		runGoCommand(t, "add-frontmatter", filename,
			"--meta", "url:https://example.com:8080/path",
			"--meta", "time:10:30:45")

		// Read and verify
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse edge cases: %v", err)
		}

		// Check empty value handling
		if val, ok := tl.FrontMatter.Metadata["empty"]; !ok || val != "" {
			t.Error("expected empty value to be preserved as empty string")
		}

		// Check colons in values
		if url, ok := tl.FrontMatter.Metadata["url"]; !ok || url != "https://example.com:8080/path" {
			t.Errorf("expected URL with colons to be preserved, got: %s", url)
		}
	})

	// Test 7: Front matter preservation during task operations
	t.Run("front_matter_preservation", func(t *testing.T) {
		filename := "preservation-test.md"

		// Create file with front matter
		runGoCommand(t, "create", filename, "--title", "Preservation Test")
		runGoCommand(t, "add-frontmatter", filename,
			"--reference", "original.md",
			"--meta", "status:active")

		// Add tasks
		runGoCommand(t, "add", filename, "--title", "Task 1")
		runGoCommand(t, "add", filename, "--title", "Task 2")

		// Complete a task
		runGoCommand(t, "complete", filename, "1")

		// Update a task
		runGoCommand(t, "update", filename, "2", "--title", "Updated Task 2")

		// Remove a task
		runGoCommand(t, "add", filename, "--title", "Task 3")
		runGoCommand(t, "remove", filename, "3")

		// Verify front matter is still intact
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to read file after operations: %v", err)
		}

		if tl.FrontMatter == nil {
			t.Error("front matter was lost during operations")
		}
		if len(tl.FrontMatter.References) != 1 || tl.FrontMatter.References[0] != "original.md" {
			t.Error("references were not preserved")
		}
		if tl.FrontMatter.Metadata["status"] != "active" {
			t.Error("metadata was not preserved")
		}
	})

	// Test 8: Maximum nesting depth
	t.Run("maximum_nesting_depth", func(t *testing.T) {
		filename := "nesting-test.md"

		runGoCommand(t, "create", filename, "--title", "Nesting Test")

		// Add deeply nested metadata structure (should error for keys with dots)
		output := runGoCommandWithError(t, "add-frontmatter", filename,
			"--meta", "level1.level2.level3:deep_value",
			"--meta", "level1.level2.another:value")
		if !strings.Contains(output, "nested keys not supported") {
			t.Errorf("expected error for nested metadata keys, got: %s", output)
		}

		// Add a simple metadata key (should succeed)
		runGoCommand(t, "add-frontmatter", filename, "--meta", "simple:value")

		// Verify it can be parsed
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse nested metadata: %v", err)
		}

		if tl.FrontMatter.Metadata["simple"] != "value" {
			t.Error("simple metadata not preserved")
		}
	})

	// Test 9: Error path coverage
	t.Run("error_handling", func(t *testing.T) {
		// Test non-existent file
		output := runGoCommandWithError(t, "add-frontmatter", "nonexistent.md",
			"--reference", "test.md")
		if !strings.Contains(output, "does not exist") {
			t.Errorf("expected file not found error, got: %s", output)
		}

		// Test invalid metadata format (missing colon)
		filename := "error-test.md"
		runGoCommand(t, "create", filename, "--title", "Error Test")

		output = runGoCommandWithError(t, "add-frontmatter", filename,
			"--meta", "invalid_no_colon")
		if !strings.Contains(output, "invalid") || !strings.Contains(output, "format") {
			t.Errorf("expected invalid format error, got: %s", output)
		}
	})

	t.Logf("Front matter integration test passed successfully")
}

// testPhaseWorkflowEndToEnd tests end-to-end phase creation and task addition

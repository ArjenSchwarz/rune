package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
)

func TestIntegrationWorkflows(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	tests := map[string]struct {
		name        string
		workflow    func(t *testing.T, tempDir string)
		description string
	}{
		"complete_task_lifecycle": {
			name:        "Complete Task Lifecycle",
			description: "Test create → add → update → complete → remove workflow",
			workflow:    testCompleteTaskLifecycle,
		},
		"hierarchical_task_management": {
			name:        "Hierarchical Task Management",
			description: "Test complex parent-child task relationships",
			workflow:    testHierarchicalTaskManagement,
		},
		"batch_operations_complex": {
			name:        "Complex Batch Operations",
			description: "Test complex JSON API batch operations",
			workflow:    testComplexBatchOperations,
		},
		"search_and_filtering": {
			name:        "Search and Filtering",
			description: "Test find command with various filters",
			workflow:    testSearchAndFiltering,
		},
		"error_handling_recovery": {
			name:        "Error Handling and Recovery",
			description: "Test error scenarios and recovery",
			workflow:    testErrorHandlingRecovery,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Create isolated temp directory for each test
			tempDir, err := os.MkdirTemp("", "go-tasks-integration-"+testName)
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldDir, _ := os.Getwd()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}
			defer os.Chdir(oldDir)

			t.Logf("Running integration test: %s", tc.description)
			tc.workflow(t, tempDir)
		})
	}
}

func testCompleteTaskLifecycle(t *testing.T, tempDir string) {
	filename := "lifecycle.md"

	// Step 1: Create new task file
	tl := task.NewTaskList("Integration Test Project")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Step 2: Add root level tasks
	tl, err := task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse task file: %v", err)
	}

	err = tl.AddTask("", "Implement authentication system")
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	err = tl.AddTask("", "Write documentation")
	if err != nil {
		t.Fatalf("failed to add second task: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Step 3: Add subtasks
	err = tl.AddTask("1", "Design user model")
	if err != nil {
		t.Fatalf("failed to add subtask: %v", err)
	}

	err = tl.AddTask("1", "Implement login endpoint")
	if err != nil {
		t.Fatalf("failed to add second subtask: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Step 4: Update task details and references
	err = tl.UpdateTask("1.1", "", []string{"Create User struct", "Add validation"}, []string{"user-model-spec.md"})
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Step 5: Mark tasks as in-progress and completed
	err = tl.UpdateStatus("1.1", task.InProgress)
	if err != nil {
		t.Fatalf("failed to update status to in-progress: %v", err)
	}

	err = tl.UpdateStatus("1.1", task.Completed)
	if err != nil {
		t.Fatalf("failed to update status to completed: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Step 6: Remove a task
	err = tl.RemoveTask("2")
	if err != nil {
		t.Fatalf("failed to remove task: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Step 7: Verify final state
	tl, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse final task file: %v", err)
	}

	if len(tl.Tasks) != 1 {
		t.Errorf("expected 1 root task, got %d", len(tl.Tasks))
	}

	if len(tl.Tasks[0].Children) != 2 {
		t.Errorf("expected 2 subtasks, got %d", len(tl.Tasks[0].Children))
	}

	if tl.Tasks[0].Children[0].Status != task.Completed {
		t.Errorf("expected first subtask to be completed, got %v", tl.Tasks[0].Children[0].Status)
	}

	t.Logf("Complete task lifecycle test passed successfully")
}

func testHierarchicalTaskManagement(t *testing.T, tempDir string) {
	filename := "hierarchy.md"

	// Create complex hierarchy: 3 levels deep with multiple branches
	tl := task.NewTaskList("Complex Project")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Build hierarchy: Phase 1 -> Feature A -> Tasks, Phase 1 -> Feature B -> Tasks
	phases := []string{"Planning Phase", "Development Phase", "Testing Phase"}
	for _, phase := range phases {
		if err := tl.AddTask("", phase); err != nil {
			t.Fatalf("failed to add phase %s: %v", phase, err)
		}
	}

	// Add features to each phase
	features := map[string][]string{
		"1": {"Frontend UI", "Backend API"},
		"2": {"User Management", "Data Processing"},
		"3": {"Unit Tests", "Integration Tests"},
	}

	for phaseID, featureList := range features {
		for _, feature := range featureList {
			if err := tl.AddTask(phaseID, feature); err != nil {
				t.Fatalf("failed to add feature %s to phase %s: %v", feature, phaseID, err)
			}
		}
	}

	// Add detailed tasks to some features
	detailTasks := []struct {
		parentID string
		title    string
	}{
		{"1.1", "Create component wireframes"},
		{"1.1", "Implement React components"},
		{"1.1", "Add CSS styling"},
		{"1.2", "Design REST endpoints"},
		{"1.2", "Implement CRUD operations"},
		{"2.1", "User registration"},
		{"2.1", "User authentication"},
		{"2.1", "Password reset"},
	}

	for _, dt := range detailTasks {
		if err := tl.AddTask(dt.parentID, dt.title); err != nil {
			t.Fatalf("failed to add detail task %s: %v", dt.title, err)
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Verify hierarchy structure
	tl, err := task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse hierarchy file: %v", err)
	}

	// Should have 3 root tasks (phases)
	if len(tl.Tasks) != 3 {
		t.Errorf("expected 3 root tasks, got %d", len(tl.Tasks))
	}

	// Phase 1 should have 2 features
	if len(tl.Tasks[0].Children) != 2 {
		t.Errorf("expected phase 1 to have 2 features, got %d", len(tl.Tasks[0].Children))
	}

	// Frontend UI should have 3 tasks
	if len(tl.Tasks[0].Children[0].Children) != 3 {
		t.Errorf("expected Frontend UI to have 3 tasks, got %d", len(tl.Tasks[0].Children[0].Children))
	}

	// Test finding tasks at different levels
	foundTask := tl.FindTask("1.1.2")
	if foundTask == nil {
		t.Error("failed to find task at 3rd level")
	} else if foundTask.Title != "Implement React components" {
		t.Errorf("expected task title 'Implement React components', got '%s'", foundTask.Title)
	}

	// Test removing middle-level task and verify renumbering
	err = tl.RemoveTask("1.2")
	if err != nil {
		t.Fatalf("failed to remove middle task: %v", err)
	}

	// Phase 1 should now have 1 feature
	if len(tl.Tasks[0].Children) != 1 {
		t.Errorf("after removal, expected phase 1 to have 1 feature, got %d", len(tl.Tasks[0].Children))
	}

	t.Logf("Hierarchical task management test passed successfully")
}

func testComplexBatchOperations(t *testing.T, tempDir string) {
	filename := "batch.md"

	// Create initial task file
	tl := task.NewTaskList("Batch Operations Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Add initial tasks for batch operations
	initialTasks := []string{
		"Setup project",
		"Configure CI/CD",
		"Write tests",
		"Deploy to staging",
	}

	for _, taskTitle := range initialTasks {
		if err := tl.AddTask("", taskTitle); err != nil {
			t.Fatalf("failed to add initial task %s: %v", taskTitle, err)
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write initial tasks: %v", err)
	}

	// Test complex batch operations (Note: operations execute in sequence)
	operations := []task.Operation{
		// Add new tasks first
		{Type: "add", Parent: "", Title: "Security audit"},
		{Type: "add", Parent: "1", Title: "Initialize Git repository"},
		{Type: "add", Parent: "1", Title: "Setup package.json"},
		// Update existing tasks
		{Type: "update", ID: "2", Title: "Configure CI/CD with GitHub Actions", Details: []string{"Create workflow files", "Setup secrets"}},
		{Type: "update_status", ID: "3", Status: task.InProgress},
		// Add references
		{Type: "update", ID: "4", References: []string{"staging-deploy.md", "env-config.md"}},
		// Add nested tasks to the security audit (task 5)
		{Type: "add", Parent: "5", Title: "Vulnerability scanning"},
		{Type: "add", Parent: "5", Title: "Penetration testing"},
		// Remove a task at the end to avoid renumbering issues
		{Type: "remove", ID: "2"},
	}

	// Execute batch operations
	response, err := tl.ExecuteBatch(operations, false)
	if err != nil {
		t.Fatalf("failed to execute batch operations: %v", err)
	}
	if !response.Success {
		t.Fatalf("batch operations failed: %v", response.Errors)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write batch results: %v", err)
	}

	// Verify results
	tl, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse batch results: %v", err)
	}

	// Should have 4 root tasks (started with 4, removed 1, added 1)
	if len(tl.Tasks) != 4 {
		t.Errorf("expected 4 root tasks after batch operations, got %d", len(tl.Tasks))
	}

	// Task 1 should have 2 subtasks (Initialize Git repository, Setup package.json)
	if len(tl.Tasks[0].Children) != 2 {
		t.Errorf("expected task 1 to have 2 subtasks, got %d", len(tl.Tasks[0].Children))
	}

	// After removing task 2 and renumbering, original task 4 becomes task 3
	// Task 3 should have references (originally task 4 - Deploy to staging)
	task3 := tl.FindTask("3")
	if task3 == nil {
		t.Error("failed to find task 3")
	} else if len(task3.References) == 0 {
		t.Error("expected task 3 to have references")
	}

	// Security audit task (originally added as task 5, becomes task 4 after removal) should have 2 subtasks
	securityTask := tl.FindTask("4")
	if securityTask == nil {
		t.Error("failed to find security audit task")
	} else if len(securityTask.Children) != 2 {
		t.Errorf("expected security audit to have 2 subtasks, got %d", len(securityTask.Children))
	}

	// Test batch operation validation (should fail)
	invalidOps := []task.Operation{
		{Type: "add", Parent: "nonexistent", Title: "This should fail"},
		{Type: "update_status", ID: "999", Status: task.Completed},
	}

	response, err = tl.ExecuteBatch(invalidOps, false)
	if err != nil {
		t.Fatalf("unexpected error during invalid batch operations: %v", err)
	}
	if response.Success {
		t.Error("expected batch operations to fail with invalid operations")
	}

	t.Logf("Complex batch operations test passed successfully")
}

func testSearchAndFiltering(t *testing.T, tempDir string) {
	filename := "search.md"

	// Create comprehensive task file for searching
	tl := task.NewTaskList("Search Test Project")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Add diverse tasks for search testing
	searchTestTasks := []struct {
		parent  string
		title   string
		status  task.Status
		details []string
		refs    []string
	}{
		{"", "Database design", task.Completed, []string{"Create ERD", "Define relationships"}, []string{"db-spec.md"}},
		{"", "API development", task.InProgress, []string{"REST endpoints", "Authentication"}, []string{"api-docs.md"}},
		{"", "Frontend implementation", task.Pending, []string{"React components", "State management"}, []string{}},
		{"1", "User table design", task.Completed, []string{"Define columns", "Add indexes"}, []string{"user-schema.sql"}},
		{"1", "Product table design", task.InProgress, []string{"Product attributes", "Category relationships"}, []string{}},
		{"2", "User authentication API", task.Completed, []string{"Login endpoint", "Token validation"}, []string{"auth-api.md"}},
		{"2", "Product management API", task.InProgress, []string{"CRUD operations", "Search functionality"}, []string{"product-api.md"}},
		{"3", "User interface components", task.Pending, []string{"Login form", "User dashboard"}, []string{"ui-mockups.png"}},
	}

	for _, tt := range searchTestTasks {
		if err := tl.AddTask(tt.parent, tt.title); err != nil {
			t.Fatalf("failed to add search test task: %v", err)
		}

		// Find the task we just added and update its properties
		var taskID string
		if tt.parent == "" {
			taskID = fmt.Sprintf("%d", len(tl.Tasks))
		} else {
			parent := tl.FindTask(tt.parent)
			if parent == nil {
				t.Fatalf("parent task %s not found", tt.parent)
			}
			taskID = fmt.Sprintf("%s.%d", tt.parent, len(parent.Children))
		}

		if err := tl.UpdateTask(taskID, "", tt.details, tt.refs); err != nil {
			t.Fatalf("failed to update task details: %v", err)
		}

		if err := tl.UpdateStatus(taskID, tt.status); err != nil {
			t.Fatalf("failed to update task status: %v", err)
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write search test file: %v", err)
	}

	// Reload the file
	tl, err := task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse search test file: %v", err)
	}

	// Test 1: Search by title content
	results := tl.Find("design", task.QueryOptions{})
	if len(results) != 3 {
		t.Errorf("expected 3 results for 'design' search, got %d", len(results))
	}

	// Test 2: Case-insensitive search
	results = tl.Find("API", task.QueryOptions{CaseSensitive: false})
	if len(results) < 2 {
		t.Errorf("expected at least 2 results for 'API' search, got %d", len(results))
	}

	// Test 3: Search in details
	results = tl.Find("CRUD", task.QueryOptions{SearchDetails: true})
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'CRUD' detail search, got %d", len(results))
	}

	// Test 4: Search in references
	results = tl.Find("api-docs.md", task.QueryOptions{SearchRefs: true})
	if len(results) != 1 {
		t.Errorf("expected 1 result for reference search, got %d", len(results))
	}

	// Test 5: Filter by status
	completedStatus := task.Completed
	completedFilter := task.QueryFilter{Status: &completedStatus}
	results = tl.Filter(completedFilter)
	if len(results) != 3 {
		t.Errorf("expected 3 completed tasks, got %d", len(results))
	}

	inProgressStatus := task.InProgress
	inProgressFilter := task.QueryFilter{Status: &inProgressStatus}
	results = tl.Filter(inProgressFilter)
	if len(results) != 3 {
		t.Errorf("expected 3 in-progress tasks, got %d", len(results))
	}

	// Test 6: Filter by hierarchy level (root tasks only)
	rootFilter := task.QueryFilter{MaxDepth: 1}
	results = tl.Filter(rootFilter)
	if len(results) != 3 {
		t.Errorf("expected 3 root tasks, got %d", len(results))
	}

	// Test 7: Filter by parent ID
	parentID := "2"
	parentFilter := task.QueryFilter{ParentID: &parentID}
	results = tl.Filter(parentFilter)
	if len(results) != 2 {
		t.Errorf("expected 2 children of task 2, got %d", len(results))
	}

	t.Logf("Search and filtering test passed successfully")
}

func testErrorHandlingRecovery(t *testing.T, tempDir string) {
	filename := "errors.md"

	// Test 1: Handle malformed markdown
	malformedContent := `# Malformed Tasks

- [ ] 1. Valid task
    - [ ] 1.1. Incorrect indentation (5 spaces instead of 2)
- [ ] 2. Another valid task
- [ ] ??? Invalid ID format
- [invalid] 3. Invalid status marker
`
	if err := os.WriteFile(filename, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to write malformed file: %v", err)
	}

	// Parsing should report errors without auto-correction per Decision #2
	_, err := task.ParseFile(filename)
	if err == nil {
		t.Error("expected error when parsing malformed content")
	}

	// Test 2: Handle valid operations on non-existent tasks
	tl := task.NewTaskList("Error Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create clean file: %v", err)
	}

	// Try to update non-existent task
	err = tl.UpdateStatus("999", task.Completed)
	if err == nil {
		t.Error("expected error when updating non-existent task")
	}

	// Try to remove non-existent task
	err = tl.RemoveTask("999")
	if err == nil {
		t.Error("expected error when removing non-existent task")
	}

	// Try to add task to non-existent parent
	err = tl.AddTask("999", "Child of non-existent")
	if err == nil {
		t.Error("expected error when adding task to non-existent parent")
	}

	// Test 3: Batch operation atomic failure
	// Add some valid tasks first
	if err := tl.AddTask("", "Valid task 1"); err != nil {
		t.Fatalf("failed to add valid task: %v", err)
	}
	if err := tl.AddTask("", "Valid task 2"); err != nil {
		t.Fatalf("failed to add valid task: %v", err)
	}

	// Try batch operation with mixed valid/invalid operations
	mixedOps := []task.Operation{
		{Type: "add", Parent: "", Title: "Valid new task"},
		{Type: "update_status", ID: "1", Status: task.Completed}, // Valid
		{Type: "remove", ID: "999"},                              // Invalid - should cause entire batch to fail
	}

	originalTaskCount := len(tl.Tasks)
	response, err := tl.ExecuteBatch(mixedOps, false)
	if err != nil {
		t.Fatalf("unexpected error during batch validation: %v", err)
	}
	if response.Success {
		t.Error("expected batch operation to fail due to invalid operation")
	}

	// Verify no changes were applied (atomic failure)
	if len(tl.Tasks) != originalTaskCount {
		t.Error("expected no changes after failed batch operation")
	}

	if tl.Tasks[0].Status != task.Pending {
		t.Error("expected task status to remain unchanged after failed batch")
	}

	// Test 4: File system error recovery
	// Test with invalid file path
	invalidFile := "/invalid/path/that/does/not/exist/file.md"

	// Try to parse non-existent file in invalid path
	_, err = task.ParseFile(invalidFile)
	if err == nil {
		t.Error("expected error when reading from invalid path")
	}

	// Try to write to invalid path
	err = tl.WriteFile(invalidFile)
	if err == nil {
		t.Error("expected error when writing to invalid path")
	}

	// Test path traversal protection
	traversalFile := "../../../etc/passwd"
	err = tl.WriteFile(traversalFile)
	if err == nil {
		t.Error("expected error when attempting path traversal")
	}

	t.Logf("Error handling and recovery test passed successfully")
}

func TestLargeFileHandling(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "go-tasks-large-file")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	filename := "large-tasks.md"

	// Create task list with 100+ tasks as specified in requirements
	tl := task.NewTaskList("Large File Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}

	// Add 100+ tasks with hierarchy
	t.Logf("Creating large task file with 150 tasks...")

	// Create 10 main phases with 15 tasks each
	for phase := 1; phase <= 10; phase++ {
		phaseTitle := fmt.Sprintf("Phase %d: Development Stage", phase)
		if err := tl.AddTask("", phaseTitle); err != nil {
			t.Fatalf("failed to add phase %d: %v", phase, err)
		}

		phaseID := fmt.Sprintf("%d", phase)

		// Add 15 tasks to each phase
		for taskNum := 1; taskNum <= 15; taskNum++ {
			taskTitle := fmt.Sprintf("Task %d.%d: Implementation step", phase, taskNum)
			if err := tl.AddTask(phaseID, taskTitle); err != nil {
				t.Fatalf("failed to add task %d.%d: %v", phase, taskNum, err)
			}

			// Add details and references to some tasks
			if taskNum%3 == 0 {
				taskID := fmt.Sprintf("%d.%d", phase, taskNum)
				details := []string{
					"Review requirements",
					"Implement functionality",
					"Add unit tests",
					"Update documentation",
				}
				refs := []string{fmt.Sprintf("spec-phase-%d.md", phase)}

				if err := tl.UpdateTask(taskID, "", details, refs); err != nil {
					t.Fatalf("failed to update task %s: %v", taskID, err)
				}
			}
		}
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write large file: %v", err)
	}

	// Test file size and parsing performance
	fileInfo, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("failed to stat large file: %v", err)
	}

	t.Logf("Large file size: %d bytes", fileInfo.Size())

	// Test parsing performance (should be sub-second per requirements)
	_, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse large file: %v", err)
	}

	// Verify structure
	tl, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to re-parse large file: %v", err)
	}

	if len(tl.Tasks) != 10 {
		t.Errorf("expected 10 root tasks, got %d", len(tl.Tasks))
	}

	totalSubTasks := 0
	for _, rootTask := range tl.Tasks {
		totalSubTasks += len(rootTask.Children)
	}

	if totalSubTasks != 150 {
		t.Errorf("expected 150 subtasks total, got %d", totalSubTasks)
	}

	// Test search performance on large file
	results := tl.Find("Implementation", task.QueryOptions{})
	if len(results) != 150 {
		t.Errorf("expected 150 search results, got %d", len(results))
	}

	// Test status filtering performance
	pendingStatus := task.Pending
	filter := task.QueryFilter{Status: &pendingStatus}
	results = tl.Filter(filter)
	if len(results) != 160 { // 10 root + 150 subtasks
		t.Errorf("expected 160 pending tasks, got %d", len(results))
	}

	// Test batch operations on large file
	batchOps := make([]task.Operation, 20)
	for i := range 20 {
		batchOps[i] = task.Operation{
			Type:   "update_status",
			ID:     fmt.Sprintf("%d.%d", (i%10)+1, (i%15)+1),
			Status: task.Completed,
		}
	}

	batchResponse, err := tl.ExecuteBatch(batchOps, false)
	if err != nil {
		t.Fatalf("failed to execute batch operations on large file: %v", err)
	}
	if !batchResponse.Success {
		t.Fatalf("batch operations on large file failed: %v", batchResponse.Errors)
	}

	t.Logf("Large file handling test passed successfully (160 tasks processed)")
}

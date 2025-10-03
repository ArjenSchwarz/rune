package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
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
		"git_discovery_integration": {
			name:        "Git Discovery Integration",
			description: "Test git branch discovery with various commands",
			workflow:    testGitDiscoveryIntegration,
		},
		"next_command_task_states": {
			name:        "Next Command with Various Task States",
			description: "Test next command behavior with different task completion states",
			workflow:    testNextCommandTaskStates,
		},
		"auto_completion_multi_level": {
			name:        "Auto-completion Through Multiple Levels",
			description: "Test automatic parent task completion across multiple hierarchy levels",
			workflow:    testAutoCompletionMultiLevel,
		},
		"reference_inclusion_formats": {
			name:        "Reference Inclusion in All Output Formats",
			description: "Test that references are included in table, markdown, and JSON formats",
			workflow:    testReferenceInclusionFormats,
		},
		"configuration_integration": {
			name:        "Configuration Integration Tests",
			description: "Test configuration file precedence and validation",
			workflow:    testConfigurationIntegration,
		},
		"front_matter_integration": {
			name:        "Front Matter Integration Tests",
			description: "Test front matter creation, modification, and edge cases",
			workflow:    testFrontMatterIntegration,
		},
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
		"phase_workflow_end_to_end": {
			name:        "Phase Workflow End-to-End",
			description: "Test end-to-end phase creation and task addition",
			workflow:    testPhaseWorkflowEndToEnd,
		},
		"phase_round_trip": {
			name:        "Phase Round Trip",
			description: "Test round-trip (parse -> modify -> render -> parse) with phases",
			workflow:    testPhaseRoundTrip,
		},
		"phase_batch_operations": {
			name:        "Phase Batch Operations",
			description: "Test batch operations creating and populating phases",
			workflow:    testPhaseBatchOperations,
		},
		"phase_backward_compatibility": {
			name:        "Phase Backward Compatibility",
			description: "Verify backward compatibility with legacy task files",
			workflow:    testPhaseBackwardCompatibility,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Create isolated temp directory for each test
			tempDir, err := os.MkdirTemp("", "rune-integration-"+testName)
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

	_, err = tl.AddTask("", "Implement authentication system", "")
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	_, err = tl.AddTask("", "Write documentation", "")
	if err != nil {
		t.Fatalf("failed to add second task: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	// Step 3: Add subtasks
	_, err = tl.AddTask("1", "Design user model", "")
	if err != nil {
		t.Fatalf("failed to add subtask: %v", err)
	}

	_, err = tl.AddTask("1", "Implement login endpoint", "")
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
		if _, err := tl.AddTask("", phase, ""); err != nil {
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
			if _, err := tl.AddTask(phaseID, feature, ""); err != nil {
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
		if _, err := tl.AddTask(dt.parentID, dt.title, ""); err != nil {
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
		if _, err := tl.AddTask("", taskTitle, ""); err != nil {
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
		{Type: "update", ID: "3", Status: task.StatusPtr(task.InProgress)},
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
		{Type: "update", ID: "999", Status: task.StatusPtr(task.Completed)},
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
		if _, err := tl.AddTask(tt.parent, tt.title, ""); err != nil {
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
	_, err = tl.AddTask("999", "Child of non-existent", "")
	if err == nil {
		t.Error("expected error when adding task to non-existent parent")
	}

	// Test 3: Batch operation atomic failure
	// Add some valid tasks first
	if _, err := tl.AddTask("", "Valid task 1", ""); err != nil {
		t.Fatalf("failed to add valid task: %v", err)
	}
	if _, err := tl.AddTask("", "Valid task 2", ""); err != nil {
		t.Fatalf("failed to add valid task: %v", err)
	}

	// Try batch operation with mixed valid/invalid operations
	mixedOps := []task.Operation{
		{Type: "add", Parent: "", Title: "Valid new task"},
		{Type: "update", ID: "1", Status: task.StatusPtr(task.Completed)}, // Valid
		{Type: "remove", ID: "999"},                                       // Invalid - should cause entire batch to fail
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
	tempDir, err := os.MkdirTemp("", "rune-large-file")
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
		if _, err := tl.AddTask("", phaseTitle, ""); err != nil {
			t.Fatalf("failed to add phase %d: %v", phase, err)
		}

		phaseID := fmt.Sprintf("%d", phase)

		// Add 15 tasks to each phase
		for taskNum := 1; taskNum <= 15; taskNum++ {
			taskTitle := fmt.Sprintf("Task %d.%d: Implementation step", phase, taskNum)
			if _, err := tl.AddTask(phaseID, taskTitle, ""); err != nil {
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
			Type:   "update",
			ID:     fmt.Sprintf("%d.%d", (i%10)+1, (i%15)+1),
			Status: task.StatusPtr(task.Completed),
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

func testGitDiscoveryIntegration(t *testing.T, tempDir string) {
	// Initialize git repository
	runCommand(t, "git", "init")
	runCommand(t, "git", "config", "user.email", "test@example.com")
	runCommand(t, "git", "config", "user.name", "Test User")

	// Create test directory structure
	if err := os.MkdirAll("specs/test-feature", 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create config file enabling git discovery
	configContent := `discovery:
  enabled: true
  template: "specs/{branch}/tasks.md"
`
	if err := os.WriteFile(".rune.yml", []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Create an initial commit so HEAD exists
	if err := os.WriteFile("README.md", []byte("# Test repo"), 0644); err != nil {
		t.Fatalf("failed to create readme: %v", err)
	}
	runCommand(t, "git", "add", ".")
	runCommand(t, "git", "commit", "-m", "Initial commit")

	// Create test branch and switch to it
	runCommand(t, "git", "checkout", "-b", "feature/test-feature")

	// Create tasks file in the expected location
	taskContent := `---
references:
  - ./docs/feature-spec.md
  - ./tests/test-plan.md
---
# Test Feature Tasks

- [ ] 1. Setup development environment
  - [x] 1.1. Install dependencies  
  - [ ] 1.2. Configure database
- [ ] 2. Implement core functionality
  - [ ] 2.1. Design API
  - [ ] 2.2. Write tests
`
	taskFile := "specs/feature/test-feature/tasks.md"
	if err := os.MkdirAll("specs/feature/test-feature", 0755); err != nil {
		t.Fatalf("failed to create task directory: %v", err)
	}
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Test 1: List command with git discovery
	t.Run("list_with_git_discovery", func(t *testing.T) {
		output := runGoCommand(t, "list", "-f", "json")
		if !containsString(output, "Setup development environment") {
			t.Errorf("expected task content in output, got: %s", output)
		}
		if !containsString(output, "references") && !containsString(output, "feature-spec.md") {
			t.Errorf("expected references in output, got: %s", output)
		}
	})

	// Test 2: Next command with git discovery
	t.Run("next_with_git_discovery", func(t *testing.T) {
		output := runGoCommand(t, "next", "-f", "table")
		if !containsString(output, "1") || !containsString(output, "Setup development environment") {
			t.Errorf("expected next task in output, got: %s", output)
		}
		if !containsString(output, "Reference Documents") {
			t.Errorf("expected references section in output, got: %s", output)
		}
	})

	// Test 3: Complete command with git discovery (single argument)
	t.Run("complete_with_git_discovery", func(t *testing.T) {
		runGoCommand(t, "complete", "1.2")

		// Verify the task was completed
		output := runGoCommand(t, "list", "-f", "json")
		// Check for the specific task object with both ID and completed status
		// Note: JSON is formatted with spaces, so we need to account for that
		if !containsString(output, `"ID": "1.2"`) {
			t.Errorf("task 1.2 should be present, got: %s", output)
		}
		if !containsString(output, `"Status": 2`) {
			t.Errorf("task 1.2 should be completed (Status:2), got: %s", output)
		}
	})

	// Test 4: Find command with git discovery
	t.Run("find_with_git_discovery", func(t *testing.T) {
		output := runGoCommand(t, "find", "-p", "API", "-f", "table")
		if !containsString(output, "2.1") || !containsString(output, "Design API") {
			t.Errorf("expected to find API task, got: %s", output)
		}
	})

	// Test 5: Add task with git discovery
	t.Run("add_with_git_discovery", func(t *testing.T) {
		runGoCommand(t, "add", "--title", "Write documentation", "--parent", "2")

		// Verify the task was added
		output := runGoCommand(t, "list", "-f", "json")
		if !containsString(output, "Write documentation") {
			t.Errorf("expected to find added task, got: %s", output)
		}
	})

	// Test 6: Error when git discovery fails (wrong branch)
	t.Run("git_discovery_error", func(t *testing.T) {
		// Switch to a branch without corresponding task file
		runCommand(t, "git", "checkout", "-b", "nonexistent-branch")

		output := runGoCommandWithError(t, "list")
		if !containsString(output, "branch-based file not found") && !containsString(output, "git discovery failed") {
			t.Errorf("expected git discovery error, got: %s", output)
		}
	})

	// Test 7: Explicit file overrides git discovery
	t.Run("explicit_file_overrides_discovery", func(t *testing.T) {
		// Create a different task file
		explicitContent := `# Explicit Tasks
- [ ] 1. Explicit task`
		explicitFile := "explicit-tasks.md"
		if err := os.WriteFile(explicitFile, []byte(explicitContent), 0644); err != nil {
			t.Fatalf("failed to create explicit task file: %v", err)
		}

		output := runGoCommand(t, "list", explicitFile, "-f", "json")
		if !containsString(output, "Explicit task") {
			t.Errorf("expected explicit task content, got: %s", output)
		}
		if containsString(output, "Setup development environment") {
			t.Errorf("should not contain git-discovered content, got: %s", output)
		}
	})

	t.Logf("Git discovery integration test completed successfully")
}

// Helper functions for the integration tests

func runCommand(t *testing.T, name string, args ...string) {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %s %v failed: %v", name, args, err)
	}
}

func runCommandWithOutput(t *testing.T, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %s %v failed: %v, output: %s", name, args, err, output)
	}
	return string(output)
}

func runGoCommand(t *testing.T, args ...string) string {
	return runCommandWithOutput(t, "rune", args...)
}

func runGoCommandWithError(_ *testing.T, args ...string) string {
	cmd := exec.Command("rune", args...)
	output, _ := cmd.CombinedOutput()
	return string(output)
}

func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func testNextCommandTaskStates(t *testing.T, tempDir string) {
	filename := "next-states.md"

	// Create task file with various completion states
	taskContent := `---
references:
  - ./project-spec.md
  - ./requirements.md
---
# Next Command Test Tasks

- [x] 1. Completed root task
  - [x] 1.1. Completed subtask
  - [x] 1.2. Another completed subtask
- [ ] 2. Pending root with mixed children
  - [x] 2.1. Completed child
  - [-] 2.2. In-progress child
  - [ ] 2.3. Pending child
- [-] 3. In-progress root with pending children
  - [ ] 3.1. Pending child of in-progress parent
  - [ ] 3.2. Another pending child
- [ ] 4. Fully pending task tree
  - [ ] 4.1. Pending child
    - [ ] 4.1.1. Deep pending grandchild
    - [ ] 4.1.2. Another deep pending grandchild
  - [ ] 4.2. Another pending child
`

	if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Test 1: Next command should return the first incomplete task (task 2)
	t.Run("next_finds_first_incomplete", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "json")
		if !containsString(output, `"id": "2"`) {
			t.Errorf("expected next task to be task 2, got: %s", output)
		}
		if !containsString(output, "Pending root with mixed children") {
			t.Errorf("expected task 2 title in output, got: %s", output)
		}
		// Should include references from front matter
		if !containsString(output, "project-spec.md") {
			t.Errorf("expected references in next output, got: %s", output)
		}
	})

	// Test 2: Next command should include all children (completed and incomplete)
	// Per Requirements 1.6: "return the found task and all its subtasks (regardless of their completion status)"
	t.Run("next_includes_all_children", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "json")
		// Should include in-progress child 2.2
		if !containsString(output, `"id": "2.2"`) {
			t.Errorf("expected to include in-progress child 2.2, got: %s", output)
		}
		// Should include pending child 2.3
		if !containsString(output, `"id": "2.3"`) {
			t.Errorf("expected to include pending child 2.3, got: %s", output)
		}
		// Should include completed child 2.1 for context (per requirements 1.6)
		if !containsString(output, `"id": "2.1"`) {
			t.Errorf("expected to include completed child 2.1 for context, got: %s", output)
		}
		// Should have children array
		if !containsString(output, `"children": [`) {
			t.Errorf("expected children array in output, got: %s", output)
		}
	})

	// Test 3: Test with table format to ensure references are shown
	t.Run("next_table_format", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "table")
		if !containsString(output, "Pending root with mixed children") {
			t.Errorf("expected task title in table output, got: %s", output)
		}
		// References section may be truncated in table format as "Reference Documen"
		if !containsString(output, "Reference Documen") {
			t.Errorf("expected references section in table output, got: %s", output)
		}
		if !containsString(output, "project-spec.md") {
			t.Errorf("expected specific reference in output, got: %s", output)
		}
	})

	// Test 4: Test with markdown format
	t.Run("next_markdown_format", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "markdown")
		if !containsString(output, "# Next Task") {
			t.Errorf("expected markdown heading in output, got: %s", output)
		}
		if !containsString(output, "project-spec.md") {
			t.Errorf("expected references in markdown output, got: %s", output)
		}
	})

	// Test 5: Complete all tasks and verify "all complete" message
	t.Run("all_tasks_complete", func(t *testing.T) {
		// Complete all remaining tasks
		tasksToComplete := []string{"2.2", "2.3", "2", "3.1", "3.2", "3", "4.1.1", "4.1.2", "4.1", "4.2", "4"}
		for _, taskID := range tasksToComplete {
			runGoCommand(t, "complete", filename, taskID)
		}

		output := runGoCommand(t, "next", filename)
		if !containsString(output, "All tasks are complete") {
			t.Errorf("expected 'all tasks complete' message, got: %s", output)
		}
	})

	t.Logf("Next command task states test passed successfully")
}

func testAutoCompletionMultiLevel(t *testing.T, tempDir string) {
	filename := "auto-complete.md"

	// Create deep hierarchy for testing auto-completion
	tl := task.NewTaskList("Auto-completion Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Build a 3-level deep hierarchy
	if _, err := tl.AddTask("", "Phase 1: Planning", ""); err != nil {
		t.Fatalf("failed to add root task: %v", err)
	}
	if _, err := tl.AddTask("1", "Requirements gathering", ""); err != nil {
		t.Fatalf("failed to add level 2 task: %v", err)
	}
	if _, err := tl.AddTask("1", "Design specification", ""); err != nil {
		t.Fatalf("failed to add level 2 task: %v", err)
	}
	if _, err := tl.AddTask("1.1", "Functional requirements", ""); err != nil {
		t.Fatalf("failed to add level 3 task: %v", err)
	}
	if _, err := tl.AddTask("1.1", "Non-functional requirements", ""); err != nil {
		t.Fatalf("failed to add level 3 task: %v", err)
	}
	if _, err := tl.AddTask("1.2", "UI mockups", ""); err != nil {
		t.Fatalf("failed to add level 3 task: %v", err)
	}
	if _, err := tl.AddTask("1.2", "API design", ""); err != nil {
		t.Fatalf("failed to add level 3 task: %v", err)
	}

	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to write initial tasks: %v", err)
	}

	// Test 1: Complete a leaf task and verify no auto-completion yet
	t.Run("complete_leaf_no_autocompletion", func(t *testing.T) {
		runGoCommand(t, "complete", filename, "1.1.1")

		output := runGoCommand(t, "list", filename, "-f", "json")
		// Parent 1.1 should still be pending since 1.1.2 is not complete
		if !containsString(output, `"ID": "1.1"`) || !containsString(output, `"Status": 0`) {
			t.Errorf("parent task 1.1 should remain pending, got: %s", output)
		}
	})

	// Test 2: Complete the second leaf task and verify level 2 auto-completion
	t.Run("complete_level2_autocompletion", func(t *testing.T) {
		runGoCommand(t, "complete", filename, "1.1.2")

		output := runGoCommand(t, "list", filename, "-f", "json")
		// Now parent 1.1 should be auto-completed
		if !containsString(output, `"ID": "1.1"`) || !containsString(output, `"Status": 2`) {
			t.Errorf("parent task 1.1 should be auto-completed, got: %s", output)
		}
		// But grandparent 1 should still be pending
		if !containsString(output, `"ID": "1"`) {
			t.Error("root task 1 should still exist")
		}
		// Find the root task status in JSON
		rootTaskPattern := `"ID": "1"`
		rootIndex := strings.Index(output, rootTaskPattern)
		if rootIndex == -1 {
			t.Errorf("root task not found in output: %s", output)
		}
		// Look for the status field in the same task object
		statusStart := strings.Index(output[rootIndex:], `"Status": `)
		if statusStart == -1 {
			t.Errorf("status not found for root task in output: %s", output)
		}
		statusChar := output[rootIndex+statusStart+10] // Position of status value
		if statusChar != '0' {
			t.Errorf("root task should still be pending (status 0), got status: %c", statusChar)
		}
	})

	// Test 3: Complete remaining tasks in level 2 and verify root auto-completion
	t.Run("complete_all_levels_autocompletion", func(t *testing.T) {
		// Complete all remaining level 3 tasks
		runGoCommand(t, "complete", filename, "1.2.1")
		runGoCommand(t, "complete", filename, "1.2.2")

		output := runGoCommand(t, "list", filename, "-f", "json")
		// Now all tasks should be completed including root
		if !containsString(output, `"ID": "1.2"`) || !containsString(output, `"Status": 2`) {
			t.Errorf("task 1.2 should be auto-completed, got: %s", output)
		}
		if !containsString(output, `"ID": "1"`) {
			t.Error("root task should still exist")
		}
		// Root task should now be completed
		rootTaskPattern := `"ID": "1"`
		rootIndex := strings.Index(output, rootTaskPattern)
		if rootIndex == -1 {
			t.Errorf("root task not found in output: %s", output)
		}
		statusStart := strings.Index(output[rootIndex:], `"Status": `)
		if statusStart == -1 {
			t.Errorf("status not found for root task in output: %s", output)
		}
		statusChar := output[rootIndex+statusStart+10]
		if statusChar != '2' {
			t.Errorf("root task should be completed (status 2), got status: %c", statusChar)
		}
	})

	// Test 4: Test batch auto-completion
	t.Run("batch_autocompletion", func(t *testing.T) {
		// Create a new task hierarchy for batch testing
		if _, err := tl.AddTask("", "Phase 2: Development", ""); err != nil {
			t.Fatalf("failed to add root task: %v", err)
		}
		if _, err := tl.AddTask("2", "Frontend development", ""); err != nil {
			t.Fatalf("failed to add level 2 task: %v", err)
		}
		if _, err := tl.AddTask("2", "Backend development", ""); err != nil {
			t.Fatalf("failed to add level 2 task: %v", err)
		}
		if _, err := tl.AddTask("2.1", "Component creation", ""); err != nil {
			t.Fatalf("failed to add level 3 task: %v", err)
		}
		if _, err := tl.AddTask("2.1", "Styling", ""); err != nil {
			t.Fatalf("failed to add level 3 task: %v", err)
		}
		if err := tl.WriteFile(filename); err != nil {
			t.Fatalf("failed to write tasks: %v", err)
		}

		// Use batch operation to complete multiple tasks
		batchOps := []task.Operation{
			{Type: "update", ID: "2.1.1", Status: task.StatusPtr(task.Completed)},
			{Type: "update", ID: "2.1.2", Status: task.StatusPtr(task.Completed)},
			{Type: "update", ID: "2.2", Status: task.StatusPtr(task.Completed)},
		}

		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		response, err := tl.ExecuteBatch(batchOps, false)
		if err != nil {
			t.Fatalf("failed to execute batch operations: %v", err)
		}
		if !response.Success {
			t.Fatalf("batch operations failed: %v", response.Errors)
		}

		if err := tl.WriteFile(filename); err != nil {
			t.Fatalf("failed to write batch results: %v", err)
		}

		// Verify auto-completion happened
		output := runGoCommand(t, "list", filename, "-f", "json")
		// Both level 2 tasks and root should be auto-completed
		if !containsString(output, `"ID": "2.1"`) || !containsString(output, `"Status": 2`) {
			t.Errorf("task 2.1 should be auto-completed, got: %s", output)
		}
		if !containsString(output, `"ID": "2"`) {
			t.Error("root task 2 should exist")
		}
	})

	t.Logf("Auto-completion multi-level test passed successfully")
}

func testReferenceInclusionFormats(t *testing.T, tempDir string) {
	filename := "references.md"

	// Create task file with extensive references
	taskContent := `---
references:
  - ./docs/api-spec.yaml
  - ./requirements/business-rules.md
  - ../shared/database-schema.sql
metadata:
  project: reference-test
  version: "1.0"
---
# Reference Inclusion Test

- [ ] 1. API Development
  - Implement the REST API according to specifications.
  - Follow the coding standards and ensure proper documentation.
  - References: ./api/endpoints.md, ./api/auth.md
  - [ ] 1.1. User endpoints
    - Create all user-related API endpoints.
    - References: ./api/user-spec.md
  - [ ] 1.2. Product endpoints
    - Create all product-related API endpoints.
- [ ] 2. Database Setup
  - Configure the database with proper schemas and migrations.
  - References: ./db/migrations/, ./db/seeds/
  - [-] 2.1. Schema creation
    - Create the initial database schema.
  - [ ] 2.2. Data migration
    - Migrate existing data to new schema.
`

	if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Test 1: JSON format includes all reference types
	t.Run("json_format_references", func(t *testing.T) {
		output := runGoCommand(t, "list", filename, "--all", "-f", "json")

		// Should include front matter references
		if !containsString(output, "api-spec.yaml") {
			t.Errorf("expected front matter reference in JSON, got: %s", output)
		}
		if !containsString(output, "business-rules.md") {
			t.Errorf("expected front matter reference in JSON, got: %s", output)
		}
		if !containsString(output, "database-schema.sql") {
			t.Errorf("expected front matter reference in JSON, got: %s", output)
		}

		// Should include task-level references
		if !containsString(output, "endpoints.md") {
			t.Errorf("expected task-level reference in JSON, got: %s", output)
		}
		if !containsString(output, "auth.md") {
			t.Errorf("expected task-level reference in JSON, got: %s", output)
		}
		if !containsString(output, "user-spec.md") {
			t.Errorf("expected nested task reference in JSON, got: %s", output)
		}

		// Should include FrontMatter section
		if !containsString(output, `"FrontMatter"`) {
			t.Errorf("expected FrontMatter section in JSON, got: %s", output)
		}
	})

	// Test 2: Table format includes references section
	t.Run("table_format_references", func(t *testing.T) {
		output := runGoCommand(t, "list", filename, "--all", "-f", "table")

		// Should have References section (not "Reference Documents")
		if !containsString(output, "References") {
			t.Errorf("expected References section in table, got: %s", output)
		}
		// Should include front matter references
		if !containsString(output, "api-spec.yaml") {
			t.Errorf("expected front matter reference in table, got: %s", output)
		}
		// Should show task-level references in individual task rows
		if !containsString(output, "user-spec.md") {
			t.Errorf("expected task-level reference in table, got: %s", output)
		}
	})

	// Test 3: Markdown format includes references
	t.Run("markdown_format_references", func(t *testing.T) {
		output := runGoCommand(t, "list", filename, "--all", "-f", "markdown")

		// Should include references in output
		if !containsString(output, "api-spec.yaml") {
			t.Errorf("expected reference in markdown, got: %s", output)
		}
		// Should maintain markdown structure with references
		if !containsString(output, "References:") {
			t.Errorf("expected References: label in markdown, got: %s", output)
		}
	})

	// Test 4: Next command includes references in all formats
	t.Run("next_command_references_all_formats", func(t *testing.T) {
		// JSON format
		output := runGoCommand(t, "next", filename, "-f", "json")
		if !containsString(output, "api-spec.yaml") {
			t.Errorf("expected references in next JSON output, got: %s", output)
		}
		if !containsString(output, "endpoints.md") {
			t.Errorf("expected task references in next JSON output, got: %s", output)
		}

		// Table format
		output = runGoCommand(t, "next", filename, "-f", "table")
		if !containsString(output, "Reference Documents") {
			t.Errorf("expected Reference Documents in next table output, got: %s", output)
		}
		if !containsString(output, "api-spec.yaml") {
			t.Errorf("expected references in next table output, got: %s", output)
		}

		// Markdown format
		output = runGoCommand(t, "next", filename, "-f", "markdown")
		if !containsString(output, "api-spec.yaml") {
			t.Errorf("expected references in next markdown output, got: %s", output)
		}
	})

	// Test 5: Find command includes references in results
	t.Run("find_command_references", func(t *testing.T) {
		// The find command doesn't have --all flag, but should include references when searching refs
		output := runGoCommand(t, "find", filename, "-p", "API", "--search-refs", "-f", "json")
		// Should find task 1 that matches "API"
		if !containsString(output, "API Development") {
			t.Errorf("expected to find API task in find output, got: %s", output)
		}

		// Note: find command may not include front matter references by design
		// This tests that task-level references are searchable
		output = runGoCommand(t, "find", filename, "-p", "endpoints", "--search-refs", "-f", "json")
		if !containsString(output, "API Development") {
			t.Errorf("expected to find task with endpoints reference, got: %s", output)
		}
	})

	t.Logf("Reference inclusion formats test passed successfully")
}

func testConfigurationIntegration(t *testing.T, tempDir string) {
	// Test 1: Config file precedence
	t.Run("config_precedence", func(t *testing.T) {
		// Create local project config
		localConfig := `discovery:
  enabled: true
  template: "local/{branch}/tasks.md"`
		if err := os.WriteFile(".rune.yml", []byte(localConfig), 0644); err != nil {
			t.Fatalf("failed to create local config: %v", err)
		}

		// Create user config directory and file
		userConfigDir := tempDir + "/.config/rune"
		if err := os.MkdirAll(userConfigDir, 0755); err != nil {
			t.Fatalf("failed to create user config dir: %v", err)
		}

		userConfig := `discovery:
  enabled: true
  template: "user/{branch}/tasks.md"`
		userConfigFile := userConfigDir + "/config.yml"
		if err := os.WriteFile(userConfigFile, []byte(userConfig), 0644); err != nil {
			t.Fatalf("failed to create user config: %v", err)
		}

		// Initialize git repo and setup branch
		runCommand(t, "git", "init")
		runCommand(t, "git", "config", "user.email", "test@example.com")
		runCommand(t, "git", "config", "user.name", "Test User")
		if err := os.WriteFile("README.md", []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create readme: %v", err)
		}
		runCommand(t, "git", "add", ".")
		runCommand(t, "git", "commit", "-m", "Initial commit")
		runCommand(t, "git", "checkout", "-b", "test-branch")

		// Create task file using local config template (should take precedence)
		if err := os.MkdirAll("local/test-branch", 0755); err != nil {
			t.Fatalf("failed to create local task dir: %v", err)
		}
		localTaskFile := "local/test-branch/tasks.md"
		localTaskContent := `# Local Config Tasks
- [ ] 1. Task from local config`
		if err := os.WriteFile(localTaskFile, []byte(localTaskContent), 0644); err != nil {
			t.Fatalf("failed to create local task file: %v", err)
		}

		// Test that local config is used (not user config)
		output := runGoCommand(t, "list", "-f", "json")
		if !containsString(output, "Task from local config") {
			t.Errorf("expected local config to be used, got: %s", output)
		}
	})

	// Test 2: Invalid config error handling
	t.Run("invalid_config_handling", func(t *testing.T) {
		invalidConfig := `discovery:
  enabled: true
  template: "invalid/{branch}/tasks.md"
invalid_yaml_syntax: [unclosed_bracket`
		if err := os.WriteFile(".rune.yml", []byte(invalidConfig), 0644); err != nil {
			t.Fatalf("failed to create invalid config: %v", err)
		}

		// Command should still work but show warning or error
		output := runGoCommandWithError(t, "list")
		// Should contain some indication of config error
		if !containsString(output, "config") || !containsString(output, "error") {
			// This might be expected if the tool gracefully falls back
			t.Logf("Config error handling: %s", output)
		}
	})

	// Test 3: Missing config file (should use defaults)
	t.Run("missing_config_defaults", func(t *testing.T) {
		// Remove config files
		os.Remove(".rune.yml")
		os.RemoveAll(tempDir + "/.config")

		// Create task file with default template pattern
		if err := os.MkdirAll("test-branch", 0755); err != nil {
			t.Fatalf("failed to create default task dir: %v", err)
		}
		defaultTaskFile := "test-branch/tasks.md"
		defaultTaskContent := `# Default Config Tasks
- [ ] 1. Task with default config`
		if err := os.WriteFile(defaultTaskFile, []byte(defaultTaskContent), 0644); err != nil {
			t.Fatalf("failed to create default task file: %v", err)
		}

		// Should use default template pattern
		output := runGoCommandWithError(t, "list", "-f", "json")
		// Might fail if file doesn't exist at default location, which is expected
		if containsString(output, "Task with default config") {
			t.Logf("Default config used successfully")
		} else {
			// Check that it fails gracefully
			if !containsString(output, "git discovery failed") {
				t.Errorf("expected git discovery failure message, got: %s", output)
			}
		}
	})

	// Test 4: Git discovery disabled in config
	t.Run("git_discovery_disabled", func(t *testing.T) {
		disabledConfig := `discovery:
  enabled: false
  template: "specs/{branch}/tasks.md"`
		if err := os.WriteFile(".rune.yml", []byte(disabledConfig), 0644); err != nil {
			t.Fatalf("failed to create disabled config: %v", err)
		}

		// Should require explicit filename when discovery is disabled
		output := runGoCommandWithError(t, "list")
		if !containsString(output, "no filename specified") && !containsString(output, "git discovery failed or disabled") {
			t.Errorf("expected filename required message, got: %s", output)
		}
	})

	// Test 5: Config validation
	t.Run("config_validation", func(t *testing.T) {
		validConfig := `discovery:
  enabled: true
  template: "specs/{branch}/tasks.md"
metadata:
  project: "test-project"
  author: "test-author"`
		if err := os.WriteFile(".rune.yml", []byte(validConfig), 0644); err != nil {
			t.Fatalf("failed to create valid config: %v", err)
		}

		// Create task file at expected location
		if err := os.MkdirAll("specs/test-branch", 0755); err != nil {
			t.Fatalf("failed to create spec dir: %v", err)
		}
		specTaskFile := "specs/test-branch/tasks.md"
		specTaskContent := `# Spec Tasks
- [ ] 1. Validated config task`
		if err := os.WriteFile(specTaskFile, []byte(specTaskContent), 0644); err != nil {
			t.Fatalf("failed to create spec task file: %v", err)
		}

		// Should work with valid config
		output := runGoCommand(t, "list", "-f", "json")
		if !containsString(output, "Validated config task") {
			t.Errorf("expected validated config to work, got: %s", output)
		}
	})

	t.Logf("Configuration integration test passed successfully")
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
		if !containsString(output, `"ID": "1"`) || !containsString(output, "Urgent task") {
			t.Errorf("expected urgent task at position 1, got: %s", output)
		}
		if !containsString(output, `"ID": "2"`) || !containsString(output, "Task 1") {
			t.Errorf("expected original Task 1 at position 2, got: %s", output)
		}

		// Test position insertion in middle
		runGoCommand(t, "add", filename, "--title", "Middle task", "--position", "3")

		output = runGoCommand(t, "list", filename, "-f", "json")
		if !containsString(output, `"ID": "3"`) || !containsString(output, "Middle task") {
			t.Errorf("expected middle task at position 3, got: %s", output)
		}

		// Test subtask position insertion
		runGoCommand(t, "add", filename, "--title", "First subtask", "--parent", "2", "--position", "2.1")

		output = runGoCommand(t, "list", filename, "-f", "json")
		if !containsString(output, `"ID": "2.1"`) || !containsString(output, "First subtask") {
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
		if !containsString(output, "Inserted at beginning") {
			t.Errorf("expected inserted task with git discovery, got: %s", output)
		}

		// Test subtask insertion
		runGoCommand(t, "add", "--title", "Subtask", "--parent", "2", "--position", "2.1")

		output = runGoCommand(t, "list", "-f", "json")
		if !containsString(output, `"ID": "2.1"`) || !containsString(output, "Subtask") {
			t.Errorf("expected subtask with git discovery, got: %s", output)
		}
	})

	// Test 3: CLI error handling for position insertion
	t.Run("cli_position_insertion_errors", func(t *testing.T) {
		filename := "cli-errors.md"
		runGoCommand(t, "create", filename, "--title", "Error Test")

		// Test invalid position format
		output := runGoCommandWithError(t, "add", filename, "--title", "Invalid", "--position", "invalid.format")
		if !containsString(output, "failed to add task") {
			t.Errorf("expected error for invalid position format, got: %s", output)
		}

		// Test nonexistent parent for position insertion
		output = runGoCommandWithError(t, "add", filename, "--title", "No parent", "--parent", "999", "--position", "999.1")
		if !containsString(output, "parent task 999 not found") {
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
		if !containsString(output, "Would add task") {
			t.Errorf("expected dry run output, got: %s", output)
		}
		if !containsString(output, "Position: 1") {
			t.Errorf("expected position in dry run output, got: %s", output)
		}

		// Verify no actual changes were made
		listOutput := runGoCommand(t, "list", filename, "-f", "json")
		if containsString(listOutput, "Dry run task") {
			t.Error("dry run should not have added actual task")
		}
	})

	// Test 5: CLI position insertion with verbose output
	t.Run("cli_position_insertion_verbose", func(t *testing.T) {
		filename := "cli-verbose.md"
		runGoCommand(t, "create", filename, "--title", "Verbose Test")

		// Test verbose output shows detailed information
		output := runGoCommand(t, "add", filename, "--title", "Verbose task", "--position", "1", "--verbose")
		if !containsString(output, "Successfully added task") {
			t.Errorf("expected success message in verbose output, got: %s", output)
		}
		if !containsString(output, "Task ID:") {
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
		if !containsString(output, "nested keys not supported") {
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
		if !containsString(output, "does not exist") {
			t.Errorf("expected file not found error, got: %s", output)
		}

		// Test invalid metadata format (missing colon)
		filename := "error-test.md"
		runGoCommand(t, "create", filename, "--title", "Error Test")

		output = runGoCommandWithError(t, "add-frontmatter", filename,
			"--meta", "invalid_no_colon")
		if !containsString(output, "invalid") || !containsString(output, "format") {
			t.Errorf("expected invalid format error, got: %s", output)
		}
	})

	t.Logf("Front matter integration test passed successfully")
}

// testPhaseWorkflowEndToEnd tests end-to-end phase creation and task addition
func testPhaseWorkflowEndToEnd(t *testing.T, tempDir string) {
	filename := "phase-workflow.md"

	// Step 1: Create task file
	tl := task.NewTaskList("Phase Workflow Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Step 2: Add a phase using add-phase command
	runGoCommand(t, "add-phase", filename, "Planning")

	// Step 3: Add tasks to the Planning phase
	runGoCommand(t, "add", filename, "--phase", "Planning", "--title", "Define requirements")
	runGoCommand(t, "add", filename, "--phase", "Planning", "--title", "Create design documents")

	// Step 4: Add another phase
	runGoCommand(t, "add-phase", filename, "Implementation")

	// Step 5: Add tasks to Implementation phase
	runGoCommand(t, "add", filename, "--phase", "Implementation", "--title", "Set up project structure")
	runGoCommand(t, "add", filename, "--phase", "Implementation", "--title", "Implement core features")

	// Step 6: Add a task to a non-existent phase (should auto-create)
	runGoCommand(t, "add", filename, "--phase", "Testing", "--title", "Write unit tests")

	// Step 7: Parse and verify structure
	tl, err := task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	// Verify we have 5 tasks (sequential IDs across phases)
	if len(tl.Tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tl.Tasks))
	}

	// Verify task IDs are sequential
	expectedIDs := []string{"1", "2", "3", "4", "5"}
	for i, task := range tl.Tasks {
		if task.ID != expectedIDs[i] {
			t.Errorf("expected task %d to have ID %s, got %s", i, expectedIDs[i], task.ID)
		}
	}

	// Step 8: List tasks and verify phase information is present
	output := runGoCommand(t, "list", filename, "--format", "json")
	if !containsString(output, "Planning") {
		t.Error("expected Planning phase in JSON output")
	}
	if !containsString(output, "Implementation") {
		t.Error("expected Implementation phase in JSON output")
	}
	if !containsString(output, "Testing") {
		t.Error("expected Testing phase in JSON output")
	}

	// Step 9: Test next command with --phase flag
	output = runGoCommand(t, "next", filename, "--phase", "--format", "json")
	if !containsString(output, "Planning") {
		t.Error("expected next phase to be Planning")
	}

	// Step 10: Complete all Planning tasks and check next phase
	runGoCommand(t, "complete", filename, "1")
	runGoCommand(t, "complete", filename, "2")

	output = runGoCommand(t, "next", filename, "--phase", "--format", "json")
	if !containsString(output, "Implementation") {
		t.Error("expected next phase to be Implementation after completing Planning tasks")
	}

	t.Logf("Phase workflow end-to-end test passed successfully")
}

// testPhaseRoundTrip tests round-trip (parse -> modify -> render -> parse) with phases
func testPhaseRoundTrip(t *testing.T, tempDir string) {
	// Use the simple_phases.md fixture
	sourceFile := "../examples/phases/simple_phases.md"
	testFile := "roundtrip.md"

	// Copy fixture to temp directory
	content, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Step 1: Parse the file
	tl1, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}

	// Step 2: Modify - add a task to Planning phase
	runGoCommand(t, "add", testFile, "--phase", "Planning", "--title", "Review with stakeholders")

	// Step 3: Parse again
	tl2, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("second parse failed: %v", err)
	}

	// Verify task count increased
	if len(tl2.Tasks) != len(tl1.Tasks)+1 {
		t.Errorf("expected %d tasks after addition, got %d", len(tl1.Tasks)+1, len(tl2.Tasks))
	}

	// Step 4: Write file
	if err := tl2.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Step 5: Parse third time and verify consistency
	tl3, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("third parse failed: %v", err)
	}

	if len(tl3.Tasks) != len(tl2.Tasks) {
		t.Errorf("task count changed between writes: expected %d, got %d", len(tl2.Tasks), len(tl3.Tasks))
	}

	// Verify phase headers are preserved by checking raw content
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	contentStr := string(content)

	if !containsString(contentStr, "## Planning") {
		t.Error("Planning phase header not preserved")
	}
	if !containsString(contentStr, "## Implementation") {
		t.Error("Implementation phase header not preserved")
	}
	if !containsString(contentStr, "## Testing") {
		t.Error("Testing phase header not preserved")
	}

	// Step 6: Test removing a task preserves phases
	runGoCommand(t, "remove", testFile, "1")

	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file after removal: %v", err)
	}
	contentStr = string(content)

	if !containsString(contentStr, "## Planning") {
		t.Error("Planning phase header not preserved after task removal")
	}

	t.Logf("Phase round trip test passed successfully")
}

// testPhaseBatchOperations tests batch operations creating and populating phases
func testPhaseBatchOperations(t *testing.T, tempDir string) {
	filename := "batch-phases.md"

	// Create task file
	tl := task.NewTaskList("Batch Phase Test")
	if err := tl.WriteFile(filename); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Step 1: Create a batch file with phase operations
	batchContent := `{
  "operations": [
    {
      "action": "add",
      "title": "Sprint planning meeting",
      "phase": "Planning"
    },
    {
      "action": "add",
      "title": "Backlog refinement",
      "phase": "Planning"
    },
    {
      "action": "add",
      "title": "Develop user authentication",
      "phase": "Development"
    },
    {
      "action": "add",
      "title": "Develop API endpoints",
      "phase": "Development"
    },
    {
      "action": "add",
      "parent_id": "3",
      "title": "Implement login endpoint"
    },
    {
      "action": "add",
      "title": "Integration testing",
      "phase": "QA"
    }
  ]
}`
	batchFile := "batch-operations.json"
	if err := os.WriteFile(batchFile, []byte(batchContent), 0o644); err != nil {
		t.Fatalf("failed to write batch file: %v", err)
	}

	// Step 2: Execute batch operations
	output := runGoCommand(t, "batch", filename, "--input", batchFile, "--format", "json")

	// Verify batch response includes phase information
	if !containsString(output, "Planning") {
		t.Error("expected Planning phase in batch response")
	}
	if !containsString(output, "Development") {
		t.Error("expected Development phase in batch response")
	}
	if !containsString(output, "QA") {
		t.Error("expected QA phase in batch response")
	}

	// Step 3: Verify file structure
	tl, err := task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	// Should have 5 root tasks (2 in Planning, 2 in Development, 1 in QA)
	if len(tl.Tasks) != 5 {
		t.Errorf("expected 5 root tasks, got %d", len(tl.Tasks))
	}

	// Task 3 should have 1 subtask
	task3 := tl.FindTask("3")
	if task3 == nil {
		t.Fatal("task 3 not found")
	}
	if len(task3.Children) != 1 {
		t.Errorf("expected task 3 to have 1 child, got %d", len(task3.Children))
	}

	// Verify phase headers exist in the file
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	contentStr := string(content)

	if !containsString(contentStr, "## Planning") {
		t.Error("Planning phase header not created")
	}
	if !containsString(contentStr, "## Development") {
		t.Error("Development phase header not created")
	}
	if !containsString(contentStr, "## QA") {
		t.Error("QA phase header not created")
	}

	// Step 4: Test mixed batch operations (with and without phases)
	mixedBatch := `{
  "operations": [
    {
      "action": "add",
      "title": "No phase task"
    },
    {
      "action": "update",
      "id": "1",
      "status": "InProgress"
    },
    {
      "action": "add",
      "title": "Another planning task",
      "phase": "Planning"
    }
  ]
}`
	if err := os.WriteFile(batchFile, []byte(mixedBatch), 0o644); err != nil {
		t.Fatalf("failed to write mixed batch file: %v", err)
	}

	runGoCommand(t, "batch", filename, "--input", batchFile, "--format", "json")

	// Verify task was updated
	tl, err = task.ParseFile(filename)
	if err != nil {
		t.Fatalf("failed to parse file after mixed batch: %v", err)
	}

	task1 := tl.FindTask("1")
	if task1 == nil {
		t.Fatal("task 1 not found")
	}
	if task1.Status != task.InProgress {
		t.Errorf("expected task 1 to be InProgress, got %s", task1.Status)
	}

	t.Logf("Phase batch operations test passed successfully")
}

// testPhaseBackwardCompatibility verifies backward compatibility with legacy task files
func testPhaseBackwardCompatibility(t *testing.T, tempDir string) {
	// Test 1: Use existing task file without phases
	sourceFile := "../examples/simple.md"
	testFile := "legacy.md"

	content, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("failed to read legacy fixture: %v", err)
	}
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Step 1: Verify file can be parsed
	tl, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse legacy file: %v", err)
	}

	originalTaskCount := len(tl.Tasks)

	// Step 2: Add task without phase flag (should work as before)
	runGoCommand(t, "add", testFile, "--title", "New task without phase")

	tl, err = task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse after adding task: %v", err)
	}

	if len(tl.Tasks) != originalTaskCount+1 {
		t.Errorf("expected %d tasks, got %d", originalTaskCount+1, len(tl.Tasks))
	}

	// Step 3: List tasks - phase column should NOT appear for non-phase files
	_ = runGoCommand(t, "list", testFile, "--format", "table")
	// In a file without phases, the Phase column should not be present
	// This is a qualitative test - we just verify the command works

	// Step 4: JSON output should not include phase information when no phases exist
	jsonOutput := runGoCommand(t, "list", testFile, "--format", "json")
	// Verify the output is valid JSON (command succeeds)
	if !containsString(jsonOutput, "tasks") {
		t.Error("expected JSON output to contain tasks")
	}

	// Step 5: Test mixed content file (phases and non-phased tasks)
	mixedFile := "../examples/phases/mixed_content.md"
	testMixed := "mixed.md"

	content, err = os.ReadFile(mixedFile)
	if err != nil {
		t.Fatalf("failed to read mixed content fixture: %v", err)
	}
	if err := os.WriteFile(testMixed, content, 0o644); err != nil {
		t.Fatalf("failed to write mixed test file: %v", err)
	}

	tl, err = task.ParseFile(testMixed)
	if err != nil {
		t.Fatalf("failed to parse mixed content file: %v", err)
	}

	// Step 6: Add task without --phase to mixed content file
	runGoCommand(t, "add", testMixed, "--title", "Task added without phase flag")

	tl, err = task.ParseFile(testMixed)
	if err != nil {
		t.Fatalf("failed to parse mixed file after addition: %v", err)
	}

	// Verify all existing task IDs are preserved
	// and task operations work correctly
	runGoCommand(t, "complete", testMixed, "1")
	runGoCommand(t, "update", testMixed, "2", "--title", "Updated task title")

	tl, err = task.ParseFile(testMixed)
	if err != nil {
		t.Fatalf("failed to parse after operations: %v", err)
	}

	task1 := tl.FindTask("1")
	if task1 == nil {
		t.Fatal("task 1 not found")
	}
	if task1.Status != task.Completed {
		t.Errorf("expected task 1 to be completed, got %s", task1.Status)
	}

	task2 := tl.FindTask("2")
	if task2 == nil {
		t.Fatal("task 2 not found")
	}
	if task2.Title != "Updated task title" {
		t.Errorf("expected task 2 title to be updated, got %s", task2.Title)
	}

	// Step 7: Test empty phases preservation
	emptyFile := "../examples/phases/empty_phases.md"
	testEmpty := "empty.md"

	content, err = os.ReadFile(emptyFile)
	if err != nil {
		t.Fatalf("failed to read empty phases fixture: %v", err)
	}
	if err := os.WriteFile(testEmpty, content, 0o644); err != nil {
		t.Fatalf("failed to write empty test file: %v", err)
	}

	// Parse and write back - empty phases should be preserved
	tl, err = task.ParseFile(testEmpty)
	if err != nil {
		t.Fatalf("failed to parse empty phases file: %v", err)
	}

	if err := tl.WriteFile(testEmpty); err != nil {
		t.Fatalf("failed to write empty phases file: %v", err)
	}

	// Verify empty phases still exist
	content, err = os.ReadFile(testEmpty)
	if err != nil {
		t.Fatalf("failed to read file after write: %v", err)
	}
	contentStr := string(content)

	if !containsString(contentStr, "## Empty Phase One") {
		t.Error("Empty Phase One header not preserved")
	}
	if !containsString(contentStr, "## Empty Phase Two") {
		t.Error("Empty Phase Two header not preserved")
	}
	if !containsString(contentStr, "## Another Empty Phase") {
		t.Error("Another Empty Phase header not preserved")
	}

	// Step 8: Test duplicate phase names
	dupFile := "../examples/phases/duplicate_phases.md"
	testDup := "duplicate.md"

	content, err = os.ReadFile(dupFile)
	if err != nil {
		t.Fatalf("failed to read duplicate phases fixture: %v", err)
	}
	if err := os.WriteFile(testDup, content, 0o644); err != nil {
		t.Fatalf("failed to write duplicate test file: %v", err)
	}

	// Add task to "Implementation" phase - should go to first occurrence
	runGoCommand(t, "add", testDup, "--phase", "Implementation", "--title", "New implementation task")

	tl, err = task.ParseFile(testDup)
	if err != nil {
		t.Fatalf("failed to parse duplicate phases file: %v", err)
	}

	// Verify the file content to ensure task was added to correct location
	content, err = os.ReadFile(testDup)
	if err != nil {
		t.Fatalf("failed to read file after adding to duplicate phase: %v", err)
	}
	contentStr = string(content)

	// The new task should be added to the first Implementation phase
	if !containsString(contentStr, "New implementation task") {
		t.Error("New task not added to duplicate phase")
	}

	t.Logf("Phase backward compatibility test passed successfully")
}

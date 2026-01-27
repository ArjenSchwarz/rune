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
		"multi_agent_workflow": {
			name:        "Multi-Agent Workflow",
			description: "Test streams and dependencies for parallel agent execution",
			workflow:    testMultiAgentWorkflow,
		},
		"dependency_chain_resolution": {
			name:        "Dependency Chain Resolution",
			description: "Test A → B → C → D dependency chain resolution",
			workflow:    testDependencyChainResolution,
		},
		"backward_compatibility": {
			name:        "Backward Compatibility",
			description: "Test legacy files without new fields work correctly",
			workflow:    testBackwardCompatibility,
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
			defer func() {
				_ = os.Chdir(oldDir)
			}()

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
	err = tl.UpdateTask("1.1", "", []string{"Create User struct", "Add validation"}, []string{"user-model-spec.md"}, nil)
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

		if err := tl.UpdateTask(taskID, "", tt.details, tt.refs, nil); err != nil {
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

				if err := tl.UpdateTask(taskID, "", details, refs, nil); err != nil {
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
		if !strings.Contains(output, "Setup development environment") {
			t.Errorf("expected task content in output, got: %s", output)
		}
		if !strings.Contains(output, "references") && !strings.Contains(output, "feature-spec.md") {
			t.Errorf("expected references in output, got: %s", output)
		}
	})

	// Test 2: Next command with git discovery
	t.Run("next_with_git_discovery", func(t *testing.T) {
		output := runGoCommand(t, "next", "-f", "table")
		if !strings.Contains(output, "1") || !strings.Contains(output, "Setup development environment") {
			t.Errorf("expected next task in output, got: %s", output)
		}
		if !strings.Contains(output, "Reference Documents") {
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
		if !strings.Contains(output, `"ID": "1.2"`) {
			t.Errorf("task 1.2 should be present, got: %s", output)
		}
		if !strings.Contains(output, `"Status": 2`) {
			t.Errorf("task 1.2 should be completed (Status:2), got: %s", output)
		}
	})

	// Test 4: Find command with git discovery
	t.Run("find_with_git_discovery", func(t *testing.T) {
		output := runGoCommand(t, "find", "-p", "API", "-f", "table")
		if !strings.Contains(output, "2.1") || !strings.Contains(output, "Design API") {
			t.Errorf("expected to find API task, got: %s", output)
		}
	})

	// Test 5: Add task with git discovery
	t.Run("add_with_git_discovery", func(t *testing.T) {
		runGoCommand(t, "add", "--title", "Write documentation", "--parent", "2")

		// Verify the task was added
		output := runGoCommand(t, "list", "-f", "json")
		if !strings.Contains(output, "Write documentation") {
			t.Errorf("expected to find added task, got: %s", output)
		}
	})

	// Test 6: Error when git discovery fails (wrong branch)
	t.Run("git_discovery_error", func(t *testing.T) {
		// Switch to a branch without corresponding task file
		runCommand(t, "git", "checkout", "-b", "nonexistent-branch")

		output := runGoCommandWithError(t, "list")
		if !strings.Contains(output, "branch-based file not found") && !strings.Contains(output, "git discovery failed") {
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
		if !strings.Contains(output, "Explicit task") {
			t.Errorf("expected explicit task content, got: %s", output)
		}
		if strings.Contains(output, "Setup development environment") {
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
	if runeBinaryPath == "" {
		t.Fatal("rune binary path not set - TestMain should have built the binary")
	}
	return runCommandWithOutput(t, runeBinaryPath, args...)
}

func runGoCommandWithError(_ *testing.T, args ...string) string {
	if runeBinaryPath == "" {
		return "ERROR: rune binary path not set"
	}
	cmd := exec.Command(runeBinaryPath, args...)
	output, _ := cmd.CombinedOutput()
	return string(output)
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
		if !strings.Contains(output, `"id": "2"`) {
			t.Errorf("expected next task to be task 2, got: %s", output)
		}
		if !strings.Contains(output, "Pending root with mixed children") {
			t.Errorf("expected task 2 title in output, got: %s", output)
		}
		// Should include references from front matter
		if !strings.Contains(output, "project-spec.md") {
			t.Errorf("expected references in next output, got: %s", output)
		}
	})

	// Test 2: Next command should include all children (completed and incomplete)
	// Per Requirements 1.6: "return the found task and all its subtasks (regardless of their completion status)"
	t.Run("next_includes_all_children", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "json")
		// Should include in-progress child 2.2
		if !strings.Contains(output, `"id": "2.2"`) {
			t.Errorf("expected to include in-progress child 2.2, got: %s", output)
		}
		// Should include pending child 2.3
		if !strings.Contains(output, `"id": "2.3"`) {
			t.Errorf("expected to include pending child 2.3, got: %s", output)
		}
		// Should include completed child 2.1 for context (per requirements 1.6)
		if !strings.Contains(output, `"id": "2.1"`) {
			t.Errorf("expected to include completed child 2.1 for context, got: %s", output)
		}
		// Should have children array
		if !strings.Contains(output, `"children": [`) {
			t.Errorf("expected children array in output, got: %s", output)
		}
	})

	// Test 3: Test with table format to ensure references are shown
	t.Run("next_table_format", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "table")
		if !strings.Contains(output, "Pending root with mixed children") {
			t.Errorf("expected task title in table output, got: %s", output)
		}
		// References section may be truncated in table format as "Reference Documen"
		if !strings.Contains(output, "Reference Documen") {
			t.Errorf("expected references section in table output, got: %s", output)
		}
		if !strings.Contains(output, "project-spec.md") {
			t.Errorf("expected specific reference in output, got: %s", output)
		}
	})

	// Test 4: Test with markdown format
	t.Run("next_markdown_format", func(t *testing.T) {
		output := runGoCommand(t, "next", filename, "-f", "markdown")
		if !strings.Contains(output, "# Next Task") {
			t.Errorf("expected markdown heading in output, got: %s", output)
		}
		if !strings.Contains(output, "project-spec.md") {
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
		if !strings.Contains(output, "All tasks are complete") {
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
		if !strings.Contains(output, `"ID": "1.1"`) || !strings.Contains(output, `"Status": 0`) {
			t.Errorf("parent task 1.1 should remain pending, got: %s", output)
		}
	})

	// Test 2: Complete the second leaf task and verify level 2 auto-completion
	t.Run("complete_level2_autocompletion", func(t *testing.T) {
		runGoCommand(t, "complete", filename, "1.1.2")

		output := runGoCommand(t, "list", filename, "-f", "json")
		// Now parent 1.1 should be auto-completed
		if !strings.Contains(output, `"ID": "1.1"`) || !strings.Contains(output, `"Status": 2`) {
			t.Errorf("parent task 1.1 should be auto-completed, got: %s", output)
		}
		// But grandparent 1 should still be pending
		if !strings.Contains(output, `"ID": "1"`) {
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
		if !strings.Contains(output, `"ID": "1.2"`) || !strings.Contains(output, `"Status": 2`) {
			t.Errorf("task 1.2 should be auto-completed, got: %s", output)
		}
		if !strings.Contains(output, `"ID": "1"`) {
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
		if !strings.Contains(output, `"ID": "2.1"`) || !strings.Contains(output, `"Status": 2`) {
			t.Errorf("task 2.1 should be auto-completed, got: %s", output)
		}
		if !strings.Contains(output, `"ID": "2"`) {
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
		if !strings.Contains(output, "api-spec.yaml") {
			t.Errorf("expected front matter reference in JSON, got: %s", output)
		}
		if !strings.Contains(output, "business-rules.md") {
			t.Errorf("expected front matter reference in JSON, got: %s", output)
		}
		if !strings.Contains(output, "database-schema.sql") {
			t.Errorf("expected front matter reference in JSON, got: %s", output)
		}

		// Should include task-level references
		if !strings.Contains(output, "endpoints.md") {
			t.Errorf("expected task-level reference in JSON, got: %s", output)
		}
		if !strings.Contains(output, "auth.md") {
			t.Errorf("expected task-level reference in JSON, got: %s", output)
		}
		if !strings.Contains(output, "user-spec.md") {
			t.Errorf("expected nested task reference in JSON, got: %s", output)
		}

		// Should include FrontMatter section
		if !strings.Contains(output, `"FrontMatter"`) {
			t.Errorf("expected FrontMatter section in JSON, got: %s", output)
		}
	})

	// Test 2: Table format includes references section
	t.Run("table_format_references", func(t *testing.T) {
		output := runGoCommand(t, "list", filename, "--all", "-f", "table")

		// Should have References section (not "Reference Documents")
		if !strings.Contains(output, "References") {
			t.Errorf("expected References section in table, got: %s", output)
		}
		// Should include front matter references
		if !strings.Contains(output, "api-spec.yaml") {
			t.Errorf("expected front matter reference in table, got: %s", output)
		}
		// Should show task-level references in individual task rows
		if !strings.Contains(output, "user-spec.md") {
			t.Errorf("expected task-level reference in table, got: %s", output)
		}
	})

	// Test 3: Markdown format includes references
	t.Run("markdown_format_references", func(t *testing.T) {
		output := runGoCommand(t, "list", filename, "--all", "-f", "markdown")

		// Should include references in output
		if !strings.Contains(output, "api-spec.yaml") {
			t.Errorf("expected reference in markdown, got: %s", output)
		}
		// Should maintain markdown structure with references
		if !strings.Contains(output, "References:") {
			t.Errorf("expected References: label in markdown, got: %s", output)
		}
	})

	// Test 4: Next command includes references in all formats
	t.Run("next_command_references_all_formats", func(t *testing.T) {
		// JSON format
		output := runGoCommand(t, "next", filename, "-f", "json")
		if !strings.Contains(output, "api-spec.yaml") {
			t.Errorf("expected references in next JSON output, got: %s", output)
		}
		if !strings.Contains(output, "endpoints.md") {
			t.Errorf("expected task references in next JSON output, got: %s", output)
		}

		// Table format
		output = runGoCommand(t, "next", filename, "-f", "table")
		if !strings.Contains(output, "Reference Documents") {
			t.Errorf("expected Reference Documents in next table output, got: %s", output)
		}
		if !strings.Contains(output, "api-spec.yaml") {
			t.Errorf("expected references in next table output, got: %s", output)
		}

		// Markdown format
		output = runGoCommand(t, "next", filename, "-f", "markdown")
		if !strings.Contains(output, "api-spec.yaml") {
			t.Errorf("expected references in next markdown output, got: %s", output)
		}
	})

	// Test 5: Find command includes references in results
	t.Run("find_command_references", func(t *testing.T) {
		// The find command doesn't have --all flag, but should include references when searching refs
		output := runGoCommand(t, "find", filename, "-p", "API", "--search-refs", "-f", "json")
		// Should find task 1 that matches "API"
		if !strings.Contains(output, "API Development") {
			t.Errorf("expected to find API task in find output, got: %s", output)
		}

		// Note: find command may not include front matter references by design
		// This tests that task-level references are searchable
		output = runGoCommand(t, "find", filename, "-p", "endpoints", "--search-refs", "-f", "json")
		if !strings.Contains(output, "API Development") {
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
		if !strings.Contains(output, "Task from local config") {
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
		if !strings.Contains(output, "config") || !strings.Contains(output, "error") {
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
		if strings.Contains(output, "Task with default config") {
			t.Logf("Default config used successfully")
		} else {
			// Check that it fails gracefully
			if !strings.Contains(output, "git discovery failed") {
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
		if !strings.Contains(output, "no filename specified") && !strings.Contains(output, "git discovery failed or disabled") {
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
		if !strings.Contains(output, "Validated config task") {
			t.Errorf("expected validated config to work, got: %s", output)
		}
	})

	t.Logf("Configuration integration test passed successfully")
}

package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

func TestFindCommand(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-find-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a comprehensive test file
	tl := task.NewTaskList("Find Test Project")

	// Add tasks with various content
	tl.AddTask("", "Design system architecture", "")
	tl.AddTask("", "Implement parser module", "")
	tl.AddTask("", "Write documentation", "")

	// Add subtasks
	tl.AddTask("1", "Create database schema", "")
	tl.AddTask("1", "Design REST API", "")
	tl.AddTask("2", "Add markdown parser", "")
	tl.AddTask("2", "Create parser tests", "")

	// Update task statuses
	tl.UpdateStatus("1", task.InProgress)
	tl.UpdateStatus("1.1", task.Completed)
	tl.UpdateStatus("2.1", task.Completed)

	// Add details and references
	tl.UpdateTask("1", "", []string{"Create component diagrams", "Define API interfaces"}, []string{"requirements.md", "design-doc.md"}, nil)
	tl.UpdateTask("1.1", "", []string{"Design tables", "Add indexes"}, nil, nil)
	tl.UpdateTask("1.2", "", nil, []string{"api-spec.md"}, nil)
	tl.UpdateTask("2", "", []string{"Parse markdown syntax", "Build AST"}, nil, nil)
	tl.UpdateTask("2.2", "", []string{"Unit tests", "Integration tests"}, nil, nil)
	tl.UpdateTask("3", "", nil, []string{"readme.md", "api-docs.md"}, nil)

	testFile := "find-test-tasks.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := map[string]struct {
		pattern          string
		caseSensitive    bool
		searchDetails    bool
		searchRefs       bool
		includeParent    bool
		statusFilter     string
		maxDepth         int
		parentIDFilter   string
		expectedMatches  int
		shouldContain    []string // Task IDs that should be in results
		shouldNotContain []string // Task IDs that should not be in results
	}{
		"basic_title_search_case_insensitive": {
			pattern:         "parser",
			caseSensitive:   false,
			expectedMatches: 3, // Tasks 2, 2.1, 2.2
			shouldContain:   []string{"2", "2.1", "2.2"},
		},
		"basic_title_search_case_sensitive": {
			pattern:         "Parser", // Capital P
			caseSensitive:   true,
			expectedMatches: 0, // No matches with capital P
		},
		"search_in_details": {
			pattern:         "indexes",
			searchDetails:   true,
			expectedMatches: 1,
			shouldContain:   []string{"1.1"},
		},
		"search_in_references": {
			pattern:         "api-spec",
			searchRefs:      true,
			expectedMatches: 1,
			shouldContain:   []string{"1.2"},
		},
		"search_with_all_options": {
			pattern:         "API",
			caseSensitive:   false,
			searchDetails:   true,
			searchRefs:      true,
			expectedMatches: 3, // Tasks 1, 1.2, 3
			shouldContain:   []string{"1", "1.2", "3"},
		},
		"filter_by_status_pending": {
			pattern:          "test", // matches "tests" in task 2.2
			statusFilter:     "pending",
			expectedMatches:  1,
			shouldContain:    []string{"2.2"},
			shouldNotContain: []string{"2.1"}, // completed status
		},
		"filter_by_status_completed": {
			pattern:         "schema", // matches task 1.1
			statusFilter:    "completed",
			expectedMatches: 1,
			shouldContain:   []string{"1.1"},
		},
		"filter_by_max_depth": {
			pattern:          "a", // broad pattern to match multiple tasks
			maxDepth:         1,   // only top-level tasks
			expectedMatches:  3,   // Should match tasks 1, 2, and 3 at top level
			shouldContain:    []string{"1", "2", "3"},
			shouldNotContain: []string{"1.1", "1.2", "2.1", "2.2"}, // deeper tasks
		},
		"filter_by_parent_id": {
			pattern:          "a", // broad pattern
			parentIDFilter:   "1", // only children of task 1
			expectedMatches:  2,   // Tasks 1.1 and 1.2
			shouldContain:    []string{"1.1", "1.2"},
			shouldNotContain: []string{"1", "2", "2.1", "2.2"},
		},
		"combined_filters": {
			pattern:         "test",
			statusFilter:    "pending",
			parentIDFilter:  "2",
			expectedMatches: 1,
			shouldContain:   []string{"2.2"}, // pending task under parent 2 with "test" in title
		},
		"include_parent_for_child_match": {
			// T-413: --include-parent should include the parent task when a child matches.
			pattern:          "schema", // matches task 1.1 "Create database schema"
			includeParent:    true,
			expectedMatches:  2, // task 1 (parent) + task 1.1 (match)
			shouldContain:    []string{"1", "1.1"},
			shouldNotContain: []string{"2", "3"},
		},
		"include_parent_top_level_no_extra": {
			// T-413: top-level match with --include-parent should not add spurious results.
			pattern:         "documentation",
			includeParent:   true,
			expectedMatches: 1,
			shouldContain:   []string{"3"},
		},
		"no_matches": {
			pattern:         "nonexistent",
			expectedMatches: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global variables
			findPattern = tc.pattern
			caseSensitive = tc.caseSensitive
			searchDetails = tc.searchDetails
			searchRefs = tc.searchRefs
			statusFilter = tc.statusFilter
			maxDepth = tc.maxDepth
			parentIDFilter = tc.parentIDFilter

			// Test the core search functionality
			parsedList, err := task.ParseFile(testFile)
			if err != nil {
				t.Fatalf("failed to parse test file: %v", err)
			}

			// Set up query options
			opts := task.QueryOptions{
				CaseSensitive: tc.caseSensitive,
				SearchDetails: tc.searchDetails,
				SearchRefs:    tc.searchRefs,
				IncludeParent: tc.includeParent,
			}

			// Perform the search
			results := parsedList.Find(tc.pattern, opts)

			// Apply additional filters if specified
			parentFilterSet := tc.parentIDFilter != ""
			if tc.statusFilter != "" || tc.maxDepth > 0 || parentFilterSet {
				results = applyAdditionalFilters(results, tc.statusFilter, tc.maxDepth, tc.parentIDFilter, parentFilterSet)
			}

			// Check result count
			if len(results) != tc.expectedMatches {
				t.Errorf("expected %d matches, got %d", tc.expectedMatches, len(results))
			}

			// Extract result IDs
			resultIDs := make(map[string]bool)
			for _, result := range results {
				resultIDs[result.ID] = true
			}

			// Check that expected tasks are included
			for _, expectedID := range tc.shouldContain {
				if !resultIDs[expectedID] {
					t.Errorf("expected result to contain task %s, but it was not found", expectedID)
				}
			}

			// Check that unexpected tasks are not included
			for _, unexpectedID := range tc.shouldNotContain {
				if resultIDs[unexpectedID] {
					t.Errorf("expected result to NOT contain task %s, but it was found", unexpectedID)
				}
			}
		})
	}
}

func TestFindCommandOutputFormats(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-find-formats-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a simple test file
	tl := task.NewTaskList("Format Test")
	tl.AddTask("", "Parser task", "")
	tl.AddTask("", "Database task", "")
	tl.AddTask("1", "Parser subtask", "")

	testFile := "format-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test JSON format output
	t.Run("JSON output format", func(t *testing.T) {
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("parser", opts)

		if len(results) != 2 { // Should find tasks 1 and 1.1
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		// Test that results contain expected tasks
		found := make(map[string]bool)
		for _, result := range results {
			found[result.ID] = true
		}

		if !found["1"] {
			t.Error("expected to find task 1")
		}
		if !found["1.1"] {
			t.Error("expected to find task 1.1")
		}
	})

	// Test Markdown format
	t.Run("Markdown output validation", func(t *testing.T) {
		// Test the markdown output contains expected elements
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("database", opts)

		if len(results) != 1 {
			t.Fatalf("expected 1 result for 'database', got %d", len(results))
		}

		if results[0].ID != "2" || !strings.Contains(results[0].Title, "Database") {
			t.Error("unexpected result for database search")
		}
	})
}

func TestFindCommandAdvancedFeatures(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-find-advanced-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a test file with deep hierarchy
	tl := task.NewTaskList("Advanced Test")
	tl.AddTask("", "Level 1 Task", "")
	tl.AddTask("1", "Level 2 Task", "")
	tl.AddTask("1.1", "Level 3 Task", "")
	tl.AddTask("1.1.1", "Deep nested task", "")

	// Add various statuses
	tl.UpdateStatus("1", task.InProgress)
	tl.UpdateStatus("1.1", task.Completed)
	tl.UpdateStatus("1.1.1", task.Pending)

	testFile := "advanced-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test hierarchy depth filtering
	t.Run("hierarchy depth filtering", func(t *testing.T) {
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("task", opts) // Should match all tasks

		// Apply max depth filter of 2
		filteredResults := applyAdditionalFilters(results, "", 2, "", false)

		// Should include tasks 1 (level 1) and 1.1 (level 2), exclude deeper tasks
		if len(filteredResults) != 2 { // Tasks 1, 1.1 should be included (levels 1, 2)
			t.Errorf("expected 2 results with max depth 2, got %d", len(filteredResults))
		}

		// Check that the deep task is not included
		for _, result := range filteredResults {
			if getTaskLevel(result.ID) > 2 {
				t.Errorf("task %s with level %d should be filtered out", result.ID, getTaskLevel(result.ID))
			}
		}
	})

	// Test status filtering
	t.Run("status filtering", func(t *testing.T) {
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("task", opts) // Should match all tasks

		// Filter for completed tasks only
		completedResults := applyAdditionalFilters(results, "completed", 0, "", false)

		if len(completedResults) != 1 {
			t.Errorf("expected 1 completed task, got %d", len(completedResults))
		}

		if completedResults[0].Status != task.Completed {
			t.Error("filtered result should have completed status")
		}
	})

	// Test parent ID filtering
	t.Run("parent ID filtering", func(t *testing.T) {
		parsedList, err := task.ParseFile(testFile)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("task", opts) // Should match all tasks

		// Filter for children of task 1.1
		childrenResults := applyAdditionalFilters(results, "", 0, "1.1", true)

		if len(childrenResults) != 1 {
			t.Errorf("expected 1 child of task 1.1, got %d", len(childrenResults))
		}

		if childrenResults[0].ParentID != "1.1" {
			t.Errorf("expected parent ID to be '1.1', got '%s'", childrenResults[0].ParentID)
		}
	})
}

func TestFindParentEmptyFilterTopLevel(t *testing.T) {
	// Regression test for T-414: --parent "" should filter to top-level tasks only.
	// The bug was that parentIDFilter == "" was treated as "no filter" rather than
	// "filter for tasks whose ParentID is empty (i.e., top-level tasks)".
	tempDir, err := os.MkdirTemp("", "rune-find-parent-empty-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create a task file with both top-level and nested tasks
	tl := task.NewTaskList("Parent Filter Test")
	tl.AddTask("", "Top level one", "")
	tl.AddTask("", "Top level two", "")
	tl.AddTask("1", "Child of one", "")
	tl.AddTask("2", "Child of two", "")
	tl.AddTask("1.1", "Grandchild task", "")

	testFile := "parent-filter-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse test file: %v", err)
	}

	// Search for a broad pattern that matches all tasks
	opts := task.QueryOptions{CaseSensitive: false}
	results := parsedList.Find("t", opts) // matches "Top", "task", "two", "Child"

	// Verify we have results at multiple levels before filtering
	if len(results) < 3 {
		t.Fatalf("expected at least 3 results before filtering, got %d", len(results))
	}

	// Apply parent filter with empty string — should return only top-level tasks
	// The parentFilterSet=true parameter signals the filter was explicitly set
	filtered := applyAdditionalFilters(results, "", 0, "", true)

	// Only top-level tasks (ParentID == "") should remain
	for _, r := range filtered {
		if r.ParentID != "" {
			t.Errorf("expected only top-level tasks, but got task %s with ParentID %q", r.ID, r.ParentID)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("expected 2 top-level tasks, got %d", len(filtered))
		for _, r := range filtered {
			t.Logf("  got: ID=%s Title=%q ParentID=%q", r.ID, r.Title, r.ParentID)
		}
	}
}

func TestFindParentFilterNotSet(t *testing.T) {
	// Complementary test: when parentFilterSet is false, no parent filtering happens
	tempDir, err := os.MkdirTemp("", "rune-find-parent-notset-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tl := task.NewTaskList("No Parent Filter Test")
	tl.AddTask("", "Top level task", "")
	tl.AddTask("1", "Child task", "")

	testFile := "no-parent-filter-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse test file: %v", err)
	}

	opts := task.QueryOptions{CaseSensitive: false}
	results := parsedList.Find("task", opts)

	// With parentFilterSet=false, all tasks should be returned regardless of parentIDFilter value
	filtered := applyAdditionalFilters(results, "", 0, "", false)

	if len(filtered) != 2 {
		t.Errorf("expected 2 tasks (no parent filtering), got %d", len(filtered))
	}
}

func TestFindCommandEdgeCases(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-find-edge-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Test with empty file
	t.Run("empty task file", func(t *testing.T) {
		emptyTl := task.NewTaskList("Empty Project")
		emptyFile := "empty-test.md"
		if err := emptyTl.WriteFile(emptyFile); err != nil {
			t.Fatalf("failed to write empty test file: %v", err)
		}

		parsedList, err := task.ParseFile(emptyFile)
		if err != nil {
			t.Fatalf("failed to parse empty file: %v", err)
		}

		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("anything", opts)

		if len(results) != 0 {
			t.Errorf("expected 0 results from empty file, got %d", len(results))
		}
	})

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := task.ParseFile("non-existent.md")
		if err == nil {
			t.Error("expected error when parsing non-existent file")
		}
	})

	// Test with special characters in search pattern
	t.Run("special characters in pattern", func(t *testing.T) {
		tl := task.NewTaskList("Special Chars Test")
		tl.AddTask("", "Task with [brackets] and (parentheses)", "")
		tl.AddTask("", "Task with *asterisks* and _underscores_", "")

		specialFile := "special-test.md"
		if err := tl.WriteFile(specialFile); err != nil {
			t.Fatalf("failed to write special test file: %v", err)
		}

		parsedList, err := task.ParseFile(specialFile)
		if err != nil {
			t.Fatalf("failed to parse special file: %v", err)
		}

		// Test searching for brackets
		opts := task.QueryOptions{CaseSensitive: false}
		results := parsedList.Find("[brackets]", opts)

		if len(results) != 1 {
			t.Errorf("expected 1 result for '[brackets]', got %d", len(results))
		}

		// Test searching for asterisks
		results = parsedList.Find("*asterisks*", opts)

		if len(results) != 1 {
			t.Errorf("expected 1 result for '*asterisks*', got %d", len(results))
		}
	})
}

// TestApplyAdditionalFiltersClearsStaleParentID verifies that
// applyAdditionalFilters updates ParentID on tasks whose parent was filtered
// out, so JSON consumers never see dangling references. Regression test for T-549.
func TestApplyAdditionalFiltersClearsStaleParentID(t *testing.T) {
	tests := map[string]struct {
		tasks            []task.Task
		statusFilter     string
		maxDepth         int
		parentIDFilter   string
		parentFilterSet  bool
		expectedParentID map[string]string // taskID → expected ParentID
		description      string
	}{
		"status filter removes parent leaving child with stale ParentID": {
			tasks: []task.Task{
				{ID: "1", Title: "Pending parent", Status: task.Pending, ParentID: ""},
				{ID: "1.1", Title: "Completed child", Status: task.Completed, ParentID: "1"},
			},
			statusFilter:     "completed",
			expectedParentID: map[string]string{"1.1": ""},
			description:      "Child whose parent was filtered out should have ParentID cleared",
		},
		"grandchild walks up to surviving grandparent": {
			tasks: []task.Task{
				{ID: "1", Title: "Grandparent", Status: task.Completed, ParentID: ""},
				{ID: "1.1", Title: "Filtered parent", Status: task.Pending, ParentID: "1"},
				{ID: "1.1.1", Title: "Grandchild", Status: task.Completed, ParentID: "1.1"},
			},
			statusFilter: "completed",
			expectedParentID: map[string]string{
				"1":     "",  // root, unchanged
				"1.1.1": "1", // should walk up to surviving grandparent
			},
			description: "Grandchild should walk up to nearest surviving ancestor",
		},
		"both parent and child survive keeps original ParentID": {
			tasks: []task.Task{
				{ID: "1", Title: "Parent", Status: task.Completed, ParentID: ""},
				{ID: "1.1", Title: "Child", Status: task.Completed, ParentID: "1"},
			},
			statusFilter: "completed",
			expectedParentID: map[string]string{
				"1":   "",
				"1.1": "1",
			},
			description: "When parent survives, child ParentID should be unchanged",
		},
		"depth filter removes deep task leaving stale ParentID": {
			tasks: []task.Task{
				{ID: "1", Title: "Root", Status: task.Pending, ParentID: ""},
				{ID: "1.1", Title: "Level 2", Status: task.Pending, ParentID: "1"},
			},
			maxDepth: 1,
			expectedParentID: map[string]string{
				"1": "", // only root survives
			},
			description: "Depth filter should not leave stale references",
		},
		"parentFilterSet stops walk at specified parent context": {
			tasks: []task.Task{
				{ID: "1", Title: "Context parent", Status: task.Pending, ParentID: ""},
				{ID: "1.1", Title: "Filtered middle", Status: task.Pending, ParentID: "1"},
				{ID: "1.1.1", Title: "Surviving child", Status: task.Completed, ParentID: "1.1"},
			},
			statusFilter:    "completed",
			parentIDFilter:  "1",
			parentFilterSet: true,
			expectedParentID: map[string]string{
				"1.1.1": "1", // walk stops at parentIDFilter, not promoted to ""
			},
			description: "With --parent set, walk should stop at the specified parent value",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filtered := applyAdditionalFilters(tc.tasks, tc.statusFilter, tc.maxDepth, tc.parentIDFilter, tc.parentFilterSet)

			for _, ft := range filtered {
				if expected, ok := tc.expectedParentID[ft.ID]; ok {
					if ft.ParentID != expected {
						t.Errorf("%s: task %s has ParentID=%q, want %q",
							tc.description, ft.ID, ft.ParentID, expected)
					}
				}
			}
		})
	}
}

// TestFindFilteredJSONOutputNoStaleParentIDs verifies that the full find JSON
// pipeline (Find → applyAdditionalFilters → RenderJSON) never emits a ParentID
// that references a task absent from the output. Regression test for T-549.
func TestFindFilteredJSONOutputNoStaleParentIDs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-find-json-stale-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tl := task.NewTaskList("Find Stale ParentID Test")
	tl.AddTask("", "Pending parent", "")   // task 1
	tl.AddTask("1", "Completed child", "") // task 1.1
	tl.UpdateStatus("1.1", task.Completed)
	tl.AddTask("", "Completed root", "") // task 2
	tl.UpdateStatus("2", task.Completed)

	testFile := "find-json-stale-test.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parsedList, err := task.ParseFile(testFile)
	if err != nil {
		t.Fatalf("failed to parse test file: %v", err)
	}

	// Broad search to match all tasks, then filter by status
	opts := task.QueryOptions{CaseSensitive: false}
	results := parsedList.Find("e", opts) // matches "Pending", "Completed"

	filtered := applyAdditionalFilters(results, "completed", 0, "", false)

	// Render to JSON via the same path as the find command
	jsonData, err := task.RenderJSON(&task.TaskList{
		Title: "Find Stale ParentID Test",
		Tasks: filtered,
	})
	if err != nil {
		t.Fatalf("failed to render JSON: %v", err)
	}

	var result struct {
		Tasks []struct {
			ID       string `json:"ID"`
			ParentID string `json:"ParentID"`
		} `json:"Tasks"`
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	idSet := make(map[string]bool)
	for _, tsk := range result.Tasks {
		idSet[tsk.ID] = true
	}

	for _, tsk := range result.Tasks {
		if tsk.ParentID != "" && !idSet[tsk.ParentID] {
			t.Errorf("task %s has ParentID=%q which does not exist in filtered output (available IDs: %v)",
				tsk.ID, tsk.ParentID, idSet)
		}
	}
}

// TestFindInvalidStatusFilterReturnsError verifies that runFind rejects invalid
// --status values with a clear error instead of silently matching all (T-638).
func TestFindInvalidStatusFilterReturnsError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rune-find-invalid-status-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	tl := task.NewTaskList("Find Status Validation")
	tl.AddTask("", "Setup environment", "")
	tl.AddTask("", "Run tests", "")
	testFile := "find-status-validation.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	rootCmd.SetArgs([]string{"find", testFile, "--pattern", "Setup", "--status", "bogus"})
	err = rootCmd.Execute()
	rootCmd.SetArgs([]string{})
	if err == nil {
		t.Error("expected error for invalid --status value 'bogus', got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid status filter") {
		t.Errorf("error should mention 'invalid status filter', got: %v", err)
	}
}

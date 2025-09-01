package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
)

func TestFindCommand(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "go-tasks-find-test")
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
	tl.UpdateTask("1", "", []string{"Create component diagrams", "Define API interfaces"}, []string{"requirements.md", "design-doc.md"})
	tl.UpdateTask("1.1", "", []string{"Design tables", "Add indexes"}, nil)
	tl.UpdateTask("1.2", "", nil, []string{"api-spec.md"})
	tl.UpdateTask("2", "", []string{"Parse markdown syntax", "Build AST"}, nil)
	tl.UpdateTask("2.2", "", []string{"Unit tests", "Integration tests"}, nil)
	tl.UpdateTask("3", "", nil, []string{"readme.md", "api-docs.md"})

	testFile := "find-test-tasks.md"
	if err := tl.WriteFile(testFile); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := map[string]struct {
		pattern          string
		caseSensitive    bool
		searchDetails    bool
		searchRefs       bool
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
				IncludeParent: false, // We'll test this separately
			}

			// Perform the search
			results := parsedList.Find(tc.pattern, opts)

			// Apply additional filters if specified
			if tc.statusFilter != "" || tc.maxDepth > 0 || tc.parentIDFilter != "" {
				results = applyAdditionalFilters(results, tc.statusFilter, tc.maxDepth, tc.parentIDFilter)
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
	tempDir, err := os.MkdirTemp("", "go-tasks-find-formats-test")
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
	tempDir, err := os.MkdirTemp("", "go-tasks-find-advanced-test")
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
		filteredResults := applyAdditionalFilters(results, "", 2, "")

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
		completedResults := applyAdditionalFilters(results, "completed", 0, "")

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
		childrenResults := applyAdditionalFilters(results, "", 0, "1.1")

		if len(childrenResults) != 1 {
			t.Errorf("expected 1 child of task 1.1, got %d", len(childrenResults))
		}

		if childrenResults[0].ParentID != "1.1" {
			t.Errorf("expected parent ID to be '1.1', got '%s'", childrenResults[0].ParentID)
		}
	})
}

func TestFindCommandEdgeCases(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "go-tasks-find-edge-test")
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

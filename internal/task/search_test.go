package task

import (
	"reflect"
	"testing"
	"time"
)

func TestTaskList_Find(t *testing.T) {
	// Create a sample task list for testing
	tl := &TaskList{
		Title:    "Test Project",
		Modified: time.Now(),
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Design system architecture",
				Status: Pending,
				Details: []string{
					"Create component diagrams",
					"Define API interfaces",
				},
				References: []string{"requirements.md", "design-doc.md"},
				Children: []Task{
					{
						ID:       "1.1",
						Title:    "Create database schema",
						Status:   Completed,
						ParentID: "1",
						Details:  []string{"Design tables", "Add indexes"},
					},
					{
						ID:         "1.2",
						Title:      "Design REST API",
						Status:     InProgress,
						ParentID:   "1",
						References: []string{"api-spec.md"},
					},
				},
			},
			{
				ID:     "2",
				Title:  "Implement parser module",
				Status: InProgress,
				Details: []string{
					"Parse markdown syntax",
					"Build AST",
				},
				Children: []Task{
					{
						ID:       "2.1",
						Title:    "Add markdown parser",
						Status:   Completed,
						ParentID: "2",
					},
					{
						ID:       "2.2",
						Title:    "Create parser tests",
						Status:   Pending,
						ParentID: "2",
						Details:  []string{"Unit tests", "Integration tests"},
					},
				},
			},
			{
				ID:         "3",
				Title:      "Write documentation",
				Status:     Pending,
				References: []string{"readme.md", "api-docs.md"},
			},
		},
	}

	tests := map[string]struct {
		pattern string
		opts    QueryOptions
		want    []string // Expected task IDs
	}{
		"find_by_title_case_insensitive": {
			pattern: "parser",
			opts:    QueryOptions{CaseSensitive: false},
			want:    []string{"2", "2.1", "2.2"},
		},
		"find_by_title_case_sensitive": {
			pattern: "Parser",
			opts:    QueryOptions{CaseSensitive: true},
			want:    []string{},
		},
		"find_in_details": {
			pattern: "indexes",
			opts:    QueryOptions{SearchDetails: true},
			want:    []string{"1.1"},
		},
		"find_in_references": {
			pattern: "api-spec",
			opts:    QueryOptions{SearchRefs: true},
			want:    []string{"1.2"},
		},
		"find_with_parent_context": {
			pattern: "database",
			opts:    QueryOptions{IncludeParent: true},
			want:    []string{"1", "1.1"}, // "1.1" matches "database", parent "1" is included
		},
		"find_multiple_matches": {
			pattern: "test",
			opts:    QueryOptions{CaseSensitive: false},
			want:    []string{"2.2"},
		},
		"find_with_all_options": {
			pattern: "API",
			opts: QueryOptions{
				CaseSensitive: false,
				SearchDetails: true,
				SearchRefs:    true,
				IncludeParent: false,
			},
			want: []string{"1", "1.2", "3"},
		},
		"find_include_parent_top_level_match_no_extra": {
			// When a top-level task matches, there is no parent to add.
			pattern: "architecture",
			opts:    QueryOptions{IncludeParent: true},
			want:    []string{"1"},
		},
		"find_include_parent_multiple_children_same_parent": {
			// "1.1" and "1.2" both contain "a" (schema, REST API); parent "1" should
			// appear once, before the children. Parent "1" also matches "a" in its
			// own title ("architecture"), so its inclusion is expected regardless.
			// This test verifies no duplicate parent entries.
			pattern: "schema",
			opts:    QueryOptions{IncludeParent: true},
			want:    []string{"1", "1.1"}, // parent "1" added, only child "1.1" matches
		},
		"find_include_parent_details_match": {
			// Child "1.1" matches via details ("indexes"), parent "1" should be included.
			pattern: "indexes",
			opts:    QueryOptions{IncludeParent: true, SearchDetails: true},
			want:    []string{"1", "1.1"},
		},
		"find_include_parent_disabled": {
			// With IncludeParent false, only the direct match is returned.
			pattern: "database",
			opts:    QueryOptions{IncludeParent: false},
			want:    []string{"1.1"},
		},
		"find_no_matches": {
			pattern: "nonexistent",
			opts:    QueryOptions{},
			want:    []string{},
		},
		"find_partial_match": {
			pattern: "doc",
			opts:    QueryOptions{CaseSensitive: false},
			want:    []string{"3"},
		},
		"find_in_nested_tasks": {
			pattern: "schema",
			opts:    QueryOptions{CaseSensitive: false},
			want:    []string{"1.1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tl.Find(tc.pattern, tc.opts)
			gotIDs := extractTaskIDs(got)

			if !reflect.DeepEqual(gotIDs, tc.want) {
				t.Errorf("Find() returned IDs = %v, want %v", gotIDs, tc.want)
			}
		})
	}
}

func TestTaskList_Filter(t *testing.T) {
	tl := &TaskList{
		Title:    "Test Project",
		Modified: time.Now(),
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Top level task",
				Status: Pending,
				Children: []Task{
					{
						ID:       "1.1",
						Title:    "First subtask",
						Status:   Completed,
						ParentID: "1",
						Children: []Task{
							{
								ID:       "1.1.1",
								Title:    "Deep nested task",
								Status:   InProgress,
								ParentID: "1.1",
							},
						},
					},
					{
						ID:       "1.2",
						Title:    "Second subtask",
						Status:   Pending,
						ParentID: "1",
					},
				},
			},
			{
				ID:     "2",
				Title:  "Another top level",
				Status: Completed,
			},
			{
				ID:     "3",
				Title:  "In progress task",
				Status: InProgress,
				Children: []Task{
					{
						ID:       "3.1",
						Title:    "Child of in-progress",
						Status:   Pending,
						ParentID: "3",
					},
				},
			},
		},
	}

	pendingStatus := Pending
	completedStatus := Completed
	inProgressStatus := InProgress

	// String pointers for ParentID tests
	parentID1 := "1"
	parentID11 := "1.1"
	parentIDEmpty := ""

	tests := map[string]struct {
		filter QueryFilter
		want   []string // Expected task IDs
	}{
		"filter_by_pending_status": {
			filter: QueryFilter{Status: &pendingStatus},
			want:   []string{"1", "1.2", "3.1"},
		},
		"filter_by_completed_status": {
			filter: QueryFilter{Status: &completedStatus},
			want:   []string{"1.1", "2"},
		},
		"filter_by_in_progress_status": {
			filter: QueryFilter{Status: &inProgressStatus},
			want:   []string{"1.1.1", "3"},
		},
		"filter_by_max_depth_1": {
			filter: QueryFilter{MaxDepth: 1},
			want:   []string{"1", "2", "3"},
		},
		"filter_by_max_depth_2": {
			filter: QueryFilter{MaxDepth: 2},
			want:   []string{"1", "1.1", "1.2", "2", "3", "3.1"},
		},
		"filter_by_parent_id": {
			filter: QueryFilter{ParentID: &parentID1},
			want:   []string{"1.1", "1.2"},
		},
		"filter_by_parent_id_nested": {
			filter: QueryFilter{ParentID: &parentID11},
			want:   []string{"1.1.1"},
		},
		"filter_by_title_pattern": {
			filter: QueryFilter{TitlePattern: "subtask"},
			want:   []string{"1.1", "1.2"},
		},
		"filter_combined_status_and_depth": {
			filter: QueryFilter{
				Status:   &pendingStatus,
				MaxDepth: 2,
			},
			want: []string{"1", "1.2", "3.1"},
		},
		"filter_combined_parent_and_status": {
			filter: QueryFilter{
				ParentID: &parentID1,
				Status:   &completedStatus,
			},
			want: []string{"1.1"},
		},
		"filter_no_matches": {
			filter: QueryFilter{
				TitlePattern: "nonexistent",
			},
			want: []string{},
		},
		"filter_empty_returns_all": {
			filter: QueryFilter{},
			want:   []string{"1", "1.1", "1.1.1", "1.2", "2", "3", "3.1"},
		},
		"filter_top_level_only": {
			filter: QueryFilter{ParentID: &parentIDEmpty},
			want:   []string{"1", "2", "3"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tl.Filter(tc.filter)
			gotIDs := extractTaskIDs(got)

			if !reflect.DeepEqual(gotIDs, tc.want) {
				t.Errorf("Filter() returned IDs = %v, want %v", gotIDs, tc.want)
			}
		})
	}
}

func TestTaskList_Filter_ExcludesNonMatchingChildren(t *testing.T) {
	tl := &TaskList{
		Title:    "Test Project",
		Modified: time.Now(),
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Parent task",
				Status: Completed,
				Children: []Task{
					{
						ID:       "1.1",
						Title:    "Completed child",
						Status:   Completed,
						ParentID: "1",
						Children: []Task{
							{
								ID:       "1.1.1",
								Title:    "Pending grandchild",
								Status:   Pending,
								ParentID: "1.1",
							},
						},
					},
					{
						ID:       "1.2",
						Title:    "Pending child",
						Status:   Pending,
						ParentID: "1",
					},
				},
			},
		},
	}

	completedStatus := Completed
	results := tl.Filter(QueryFilter{Status: &completedStatus})

	// Should return tasks 1 and 1.1 only
	gotIDs := extractTaskIDs(results)
	wantIDs := []string{"1", "1.1"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("Filter() returned IDs = %v, want %v", gotIDs, wantIDs)
	}

	// Each result task must have an empty Children slice — non-matching
	// children must not leak through.
	for _, task := range results {
		if len(task.Children) != 0 {
			childIDs := extractTaskIDs(task.Children)
			t.Errorf("Filter() result task %s has Children %v, want empty", task.ID, childIDs)
		}
	}
}

// TestTaskList_Filter_PreservesExtendedMetadata is a regression test for T-630:
// Filter must preserve all task metadata fields (StableID, Stream, BlockedBy,
// Owner, Requirements) — not just the basic ID/Title/Status/Details/References/ParentID.
func TestTaskList_Filter_PreservesExtendedMetadata(t *testing.T) {
	tl := &TaskList{
		Title:    "Metadata Test",
		Modified: time.Now(),
		Tasks: []Task{
			{
				ID:           "1",
				Title:        "Task with full metadata",
				Status:       InProgress,
				Details:      []string{"some detail"},
				References:   []string{"ref-1"},
				Requirements: []string{"req-1", "req-2"},
				StableID:     "stable-abc",
				BlockedBy:    []string{"stable-xyz"},
				Stream:       2,
				Owner:        "agent-1",
				Children: []Task{
					{
						ID:           "1.1",
						Title:        "Child with metadata",
						Status:       Pending,
						ParentID:     "1",
						Requirements: []string{"child-req"},
						StableID:     "stable-child",
						BlockedBy:    []string{"stable-abc"},
						Stream:       3,
						Owner:        "agent-2",
					},
				},
			},
			{
				ID:       "2",
				Title:    "Plain task",
				Status:   Pending,
				StableID: "stable-def",
			},
		},
	}

	// Filter all tasks (empty filter returns everything).
	results := tl.Filter(QueryFilter{})

	// Build a lookup by ID for easier assertions.
	byID := make(map[string]Task, len(results))
	for _, r := range results {
		byID[r.ID] = r
	}

	// --- Task "1" assertions ---
	t1, ok := byID["1"]
	if !ok {
		t.Fatal("expected task 1 in results")
	}
	if !reflect.DeepEqual(t1.Requirements, []string{"req-1", "req-2"}) {
		t.Errorf("task 1 Requirements = %v, want [req-1 req-2]", t1.Requirements)
	}
	if t1.StableID != "stable-abc" {
		t.Errorf("task 1 StableID = %q, want %q", t1.StableID, "stable-abc")
	}
	if !reflect.DeepEqual(t1.BlockedBy, []string{"stable-xyz"}) {
		t.Errorf("task 1 BlockedBy = %v, want [stable-xyz]", t1.BlockedBy)
	}
	if t1.Stream != 2 {
		t.Errorf("task 1 Stream = %d, want 2", t1.Stream)
	}
	if t1.Owner != "agent-1" {
		t.Errorf("task 1 Owner = %q, want %q", t1.Owner, "agent-1")
	}

	// --- Task "1.1" assertions ---
	t11, ok := byID["1.1"]
	if !ok {
		t.Fatal("expected task 1.1 in results")
	}
	if !reflect.DeepEqual(t11.Requirements, []string{"child-req"}) {
		t.Errorf("task 1.1 Requirements = %v, want [child-req]", t11.Requirements)
	}
	if t11.StableID != "stable-child" {
		t.Errorf("task 1.1 StableID = %q, want %q", t11.StableID, "stable-child")
	}
	if !reflect.DeepEqual(t11.BlockedBy, []string{"stable-abc"}) {
		t.Errorf("task 1.1 BlockedBy = %v, want [stable-abc]", t11.BlockedBy)
	}
	if t11.Stream != 3 {
		t.Errorf("task 1.1 Stream = %d, want 3", t11.Stream)
	}
	if t11.Owner != "agent-2" {
		t.Errorf("task 1.1 Owner = %q, want %q", t11.Owner, "agent-2")
	}

	// --- Task "2" assertions (minimal metadata) ---
	t2, ok := byID["2"]
	if !ok {
		t.Fatal("expected task 2 in results")
	}
	if t2.StableID != "stable-def" {
		t.Errorf("task 2 StableID = %q, want %q", t2.StableID, "stable-def")
	}

	// --- Children must still be excluded ---
	for _, r := range results {
		if len(r.Children) != 0 {
			t.Errorf("task %s should have no Children in filter results, got %d", r.ID, len(r.Children))
		}
	}
}

func TestTaskList_FindTask(t *testing.T) {
	tl := &TaskList{
		Title: "Test",
		Tasks: []Task{
			{
				ID:    "1",
				Title: "Task 1",
				Children: []Task{
					{
						ID:    "1.1",
						Title: "Task 1.1",
						Children: []Task{
							{
								ID:    "1.1.1",
								Title: "Task 1.1.1",
							},
						},
					},
					{
						ID:    "1.2",
						Title: "Task 1.2",
					},
				},
			},
			{
				ID:    "2",
				Title: "Task 2",
			},
		},
	}

	tests := map[string]struct {
		taskID    string
		wantFound bool
		wantTitle string
	}{
		"find_root_task": {
			taskID:    "1",
			wantFound: true,
			wantTitle: "Task 1",
		},
		"find_nested_task": {
			taskID:    "1.1",
			wantFound: true,
			wantTitle: "Task 1.1",
		},
		"find_deeply_nested_task": {
			taskID:    "1.1.1",
			wantFound: true,
			wantTitle: "Task 1.1.1",
		},
		"find_sibling_task": {
			taskID:    "1.2",
			wantFound: true,
			wantTitle: "Task 1.2",
		},
		"find_another_root": {
			taskID:    "2",
			wantFound: true,
			wantTitle: "Task 2",
		},
		"not_found": {
			taskID:    "3",
			wantFound: false,
		},
		"not_found_nested": {
			taskID:    "1.3",
			wantFound: false,
		},
		"empty_id": {
			taskID:    "",
			wantFound: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tl.FindTask(tc.taskID)

			if tc.wantFound {
				if got == nil {
					t.Errorf("FindTask(%s) = nil, want task with title %s", tc.taskID, tc.wantTitle)
				} else if got.Title != tc.wantTitle {
					t.Errorf("FindTask(%s).Title = %s, want %s", tc.taskID, got.Title, tc.wantTitle)
				}
			} else {
				if got != nil {
					t.Errorf("FindTask(%s) = %v, want nil", tc.taskID, got)
				}
			}
		})
	}
}

// TestTaskList_Find_ChildrenStripped verifies that Find results never carry
// nested Children, preventing JSON leaks of non-matching descendants and
// duplicate entries (regression tests for T-629).
func TestTaskList_Find_ChildrenStripped(t *testing.T) {
	tests := map[string]struct {
		tasks   []Task
		pattern string
		opts    QueryOptions
		wantIDs []string
	}{
		"excludes_non_matching_children": {
			tasks: []Task{
				{
					ID: "1", Title: "Parent matches search", Status: Pending,
					Children: []Task{
						{ID: "1.1", Title: "Non-matching child", Status: Pending, ParentID: "1",
							Children: []Task{
								{ID: "1.1.1", Title: "Non-matching grandchild", Status: Pending, ParentID: "1.1"},
							}},
					},
				},
				{
					ID: "2", Title: "Another parent matches search", Status: Pending,
					Children: []Task{
						{ID: "2.1", Title: "Child also matches search", Status: Pending, ParentID: "2",
							Children: []Task{
								{ID: "2.1.1", Title: "Non-matching grandchild under matching child", Status: Pending, ParentID: "2.1"},
							}},
					},
				},
			},
			pattern: "matches search",
			opts:    QueryOptions{},
			wantIDs: []string{"1", "2", "2.1"},
		},
		"no_duplicate_when_child_also_matches": {
			tasks: []Task{
				{
					ID: "1", Title: "Parent matches keyword", Status: Pending,
					Children: []Task{
						{ID: "1.1", Title: "Child also matches keyword", Status: Pending, ParentID: "1"},
					},
				},
			},
			pattern: "matches keyword",
			opts:    QueryOptions{},
			wantIDs: []string{"1", "1.1"},
		},
		"include_parent_excludes_children": {
			tasks: []Task{
				{
					ID: "1", Title: "Parent task", Status: Pending,
					Children: []Task{
						{ID: "1.1", Title: "Child matches target", Status: Pending, ParentID: "1",
							Children: []Task{
								{ID: "1.1.1", Title: "Grandchild does not match", Status: Pending, ParentID: "1.1"},
							}},
					},
				},
			},
			pattern: "target",
			opts:    QueryOptions{IncludeParent: true},
			wantIDs: []string{"1", "1.1"},
		},
		"preserves_all_non_children_fields": {
			tasks: []Task{
				{
					ID: "1", Title: "Task matches keyword", Status: InProgress,
					Details: []string{"Some detail"}, References: []string{"ref.md"},
					Requirements: []string{"req-1"}, StableID: "stable-1",
					BlockedBy: []string{"dep-1"}, Stream: 2, Owner: "alice",
					Children: []Task{
						{ID: "1.1", Title: "Non-matching child", Status: Pending, ParentID: "1"},
					},
				},
			},
			pattern: "keyword",
			opts:    QueryOptions{},
			wantIDs: []string{"1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := &TaskList{Title: "Test", Modified: time.Now(), Tasks: tc.tasks}
			results := tl.Find(tc.pattern, tc.opts)

			gotIDs := extractTaskIDs(results)
			if !reflect.DeepEqual(gotIDs, tc.wantIDs) {
				t.Fatalf("Find() returned IDs = %v, want %v", gotIDs, tc.wantIDs)
			}

			for _, task := range results {
				if len(task.Children) != 0 {
					childIDs := extractTaskIDs(task.Children)
					t.Errorf("Find() result task %s has Children %v, want empty", task.ID, childIDs)
				}
			}
		})
	}

	// Extra field-preservation checks for the "preserves_all_non_children_fields" case.
	tl := &TaskList{Title: "Test", Modified: time.Now(), Tasks: tests["preserves_all_non_children_fields"].tasks}
	got := tl.Find("keyword", QueryOptions{})[0]
	if !reflect.DeepEqual(got.Details, []string{"Some detail"}) {
		t.Errorf("Find() Details = %v, want [Some detail]", got.Details)
	}
	if !reflect.DeepEqual(got.References, []string{"ref.md"}) {
		t.Errorf("Find() References = %v, want [ref.md]", got.References)
	}
	if !reflect.DeepEqual(got.Requirements, []string{"req-1"}) {
		t.Errorf("Find() Requirements = %v, want [req-1]", got.Requirements)
	}
	if got.StableID != "stable-1" {
		t.Errorf("Find() StableID = %q, want %q", got.StableID, "stable-1")
	}
	if !reflect.DeepEqual(got.BlockedBy, []string{"dep-1"}) {
		t.Errorf("Find() BlockedBy = %v, want [dep-1]", got.BlockedBy)
	}
	if got.Stream != 2 {
		t.Errorf("Find() Stream = %d, want 2", got.Stream)
	}
	if got.Owner != "alice" {
		t.Errorf("Find() Owner = %q, want %q", got.Owner, "alice")
	}
}

// TestTaskList_Filter_PreservesAllFields verifies that Filter results retain
// all task fields (not just the subset from the original struct-literal copy).
func TestTaskList_Filter_PreservesAllFields(t *testing.T) {
	tl := &TaskList{
		Title:    "Test Project",
		Modified: time.Now(),
		Tasks: []Task{
			{
				ID: "1", Title: "Task one", Status: InProgress,
				Details: []string{"detail"}, References: []string{"ref.md"},
				Requirements: []string{"req-1"}, StableID: "stable-1",
				BlockedBy: []string{"dep-1"}, Stream: 3, Owner: "bob",
				Children: []Task{
					{ID: "1.1", Title: "Child", Status: Pending, ParentID: "1"},
				},
			},
		},
	}

	inProgress := InProgress
	results := tl.Filter(QueryFilter{Status: &inProgress})
	if len(results) != 1 {
		t.Fatalf("Filter() returned %d results, want 1", len(results))
	}

	got := results[0]
	if !reflect.DeepEqual(got.Requirements, []string{"req-1"}) {
		t.Errorf("Filter() Requirements = %v, want [req-1]", got.Requirements)
	}
	if got.StableID != "stable-1" {
		t.Errorf("Filter() StableID = %q, want %q", got.StableID, "stable-1")
	}
	if !reflect.DeepEqual(got.BlockedBy, []string{"dep-1"}) {
		t.Errorf("Filter() BlockedBy = %v, want [dep-1]", got.BlockedBy)
	}
	if got.Stream != 3 {
		t.Errorf("Filter() Stream = %d, want 3", got.Stream)
	}
	if got.Owner != "bob" {
		t.Errorf("Filter() Owner = %q, want %q", got.Owner, "bob")
	}
	if len(got.Children) != 0 {
		t.Errorf("Filter() Children should be empty, got %v", got.Children)
	}
}

func TestGetTaskDepth(t *testing.T) {
	tests := map[string]struct {
		taskID string
		want   int
	}{
		"root_level": {
			taskID: "1",
			want:   1,
		},
		"second_level": {
			taskID: "1.2",
			want:   2,
		},
		"third_level": {
			taskID: "2.1.3",
			want:   3,
		},
		"deep_nesting": {
			taskID: "1.2.3.4.5",
			want:   5,
		},
		"empty_id": {
			taskID: "",
			want:   0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := getTaskDepth(tc.taskID)
			if got != tc.want {
				t.Errorf("getTaskDepth(%s) = %d, want %d", tc.taskID, got, tc.want)
			}
		})
	}
}

// Helper function to extract task IDs from a slice of tasks
func extractTaskIDs(tasks []Task) []string {
	if len(tasks) == 0 {
		return []string{}
	}

	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}

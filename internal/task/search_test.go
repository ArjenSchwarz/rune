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
			want:    []string{"1.1"},
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

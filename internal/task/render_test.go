package task

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := map[string]struct {
		input       *TaskList
		wantContent string
	}{
		"empty_task_list": {
			input: &TaskList{
				Title: "Empty List",
				Tasks: []Task{},
			},
			wantContent: "# Empty List\n\n",
		},
		"simple_tasks": {
			input: &TaskList{
				Title: "Simple Tasks",
				Tasks: []Task{
					{ID: "1", Title: "First task", Status: Pending},
					{ID: "2", Title: "Second task", Status: InProgress},
					{ID: "3", Title: "Third task", Status: Completed},
				},
			},
			wantContent: `# Simple Tasks

- [ ] 1. First task
- [-] 2. Second task
- [x] 3. Third task
`,
		},
		"tasks_with_hierarchy": {
			input: &TaskList{
				Title: "Hierarchical Tasks",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Main task",
						Status: InProgress,
						Children: []Task{
							{ID: "1.1", Title: "Subtask one", Status: Completed, ParentID: "1"},
							{
								ID:       "1.2",
								Title:    "Subtask two",
								Status:   Pending,
								ParentID: "1",
								Children: []Task{
									{ID: "1.2.1", Title: "Deep subtask", Status: Pending, ParentID: "1.2"},
								},
							},
						},
					},
					{ID: "2", Title: "Another main task", Status: Pending},
				},
			},
			wantContent: `# Hierarchical Tasks

- [-] 1. Main task
  - [x] 1.1. Subtask one
  - [ ] 1.2. Subtask two
    - [ ] 1.2.1. Deep subtask
- [ ] 2. Another main task
`,
		},
		"tasks_with_details": {
			input: &TaskList{
				Title: "Tasks with Details",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Task with details",
						Status: Pending,
						Details: []string{
							"First detail point",
							"Second detail point",
							"Third detail point",
						},
					},
					{
						ID:     "2",
						Title:  "Task without details",
						Status: Pending,
					},
				},
			},
			wantContent: `# Tasks with Details

- [ ] 1. Task with details
  - First detail point
  - Second detail point
  - Third detail point
- [ ] 2. Task without details
`,
		},
		"tasks_with_references": {
			input: &TaskList{
				Title: "Tasks with References",
				Tasks: []Task{
					{
						ID:         "1",
						Title:      "Task with single reference",
						Status:     Pending,
						References: []string{"doc.md"},
					},
					{
						ID:         "2",
						Title:      "Task with multiple references",
						Status:     Pending,
						References: []string{"spec.md", "design.md", "api.json"},
					},
				},
			},
			wantContent: `# Tasks with References

- [ ] 1. Task with single reference
  - References: doc.md
- [ ] 2. Task with multiple references
  - References: spec.md, design.md, api.json
`,
		},
		"tasks_with_all_features": {
			input: &TaskList{
				Title: "Complete Task List",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Comprehensive task",
						Status: InProgress,
						Details: []string{
							"Implement parser",
							"Add unit tests",
						},
						References: []string{"requirements.md", "design.md"},
						Children: []Task{
							{
								ID:         "1.1",
								Title:      "Subtask with details",
								Status:     Completed,
								ParentID:   "1",
								Details:    []string{"Write code", "Run tests"},
								References: []string{"test-spec.md"},
							},
						},
					},
				},
			},
			wantContent: `# Complete Task List

- [-] 1. Comprehensive task
  - Implement parser
  - Add unit tests
  - References: requirements.md, design.md
  - [x] 1.1. Subtask with details
    - Write code
    - Run tests
    - References: test-spec.md
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := RenderMarkdown(tc.input)
			if string(got) != tc.wantContent {
				t.Errorf("RenderMarkdown() mismatch:\ngot:\n%s\nwant:\n%s", got, tc.wantContent)
			}
		})
	}
}

func TestRenderIndentation(t *testing.T) {
	// Test that indentation is exactly 2 spaces per level
	tl := &TaskList{
		Title: "Indentation Test",
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Level 0",
				Status: Pending,
				Children: []Task{
					{
						ID:       "1.1",
						Title:    "Level 1",
						Status:   Pending,
						ParentID: "1",
						Children: []Task{
							{
								ID:       "1.1.1",
								Title:    "Level 2",
								Status:   Pending,
								ParentID: "1.1",
								Children: []Task{
									{
										ID:       "1.1.1.1",
										Title:    "Level 3",
										Status:   Pending,
										ParentID: "1.1.1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := string(RenderMarkdown(tl))
	lines := strings.Split(got, "\n")

	// Check indentation levels
	expectedIndents := map[string]int{
		"- [ ] 1. Level 0":             0,
		"  - [ ] 1.1. Level 1":         2,
		"    - [ ] 1.1.1. Level 2":     4,
		"      - [ ] 1.1.1.1. Level 3": 6,
	}

	for expected, spaces := range expectedIndents {
		found := false
		for _, line := range lines {
			if line == expected {
				found = true
				// Check that the line has exactly the expected number of leading spaces
				if spaces > 0 && !strings.HasPrefix(line, strings.Repeat(" ", spaces)) {
					t.Errorf("Line %q should have %d leading spaces", line, spaces)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected line not found: %q", expected)
		}
	}
}

func TestRenderRoundTrip(t *testing.T) {
	// Test that parse → render → parse produces identical structure
	original := &TaskList{
		Title: "Round Trip Test",
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Main task",
				Status: InProgress,
				Details: []string{
					"Detail one",
					"Detail two",
				},
				References: []string{"ref1.md", "ref2.md"},
				Children: []Task{
					{
						ID:         "1.1",
						Title:      "Subtask",
						Status:     Completed,
						ParentID:   "1",
						Details:    []string{"Sub detail"},
						References: []string{"subref.md"},
					},
				},
			},
			{
				ID:     "2",
				Title:  "Second task",
				Status: Pending,
			},
		},
	}

	// Render to markdown
	rendered := RenderMarkdown(original)

	// Parse the rendered markdown
	parsed, err := ParseMarkdown(rendered)
	if err != nil {
		t.Fatalf("Failed to parse rendered markdown: %v", err)
	}

	// Compare structures
	if parsed.Title != original.Title {
		t.Errorf("Title mismatch: got %q, want %q", parsed.Title, original.Title)
	}

	if len(parsed.Tasks) != len(original.Tasks) {
		t.Fatalf("Task count mismatch: got %d, want %d", len(parsed.Tasks), len(original.Tasks))
	}

	// Check first task
	compareTask(t, &parsed.Tasks[0], &original.Tasks[0])
}

func compareTask(t *testing.T, got, want *Task) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("Task ID mismatch: got %q, want %q", got.ID, want.ID)
	}
	if got.Title != want.Title {
		t.Errorf("Task Title mismatch: got %q, want %q", got.Title, want.Title)
	}
	if got.Status != want.Status {
		t.Errorf("Task Status mismatch: got %v, want %v", got.Status, want.Status)
	}

	// Compare details
	if len(got.Details) != len(want.Details) {
		t.Errorf("Details count mismatch for task %s: got %d, want %d", got.ID, len(got.Details), len(want.Details))
	} else {
		for i := range got.Details {
			if got.Details[i] != want.Details[i] {
				t.Errorf("Detail[%d] mismatch for task %s: got %q, want %q", i, got.ID, got.Details[i], want.Details[i])
			}
		}
	}

	// Compare references
	if len(got.References) != len(want.References) {
		t.Errorf("References count mismatch for task %s: got %d, want %d", got.ID, len(got.References), len(want.References))
	} else {
		for i := range got.References {
			if got.References[i] != want.References[i] {
				t.Errorf("Reference[%d] mismatch for task %s: got %q, want %q", i, got.ID, got.References[i], want.References[i])
			}
		}
	}

	// Compare children recursively
	if len(got.Children) != len(want.Children) {
		t.Errorf("Children count mismatch for task %s: got %d, want %d", got.ID, len(got.Children), len(want.Children))
	} else {
		for i := range got.Children {
			compareTask(t, &got.Children[i], &want.Children[i])
		}
	}
}

func TestRenderJSON(t *testing.T) {
	tl := &TaskList{
		Title: "JSON Test",
		Tasks: []Task{
			{
				ID:         "1",
				Title:      "Test task",
				Status:     InProgress,
				Details:    []string{"Detail 1"},
				References: []string{"ref.md"},
				Children: []Task{
					{
						ID:       "1.1",
						Title:    "Subtask",
						Status:   Completed,
						ParentID: "1",
					},
				},
			},
		},
	}

	jsonBytes, err := RenderJSON(tl)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var parsed TaskList
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal rendered JSON: %v", err)
	}

	// Verify content
	if parsed.Title != tl.Title {
		t.Errorf("JSON title mismatch: got %q, want %q", parsed.Title, tl.Title)
	}
	if len(parsed.Tasks) != len(tl.Tasks) {
		t.Errorf("JSON tasks count mismatch: got %d, want %d", len(parsed.Tasks), len(tl.Tasks))
	}

	// Check indentation (should be 2 spaces)
	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, "  ") {
		t.Error("JSON should be indented with 2 spaces")
	}
}

func TestRenderEmptyTaskList(t *testing.T) {
	// Test handling of nil/empty task lists
	tests := map[string]*TaskList{
		"nil_tasks": {
			Title: "Nil Tasks Test",
			Tasks: nil,
		},
		"empty_tasks": {
			Title: "Empty Tasks Test",
			Tasks: []Task{},
		},
		"empty_title": {
			Title: "",
			Tasks: []Task{
				{ID: "1", Title: "Task", Status: Pending},
			},
		},
	}

	for name, tl := range tests {
		t.Run(name, func(t *testing.T) {
			// Should not panic
			rendered := RenderMarkdown(tl)
			if rendered == nil {
				t.Error("RenderMarkdown should not return nil")
			}

			// JSON rendering should also work
			jsonBytes, err := RenderJSON(tl)
			if err != nil {
				t.Errorf("RenderJSON() error: %v", err)
			}
			if jsonBytes == nil {
				t.Error("RenderJSON should not return nil")
			}
		})
	}
}

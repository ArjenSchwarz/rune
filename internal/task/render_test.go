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

func TestRenderTaskListReferences(t *testing.T) {
	// Test rendering of TaskList-level references from FrontMatter
	tests := map[string]struct {
		input              *TaskList
		wantMarkdownSuffix string
		wantJSONReferences []string
	}{
		"no_references": {
			input: &TaskList{
				Title: "No References",
				Tasks: []Task{
					{ID: "1", Title: "Simple task", Status: Pending},
				},
				FrontMatter: &FrontMatter{},
			},
			wantMarkdownSuffix: "",
			wantJSONReferences: nil,
		},
		"single_reference": {
			input: &TaskList{
				Title: "Single Reference",
				Tasks: []Task{
					{ID: "1", Title: "Simple task", Status: Pending},
				},
				FrontMatter: &FrontMatter{
					References: []string{"./docs/guide.md"},
				},
			},
			wantMarkdownSuffix: "", // Will be checked differently since frontmatter is at the beginning
			wantJSONReferences: []string{"./docs/guide.md"},
		},
		"multiple_references": {
			input: &TaskList{
				Title: "Multiple References",
				Tasks: []Task{
					{ID: "1", Title: "Simple task", Status: Pending},
				},
				FrontMatter: &FrontMatter{
					References: []string{"./docs/requirements.md", "./docs/design.md", "./specs/api.yaml"},
				},
			},
			wantMarkdownSuffix: "", // Will be checked differently since frontmatter is at the beginning
			wantJSONReferences: []string{"./docs/requirements.md", "./docs/design.md", "./specs/api.yaml"},
		},
		"nil_frontmatter": {
			input: &TaskList{
				Title: "Nil FrontMatter",
				Tasks: []Task{
					{ID: "1", Title: "Simple task", Status: Pending},
				},
				FrontMatter: nil,
			},
			wantMarkdownSuffix: "",
			wantJSONReferences: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test Markdown rendering
			got := string(RenderMarkdown(tc.input))

			// RenderMarkdown no longer includes front matter (that's handled by WriteFile)
			// So we should NOT see front matter in the output
			if strings.HasPrefix(got, "---\n") {
				t.Errorf("RenderMarkdown should not include front matter")
			}
			// Should not contain markdown references section
			if strings.Contains(got, "## References") {
				t.Errorf("Markdown should not contain References section when using frontmatter")
			}

			// Test JSON rendering
			jsonBytes, err := RenderJSON(tc.input)
			if err != nil {
				t.Fatalf("RenderJSON() error: %v", err)
			}

			var parsed TaskList
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			// Check FrontMatter references in JSON
			if tc.wantJSONReferences == nil {
				if parsed.FrontMatter != nil && len(parsed.FrontMatter.References) > 0 {
					t.Errorf("JSON should not contain references when none expected, got: %v", parsed.FrontMatter.References)
				}
			} else {
				if parsed.FrontMatter == nil {
					t.Errorf("JSON should contain FrontMatter with references")
				} else if len(parsed.FrontMatter.References) != len(tc.wantJSONReferences) {
					t.Errorf("JSON references count mismatch: got %d, want %d", len(parsed.FrontMatter.References), len(tc.wantJSONReferences))
				} else {
					for i, ref := range tc.wantJSONReferences {
						if parsed.FrontMatter.References[i] != ref {
							t.Errorf("JSON reference[%d] mismatch: got %q, want %q", i, parsed.FrontMatter.References[i], ref)
						}
					}
				}
			}
		})
	}
}

func TestRenderTableReferences(t *testing.T) {
	// Test that table rendering includes references section when FrontMatter has references
	// Note: This tests the data preparation, actual table output is handled by go-output/v2
	tests := map[string]struct {
		input              *TaskList
		expectReferencesIn string
	}{
		"with_references": {
			input: &TaskList{
				Title: "Test with References",
				Tasks: []Task{
					{ID: "1", Title: "Test task", Status: Pending},
				},
				FrontMatter: &FrontMatter{
					References: []string{"./docs/guide.md", "./specs/api.yaml"},
				},
			},
			expectReferencesIn: "./docs/guide.md, ./specs/api.yaml",
		},
		"without_references": {
			input: &TaskList{
				Title: "Test without References",
				Tasks: []Task{
					{ID: "1", Title: "Test task", Status: Pending},
				},
				FrontMatter: &FrontMatter{},
			},
			expectReferencesIn: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// This test verifies the helper functions used in table rendering
			// The actual table output is handled by the go-output/v2 library

			if tc.input.FrontMatter != nil && len(tc.input.FrontMatter.References) > 0 {
				refs := FormatTaskListReferences(tc.input.FrontMatter.References)
				if refs != tc.expectReferencesIn {
					t.Errorf("FormatTaskListReferences() = %q, want %q", refs, tc.expectReferencesIn)
				}
			}
		})
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

func TestRenderRequirements(t *testing.T) {
	tests := map[string]struct {
		input       *TaskList
		wantContent string
	}{
		"single_requirement": {
			input: &TaskList{
				Title:            "Tasks with Requirements",
				RequirementsFile: "requirements.md",
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Implement feature",
						Status:       Pending,
						Requirements: []string{"1.1"},
					},
				},
			},
			wantContent: `# Tasks with Requirements

- [ ] 1. Implement feature
  - Requirements: [1.1](requirements.md#1.1)
`,
		},
		"multiple_requirements": {
			input: &TaskList{
				Title:            "Tasks with Requirements",
				RequirementsFile: "specs/requirements.md",
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Implement feature",
						Status:       Pending,
						Requirements: []string{"1.1", "1.2", "2.3"},
					},
				},
			},
			wantContent: `# Tasks with Requirements

- [ ] 1. Implement feature
  - Requirements: [1.1](specs/requirements.md#1.1), [1.2](specs/requirements.md#1.2), [2.3](specs/requirements.md#2.3)
`,
		},
		"requirements_with_details_and_references": {
			input: &TaskList{
				Title:            "Complete Task",
				RequirementsFile: "requirements.md",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Complex task",
						Status: InProgress,
						Details: []string{
							"First step",
							"Second step",
						},
						Requirements: []string{"1.1", "1.2"},
						References:   []string{"design.md", "spec.md"},
					},
				},
			},
			wantContent: `# Complete Task

- [-] 1. Complex task
  - First step
  - Second step
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
  - References: design.md, spec.md
`,
		},
		"requirements_in_nested_tasks": {
			input: &TaskList{
				Title:            "Nested Tasks",
				RequirementsFile: "requirements.md",
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Parent task",
						Status:       InProgress,
						Requirements: []string{"1.1"},
						Children: []Task{
							{
								ID:           "1.1",
								Title:        "Child task",
								Status:       Completed,
								ParentID:     "1",
								Requirements: []string{"1.2", "1.3"},
							},
							{
								ID:           "1.2",
								Title:        "Another child",
								Status:       Pending,
								ParentID:     "1",
								Requirements: []string{"2.1"},
								References:   []string{"ref.md"},
							},
						},
					},
				},
			},
			wantContent: `# Nested Tasks

- [-] 1. Parent task
  - Requirements: [1.1](requirements.md#1.1)
  - [x] 1.1. Child task
    - Requirements: [1.2](requirements.md#1.2), [1.3](requirements.md#1.3)
  - [ ] 1.2. Another child
    - Requirements: [2.1](requirements.md#2.1)
    - References: ref.md
`,
		},
		"no_requirements": {
			input: &TaskList{
				Title: "Tasks without Requirements",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Simple task",
						Status: Pending,
					},
				},
			},
			wantContent: `# Tasks without Requirements

- [ ] 1. Simple task
`,
		},
		"default_requirements_file": {
			input: &TaskList{
				Title: "Default Requirements File",
				// RequirementsFile not set, should default to "requirements.md"
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Task with requirements",
						Status:       Pending,
						Requirements: []string{"1.1"},
					},
				},
			},
			wantContent: `# Default Requirements File

- [ ] 1. Task with requirements
  - Requirements: [1.1](requirements.md#1.1)
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

func TestRenderRequirementsRoundTrip(t *testing.T) {
	// Test that parse → render → parse preserves requirements
	original := &TaskList{
		Title:            "Round Trip Test",
		RequirementsFile: "specs/requirements.md",
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Main task",
				Status: InProgress,
				Details: []string{
					"Detail one",
					"Detail two",
				},
				Requirements: []string{"1.1", "1.2", "2.3"},
				References:   []string{"ref1.md", "ref2.md"},
				Children: []Task{
					{
						ID:           "1.1",
						Title:        "Subtask",
						Status:       Completed,
						ParentID:     "1",
						Details:      []string{"Sub detail"},
						Requirements: []string{"3.1"},
						References:   []string{"subref.md"},
					},
				},
			},
			{
				ID:           "2",
				Title:        "Second task",
				Status:       Pending,
				Requirements: []string{"4.1", "4.2"},
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

	// Check first task with requirements
	compareTaskWithRequirements(t, &parsed.Tasks[0], &original.Tasks[0])

	// Check second task
	compareTaskWithRequirements(t, &parsed.Tasks[1], &original.Tasks[1])
}

func compareTaskWithRequirements(t *testing.T, got, want *Task) {
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

	// Compare requirements
	if len(got.Requirements) != len(want.Requirements) {
		t.Errorf("Requirements count mismatch for task %s: got %d, want %d", got.ID, len(got.Requirements), len(want.Requirements))
	} else {
		for i := range got.Requirements {
			if got.Requirements[i] != want.Requirements[i] {
				t.Errorf("Requirement[%d] mismatch for task %s: got %q, want %q", i, got.ID, got.Requirements[i], want.Requirements[i])
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
			compareTaskWithRequirements(t, &got.Children[i], &want.Children[i])
		}
	}
}

func TestRenderRequirementsMarkdownLinkFormat(t *testing.T) {
	// Test that requirements are rendered in correct markdown link format
	tl := &TaskList{
		Title:            "Link Format Test",
		RequirementsFile: "requirements.md",
		Tasks: []Task{
			{
				ID:           "1",
				Title:        "Test task",
				Status:       Pending,
				Requirements: []string{"1.1", "2.3.4"},
			},
		},
	}

	got := string(RenderMarkdown(tl))

	// Check that requirements are rendered as markdown links
	expectedLink1 := "[1.1](requirements.md#1.1)"
	expectedLink2 := "[2.3.4](requirements.md#2.3.4)"

	if !strings.Contains(got, expectedLink1) {
		t.Errorf("Output should contain link %q, got:\n%s", expectedLink1, got)
	}
	if !strings.Contains(got, expectedLink2) {
		t.Errorf("Output should contain link %q, got:\n%s", expectedLink2, got)
	}

	// Check that requirements are comma-separated
	if !strings.Contains(got, expectedLink1+", "+expectedLink2) {
		t.Errorf("Requirements should be comma-separated, got:\n%s", got)
	}

	// Check plain text format (no italic formatting)
	if strings.Contains(got, "*Requirements:") {
		t.Errorf("Requirements should not have italic formatting, got:\n%s", got)
	}
}

func TestRenderRequirementsPositioning(t *testing.T) {
	// Test that requirements are rendered before references
	tl := &TaskList{
		Title:            "Positioning Test",
		RequirementsFile: "requirements.md",
		Tasks: []Task{
			{
				ID:     "1",
				Title:  "Task with both",
				Status: Pending,
				Details: []string{
					"Detail line",
				},
				Requirements: []string{"1.1"},
				References:   []string{"ref.md"},
			},
		},
	}

	got := string(RenderMarkdown(tl))
	lines := strings.Split(got, "\n")

	requirementsIndex := -1
	referencesIndex := -1

	for i, line := range lines {
		if strings.Contains(line, "Requirements:") {
			requirementsIndex = i
		}
		if strings.Contains(line, "References:") {
			referencesIndex = i
		}
	}

	if requirementsIndex == -1 {
		t.Error("Requirements line not found in output")
	}
	if referencesIndex == -1 {
		t.Error("References line not found in output")
	}

	if requirementsIndex >= referencesIndex {
		t.Errorf("Requirements should appear before References, got Requirements at line %d, References at line %d", requirementsIndex, referencesIndex)
	}
}

func TestRenderJSONWithRequirements(t *testing.T) {
	// Test that JSON output includes requirements fields
	// Requirement 7.1: JSON output SHALL include a "requirements" field containing requirement ID strings
	// Requirement 7.2: JSON output SHALL include a "requirements_file" field in TaskList metadata when set
	tests := map[string]struct {
		input                  *TaskList
		wantRequirementsFile   string
		wantTaskRequirements   []string
		wantChildRequirements  []string
		checkRequirementsFile  bool
		checkTaskRequirements  bool
		checkChildRequirements bool
	}{
		"task_with_requirements_and_file": {
			input: &TaskList{
				Title:            "JSON Requirements Test",
				RequirementsFile: "specs/requirements.md",
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Test task",
						Status:       InProgress,
						Details:      []string{"Detail 1"},
						Requirements: []string{"1.1", "1.2", "2.3"},
						References:   []string{"ref.md"},
					},
				},
			},
			wantRequirementsFile:  "specs/requirements.md",
			wantTaskRequirements:  []string{"1.1", "1.2", "2.3"},
			checkRequirementsFile: true,
			checkTaskRequirements: true,
		},
		"nested_tasks_with_requirements": {
			input: &TaskList{
				Title:            "Nested Requirements Test",
				RequirementsFile: "requirements.md",
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Parent task",
						Status:       InProgress,
						Requirements: []string{"1.1"},
						Children: []Task{
							{
								ID:           "1.1",
								Title:        "Child task",
								Status:       Completed,
								ParentID:     "1",
								Requirements: []string{"1.2", "1.3"},
							},
						},
					},
				},
			},
			wantRequirementsFile:   "requirements.md",
			wantTaskRequirements:   []string{"1.1"},
			wantChildRequirements:  []string{"1.2", "1.3"},
			checkRequirementsFile:  true,
			checkTaskRequirements:  true,
			checkChildRequirements: true,
		},
		"task_without_requirements": {
			input: &TaskList{
				Title: "No Requirements Test",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Simple task",
						Status: Pending,
					},
				},
			},
			checkRequirementsFile: false,
			checkTaskRequirements: false,
		},
		"empty_requirements_array": {
			input: &TaskList{
				Title:            "Empty Requirements Test",
				RequirementsFile: "requirements.md",
				Tasks: []Task{
					{
						ID:           "1",
						Title:        "Task with empty requirements",
						Status:       Pending,
						Requirements: []string{},
					},
				},
			},
			wantRequirementsFile:  "requirements.md",
			checkRequirementsFile: true,
			checkTaskRequirements: false, // Empty array should be omitted due to omitempty
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Render to JSON
			jsonBytes, err := RenderJSON(tc.input)
			if err != nil {
				t.Fatalf("RenderJSON() error: %v", err)
			}

			// Unmarshal to verify structure
			var parsed TaskList
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal rendered JSON: %v", err)
			}

			// Verify RequirementsFile field
			if tc.checkRequirementsFile {
				if parsed.RequirementsFile != tc.wantRequirementsFile {
					t.Errorf("RequirementsFile mismatch: got %q, want %q", parsed.RequirementsFile, tc.wantRequirementsFile)
				}
			}

			// Verify Task.Requirements field
			if tc.checkTaskRequirements {
				if len(parsed.Tasks) == 0 {
					t.Fatal("No tasks in parsed JSON")
				}
				task := parsed.Tasks[0]
				if len(task.Requirements) != len(tc.wantTaskRequirements) {
					t.Errorf("Task requirements count mismatch: got %d, want %d", len(task.Requirements), len(tc.wantTaskRequirements))
				}
				for i, req := range tc.wantTaskRequirements {
					if i >= len(task.Requirements) {
						t.Errorf("Missing requirement at index %d", i)
						continue
					}
					if task.Requirements[i] != req {
						t.Errorf("Requirement[%d] mismatch: got %q, want %q", i, task.Requirements[i], req)
					}
				}
			}

			// Verify Child Task.Requirements field
			if tc.checkChildRequirements {
				if len(parsed.Tasks) == 0 {
					t.Fatal("No tasks in parsed JSON")
				}
				if len(parsed.Tasks[0].Children) == 0 {
					t.Fatal("No child tasks in parsed JSON")
				}
				child := parsed.Tasks[0].Children[0]
				if len(child.Requirements) != len(tc.wantChildRequirements) {
					t.Errorf("Child requirements count mismatch: got %d, want %d", len(child.Requirements), len(tc.wantChildRequirements))
				}
				for i, req := range tc.wantChildRequirements {
					if i >= len(child.Requirements) {
						t.Errorf("Missing child requirement at index %d", i)
						continue
					}
					if child.Requirements[i] != req {
						t.Errorf("Child Requirement[%d] mismatch: got %q, want %q", i, child.Requirements[i], req)
					}
				}
			}

			// Verify JSON structure by checking raw JSON string
			jsonStr := string(jsonBytes)

			// If requirements_file is set, it should appear in JSON
			if tc.checkRequirementsFile && tc.wantRequirementsFile != "" {
				if !strings.Contains(jsonStr, `"requirements_file"`) {
					t.Error("JSON should contain 'requirements_file' field")
				}
			}

			// If task has requirements, they should appear in JSON
			if tc.checkTaskRequirements && len(tc.wantTaskRequirements) > 0 {
				if !strings.Contains(jsonStr, `"requirements"`) {
					t.Error("JSON should contain 'requirements' field in task")
				}
			}
		})
	}
}

func TestRenderMarkdownWithStableID(t *testing.T) {
	// Task 16: Test stable ID inclusion in markdown output
	tests := map[string]struct {
		input       *TaskList
		wantContent string
	}{
		"task_with_stable_id": {
			input: &TaskList{
				Title: "Tasks with Stable IDs",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "First task",
						Status:   Pending,
						StableID: "abc1234",
					},
				},
			},
			wantContent: `# Tasks with Stable IDs

- [ ] 1. First task <!-- id:abc1234 -->
`,
		},
		"task_without_stable_id": {
			input: &TaskList{
				Title: "Tasks without Stable IDs",
				Tasks: []Task{
					{
						ID:     "1",
						Title:  "Legacy task",
						Status: Pending,
						// No StableID - legacy task
					},
				},
			},
			wantContent: `# Tasks without Stable IDs

- [ ] 1. Legacy task
`,
		},
		"mixed_stable_ids": {
			input: &TaskList{
				Title: "Mixed Tasks",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Task with ID",
						Status:   Pending,
						StableID: "def5678",
					},
					{
						ID:     "2",
						Title:  "Legacy task",
						Status: InProgress,
						// No StableID
					},
					{
						ID:       "3",
						Title:    "Another with ID",
						Status:   Completed,
						StableID: "ghi9012",
					},
				},
			},
			wantContent: `# Mixed Tasks

- [ ] 1. Task with ID <!-- id:def5678 -->

- [-] 2. Legacy task

- [x] 3. Another with ID <!-- id:ghi9012 -->
`,
		},
		"nested_tasks_with_stable_ids": {
			input: &TaskList{
				Title: "Nested Tasks",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Parent task",
						Status:   InProgress,
						StableID: "par1234",
						Children: []Task{
							{
								ID:       "1.1",
								Title:    "Child task",
								Status:   Pending,
								ParentID: "1",
								StableID: "chi5678",
							},
						},
					},
				},
			},
			wantContent: `# Nested Tasks

- [-] 1. Parent task <!-- id:par1234 -->
  - [ ] 1.1. Child task <!-- id:chi5678 -->
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

func TestRenderMarkdownWithBlockedBy(t *testing.T) {
	// Task 16: Test Blocked-by formatting with title hints
	tests := map[string]struct {
		input       *TaskList
		wantContent string
	}{
		"single_blocked_by": {
			input: &TaskList{
				Title: "Tasks with Dependencies",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "First task",
						Status:   Completed,
						StableID: "abc1234",
					},
					{
						ID:        "2",
						Title:     "Second task",
						Status:    Pending,
						StableID:  "def5678",
						BlockedBy: []string{"abc1234"},
					},
				},
			},
			wantContent: `# Tasks with Dependencies

- [x] 1. First task <!-- id:abc1234 -->

- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc1234 (First task)
`,
		},
		"multiple_blocked_by": {
			input: &TaskList{
				Title: "Tasks with Multiple Dependencies",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Setup task",
						Status:   Completed,
						StableID: "set1234",
					},
					{
						ID:       "2",
						Title:    "Config task",
						Status:   Completed,
						StableID: "cfg5678",
					},
					{
						ID:        "3",
						Title:     "Main task",
						Status:    Pending,
						StableID:  "mai9012",
						BlockedBy: []string{"set1234", "cfg5678"},
					},
				},
			},
			wantContent: `# Tasks with Multiple Dependencies

- [x] 1. Setup task <!-- id:set1234 -->

- [x] 2. Config task <!-- id:cfg5678 -->

- [ ] 3. Main task <!-- id:mai9012 -->
  - Blocked-by: set1234 (Setup task), cfg5678 (Config task)
`,
		},
		"blocked_by_with_missing_reference": {
			input: &TaskList{
				Title: "Tasks with Missing Reference",
				Tasks: []Task{
					{
						ID:        "1",
						Title:     "Task blocked by unknown",
						Status:    Pending,
						StableID:  "tsk1234",
						BlockedBy: []string{"unknown1"}, // ID not in index
					},
				},
			},
			// When reference is not found, only the ID is shown
			wantContent: `# Tasks with Missing Reference

- [ ] 1. Task blocked by unknown <!-- id:tsk1234 -->
  - Blocked-by: unknown1
`,
		},
		"no_blocked_by": {
			input: &TaskList{
				Title: "Tasks without Dependencies",
				Tasks: []Task{
					{
						ID:        "1",
						Title:     "Independent task",
						Status:    Pending,
						StableID:  "ind1234",
						BlockedBy: []string{}, // Empty slice
					},
				},
			},
			wantContent: `# Tasks without Dependencies

- [ ] 1. Independent task <!-- id:ind1234 -->
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

func TestRenderMarkdownWithStream(t *testing.T) {
	// Task 16: Test Stream rendering (only when non-zero)
	tests := map[string]struct {
		input       *TaskList
		wantContent string
	}{
		"task_with_stream": {
			input: &TaskList{
				Title: "Tasks with Streams",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Stream 2 task",
						Status:   Pending,
						StableID: "str1234",
						Stream:   2,
					},
				},
			},
			wantContent: `# Tasks with Streams

- [ ] 1. Stream 2 task <!-- id:str1234 -->
  - Stream: 2
`,
		},
		"task_without_stream_explicit": {
			input: &TaskList{
				Title: "Tasks without Explicit Stream",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Default stream task",
						Status:   Pending,
						StableID: "def1234",
						Stream:   0, // Not explicitly set, should not render
					},
				},
			},
			wantContent: `# Tasks without Explicit Stream

- [ ] 1. Default stream task <!-- id:def1234 -->
`,
		},
		"task_with_stream_1_explicit": {
			input: &TaskList{
				Title: "Tasks with Explicit Stream 1",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Explicit stream 1",
						Status:   Pending,
						StableID: "exp1234",
						Stream:   1, // Explicitly set to 1, should render
					},
				},
			},
			wantContent: `# Tasks with Explicit Stream 1

- [ ] 1. Explicit stream 1 <!-- id:exp1234 -->
  - Stream: 1
`,
		},
		"multiple_streams": {
			input: &TaskList{
				Title: "Multi-Stream Tasks",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Stream 1",
						Status:   Pending,
						StableID: "s1a1234",
						Stream:   1,
					},
					{
						ID:       "2",
						Title:    "Stream 2",
						Status:   Pending,
						StableID: "s2a1234",
						Stream:   2,
					},
					{
						ID:       "3",
						Title:    "Default stream",
						Status:   Pending,
						StableID: "dfa1234",
						// Stream: 0 - not set
					},
				},
			},
			wantContent: `# Multi-Stream Tasks

- [ ] 1. Stream 1 <!-- id:s1a1234 -->
  - Stream: 1

- [ ] 2. Stream 2 <!-- id:s2a1234 -->
  - Stream: 2

- [ ] 3. Default stream <!-- id:dfa1234 -->
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

func TestRenderMarkdownWithOwner(t *testing.T) {
	// Task 16: Test Owner rendering
	tests := map[string]struct {
		input       *TaskList
		wantContent string
	}{
		"task_with_owner": {
			input: &TaskList{
				Title: "Tasks with Owner",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Owned task",
						Status:   InProgress,
						StableID: "own1234",
						Owner:    "agent-1",
					},
				},
			},
			wantContent: `# Tasks with Owner

- [-] 1. Owned task <!-- id:own1234 -->
  - Owner: agent-1
`,
		},
		"task_without_owner": {
			input: &TaskList{
				Title: "Tasks without Owner",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Unowned task",
						Status:   Pending,
						StableID: "unw1234",
						Owner:    "", // Empty - no owner
					},
				},
			},
			wantContent: `# Tasks without Owner

- [ ] 1. Unowned task <!-- id:unw1234 -->
`,
		},
		"task_with_complex_owner": {
			input: &TaskList{
				Title: "Tasks with Complex Owner",
				Tasks: []Task{
					{
						ID:       "1",
						Title:    "Task with complex owner ID",
						Status:   InProgress,
						StableID: "cmp1234",
						Owner:    "claude-code-session-abc123",
					},
				},
			},
			wantContent: `# Tasks with Complex Owner

- [-] 1. Task with complex owner ID <!-- id:cmp1234 -->
  - Owner: claude-code-session-abc123
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

func TestRenderJSONExcludesStableIDs(t *testing.T) {
	// Task 16: Test JSON output excludes stable IDs
	tl := &TaskList{
		Title: "JSON Test",
		Tasks: []Task{
			{
				ID:       "1",
				Title:    "Task with stable ID",
				Status:   Pending,
				StableID: "abc1234", // Should NOT appear in JSON
			},
			{
				ID:       "2",
				Title:    "Another task",
				Status:   InProgress,
				StableID: "def5678", // Should NOT appear in JSON
				Children: []Task{
					{
						ID:       "2.1",
						Title:    "Child task",
						Status:   Pending,
						ParentID: "2",
						StableID: "ghi9012", // Should NOT appear in JSON
					},
				},
			},
		},
	}

	jsonBytes, err := RenderJSON(tl)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// StableID field should not appear in JSON
	if strings.Contains(jsonStr, `"stableID"`) || strings.Contains(jsonStr, `"StableID"`) ||
		strings.Contains(jsonStr, `"stable_id"`) {
		t.Error("JSON should not contain stableID field")
	}

	// The actual stable ID values should not appear either
	if strings.Contains(jsonStr, "abc1234") ||
		strings.Contains(jsonStr, "def5678") ||
		strings.Contains(jsonStr, "ghi9012") {
		t.Error("JSON should not contain stable ID values")
	}
}

func TestRenderJSONBlockedByUsesHierarchicalIDs(t *testing.T) {
	// Task 16: Test JSON BlockedBy uses hierarchical IDs
	// Note: This test requires MarshalTasksJSON to be implemented
	// For now, test that the basic JSON rendering includes blockedBy field

	tl := &TaskList{
		Title: "JSON BlockedBy Test",
		Tasks: []Task{
			{
				ID:       "1",
				Title:    "First task",
				Status:   Completed,
				StableID: "abc1234",
			},
			{
				ID:        "2",
				Title:     "Second task",
				Status:    Pending,
				StableID:  "def5678",
				BlockedBy: []string{"abc1234"}, // Stable ID reference
			},
		},
	}

	jsonBytes, err := RenderJSON(tl)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	// Unmarshal to check structure
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Get Tasks array
	tasks, ok := parsed["Tasks"].([]any)
	if !ok {
		t.Fatal("Tasks field not found or wrong type")
	}

	if len(tasks) < 2 {
		t.Fatal("Expected at least 2 tasks")
	}

	// Check second task has blockedBy field
	task2, ok := tasks[1].(map[string]any)
	if !ok {
		t.Fatal("Task 2 not found or wrong type")
	}

	// blockedBy should be present (as stable IDs for now until MarshalTasksJSON is implemented)
	blockedBy, ok := task2["blockedBy"]
	if !ok {
		t.Error("Task 2 should have blockedBy field")
		return
	}

	blockedByArr, ok := blockedBy.([]any)
	if !ok {
		t.Error("blockedBy should be an array")
		return
	}

	if len(blockedByArr) != 1 {
		t.Errorf("blockedBy should have 1 element, got %d", len(blockedByArr))
	}
}

func TestRenderMarkdownMetadataOrder(t *testing.T) {
	// Test that metadata is rendered in the correct order:
	// Details, Blocked-by, Stream, Owner, Requirements, References
	tl := &TaskList{
		Title:            "Metadata Order Test",
		RequirementsFile: "requirements.md",
		Tasks: []Task{
			{
				ID:       "1",
				Title:    "Complete task",
				Status:   InProgress,
				StableID: "ord1234",
				Details: []string{
					"First detail",
					"Second detail",
				},
				BlockedBy:    []string{"dep1234"},
				Stream:       2,
				Owner:        "agent-1",
				Requirements: []string{"1.1", "1.2"},
				References:   []string{"design.md"},
			},
		},
	}

	got := string(RenderMarkdown(tl))
	lines := strings.Split(got, "\n")

	// Find indices of each metadata line
	var detailIdx, blockedByIdx, streamIdx, ownerIdx, reqIdx, refIdx int = -1, -1, -1, -1, -1, -1

	for i, line := range lines {
		switch {
		case strings.Contains(line, "First detail"):
			detailIdx = i
		case strings.Contains(line, "Blocked-by:"):
			blockedByIdx = i
		case strings.Contains(line, "Stream:"):
			streamIdx = i
		case strings.Contains(line, "Owner:"):
			ownerIdx = i
		case strings.Contains(line, "Requirements:"):
			reqIdx = i
		case strings.Contains(line, "References:"):
			refIdx = i
		}
	}

	// Verify order: Details < Blocked-by < Stream < Owner < Requirements < References
	if detailIdx == -1 || blockedByIdx == -1 || streamIdx == -1 || ownerIdx == -1 || reqIdx == -1 || refIdx == -1 {
		t.Errorf("Not all metadata fields found in output:\n%s", got)
		return
	}

	if !(detailIdx < blockedByIdx && blockedByIdx < streamIdx && streamIdx < ownerIdx && ownerIdx < reqIdx && reqIdx < refIdx) {
		t.Errorf("Metadata not in correct order. Got indices: details=%d, blocked-by=%d, stream=%d, owner=%d, requirements=%d, references=%d\nOutput:\n%s",
			detailIdx, blockedByIdx, streamIdx, ownerIdx, reqIdx, refIdx, got)
	}
}

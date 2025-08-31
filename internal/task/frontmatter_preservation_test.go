package task

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFrontMatterPreservationDuringFileOperations(t *testing.T) {
	tests := map[string]struct {
		initialContent  string
		operation       func(tl *TaskList) error
		wantFrontMatter *FrontMatter
	}{
		"add_task_preserves_front_matter": {
			initialContent: `---
references:
  - ./docs/requirements.md
  - ./specs/design.md
metadata:
  project: test-project
  created: "2024-01-30"
---
# Test Tasks

- [ ] 1. Existing task
`,
			operation: func(tl *TaskList) error {
				return tl.AddTask("", "New root task")
			},
			wantFrontMatter: &FrontMatter{
				References: []string{"./docs/requirements.md", "./specs/design.md"},
				Metadata: map[string]any{
					"project": "test-project",
					"created": "2024-01-30",
				},
			},
		},
		"add_subtask_preserves_front_matter": {
			initialContent: `---
references:
  - ./docs/api.md
---
# Test Tasks

- [ ] 1. Parent task
`,
			operation: func(tl *TaskList) error {
				return tl.AddTask("1", "New subtask")
			},
			wantFrontMatter: &FrontMatter{
				References: []string{"./docs/api.md"},
			},
		},
		"update_status_preserves_front_matter": {
			initialContent: `---
references:
  - ./reference.md
metadata:
  version: "1.0"
---
# Test Tasks

- [ ] 1. Task to complete
`,
			operation: func(tl *TaskList) error {
				return tl.UpdateStatus("1", Completed)
			},
			wantFrontMatter: &FrontMatter{
				References: []string{"./reference.md"},
				Metadata: map[string]any{
					"version": "1.0",
				},
			},
		},
		"update_task_details_preserves_front_matter": {
			initialContent: `---
references:
  - ./detailed-spec.md
---
# Test Tasks

- [ ] 1. Task to update
`,
			operation: func(tl *TaskList) error {
				return tl.UpdateTask("1", "Updated task title",
					[]string{"Detail 1", "Detail 2"},
					[]string{"new-ref.md"})
			},
			wantFrontMatter: &FrontMatter{
				References: []string{"./detailed-spec.md"},
			},
		},
		"remove_task_preserves_front_matter": {
			initialContent: `---
references:
  - ./removal-spec.md
metadata:
  author: "test-user"
---
# Test Tasks

- [ ] 1. Task to keep
- [ ] 2. Task to remove
`,
			operation: func(tl *TaskList) error {
				return tl.RemoveTask("2")
			},
			wantFrontMatter: &FrontMatter{
				References: []string{"./removal-spec.md"},
				Metadata: map[string]any{
					"author": "test-user",
				},
			},
		},
		"operations_on_file_without_front_matter": {
			initialContent: `# Test Tasks

- [ ] 1. Simple task
`,
			operation: func(tl *TaskList) error {
				return tl.AddTask("", "Another task")
			},
			wantFrontMatter: &FrontMatter{}, // Should remain empty
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary file in current directory to satisfy path validation
			tmpFile := fmt.Sprintf("test_frontmatter_%s.md", strings.ReplaceAll(name, " ", "_"))
			defer os.Remove(tmpFile)

			// Write initial content
			err := os.WriteFile(tmpFile, []byte(tc.initialContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write initial content: %v", err)
			}

			// Parse initial file
			tl, err := ParseFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to parse initial file: %v", err)
			}

			// Perform operation
			err = tc.operation(tl)
			if err != nil {
				t.Fatalf("Operation failed: %v", err)
			}

			// Write file back
			err = tl.WriteFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}

			// Read the written file and verify front matter is preserved
			writtenContent, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read written file: %v", err)
			}

			// Parse the written content to verify front matter
			parsedTl, err := ParseMarkdown(writtenContent)
			if err != nil {
				t.Fatalf("Failed to parse written content: %v", err)
			}

			// Verify front matter is preserved
			compareFrontMatter(t, parsedTl.FrontMatter, tc.wantFrontMatter)

			// Verify the file starts with front matter if expected
			contentStr := string(writtenContent)
			if len(tc.wantFrontMatter.References) > 0 || len(tc.wantFrontMatter.Metadata) > 0 {
				if !strings.HasPrefix(contentStr, "---\n") {
					t.Errorf("File should start with front matter delimiter, but got: %s",
						contentStr[:min(50, len(contentStr))])
				}
				// Check that all expected references are present
				for _, ref := range tc.wantFrontMatter.References {
					if !strings.Contains(contentStr, ref) {
						t.Errorf("File should contain reference %q", ref)
					}
				}
			} else {
				// File without front matter should not start with ---
				if strings.HasPrefix(contentStr, "---\n") {
					t.Errorf("File without front matter should not start with ---, but got: %s",
						contentStr[:min(50, len(contentStr))])
				}
			}
		})
	}
}

func TestBatchOperationsPreserveFrontMatter(t *testing.T) {
	// Test that batch operations also preserve front matter
	initialContent := `---
references:
  - ./batch-spec.md
  - ./batch-design.md
metadata:
  batch_test: true
  version: "2.0"
---
# Batch Test Tasks

- [ ] 1. First task
- [ ] 2. Second task
  - [ ] 2.1. Subtask one
  - [ ] 2.2. Subtask two
- [ ] 3. Third task
`

	// Create temporary file in current directory
	tmpFile := "test_batch_frontmatter.md"
	defer os.Remove(tmpFile)

	// Write initial content
	err := os.WriteFile(tmpFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Parse initial file
	tl, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse initial file: %v", err)
	}

	// Perform multiple operations in sequence (simulating batch operations)
	operations := []func(tl *TaskList) error{
		func(tl *TaskList) error { return tl.UpdateStatus("1", Completed) },
		func(tl *TaskList) error { return tl.UpdateStatus("2.1", InProgress) },
		func(tl *TaskList) error { return tl.AddTask("3", "New subtask") },
		func(tl *TaskList) error { return tl.UpdateTask("2", "", []string{"Updated detail"}, nil) },
	}

	for i, op := range operations {
		err = op(tl)
		if err != nil {
			t.Fatalf("Batch operation %d failed: %v", i+1, err)
		}

		// Write and re-read after each operation to simulate real usage
		err = tl.WriteFile(tmpFile)
		if err != nil {
			t.Fatalf("Failed to write file after operation %d: %v", i+1, err)
		}

		// Re-parse to verify front matter is still there
		tl, err = ParseFile(tmpFile)
		if err != nil {
			t.Fatalf("Failed to re-parse file after operation %d: %v", i+1, err)
		}

		// Verify front matter is preserved
		expectedFrontMatter := &FrontMatter{
			References: []string{"./batch-spec.md", "./batch-design.md"},
			Metadata: map[string]any{
				"batch_test": true,
				"version":    "2.0",
			},
		}
		compareFrontMatter(t, tl.FrontMatter, expectedFrontMatter)
	}

	// Final verification: read the file content and ensure it has front matter
	finalContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read final content: %v", err)
	}

	contentStr := string(finalContent)
	if !strings.HasPrefix(contentStr, "---\n") {
		t.Errorf("Final file should start with front matter delimiter")
	}

	// Verify all expected references are still present
	expectedRefs := []string{"./batch-spec.md", "./batch-design.md"}
	for _, ref := range expectedRefs {
		if !strings.Contains(contentStr, ref) {
			t.Errorf("Final file should contain reference %q", ref)
		}
	}

	// Verify metadata is still present
	if !strings.Contains(contentStr, "batch_test: true") {
		t.Errorf("Final file should contain metadata 'batch_test: true'")
	}
}

func TestComplexFrontMatterPreservation(t *testing.T) {
	// Test with complex front matter including various data types
	initialContent := `---
references:
  - ./docs/complex.md
  - ./specs/advanced.yaml
  - ../shared/common.md
metadata:
  project: "advanced-tasks"
  version: 3.14
  active: true
  tags:
    - important
    - complex
    - test
  author:
    name: "Test User"
    email: "test@example.com"
  created: "2024-01-30T10:00:00Z"
---
# Complex Front Matter Test

- [ ] 1. Main task with complex metadata
  This task has detailed information.
  References: ./task-specific.md
  - [ ] 1.1. Subtask
- [-] 2. In-progress task
`

	// Create temporary file in current directory
	tmpFile := "test_complex_frontmatter.md"
	defer os.Remove(tmpFile)

	// Write initial content
	err := os.WriteFile(tmpFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Parse, modify, and write back
	tl, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse initial file: %v", err)
	}

	// Perform complex operations
	err = tl.UpdateStatus("2", Completed)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	err = tl.AddTask("", "New complex task")
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	err = tl.WriteFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Verify complex front matter is preserved
	finalContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read final content: %v", err)
	}

	// Parse final content
	finalTl, err := ParseMarkdown(finalContent)
	if err != nil {
		t.Fatalf("Failed to parse final content: %v", err)
	}

	// Verify all components of complex front matter
	if finalTl.FrontMatter == nil {
		t.Fatal("Front matter should not be nil")
	}

	// Check references
	expectedRefs := []string{"./docs/complex.md", "./specs/advanced.yaml", "../shared/common.md"}
	if len(finalTl.FrontMatter.References) != len(expectedRefs) {
		t.Errorf("References count mismatch: got %d, want %d",
			len(finalTl.FrontMatter.References), len(expectedRefs))
	}
	for i, ref := range expectedRefs {
		if i >= len(finalTl.FrontMatter.References) || finalTl.FrontMatter.References[i] != ref {
			t.Errorf("Reference[%d] mismatch: got %q, want %q",
				i, getRefSafely(finalTl.FrontMatter.References, i), ref)
		}
	}

	// Check that complex metadata is preserved (verify presence in raw content)
	contentStr := string(finalContent)
	expectedMetadataItems := []string{
		`project: advanced-tasks`, // YAML serializer may remove quotes when not needed
		"version: 3.14",
		"active: true",
		"- important",
		"- complex",
		"- test",
		`name: Test User`,                 // YAML serializer may remove quotes when not needed
		`email: test@example.com`,         // YAML serializer may remove quotes when not needed
		`created: "2024-01-30T10:00:00Z"`, // Timestamp keeps quotes
	}

	for _, item := range expectedMetadataItems {
		if !strings.Contains(contentStr, item) {
			t.Errorf("Final content should contain metadata item %q. Actual content:\n%s", item, contentStr)
		}
	}
}

// Helper function to safely access slice elements
func getRefSafely(refs []string, index int) string {
	if index >= 0 && index < len(refs) {
		return refs[index]
	}
	return "<out of bounds>"
}

// Helper function to compare FrontMatter structs
func compareFrontMatter(t *testing.T, got, want *FrontMatter) {
	t.Helper()

	if want == nil && got == nil {
		return
	}

	if want == nil && got != nil {
		if len(got.References) == 0 && len(got.Metadata) == 0 {
			return // Empty front matter is equivalent to nil
		}
		t.Errorf("Expected nil front matter, got non-empty: %+v", got)
		return
	}

	if want != nil && got == nil {
		t.Errorf("Expected front matter %+v, got nil", want)
		return
	}

	// Compare references
	if len(got.References) != len(want.References) {
		t.Errorf("References count mismatch: got %d, want %d",
			len(got.References), len(want.References))
	} else {
		for i, ref := range want.References {
			if got.References[i] != ref {
				t.Errorf("Reference[%d] mismatch: got %q, want %q", i, got.References[i], ref)
			}
		}
	}

	// Compare metadata
	if len(got.Metadata) != len(want.Metadata) {
		t.Errorf("Metadata count mismatch: got %d, want %d",
			len(got.Metadata), len(want.Metadata))
	} else {
		for key, wantValue := range want.Metadata {
			gotValue, exists := got.Metadata[key]
			if !exists {
				t.Errorf("Metadata key %q not found", key)
				continue
			}
			// For simplicity, compare string representations
			if fmt.Sprintf("%v", gotValue) != fmt.Sprintf("%v", wantValue) {
				t.Errorf("Metadata[%q] mismatch: got %v, want %v", key, gotValue, wantValue)
			}
		}
	}
}

// min is a helper function for Go versions < 1.21
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

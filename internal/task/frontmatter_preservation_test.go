package task

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFrontMatterPreservation(t *testing.T) {
	// Test comprehensive front matter preservation across all operations
	initialContent := `---
references:
  - ./docs/requirements.md
  - ./specs/design.md
metadata:
  project: test-project
  version: "1.0"
  tags: "important,test"
---
# Test Tasks

- [ ] 1. First task
  Details about first task
  References: ./task1.md
- [ ] 2. Second task
  - [ ] 2.1. Subtask one
  - [ ] 2.2. Subtask two
- [-] 3. Third task
`

	// Create temporary file
	tmpFile := "test_frontmatter_preservation.md"
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

	// Define expected front matter that should be preserved
	expectedFrontMatter := &FrontMatter{
		References: []string{"./docs/requirements.md", "./specs/design.md"},
		Metadata: map[string]string{
			"project": "test-project",
			"version": "1.0",
			"tags":    "important,test",
		},
	}

	// Test various operations and verify front matter is preserved
	operations := []struct {
		name string
		op   func(tl *TaskList) error
	}{
		{
			name: "add_root_task",
			op: func(tl *TaskList) error {
				_, err := tl.AddTask("", "New root task", "")
				return err
			},
		},
		{
			name: "add_subtask",
			op: func(tl *TaskList) error {
				_, err := tl.AddTask("1", "New subtask", "")
				return err
			},
		},
		{
			name: "update_status",
			op: func(tl *TaskList) error {
				return tl.UpdateStatus("2.1", Completed)
			},
		},
		{
			name: "update_task_details",
			op: func(tl *TaskList) error {
				return tl.UpdateTask("3", "Updated task", []string{"New detail"}, []string{"ref.md"})
			},
		},
		{
			name: "remove_task",
			op: func(tl *TaskList) error {
				return tl.RemoveTask("2.2")
			},
		},
	}

	for _, test := range operations {
		t.Run(test.name, func(t *testing.T) {
			// Perform operation
			err := test.op(tl)
			if err != nil {
				t.Fatalf("Operation failed: %v", err)
			}

			// Write file back
			err = tl.WriteFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}

			// Re-parse to verify front matter is preserved
			tl, err = ParseFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to re-parse file: %v", err)
			}

			// Verify front matter is preserved
			compareFrontMatter(t, tl.FrontMatter, expectedFrontMatter)
		})
	}

	// Final verification of raw content
	finalContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read final content: %v", err)
	}

	contentStr := string(finalContent)
	if !strings.HasPrefix(contentStr, "---\n") {
		t.Errorf("Final file should start with front matter delimiter")
	}

	// Verify key metadata items are present
	expectedItems := []string{
		"./docs/requirements.md",
		"./specs/design.md",
		"project: test-project",
		`version: "1.0"`,
		"tags: important,test",
	}
	for _, item := range expectedItems {
		if !strings.Contains(contentStr, item) {
			t.Errorf("Final content should contain %q", item)
		}
	}
}

// Helper function to compare FrontMatter structs
func compareFrontMatter(t *testing.T, got, want *FrontMatter) {
	t.Helper()

	if want == nil && got == nil {
		return
	}

	if want == nil || got == nil {
		t.Errorf("Front matter mismatch: got %v, want %v", got, want)
		return
	}

	// Compare references
	if len(got.References) != len(want.References) {
		t.Errorf("References count: got %d, want %d", len(got.References), len(want.References))
	} else {
		for i, ref := range want.References {
			if got.References[i] != ref {
				t.Errorf("Reference[%d]: got %q, want %q", i, got.References[i], ref)
			}
		}
	}

	// Compare metadata count
	if len(got.Metadata) != len(want.Metadata) {
		t.Errorf("Metadata count: got %d, want %d", len(got.Metadata), len(want.Metadata))
	}

	// Compare metadata values
	for key, wantValue := range want.Metadata {
		gotValue, exists := got.Metadata[key]
		if !exists {
			t.Errorf("Metadata key %q not found", key)
			continue
		}
		if fmt.Sprintf("%v", gotValue) != fmt.Sprintf("%v", wantValue) {
			t.Errorf("Metadata[%q]: got %v, want %v", key, gotValue, wantValue)
		}
	}
}

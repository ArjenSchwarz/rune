package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

// TestIntegrationRequirements tests requirements workflow
func TestIntegrationRequirements(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	tests := map[string]struct {
		name        string
		workflow    func(t *testing.T, tempDir string)
		description string
	}{
		"requirements_workflow": {
			name:        "Requirements Workflow",
			description: "Test complete requirements linking workflow",
			workflow:    testRequirementsWorkflow,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "rune-integration-requirements-"+testName)
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

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

func testRequirementsWorkflow(t *testing.T, tempDir string) {
	// Test complete requirements linking workflow
	// Requirement 4.5: Requirements SHALL be preserved during round-trip parsing and rendering

	filename := "requirements-test.md"

	// Step 1: Create task file
	t.Run("create_task_file", func(t *testing.T) {
		tl := task.NewTaskList("Requirements Workflow Test")
		if err := tl.WriteFile(filename); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}
		t.Logf("Created task file: %s", filename)
	})

	// Step 2: Add task with --requirements and --requirements-file flags
	t.Run("add_task_with_requirements", func(t *testing.T) {
		output := runGoCommand(t, "add", filename,
			"--title", "Implement authentication",
			"--requirements", "1.1,1.2,2.3",
			"--requirements-file", "specs/requirements.md")

		if !strings.Contains(output, "Added task") {
			t.Errorf("Expected success message, got: %s", output)
		}
		t.Logf("Added task with requirements")
	})

	// Step 3: Verify requirements rendered as markdown links
	t.Run("verify_markdown_rendering", func(t *testing.T) {
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		markdown := string(content)

		// Check for requirements line
		if !strings.Contains(markdown, "Requirements:") {
			t.Errorf("Markdown should contain 'Requirements:' line")
		}

		// Check for markdown links with correct format
		expectedLinks := []string{
			"[1.1](specs/requirements.md#1.1)",
			"[1.2](specs/requirements.md#1.2)",
			"[2.3](specs/requirements.md#2.3)",
		}
		for _, link := range expectedLinks {
			if !strings.Contains(markdown, link) {
				t.Errorf("Markdown should contain link: %s\nGot:\n%s", link, markdown)
			}
		}

		t.Logf("Verified markdown rendering with links")
	})

	// Step 4: Parse file and verify Requirements field populated
	t.Run("verify_parsed_requirements", func(t *testing.T) {
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		if len(tl.Tasks) == 0 {
			t.Fatal("expected at least one task")
		}

		task := tl.Tasks[0]

		// Verify Requirements field
		if len(task.Requirements) != 3 {
			t.Errorf("expected 3 requirements, got %d", len(task.Requirements))
		}

		expectedReqs := []string{"1.1", "1.2", "2.3"}
		for i, expected := range expectedReqs {
			if i >= len(task.Requirements) {
				t.Errorf("missing requirement at index %d", i)
				continue
			}
			if task.Requirements[i] != expected {
				t.Errorf("requirement[%d] = %q, want %q", i, task.Requirements[i], expected)
			}
		}

		// Verify RequirementsFile field
		if tl.RequirementsFile != "specs/requirements.md" {
			t.Errorf("RequirementsFile = %q, want %q", tl.RequirementsFile, "specs/requirements.md")
		}

		t.Logf("Verified parsed requirements field")
	})

	// Step 5: Update requirements via batch command
	t.Run("update_requirements_via_batch", func(t *testing.T) {
		batchJSON := `{
			"file": "requirements-test.md",
			"requirements_file": "requirements.md",
			"operations": [
				{
					"type": "update",
					"id": "1",
					"requirements": ["3.1", "3.2"]
				}
			]
		}`

		batchFile := "batch.json"
		if err := os.WriteFile(batchFile, []byte(batchJSON), 0o644); err != nil {
			t.Fatalf("failed to write batch file: %v", err)
		}

		output := runGoCommand(t, "batch", batchFile)

		if !strings.Contains(output, "Batch operation successful") && !strings.Contains(output, "operation") {
			t.Errorf("Expected success message, got: %s", output)
		}

		t.Logf("Updated requirements via batch command")
	})

	// Step 6: Verify changes persisted correctly
	t.Run("verify_batch_update_persisted", func(t *testing.T) {
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		if len(tl.Tasks) == 0 {
			t.Fatal("expected at least one task")
		}

		task := tl.Tasks[0]

		// Verify updated requirements
		if len(task.Requirements) != 2 {
			t.Errorf("expected 2 requirements after update, got %d", len(task.Requirements))
		}

		expectedReqs := []string{"3.1", "3.2"}
		for i, expected := range expectedReqs {
			if i >= len(task.Requirements) {
				t.Errorf("missing requirement at index %d", i)
				continue
			}
			if task.Requirements[i] != expected {
				t.Errorf("requirement[%d] = %q, want %q", i, task.Requirements[i], expected)
			}
		}

		// Verify RequirementsFile was updated
		if tl.RequirementsFile != "requirements.md" {
			t.Errorf("RequirementsFile = %q, want %q", tl.RequirementsFile, "requirements.md")
		}

		// Verify markdown reflects changes
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		markdown := string(content)
		expectedLinks := []string{
			"[3.1](requirements.md#3.1)",
			"[3.2](requirements.md#3.2)",
		}
		for _, link := range expectedLinks {
			if !strings.Contains(markdown, link) {
				t.Errorf("Markdown should contain updated link: %s\nGot:\n%s", link, markdown)
			}
		}

		t.Logf("Verified batch update persisted")
	})

	// Step 7: Test round-trip preservation
	t.Run("test_round_trip_preservation", func(t *testing.T) {
		// Parse file
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		originalReqs := tl.Tasks[0].Requirements
		originalFile := tl.RequirementsFile

		// Render to markdown
		rendered := task.RenderMarkdown(tl)

		// Parse again
		tl2, err := task.ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("failed to parse rendered markdown: %v", err)
		}

		// Verify requirements preserved
		if len(tl2.Tasks) == 0 {
			t.Fatal("expected at least one task after round-trip")
		}

		roundTripReqs := tl2.Tasks[0].Requirements
		if len(roundTripReqs) != len(originalReqs) {
			t.Errorf("requirement count mismatch after round-trip: got %d, want %d",
				len(roundTripReqs), len(originalReqs))
		}

		for i := range originalReqs {
			if i >= len(roundTripReqs) {
				t.Errorf("missing requirement at index %d after round-trip", i)
				continue
			}
			if roundTripReqs[i] != originalReqs[i] {
				t.Errorf("requirement[%d] mismatch after round-trip: got %q, want %q",
					i, roundTripReqs[i], originalReqs[i])
			}
		}

		// Verify RequirementsFile preserved (extracted from links during parse)
		if tl2.RequirementsFile != originalFile {
			t.Errorf("RequirementsFile mismatch after round-trip: got %q, want %q",
				tl2.RequirementsFile, originalFile)
		}

		t.Logf("Verified round-trip preservation")
	})

	// Step 8: Test JSON output includes requirements fields
	t.Run("verify_json_output", func(t *testing.T) {
		output := runGoCommand(t, "list", filename, "--format", "json")

		var tl task.TaskList
		if err := json.Unmarshal([]byte(output), &tl); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
		}

		// Verify requirements_file in JSON
		if tl.RequirementsFile != "requirements.md" {
			t.Errorf("JSON RequirementsFile = %q, want %q", tl.RequirementsFile, "requirements.md")
		}

		// Verify requirements array in task
		if len(tl.Tasks) == 0 {
			t.Fatal("expected at least one task in JSON output")
		}

		task := tl.Tasks[0]
		if len(task.Requirements) != 2 {
			t.Errorf("JSON task should have 2 requirements, got %d", len(task.Requirements))
		}

		// Verify raw JSON contains the fields
		if !strings.Contains(output, `"requirements_file"`) {
			t.Error("JSON output should contain 'requirements_file' field")
		}
		if !strings.Contains(output, `"requirements"`) {
			t.Error("JSON output should contain 'requirements' field")
		}

		t.Logf("Verified JSON output includes requirements fields")
	})

	// Step 9: Test adding task with nested requirements
	t.Run("nested_task_with_requirements", func(t *testing.T) {
		output := runGoCommand(t, "add", filename,
			"--parent", "1",
			"--title", "Implement JWT tokens",
			"--requirements", "1.1.1,1.1.2")

		if !strings.Contains(output, "Added task") {
			t.Errorf("Expected success message, got: %s", output)
		}

		// Verify nested task has requirements
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		if len(tl.Tasks) == 0 || len(tl.Tasks[0].Children) == 0 {
			t.Fatal("expected parent task with child")
		}

		child := tl.Tasks[0].Children[0]
		if len(child.Requirements) != 2 {
			t.Errorf("child task should have 2 requirements, got %d", len(child.Requirements))
		}

		expectedReqs := []string{"1.1.1", "1.1.2"}
		for i, expected := range expectedReqs {
			if i >= len(child.Requirements) {
				t.Errorf("missing child requirement at index %d", i)
				continue
			}
			if child.Requirements[i] != expected {
				t.Errorf("child requirement[%d] = %q, want %q", i, child.Requirements[i], expected)
			}
		}

		t.Logf("Verified nested task with requirements")
	})

	// Step 10: Test clearing requirements
	t.Run("clear_requirements", func(t *testing.T) {
		output := runGoCommand(t, "update", filename, "1.1",
			"--clear-requirements")

		if !strings.Contains(output, "Updated task") {
			t.Errorf("Expected success message, got: %s", output)
		}

		// Verify requirements cleared
		tl, err := task.ParseFile(filename)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		if len(tl.Tasks) == 0 || len(tl.Tasks[0].Children) == 0 {
			t.Fatal("expected parent task with child")
		}

		child := tl.Tasks[0].Children[0]
		if len(child.Requirements) != 0 {
			t.Errorf("child task requirements should be cleared, got %d requirements", len(child.Requirements))
		}

		// Verify markdown doesn't have Requirements line for this task
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		markdown := string(content)
		lines := strings.Split(markdown, "\n")

		// Find the subtask line and check it doesn't have requirements
		foundSubtask := false
		for i, line := range lines {
			if strings.Contains(line, "1.1.") && strings.Contains(line, "Implement JWT tokens") {
				foundSubtask = true
				// Check next few lines for Requirements
				for j := i + 1; j < min(i+5, len(lines)); j++ {
					if strings.Contains(lines[j], "Requirements:") {
						// Make sure it's not from another task
						if !strings.Contains(lines[j], "  - [") {
							t.Errorf("Subtask 1.1 should not have Requirements line after clearing")
						}
					}
				}
				break
			}
		}

		if !foundSubtask {
			t.Error("Could not find subtask 1.1 in markdown")
		}

		t.Logf("Verified requirements cleared")
	})

	t.Logf("Requirements workflow test passed successfully")
}

// testRenumberEndToEnd tests the complete renumber workflow:
// validate → parse → backup → renumber → write

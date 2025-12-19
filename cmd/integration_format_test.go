package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestIntegrationFormatSpecificOutputs tests the consistent output format feature
// for all commands with --format json/markdown/table flags.
func TestIntegrationFormatSpecificOutputs(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	tests := map[string]struct {
		name        string
		workflow    func(t *testing.T, tempDir string)
		description string
	}{
		"mutation_commands_json_format": {
			name:        "Mutation Commands JSON Format",
			description: "Test complete, uncomplete, progress, add, remove, update with --format json",
			workflow:    testMutationCommandsJSONFormat,
		},
		"create_command_json_format": {
			name:        "Create Command JSON Format",
			description: "Test create command with --format json",
			workflow:    testCreateCommandJSONFormat,
		},
		"empty_state_json_format": {
			name:        "Empty State JSON Format",
			description: "Test next all-complete, list empty, find no-matches with --format json",
			workflow:    testEmptyStateJSONFormat,
		},
		"verbose_json_stderr": {
			name:        "Verbose JSON Stderr",
			description: "Verify verbose output goes to stderr when --format json",
			workflow:    testVerboseJSONStderr,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Create isolated temp directory for each test
			tempDir, err := os.MkdirTemp("", "rune-format-"+testName)
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

// testMutationCommandsJSONFormat tests that mutation commands return proper JSON
func testMutationCommandsJSONFormat(t *testing.T, _ string) {
	filename := "mutation-test.md"

	// Create a task file with some tasks
	taskContent := `# Mutation Test

- [ ] 1. First task
  - [ ] 1.1. Subtask one
  - [ ] 1.2. Subtask two
- [ ] 2. Second task
- [ ] 3. Third task
`
	if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Test 1: complete command with --format json
	t.Run("complete_json", func(t *testing.T) {
		output := runGoCommand(t, "complete", filename, "1.1", "--format", "json")

		var resp CompleteResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse complete JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in complete response")
		}
		if resp.TaskID != "1.1" {
			t.Errorf("expected task_id: 1.1, got: %s", resp.TaskID)
		}
		if resp.Title != "Subtask one" {
			t.Errorf("expected title: Subtask one, got: %s", resp.Title)
		}
		if !strings.Contains(resp.Message, "Completed") {
			t.Errorf("expected message to contain 'Completed', got: %s", resp.Message)
		}
	})

	// Test 2: progress command with --format json
	t.Run("progress_json", func(t *testing.T) {
		output := runGoCommand(t, "progress", filename, "1.2", "--format", "json")

		var resp ProgressResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse progress JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in progress response")
		}
		if resp.TaskID != "1.2" {
			t.Errorf("expected task_id: 1.2, got: %s", resp.TaskID)
		}
		if resp.Title != "Subtask two" {
			t.Errorf("expected title: Subtask two, got: %s", resp.Title)
		}
	})

	// Test 3: uncomplete command with --format json
	t.Run("uncomplete_json", func(t *testing.T) {
		// First complete the task, then uncomplete it
		runGoCommand(t, "complete", filename, "2")
		output := runGoCommand(t, "uncomplete", filename, "2", "--format", "json")

		var resp UncompleteResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse uncomplete JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in uncomplete response")
		}
		if resp.TaskID != "2" {
			t.Errorf("expected task_id: 2, got: %s", resp.TaskID)
		}
	})

	// Test 4: add command with --format json
	t.Run("add_json", func(t *testing.T) {
		output := runGoCommand(t, "add", filename, "--title", "New task", "--format", "json")

		var resp AddResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse add JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in add response")
		}
		if resp.Title != "New task" {
			t.Errorf("expected title: New task, got: %s", resp.Title)
		}
		if resp.TaskID == "" {
			t.Error("expected non-empty task_id in add response")
		}
	})

	// Test 5: add command with parent --format json
	t.Run("add_with_parent_json", func(t *testing.T) {
		output := runGoCommand(t, "add", filename, "--title", "Subtask", "--parent", "3", "--format", "json")

		var resp AddResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse add JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in add response")
		}
		if resp.Parent != "3" {
			t.Errorf("expected parent: 3, got: %s", resp.Parent)
		}
	})

	// Test 6: update command with --format json
	t.Run("update_json", func(t *testing.T) {
		output := runGoCommand(t, "update", filename, "3", "--title", "Updated third task", "--format", "json")

		var resp UpdateResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse update JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in update response")
		}
		if resp.TaskID != "3" {
			t.Errorf("expected task_id: 3, got: %s", resp.TaskID)
		}
		if len(resp.FieldsUpdated) == 0 {
			t.Error("expected fields_updated to contain at least one field")
		}
	})

	// Test 7: remove command with --format json
	t.Run("remove_json", func(t *testing.T) {
		// Note: we remove task 4 (the "New task" we added earlier)
		output := runGoCommand(t, "remove", filename, "4", "--format", "json")

		var resp RemoveResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse remove JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in remove response")
		}
		if resp.TaskID != "4" {
			t.Errorf("expected task_id: 4, got: %s", resp.TaskID)
		}
	})

	t.Log("Mutation commands JSON format test passed successfully")
}

// testCreateCommandJSONFormat tests the create command with --format json
func testCreateCommandJSONFormat(t *testing.T, _ string) {
	// Test 1: Basic create with JSON format
	t.Run("create_basic_json", func(t *testing.T) {
		filename := "create-basic.md"
		output := runGoCommand(t, "create", filename, "--title", "Test Project", "--format", "json")

		var resp CreateResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse create JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in create response")
		}
		if resp.Path != filename {
			t.Errorf("expected path: %s, got: %s", filename, resp.Path)
		}
		if resp.Title != "Test Project" {
			t.Errorf("expected title: Test Project, got: %s", resp.Title)
		}
	})

	// Test 2: Create with references using JSON format
	t.Run("create_with_references_json", func(t *testing.T) {
		filename := "create-refs.md"
		output := runGoCommand(t, "create", filename, "--title", "Refs Project",
			"--reference", "design.md", "--reference", "spec.md", "--format", "json")

		var resp CreateResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse create JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in create response")
		}
		if resp.References != 2 {
			t.Errorf("expected references: 2, got: %d", resp.References)
		}
	})

	// Test 3: Create with metadata using JSON format
	t.Run("create_with_metadata_json", func(t *testing.T) {
		filename := "create-meta.md"
		output := runGoCommand(t, "create", filename, "--title", "Meta Project",
			"--meta", "author:Test", "--meta", "version:1.0", "--format", "json")

		var resp CreateResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse create JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in create response")
		}
		if resp.Metadata != 2 {
			t.Errorf("expected metadata: 2, got: %d", resp.Metadata)
		}
	})

	t.Log("Create command JSON format test passed successfully")
}

// testEmptyStateJSONFormat tests empty state handling with --format json
func testEmptyStateJSONFormat(t *testing.T, _ string) {
	// Test 1: next command when all tasks are complete
	t.Run("next_all_complete_json", func(t *testing.T) {
		filename := "empty-next.md"
		taskContent := `# Empty Next Test

- [x] 1. Completed task
- [x] 2. Another completed task
`
		if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}

		output := runGoCommand(t, "next", filename, "--format", "json")

		var resp NextEmptyResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse next empty JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in next empty response")
		}
		if resp.Data != nil {
			t.Errorf("expected data: null, got: %v", resp.Data)
		}
		if !strings.Contains(resp.Message, "complete") {
			t.Errorf("expected message to mention completion, got: %s", resp.Message)
		}
	})

	// Test 2: next --phase when all tasks are complete
	t.Run("next_phase_all_complete_json", func(t *testing.T) {
		filename := "empty-phase.md"
		taskContent := `# Empty Phase Test

## Phase: Development

- [x] 1. Completed dev task

## Phase: Testing

- [x] 2. Completed test task
`
		if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}

		output := runGoCommand(t, "next", filename, "--phase", "--format", "json")

		var resp NextPhaseEmptyResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse next phase empty JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in next phase empty response")
		}
		if len(resp.Tasks) != 0 {
			t.Errorf("expected tasks: [], got: %v", resp.Tasks)
		}
	})

	// Test 3: list command with empty file
	t.Run("list_empty_json", func(t *testing.T) {
		filename := "empty-list.md"
		taskContent := `# Empty List Test
`
		if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}

		output := runGoCommand(t, "list", filename, "--format", "json")

		var resp ListEmptyResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse list empty JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in list empty response")
		}
		if resp.Count != 0 {
			t.Errorf("expected count: 0, got: %d", resp.Count)
		}
		if len(resp.Data) != 0 {
			t.Errorf("expected data: [], got: %v", resp.Data)
		}
	})

	// Test 4: find command with no matches
	t.Run("find_no_matches_json", func(t *testing.T) {
		filename := "find-nomatch.md"
		taskContent := `# Find No Match Test

- [ ] 1. First task
- [ ] 2. Second task
`
		if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}

		output := runGoCommand(t, "find", filename, "-p", "nonexistent", "--format", "json")

		var resp FindEmptyResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("failed to parse find empty JSON response: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true in find empty response")
		}
		if resp.Count != 0 {
			t.Errorf("expected count: 0, got: %d", resp.Count)
		}
		if len(resp.Data) != 0 {
			t.Errorf("expected data: [], got: %v", resp.Data)
		}
	})

	// Test 5: Verify empty state with markdown format outputs blockquote
	t.Run("next_all_complete_markdown", func(t *testing.T) {
		filename := "empty-next-md.md"
		taskContent := `# Empty Next Markdown Test

- [x] 1. Completed task
`
		if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}

		output := runGoCommand(t, "next", filename, "--format", "markdown")

		if !strings.HasPrefix(output, "> ") {
			t.Errorf("expected markdown blockquote format (> ), got: %s", output)
		}
	})

	t.Log("Empty state JSON format test passed successfully")
}

// testVerboseJSONStderr tests that verbose output goes to stderr when --format json is used
func testVerboseJSONStderr(t *testing.T, _ string) {
	filename := "verbose-test.md"
	taskContent := `# Verbose Test

- [ ] 1. Test task
`
	if err := os.WriteFile(filename, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	// Test that stdout is valid JSON when using --verbose --format json
	t.Run("complete_verbose_json_stdout_valid", func(t *testing.T) {
		output := runGoCommandStdoutOnly(t, "complete", filename, "1", "--verbose", "--format", "json")

		// stdout should be valid JSON
		var resp CompleteResponse
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			t.Fatalf("stdout should be valid JSON when using --verbose --format json: %v\nOutput: %s", err, output)
		}

		if !resp.Success {
			t.Error("expected success: true")
		}
	})

	// Test verbose output goes to stderr
	t.Run("list_verbose_json_stderr_has_verbose", func(t *testing.T) {
		// Create a new file for this test
		testFile := "verbose-stderr.md"
		content := `# Verbose Stderr Test

- [ ] 1. Another task
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}

		stdout, stderr := runGoCommandSeparateOutputs(t, "list", testFile, "--verbose", "--format", "json")

		// Stderr should have verbose output
		if !strings.Contains(stderr, "Using task file") {
			t.Errorf("expected stderr to contain verbose output about task file, got: %s", stderr)
		}

		// Stdout should still be valid JSON (either task list or empty response)
		if len(stdout) > 0 && !strings.HasPrefix(strings.TrimSpace(stdout), "{") {
			t.Errorf("stdout should start with JSON object, got: %s", stdout)
		}
	})

	t.Log("Verbose JSON stderr test passed successfully")
}

// runGoCommandStdoutOnly runs a command and returns only stdout
func runGoCommandStdoutOnly(t *testing.T, args ...string) string {
	if runeBinaryPath == "" {
		t.Fatal("rune binary path not set - TestMain should have built the binary")
	}
	cmd := exec.Command(runeBinaryPath, args...)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v, stdout: %s", err, stdout)
	}
	return string(stdout)
}

// runGoCommandSeparateOutputs runs a command and returns stdout and stderr separately
func runGoCommandSeparateOutputs(t *testing.T, args ...string) (string, string) {
	if runeBinaryPath == "" {
		t.Fatal("rune binary path not set - TestMain should have built the binary")
	}
	cmd := exec.Command(runeBinaryPath, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// Don't fail on error, just return outputs
		t.Logf("command returned error: %v", err)
	}
	return stdout.String(), stderr.String()
}

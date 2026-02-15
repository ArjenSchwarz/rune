package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

// resetBatchFlags resets the batch command's flag state between tests.
// Cobra flag values and Changed bits persist across Execute() calls in the same process.
func resetBatchFlags() {
	batchInput = ""
	if f := batchCmd.Flags().Lookup("input"); f != nil {
		f.Changed = false
	}
}

func TestBatchCommand_BasicOperations(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
  - Details about first task
  - References: doc1.md
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create batch request
	req := task.BatchRequest{
		File: "test_tasks.md",
		Operations: []task.Operation{
			{
				Type:  "add",
				Title: "Second task",
			},
			{
				Type:   "update",
				ID:     "1",
				Status: task.StatusPtr(task.InProgress),
			},
		},
		DryRun: false,
	}

	// Execute batch command
	jsonData, _ := json.Marshal(req)

	// Capture output
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"batch", "--input", string(jsonData), "--format", "table"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Batch command failed: %v", err)
	}

	// Verify file was updated
	updatedContent, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)
	if !strings.Contains(updatedStr, "Second task") {
		t.Error("Expected 'Second task' to be added")
	}
	if !strings.Contains(updatedStr, "[-] 1. First task") {
		t.Error("Expected first task to be in progress")
	}
}

func TestBatchCommand_DryRun(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create batch request with dry run
	req := task.BatchRequest{
		File: "test_tasks.md",
		Operations: []task.Operation{
			{
				Type:  "add",
				Title: "New task",
			},
		},
		DryRun: true,
	}

	// Execute batch command
	jsonData, _ := json.Marshal(req)

	// Capture output
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"batch", "--input", string(jsonData), "--format", "table"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Batch command failed: %v", err)
	}

	// Verify original file was not changed
	currentContent, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(currentContent) != initialContent {
		t.Error("Original file should not be modified during dry run")
	}

	// Verify output mentions dry run
	outputStr := output.String()
	if !strings.Contains(outputStr, "Dry run successful") || !strings.Contains(outputStr, "operations validated") {
		t.Errorf("Expected dry run success message, got: %q", outputStr)
	}
}

func TestBatchCommand_ValidationFailures(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	tests := map[string]struct {
		req     task.BatchRequest
		wantErr bool
	}{
		"missing file": {
			req: task.BatchRequest{
				Operations: []task.Operation{{Type: "add", Title: "Test"}},
			},
			wantErr: true,
		},
		"no operations": {
			req: task.BatchRequest{
				File:       "test_tasks.md",
				Operations: []task.Operation{},
			},
			wantErr: true,
		},
		"invalid operation": {
			req: task.BatchRequest{
				File: "test_tasks.md",
				Operations: []task.Operation{
					{Type: "remove", ID: "999"}, // Non-existent task
				},
			},
			wantErr: false, // Should succeed but with validation errors
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tc.req)

			// Reset command state
			rootCmd.SetArgs([]string{"batch", "--input", string(jsonData)})

			err := rootCmd.Execute()

			if tc.wantErr && err == nil {
				t.Error("Expected command to fail")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected command to succeed, got: %v", err)
			}
		})
	}
}

func TestBatchCommand_JSONOutput(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create batch request
	req := task.BatchRequest{
		File: "test_tasks.md",
		Operations: []task.Operation{
			{
				Type:  "add",
				Title: "New task",
			},
		},
		DryRun: true,
	}

	// Execute batch command with JSON format
	jsonData, _ := json.Marshal(req)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"batch", "--input", string(jsonData), "--format", "json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Batch command failed: %v", err)
	}

	// Combine output from both sources
	outputStr := strings.TrimSpace(stdout.String() + stderr.String())
	t.Logf("Debug: captured output: %q", outputStr)

	// Verify output is valid JSON
	var response task.BatchResponse
	if err := json.Unmarshal([]byte(outputStr), &response); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nStdout: %q\nStderr: %q", err, stdout.String(), stderr.String())
	}

	// Verify response content
	if !response.Success {
		t.Error("Expected successful response")
	}
	if response.Applied != 1 {
		t.Errorf("Expected 1 applied operation, got %d", response.Applied)
	}
	if response.Preview == "" {
		t.Error("Expected preview in dry run response")
	}
}

func TestBatchCommand_StdinInput(t *testing.T) {
	resetBatchFlags()

	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create batch request
	req := task.BatchRequest{
		File: "test_tasks.md",
		Operations: []task.Operation{
			{
				Type:  "add",
				Title: "Stdin task",
			},
		},
		DryRun: true,
	}

	jsonData, _ := json.Marshal(req)

	// Set up stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write to stdin
	go func() {
		defer w.Close()
		w.Write(jsonData)
	}()

	// Capture output
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"batch", "--format", "table"})

	err := rootCmd.Execute()

	// Restore stdin
	os.Stdin = oldStdin

	if err != nil {
		t.Fatalf("Batch command failed: %v", err)
	}

	// Verify output
	outputStr := output.String()
	if !strings.Contains(outputStr, "Dry run successful") || !strings.Contains(outputStr, "operations validated") {
		t.Errorf("Expected dry run success message, got: %q", outputStr)
	}
}

func TestBatchCommand_FileInput(t *testing.T) {
	resetBatchFlags()

	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")
	batchFile := filepath.Join(tmpDir, "operations.json")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create batch operations file
	req := task.BatchRequest{
		File: "test_tasks.md",
		Operations: []task.Operation{
			{
				Type:  "add",
				Title: "File input task",
			},
		},
		DryRun: true,
	}

	jsonData, _ := json.Marshal(req)
	if err := os.WriteFile(batchFile, jsonData, 0644); err != nil {
		t.Fatalf("Failed to create batch file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Execute batch command
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"batch", "operations.json", "--format", "table"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Batch command failed: %v", err)
	}

	// Verify output
	outputStr := output.String()
	if !strings.Contains(outputStr, "Dry run successful") || !strings.Contains(outputStr, "operations validated") {
		t.Errorf("Expected dry run success message, got: %q", outputStr)
	}
}

func TestBatchCommand_StdinViaDash(t *testing.T) {
	// Regression test: --input - should read JSON from stdin, not treat "-" as literal JSON.
	resetBatchFlags()

	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	req := task.BatchRequest{
		Operations: []task.Operation{
			{
				Type:  "add",
				Title: "Stdin dash task",
			},
		},
		DryRun: true,
	}

	jsonData, _ := json.Marshal(req)

	// Set up stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer w.Close()
		w.Write(jsonData)
	}()

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"batch", "test_tasks.md", "--input", "-", "--format", "table"})

	err := rootCmd.Execute()

	os.Stdin = oldStdin

	if err != nil {
		t.Fatalf("Batch command failed: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Dry run successful") || !strings.Contains(outputStr, "operations validated") {
		t.Errorf("Expected dry run success message, got: %q", outputStr)
	}
}

func TestBatchCommand_PositionalFileArg(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	tests := map[string]struct {
		args    []string
		input   string
		wantErr string
	}{
		"positional file fills missing file field": {
			args:  []string{"batch", "test_tasks.md", "--input", `{"operations":[{"type":"add","title":"New task"}]}`, "--dry-run"},
			input: "",
		},
		"positional file matches json file field": {
			args:  []string{"batch", "test_tasks.md", "--input", `{"file":"test_tasks.md","operations":[{"type":"add","title":"New task"}]}`, "--dry-run"},
			input: "",
		},
		"positional file conflicts with json file field": {
			args:    []string{"batch", "other.md", "--input", `{"file":"test_tasks.md","operations":[{"type":"add","title":"New task"}]}`, "--dry-run"},
			wantErr: "conflicting file",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			rootCmd.SetOut(&output)
			rootCmd.SetArgs(tc.args)

			err := rootCmd.Execute()

			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Expected error containing %q, got: %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Command failed: %v", err)
			}
		})
	}
}

func TestBatchCommand_MaxOperationsLimit(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "test_tasks.md")

	// Create initial task file
	initialContent := `# Test Tasks

- [ ] 1. First task
`
	if err := os.WriteFile(taskFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory for path validation
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create batch request with too many operations
	ops := make([]task.Operation, 101) // Over the limit of 100
	for i := range ops {
		ops[i] = task.Operation{
			Type:  "add",
			Title: fmt.Sprintf("Task %d", i),
		}
	}

	req := task.BatchRequest{
		File:       "test_tasks.md",
		Operations: ops,
	}

	// Execute batch command
	jsonData, _ := json.Marshal(req)

	// Capture output
	rootCmd.SetArgs([]string{"batch", "--input", string(jsonData)})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected command to fail due to operation limit")
	}
	if !strings.Contains(err.Error(), "maximum of 100 operations") {
		t.Errorf("Expected operation limit error, got: %v", err)
	}
}

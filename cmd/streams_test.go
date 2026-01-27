package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

func TestStreamsCommand(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-streams-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	tests := map[string]struct {
		fileContent       string
		fileName          string
		availableFlag     bool
		jsonFlag          bool
		wantErr           bool
		expectInOutput    []string
		expectNotInOutput []string
	}{
		"multiple streams with ready and blocked tasks": {
			fileContent: `# Multi-Stream Project

- [ ] 1. Task in stream 1 <!-- id:task001 -->
  - Stream: 1

- [ ] 2. Task in stream 2 <!-- id:task002 -->
  - Stream: 2

- [ ] 3. Blocked task in stream 1 <!-- id:task003 -->
  - Stream: 1
  - Blocked-by: task001 (Task in stream 1)

- [-] 4. Active task in stream 2 <!-- id:task004 -->
  - Stream: 2
`,
			fileName:       "multi-stream.md",
			availableFlag:  false,
			jsonFlag:       false,
			expectInOutput: []string{"1", "2", "READY", "BLOCKED", "ACTIVE"},
		},
		"available flag filters to streams with ready tasks": {
			fileContent: `# Available Streams Test

- [ ] 1. Ready task in stream 1 <!-- id:ready01 -->
  - Stream: 1

- [ ] 2. Blocked task in stream 2 <!-- id:block01 -->
  - Stream: 2
  - Blocked-by: ready01 (Ready task in stream 1)
`,
			fileName:          "available-streams.md",
			availableFlag:     true,
			jsonFlag:          false,
			expectInOutput:    []string{"1"},
			expectNotInOutput: []string{"2"},
		},
		"json output format": {
			fileContent: `# JSON Output Test

- [ ] 1. Task A <!-- id:jsonid1 -->
  - Stream: 1

- [ ] 2. Task B <!-- id:jsonid2 -->
  - Stream: 2
`,
			fileName:       "json-output.md",
			availableFlag:  false,
			jsonFlag:       true,
			expectInOutput: []string{`"streams"`, `"available"`, `"id"`, `"ready"`, `"blocked"`, `"active"`},
		},
		"empty streams result when all tasks completed table format": {
			fileContent: `# All Complete

- [x] 1. Completed task <!-- id:done001 -->
  - Stream: 1
`,
			fileName:       "all-complete.md",
			availableFlag:  false,
			jsonFlag:       false,
			expectInOutput: []string{"No streams with pending tasks"},
		},
		"json output with available flag": {
			fileContent: `# JSON Available Test

- [ ] 1. Ready in stream 1 <!-- id:av0001 -->
  - Stream: 1

- [-] 2. Active in stream 2 <!-- id:av0002 -->
  - Stream: 2
`,
			fileName:       "json-available.md",
			availableFlag:  true,
			jsonFlag:       true,
			expectInOutput: []string{`"id": 1`, `"available": [`, `1`},
		},
		"default stream when not explicitly set": {
			fileContent: `# Default Stream Test

- [ ] 1. Task without explicit stream <!-- id:def001 -->

- [ ] 2. Task with stream 2 <!-- id:def002 -->
  - Stream: 2
`,
			fileName:       "default-stream.md",
			availableFlag:  false,
			jsonFlag:       false,
			expectInOutput: []string{"1", "2"},
		},
		"nonexistent file": {
			fileName: "nonexistent.md",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags to defaults before each test
			streamsAvailable = false
			streamsJSON = false

			// Write test file if content provided
			if tc.fileContent != "" {
				if err := os.WriteFile(tc.fileName, []byte(tc.fileContent), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Build command args
			args := []string{"streams", tc.fileName}
			if tc.availableFlag {
				args = append(args, "--available")
			}
			if tc.jsonFlag {
				args = append(args, "--json")
			}

			rootCmd.SetArgs(args)
			err := rootCmd.Execute()

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Reset command args for next test
			rootCmd.SetArgs([]string{})

			// Check error expectation
			if tc.wantErr && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Skip output checks if we expected an error
			if tc.wantErr {
				return
			}

			// Check expected strings in output
			for _, expected := range tc.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("expected '%s' in output, got: %s", expected, output)
				}
			}

			// Check strings that should not be in output
			for _, notExpected := range tc.expectNotInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("did not expect '%s' in output, got: %s", notExpected, output)
				}
			}

			// For JSON, validate it's valid JSON
			if tc.jsonFlag && !tc.wantErr {
				var jsonObj any
				if err := json.Unmarshal([]byte(output), &jsonObj); err != nil {
					t.Errorf("JSON format produced invalid JSON: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

func TestStreamsCommandJSONStructure(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-streams-json-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	fileContent := `# JSON Structure Test

- [ ] 1. Ready task stream 1 <!-- id:str0001 -->
  - Stream: 1

- [ ] 2. Blocked task stream 1 <!-- id:str0002 -->
  - Stream: 1
  - Blocked-by: str0001 (Ready task stream 1)

- [-] 3. Active task stream 2 <!-- id:str0003 -->
  - Stream: 2

- [ ] 4. Ready task stream 2 <!-- id:str0004 -->
  - Stream: 2
`

	fileName := "json-structure.md"
	if err := os.WriteFile(fileName, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"streams", fileName, "--json"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse JSON and verify structure
	var result task.StreamsResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, output)
	}

	// Verify we have 2 streams
	if len(result.Streams) != 2 {
		t.Errorf("expected 2 streams, got %d", len(result.Streams))
	}

	// Find stream 1 and verify its contents
	var stream1, stream2 *task.StreamStatus
	for i := range result.Streams {
		switch result.Streams[i].ID {
		case 1:
			stream1 = &result.Streams[i]
		case 2:
			stream2 = &result.Streams[i]
		}
	}

	if stream1 == nil {
		t.Fatal("stream 1 not found")
	}
	if stream2 == nil {
		t.Fatal("stream 2 not found")
	}

	// Stream 1: 1 ready (task 1), 1 blocked (task 2), 0 active
	if len(stream1.Ready) != 1 || stream1.Ready[0] != "1" {
		t.Errorf("stream 1 ready: expected [1], got %v", stream1.Ready)
	}
	if len(stream1.Blocked) != 1 || stream1.Blocked[0] != "2" {
		t.Errorf("stream 1 blocked: expected [2], got %v", stream1.Blocked)
	}
	if len(stream1.Active) != 0 {
		t.Errorf("stream 1 active: expected [], got %v", stream1.Active)
	}

	// Stream 2: 1 ready (task 4), 0 blocked, 1 active (task 3)
	if len(stream2.Ready) != 1 || stream2.Ready[0] != "4" {
		t.Errorf("stream 2 ready: expected [4], got %v", stream2.Ready)
	}
	if len(stream2.Blocked) != 0 {
		t.Errorf("stream 2 blocked: expected [], got %v", stream2.Blocked)
	}
	if len(stream2.Active) != 1 || stream2.Active[0] != "3" {
		t.Errorf("stream 2 active: expected [3], got %v", stream2.Active)
	}

	// Verify available streams (both have ready tasks)
	if len(result.Available) != 2 {
		t.Errorf("expected 2 available streams, got %d: %v", len(result.Available), result.Available)
	}
}

func TestStreamsCommandWithAvailableFilter(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-streams-available-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Stream 1 has ready tasks, stream 2 has only blocked tasks
	fileContent := `# Available Filter Test

- [ ] 1. Ready task <!-- id:avail01 -->
  - Stream: 1

- [ ] 2. Another ready in stream 1 <!-- id:avail02 -->
  - Stream: 1

- [ ] 3. Blocked task in stream 2 <!-- id:avail03 -->
  - Stream: 2
  - Blocked-by: avail01 (Ready task)

- [ ] 4. Another blocked in stream 2 <!-- id:avail04 -->
  - Stream: 2
  - Blocked-by: avail02 (Another ready in stream 1)
`

	fileName := "available-filter.md"
	if err := os.WriteFile(fileName, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"streams", fileName, "--available", "--json"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result task.StreamsResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// With --available, only stream 1 should be shown
	if len(result.Streams) != 1 {
		t.Errorf("expected 1 stream with --available, got %d", len(result.Streams))
	}

	if result.Streams[0].ID != 1 {
		t.Errorf("expected stream 1, got stream %d", result.Streams[0].ID)
	}
}

func TestStreamsCommandEmptyResult(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rune-streams-empty-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	fileContent := `# Empty Streams Test

- [x] 1. All done <!-- id:empty01 -->
  - Stream: 1

- [x] 2. Also done <!-- id:empty02 -->
  - Stream: 2
`

	fileName := "empty-streams.md"
	if err := os.WriteFile(fileName, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"streams", fileName, "--json"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	rootCmd.SetArgs([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result task.StreamsResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// All tasks completed, so no streams should be returned
	if len(result.Streams) != 0 {
		t.Errorf("expected 0 streams (all completed), got %d", len(result.Streams))
	}

	if len(result.Available) != 0 {
		t.Errorf("expected 0 available streams, got %d", len(result.Available))
	}
}

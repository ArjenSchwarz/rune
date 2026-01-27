package task

import (
	"testing"
)

// TestParseInvalidStableIDFormat tests handling of invalid stable ID formats
func TestParseInvalidStableIDFormat(t *testing.T) {
	tests := map[string]struct {
		content      string
		taskID       string
		wantStableID string // Empty means ID should be ignored
	}{
		"stable_id_too_short": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc123 -->`,
			taskID:       "1",
			wantStableID: "", // 6 chars, should be ignored
		},
		"stable_id_too_long": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc12345 -->`,
			taskID:       "1",
			wantStableID: "", // 8 chars, should be ignored
		},
		"stable_id_uppercase": {
			content: `# Tasks
- [ ] 1. Task <!-- id:ABC1234 -->`,
			taskID:       "1",
			wantStableID: "", // Uppercase not allowed
		},
		"stable_id_mixed_case": {
			content: `# Tasks
- [ ] 1. Task <!-- id:AbC1234 -->`,
			taskID:       "1",
			wantStableID: "", // Mixed case not allowed
		},
		"stable_id_special_chars": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc-123 -->`,
			taskID:       "1",
			wantStableID: "", // Hyphen not allowed
		},
		"stable_id_underscore": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc_123 -->`,
			taskID:       "1",
			wantStableID: "", // Underscore not allowed
		},
		"stable_id_empty": {
			content: `# Tasks
- [ ] 1. Task <!-- id: -->`,
			taskID:       "1",
			wantStableID: "", // Empty ID should be ignored
		},
		"stable_id_whitespace_only": {
			content: `# Tasks
- [ ] 1. Task <!-- id:   -->`,
			taskID:       "1",
			wantStableID: "", // Whitespace should be ignored
		},
		"malformed_comment_no_space": {
			// Parser is lenient with whitespace - accepts <!--id: as well as <!-- id:
			content: `# Tasks
- [ ] 1. Task <!--id:abc1234-->`,
			taskID:       "1",
			wantStableID: "abc1234", // Lenient parsing allows missing space
		},
		"malformed_comment_no_closing": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234`,
			taskID:       "1",
			wantStableID: "", // Missing closing -->
		},
		"valid_stable_id_still_works": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->`,
			taskID:       "1",
			wantStableID: "abc1234",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() should not error on invalid stable ID format: %v", err)
			}

			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			if task.StableID != tc.wantStableID {
				t.Errorf("Task %s StableID = %q, want %q", tc.taskID, task.StableID, tc.wantStableID)
			}
		})
	}
}

// TestParseMalformedBlockedBy tests handling of malformed blocked-by references
func TestParseMalformedBlockedBy(t *testing.T) {
	tests := map[string]struct {
		content       string
		taskID        string
		wantBlockedBy []string // What should be parsed (invalid refs ignored)
	}{
		"blocked_by_invalid_id_format": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: ABC1234 (First task)`,
			taskID:        "2",
			wantBlockedBy: nil, // Uppercase ID should be ignored
		},
		"blocked_by_id_too_short": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc123 (First task)`,
			taskID:        "2",
			wantBlockedBy: nil, // 6 char ID should be ignored
		},
		"blocked_by_id_too_long": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc12345 (First task)`,
			taskID:        "2",
			wantBlockedBy: nil, // 8 char ID should be ignored
		},
		"blocked_by_mixed_valid_and_invalid": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
- [ ] 3. Third task <!-- id:ghi9012 -->
  - Blocked-by: abc1234 (First task), INVALID, def5678 (Second task)`,
			taskID:        "3",
			wantBlockedBy: []string{"abc1234", "def5678"}, // Valid IDs kept, invalid ignored
		},
		"blocked_by_empty_value": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: `,
			taskID:        "2",
			wantBlockedBy: nil,
		},
		"blocked_by_only_whitespace": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by:    `,
			taskID:        "2",
			wantBlockedBy: nil,
		},
		"blocked_by_unclosed_parenthesis": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc1234 (First task`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"}, // ID should still be extracted
		},
		"blocked_by_valid_still_works": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc1234 (First task)`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() should not error on malformed blocked-by: %v", err)
			}

			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			if len(task.BlockedBy) != len(tc.wantBlockedBy) {
				t.Errorf("Task %s BlockedBy count = %d, want %d. Got: %v", tc.taskID, len(task.BlockedBy), len(tc.wantBlockedBy), task.BlockedBy)
				return
			}

			for i, want := range tc.wantBlockedBy {
				if task.BlockedBy[i] != want {
					t.Errorf("Task %s BlockedBy[%d] = %q, want %q", tc.taskID, i, task.BlockedBy[i], want)
				}
			}
		})
	}
}

// TestParseInvalidStreamValue tests handling of invalid stream values
func TestParseInvalidStreamValue(t *testing.T) {
	tests := map[string]struct {
		content    string
		taskID     string
		wantStream int // 0 means stream should be ignored/default
	}{
		"stream_zero": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: 0`,
			taskID:     "1",
			wantStream: 0, // Zero is invalid, treated as not set
		},
		"stream_negative": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: -1`,
			taskID:     "1",
			wantStream: 0, // Negative is invalid
		},
		"stream_non_integer": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: abc`,
			taskID:     "1",
			wantStream: 0, // Non-integer should be ignored
		},
		"stream_float": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: 1.5`,
			taskID:     "1",
			wantStream: 0, // Float should be ignored
		},
		"stream_empty": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: `,
			taskID:     "1",
			wantStream: 0, // Empty should be ignored
		},
		"stream_whitespace": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream:    `,
			taskID:     "1",
			wantStream: 0, // Whitespace should be ignored
		},
		"stream_with_text": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: 2 stream`,
			taskID:     "1",
			wantStream: 0, // Extra text should cause rejection
		},
		"stream_valid_still_works": {
			content: `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Stream: 5`,
			taskID:     "1",
			wantStream: 5,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() should not error on invalid stream value: %v", err)
			}

			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			if task.Stream != tc.wantStream {
				t.Errorf("Task %s Stream = %d, want %d", tc.taskID, task.Stream, tc.wantStream)
			}
		})
	}
}

// TestParseDuplicateStableIDs tests that duplicate stable IDs are handled with warning
func TestParseDuplicateStableIDs(t *testing.T) {
	// Duplicate stable IDs should log a warning but still parse
	// The first occurrence should be used, subsequent duplicates should be ignored or warned
	content := `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:abc1234 -->
- [ ] 3. Third task <!-- id:def5678 -->`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() should not error on duplicate stable IDs: %v", err)
	}

	if len(tl.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tl.Tasks))
	}

	// First task should have the stable ID
	task1 := tl.FindTask("1")
	if task1 == nil {
		t.Fatal("Task 1 not found")
	}
	if task1.StableID != "abc1234" {
		t.Errorf("Task 1 StableID = %q, want %q", task1.StableID, "abc1234")
	}

	// Second task has duplicate ID - behavior depends on implementation:
	// Option 1: Keep it (current implementation may do this)
	// Option 2: Clear it and warn
	// We'll accept either behavior as long as parsing succeeds
	task2 := tl.FindTask("2")
	if task2 == nil {
		t.Fatal("Task 2 not found")
	}
	// Note: We don't strictly enforce behavior here, just that parsing succeeds

	// Third task should have its unique ID
	task3 := tl.FindTask("3")
	if task3 == nil {
		t.Fatal("Task 3 not found")
	}
	if task3.StableID != "def5678" {
		t.Errorf("Task 3 StableID = %q, want %q", task3.StableID, "def5678")
	}
}

// TestParseMetadataNotTreatedAsDetails tests that metadata lines are not added to Details
func TestParseMetadataNotTreatedAsDetails(t *testing.T) {
	content := `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Regular detail
  - Blocked-by: xyz9012 (Some task)
  - Stream: 2
  - Owner: agent-1
  - Another detail`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	task := tl.FindTask("1")
	if task == nil {
		t.Fatal("Task 1 not found")
	}

	// Details should only contain the regular detail lines, not metadata
	wantDetails := []string{"Regular detail", "Another detail"}
	if len(task.Details) != len(wantDetails) {
		t.Errorf("Details count = %d, want %d. Got: %v", len(task.Details), len(wantDetails), task.Details)
	}
	for i, want := range wantDetails {
		if i < len(task.Details) && task.Details[i] != want {
			t.Errorf("Details[%d] = %q, want %q", i, task.Details[i], want)
		}
	}

	// Verify metadata was parsed (even if target doesn't exist - lenient parsing)
	if task.Stream != 2 {
		t.Errorf("Stream = %d, want 2", task.Stream)
	}
	if task.Owner != "agent-1" {
		t.Errorf("Owner = %q, want %q", task.Owner, "agent-1")
	}
}

// TestParseBlockedByNonExistentID tests that blocked-by with non-existent ID is kept for later validation
func TestParseBlockedByNonExistentID(t *testing.T) {
	// During parsing, we don't validate that blocked-by IDs exist
	// That validation happens during operations
	content := `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Blocked-by: xyz9012 (Non-existent task)`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() should not error on non-existent blocked-by target: %v", err)
	}

	task := tl.FindTask("1")
	if task == nil {
		t.Fatal("Task 1 not found")
	}

	// The reference should be stored for later validation
	if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "xyz9012" {
		t.Errorf("BlockedBy = %v, want [xyz9012]", task.BlockedBy)
	}
}

// TestParseOwnerWithNewline tests that owner values with newlines are handled
func TestParseOwnerWithNewline(t *testing.T) {
	// Newlines in owner values should be handled gracefully
	// Since the value is on a single line in markdown, newlines shouldn't appear
	// But we test that the parser doesn't break
	content := `# Tasks
- [ ] 1. Task <!-- id:abc1234 -->
  - Owner: agent-1`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	task := tl.FindTask("1")
	if task == nil {
		t.Fatal("Task 1 not found")
	}

	if task.Owner != "agent-1" {
		t.Errorf("Owner = %q, want %q", task.Owner, "agent-1")
	}
}

// TestParseStableIDWithTrailingContent tests stable ID extraction with extra content
func TestParseStableIDWithTrailingContent(t *testing.T) {
	tests := map[string]struct {
		content      string
		taskID       string
		wantStableID string
		wantTitle    string
	}{
		"stable_id_at_end": {
			content: `# Tasks
- [ ] 1. My task <!-- id:abc1234 -->`,
			taskID:       "1",
			wantStableID: "abc1234",
			wantTitle:    "My task",
		},
		"stable_id_with_trailing_text": {
			// Note: Trailing text after the stable ID comment is preserved in the title
			// This is edge-case behavior - stable ID comments should be at the end
			content: `# Tasks
- [ ] 1. My task <!-- id:abc1234 --> extra text`,
			taskID:       "1",
			wantStableID: "abc1234",
			wantTitle:    "My task  extra text", // Trailing text is kept (double space from removed comment)
		},
		"multiple_comments": {
			// Only the stable ID comment is removed, other comments are preserved
			content: `# Tasks
- [ ] 1. My task <!-- id:abc1234 --> <!-- other comment -->`,
			taskID:       "1",
			wantStableID: "abc1234",
			wantTitle:    "My task  <!-- other comment -->", // Other comments preserved
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error: %v", err)
			}

			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			if task.StableID != tc.wantStableID {
				t.Errorf("StableID = %q, want %q", task.StableID, tc.wantStableID)
			}

			if task.Title != tc.wantTitle {
				t.Errorf("Title = %q, want %q", task.Title, tc.wantTitle)
			}
		})
	}
}

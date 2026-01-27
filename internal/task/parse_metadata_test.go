package task

import (
	"strings"
	"testing"
)

// TestParseStableID tests extraction of stable IDs from HTML comments
func TestParseStableID(t *testing.T) {
	tests := map[string]struct {
		content      string
		taskID       string
		wantStableID string
	}{
		"task_with_stable_id": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->`,
			taskID:       "1",
			wantStableID: "abc1234",
		},
		"task_with_stable_id_spaces": {
			content: `# Tasks
- [ ] 1. First task <!--  id:abc1234  -->`,
			taskID:       "1",
			wantStableID: "abc1234",
		},
		"task_without_stable_id": {
			content: `# Tasks
- [ ] 1. First task`,
			taskID:       "1",
			wantStableID: "",
		},
		"subtask_with_stable_id": {
			content: `# Tasks
- [ ] 1. Parent task <!-- id:abc1234 -->
  - [ ] 1.1. Child task <!-- id:def5678 -->`,
			taskID:       "1.1",
			wantStableID: "def5678",
		},
		"multiple_tasks_with_ids": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:xyz9012 -->`,
			taskID:       "2",
			wantStableID: "xyz9012",
		},
		"stable_id_with_numbers_only": {
			content: `# Tasks
- [ ] 1. Task with numeric ID <!-- id:1234567 -->`,
			taskID:       "1",
			wantStableID: "1234567",
		},
		"stable_id_with_letters_only": {
			content: `# Tasks
- [ ] 1. Task with alpha ID <!-- id:abcdefg -->`,
			taskID:       "1",
			wantStableID: "abcdefg",
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
				t.Errorf("Task %s StableID = %q, want %q", tc.taskID, task.StableID, tc.wantStableID)
			}
		})
	}
}

// TestParseBlockedBy tests parsing of Blocked-by metadata with title hints
func TestParseBlockedBy(t *testing.T) {
	tests := map[string]struct {
		content       string
		taskID        string
		wantBlockedBy []string
	}{
		"single_blocked_by": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc1234 (First task)`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
		},
		"multiple_blocked_by": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
- [ ] 3. Third task <!-- id:ghi9012 -->
  - Blocked-by: abc1234 (First task), def5678 (Second task)`,
			taskID:        "3",
			wantBlockedBy: []string{"abc1234", "def5678"},
		},
		"blocked_by_without_title_hint": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc1234`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
		},
		"blocked_by_mixed_with_and_without_hints": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
- [ ] 3. Third task <!-- id:ghi9012 -->
  - Blocked-by: abc1234 (First task), def5678`,
			taskID:        "3",
			wantBlockedBy: []string{"abc1234", "def5678"},
		},
		"task_without_blocked_by": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->`,
			taskID:        "1",
			wantBlockedBy: nil,
		},
		"blocked_by_with_parentheses_in_title": {
			content: `# Tasks
- [ ] 1. Setup (Phase 1) <!-- id:abc1234 -->
- [ ] 2. Build <!-- id:def5678 -->
  - Blocked-by: abc1234 (Setup (Phase 1))`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
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

			if len(task.BlockedBy) != len(tc.wantBlockedBy) {
				t.Errorf("Task %s BlockedBy count = %d, want %d", tc.taskID, len(task.BlockedBy), len(tc.wantBlockedBy))
				return
			}

			for i, want := range tc.wantBlockedBy {
				if i >= len(task.BlockedBy) {
					break
				}
				if task.BlockedBy[i] != want {
					t.Errorf("Task %s BlockedBy[%d] = %q, want %q", tc.taskID, i, task.BlockedBy[i], want)
				}
			}
		})
	}
}

// TestParseBlockedByCaseInsensitive tests case-insensitive parsing of Blocked-by
func TestParseBlockedByCaseInsensitive(t *testing.T) {
	tests := map[string]struct {
		content       string
		taskID        string
		wantBlockedBy []string
	}{
		"lowercase_blocked_by": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - blocked-by: abc1234 (First task)`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
		},
		"uppercase_blocked_by": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - BLOCKED-BY: abc1234 (First task)`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
		},
		"mixed_case_blocked_by": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-By: abc1234 (First task)`,
			taskID:        "2",
			wantBlockedBy: []string{"abc1234"},
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

			if len(task.BlockedBy) != len(tc.wantBlockedBy) {
				t.Errorf("Task %s BlockedBy count = %d, want %d", tc.taskID, len(task.BlockedBy), len(tc.wantBlockedBy))
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

// TestParseStream tests parsing of Stream metadata
func TestParseStream(t *testing.T) {
	tests := map[string]struct {
		content    string
		taskID     string
		wantStream int
	}{
		"task_with_stream": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Stream: 2`,
			taskID:     "1",
			wantStream: 2,
		},
		"task_with_stream_1": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Stream: 1`,
			taskID:     "1",
			wantStream: 1,
		},
		"task_without_stream": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->`,
			taskID:     "1",
			wantStream: 0, // Not explicitly set
		},
		"subtask_with_stream": {
			content: `# Tasks
- [ ] 1. Parent task <!-- id:abc1234 -->
  - [ ] 1.1. Child task <!-- id:def5678 -->
    - Stream: 3`,
			taskID:     "1.1",
			wantStream: 3,
		},
		"task_with_large_stream": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Stream: 100`,
			taskID:     "1",
			wantStream: 100,
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

			if task.Stream != tc.wantStream {
				t.Errorf("Task %s Stream = %d, want %d", tc.taskID, task.Stream, tc.wantStream)
			}
		})
	}
}

// TestParseStreamCaseInsensitive tests case-insensitive parsing of Stream
func TestParseStreamCaseInsensitive(t *testing.T) {
	tests := map[string]struct {
		content    string
		taskID     string
		wantStream int
	}{
		"lowercase_stream": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - stream: 2`,
			taskID:     "1",
			wantStream: 2,
		},
		"uppercase_stream": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - STREAM: 2`,
			taskID:     "1",
			wantStream: 2,
		},
		"mixed_case_stream": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Stream: 2`,
			taskID:     "1",
			wantStream: 2,
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

			if task.Stream != tc.wantStream {
				t.Errorf("Task %s Stream = %d, want %d", tc.taskID, task.Stream, tc.wantStream)
			}
		})
	}
}

// TestParseOwner tests parsing of Owner metadata
func TestParseOwner(t *testing.T) {
	tests := map[string]struct {
		content   string
		taskID    string
		wantOwner string
	}{
		"task_with_owner": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Owner: agent-1`,
			taskID:    "1",
			wantOwner: "agent-1",
		},
		"task_without_owner": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->`,
			taskID:    "1",
			wantOwner: "",
		},
		"task_with_owner_spaces": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Owner: my agent name`,
			taskID:    "1",
			wantOwner: "my agent name",
		},
		"task_with_owner_special_chars": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Owner: agent_v2.1-beta`,
			taskID:    "1",
			wantOwner: "agent_v2.1-beta",
		},
		"subtask_with_owner": {
			content: `# Tasks
- [ ] 1. Parent task <!-- id:abc1234 -->
  - [ ] 1.1. Child task <!-- id:def5678 -->
    - Owner: sub-agent`,
			taskID:    "1.1",
			wantOwner: "sub-agent",
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

			if task.Owner != tc.wantOwner {
				t.Errorf("Task %s Owner = %q, want %q", tc.taskID, task.Owner, tc.wantOwner)
			}
		})
	}
}

// TestParseOwnerCaseInsensitive tests case-insensitive parsing of Owner
func TestParseOwnerCaseInsensitive(t *testing.T) {
	tests := map[string]struct {
		content   string
		taskID    string
		wantOwner string
	}{
		"lowercase_owner": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - owner: agent-1`,
			taskID:    "1",
			wantOwner: "agent-1",
		},
		"uppercase_owner": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - OWNER: agent-1`,
			taskID:    "1",
			wantOwner: "agent-1",
		},
		"mixed_case_owner": {
			content: `# Tasks
- [ ] 1. First task <!-- id:abc1234 -->
  - Owner: agent-1`,
			taskID:    "1",
			wantOwner: "agent-1",
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

			if task.Owner != tc.wantOwner {
				t.Errorf("Task %s Owner = %q, want %q", tc.taskID, task.Owner, tc.wantOwner)
			}
		})
	}
}

// TestParseLegacyFiles tests that files without new metadata fields parse successfully
func TestParseLegacyFiles(t *testing.T) {
	tests := map[string]struct {
		content   string
		wantTasks int
	}{
		"legacy_simple_tasks": {
			content: `# Tasks
- [ ] 1. First task
- [-] 2. Second task
- [x] 3. Third task`,
			wantTasks: 3,
		},
		"legacy_with_details": {
			content: `# Tasks
- [ ] 1. Main task
  - Detail one
  - Detail two
  - References: ref.md`,
			wantTasks: 1,
		},
		"legacy_with_subtasks": {
			content: `# Tasks
- [ ] 1. Parent
  - [ ] 1.1. Child one
  - [ ] 1.2. Child two`,
			wantTasks: 1,
		},
		"legacy_with_requirements": {
			content: `# Tasks
- [ ] 1. Task with requirement
  - Requirements: [1.1](requirements.md#1.1)`,
			wantTasks: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error: %v", err)
			}

			if len(tl.Tasks) != tc.wantTasks {
				t.Errorf("ParseMarkdown() tasks count = %d, want %d", len(tl.Tasks), tc.wantTasks)
			}

			// Verify legacy tasks have empty new fields
			for _, task := range tl.Tasks {
				if task.StableID != "" {
					t.Errorf("Legacy task %s should have empty StableID, got %q", task.ID, task.StableID)
				}
				if len(task.BlockedBy) != 0 {
					t.Errorf("Legacy task %s should have empty BlockedBy, got %v", task.ID, task.BlockedBy)
				}
				if task.Stream != 0 {
					t.Errorf("Legacy task %s should have Stream = 0, got %d", task.ID, task.Stream)
				}
				if task.Owner != "" {
					t.Errorf("Legacy task %s should have empty Owner, got %q", task.ID, task.Owner)
				}
			}
		})
	}
}

// TestParseAllMetadataTogether tests parsing tasks with all new metadata fields
func TestParseAllMetadataTogether(t *testing.T) {
	content := `# Tasks

- [ ] 1. First task <!-- id:abc1234 -->
  - Setup the project
  - Stream: 1

- [ ] 2. Second task <!-- id:def5678 -->
  - Blocked-by: abc1234 (First task)
  - Stream: 2
  - Owner: agent-backend

- [ ] 3. Third task <!-- id:ghi9012 -->
  - Blocked-by: abc1234 (First task), def5678 (Second task)
  - Stream: 2
  - Owner: agent-frontend
  - References: design.md`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	if len(tl.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tl.Tasks))
	}

	// Task 1
	task1 := tl.FindTask("1")
	if task1 == nil {
		t.Fatal("Task 1 not found")
	}
	if task1.StableID != "abc1234" {
		t.Errorf("Task 1 StableID = %q, want %q", task1.StableID, "abc1234")
	}
	if task1.Stream != 1 {
		t.Errorf("Task 1 Stream = %d, want %d", task1.Stream, 1)
	}
	if len(task1.Details) != 1 || task1.Details[0] != "Setup the project" {
		t.Errorf("Task 1 Details = %v, want [Setup the project]", task1.Details)
	}

	// Task 2
	task2 := tl.FindTask("2")
	if task2 == nil {
		t.Fatal("Task 2 not found")
	}
	if task2.StableID != "def5678" {
		t.Errorf("Task 2 StableID = %q, want %q", task2.StableID, "def5678")
	}
	if len(task2.BlockedBy) != 1 || task2.BlockedBy[0] != "abc1234" {
		t.Errorf("Task 2 BlockedBy = %v, want [abc1234]", task2.BlockedBy)
	}
	if task2.Stream != 2 {
		t.Errorf("Task 2 Stream = %d, want %d", task2.Stream, 2)
	}
	if task2.Owner != "agent-backend" {
		t.Errorf("Task 2 Owner = %q, want %q", task2.Owner, "agent-backend")
	}

	// Task 3
	task3 := tl.FindTask("3")
	if task3 == nil {
		t.Fatal("Task 3 not found")
	}
	if task3.StableID != "ghi9012" {
		t.Errorf("Task 3 StableID = %q, want %q", task3.StableID, "ghi9012")
	}
	if len(task3.BlockedBy) != 2 {
		t.Errorf("Task 3 BlockedBy count = %d, want 2", len(task3.BlockedBy))
	}
	if task3.Stream != 2 {
		t.Errorf("Task 3 Stream = %d, want %d", task3.Stream, 2)
	}
	if task3.Owner != "agent-frontend" {
		t.Errorf("Task 3 Owner = %q, want %q", task3.Owner, "agent-frontend")
	}
	if len(task3.References) != 1 || task3.References[0] != "design.md" {
		t.Errorf("Task 3 References = %v, want [design.md]", task3.References)
	}
}

// TestParseMixedLegacyAndNewTasks tests files with both legacy and new-style tasks
func TestParseMixedLegacyAndNewTasks(t *testing.T) {
	content := `# Tasks

- [ ] 1. Legacy task without stable ID
  - Some detail

- [ ] 2. New task with stable ID <!-- id:abc1234 -->
  - Stream: 2
  - Owner: agent-1

- [ ] 3. Another legacy task`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	if len(tl.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tl.Tasks))
	}

	// Task 1 - legacy
	task1 := tl.FindTask("1")
	if task1.StableID != "" {
		t.Errorf("Task 1 should have no StableID, got %q", task1.StableID)
	}

	// Task 2 - new style
	task2 := tl.FindTask("2")
	if task2.StableID != "abc1234" {
		t.Errorf("Task 2 StableID = %q, want %q", task2.StableID, "abc1234")
	}
	if task2.Stream != 2 {
		t.Errorf("Task 2 Stream = %d, want 2", task2.Stream)
	}
	if task2.Owner != "agent-1" {
		t.Errorf("Task 2 Owner = %q, want %q", task2.Owner, "agent-1")
	}

	// Task 3 - legacy
	task3 := tl.FindTask("3")
	if task3.StableID != "" {
		t.Errorf("Task 3 should have no StableID, got %q", task3.StableID)
	}
}

// TestParseMetadataOrder tests that metadata can appear in any order
func TestParseMetadataOrder(t *testing.T) {
	tests := map[string]struct {
		content       string
		wantBlockedBy []string
		wantStream    int
		wantOwner     string
	}{
		"stream_blocked_by_owner": {
			content: `# Tasks
- [ ] 1. Dep task <!-- id:abc1234 -->
- [ ] 2. Task <!-- id:def5678 -->
  - Stream: 2
  - Blocked-by: abc1234 (Dep task)
  - Owner: agent-1`,
			wantBlockedBy: []string{"abc1234"},
			wantStream:    2,
			wantOwner:     "agent-1",
		},
		"owner_stream_blocked_by": {
			content: `# Tasks
- [ ] 1. Dep task <!-- id:abc1234 -->
- [ ] 2. Task <!-- id:def5678 -->
  - Owner: agent-1
  - Stream: 2
  - Blocked-by: abc1234 (Dep task)`,
			wantBlockedBy: []string{"abc1234"},
			wantStream:    2,
			wantOwner:     "agent-1",
		},
		"blocked_by_owner_stream": {
			content: `# Tasks
- [ ] 1. Dep task <!-- id:abc1234 -->
- [ ] 2. Task <!-- id:def5678 -->
  - Blocked-by: abc1234 (Dep task)
  - Owner: agent-1
  - Stream: 2`,
			wantBlockedBy: []string{"abc1234"},
			wantStream:    2,
			wantOwner:     "agent-1",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error: %v", err)
			}

			task := tl.FindTask("2")
			if task == nil {
				t.Fatal("Task 2 not found")
			}

			if len(task.BlockedBy) != len(tc.wantBlockedBy) {
				t.Errorf("BlockedBy count = %d, want %d", len(task.BlockedBy), len(tc.wantBlockedBy))
			}
			for i, want := range tc.wantBlockedBy {
				if i < len(task.BlockedBy) && task.BlockedBy[i] != want {
					t.Errorf("BlockedBy[%d] = %q, want %q", i, task.BlockedBy[i], want)
				}
			}

			if task.Stream != tc.wantStream {
				t.Errorf("Stream = %d, want %d", task.Stream, tc.wantStream)
			}

			if task.Owner != tc.wantOwner {
				t.Errorf("Owner = %q, want %q", task.Owner, tc.wantOwner)
			}
		})
	}
}

// TestParseMetadataWithDetails tests that details and metadata coexist correctly
func TestParseMetadataWithDetails(t *testing.T) {
	content := `# Tasks

- [ ] 1. Dependency task <!-- id:abc1234 -->
- [ ] 2. Main task <!-- id:def5678 -->
  - First detail
  - Blocked-by: abc1234 (Dependency task)
  - Second detail
  - Stream: 2
  - Third detail
  - Owner: agent-1
  - References: ref.md`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	task := tl.FindTask("2")
	if task == nil {
		t.Fatal("Task 2 not found")
	}

	// Details should not include metadata lines
	wantDetails := []string{"First detail", "Second detail", "Third detail"}
	if len(task.Details) != len(wantDetails) {
		t.Errorf("Details count = %d, want %d. Got: %v", len(task.Details), len(wantDetails), task.Details)
	}
	for i, want := range wantDetails {
		if i < len(task.Details) && task.Details[i] != want {
			t.Errorf("Details[%d] = %q, want %q", i, task.Details[i], want)
		}
	}

	// Metadata should be parsed correctly
	if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "abc1234" {
		t.Errorf("BlockedBy = %v, want [abc1234]", task.BlockedBy)
	}
	if task.Stream != 2 {
		t.Errorf("Stream = %d, want 2", task.Stream)
	}
	if task.Owner != "agent-1" {
		t.Errorf("Owner = %q, want %q", task.Owner, "agent-1")
	}
	if len(task.References) != 1 || task.References[0] != "ref.md" {
		t.Errorf("References = %v, want [ref.md]", task.References)
	}
}

// TestParseStableIDNotInTitle tests that stable ID is removed from title
func TestParseStableIDNotInTitle(t *testing.T) {
	content := `# Tasks
- [ ] 1. My task title <!-- id:abc1234 -->`

	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error: %v", err)
	}

	task := tl.FindTask("1")
	if task == nil {
		t.Fatal("Task 1 not found")
	}

	// Title should not contain the HTML comment
	if strings.Contains(task.Title, "<!--") {
		t.Errorf("Title should not contain HTML comment, got %q", task.Title)
	}
	if strings.Contains(task.Title, "id:") {
		t.Errorf("Title should not contain 'id:', got %q", task.Title)
	}
	if task.Title != "My task title" {
		t.Errorf("Title = %q, want %q", task.Title, "My task title")
	}
}

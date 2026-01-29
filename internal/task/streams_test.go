package task

import (
	"testing"
)

func TestGetEffectiveStream(t *testing.T) {
	tests := map[string]struct {
		task       *Task
		wantStream int
	}{
		"stream_not_set": {
			task:       &Task{ID: "1", Title: "Task", Stream: 0},
			wantStream: 1,
		},
		"stream_explicitly_set_to_1": {
			task:       &Task{ID: "1", Title: "Task", Stream: 1},
			wantStream: 1,
		},
		"stream_set_to_2": {
			task:       &Task{ID: "1", Title: "Task", Stream: 2},
			wantStream: 2,
		},
		"stream_set_to_large_value": {
			task:       &Task{ID: "1", Title: "Task", Stream: 100},
			wantStream: 100,
		},
		"negative_stream_defaults_to_1": {
			task:       &Task{ID: "1", Title: "Task", Stream: -1},
			wantStream: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetEffectiveStream(tc.task)
			if got != tc.wantStream {
				t.Errorf("GetEffectiveStream() = %d, want %d", got, tc.wantStream)
			}
		})
	}
}

func TestFilterByStream(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Stream 1 task", Stream: 1},
		{ID: "2", Title: "Stream 2 task", Stream: 2},
		{ID: "3", Title: "Default stream task", Stream: 0}, // defaults to stream 1
		{ID: "4", Title: "Another stream 2", Stream: 2},
		{ID: "5", Title: "Stream 3 task", Stream: 3},
	}

	tests := map[string]struct {
		stream  int
		wantIDs []string
	}{
		"filter_stream_1": {
			stream:  1,
			wantIDs: []string{"1", "3"}, // includes default stream task
		},
		"filter_stream_2": {
			stream:  2,
			wantIDs: []string{"2", "4"},
		},
		"filter_stream_3": {
			stream:  3,
			wantIDs: []string{"5"},
		},
		"filter_non_existent_stream": {
			stream:  99,
			wantIDs: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FilterByStream(tasks, tc.stream)
			if len(got) != len(tc.wantIDs) {
				t.Errorf("FilterByStream(stream=%d) returned %d tasks, want %d",
					tc.stream, len(got), len(tc.wantIDs))
				return
			}
			for i, wantID := range tc.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("FilterByStream(stream=%d)[%d].ID = %q, want %q",
						tc.stream, i, got[i].ID, wantID)
				}
			}
		})
	}
}

func TestAnalyzeStreams_SingleStream(t *testing.T) {
	// All tasks in default stream (1)
	tasks := []Task{
		{ID: "1", Title: "Ready task", StableID: "abc0001", Status: Pending},
		{ID: "2", Title: "Active task", StableID: "abc0002", Status: InProgress},
		{ID: "3", Title: "Blocked task", StableID: "abc0003", Status: Pending, BlockedBy: []string{"abc0001"}},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	if len(result.Streams) != 1 {
		t.Errorf("AnalyzeStreams() returned %d streams, want 1", len(result.Streams))
		return
	}

	stream := result.Streams[0]
	if stream.ID != 1 {
		t.Errorf("Stream ID = %d, want 1", stream.ID)
	}

	// Task 1 is ready (no blockers, pending status)
	if len(stream.Ready) != 1 || stream.Ready[0] != "1" {
		t.Errorf("Ready tasks = %v, want [1]", stream.Ready)
	}

	// Task 2 is active (in progress)
	if len(stream.Active) != 1 || stream.Active[0] != "2" {
		t.Errorf("Active tasks = %v, want [2]", stream.Active)
	}

	// Task 3 is blocked (depends on task 1 which is not completed)
	if len(stream.Blocked) != 1 || stream.Blocked[0] != "3" {
		t.Errorf("Blocked tasks = %v, want [3]", stream.Blocked)
	}

	// Stream 1 has a ready task
	if len(result.Available) != 1 || result.Available[0] != 1 {
		t.Errorf("Available streams = %v, want [1]", result.Available)
	}
}

func TestAnalyzeStreams_MultipleStreams(t *testing.T) {
	tasks := []Task{
		// Stream 1
		{ID: "1", Title: "Stream 1 ready", StableID: "abc0001", Status: Pending, Stream: 1},
		{ID: "2", Title: "Stream 1 active", StableID: "abc0002", Status: InProgress, Stream: 1},
		// Stream 2
		{ID: "3", Title: "Stream 2 blocked", StableID: "abc0003", Status: Pending, Stream: 2, BlockedBy: []string{"abc0001"}},
		// Stream 3
		{ID: "4", Title: "Stream 3 ready", StableID: "abc0004", Status: Pending, Stream: 3},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	if len(result.Streams) != 3 {
		t.Errorf("AnalyzeStreams() returned %d streams, want 3", len(result.Streams))
		return
	}

	// Verify stream ordering (should be sorted by ID)
	for i, s := range result.Streams {
		if s.ID != i+1 {
			t.Errorf("Streams[%d].ID = %d, want %d", i, s.ID, i+1)
		}
	}

	// Stream 1: 1 ready, 1 active, 0 blocked
	stream1 := result.Streams[0]
	if len(stream1.Ready) != 1 {
		t.Errorf("Stream 1 ready count = %d, want 1", len(stream1.Ready))
	}
	if len(stream1.Active) != 1 {
		t.Errorf("Stream 1 active count = %d, want 1", len(stream1.Active))
	}
	if len(stream1.Blocked) != 0 {
		t.Errorf("Stream 1 blocked count = %d, want 0", len(stream1.Blocked))
	}

	// Stream 2: 0 ready, 0 active, 1 blocked
	stream2 := result.Streams[1]
	if len(stream2.Ready) != 0 {
		t.Errorf("Stream 2 ready count = %d, want 0", len(stream2.Ready))
	}
	if len(stream2.Blocked) != 1 {
		t.Errorf("Stream 2 blocked count = %d, want 1", len(stream2.Blocked))
	}

	// Stream 3: 1 ready, 0 active, 0 blocked
	stream3 := result.Streams[2]
	if len(stream3.Ready) != 1 {
		t.Errorf("Stream 3 ready count = %d, want 0", len(stream3.Ready))
	}

	// Available streams: 1 and 3 (have ready tasks)
	if len(result.Available) != 2 {
		t.Errorf("Available count = %d, want 2", len(result.Available))
	}
}

func TestAnalyzeStreams_DefaultStreamAssignment(t *testing.T) {
	// Tasks without explicit stream should be in stream 1
	tasks := []Task{
		{ID: "1", Title: "No stream set", StableID: "abc0001", Status: Pending, Stream: 0},
		{ID: "2", Title: "Explicit stream 1", StableID: "abc0002", Status: Pending, Stream: 1},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	// Should have only one stream (stream 1)
	if len(result.Streams) != 1 {
		t.Errorf("AnalyzeStreams() returned %d streams, want 1", len(result.Streams))
		return
	}

	// Both tasks should be in stream 1
	stream := result.Streams[0]
	if stream.ID != 1 {
		t.Errorf("Stream ID = %d, want 1", stream.ID)
	}
	if len(stream.Ready) != 2 {
		t.Errorf("Stream 1 ready count = %d, want 2", len(stream.Ready))
	}
}

func TestAnalyzeStreams_NestedTasks(t *testing.T) {
	tasks := []Task{
		{
			ID:       "1",
			Title:    "Parent",
			StableID: "abc0001",
			Status:   Pending,
			Stream:   1,
			Children: []Task{
				{ID: "1.1", Title: "Child stream 1", StableID: "abc0011", Status: Pending, Stream: 1},
				{ID: "1.2", Title: "Child stream 2", StableID: "abc0012", Status: Pending, Stream: 2},
			},
		},
		{ID: "2", Title: "Stream 2 task", StableID: "abc0002", Status: Pending, Stream: 2},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	if len(result.Streams) != 2 {
		t.Errorf("AnalyzeStreams() returned %d streams, want 2", len(result.Streams))
		return
	}

	// Stream 1: parent (1) and child (1.1)
	stream1 := result.Streams[0]
	if len(stream1.Ready) != 2 {
		t.Errorf("Stream 1 ready count = %d, want 2 (tasks 1 and 1.1)", len(stream1.Ready))
	}

	// Stream 2: child (1.2) and task 2
	stream2 := result.Streams[1]
	if len(stream2.Ready) != 2 {
		t.Errorf("Stream 2 ready count = %d, want 2 (tasks 1.2 and 2)", len(stream2.Ready))
	}
}

func TestAnalyzeStreams_EmptyTaskList(t *testing.T) {
	tasks := []Task{}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	if len(result.Streams) != 0 {
		t.Errorf("AnalyzeStreams() returned %d streams for empty list, want 0", len(result.Streams))
	}
	if len(result.Available) != 0 {
		t.Errorf("Available streams = %v for empty list, want []", result.Available)
	}
}

func TestAnalyzeStreams_AllTasksCompleted(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Completed 1", StableID: "abc0001", Status: Completed, Stream: 1},
		{ID: "2", Title: "Completed 2", StableID: "abc0002", Status: Completed, Stream: 2},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	// Both streams exist but have no ready tasks
	if len(result.Streams) != 2 {
		t.Errorf("AnalyzeStreams() returned %d streams, want 2", len(result.Streams))
	}

	// No streams should be available (no ready tasks)
	if len(result.Available) != 0 {
		t.Errorf("Available streams = %v, want [] (all tasks completed)", result.Available)
	}
}

func TestAnalyzeStreams_CrossStreamDependencies(t *testing.T) {
	// Task in stream 2 depends on task in stream 1
	tasks := []Task{
		{ID: "1", Title: "Stream 1 task", StableID: "abc0001", Status: Pending, Stream: 1},
		{ID: "2", Title: "Stream 2 blocked by stream 1", StableID: "abc0002", Status: Pending, Stream: 2, BlockedBy: []string{"abc0001"}},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	// Stream 2 task should be blocked since stream 1 task is not completed
	stream2 := result.Streams[1]
	if len(stream2.Blocked) != 1 || stream2.Blocked[0] != "2" {
		t.Errorf("Stream 2 blocked tasks = %v, want [2]", stream2.Blocked)
	}
	if len(stream2.Ready) != 0 {
		t.Errorf("Stream 2 ready tasks = %v, want []", stream2.Ready)
	}
}

func TestAnalyzeStreams_ReadyTaskWithOwnerNotAvailable(t *testing.T) {
	// Task with owner should be counted as active, not ready
	// Design says: Ready = All blockers Completed AND Status == Pending AND no Owner
	tasks := []Task{
		{ID: "1", Title: "Owned pending task", StableID: "abc0001", Status: Pending, Owner: "agent-1"},
		{ID: "2", Title: "Unowned pending task", StableID: "abc0002", Status: Pending},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	stream := result.Streams[0]
	// Task 1 has an owner, so it should not be in Ready
	// Task 2 is unowned and pending, so it should be Ready
	if len(stream.Ready) != 1 || stream.Ready[0] != "2" {
		t.Errorf("Ready tasks = %v, want [2]", stream.Ready)
	}
}

func TestAnalyzeStreams_StreamsSortedByID(t *testing.T) {
	// Create tasks in non-sequential stream order
	tasks := []Task{
		{ID: "1", Title: "Stream 5", StableID: "abc0001", Status: Pending, Stream: 5},
		{ID: "2", Title: "Stream 2", StableID: "abc0002", Status: Pending, Stream: 2},
		{ID: "3", Title: "Stream 10", StableID: "abc0003", Status: Pending, Stream: 10},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	if len(result.Streams) != 3 {
		t.Errorf("AnalyzeStreams() returned %d streams, want 3", len(result.Streams))
		return
	}

	// Should be sorted: 2, 5, 10
	expectedOrder := []int{2, 5, 10}
	for i, expected := range expectedOrder {
		if result.Streams[i].ID != expected {
			t.Errorf("Streams[%d].ID = %d, want %d", i, result.Streams[i].ID, expected)
		}
	}
}

func TestAnalyzeStreams_AvailableStreamsSorted(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Stream 5 ready", StableID: "abc0001", Status: Pending, Stream: 5},
		{ID: "2", Title: "Stream 2 ready", StableID: "abc0002", Status: Pending, Stream: 2},
		{ID: "3", Title: "Stream 3 blocked", StableID: "abc0003", Status: Pending, Stream: 3, BlockedBy: []string{"abc0001"}},
	}
	idx := BuildDependencyIndex(tasks)

	result := AnalyzeStreams(tasks, idx)

	// Available streams should be sorted: 2, 5 (not 3, it's blocked)
	if len(result.Available) != 2 {
		t.Errorf("Available count = %d, want 2", len(result.Available))
		return
	}
	if result.Available[0] != 2 || result.Available[1] != 5 {
		t.Errorf("Available = %v, want [2, 5]", result.Available)
	}
}

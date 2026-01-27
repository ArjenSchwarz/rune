package task

import "sort"

// StreamStatus represents the state of a single stream
type StreamStatus struct {
	ID      int      `json:"id"`
	Ready   []string `json:"ready"`   // Hierarchical IDs of ready tasks
	Blocked []string `json:"blocked"` // Hierarchical IDs of blocked tasks
	Active  []string `json:"active"`  // Hierarchical IDs of in-progress tasks
}

// StreamsResult contains status for all streams
type StreamsResult struct {
	Streams   []StreamStatus `json:"streams"`
	Available []int          `json:"available"` // Stream IDs with ready tasks
}

// AnalyzeStreams computes stream status from a task list.
// Ready tasks are pending with all blockers completed and no owner.
// Blocked tasks are pending with at least one incomplete blocker.
// Active tasks are in-progress.
func AnalyzeStreams(tasks []Task, index *DependencyIndex) *StreamsResult {
	// Map stream ID -> StreamStatus
	streamMap := make(map[int]*StreamStatus)

	// Recursively process all tasks
	var processTasks func(tasks []Task)
	processTasks = func(tasks []Task) {
		for i := range tasks {
			task := &tasks[i]
			streamID := GetEffectiveStream(task)

			// Initialize stream if not seen before
			if _, exists := streamMap[streamID]; !exists {
				streamMap[streamID] = &StreamStatus{
					ID:      streamID,
					Ready:   []string{},
					Blocked: []string{},
					Active:  []string{},
				}
			}

			stream := streamMap[streamID]

			// Classify the task
			switch {
			case task.Status == InProgress:
				// Active: task is in progress
				stream.Active = append(stream.Active, task.ID)
			case task.Status == Completed:
				// Completed tasks are not included in any category
			case index.IsBlocked(task):
				// Blocked: has incomplete blockers
				stream.Blocked = append(stream.Blocked, task.ID)
			case task.Owner != "":
				// Owned pending tasks are not ready (someone claimed but hasn't started)
				// They're neither blocked nor active - skip them
			default:
				// Ready: pending, no blockers (or all blockers completed), no owner
				stream.Ready = append(stream.Ready, task.ID)
			}

			// Process children
			if len(task.Children) > 0 {
				processTasks(task.Children)
			}
		}
	}

	processTasks(tasks)

	// Convert map to sorted slice
	result := &StreamsResult{
		Streams:   make([]StreamStatus, 0, len(streamMap)),
		Available: []int{},
	}

	// Get stream IDs and sort them
	streamIDs := make([]int, 0, len(streamMap))
	for id := range streamMap {
		streamIDs = append(streamIDs, id)
	}
	sort.Ints(streamIDs)

	// Build result in sorted order
	for _, id := range streamIDs {
		stream := streamMap[id]
		result.Streams = append(result.Streams, *stream)

		// Stream is available if it has ready tasks
		if len(stream.Ready) > 0 {
			result.Available = append(result.Available, id)
		}
	}

	return result
}

// FilterByStream returns tasks belonging to the specified stream.
// Uses GetEffectiveStream to handle default stream (1) for tasks without explicit stream.
func FilterByStream(tasks []Task, stream int) []Task {
	var result []Task
	for _, task := range tasks {
		if GetEffectiveStream(&task) == stream {
			result = append(result, task)
		}
	}
	if result == nil {
		return []Task{}
	}
	return result
}

package task

// PhaseMarker represents a phase header found during parsing
// This is a transient structure used only during parsing/rendering
type PhaseMarker struct {
	Name        string // Phase name from H2 header
	AfterTaskID string // ID of task that precedes this phase (empty if at start)
}

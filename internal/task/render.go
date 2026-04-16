package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// jsonTask is a JSON-specific task representation that emits effective stream
// values (never omitted) and translates BlockedBy stable IDs to hierarchical IDs.
// When adding new fields to Task, update this struct and toJSONTask accordingly.
type jsonTask struct {
	ID           string     `json:"ID"`
	Title        string     `json:"Title"`
	Status       Status     `json:"Status"`
	Details      []string   `json:"Details"`
	References   []string   `json:"References"`
	Requirements []string   `json:"requirements,omitempty"`
	Children     []jsonTask `json:"Children"`
	ParentID     string     `json:"ParentID"`
	BlockedBy    []string   `json:"blockedBy,omitempty"`
	Stream       int        `json:"stream"`
	Owner        string     `json:"owner,omitempty"`
}

// toJSONTasks converts a slice of Tasks to jsonTask representations,
// using the dependency index to translate BlockedBy stable IDs to hierarchical IDs
// and GetEffectiveStream to emit the effective stream value.
func toJSONTasks(tasks []Task, index *DependencyIndex) []jsonTask {
	result := make([]jsonTask, len(tasks))
	for i := range tasks {
		result[i] = toJSONTask(&tasks[i], index)
	}
	return result
}

// toJSONTask converts a single Task to its JSON representation.
func toJSONTask(t *Task, index *DependencyIndex) jsonTask {
	jt := jsonTask{
		ID:           t.ID,
		Title:        t.Title,
		Status:       t.Status,
		Details:      t.Details,
		References:   t.References,
		Requirements: t.Requirements,
		ParentID:     t.ParentID,
		Stream:       GetEffectiveStream(t),
		Owner:        t.Owner,
	}

	// Translate BlockedBy stable IDs to hierarchical IDs
	if len(t.BlockedBy) > 0 && index != nil {
		jt.BlockedBy = index.TranslateToHierarchical(t.BlockedBy)
	}

	// Recursively convert children
	if len(t.Children) > 0 {
		jt.Children = toJSONTasks(t.Children, index)
	}

	return jt
}

// TaskListJSON represents a TaskList with statistics for JSON output
type TaskListJSON struct {
	Success          bool         `json:"success"`
	Count            int          `json:"count"`
	Title            string       `json:"Title"`
	Tasks            []jsonTask   `json:"Tasks"`
	Stats            Stats        `json:"Stats"`
	FrontMatter      *FrontMatter `json:"FrontMatter,omitempty"`
	RequirementsFile string       `json:"requirements_file,omitempty"`
}

// TaskListJSONWithPhases represents a TaskList with phases and statistics for JSON output
type TaskListJSONWithPhases struct {
	Success      bool                `json:"success"`
	Count        int                 `json:"count"`
	Title        string              `json:"Title"`
	Tasks        []jsonTaskWithPhase `json:"Tasks"`
	Stats        Stats               `json:"Stats"`
	FrontMatter  *FrontMatter        `json:"FrontMatter,omitempty"`
	PhaseMarkers []PhaseMarker       `json:"PhaseMarkers,omitempty"`
}

// jsonTaskWithPhase represents a JSON task with its phase information
type jsonTaskWithPhase struct {
	jsonTask
	Phase string `json:"Phase,omitempty"`
}

// TaskWithPhase represents a task with its phase information (used for non-JSON rendering)
type TaskWithPhase struct {
	*Task
	Phase string `json:"Phase,omitempty"`
}

// RenderContext provides dependencies needed for rendering
type RenderContext struct {
	RequirementsFile string
	DependencyIndex  *DependencyIndex // For title hint lookup in Blocked-by
}

// RenderMarkdown converts a TaskList to markdown format with consistent formatting
// Note: This does NOT include front matter - that's handled by WriteFile
func RenderMarkdown(tl *TaskList) []byte {
	var buf strings.Builder

	// Write title as H1 header
	if tl.Title != "" {
		fmt.Fprintf(&buf, "# %s\n\n", tl.Title)
	} else {
		buf.WriteString("# \n\n")
	}

	// Determine requirements file (default if not set)
	reqFile := tl.RequirementsFile
	if reqFile == "" {
		reqFile = DefaultRequirementsFile
	}

	// Build dependency index for blocked-by title hints
	index := BuildDependencyIndex(tl.Tasks)

	ctx := &RenderContext{
		RequirementsFile: reqFile,
		DependencyIndex:  index,
	}

	// Render each root-level task
	for i, task := range tl.Tasks {
		// Add a blank line before each top-level task except the first
		if i > 0 {
			buf.WriteString("\n")
		}
		renderTask(&buf, &task, 0, ctx)
	}

	return []byte(buf.String())
}

// renderTask recursively renders a task and its children with proper indentation
func renderTask(buf *strings.Builder, task *Task, depth int, ctx *RenderContext) {
	// Calculate indentation (2 spaces per level)
	indent := strings.Repeat("  ", depth)

	// Render the task checkbox, title, and stable ID (if present)
	if task.StableID != "" {
		fmt.Fprintf(buf, "%s- %s %s. %s <!-- id:%s -->\n",
			indent, task.Status.String(), task.ID, task.Title, task.StableID)
	} else {
		fmt.Fprintf(buf, "%s- %s %s. %s\n",
			indent, task.Status.String(), task.ID, task.Title)
	}

	// Render task details as bullet points (first in metadata order)
	for _, detail := range task.Details {
		fmt.Fprintf(buf, "%s  - %s\n", indent, detail)
	}

	// Render Blocked-by with title hints (second in metadata order)
	if len(task.BlockedBy) > 0 {
		refs := formatBlockedByRefs(task.BlockedBy, ctx.DependencyIndex)
		fmt.Fprintf(buf, "%s  - Blocked-by: %s\n", indent, refs)
	}

	// Render Stream (third in metadata order) - only if explicitly set (> 0)
	if task.Stream > 0 {
		fmt.Fprintf(buf, "%s  - Stream: %d\n", indent, task.Stream)
	}

	// Render Owner (fourth in metadata order) - only if non-empty
	if task.Owner != "" {
		fmt.Fprintf(buf, "%s  - Owner: %s\n", indent, task.Owner)
	}

	// Render requirements if present (fifth in metadata order)
	if len(task.Requirements) > 0 {
		links := make([]string, len(task.Requirements))
		for i, reqID := range task.Requirements {
			links[i] = fmt.Sprintf("[%s](%s#%s)", reqID, ctx.RequirementsFile, reqID)
		}
		fmt.Fprintf(buf, "%s  - Requirements: %s\n",
			indent, strings.Join(links, ", "))
	}

	// Render references if present (last in metadata order)
	if len(task.References) > 0 {
		fmt.Fprintf(buf, "%s  - References: %s\n",
			indent, strings.Join(task.References, ", "))
	}

	// Recursively render children
	for _, child := range task.Children {
		renderTask(buf, &child, depth+1, ctx)
	}
}

// formatBlockedByRefs formats stable IDs with current title hints
func formatBlockedByRefs(stableIDs []string, index *DependencyIndex) string {
	if len(stableIDs) == 0 {
		return ""
	}
	refs := make([]string, 0, len(stableIDs))
	for _, id := range stableIDs {
		if index != nil {
			if task := index.GetTask(id); task != nil {
				refs = append(refs, fmt.Sprintf("%s (%s)", id, task.Title))
				continue
			}
		}
		// Fallback: ID only if task not found
		refs = append(refs, id)
	}
	return strings.Join(refs, ", ")
}

// RenderMarkdownWithPhases converts a TaskList to markdown format with phase headers.
// The optional phaseSource parameter, when non-nil, is used for resolving phase
// boundaries instead of tl. This is needed when tl has been filtered and boundary
// tasks (referenced by PhaseMarker.AfterTaskID) may no longer be present (T-698).
func RenderMarkdownWithPhases(tl *TaskList, phaseMarkers []PhaseMarker, phaseSource *TaskList) []byte {
	var buf strings.Builder

	// Write title as H1 header
	if tl.Title != "" {
		fmt.Fprintf(&buf, "# %s\n\n", tl.Title)
	} else {
		buf.WriteString("# \n\n")
	}

	// Determine requirements file (default if not set)
	reqFile := tl.RequirementsFile
	if reqFile == "" {
		reqFile = DefaultRequirementsFile
	}

	// Build dependency index for blocked-by title hints
	index := BuildDependencyIndex(tl.Tasks)

	ctx := &RenderContext{
		RequirementsFile: reqFile,
		DependencyIndex:  index,
	}

	// Use phaseSource for boundary resolution when the rendered list is filtered.
	resolutionList := tl
	if phaseSource != nil {
		resolutionList = phaseSource
	}

	// Build a position lookup from the resolution list.
	positionOf := make(map[string]int, len(resolutionList.Tasks))
	for i, t := range resolutionList.Tasks {
		positionOf[t.ID] = i
	}

	// Compute each marker's "start position" — the position of the first task
	// that belongs to this phase. AfterTaskID="" means start=0; AfterTaskID="X"
	// at index i means start=i+1.
	type indexedMarker struct {
		name  string
		start int
	}
	iMarkers := make([]indexedMarker, 0, len(phaseMarkers))
	for _, pm := range phaseMarkers {
		if pm.AfterTaskID == "" {
			iMarkers = append(iMarkers, indexedMarker{name: pm.Name, start: 0})
		} else if p, ok := positionOf[pm.AfterTaskID]; ok {
			iMarkers = append(iMarkers, indexedMarker{name: pm.Name, start: p + 1})
		}
	}

	// nextMarker tracks the next marker to emit.
	nextMarker := 0

	// emitHeaders writes all pending phase headers whose start position <= pos.
	emitHeaders := func(pos int) {
		for nextMarker < len(iMarkers) && iMarkers[nextMarker].start <= pos {
			// Add separator before the header only if the buffer doesn't
			// already end with a blank line (avoids triple newlines between
			// consecutive empty phase headers).
			s := buf.String()
			if !strings.HasSuffix(s, "\n\n") {
				buf.WriteString("\n")
			}
			fmt.Fprintf(&buf, "## %s\n\n", iMarkers[nextMarker].name)
			nextMarker++
		}
	}

	for i := range tl.Tasks {
		taskPos, ok := positionOf[tl.Tasks[i].ID]
		if !ok {
			if i > 0 {
				buf.WriteString("\n")
			}
			renderTask(&buf, &tl.Tasks[i], 0, ctx)
			continue
		}

		// Emit phase headers that start at or before this task's position.
		emitHeaders(taskPos)

		// Add spacing between tasks when no header was just emitted.
		s := buf.String()
		if i > 0 && !strings.HasSuffix(s, "\n\n") {
			buf.WriteString("\n")
		}

		renderTask(&buf, &tl.Tasks[i], 0, ctx)
	}

	// Emit trailing phase markers that come after all rendered tasks,
	// or all markers when the task list is empty.
	for nextMarker < len(iMarkers) {
		s := buf.String()
		if !strings.HasSuffix(s, "\n\n") {
			buf.WriteString("\n")
		}
		fmt.Fprintf(&buf, "## %s\n", iMarkers[nextMarker].name)
		nextMarker++
	}

	// Trim any extra newlines at the end but keep one
	result := buf.String()
	result = strings.TrimRight(result, "\n") + "\n"

	return []byte(result)
}

// GetTaskPhase returns the phase name for a given task ID based on position
func GetTaskPhase(tl *TaskList, phaseMarkers []PhaseMarker, taskID string) string {
	if len(phaseMarkers) == 0 {
		return ""
	}

	// Handle subtasks by getting parent task ID
	rootTaskID := taskID
	if strings.Contains(taskID, ".") {
		parts := strings.Split(taskID, ".")
		rootTaskID = parts[0]
	}

	// Find the root task position
	taskPosition := -1
	for i, task := range tl.Tasks {
		if task.ID == rootTaskID {
			taskPosition = i
			break
		}
	}

	if taskPosition == -1 {
		return ""
	}

	// Find which phase this task belongs to
	currentPhase := ""
	for _, marker := range phaseMarkers {
		if marker.AfterTaskID == "" {
			// This phase starts at the beginning
			currentPhase = marker.Name
		} else {
			// Check if we've passed this phase boundary
			boundaryPassed := false
			for i := 0; i < taskPosition; i++ {
				if tl.Tasks[i].ID == marker.AfterTaskID {
					boundaryPassed = true
					break
				}
			}
			if boundaryPassed {
				currentPhase = marker.Name
			}
		}
	}

	return currentPhase
}

// RenderJSONWithPhases converts a TaskList to JSON format with phase information.
// The optional phaseSource parameter, when non-nil, is used for resolving phase
// boundaries instead of tl. This is needed when tl has been filtered and boundary
// tasks (referenced by PhaseMarker.AfterTaskID) may no longer be present.
func RenderJSONWithPhases(tl *TaskList, phaseMarkers []PhaseMarker, phaseSource *TaskList) []byte {
	// Calculate statistics
	stats := tl.CalculateStats()

	// Build dependency index for BlockedBy translation
	index := BuildDependencyIndex(tl.Tasks)

	// Only include phase information if phases exist
	if len(phaseMarkers) == 0 {
		// No phases, use regular JSON rendering with stats
		result := TaskListJSON{
			Success:          true,
			Count:            len(tl.Tasks),
			Title:            tl.Title,
			Tasks:            toJSONTasks(tl.Tasks, index),
			Stats:            stats,
			FrontMatter:      tl.FrontMatter,
			RequirementsFile: tl.RequirementsFile,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return data
	}

	// Use phaseSource for phase resolution if provided, otherwise use tl.
	// This ensures phase boundaries are resolved against the original unfiltered
	// task list even when tl has been filtered (T-537).
	phaseResolutionList := tl
	if phaseSource != nil {
		phaseResolutionList = phaseSource
	}

	// Build tasks with phase information
	tasksWithPhases := make([]jsonTaskWithPhase, 0, len(tl.Tasks))
	for i := range tl.Tasks {
		phase := GetTaskPhase(phaseResolutionList, phaseMarkers, tl.Tasks[i].ID)
		tasksWithPhases = append(tasksWithPhases, jsonTaskWithPhase{
			jsonTask: toJSONTask(&tl.Tasks[i], index),
			Phase:    phase,
		})
	}

	result := TaskListJSONWithPhases{
		Success:      true,
		Count:        len(tasksWithPhases),
		Title:        tl.Title,
		Tasks:        tasksWithPhases,
		Stats:        stats,
		FrontMatter:  tl.FrontMatter,
		PhaseMarkers: phaseMarkers,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return data
}

// RenderJSON converts a TaskList to indented JSON format
func RenderJSON(tl *TaskList) ([]byte, error) {
	stats := tl.CalculateStats()

	// Build dependency index for BlockedBy translation
	index := BuildDependencyIndex(tl.Tasks)

	result := TaskListJSON{
		Success:          true,
		Count:            len(tl.Tasks),
		Title:            tl.Title,
		Tasks:            toJSONTasks(tl.Tasks, index),
		Stats:            stats,
		FrontMatter:      tl.FrontMatter,
		RequirementsFile: tl.RequirementsFile,
	}

	return json.MarshalIndent(result, "", "  ")
}

// FormatTaskListReferences formats TaskList-level references for display in table output
func FormatTaskListReferences(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	return strings.Join(refs, ", ")
}

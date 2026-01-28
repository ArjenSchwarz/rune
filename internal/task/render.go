package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TaskListJSON represents a TaskList with statistics for JSON output
type TaskListJSON struct {
	Success          bool         `json:"success"`
	Count            int          `json:"count"`
	Title            string       `json:"Title"`
	Tasks            []Task       `json:"Tasks"`
	Stats            Stats        `json:"Stats"`
	FrontMatter      *FrontMatter `json:"FrontMatter,omitempty"`
	RequirementsFile string       `json:"requirements_file,omitempty"`
}

// TaskListJSONWithPhases represents a TaskList with phases and statistics for JSON output
type TaskListJSONWithPhases struct {
	Success      bool            `json:"success"`
	Count        int             `json:"count"`
	Title        string          `json:"Title"`
	Tasks        []TaskWithPhase `json:"Tasks"`
	Stats        Stats           `json:"Stats"`
	FrontMatter  *FrontMatter    `json:"FrontMatter,omitempty"`
	PhaseMarkers []PhaseMarker   `json:"PhaseMarkers,omitempty"`
}

// TaskWithPhase represents a task with its phase information
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
		buf.WriteString(fmt.Sprintf("# %s\n\n", tl.Title))
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

// RenderMarkdownWithPhases converts a TaskList to markdown format with phase headers
func RenderMarkdownWithPhases(tl *TaskList, phaseMarkers []PhaseMarker) []byte {
	var buf strings.Builder

	// Write title as H1 header
	if tl.Title != "" {
		buf.WriteString(fmt.Sprintf("# %s\n\n", tl.Title))
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

	// Track which phase marker we're at
	markerIndex := 0

	// Handle phases that come before any tasks
	for markerIndex < len(phaseMarkers) && phaseMarkers[markerIndex].AfterTaskID == "" {
		buf.WriteString(fmt.Sprintf("## %s\n\n", phaseMarkers[markerIndex].Name))
		markerIndex++
	}

	// Render each root-level task with phase headers
	for i, task := range tl.Tasks {
		// Check if we need to insert phase headers after the previous task
		for markerIndex < len(phaseMarkers) {
			prevTaskID := ""
			if i > 0 {
				prevTaskID = tl.Tasks[i-1].ID
			}

			// Insert phase headers that come after the previous task
			if phaseMarkers[markerIndex].AfterTaskID == prevTaskID {
				// Add blank line before phase header if not first item
				if i > 0 {
					buf.WriteString("\n")
				}
				buf.WriteString(fmt.Sprintf("## %s\n\n", phaseMarkers[markerIndex].Name))
				markerIndex++
			} else {
				break
			}
		}

		// Add a blank line before each top-level task except the first
		// (unless we just added a phase header)
		if i > 0 && (markerIndex == 0 ||
			(markerIndex > 0 && phaseMarkers[markerIndex-1].AfterTaskID != tl.Tasks[i-1].ID)) {
			buf.WriteString("\n")
		}
		renderTask(&buf, &task, 0, ctx)
	}

	// Handle any remaining phase markers that come after all tasks
	if len(tl.Tasks) > 0 {
		lastTaskID := tl.Tasks[len(tl.Tasks)-1].ID
		for markerIndex < len(phaseMarkers) {
			if phaseMarkers[markerIndex].AfterTaskID == lastTaskID {
				buf.WriteString("\n")
				buf.WriteString(fmt.Sprintf("## %s\n", phaseMarkers[markerIndex].Name))
				markerIndex++
			} else {
				break
			}
		}
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

// RenderJSONWithPhases converts a TaskList to JSON format with phase information
func RenderJSONWithPhases(tl *TaskList, phaseMarkers []PhaseMarker) []byte {
	// Calculate statistics
	stats := tl.CalculateStats()

	// Only include phase information if phases exist
	if len(phaseMarkers) == 0 {
		// No phases, use regular JSON rendering with stats
		result := TaskListJSON{
			Success:          true,
			Count:            len(tl.Tasks),
			Title:            tl.Title,
			Tasks:            tl.Tasks,
			Stats:            stats,
			FrontMatter:      tl.FrontMatter,
			RequirementsFile: tl.RequirementsFile,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return data
	}

	// Build tasks with phase information
	tasksWithPhases := make([]TaskWithPhase, 0, len(tl.Tasks))
	for _, task := range tl.Tasks {
		phase := GetTaskPhase(tl, phaseMarkers, task.ID)
		tasksWithPhases = append(tasksWithPhases, TaskWithPhase{
			Task:  &task,
			Phase: phase,
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
	result := TaskListJSON{
		Success:          true,
		Count:            len(tl.Tasks),
		Title:            tl.Title,
		Tasks:            tl.Tasks,
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

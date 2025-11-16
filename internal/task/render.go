package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TaskListJSON represents a TaskList with statistics for JSON output
type TaskListJSON struct {
	Title            string       `json:"title"`
	Tasks            []Task       `json:"tasks"`
	Stats            Stats        `json:"stats"`
	FrontMatter      *FrontMatter `json:"front_matter,omitempty"`
	RequirementsFile string       `json:"requirements_file,omitempty"`
}

// TaskListJSONWithPhases represents a TaskList with phases and statistics for JSON output
type TaskListJSONWithPhases struct {
	Title        string          `json:"title"`
	Tasks        []TaskWithPhase `json:"tasks"`
	Stats        Stats           `json:"stats"`
	FrontMatter  *FrontMatter    `json:"front_matter,omitempty"`
	PhaseMarkers []PhaseMarker   `json:"phase_markers,omitempty"`
}

// TaskWithPhase represents a task with its phase information
type TaskWithPhase struct {
	*Task
	Phase string `json:"phase,omitempty"`
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

	// Render each root-level task
	for i, task := range tl.Tasks {
		// Add a blank line before each top-level task except the first
		if i > 0 {
			buf.WriteString("\n")
		}
		renderTask(&buf, &task, 0, reqFile)
	}

	return []byte(buf.String())
}

// renderTask recursively renders a task and its children with proper indentation
func renderTask(buf *strings.Builder, task *Task, depth int, reqFile string) {
	// Calculate indentation (2 spaces per level)
	indent := strings.Repeat("  ", depth)

	// Render the task checkbox and title
	fmt.Fprintf(buf, "%s- %s %s. %s\n",
		indent, task.Status.String(), task.ID, task.Title)

	// Render task details as bullet points
	for _, detail := range task.Details {
		fmt.Fprintf(buf, "%s  - %s\n", indent, detail)
	}

	// Render requirements if present
	if len(task.Requirements) > 0 {
		links := make([]string, len(task.Requirements))
		for i, reqID := range task.Requirements {
			links[i] = fmt.Sprintf("[%s](%s#%s)", reqID, reqFile, reqID)
		}
		fmt.Fprintf(buf, "%s  - Requirements: %s\n",
			indent, strings.Join(links, ", "))
	}

	// Render references if present
	if len(task.References) > 0 {
		fmt.Fprintf(buf, "%s  - References: %s\n",
			indent, strings.Join(task.References, ", "))
	}

	// Recursively render children
	for _, child := range task.Children {
		renderTask(buf, &child, depth+1, reqFile)
	}
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
		renderTask(&buf, &task, 0, reqFile)
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

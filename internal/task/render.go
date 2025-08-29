package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderMarkdown converts a TaskList to markdown format with consistent formatting
func RenderMarkdown(tl *TaskList) []byte {
	var buf strings.Builder

	// Write title as H1 header
	if tl.Title != "" {
		buf.WriteString(fmt.Sprintf("# %s\n\n", tl.Title))
	} else {
		buf.WriteString("# \n\n")
	}

	// Render each root-level task
	for _, task := range tl.Tasks {
		renderTask(&buf, &task, 0)
	}

	return []byte(buf.String())
}

// renderTask recursively renders a task and its children with proper indentation
func renderTask(buf *strings.Builder, task *Task, depth int) {
	// Calculate indentation (2 spaces per level)
	indent := strings.Repeat("  ", depth)

	// Render the task checkbox and title
	fmt.Fprintf(buf, "%s- %s %s. %s\n",
		indent, task.Status.String(), task.ID, task.Title)

	// Render task details as bullet points
	for _, detail := range task.Details {
		fmt.Fprintf(buf, "%s  - %s\n", indent, detail)
	}

	// Render references if present
	if len(task.References) > 0 {
		fmt.Fprintf(buf, "%s  - References: %s\n",
			indent, strings.Join(task.References, ", "))
	}

	// Recursively render children
	for _, child := range task.Children {
		renderTask(buf, &child, depth+1)
	}
}

// RenderJSON converts a TaskList to indented JSON format
func RenderJSON(tl *TaskList) ([]byte, error) {
	return json.MarshalIndent(tl, "", "  ")
}

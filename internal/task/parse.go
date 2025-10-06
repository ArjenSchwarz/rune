package task

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// MaxFileSize is the maximum allowed size for task markdown files (10MB)
const MaxFileSize = 10 * 1024 * 1024

var (
	taskLinePattern    = regexp.MustCompile(`^(\s*)- (\[[ \-xX]\]) (\d+(?:\.\d+)*)\. (.+)$`)
	detailLinePattern  = regexp.MustCompile(`^(\s*)- (.+)$`)
	phaseHeaderPattern = regexp.MustCompile(`^## (.+)$`)
)

// ParseMarkdown parses markdown content into a TaskList structure
func ParseMarkdown(content []byte) (*TaskList, error) {
	if len(content) > MaxFileSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", MaxFileSize)
	}
	return parseContent(string(content))
}

// ParseFile reads and parses a markdown file
func ParseFile(filepath string) (*TaskList, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	taskList, err := ParseMarkdown(content)
	if err != nil {
		return nil, err
	}
	taskList.FilePath = filepath
	return taskList, nil
}

// ParseFileWithPhases reads and parses a markdown file, returning both the TaskList and phase markers
func ParseFileWithPhases(filepath string) (*TaskList, []PhaseMarker, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse the task list
	taskList, err := ParseMarkdown(content)
	if err != nil {
		return nil, nil, err
	}
	taskList.FilePath = filepath

	// Extract phase markers from the content
	lines := strings.Split(string(content), "\n")
	// Skip front matter if present
	if strings.HasPrefix(strings.TrimSpace(string(content)), "---") {
		inFrontMatter := false
		frontMatterCount := 0
		newLines := []string{}
		for _, line := range lines {
			if strings.TrimSpace(line) == "---" {
				frontMatterCount++
				if frontMatterCount == 2 {
					inFrontMatter = false
					continue
				} else {
					inFrontMatter = true
					continue
				}
			}
			if !inFrontMatter && frontMatterCount > 0 {
				newLines = append(newLines, line)
			}
		}
		if frontMatterCount >= 2 {
			lines = newLines
		}
	}

	phaseMarkers := ExtractPhaseMarkers(lines)

	return taskList, phaseMarkers, nil
}

func parseContent(content string) (*TaskList, error) {
	taskList := &TaskList{}

	// Extract front matter first if present
	frontMatter, remainingContent, err := ParseFrontMatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing front matter: %w", err)
	}
	taskList.FrontMatter = frontMatter

	// Now parse the remaining content
	lines := strings.Split(remainingContent, "\n")

	// Clean up lines - handle different line endings
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}

	// Extract title if present
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			taskList.Title = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			// Remove title line and continue parsing from next line
			lines = append(lines[:i], lines[i+1:]...)
			break
		}
	}

	// Parse tasks starting from root level
	tasks, _, err := parseTasksAtLevel(lines, 0, 0, "")
	if err != nil {
		return nil, err
	}

	taskList.Tasks = tasks
	taskList.FilePath = "" // Will be set by ParseFile if used
	return taskList, nil
}

func parseTasksAtLevel(lines []string, startIdx, expectedIndent int, parentID string) ([]Task, int, error) {
	var tasks []Task

	for i := startIdx; i < len(lines); i++ {
		// Skip empty lines
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}

		indent := countIndent(lines[i])

		// Check for tabs (invalid indentation)
		if indent == -1 {
			return nil, i, fmt.Errorf("line %d: unexpected indentation (tabs not allowed)", i+1)
		}

		// If indentation is less than expected, we've reached the end of this level
		if indent < expectedIndent {
			return tasks, i - 1, nil
		}

		// Check for unexpected indentation (must be exactly 2 spaces per level)
		if indent > expectedIndent && indent != expectedIndent+2 {
			return nil, i, fmt.Errorf("line %d: unexpected indentation", i+1)
		}

		// Try to parse as a task line
		task, ok, err := parseTaskLineWithError(lines[i])
		switch {
		case err != nil:
			return nil, i, fmt.Errorf("line %d: %w", i+1, err)
		case ok:
			// Verify indentation matches expected level
			if indent != expectedIndent {
				return nil, i, fmt.Errorf("line %d: unexpected indentation", i+1)
			}

			// Assign ID based on position
			if parentID == "" {
				task.ID = fmt.Sprintf("%d", len(tasks)+1)
			} else {
				task.ID = fmt.Sprintf("%s.%d", parentID, len(tasks)+1)
				task.ParentID = parentID
			}

			// Look ahead for details and subtasks
			detailsAndChildren, newIdx, err := parseDetailsAndChildren(lines, i+1, expectedIndent+2, task.ID)
			if err != nil {
				return nil, newIdx, err
			}

			// Separate details from children
			for _, item := range detailsAndChildren {
				switch v := item.(type) {
				case Task:
					task.Children = append(task.Children, v)
				case string:
					// Check if it's a reference line
					if refs, ok := strings.CutPrefix(v, "References: "); ok {
						task.References = parseReferences(refs)
					} else {
						task.Details = append(task.Details, v)
					}
				}
			}

			tasks = append(tasks, task)
			i = newIdx
		case indent == expectedIndent:
			// Check if this is a phase header (H2) - skip it
			if phaseHeaderPattern.MatchString(strings.TrimSpace(lines[i])) && expectedIndent == 0 {
				// Phase headers are allowed at root level
				continue
			}
			// This is a detail line at the wrong level
			return nil, i, fmt.Errorf("line %d: unexpected content at this indentation level", i+1)
		default:
			// Skip lines that don't match task pattern but have deeper indentation
			// These will be caught as unexpected indentation if they're in the wrong place
			continue
		}
	}

	return tasks, len(lines) - 1, nil
}

func parseDetailsAndChildren(lines []string, startIdx, expectedIndent int, parentID string) ([]any, int, error) {
	var items []any

	for i := startIdx; i < len(lines); i++ {
		// Skip empty lines
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}

		indent := countIndent(lines[i])

		// Check for tabs (invalid indentation)
		if indent == -1 {
			return nil, i, fmt.Errorf("line %d: unexpected indentation (tabs not allowed)", i+1)
		}

		// If indentation is less than expected, we're done with this section
		if indent < expectedIndent {
			return items, i - 1, nil
		}

		// Check for unexpected indentation
		if indent > expectedIndent && indent != expectedIndent+2 {
			return nil, i, fmt.Errorf("line %d: unexpected indentation", i+1)
		}

		// Try to parse as a task
		if _, ok := parseTaskLine(lines[i]); ok {
			if indent != expectedIndent {
				return nil, i, fmt.Errorf("line %d: unexpected indentation", i+1)
			}

			// This is a subtask
			children, newIdx, err := parseTasksAtLevel(lines, i, expectedIndent, parentID)
			if err != nil {
				return nil, newIdx, err
			}

			// Add all children tasks
			for _, child := range children {
				items = append(items, child)
			}

			return items, newIdx, nil
		} else if indent == expectedIndent {
			// This is a detail line
			if detail := parseDetailLine(lines[i]); detail != "" {
				items = append(items, detail)
			}
		} else {
			// Deeper indentation without being a task or detail
			return nil, i, fmt.Errorf("line %d: unexpected indentation", i+1)
		}
	}

	return items, len(lines) - 1, nil
}

func parseTaskLineWithError(line string) (Task, bool, error) {
	trimmed := strings.TrimSpace(line)

	// Check if it looks like a task but with invalid status
	if strings.HasPrefix(trimmed, "- [") && strings.Contains(trimmed, "]") {
		matches := taskLinePattern.FindStringSubmatch(line)
		if matches == nil || len(matches) != 5 {
			// Extract the checkbox part to check status
			checkboxEnd := strings.Index(trimmed, "]")
			if checkboxEnd > 2 {
				checkbox := trimmed[2 : checkboxEnd+1]
				if checkbox != checkboxPending && checkbox != checkboxInProgress && checkbox != checkboxCompleted && checkbox != "[X]" {
					return Task{}, false, fmt.Errorf("invalid status: %s", checkbox)
				}
			}
			// Check if it's missing the number
			if !regexp.MustCompile(`^\s*- \[[ \-xX]\] \d+`).MatchString(line) {
				if regexp.MustCompile(`^\s*- \[[ \-xX]\][^\d]`).MatchString(line) {
					return Task{}, false, fmt.Errorf("invalid task format: missing task number")
				}
				if regexp.MustCompile(`^\s*- \[[ \-xX]\]\d+`).MatchString(line) {
					return Task{}, false, fmt.Errorf("invalid task format: missing space after checkbox")
				}
			}
			return Task{}, false, fmt.Errorf("invalid task format")
		}

		// Parse the status
		status, err := ParseStatus(matches[2])
		if err != nil {
			return Task{}, false, err
		}

		return Task{
			Title:  matches[4],
			Status: status,
		}, true, nil
	}

	// Check if it looks like a malformed checkbox
	if strings.HasPrefix(trimmed, "- []") || strings.HasPrefix(trimmed, "-[]") {
		return Task{}, false, fmt.Errorf("invalid task format: missing space in checkbox")
	}

	// Not a task line at all
	return Task{}, false, nil
}

func parseTaskLine(line string) (Task, bool) {
	task, ok, _ := parseTaskLineWithError(line)
	return task, ok
}

func parseDetailLine(line string) string {
	matches := detailLinePattern.FindStringSubmatch(line)
	if len(matches) == 3 {
		return matches[2]
	}
	return ""
}

func parseReferences(refs string) []string {
	parts := strings.Split(refs, ",")
	references := make([]string, 0, len(parts))

	for _, part := range parts {
		ref := strings.TrimSpace(part)
		if ref != "" {
			references = append(references, ref)
		}
	}

	return references
}

func countIndent(line string) int {
	count := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			count++
		case '\t':
			// Tab counts as wrong indentation - we only accept spaces
			return -1
		default:
			return count
		}
	}
	return count
}

// ExtractPhaseMarkers scans lines for H2 headers and returns phase markers with their positions
func ExtractPhaseMarkers(lines []string) []PhaseMarker {
	markers := []PhaseMarker{}
	var lastTaskID string

	for _, line := range lines {
		// Phase headers must start at the beginning of the line (no indentation)
		// Check if line is a phase header (H2) - use original line, not trimmed
		if matches := phaseHeaderPattern.FindStringSubmatch(line); matches != nil {
			phaseName := strings.TrimSpace(matches[1])
			markers = append(markers, PhaseMarker{
				Name:        phaseName,
				AfterTaskID: lastTaskID,
			})
		} else if _, ok := parseTaskLine(line); ok {
			// Extract task ID from the line
			// The task ID is captured in the regex pattern
			if taskMatches := taskLinePattern.FindStringSubmatch(line); len(taskMatches) >= 4 {
				// Only update lastTaskID for top-level tasks (not subtasks)
				// Top-level tasks don't have dots in their IDs
				taskID := taskMatches[3]
				if !strings.Contains(taskID, ".") {
					lastTaskID = taskID
				}
			}
		}
	}

	return markers
}

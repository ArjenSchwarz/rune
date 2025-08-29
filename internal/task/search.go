package task

import (
	"strings"
)

// QueryOptions configures search behavior
type QueryOptions struct {
	CaseSensitive bool // Whether to match case exactly
	SearchDetails bool // Search in task details
	SearchRefs    bool // Search in task references
	IncludeParent bool // Include parent context in results
}

// QueryFilter configures filtering criteria
type QueryFilter struct {
	Status       *Status // Filter by task status (nil means any status)
	MaxDepth     int     // Maximum hierarchy depth (0 means no limit)
	ParentID     *string // Filter by parent task ID (nil means no filter, "" means top-level)
	TitlePattern string  // Pattern to match in titles
}

// Find searches for tasks matching the given pattern
func (tl *TaskList) Find(pattern string, opts QueryOptions) []Task {
	var results []Task

	// Prepare pattern for case-insensitive search if needed
	searchPattern := pattern
	if !opts.CaseSensitive {
		searchPattern = strings.ToLower(pattern)
	}

	// Search through all tasks recursively
	tl.findInTasks(tl.Tasks, searchPattern, opts, &results)

	return results
}

// findInTasks recursively searches tasks and adds matches to results
func (tl *TaskList) findInTasks(tasks []Task, pattern string, opts QueryOptions, results *[]Task) {
	for _, task := range tasks {
		found := false

		// Search in title
		title := task.Title
		if !opts.CaseSensitive {
			title = strings.ToLower(title)
		}
		if strings.Contains(title, pattern) {
			found = true
		}

		// Search in details if requested
		if !found && opts.SearchDetails {
			for _, detail := range task.Details {
				searchDetail := detail
				if !opts.CaseSensitive {
					searchDetail = strings.ToLower(detail)
				}
				if strings.Contains(searchDetail, pattern) {
					found = true
					break
				}
			}
		}

		// Search in references if requested
		if !found && opts.SearchRefs {
			for _, ref := range task.References {
				searchRef := ref
				if !opts.CaseSensitive {
					searchRef = strings.ToLower(ref)
				}
				if strings.Contains(searchRef, pattern) {
					found = true
					break
				}
			}
		}

		if found {
			*results = append(*results, task)
		}

		// Search in children recursively
		if len(task.Children) > 0 {
			tl.findInTasks(task.Children, pattern, opts, results)
		}
	}
}

// Filter returns tasks matching the given filter criteria
func (tl *TaskList) Filter(filter QueryFilter) []Task {
	var results []Task

	// Start filtering from root tasks
	tl.filterTasks(tl.Tasks, filter, 1, "", &results)

	return results
}

// filterTasks recursively filters tasks based on criteria
func (tl *TaskList) filterTasks(tasks []Task, filter QueryFilter, currentDepth int, parentID string, results *[]Task) {
	for i := range tasks {
		task := &tasks[i]
		include := true

		// Check status filter
		if filter.Status != nil && task.Status != *filter.Status {
			include = false
		}

		// Check max depth filter
		if filter.MaxDepth > 0 && currentDepth > filter.MaxDepth {
			include = false
		}

		// Check parent ID filter
		if filter.ParentID != nil {
			// ParentID filter is explicitly set
			if task.ParentID != *filter.ParentID {
				include = false
			}
		}

		// Check title pattern filter
		if filter.TitlePattern != "" {
			if !strings.Contains(strings.ToLower(task.Title), strings.ToLower(filter.TitlePattern)) {
				include = false
			}
		}

		if include {
			// Create a copy of the task to avoid modifying the original
			resultTask := Task{
				ID:         task.ID,
				Title:      task.Title,
				Status:     task.Status,
				Details:    task.Details,
				References: task.References,
				Children:   task.Children,
				ParentID:   task.ParentID,
			}
			*results = append(*results, resultTask)
		}

		// Recursively filter children
		if len(task.Children) > 0 {
			tl.filterTasks(task.Children, filter, currentDepth+1, task.ID, results)
		}
	}
}

// getTaskDepth returns the depth of a task based on its ID
func getTaskDepth(taskID string) int {
	if taskID == "" {
		return 0
	}
	return strings.Count(taskID, ".") + 1
}

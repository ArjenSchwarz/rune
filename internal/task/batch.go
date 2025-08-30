package task

import (
	"fmt"
	"strings"
)

// Operation represents a single operation in a batch
type Operation struct {
	Type       string   `json:"type"`
	ID         string   `json:"id,omitempty"`
	Parent     string   `json:"parent,omitempty"`
	Title      string   `json:"title,omitempty"`
	Status     Status   `json:"status,omitempty"`
	Details    []string `json:"details,omitempty"`
	References []string `json:"references,omitempty"`
}

// BatchRequest represents a request for multiple operations
type BatchRequest struct {
	File       string      `json:"file"`
	Operations []Operation `json:"operations"`
	DryRun     bool        `json:"dry_run"`
}

// BatchResponse represents the response from a batch operation
type BatchResponse struct {
	Success bool     `json:"success"`
	Applied int      `json:"applied"`
	Errors  []string `json:"errors,omitempty"`
	Preview string   `json:"preview,omitempty"`
}

// ExecuteBatch validates and executes a batch of operations atomically
func (tl *TaskList) ExecuteBatch(ops []Operation, dryRun bool) (*BatchResponse, error) {
	response := &BatchResponse{
		Success: true,
		Applied: 0,
		Errors:  []string{},
	}

	// Create a deep copy to test all operations first (per Decision #12)
	testList, err := tl.deepCopy()
	if err != nil {
		return nil, fmt.Errorf("creating test copy: %w", err)
	}

	// Validate and apply operations to test copy sequentially
	for i, op := range ops {
		if err := validateOperation(testList, op); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil
		}

		// Apply operation to test copy for subsequent validations
		if err := applyOperation(testList, op); err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("operation %d: %v", i+1, err))
			return response, nil
		}
	}

	// For dry run, return preview without applying to original
	if dryRun {
		response.Preview = string(RenderMarkdown(testList))
		response.Applied = len(ops)
		return response, nil
	}

	// Apply all operations to original (atomic - all succeed or all fail)
	for _, op := range ops {
		if err := applyOperation(tl, op); err != nil {
			return nil, fmt.Errorf("applying operation: %w", err)
		}
		response.Applied++
	}

	return response, nil
}

// validateOperation checks if an operation is valid without applying it
func validateOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case "add":
		if op.Title == "" {
			return fmt.Errorf("add operation requires title")
		}
		if len(op.Title) > 500 {
			return fmt.Errorf("title exceeds 500 characters")
		}
		if op.Parent != "" && tl.FindTask(op.Parent) == nil {
			return fmt.Errorf("parent task %s not found", op.Parent)
		}
	case "remove":
		if op.ID == "" {
			return fmt.Errorf("remove operation requires id")
		}
		if tl.FindTask(op.ID) == nil {
			return fmt.Errorf("task %s not found", op.ID)
		}
	case "update_status":
		if op.ID == "" {
			return fmt.Errorf("update_status operation requires id")
		}
		if tl.FindTask(op.ID) == nil {
			return fmt.Errorf("task %s not found", op.ID)
		}
		if op.Status < Pending || op.Status > Completed {
			return fmt.Errorf("invalid status value: %d", op.Status)
		}
	case "update":
		if op.ID == "" {
			return fmt.Errorf("update operation requires id")
		}
		if tl.FindTask(op.ID) == nil {
			return fmt.Errorf("task %s not found", op.ID)
		}
		if op.Title != "" && len(op.Title) > 500 {
			return fmt.Errorf("title exceeds 500 characters")
		}
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
	return nil
}

// applyOperation executes a single operation
func applyOperation(tl *TaskList, op Operation) error {
	switch strings.ToLower(op.Type) {
	case "add":
		// First add the task
		if err := tl.AddTask(op.Parent, op.Title); err != nil {
			return err
		}
		// If details or references are provided, update the newly added task
		if len(op.Details) > 0 || len(op.References) > 0 {
			// Find the newly added task ID
			var newTaskID string
			if op.Parent != "" {
				parent := tl.FindTask(op.Parent)
				if parent != nil && len(parent.Children) > 0 {
					newTaskID = parent.Children[len(parent.Children)-1].ID
				}
			} else if len(tl.Tasks) > 0 {
				newTaskID = tl.Tasks[len(tl.Tasks)-1].ID
			}
			if newTaskID != "" {
				return tl.UpdateTask(newTaskID, "", op.Details, op.References)
			}
		}
		return nil
	case "remove":
		return tl.RemoveTask(op.ID)
	case "update_status":
		return tl.UpdateStatus(op.ID, op.Status)
	case "update":
		return tl.UpdateTask(op.ID, op.Title, op.Details, op.References)
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// deepCopy creates a deep copy of the TaskList for dry-run operations
func (tl *TaskList) deepCopy() (*TaskList, error) {
	// Simple approach: render to markdown and parse back
	content := RenderMarkdown(tl)
	copy, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("creating deep copy: %w", err)
	}
	copy.FilePath = tl.FilePath
	return copy, nil
}

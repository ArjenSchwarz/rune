package task

import (
	"fmt"
	"testing"
	"time"
)

func TestAddTask(t *testing.T) {
	t.Run("add root task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}
		err := tl.AddTask("", "First task")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		if len(tl.Tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(tl.Tasks))
		}

		task := tl.Tasks[0]
		if task.ID != "1" {
			t.Errorf("expected ID '1', got %s", task.ID)
		}
		if task.Title != "First task" {
			t.Errorf("expected title 'First task', got %s", task.Title)
		}
		if task.Status != Pending {
			t.Errorf("expected status Pending, got %v", task.Status)
		}
		if task.ParentID != "" {
			t.Errorf("expected empty ParentID, got %s", task.ParentID)
		}
	})

	t.Run("add multiple root tasks", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		for i := 1; i <= 3; i++ {
			err := tl.AddTask("", fmt.Sprintf("Task %d", i))
			if err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}
		}

		if len(tl.Tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tl.Tasks))
		}

		for i, task := range tl.Tasks {
			expectedID := fmt.Sprintf("%d", i+1)
			if task.ID != expectedID {
				t.Errorf("task %d: expected ID %s, got %s", i, expectedID, task.ID)
			}
		}
	})

	t.Run("add subtask to existing task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Parent task")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1", "Child task")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		parent := tl.FindTask("1")
		if parent == nil {
			t.Fatal("parent task not found")
		}

		if len(parent.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(parent.Children))
		}

		child := parent.Children[0]
		if child.ID != "1.1" {
			t.Errorf("expected child ID '1.1', got %s", child.ID)
		}
		if child.Title != "Child task" {
			t.Errorf("expected title 'Child task', got %s", child.Title)
		}
		if child.ParentID != "1" {
			t.Errorf("expected ParentID '1', got %s", child.ParentID)
		}
	})

	t.Run("add deep nested subtask", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Level 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1", "Level 2")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1.1", "Level 3")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		level3 := tl.FindTask("1.1.1")
		if level3 == nil {
			t.Fatal("level 3 task not found")
		}
		if level3.ID != "1.1.1" {
			t.Errorf("expected ID '1.1.1', got %s", level3.ID)
		}
		if level3.Title != "Level 3" {
			t.Errorf("expected title 'Level 3', got %s", level3.Title)
		}
	})

	t.Run("add task to non-existent parent", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("99", "Orphan task")
		if err == nil {
			t.Error("expected error for non-existent parent, got nil")
		}
		if err.Error() != "parent task 99 not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("modified timestamp is updated", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}
		before := tl.Modified

		time.Sleep(10 * time.Millisecond)

		err := tl.AddTask("", "New task")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		if !tl.Modified.After(before) {
			t.Error("Modified timestamp was not updated")
		}
	})
}

func TestRemoveTask(t *testing.T) {
	t.Run("remove root task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		for i := 1; i <= 3; i++ {
			err := tl.AddTask("", fmt.Sprintf("Task %d", i))
			if err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}
		}

		err := tl.RemoveTask("2")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		if len(tl.Tasks) != 2 {
			t.Errorf("expected 2 tasks after removal, got %d", len(tl.Tasks))
		}

		if tl.Tasks[0].Title != "Task 1" {
			t.Errorf("first task should be 'Task 1', got %s", tl.Tasks[0].Title)
		}
		if tl.Tasks[1].Title != "Task 3" {
			t.Errorf("second task should be 'Task 3', got %s", tl.Tasks[1].Title)
		}
	})

	t.Run("remove task with automatic renumbering", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		for i := 1; i <= 5; i++ {
			err := tl.AddTask("", fmt.Sprintf("Task %d", i))
			if err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}
		}

		err := tl.RemoveTask("3")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		expectedIDs := []string{"1", "2", "3", "4"}
		expectedTitles := []string{"Task 1", "Task 2", "Task 4", "Task 5"}

		for i, task := range tl.Tasks {
			if task.ID != expectedIDs[i] {
				t.Errorf("task %d: expected ID %s, got %s", i, expectedIDs[i], task.ID)
			}
			if task.Title != expectedTitles[i] {
				t.Errorf("task %d: expected title %s, got %s", i, expectedTitles[i], task.Title)
			}
		}
	})

	t.Run("remove subtask with renumbering", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Parent")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		for i := 1; i <= 4; i++ {
			err := tl.AddTask("1", fmt.Sprintf("Child %d", i))
			if err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}
		}

		err = tl.RemoveTask("1.2")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		parent := tl.FindTask("1")
		if parent == nil {
			t.Fatal("parent task not found")
		}

		if len(parent.Children) != 3 {
			t.Errorf("expected 3 children after removal, got %d", len(parent.Children))
		}

		expectedIDs := []string{"1.1", "1.2", "1.3"}
		expectedTitles := []string{"Child 1", "Child 3", "Child 4"}

		for i, child := range parent.Children {
			if child.ID != expectedIDs[i] {
				t.Errorf("child %d: expected ID %s, got %s", i, expectedIDs[i], child.ID)
			}
			if child.Title != expectedTitles[i] {
				t.Errorf("child %d: expected title %s, got %s", i, expectedTitles[i], child.Title)
			}
			if child.ParentID != "1" {
				t.Errorf("child %d: expected ParentID '1', got %s", i, child.ParentID)
			}
		}
	})

	t.Run("remove task with children", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Parent")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1", "Child 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1.1", "Grandchild")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.RemoveTask("1")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		if len(tl.Tasks) != 0 {
			t.Errorf("expected 0 tasks after removing parent, got %d", len(tl.Tasks))
		}
	})

	t.Run("remove non-existent task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.RemoveTask("99")
		if err == nil {
			t.Error("expected error for non-existent task, got nil")
		}
		if err.Error() != "task 99 not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("modified timestamp is updated", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}
		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		before := tl.Modified
		time.Sleep(10 * time.Millisecond)

		err = tl.RemoveTask("1")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		if !tl.Modified.After(before) {
			t.Error("Modified timestamp was not updated")
		}
	})
}

func TestUpdateStatus(t *testing.T) {
	t.Run("update status to completed", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.UpdateStatus("1", Completed)
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		task := tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Status != Completed {
			t.Errorf("expected status Completed, got %v", task.Status)
		}
	})

	t.Run("update status to in-progress", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.UpdateStatus("1", InProgress)
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		task := tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Status != InProgress {
			t.Errorf("expected status InProgress, got %v", task.Status)
		}
	})

	t.Run("update status transitions", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		transitions := []struct {
			status   Status
			expected Status
		}{
			{InProgress, InProgress},
			{Completed, Completed},
			{Pending, Pending},
			{InProgress, InProgress},
		}

		for _, tt := range transitions {
			err = tl.UpdateStatus("1", tt.status)
			if err != nil {
				t.Fatalf("UpdateStatus failed: %v", err)
			}

			task := tl.FindTask("1")
			if task == nil {
				t.Fatal("task not found")
			}

			if task.Status != tt.expected {
				t.Errorf("expected status %v, got %v", tt.expected, task.Status)
			}
		}
	})

	t.Run("update status of subtask", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Parent")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1", "Child")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.UpdateStatus("1.1", Completed)
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		child := tl.FindTask("1.1")
		if child == nil {
			t.Fatal("child task not found")
		}

		if child.Status != Completed {
			t.Errorf("expected status Completed, got %v", child.Status)
		}
	})

	t.Run("update status of non-existent task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.UpdateStatus("99", Completed)
		if err == nil {
			t.Error("expected error for non-existent task, got nil")
		}
		if err.Error() != "task 99 not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("modified timestamp is updated", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}
		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		before := tl.Modified
		time.Sleep(10 * time.Millisecond)

		err = tl.UpdateStatus("1", Completed)
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		if !tl.Modified.After(before) {
			t.Error("Modified timestamp was not updated")
		}
	})
}

func TestUpdateTask(t *testing.T) {
	t.Run("update title only", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Original title")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.UpdateTask("1", "New title", nil, nil)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		task := tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Title != "New title" {
			t.Errorf("expected title 'New title', got %s", task.Title)
		}
	})

	t.Run("update details only", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		details := []string{"Detail 1", "Detail 2", "Detail 3"}
		err = tl.UpdateTask("1", "", details, nil)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		task := tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if len(task.Details) != 3 {
			t.Errorf("expected 3 details, got %d", len(task.Details))
		}

		for i, detail := range task.Details {
			if detail != details[i] {
				t.Errorf("detail %d: expected %s, got %s", i, details[i], detail)
			}
		}
	})

	t.Run("update references only", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		refs := []string{"ref1.md", "ref2.md"}
		err = tl.UpdateTask("1", "", nil, refs)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		task := tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if len(task.References) != 2 {
			t.Errorf("expected 2 references, got %d", len(task.References))
		}

		for i, ref := range task.References {
			if ref != refs[i] {
				t.Errorf("reference %d: expected %s, got %s", i, refs[i], ref)
			}
		}
	})

	t.Run("update all fields", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Original")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		details := []string{"New detail"}
		refs := []string{"new-ref.md"}
		err = tl.UpdateTask("1", "Updated", details, refs)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		task := tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Title != "Updated" {
			t.Errorf("expected title 'Updated', got %s", task.Title)
		}
		if len(task.Details) != 1 || task.Details[0] != "New detail" {
			t.Errorf("expected details ['New detail'], got %v", task.Details)
		}
		if len(task.References) != 1 || task.References[0] != "new-ref.md" {
			t.Errorf("expected references ['new-ref.md'], got %v", task.References)
		}
	})

	t.Run("clear details and references", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		task := tl.FindTask("1")
		task.Details = []string{"Old detail"}
		task.References = []string{"old-ref.md"}

		emptySlice := []string{}
		err = tl.UpdateTask("1", "", emptySlice, emptySlice)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		task = tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if len(task.Details) != 0 {
			t.Errorf("expected empty details, got %v", task.Details)
		}
		if len(task.References) != 0 {
			t.Errorf("expected empty references, got %v", task.References)
		}
	})

	t.Run("update non-existent task", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.UpdateTask("99", "New title", nil, nil)
		if err == nil {
			t.Error("expected error for non-existent task, got nil")
		}
		if err.Error() != "task 99 not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("preserve existing values when not updating", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Original title")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		task := tl.FindTask("1")
		task.Details = []string{"Existing detail"}
		task.References = []string{"existing.md"}

		err = tl.UpdateTask("1", "", nil, nil)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		task = tl.FindTask("1")
		if task == nil {
			t.Fatal("task not found")
		}

		if task.Title != "Original title" {
			t.Errorf("title should not change, got %s", task.Title)
		}
		if len(task.Details) != 1 || task.Details[0] != "Existing detail" {
			t.Errorf("details should not change, got %v", task.Details)
		}
		if len(task.References) != 1 || task.References[0] != "existing.md" {
			t.Errorf("references should not change, got %v", task.References)
		}
	})

	t.Run("modified timestamp is updated", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}
		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		before := tl.Modified
		time.Sleep(10 * time.Millisecond)

		err = tl.UpdateTask("1", "Updated", nil, nil)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		if !tl.Modified.After(before) {
			t.Error("Modified timestamp was not updated")
		}
	})
}

func TestParentChildRelationshipIntegrity(t *testing.T) {
	t.Run("maintain parent-child relationships during operations", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Parent 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1", "Child 1.1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1", "Child 1.2")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("1.1", "Grandchild 1.1.1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		grandchild := tl.FindTask("1.1.1")
		if grandchild == nil {
			t.Fatal("grandchild not found")
		}
		if grandchild.ParentID != "1.1" {
			t.Errorf("expected ParentID '1.1', got %s", grandchild.ParentID)
		}

		err = tl.RemoveTask("1.2")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		parent := tl.FindTask("1")
		if parent == nil {
			t.Fatal("parent not found")
		}
		if len(parent.Children) != 1 {
			t.Errorf("expected 1 child after removal, got %d", len(parent.Children))
		}

		child := parent.Children[0]
		if child.ID != "1.1" {
			t.Errorf("remaining child should have ID '1.1', got %s", child.ID)
		}
		if child.ParentID != "1" {
			t.Errorf("remaining child should have ParentID '1', got %s", child.ParentID)
		}
	})

	t.Run("deep hierarchy renumbering", func(t *testing.T) {
		tl := &TaskList{Title: "Test Tasks"}

		err := tl.AddTask("", "Task 1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
		err = tl.AddTask("", "Task 2")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("2", "Task 2.1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
		err = tl.AddTask("2", "Task 2.2")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.AddTask("2.1", "Task 2.1.1")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
		err = tl.AddTask("2.1", "Task 2.1.2")
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		err = tl.RemoveTask("1")
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		newRoot := tl.FindTask("1")
		if newRoot == nil {
			t.Fatal("renumbered root task not found")
		}
		if newRoot.Title != "Task 2" {
			t.Errorf("expected 'Task 2' to become new task 1, got %s", newRoot.Title)
		}

		child := tl.FindTask("1.1")
		if child == nil {
			t.Fatal("renumbered child not found")
		}
		if child.Title != "Task 2.1" {
			t.Errorf("expected 'Task 2.1' to become '1.1', got %s", child.Title)
		}
		if child.ParentID != "1" {
			t.Errorf("expected ParentID '1', got %s", child.ParentID)
		}

		grandchild1 := tl.FindTask("1.1.1")
		if grandchild1 == nil {
			t.Fatal("renumbered grandchild 1 not found")
		}
		if grandchild1.Title != "Task 2.1.1" {
			t.Errorf("expected 'Task 2.1.1' to become '1.1.1', got %s", grandchild1.Title)
		}
		if grandchild1.ParentID != "1.1" {
			t.Errorf("expected ParentID '1.1', got %s", grandchild1.ParentID)
		}

		grandchild2 := tl.FindTask("1.1.2")
		if grandchild2 == nil {
			t.Fatal("renumbered grandchild 2 not found")
		}
		if grandchild2.Title != "Task 2.1.2" {
			t.Errorf("expected 'Task 2.1.2' to become '1.1.2', got %s", grandchild2.Title)
		}
		if grandchild2.ParentID != "1.1" {
			t.Errorf("expected ParentID '1.1', got %s", grandchild2.ParentID)
		}
	})
}

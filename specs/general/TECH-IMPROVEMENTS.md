# Technical Improvements

This document tracks efficiency issues and optimization opportunities identified during code reviews.

## 2025-09-01 - Position-Based Task Insertion Efficiency Review

### Issue: Inefficient Array Insertion Pattern
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/operations.go` (lines 106-107, 122-123)
**Description**: The `addTaskAtPosition` function uses a double-append pattern for array insertion that creates unnecessary intermediate slices and copies data multiple times. The current implementation:
```go
parent.Children = append(parent.Children[:targetIndex],
    append([]Task{newTask}, parent.Children[targetIndex:]...)...)
```
**Impact**: O(n) space complexity and O(n) time complexity with unnecessary allocations and memory copies, especially problematic for large task lists.
**Solution**:
```go
// Pre-allocate slice with final capacity
newChildren := make([]Task, len(parent.Children)+1)
// Copy prefix in one operation
copy(newChildren, parent.Children[:targetIndex])
// Insert new task
newChildren[targetIndex] = newTask
// Copy suffix in one operation  
copy(newChildren[targetIndex+1:], parent.Children[targetIndex:])
parent.Children = newChildren
```
**Trade-offs**: Slightly more verbose code but significantly better performance and memory efficiency.

---

### Issue: Redundant Full Renumbering After Insertion
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/operations.go` (lines 127, 155-168)
**Description**: After every position-based insertion, `renumberTasks()` is called which recursively renumbers ALL tasks in the entire hierarchy, even when only tasks at the insertion point and after need renumbering.
**Impact**: O(n) operation for every insertion where n is the total number of tasks in the hierarchy. For batch operations or frequent insertions, this becomes O(n×m) where m is the number of insertions.
**Solution**:
```go
// Add selective renumbering method
func (tl *TaskList) renumberFromPosition(parentID string, startIndex int) {
    if parentID == "" {
        // Renumber root tasks from startIndex
        for i := startIndex; i < len(tl.Tasks); i++ {
            tl.Tasks[i].ID = fmt.Sprintf("%d", i+1)
            renumberChildren(&tl.Tasks[i])
        }
    } else {
        parent := tl.FindTask(parentID)
        if parent != nil {
            // Renumber children from startIndex
            for i := startIndex; i < len(parent.Children); i++ {
                parent.Children[i].ID = fmt.Sprintf("%s.%d", parentID, i+1)
                parent.Children[i].ParentID = parentID
                renumberChildren(&parent.Children[i])
            }
        }
    }
}
```
**Trade-offs**: More complex code but dramatically better performance, especially for batch operations.

---

### Issue: Manual String-to-Integer Parsing
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/operations.go` (lines 74-82)
**Description**: The position parsing uses manual character-by-character parsing instead of Go's optimized `strconv.Atoi()`.
**Impact**: Slightly slower parsing and more complex, error-prone code.
**Solution**:
```go
import "strconv"

// Parse position to get insertion index
parts := strings.Split(position, ".")
lastPart := parts[len(parts)-1]
targetIndex, err := strconv.Atoi(lastPart)
if err != nil || targetIndex < 1 {
    return fmt.Errorf("invalid position format: %s", position)
}
targetIndex-- // Convert to 0-based index
```
**Trade-offs**: Simpler code with better performance and error handling.

---

### Issue: Inefficient Task Counting for Resource Limits
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/operations.go` (lines 371-387)
**Description**: `countTotalTasks()` performs a full recursive traversal on every task addition, which is O(n) where n is the total number of tasks.
**Impact**: Makes task addition O(n) instead of O(1), particularly problematic for batch operations.
**Solution**:
```go
// Add task count field to TaskList
type TaskList struct {
    Title       string
    Tasks       []Task
    FrontMatter *FrontMatter
    FilePath    string
    Modified    time.Time
    TaskCount   int // New field to track total tasks
}

// Update count incrementally instead of full recalculation
func (tl *TaskList) incrementTaskCount() {
    tl.TaskCount++
}

func (tl *TaskList) decrementTaskCount() {
    if tl.TaskCount > 0 {
        tl.TaskCount--
    }
}
```
**Trade-offs**: Additional field to maintain but eliminates O(n) operation on every insertion.

---

### Issue: Repeated FindTask Calls
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/operations.go` (lines 38, 87)
**Description**: When adding tasks with a parent ID, `FindTask()` is called multiple times within the same operation, each performing an O(n) traversal.
**Impact**: Multiple O(n) operations when one would suffice.
**Solution**: Cache the parent task reference and reuse it within the same operation.
**Trade-offs**: Minimal - just better code organization.

---

### Issue: Inefficient Batch Operation Deep Copy
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/batch.go` (lines 234-254)
**Description**: The `deepCopy()` method used for dry-run validation serializes the entire TaskList to markdown and then parses it back, which is extremely inefficient for large task lists.
**Impact**: O(n) serialization + O(n) parsing for every batch operation, creating unnecessary string allocations and parsing overhead.
**Solution**:
```go
// Implement proper deep copy using struct copying
func (tl *TaskList) deepCopy() (*TaskList, error) {
    copy := &TaskList{
        Title:    tl.Title,
        Tasks:    make([]Task, len(tl.Tasks)),
        FilePath: tl.FilePath,
        Modified: tl.Modified,
    }
    
    // Deep copy tasks
    for i, task := range tl.Tasks {
        copy.Tasks[i] = deepCopyTask(task)
    }
    
    // Deep copy front matter if present
    if tl.FrontMatter != nil {
        copy.FrontMatter = &FrontMatter{
            References: make([]string, len(tl.FrontMatter.References)),
            Metadata:   make(map[string]any),
        }
        copy(copy.FrontMatter.References, tl.FrontMatter.References)
        for k, v := range tl.FrontMatter.Metadata {
            copy.FrontMatter.Metadata[k] = v
        }
    }
    
    return copy, nil
}

func deepCopyTask(t Task) Task {
    copy := Task{
        ID:         t.ID,
        Title:      t.Title,
        Status:     t.Status,
        ParentID:   t.ParentID,
        Details:    make([]string, len(t.Details)),
        References: make([]string, len(t.References)),
        Children:   make([]Task, len(t.Children)),
    }
    
    copy(copy.Details, t.Details)
    copy(copy.References, t.References)
    
    for i, child := range t.Children {
        copy.Children[i] = deepCopyTask(child)
    }
    
    return copy
}
```
**Trade-offs**: More complex implementation but eliminates serialization overhead and provides much better performance.

---

### Issue: Redundant FindTask Calls in Batch Validation
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/batch.go` (lines 131, 143, 150)
**Description**: During batch operation validation, `FindTask()` is called multiple times for the same operations, each performing an O(n) search through the task hierarchy.
**Impact**: Multiple O(n) operations during validation phase, compounded by the test copy creation.
**Solution**: Cache task lookups during validation or batch the lookups together.
**Trade-offs**: Additional complexity for significant performance improvement in batch operations.

---

### Issue: Inefficient New Task ID Detection for Details/References
**Location**: `/Users/arjenschwarz/projects/personal/rune/internal/task/batch.go` (lines 176-188)
**Description**: After adding a task, the code assumes the new task is the last one in the children/tasks slice, but with position-based insertion, this assumption is incorrect and leads to wrong task updates.
**Impact**: Details and references may be applied to the wrong task when using position-based insertion.
**Solution**: Modify `AddTask()` to return the ID of the newly created task, or implement a more reliable way to track new task IDs.
**Trade-offs**: API change required but fixes correctness issue and improves efficiency.

---
# Rune: Simplified Implementation Plan

## Project Overview
A standalone Go tool for AI agents to create and manage hierarchical markdown task lists with consistent formatting.

## Core Requirements

### Task Operations
1. **Create** new task markdown files with clean structure
2. **Parse** existing task files into structured data  
3. **Update** task status (toggle [x] ↔ [ ])
4. **Add/Remove** tasks and subtasks at any hierarchy level
5. **Modify** task content, details, and references
6. **Render** with consistent formatting and proper indentation

### Design Principles
- **Consistency over preservation**: Generate clean, standardized markdown format
- **Structure-focused**: Work with logical task hierarchy, not raw text
- **AI-agent optimized**: Simple JSON API with predictable operations
- **Version-control friendly**: Minimize diffs by updating only changed sections

## Simplified Architecture

### Core Data Model
```go
type TaskList struct {
    Title     string
    Tasks     []Task
}

type Task struct {
    ID          string    // Hierarchical: "1", "1.1", "1.2.1"
    Title       string    // Main task description
    Status      Status    // Pending, InProgress, Completed
    Details     []string  // Bullet points under task
    References  []string  // Links to requirements/design docs
    Children    []Task    // Subtasks
}

type Status int
const (
    Pending Status = iota
    InProgress  // [ ] with special marker or explicit status
    Completed   // [x]
)
```

### Key Components

#### 1. Parser (`parser/`)
- **Simple line parser**: Process markdown line-by-line
- **Hierarchy detector**: Use indentation to build task tree
- **Content extractor**: Separate titles, details, references from formatting
- **Status parser**: Extract checkbox states and task numbers

#### 2. Mutator (`mutator/`)
- **Task operations**: Add, remove, update tasks in memory structure
- **ID management**: Automatically renumber tasks after removals
- **Hierarchy maintenance**: Ensure parent-child relationships stay valid
- **Content updates**: Modify titles, details, references independently

#### 3. Renderer (`renderer/`)
- **Consistent output**: Always generate same clean format
- **Hierarchy formatting**: Proper indentation for nested tasks
- **go-output/v2 integration**: Use TableContent for structured rendering
- **Multiple formats**: Markdown (primary), JSON (API), Table (status)

#### 4. CLI Interface (`cmd/`)
```bash
# Create new task file
rune create project-tasks.md --title="Project Implementation"

# Parse and display
rune list project-tasks.md --format=table

# Add tasks
rune add project-tasks.md --title="Implement feature X"
rune add project-tasks.md --parent="1.2" --title="Write tests"

# Update task status
rune complete project-tasks.md --id="1.1" 
rune uncomplete project-tasks.md --id="1.1"

# Update task content  
rune update project-tasks.md --id="1.1" --title="New task title"
rune update project-tasks.md --id="1.1" --add-detail="Additional implementation note"
rune update project-tasks.md --id="1.1" --add-reference="Requirement 2.3"

# Remove tasks (auto-renumbers)
rune remove project-tasks.md --id="2.1"

# Batch operations via JSON
rune batch project-tasks.md --operations=ops.json
```

#### 5. JSON API (`api/`)
- **Structured input/output**: All operations accept/return JSON
- **Batch operations**: Multiple mutations in single transaction
- **Dry-run mode**: Preview changes before applying
- **Error reporting**: Clear validation messages for agents

## Implementation Phases

### Phase 1: Core Functionality
1. Define task structures and basic operations
2. Implement simple markdown parser (normalize on read)
3. Build task mutator with ID management
4. Create consistent markdown renderer

### Phase 2: CLI Interface
1. Build command-line interface with cobra/flag
2. Implement individual task operations  
3. Add JSON input/output modes
4. Create batch operation processor

### Phase 3: go-output/v2 Integration  
1. Use TableContent for task status summaries
2. Add multiple output format support
3. Create progress visualization features
4. Build statistics and reporting dashboard

### Phase 4: AI Agent Optimizations
1. Optimize JSON API for token efficiency
2. Add comprehensive error messages
3. Create usage examples for different AI agents
4. Document best practices and patterns

## Standard Output Format

### Consistent Markdown Structure
```markdown
# Task List Title

## Core Implementation Tasks

- [ ] 1. Create DataTransformer Interface
  - Define DataTransformer interface methods
  - Create TransformContext struct  
  - References: Requirement 1.1, Design DataTransformer Interface

- [x] 2. Implement Pipeline API Foundation
  - [x] 2.1. Write unit tests for Pipeline struct
    - Test Pipeline initialization
    - Test operation chaining
    - References: Requirement 3.4
  - [ ] 2.2. Create Pipeline struct
    - Add Pipeline() method to Document
    - Implement PipelineOptions struct
    - References: Requirement 3.1
```

### Formatting Rules
- **Consistent indentation**: 2 spaces per level
- **Proper numbering**: Hierarchical (1, 1.1, 1.2, 1.2.1)
- **Status markers**: `[ ]` for pending, `[x]` for completed
- **Details formatting**: Simple bullet points with consistent spacing
- **References formatting**: "References: " prefix with comma-separated list

## Key Algorithms

### Simple Parsing Strategy
```go
func parseTaskFile(content []byte) (*TaskList, error) {
    lines := strings.Split(string(content), "\n")
    var currentTask *Task
    var taskStack []*Task
    
    for _, line := range lines {
        if isTaskLine(line) {
            level, id, status, title := parseTaskLine(line)
            task := &Task{ID: id, Status: status, Title: title}
            
            // Insert into hierarchy based on indentation level
            insertTaskAtLevel(task, level, &taskStack)
            currentTask = task
            
        } else if isDetailLine(line) && currentTask != nil {
            detail := parseDetailLine(line)
            currentTask.Details = append(currentTask.Details, detail)
            
        } else if isReferenceLine(line) && currentTask != nil {
            refs := parseReferenceLine(line)
            currentTask.References = refs
        }
    }
    
    return buildTaskList(taskStack), nil
}
```

### Clean Rendering Strategy
```go
func renderTaskList(list *TaskList) []byte {
    var buf bytes.Buffer
    
    buf.WriteString("# " + list.Title + "\n\n")
    
    for _, task := range list.Tasks {
        renderTask(&buf, task, 0) // Start at level 0
    }
    
    return buf.Bytes()
}

func renderTask(buf *bytes.Buffer, task *Task, level int) {
    indent := strings.Repeat("  ", level)
    status := "[ ]"
    if task.Status == Completed {
        status = "[x]"
    }
    
    fmt.Fprintf(buf, "%s- %s %s. %s\n", indent, status, task.ID, task.Title)
    
    // Render details
    for _, detail := range task.Details {
        fmt.Fprintf(buf, "%s  - %s\n", indent, detail)
    }
    
    // Render references
    if len(task.References) > 0 {
        refs := strings.Join(task.References, ", ")
        fmt.Fprintf(buf, "%s  - References: %s\n", indent, refs)
    }
    
    // Render children
    for _, child := range task.Children {
        renderTask(buf, child, level+1)
    }
}
```

## Testing Strategy

### Unit Tests
- Task parsing from various markdown formats
- Hierarchy building and ID assignment
- Task mutations and renumbering logic
- Round-trip testing (parse → render → parse)

### Integration Tests  
- Full file operations with complex hierarchies
- Batch operations via JSON API
- Error handling and validation
- CLI interface testing

### AI Agent Tests
- JSON API validation and error reporting
- Token efficiency measurements
- Operation idempotency verification
- Concurrent operation safety

## Success Criteria
1. **Consistent output**: All rendered markdown follows same formatting rules
2. **Reliable parsing**: Handles existing task files from various sources
3. **Efficient operations**: Sub-second response for files with 100+ tasks
4. **AI-friendly API**: Clear JSON interface with comprehensive error messages
5. **Version-control friendly**: Minimal diffs for small changes

## Example Usage Scenarios

### Create New Task File
```bash
rune create implementation.md --title="Feature Implementation Tasks"
# Creates clean task file with proper structure
```

### Add Hierarchical Tasks
```json
{
  "operations": [
    {"type": "add", "title": "Implement core feature"},
    {"type": "add", "parent": "1", "title": "Write unit tests"},
    {"type": "add", "parent": "1", "title": "Create integration tests"},
    {"type": "add", "parent": "1.1", "title": "Test edge cases"}
  ]
}
```

### Update Task Details
```bash
rune update implementation.md --id="1.1" \
  --add-detail="Focus on error handling" \
  --add-reference="Design Document 2.3"
```

### Track Progress
```bash
rune list implementation.md --format=table --status=pending
# Shows pending tasks in clean table format using go-output/v2
```

## Next Steps
1. Set up repository with Go modules
2. Implement core data structures
3. Build simple markdown parser  
4. Create basic CLI commands
5. Add go-output/v2 integration for reporting
6. Write comprehensive tests
7. Document AI agent integration patterns
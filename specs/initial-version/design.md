# Go-Tasks Initial Version Design

## Overview

Go-Tasks is a command-line tool for managing hierarchical markdown task lists, designed specifically for AI agents and developers who need consistent, programmatic task management. This design document outlines the technical architecture and implementation approach for the initial MVP version.

### Design Goals

1. **Consistency**: Generate standardized markdown format regardless of input variations
2. **AI-Agent Optimization**: JSON API with predictable operations and comprehensive error reporting
3. **Hierarchical Management**: Maintain parent-child task relationships with automatic ID management
4. **Performance**: Sub-second response for files with 100+ tasks
5. **Extensibility**: Clean architecture supporting future enhancements

### Key Design Decisions

- Use Cobra framework for CLI implementation (Decision #4)
- Parse markdown line-by-line with indentation-based hierarchy detection
- Maintain task structure in memory for efficient manipulation
- Render with consistent formatting using go-output/v2 for table displays
- Support three task states: Pending `[ ]`, InProgress `[-]`, Completed `[x]` (Decision #1)

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Interface (Cobra)                    │
├─────────────────────────────────────────────────────────────┤
│                      API Layer (JSON)                        │
├─────────────────────────────────────────────────────────────┤
│                    Core Business Logic                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Parser  │  │ Mutator  │  │ Renderer │  │  Query   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
├─────────────────────────────────────────────────────────────┤
│                      Data Layer                             │
│  ┌──────────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │   TaskList   │  │   Task   │  │      Schema          │  │
│  └──────────────┘  └──────────┘  └──────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                   Output Layer (go-output/v2)               │
└─────────────────────────────────────────────────────────────┘
```

### Package Structure

```
rune/
├── cmd/                    # CLI commands (kept separate for clarity)
│   ├── root.go            # Root command setup
│   ├── create.go          # Create new task file
│   ├── list.go            # List tasks
│   ├── add.go             # Add task
│   ├── complete.go        # Complete task
│   ├── uncomplete.go      # Uncomplete task
│   ├── update.go          # Update task
│   ├── remove.go          # Remove task
│   ├── batch.go           # Batch operations
│   └── find.go            # Search tasks
└── internal/
    └── task/              # All task-related functionality
        ├── task.go        # Task and TaskList structs
        ├── parse.go       # Markdown parsing functions
        ├── render.go      # Output rendering functions
        ├── operations.go  # Task mutation methods
        └── search.go      # Search and filter methods
```

## Components and Interfaces

### Core Data Models

#### TaskList Structure
```go
type TaskList struct {
    Title     string    // Document title
    Tasks     []Task    // Root-level tasks
    FilePath  string    // Source file path
    Modified  time.Time // Last modification time
}
```

#### Task Structure
```go
type Task struct {
    ID         string   // Hierarchical ID: "1", "1.1", "1.2.1"
    Title      string   // Task description
    Status     Status   // Pending, InProgress, Completed
    Details    []string // Bullet point details
    References []string // Links to docs/requirements
    Children   []Task   // Subtasks
    ParentID   string   // Parent task ID (empty for root)
}

type Status int

const (
    Pending Status = iota    // [ ]
    InProgress               // [-]
    Completed               // [x]
)
```

### Parser Component

Simple parsing functions to process markdown files:

```go
// Direct parsing functions - no interface or struct needed
const MaxFileSize = 10 * 1024 * 1024 // 10MB

func ParseMarkdown(content []byte) (*TaskList, error) {
    if len(content) > MaxFileSize {
        return nil, fmt.Errorf("file exceeds maximum size of %d bytes", MaxFileSize)
    }
    return parseContent(string(content))
}

func ParseFile(filepath string) (*TaskList, error) {
    content, err := os.ReadFile(filepath)
    if err != nil {
        return nil, fmt.Errorf("reading file: %w", err)
    }
    return ParseMarkdown(content)
}
```

#### Parsing Algorithm

**Important**: Per Decision #2, parser reports errors without auto-correction.

```go
func parseContent(content string) (*TaskList, error) {
    lines := strings.Split(content, "\n")
    taskList := &TaskList{}
    
    // Simple recursive parsing
    tasks, _, err := parseTasksAtLevel(lines, 0, 0, "")
    if err != nil {
        return nil, err
    }
    
    taskList.Tasks = tasks
    return taskList, nil
}

func parseTasksAtLevel(lines []string, startIdx, expectedIndent int, parentID string) ([]Task, int, error) {
    var tasks []Task
    
    for i := startIdx; i < len(lines); i++ {
        indent := countIndent(lines[i])
        
        if indent < expectedIndent {
            return tasks, i-1, nil // End of this level
        }
        
        if indent > expectedIndent {
            return nil, i, fmt.Errorf("line %d: unexpected indentation", i+1)
        }
        
        if task, ok := parseTaskLine(lines[i]); ok {
            // Assign ID based on position
            if parentID == "" {
                task.ID = fmt.Sprintf("%d", len(tasks)+1)
            } else {
                task.ID = fmt.Sprintf("%s.%d", parentID, len(tasks)+1)
            }
            
            // Parse children recursively
            children, newIdx, err := parseTasksAtLevel(lines, i+1, expectedIndent+2, task.ID)
            if err != nil {
                return nil, newIdx, err
            }
            task.Children = children
            i = newIdx
            
            tasks = append(tasks, task)
        }
    }
    
    return tasks, len(lines)-1, nil
}
```

### Task Operations

Direct methods on TaskList for all modifications:

```go
// Simple task operations as methods on TaskList
func (tl *TaskList) AddTask(parentID, title string) error {
    parent := tl.FindTask(parentID)
    if parentID != "" && parent == nil {
        return fmt.Errorf("parent task %s not found", parentID)
    }
    
    newTask := Task{
        Title:    title,
        Status:   Pending,
        ParentID: parentID,
    }
    
    if parent != nil {
        newTask.ID = fmt.Sprintf("%s.%d", parentID, len(parent.Children)+1)
        parent.Children = append(parent.Children, newTask)
    } else {
        newTask.ID = fmt.Sprintf("%d", len(tl.Tasks)+1)
        tl.Tasks = append(tl.Tasks, newTask)
    }
    
    tl.Modified = time.Now()
    return nil
}

func (tl *TaskList) RemoveTask(taskID string) error {
    // Remove and renumber - simple approach
    if removed := tl.removeTaskRecursive(&tl.Tasks, taskID); removed {
        tl.renumberTasks()
        tl.Modified = time.Now()
        return nil
    }
    return fmt.Errorf("task %s not found", taskID)
}

func (tl *TaskList) UpdateStatus(taskID string, status Status) error {
    task := tl.FindTask(taskID)
    if task == nil {
        return fmt.Errorf("task %s not found", taskID)
    }
    task.Status = status
    tl.Modified = time.Now()
    return nil
}

func (tl *TaskList) UpdateTask(taskID, title string, details, refs []string) error {
    task := tl.FindTask(taskID)
    if task == nil {
        return fmt.Errorf("task %s not found", taskID)
    }
    
    if title != "" {
        task.Title = title
    }
    if details != nil {
        task.Details = details
    }
    if refs != nil {
        task.References = refs
    }
    
    tl.Modified = time.Now()
    return nil
}
```

### Rendering

Simple rendering functions for different output formats:

```go
// Direct rendering functions - no interface needed
func RenderMarkdown(tl *TaskList) []byte {
    var buf strings.Builder
    buf.WriteString(fmt.Sprintf("# %s\n\n", tl.Title))
    
    for _, task := range tl.Tasks {
        renderTask(&buf, &task, 0)
    }
    
    return []byte(buf.String())
}

func renderTask(buf *strings.Builder, task *Task, depth int) {
    indent := strings.Repeat("  ", depth)
    checkbox := "[ ]"
    switch task.Status {
    case Completed:
        checkbox = "[x]"
    case InProgress:
        checkbox = "[-]"
    }
    
    buf.WriteString(fmt.Sprintf("%s- %s %s. %s\n", 
        indent, checkbox, task.ID, task.Title))
    
    for _, detail := range task.Details {
        buf.WriteString(fmt.Sprintf("%s  - %s\n", indent, detail))
    }
    
    if len(task.References) > 0 {
        buf.WriteString(fmt.Sprintf("%s  - References: %s\n", 
            indent, strings.Join(task.References, ", ")))
    }
    
    for _, child := range task.Children {
        renderTask(buf, &child, depth+1)
    }
}

func RenderJSON(tl *TaskList) ([]byte, error) {
    return json.MarshalIndent(tl, "", "  ")
}
```

#### Rendering Rules
- Title as H1 header: `# {title}`
- Tasks with hierarchical numbering: `- [ ] 1. Task title`
- Consistent 2-space indentation per level
- Details as indented bullet points
- References with "References: " prefix
- Empty lines between major sections

### Query Component

Provides search and filtering capabilities.

#### Query Interface
```go
type Query interface {
    Find(list *TaskList, pattern string, opts QueryOptions) []Task
    Filter(list *TaskList, filter QueryFilter) []Task
}

type QueryOptions struct {
    CaseSensitive bool
    SearchDetails bool
    SearchRefs    bool
    IncludeParent bool
}

type QueryFilter struct {
    Status       *Status
    MaxDepth     int
    ParentID     string
    TitlePattern string
}
```

### CLI Layer (Cobra)

Each command follows Cobra patterns with structured command definitions.

#### Command Structure
```go
var addCmd = &cobra.Command{
    Use:   "add [file] --title [title]",
    Short: "Add a new task to the file",
    Long:  `Add a new task or subtask to the specified task file`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(addCmd)
    addCmd.Flags().StringP("title", "t", "", "Task title")
    addCmd.Flags().StringP("parent", "p", "", "Parent task ID")
    addCmd.MarkFlagRequired("title")
}
```

### Output Integration (go-output/v2)

Leverage go-output/v2 for formatted table output and multiple format support.

#### Table Rendering
```go
func renderTaskTable(tasks []Task) error {
    // Convert tasks to table data
    tableData := convertTasksToTableData(tasks)
    
    // Create document with preserved column order
    doc := output.New().
        Table("Tasks", tableData, 
            output.WithKeys("ID", "Status", "Title", "Details", "Children")).
        Build()
    
    // Render with table format
    out := output.NewOutput(
        output.WithFormat(output.Table),
        output.WithWriter(output.NewStdoutWriter()),
    )
    
    return out.Render(context.Background(), doc)
}
```

## Data Models

### File Format Specification

#### Standard Markdown Structure
```markdown
# Project Tasks

- [ ] 1. Design system architecture
  - Create component diagrams
  - Define interfaces
  - References: requirements.md, design-doc.md
  
- [-] 2. Implement core features
  - [x] 2.1. Set up project structure
    - Initialize Go modules
    - Create directory layout
  - [ ] 2.2. Build parser module
    - Implement line parser
    - Add hierarchy detection
    - References: parser-spec.md
```

### JSON API Schema

#### Batch Operations Request
```json
{
  "file": "tasks.md",
  "operations": [
    {
      "type": "add",
      "parent": "1",
      "title": "Write unit tests",
      "details": ["Test parser", "Test renderer"]
    },
    {
      "type": "update_status",
      "id": "2.1",
      "status": "completed"
    },
    {
      "type": "remove",
      "id": "3.2"
    }
  ],
  "dry_run": false
}
```

#### Query Response
```json
{
  "query": "parser",
  "matches": [
    {
      "id": "2.2",
      "title": "Build parser module",
      "status": "pending",
      "path": ["2", "2.2"],
      "parent": {
        "id": "2",
        "title": "Implement core features"
      }
    }
  ],
  "total": 1
}
```

## Error Handling

### Error Categories

1. **Parse Errors**: Malformed markdown structure
2. **Validation Errors**: Invalid operations or parameters
3. **File Errors**: I/O operations
4. **Logic Errors**: Invalid task IDs, circular references

### Error Strategy

Per Decision #2, the system will report errors for malformed content without attempting automatic fixes.

```go
// Use standard Go error patterns with clear messages
func validateTask(task *Task) error {
    if task.Title == "" {
        return fmt.Errorf("task title cannot be empty")
    }
    if len(task.Title) > 500 {
        return fmt.Errorf("task title exceeds 500 characters")
    }
    if !isValidID(task.ID) {
        return fmt.Errorf("invalid task ID format: %s", task.ID)
    }
    return nil
}

// Wrap errors with context
func parseError(line int, msg string) error {
    return fmt.Errorf("line %d: %s", line, msg)
}
```

### Error Reporting

- CLI: Human-readable messages with suggestions
- JSON API: Structured error objects with error codes
- Batch Operations: Atomic - all operations succeed or all fail (rollback on error)
- Dry Run: Validate without applying changes

### Batch Operations

Simple atomic batch operations:

```go
// Simple batch execution - validate all, then apply all
func ExecuteBatch(tl *TaskList, ops []Operation) error {
    // First validate all operations
    for i, op := range ops {
        if err := validateOperation(tl, op); err != nil {
            return fmt.Errorf("operation %d: %w", i+1, err)
        }
    }
    
    // All valid - apply them
    for _, op := range ops {
        switch op.Type {
        case "add":
            if err := tl.AddTask(op.Parent, op.Title); err != nil {
                return err
            }
        case "remove":
            if err := tl.RemoveTask(op.ID); err != nil {
                return err
            }
        case "update_status":
            if err := tl.UpdateStatus(op.ID, op.Status); err != nil {
                return err
            }
        default:
            return fmt.Errorf("unknown operation type: %s", op.Type)
        }
    }
    
    return nil
}

type Operation struct {
    Type   string `json:"type"`
    ID     string `json:"id,omitempty"`
    Parent string `json:"parent,omitempty"`
    Title  string `json:"title,omitempty"`
    Status Status `json:"status,omitempty"`
}
```

## Testing Strategy

### Unit Testing

#### Parser Tests
- Valid markdown parsing with various formats
- Malformed content handling
- ID collision resolution
- Hierarchy building edge cases
- Round-trip consistency (parse → render → parse)

#### Mutator Tests
- Task addition at all hierarchy levels
- Task removal with renumbering
- Status updates and validation
- Batch operation atomicity
- Parent-child relationship integrity

#### Renderer Tests
- Consistent formatting output
- Indentation accuracy
- Special character escaping
- Empty task list handling

### Integration Testing

#### File Operations
- Large file handling (100+ tasks)
- Concurrent file access scenarios
- File permission errors
- Non-existent file handling

#### CLI Commands
- Command flag validation
- Output format verification
- Error message clarity
- Help text accuracy

#### JSON API
- Request/response schema validation
- Batch operation transactions
- Error response formatting
- Dry-run mode verification

### Performance Testing

#### Benchmarks
- Parse time for various file sizes
- Render time for different formats
- Search performance on large task lists
- Memory usage profiling

#### Target Metrics
- < 1 second for 100 task operations
- < 10 MB memory for typical usage
- < 100ms for search operations

### Test Data

Create comprehensive test fixtures in `testdata/`:
- `simple.md`: Basic task list
- `complex.md`: Deep hierarchy with all features
- `malformed.md`: Various parsing edge cases
- `large.md`: Performance testing (500+ tasks)

## Security Considerations

### Input Validation

#### File Path Security
```go
func validateFilePath(path string) error {
    // Clean and resolve path
    cleanPath := filepath.Clean(path)
    absPath, err := filepath.Abs(cleanPath)
    
    // Ensure path is within working directory
    workDir, _ := os.Getwd()
    if !strings.HasPrefix(absPath, workDir) {
        return SecurityError{"path traversal attempt"}
    }
    
    // Check file extension (optional)
    ext := filepath.Ext(cleanPath)
    if ext != "" && ext != ".md" && ext != ".txt" {
        return ValidationError{"unsupported file type"}
    }
    
    return nil
}
```

#### Size Limits
- Maximum file size: 10MB
- Maximum task count: 10,000
- Maximum hierarchy depth: 10 levels
- Maximum JSON payload: 1MB
- Maximum batch operations: 100

#### Input Sanitization
- Escape markdown special characters in user input
- Validate task IDs match pattern `\d+(\.\d+)*`
- Limit string lengths (title: 500 chars, detail: 1000 chars)
- Reject null bytes and control characters

### File Operations

#### Atomic Writes
```go
func atomicWrite(path string, data []byte) error {
    // Write to temp file first
    tmpFile := path + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return err
    }
    
    // Atomic rename
    return os.Rename(tmpFile, path)
}
```

#### Concurrency Protection
```go
type SafeFileOps struct {
    mu sync.Mutex  // Per-file mutex map
    locks map[string]*sync.RWMutex
}
```

## Performance Considerations

### Simple Approach First
- Use strings.Builder for rendering
- Read entire files into memory (10MB limit prevents issues)
- Linear search for task lookups (fast enough for < 10,000 tasks)
- Simple recursive algorithms

### ID Renumbering
```go
// Simple renumbering after removal
func (tl *TaskList) renumberTasks() {
    for i := range tl.Tasks {
        tl.Tasks[i].ID = fmt.Sprintf("%d", i+1)
        renumberChildren(&tl.Tasks[i])
    }
}

func renumberChildren(parent *Task) {
    for i := range parent.Children {
        parent.Children[i].ID = fmt.Sprintf("%s.%d", parent.ID, i+1)
        renumberChildren(&parent.Children[i])
    }
}
```

### Performance Targets
- Parse 100 tasks: < 100ms
- Render 100 tasks: < 50ms  
- Search 1000 tasks: < 100ms
- All easily achievable with simple algorithms

## Future Extensibility

### Planned Extensions
- Plugin architecture for custom transformations
- Multiple file batch operations
- Task dependencies and relationships
- Custom field support
- Advanced query language

### Extension Points
- Content interface for new content types
- Renderer interface for new formats
- Transformer interface for data processing
- Writer interface for new destinations

## Implementation Priority

### Phase 1: Core Foundation
1. Data models and structures
2. Basic parser implementation
3. Simple markdown renderer
4. File I/O operations

### Phase 2: CLI Interface
1. Cobra command structure
2. Create and list commands
3. Add and remove commands
4. Status update commands

### Phase 3: Advanced Features
1. Query and search implementation
2. Batch operations via JSON
3. go-output/v2 integration
4. Multiple output formats

### Phase 4: Polish and Optimization
1. Comprehensive error handling
2. Performance optimizations
3. Extended test coverage
4. Documentation and examples
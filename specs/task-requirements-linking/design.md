# Task Requirements Linking Design

## Overview

This feature adds support for linking tasks to specific requirement acceptance criteria in a requirements file. The implementation follows rune's existing patterns for markdown parsing, rendering, and command-line interaction. Requirements will be stored in task metadata and rendered as clickable markdown links pointing to anchors in a requirements file.

### Key Design Principles

1. **Minimal Complexity**: Keep implementation simple per user directive - no requirement ID validation, no complex file path management
2. **Consistency**: Follow existing patterns for References field - same parsing/rendering approach
3. **Single Source of Truth**: RequirementsFile path stored once in TaskList, not extracted from individual links
4. **Backward Compatibility**: Optional field that doesn't break existing task files

## Architecture

### Data Flow

1. **CLI Input** → Command flags (--requirements, --requirements-file)
2. **Parsing** → Extract requirements from markdown detail lines
3. **In-Memory** → Store in Task.Requirements ([]string), TaskList.RequirementsFile (string)
4. **Rendering** → Convert requirement IDs to markdown links [ID](file#ID)
5. **File Output** → Write formatted markdown with clickable links

### Component Interactions

```
┌─────────────────┐
│  CLI Commands   │  add, update, batch
│  (cmd/)         │
└────────┬────────┘
         │ validates requirements format
         ▼
┌─────────────────┐
│  Task Structure │  Task.Requirements, TaskList.RequirementsFile
│  (internal/task)│
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌────────┐
│ Parser │ │Renderer│  parse.go, render.go
└────────┘ └────────┘
    │         │
    └────┬────┘
         │ reads/writes markdown files
         ▼
┌─────────────────┐
│ Markdown Files  │  tasks.md with requirement links
└─────────────────┘
```

## Components and Interfaces

### 1. Task Structure Changes

**File**: [internal/task/task.go](internal/task/task.go)

Add two new fields:

```go
// Task represents a single task in a hierarchical task list
type Task struct {
    ID           string
    Title        string
    Status       Status
    Details      []string
    References   []string
    Requirements []string  // NEW: requirement IDs (e.g., "1.1", "2.3")
    Children     []Task
    ParentID     string
}

// TaskList represents a collection of tasks with metadata
type TaskList struct {
    Title            string
    Tasks            []Task
    FrontMatter      *FrontMatter
    FilePath         string
    RequirementsFile string  // NEW: path to requirements file
    Modified         time.Time
}
```

**Validation**:
- Requirements array is optional (can be nil or empty)
- Each requirement ID must match pattern `^\d+(\.\d+)*$`
- No validation that IDs exist in requirements file (per Decision 2)
- RequirementsFile defaults to "requirements.md" when not specified

### 2. Markdown Parsing

**File**: [internal/task/parse.go](internal/task/parse.go)

Add requirements parsing to `parseDetailsAndChildren()` function:

```go
// In parseDetailsAndChildren function, after existing References parsing:

// Parse requirements line (format: "Requirements: [1.1](file#1.1), [2.3](file#2.3)")
if reqs, ok := strings.CutPrefix(v, "Requirements: "); ok {
    reqIDs, reqFile := parseRequirements(reqs)
    task.Requirements = reqIDs
    // Store requirements file path in TaskList if not already set
    if reqFile != "" && taskList.RequirementsFile == "" {
        taskList.RequirementsFile = reqFile
    }
}
```

New helper function (add after parseReferences at line 332):

```go
// Compile regex once at package level for performance
var requirementLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^#\)]+)#[^\)]+\)`)

// parseRequirements extracts requirement IDs from markdown links
// Input: "[1.1](requirements.md#1.1), [1.2](requirements.md#1.2)"
// Returns: requirement IDs and the requirements file path
// Placed after parseReferences function for consistency
func parseRequirements(reqs string) ([]string, string) {
    parts := strings.Split(reqs, ",")  // Use standard strings.Split
    requirements := make([]string, 0, len(parts))
    var reqFile string

    // Pattern matches: [ID](path#ID) or [ID](path#anchor)
    for _, part := range parts {
        part = strings.TrimSpace(part)
        if matches := requirementLinkPattern.FindStringSubmatch(part); matches != nil {
            reqID := strings.TrimSpace(matches[1])
            if reqID != "" && isValidID(reqID) {  // Reuse existing isValidID function
                requirements = append(requirements, reqID)
                // Extract file path from first valid link
                if reqFile == "" {
                    reqFile = matches[2]
                }
            }
        }
    }

    return requirements, reqFile
}
```

**Error Handling**: Malformed requirement lines are treated as plain text details (per Decision 5)

### 3. Markdown Rendering

**File**: [internal/task/render.go](internal/task/render.go)

Add requirements rendering to `renderTask()` function:

```go
// In renderTask function, before references rendering:

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
```

**Formatting**:
- Requirements line format: `  - Requirements: [1.1](file#1.1), [1.2](file#1.2)`
- Plain text (NO italic formatting for easier parsing and consistency)
- Positioned before References line
- 2-space indentation per task depth level

**Signature Change**: Modify renderTask to accept reqFile parameter:

```go
// Change from:
func renderTask(buf *strings.Builder, task *Task, depth int)

// To:
func renderTask(buf *strings.Builder, task *Task, depth int, reqFile string)
```

This parameter propagates through recursive calls automatically since renderTask calls itself for children.

Pass reqFile from RenderMarkdown:

```go
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
        if i > 0 {
            buf.WriteString("\n")
        }
        renderTask(&buf, &task, 0, reqFile)  // Pass reqFile parameter
    }

    return []byte(buf.String())
}
```

Also update RenderMarkdownWithPhases similarly to pass reqFile parameter.

### 4. Command-Line Interface

#### Add Command

**File**: [cmd/add.go](cmd/add.go)

Add new flags:

```go
var (
    addTitle           string
    addParent          string
    addPosition        string
    addPhase           string
    addRequirements    string  // NEW: comma-separated requirement IDs
    addRequirementsFile string // NEW: path to requirements file
)

func init() {
    // ... existing flags ...
    addCmd.Flags().StringVar(&addRequirements, "requirements", "",
        "comma-separated requirement IDs (e.g., \"1.1,1.2,2.3\")")
    addCmd.Flags().StringVar(&addRequirementsFile, "requirements-file", "",
        "path to requirements file (default: requirements.md)")
}
```

In `runAdd()`:

```go
// After task is added via tl.AddTask(), update requirements if provided
if addRequirements != "" {
    // Parse comma-separated IDs
    reqIDs := parseRequirementIDs(addRequirements)

    // Validate format using existing validation
    for _, reqID := range reqIDs {
        if !isValidID(reqID) {  // Use existing isValidID from task package
            return fmt.Errorf("invalid requirement ID format: %s", reqID)
        }
    }

    // Update task with requirements
    if newTask := tl.FindTask(newTaskID); newTask != nil {
        newTask.Requirements = reqIDs
    }
}

// Set requirements file path if provided, otherwise use default
if addRequirementsFile != "" {
    tl.RequirementsFile = addRequirementsFile
} else if tl.RequirementsFile == "" {
    tl.RequirementsFile = task.DefaultRequirementsFile
}
```

Helper function:

```go
func parseRequirementIDs(input string) []string {
    parts := strings.Split(input, ",")  // Use standard strings.Split
    ids := make([]string, 0)
    for _, part := range parts {
        if trimmed := strings.TrimSpace(part); trimmed != "" {
            ids = append(ids, trimmed)
        }
    }
    return ids
}
```

#### Update Command

**File**: [cmd/update.go](cmd/update.go)

Add new flags:

```go
var (
    // ... existing flags ...
    updateRequirements string  // NEW
    clearRequirements  bool    // NEW
)

func init() {
    // ... existing flags ...
    updateCmd.Flags().StringVar(&updateRequirements, "requirements", "",
        "comma-separated requirement IDs")
    updateCmd.Flags().BoolVar(&clearRequirements, "clear-requirements", false,
        "clear all requirements from the task")
}
```

In `runUpdate()`:

```go
// Handle requirements
var newRequirements []string
if clearRequirements {
    newRequirements = []string{} // empty slice to clear
} else if updateRequirements != "" {
    newRequirements = parseRequirementIDs(updateRequirements)

    // Validate format using existing validation
    for _, reqID := range newRequirements {
        if !isValidID(reqID) {  // Reuse existing isValidID from task package
            return fmt.Errorf("invalid requirement ID format: %s", reqID)
        }
    }
}

// Update task
if err := tl.UpdateTask(taskID, updateTitle, newDetails, newReferences, newRequirements); err != nil {
    return fmt.Errorf("failed to update task: %w", err)
}
```

Modify `UpdateTask()` signature in [internal/task/operations.go](internal/task/operations.go):

```go
// UpdateTask updates task with title, details, references, and requirements
func (tl *TaskList) UpdateTask(taskID, title string, details, references, requirements []string) error {
    task := tl.FindTask(taskID)
    if task == nil {
        return fmt.Errorf("task %s not found", taskID)
    }

    // Update title if provided
    if title != "" {
        if len(title) > 500 {
            return fmt.Errorf("task title exceeds 500 characters")
        }
        task.Title = title
    }

    // Update details if provided (nil means don't update, empty slice means clear)
    if details != nil {
        task.Details = details
    }

    // Update references if provided (nil means don't update, empty slice means clear)
    if references != nil {
        task.References = references
    }

    // Update requirements if provided (nil means don't update, empty slice means clear)
    if requirements != nil {
        task.Requirements = requirements
    }

    return nil
}
```

**Implementation Note**: All existing call sites of `UpdateTask()` need to be updated to pass `nil` for requirements parameter if they don't need to modify requirements.

#### Batch Command

**File**: [internal/task/batch.go](internal/task/batch.go)

Add fields to Operation struct:

```go
type Operation struct {
    Type              string   `json:"type"`
    ID                string   `json:"id,omitempty"`
    Parent            string   `json:"parent,omitempty"`
    Title             string   `json:"title,omitempty"`
    Status            *Status  `json:"status,omitempty"`
    Details           []string `json:"details,omitempty"`
    References        []string `json:"references,omitempty"`
    Requirements      []string `json:"requirements,omitempty"`      // NEW
    Position          string   `json:"position,omitempty"`
    Phase             string   `json:"phase,omitempty"`
}

type BatchRequest struct {
    File             string      `json:"file"`
    Operations       []Operation `json:"operations"`
    DryRun           bool        `json:"dry_run"`
    RequirementsFile string      `json:"requirements_file,omitempty"` // NEW
}
```

In `validateOperation()` and `applyOperation()`:

```go
// In validateOperation - add validation for requirements in add/update ops
case addOperation:
    // ... existing validations ...
    if len(op.Requirements) > 0 {
        for _, reqID := range op.Requirements {
            if !validateTaskIDFormat(reqID) {  // This already exists in batch.go
                return fmt.Errorf("invalid requirement ID format: %s", reqID)
            }
        }
    }

case updateOperation:
    // ... existing validations ...
    if len(op.Requirements) > 0 {
        for _, reqID := range op.Requirements {
            if !validateTaskIDFormat(reqID) {  // This already exists in batch.go
                return fmt.Errorf("invalid requirement ID format: %s", reqID)
            }
        }
    }

// In applyOperation - handle requirements
case addOperation:
    newTaskID, err := tl.AddTask(op.Parent, op.Title, op.Position)
    if err != nil {
        return err
    }
    // Update with details, references, AND requirements
    if len(op.Details) > 0 || len(op.References) > 0 || len(op.Requirements) > 0 {
        if newTaskID != "" {
            return tl.UpdateTask(newTaskID, "", op.Details, op.References, op.Requirements)
        }
    }
    return nil

case updateOperation:
    // ... existing status handling ...
    return tl.UpdateTask(op.ID, op.Title, op.Details, op.References, op.Requirements)
```

In `runBatch()` in [cmd/batch.go](cmd/batch.go):

```go
// After parsing JSON request, set requirements file
if req.RequirementsFile != "" {
    taskList.RequirementsFile = req.RequirementsFile
} else if taskList.RequirementsFile == "" {
    taskList.RequirementsFile = task.DefaultRequirementsFile
}
```

### 5. JSON API

**File**: [internal/task/render.go](internal/task/render.go) - JSON marshaling

Task struct will automatically include Requirements field in JSON output (standard Go marshaling).

TaskList JSON needs manual handling:

```go
// RenderJSON needs to include RequirementsFile
func RenderJSON(tl *TaskList) ([]byte, error) {
    // Standard marshaling already includes all fields
    return json.MarshalIndent(tl, "", "  ")
}
```

Example JSON output:

```json
{
  "title": "Project Tasks",
  "tasks": [
    {
      "ID": "1",
      "Title": "Implement authentication",
      "Status": 0,
      "Details": ["Use JWT tokens"],
      "References": [],
      "Requirements": ["1.1", "1.2"],
      "Children": [],
      "ParentID": ""
    }
  ],
  "requirements_file": "requirements.md"
}
```

## Data Models

### Task Requirements

```go
type Task struct {
    // ... existing fields ...
    Requirements []string `json:"requirements,omitempty"`
}
```

- Type: `[]string` - array of requirement IDs
- Format: Each ID matches `^\d+(\.\d+)*$` (e.g., "1", "1.1", "1.2.3")
- Optional: Can be nil or empty array
- No validation that IDs exist in requirements file

### TaskList Requirements File

```go
const DefaultRequirementsFile = "requirements.md"

type TaskList struct {
    // ... existing fields ...
    RequirementsFile string `json:"requirements_file,omitempty"`
}
```

- Type: `string` - path to requirements file
- Default: `DefaultRequirementsFile` constant ("requirements.md") if not specified
- Scope: Applies to all tasks in the file (single source of truth)
- Persistence: Stored in memory only, set via CLI flags or derived from first requirement link during parsing. Once set, it persists in TaskList structure for the session. Not persisted to markdown to keep files clean.

### Markdown Format

Requirements rendered in task details:

```markdown
- [ ] 1. Implement user authentication
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
  - References: auth-spec.md, security-guidelines.md
```

Key characteristics:
- Detail line format: `  - Requirements: <links>`
- Markdown links: `[ID](file#ID)`
- Multiple requirements: Comma-separated
- Plain text (NO italic formatting to keep parsing simple)
- Positioned before References line

## Error Handling

### Validation Errors

1. **Invalid Requirement ID Format**
   - When: CLI flags or batch JSON contain invalid ID format
   - Action: Return error with clear message
   - Example: `"invalid requirement ID format: abc (must match pattern ^\d+(\.\d+)*$)"`

2. **Malformed Requirements Line**
   - When: Parsing markdown with incorrectly formatted requirements
   - Action: Treat as plain text detail, continue parsing (graceful degradation)
   - Rationale: Preserves user content, consistent with parser philosophy

3. **Missing Required Fields**
   - When: Batch operations missing title (add) or id (update/remove)
   - Action: Return validation error before applying any operations
   - Example: `"operation 3: add operation requires title"`

### Non-Errors (No Validation)

Per Decision 2, the following are **NOT** validated:

1. **Requirement ID Existence**: No check that IDs actually exist in requirements file
2. **Requirements File Existence**: No check that requirements file exists
3. **Broken Links**: User's responsibility to ensure anchors exist

### Error Propagation

- CLI commands: Return errors to user with context
- Batch operations: Atomic - all operations validated before any applied
- Parsing: Continue on malformed requirements, collect errors for other issues

## Testing Strategy

### Unit Tests

#### Task Structure Tests
**File**: `internal/task/task_test.go`

```go
func TestTaskRequirementsValidation(t *testing.T) {
    tests := map[string]struct {
        requirements []string
        wantValid    bool
    }{
        "valid single requirement": {
            requirements: []string{"1.1"},
            wantValid:    true,
        },
        "valid multiple requirements": {
            requirements: []string{"1.1", "2.3", "3.4.5"},
            wantValid:    true,
        },
        "invalid format": {
            requirements: []string{"abc"},
            wantValid:    false,
        },
        "empty array is valid": {
            requirements: []string{},
            wantValid:    true,
        },
    }
    // Test validation logic
}
```

#### Parsing Tests
**File**: `internal/task/parse_test.go`

```go
func TestParseRequirements(t *testing.T) {
    tests := map[string]struct {
        input       string
        wantIDs     []string
        wantFile    string
    }{
        "single requirement": {
            input:    "[1.1](requirements.md#1.1)",
            wantIDs:  []string{"1.1"},
            wantFile: "requirements.md",
        },
        "multiple requirements": {
            input:    "[1.1](requirements.md#1.1), [1.2](requirements.md#1.2)",
            wantIDs:  []string{"1.1", "1.2"},
            wantFile: "requirements.md",
        },
        "malformed link": {
            input:    "1.1, 1.2", // No markdown links
            wantIDs:  []string{},
            wantFile: "",
        },
    }
    // Test parseRequirements helper
}

func TestParseMarkdownWithRequirements(t *testing.T) {
    markdown := `# Tasks
- [ ] 1. Implement feature
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
`
    tl, err := ParseMarkdown([]byte(markdown))
    // Verify task.Requirements = ["1.1", "1.2"]
    // Verify tl.RequirementsFile = "requirements.md"
}
```

#### Rendering Tests
**File**: `internal/task/render_test.go`

```go
func TestRenderRequirements(t *testing.T) {
    tl := &TaskList{
        Title: "Tasks",
        RequirementsFile: "requirements.md",
        Tasks: []Task{
            {
                ID:           "1",
                Title:        "Test task",
                Status:       Pending,
                Requirements: []string{"1.1", "1.2"},
            },
        },
    }

    output := RenderMarkdown(tl)

    // Verify output contains:
    // - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
}

func TestRoundTripRequirements(t *testing.T) {
    // Parse markdown → modify task → render → parse again
    // Verify requirements preserved
}
```

#### Batch Operations Tests
**File**: `internal/task/batch_test.go`

```go
func TestBatchAddWithRequirements(t *testing.T) {
    ops := []Operation{
        {
            Type:         "add",
            Title:        "New task",
            Requirements: []string{"1.1", "1.2"},
        },
    }

    // Test requirements are added to task
}

func TestBatchUpdateRequirements(t *testing.T) {
    ops := []Operation{
        {
            Type:         "update",
            ID:           "1",
            Requirements: []string{"2.1", "2.2"},
        },
    }

    // Test requirements are updated
}

func TestBatchRequirementsValidation(t *testing.T) {
    ops := []Operation{
        {
            Type:         "add",
            Title:        "Task",
            Requirements: []string{"invalid"},
        },
    }

    // Test validation fails, no operations applied (atomic)
}
```

### Integration Tests
**File**: `cmd/integration_test.go`

```go
func TestIntegrationRequirementsWorkflow(t *testing.T) {
    // Create task file
    // Add task with --requirements flag
    // Verify requirements in markdown
    // Parse file and verify Requirements field
    // Update requirements via batch command
    // Verify changes persisted
}
```

### Command Tests
**File**: `cmd/add_test.go`, `cmd/update_test.go`

```go
func TestAddCommandWithRequirements(t *testing.T) {
    // Test --requirements flag parsing
    // Test --requirements-file flag
    // Test validation errors
}

func TestUpdateCommandRequirements(t *testing.T) {
    // Test --requirements flag
    // Test --clear-requirements flag
}
```

### Test Coverage Goals

- Unit tests: >80% coverage for new code
- Integration tests: Cover complete workflows
- Edge cases: Malformed input, empty values, mixed operations
- Backward compatibility: Verify existing files without requirements still work

## Implementation Notes

### Existing Patterns to Follow

1. **References Field Pattern**: Requirements follows exact same approach
   - Parse from detail line with prefix
   - Store as string array
   - Render with formatting

2. **Validation Pattern**: Use existing validators
   - `taskIDPattern` regex for requirement ID format
   - `validateTaskIDFormat()` helper function
   - Validation in both CLI and batch operations

3. **Batch Operation Pattern**: Extend existing Operation struct
   - Add optional fields
   - Validate in `validateOperation()`
   - Apply in `applyOperation()`
   - Atomic execution via test copy

4. **Command Flag Pattern**: Consistent with existing flags
   - String flags for input
   - Boolean flags for clearing
   - Clear validation error messages

### Complexity Considerations

**What to Keep Simple** (per user directive):
- No validation of requirement ID existence in file
- No complex path resolution (just store the path string)
- No special UI formatting (standard detail line)
- Default to "requirements.md" everywhere

**What Needs Care**:
- Round-trip parsing/rendering must preserve requirements
- Batch operations must be atomic (validate all before applying)
- Requirement IDs must be validated for format
- TaskList.RequirementsFile as single source of truth

### Default Behavior

1. If `--requirements-file` not specified: Default to `DefaultRequirementsFile` constant
2. If no requirements file in parsed markdown: Default to `DefaultRequirementsFile` constant
3. If `--requirements` not specified: Don't modify task requirements
4. If batch JSON omits `requirements_file`: Default to `DefaultRequirementsFile` constant
5. RequirementsFile persists in TaskList for the session but is not written to markdown (keeps files clean)

### Phase Interaction

Requirements feature works independently of phases:
- Tasks with requirements can be in any phase
- Phase operations don't affect requirements
- Both features can be used together without conflict

## Documentation Requirements

### README.md Updates

Add section after References documentation:

```markdown
### Requirements

Link tasks to specific requirement acceptance criteria using the `--requirements` flag:

rune add tasks.md --title "Implement login" --requirements "1.1,1.2,2.3"

Specify a custom requirements file:

rune add tasks.md --title "Implement login" --requirements "1.1,1.2" --requirements-file "specs/requirements.md"

Requirements are rendered as clickable markdown links:

- [ ] 1. Implement login
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)

Update requirements:

rune update tasks.md 1 --requirements "3.1,3.2"

Clear requirements:

rune update tasks.md 1 --clear-requirements

**Requirements vs References:**
- Requirements: Links to acceptance criteria with automatic link generation
- References: Free-form text without link generation
```

### docs/json-api.md Updates

Add to data model documentation:

```markdown
## Requirements Field

Tasks can include a `requirements` field containing an array of requirement IDs:

{
  "ID": "1",
  "Title": "Implement authentication",
  "Requirements": ["1.1", "1.2", "2.3"]
}

The TaskList includes a `requirements_file` field:

{
  "title": "Project Tasks",
  "requirements_file": "requirements.md",
  "tasks": [...]
}

### Batch Operations

Add requirements to tasks:

{
  "type": "add",
  "title": "New task",
  "requirements": ["1.1", "1.2"]
}

Update requirements:

{
  "type": "update",
  "id": "1",
  "requirements": ["3.1", "3.2"]
}

Specify requirements file path:

{
  "file": "tasks.md",
  "requirements_file": "specs/requirements.md",
  "operations": [...]
}
```

## Critical Design Decisions

### 1. RequirementsFile Persistence Strategy

**Problem**: If RequirementsFile is not stored in markdown, how is it managed across sessions?

**Solution**:
- RequirementsFile is stored in TaskList struct (in-memory only)
- Set via CLI flags (--requirements-file) when adding/updating tasks
- Derived from first requirement link during parsing (extract from markdown link paths)
- Persists for the session duration but not written to markdown
- When file is reopened, it's re-derived from existing requirement links or defaults to "requirements.md"

**Rationale**: Keeps markdown files clean while supporting different requirement file paths. The path can be reconstructed from existing links, so no data is lost.

### 2. No Italic Formatting for Requirements

**Problem**: Italic formatting requires parsing `*Requirements: ...*` vs `Requirements: ...`

**Solution**: Use plain text format without italics for Requirements line.

**Rationale**: Simplifies parsing (no need to strip asterisks). Maintains consistency and reduces chance of round-trip parsing bugs. Both Requirements and References use plain text formatting.

### 3. Reuse Existing Validation Functions

**Problem**: Creating new validation functions duplicates existing logic.

**Solution**: Reuse existing `isValidID()` and `validateTaskIDFormat()` functions from task.go and batch.go.

**Rationale**: DRY principle. Requirement IDs have the same format as task IDs, so same validation applies.

### 4. UpdateTask Signature Change

**Decision**: Modify existing `UpdateTask()` method signature to include requirements parameter.

**Rationale**: Since the tool is internal and all call sites can be easily updated, there's no need to maintain backward compatibility or create multiple functions. Simpler is better.

## Open Questions

1. **Path Resolution**: Should requirements file path be relative to task file directory or current working directory?
   - **Answer**: Store as-is, let markdown renderer handle relative paths (standard markdown behavior)

2. **Multiple Requirements Files**: Should we support different requirements files per task?
   - **Answer**: No - single file per TaskList keeps implementation simple (per Decision 3)

3. **JSON API Field Names**: Use `requirements_file` (snake_case) or `RequirementsFile` (PascalCase)?
   - **Answer**: Use `json:"requirements_file"` struct tag for snake_case in JSON output (consistent with Go conventions)

## Dependencies

- No new external dependencies required
- Uses existing Go standard library packages:
  - `strings` - string manipulation
  - `regexp` - pattern validation
  - `encoding/json` - JSON marshaling
  - `fmt` - formatting

## Performance Considerations

- Requirements parsing adds minimal overhead (simple string operations)
- Validation uses existing compiled regex patterns
- No additional file I/O beyond existing operations
- Batch operations maintain atomic execution without performance degradation

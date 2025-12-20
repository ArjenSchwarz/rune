# Design: Consistent Output Format

## Overview

This design ensures all commands respect the `--format` flag consistently. Rather than forcing unified response types, each command keeps its natural data structure while following common conventions.

## Design Principles

1. **Format flag must be respected** - No command should output plain text when JSON is requested
2. **Command-specific structures** - Each command returns data in its natural shape
3. **Common conventions** - All JSON responses include `success` field; empty states handled consistently
4. **Minimal changes** - Fix the problem without over-engineering

## Architecture

### No Centralized OutputHelper

Instead of a centralized helper with generic response types, each command:
1. Defines its own response struct matching its data
2. Uses a small set of shared utility functions for common patterns
3. Handles format routing locally

### Shared Utilities in `cmd/format.go`

```go
// Small utility file for common format operations

// outputJSON marshals any struct to stdout as JSON
func outputJSON(data any) error {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}

// outputMarkdownMessage writes a blockquote message
func outputMarkdownMessage(message string) {
    fmt.Printf("> %s\n", message)
}

// outputMessage writes a plain text message (for table format)
func outputMessage(message string) {
    fmt.Println(message)
}

// verboseStderr writes verbose output to stderr (safe with JSON)
func verboseStderr(format string, args ...any) {
    if verbose {
        fmt.Fprintf(os.Stderr, format+"\n", args...)
    }
}
```

### Common Conventions

All commands follow these conventions:

1. **JSON responses include `success: true`** when operation completes normally
2. **Empty lists return `[]`** not `null`
3. **Empty single items return `null`** for the data field
4. **Verbose output goes to stderr** when JSON format is requested
5. **Errors go to stderr** as plain text (standard CLI behavior)

## Command-Specific Response Types

Each command defines its own response type:

### Mutation Commands (complete, uncomplete, progress)

```go
// In complete.go
type CompleteResponse struct {
    Success       bool     `json:"success"`
    Message       string   `json:"message"`
    TaskID        string   `json:"task_id"`
    Title         string   `json:"title"`
    AutoCompleted []string `json:"auto_completed,omitempty"`
}
```

### Add Command

```go
type AddResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    TaskID  string `json:"task_id"`
    Title   string `json:"title"`
    Parent  string `json:"parent,omitempty"`
}
```

### Remove Command

```go
type RemoveResponse struct {
    Success      bool `json:"success"`
    Message      string `json:"message"`
    TaskID       string `json:"task_id"`
    ChildCount   int    `json:"children_removed"`
}
```

### Create Command

```go
type CreateResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Path    string `json:"path"`
    Title   string `json:"title,omitempty"`
}
```

### Next Command (empty state)

```go
type NextEmptyResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Data    any    `json:"data"` // null when no next task
}
```

### List/Find Commands (empty state)

```go
type ListEmptyResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Count   int    `json:"count"` // 0
    Data    []any  `json:"data"`  // []
}
```

### Renumber Command

Keeps existing structure, adds `success` field:

```go
type RenumberResponse struct {
    Success    bool   `json:"success"`
    TaskCount  int    `json:"task_count"`
    BackupFile string `json:"backup_file"`
}
```

## Implementation Pattern

Each command follows this pattern:

```go
func runComplete(cmd *cobra.Command, args []string) error {
    // Use stderr for verbose when JSON requested
    if format == formatJSON && verbose {
        fmt.Fprintf(os.Stderr, "Using task file: %s\n", filename)
    } else if verbose {
        fmt.Printf("Using task file: %s\n", filename)
    }

    // ... business logic ...

    // Format-aware output
    switch format {
    case formatJSON:
        return outputJSON(CompleteResponse{
            Success:       true,
            Message:       fmt.Sprintf("Completed task %s", taskID),
            TaskID:        taskID,
            Title:         task.Title,
            AutoCompleted: autoCompleted,
        })
    case formatMarkdown:
        fmt.Printf("**Completed:** %s - %s\n", taskID, task.Title)
        return nil
    default: // table
        fmt.Printf("Completed task %s: %s\n", taskID, task.Title)
        return nil
    }
}
```

## Empty State Handling

### Single-Item Commands (next)

```go
// In next.go, when no next task
switch format {
case formatJSON:
    return outputJSON(NextEmptyResponse{
        Success: true,
        Message: "All tasks are complete!",
        Data:    nil,
    })
case formatMarkdown:
    outputMarkdownMessage("All tasks are complete!")
    return nil
default:
    fmt.Println("All tasks are complete!")
    return nil
}
```

### List Commands (list, find)

```go
// In find.go, when no matches
switch format {
case formatJSON:
    return outputJSON(ListEmptyResponse{
        Success: true,
        Message: "No matching tasks found",
        Count:   0,
        Data:    []any{},
    })
case formatMarkdown:
    outputMarkdownMessage("No matching tasks found")
    return nil
default:
    fmt.Println("No matching tasks found")
    return nil
}
```

## Files to Modify

| File | Changes |
|------|---------|
| `cmd/format.go` | New file - small utility functions |
| `cmd/complete.go` | Add response type, format switch |
| `cmd/uncomplete.go` | Add response type, format switch |
| `cmd/progress.go` | Add response type, format switch |
| `cmd/add.go` | Add response type, format switch |
| `cmd/remove.go` | Add response type, format switch |
| `cmd/update.go` | Add response type, format switch |
| `cmd/create.go` | Add response type, format switch |
| `cmd/add_phase.go` | Add response type, format switch |
| `cmd/add_frontmatter.go` | Add response type, format switch |
| `cmd/next.go` | Fix empty state output |
| `cmd/list.go` | Fix empty state output |
| `cmd/find.go` | Fix empty state output |
| `cmd/renumber.go` | Add success field to JSON |

## Testing Strategy

### Per-Command Format Tests

Each command gets format-specific tests:

```go
func TestComplete_JSONFormat(t *testing.T) {
    // Run complete with --format json
    // Verify: valid JSON with success, message, task_id fields
}

func TestComplete_MarkdownFormat(t *testing.T) {
    // Run complete with --format markdown
    // Verify: markdown formatted output
}

func TestNext_AllComplete_JSONFormat(t *testing.T) {
    // Run next with all tasks complete, --format json
    // Verify: {"success": true, "message": "...", "data": null}
}

func TestFind_NoMatches_JSONFormat(t *testing.T) {
    // Run find with no matches, --format json
    // Verify: {"success": true, "message": "...", "count": 0, "data": []}
}
```

### Verbose + JSON Tests

```go
func TestComplete_Verbose_JSON_StderrOutput(t *testing.T) {
    // Run complete with --verbose --format json
    // Verify: verbose output on stderr, JSON on stdout
}
```

## Migration Strategy

### Phase 1: Add format.go utilities
- Create `cmd/format.go` with shared functions
- Add tests

### Phase 2: Fix mutation commands
- Update each mutation command to use format switch
- Add command-specific response types
- Move verbose to stderr when JSON

### Phase 3: Fix read command empty states
- Update next.go empty state handling
- Update list.go empty state handling
- Update find.go empty state handling

### Phase 4: Align existing JSON outputs
- Add `success` field to renumber JSON
- Verify batch.go consistency

### Phase 5: Integration tests
- Add format-specific integration tests
- Document JSON schemas per command

## Error Handling

Errors continue using Cobra's standard handling - returned from `RunE`, written to stderr by Cobra. This design does not change error handling.

## Backward Compatibility

Commands that already output JSON (next, list, find, renumber) will have minor changes:
- Addition of `success` field
- Empty states return structured responses instead of plain text

These are additive changes. Existing JSON consumers should not break unless they were parsing the plain text error messages.

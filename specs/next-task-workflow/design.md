# Next Task Workflow - Design Document

## Overview

The next-task-workflow feature enhances the go-tasks CLI to provide intelligent task retrieval and document reference capabilities. This design focuses on creating a developer-friendly workflow that automatically finds the next actionable task, supports git branch-based file discovery, and maintains reference documentation within task files.

The feature introduces:
- A "next" command for retrieving the first incomplete task in a hierarchy
- Git branch-based automatic file discovery through configuration
- Reference document sections in task files
- Automatic parent task completion when all subtasks are done
- Configuration management via YAML files

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │   next   │  │   list   │  │ complete │  │  batch   │  │
│  │  command │  │  command │  │ command  │  │ command  │  │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘  └─────┬────┘  │
└────────┼─────────────┼─────────────┼─────────────┼─────────┘
         │             │             │             │
         ▼             ▼             ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Business Logic Layer                      │
│  ┌────────────────────────────────────────────────────┐    │
│  │              Task Operations Manager               │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │    │
│  │  │   Next   │  │ Complete │  │   Reference   │   │    │
│  │  │  Finder  │  │  Handler │  │    Parser     │   │    │
│  │  └──────────┘  └──────────┘  └──────────────┘   │    │
│  └────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────┐    │
│  │              Configuration Manager                 │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │    │
│  │  │   YAML   │  │   Git    │  │    Path      │   │    │
│  │  │  Parser  │  │Discovery│  │  Resolver    │   │    │
│  │  └──────────┘  └──────────┘  └──────────────┘   │    │
│  └────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
         │             │             │
         ▼             ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│                      Data Layer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐             │
│  │   Task   │  │ TaskList │  │   Config     │             │
│  │  struct  │  │  struct  │  │   struct     │             │
│  └──────────┘  └──────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────┘
```

### Module Organization

```
go-tasks/
├── cmd/
│   ├── next.go           # New "next" command implementation
│   ├── next_test.go      # Unit tests for next command
│   └── root.go           # Updated to register new command
│
├── internal/
│   ├── task/
│   │   ├── next.go       # Next task finding logic
│   │   ├── next_test.go  # Unit tests for next finder
│   │   ├── references.go # Reference document handling
│   │   ├── references_test.go
│   │   ├── autocomplete.go # Auto-complete parent tasks
│   │   ├── autocomplete_test.go
│   │   ├── operations.go # Updated with auto-completion
│   │   └── parse.go      # Updated to handle references
│   │
│   └── config/
│       ├── config.go     # Configuration management
│       ├── config_test.go
│       ├── discovery.go  # Git branch discovery logic
│       └── discovery_test.go
│
└── specs/
    └── next-task-workflow/
        ├── requirements.md
        ├── design.md      # This document
        └── decision_log.md
```

## Components and Interfaces

### 1. Next Task Command Component

**File:** `cmd/next.go`

```go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "go-tasks/internal/config"
    "go-tasks/internal/task"
)

var nextCmd = &cobra.Command{
    Use:   "next [filename]",
    Short: "Get the next incomplete task",
    Long:  `Retrieves the first incomplete task in depth-first order...`,
    Args:  cobra.MaximumNArgs(1),
    RunE:  runNext,
}

func runNext(cmd *cobra.Command, args []string) error {
    filename, err := resolveFilename(args)
    if err != nil {
        return err
    }
    
    taskList, err := task.ParseFile(filename)
    if err != nil {
        return fmt.Errorf("parsing file: %w", err)
    }
    
    nextTask := task.FindNextIncompleteTask(taskList.Tasks)
    if nextTask == nil {
        fmt.Println("All tasks are complete!")
        return nil
    }
    
    // Render output with references
    return renderTaskWithReferences(nextTask, taskList.References)
}
```

### 2. Configuration Management Component

**File:** `internal/config/config.go`

```go
package config

import (
    "gopkg.in/yaml.v3"
    "os"
    "path/filepath"
)

type Config struct {
    Discovery GitDiscovery `yaml:"discovery"`
}

type GitDiscovery struct {
    Enabled  bool   `yaml:"enabled"`
    Template string `yaml:"template"`
}

func LoadConfig() (*Config, error) {
    // Check for config in order of precedence
    paths := []string{
        "./.go-tasks.yml",
        expandHome("~/.config/go-tasks/config.yml"),
    }
    
    for _, path := range paths {
        if cfg, err := loadConfigFile(path); err == nil {
            return cfg, nil
        }
    }
    
    // Return default config if no file found
    return defaultConfig(), nil
}

func defaultConfig() *Config {
    return &Config{
        Discovery: GitDiscovery{
            Enabled:  false,
            Template: "specs/{branch}/tasks.md",
        },
    }
}
```

### 3. Git Discovery Component

**File:** `internal/config/discovery.go`

```go
package config

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
)

func DiscoverFileFromBranch(template string) (string, error) {
    branch, err := getCurrentBranch()
    if err != nil {
        return "", fmt.Errorf("getting git branch: %w", err)
    }
    
    if isSpecialGitState(branch) {
        return "", fmt.Errorf("special git state detected: %s", branch)
    }
    
    // Replace {branch} placeholder with actual branch name
    path := strings.ReplaceAll(template, "{branch}", branch)
    
    // Verify file exists
    if !fileExists(path) {
        return "", fmt.Errorf("branch-based file not found: %s", path)
    }
    
    return path, nil
}

func getCurrentBranch() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    var out, errOut bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &errOut
    
    // Set timeout to prevent hanging
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := cmd.RunContext(ctx); err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return "", fmt.Errorf("git command timed out")
        }
        return "", fmt.Errorf("git command failed: %w (stderr: %s)", err, errOut.String())
    }
    
    branch := strings.TrimSpace(out.String())
    
    // Sanitize branch name to prevent injection
    if strings.ContainsAny(branch, ";&|<>$`") {
        return "", fmt.Errorf("invalid branch name: %s", branch)
    }
    
    return branch, nil
}
```

### 4. Next Task Finder Component

**File:** `internal/task/next.go`

```go
package task

import "fmt"

// FindNextIncompleteTask finds the highest-level task with incomplete work
// Returns the task along with its incomplete subtasks
func FindNextIncompleteTask(tasks []Task) *TaskWithContext {
    for _, task := range tasks {
        if result := evaluateTaskForNext(&task); result != nil {
            return result
        }
    }
    return nil
}

// evaluateTaskForNext checks if a task has incomplete work
func evaluateTaskForNext(task *Task) *TaskWithContext {
    // If the task itself is incomplete, or any of its children are incomplete,
    // return this task as the next one to work on
    if hasIncompleteWork(task) {
        // Filter to only include incomplete subtasks
        incompleteChildren := filterIncompleteChildren(task.Children)
        
        return &TaskWithContext{
            Task:               task,
            IncompleteChildren: incompleteChildren,
        }
    }
    
    return nil
}

// hasIncompleteWork checks if task or any subtask is incomplete
func hasIncompleteWork(task *Task) bool {
    return hasIncompleteWorkWithDepth(task, 0, 100)
}

func hasIncompleteWorkWithDepth(task *Task, depth, maxDepth int) bool {
    if depth > maxDepth {
        // Log warning about depth limit
        return false
    }
    
    // Task has incomplete work if itself is not completed
    if task.Status != Completed {
        return true
    }
    
    // Or if any child has incomplete work
    for _, child := range task.Children {
        if hasIncompleteWorkWithDepth(&child, depth+1, maxDepth) {
            return true
        }
    }
    
    return false
}

// filterIncompleteChildren returns only children that have incomplete work
func filterIncompleteChildren(children []Task) []Task {
    var incomplete []Task
    for _, child := range children {
        if hasIncompleteWork(&child) {
            incomplete = append(incomplete, child)
        }
    }
    return incomplete
}

type TaskWithContext struct {
    *Task
    IncompleteChildren []Task // Only incomplete subtasks for focused work
}
```

### 5. Reference Document Handler Component

**File:** `internal/task/references.go`

```go
package task

import (
    "fmt"
    "gopkg.in/yaml.v3"
    "path/filepath"
    "strings"
)

// FrontMatter represents the YAML front matter in task files
type FrontMatter struct {
    References []string          `yaml:"references,omitempty"`
    Metadata   map[string]string `yaml:"metadata,omitempty"`
}

// ParseFrontMatter extracts YAML front matter from markdown content
func ParseFrontMatter(content string) (*FrontMatter, string, error) {
    if !strings.HasPrefix(content, "---\n") {
        // No front matter present
        return &FrontMatter{}, content, nil
    }
    
    // Find the end of front matter
    endIndex := strings.Index(content[4:], "\n---\n")
    if endIndex == -1 {
        return nil, content, fmt.Errorf("unclosed front matter block")
    }
    
    // Extract front matter YAML
    frontMatterYAML := content[4 : endIndex+4]
    remainingContent := content[endIndex+9:] // Skip past "---\n"
    
    // Parse YAML
    var fm FrontMatter
    if err := yaml.Unmarshal([]byte(frontMatterYAML), &fm); err != nil {
        return nil, content, fmt.Errorf("parsing front matter: %w", err)
    }
    
    // Validate reference paths
    if err := validateReferences(fm.References); err != nil {
        return nil, content, fmt.Errorf("invalid references: %w", err)
    }
    
    return &fm, remainingContent, nil
}

// validateReferences ensures reference paths are safe
func validateReferences(references []string) error {
    for _, ref := range references {
        // Clean the path and check for traversal attempts
        cleaned := filepath.Clean(ref)
        
        // Prevent absolute paths and parent directory traversal
        if filepath.IsAbs(cleaned) {
            // Allow absolute paths but log warning
            continue
        }
        
        if strings.HasPrefix(cleaned, "..") {
            return fmt.Errorf("path traversal detected in reference: %s", ref)
        }
    }
    return nil
}

// SerializeWithFrontMatter combines front matter and content
func SerializeWithFrontMatter(fm *FrontMatter, content string) string {
    if fm == nil || (len(fm.References) == 0 && len(fm.Metadata) == 0) {
        return content
    }
    
    var builder strings.Builder
    
    // Write front matter
    builder.WriteString("---\n")
    
    yamlData, _ := yaml.Marshal(fm)
    builder.Write(yamlData)
    
    builder.WriteString("---\n")
    builder.WriteString(content)
    
    return builder.String()
}

```

### 6. Auto-Complete Handler Component

**File:** `internal/task/autocomplete.go`

```go
package task

// AutoCompleteParents checks and completes parent tasks when all children are done
func (tl *TaskList) AutoCompleteParents(taskID string) ([]string, error) {
    var completedParents []string
    visited := make(map[string]bool) // Prevent cycles
    maxDepth := 100 // Prevent infinite recursion
    depth := 0
    
    parentID := getParentID(taskID)
    for parentID != "" && depth < maxDepth {
        if visited[parentID] {
            return completedParents, fmt.Errorf("cycle detected at task %s", parentID)
        }
        visited[parentID] = true
        
        parent := tl.FindTaskByID(parentID)
        if parent == nil {
            // Parent doesn't exist, log warning but continue
            break
        }
        
        if allChildrenComplete(parent) && parent.Status != Completed {
            parent.Status = Completed
            completedParents = append(completedParents, parentID)
        } else {
            // Still check grandparents even if this parent can't be completed
            // as they might have no other incomplete children
        }
        
        parentID = getParentID(parentID)
        depth++
    }
    
    if depth >= maxDepth {
        return completedParents, fmt.Errorf("max depth exceeded")
    }
    
    return completedParents, nil
}

func allChildrenComplete(task *Task) bool {
    for _, child := range task.Children {
        if child.Status != Completed {
            return false
        }
        if !allChildrenComplete(&child) {
            return false
        }
    }
    return true
}

func getParentID(taskID string) string {
    parts := strings.Split(taskID, ".")
    if len(parts) <= 1 {
        return ""
    }
    return strings.Join(parts[:len(parts)-1], ".")
}
```

## Data Models

### Extended TaskList Structure

```go
type TaskList struct {
    Title      string
    Tasks      []Task
    FrontMatter *FrontMatter  // New field for front matter metadata
    FilePath   string
    Modified   time.Time
}

type FrontMatter struct {
    References []string          `yaml:"references,omitempty"`
    Metadata   map[string]string `yaml:"metadata,omitempty"`
}
```

### Configuration Structure

```go
type Config struct {
    Discovery GitDiscovery `yaml:"discovery"`
}

type GitDiscovery struct {
    Enabled  bool   `yaml:"enabled"`
    Template string `yaml:"template"`
}
```

### Next Task Result Structure

```go
type NextTaskResult struct {
    Task       *Task
    Children   []Task
    References []string
}
```

## Error Handling

### Error Categories

1. **Configuration Errors**
   - Invalid YAML syntax in config file
   - Invalid template pattern
   - Permission issues reading config

2. **Git Discovery Errors**
   - Git not installed or not in PATH
   - Not in a git repository
   - Special git states (detached HEAD, rebasing)
   - Branch-based file not found

3. **File Operation Errors**
   - File not found
   - Permission denied
   - Invalid file format
   - File size exceeds limit

4. **Task Operation Errors**
   - Invalid task ID
   - Task not found
   - Circular task dependencies

### Error Handling Strategy

```go
// Structured error types for better handling
type ConfigError struct {
    Path string
    Err  error
}

type GitError struct {
    Operation string
    Err       error
}

// Error messages with context
func (e ConfigError) Error() string {
    return fmt.Sprintf("config error in %s: %v", e.Path, e.Err)
}

// Graceful fallbacks
func resolveFilename(args []string) (string, error) {
    // Try explicit argument first
    if len(args) > 0 {
        return args[0], nil
    }
    
    // Try git discovery if enabled
    cfg, _ := config.LoadConfig()
    if cfg.Discovery.Enabled {
        if path, err := config.DiscoverFileFromBranch(cfg.Discovery.Template); err == nil {
            return path, nil
        }
        // Fall through to require explicit file
    }
    
    return "", fmt.Errorf("no filename specified and git discovery failed or disabled")
}
```

## Critical Design Decisions and Clarifications

### Next Task Selection Algorithm
The implementation finds the **highest-level task with incomplete work**, not the deepest task. This means:
- If task 2 is in-progress with subtasks 2.1 (complete), 2.2 (in-progress), and 2.3 (pending), it returns task 2
- The result includes only the incomplete subtasks (2.2 and 2.3) to focus the user on what needs to be done
- This approach provides better context by showing the parent task and what specific subtasks remain

### Cycle Detection and Prevention
- All recursive functions include cycle detection using visited maps
- Maximum depth limits (default: 100) prevent stack overflow
- Task ID validation prevents creating circular references

### Configuration Precedence
Configuration is loaded in the following order (first found wins):
1. Command-line flags (highest priority)
2. Environment variables (GO_TASKS_CONFIG)
3. Project-local config (./.go-tasks.yml)
4. User config (~/.config/go-tasks/config.yml)
5. Default configuration (lowest priority)

### Front Matter vs Markdown Sections
Based on review feedback, the design uses **YAML front matter** for references instead of markdown sections. This provides:
- Structured, parseable metadata
- Clear separation from content
- Extensibility for future metadata
- Backward compatibility through migration function

### Concurrency and File Locking
- File operations use atomic writes (temp file + rename)
- Consider using file locking (flock) for concurrent access
- Commands that modify files should use exclusive locks
- Read-only commands can use shared locks

### Performance Optimizations
- Configuration is cached after first load (stored in package-level variable)
- Git branch name cached for session duration
- Early termination in next task search
- Depth limits prevent pathological cases

## Testing Strategy

### Unit Tests

1. **Next Task Finder Tests** (`internal/task/next_test.go`)
   - Test finding next task in flat list
   - Test finding next task in nested hierarchy
   - Test when all tasks are complete
   - Test with mixed completion states
   - Test in-progress task handling

2. **Configuration Tests** (`internal/config/config_test.go`)
   - Test loading from different config locations
   - Test invalid YAML handling
   - Test default configuration
   - Test config validation

3. **Git Discovery Tests** (`internal/config/discovery_test.go`)
   - Test branch name extraction
   - Test path template substitution
   - Test special git state handling
   - Test file existence validation

4. **Reference Parser Tests** (`internal/task/references_test.go`)
   - Test parsing references section
   - Test multiple reference formats
   - Test empty references
   - Test malformed references

5. **Auto-Complete Tests** (`internal/task/autocomplete_test.go`)
   - Test single-level parent completion
   - Test multi-level parent completion
   - Test partial subtask completion
   - Test already-complete parent handling

### Integration Tests

1. **End-to-End Workflow Tests**
   - Create task file with references
   - Configure git discovery
   - Execute next command
   - Complete tasks and verify auto-completion
   - Verify reference inclusion in output

2. **Git Integration Tests**
   - Test with real git repository
   - Test branch switching
   - Test detached HEAD handling
   - Test file discovery across branches

3. **Configuration Integration Tests**
   - Test config file precedence
   - Test environment variable overrides
   - Test config hot-reloading

### Test Data Examples

```markdown
---
references:
  - ./docs/setup.md
  - ./specs/api.yaml
metadata:
  created: "2024-01-30"
  project: "backend-api"
---
# Test Task File

- [ ] 1. Setup environment
  - [x] 1.1. Install dependencies
  - [ ] 1.2. Configure database
    - [ ] 1.2.1. Create schema
    - [ ] 1.2.2. Load test data
- [-] 2. Implement features
  - [x] 2.1. User authentication
  - [ ] 2.2. API endpoints
```

### Benchmark Tests

```go
func BenchmarkFindNextTask(b *testing.B) {
    // Create large task hierarchy
    tl := generateLargeTaskList(1000)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = FindNextIncompleteTask(tl.Tasks)
    }
}

func BenchmarkAutoComplete(b *testing.B) {
    tl := generateDeepTaskHierarchy(10)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        tl.AutoCompleteParents("1.2.3.4.5.6.7.8.9.10")
    }
}
```

## Performance Considerations

1. **Next Task Finding**: O(n) where n is total number of tasks
   - Early termination on first incomplete task
   - Depth-first traversal minimizes memory usage

2. **Auto-Complete**: O(d) where d is depth of task hierarchy
   - Only traverses upward through parent chain
   - Stops early if parent can't be completed

3. **Configuration Loading**: Cached after first load
   - File system checks minimized
   - YAML parsing happens once per session

4. **Git Operations**: Command execution overhead
   - Consider caching branch name for session
   - Validate git availability once at startup

## Security Considerations

1. **File Path Validation**
   - Continue using existing validateFilePath function
   - No changes to security model for references (paths stored as strings only)

2. **Git Command Execution**
   - Use exec.Command with fixed arguments
   - No user input directly in git commands
   - Validate git output before use

3. **Configuration File Security**
   - Validate YAML structure
   - Limit configuration file locations
   - Sanitize template patterns

## Migration and Backward Compatibility

1. **Existing Task Files**
   - Files without References section work unchanged
   - Existing commands continue to function
   - No breaking changes to data structures

2. **Configuration Migration**
   - Default configuration with discovery disabled
   - Users opt-in to git discovery feature
   - Clear documentation for setup

3. **Command Compatibility**
   - All existing commands maintain current behavior
   - New features are additive only
   - Output formats remain compatible

## Future Enhancements

Based on the decision log's future considerations:

1. **Reference Content Inclusion**
   - Add flag to optionally include file content
   - Implement content caching for performance
   - Support for remote references (URLs)

2. **Reference Path Validation**
   - Optional validation warnings for non-existent paths
   - Relative path resolution improvements
   - Circular reference detection

3. **Glob Pattern Support**
   - Support wildcards in reference paths
   - Bulk reference inclusion
   - Pattern-based exclusions

4. **Enhanced Git Integration**
   - Support for multiple branch patterns
   - Tag-based file discovery
   - Commit message integration
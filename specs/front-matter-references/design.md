# Front Matter References Feature Design

## Overview

This feature extends the rune CLI with the ability to add YAML front matter content through command-line operations. The design focuses on two primary command enhancements: extending the `create` command to accept front matter during file creation, and adding a new `add-frontmatter` command for modifying existing files.

The implementation leverages and extends the existing front matter infrastructure in the codebase, which already handles parsing and rendering YAML front matter through the `FrontMatter` struct and the `ParseFrontMatter`/`SerializeWithFrontMatter` functions in the `internal/task` package.

## Architecture

### System Context

The feature integrates into the existing command structure following the established Cobra CLI pattern:
- Commands are defined in individual files within the `cmd/` package
- Business logic resides in the `internal/task/` package
- Front matter handling uses the existing `gopkg.in/yaml.v3` dependency

### Component Interaction Flow

```
User Input → Cobra Command → Validation → Task Package Operations → File System
                ↓                              ↓
           Flag Parsing              FrontMatter Struct Manipulation
                                              ↓
                                     YAML Serialization
```

## Components and Interfaces

### 1. Enhanced Create Command (`cmd/create.go`)

**Modifications:**
- Add new flag handlers for `--reference` and `--meta`
- Parse and validate input flags during command initialization
- Pass front matter data to TaskList creation

**New Flags:**
```go
var (
    createReferences []string  // Repeatable flag for references
    createMetadata   []string  // Repeatable flag for metadata in "key:value" format
)
```

**Flag Registration:**
```go
createCmd.Flags().StringSliceVar(&createReferences, "reference", []string{}, "add reference paths (repeatable)")
createCmd.Flags().StringSliceVar(&createMetadata, "meta", []string{}, "add metadata in key:value format (repeatable)")
```

### 2. New Add-FrontMatter Command (`cmd/add_frontmatter.go`)

**Structure:**
```go
var addFrontMatterCmd = &cobra.Command{
    Use:   "add-frontmatter [file] [flags]",
    Short: "Add front matter content to a task file",
    Long:  `Add or merge front matter content (references and metadata) to an existing task file.`,
    Args:  cobra.ExactArgs(1),
    RunE:  runAddFrontMatter,
}
```

**Flags:**
```go
var (
    addFMReferences []string  // References to add
    addFMMetadata   []string  // Metadata to add
)
```

**Dry-Run Support:**
The command respects the global `--dry-run` flag. In dry-run mode:
- Shows what would be added/merged
- Displays the resulting front matter structure
- Does not modify the file

### 3. Task Package Extensions (`internal/task/`)

**Modified Functions in `operations.go`:**
```go
// Extend existing NewTaskList to accept optional front matter
func NewTaskList(title string, frontMatter ...*FrontMatter) *TaskList {
    tl := &TaskList{
        Title:    title,
        Tasks:    []Task{},
        Modified: time.Now(),
    }
    if len(frontMatter) > 0 && frontMatter[0] != nil {
        tl.FrontMatter = frontMatter[0]
    }
    return tl
}

// AddFrontMatterContent merges new front matter content into existing
func (tl *TaskList) AddFrontMatterContent(references []string, metadata map[string]any) error
```

**New Functions in `frontmatter.go`:**
```go
// ParseMetadataFlags converts "key:value" strings to map[string]any with validation
func ParseMetadataFlags(flags []string) (map[string]any, error)

// MergeFrontMatter implements the merge strategy for combining front matter
func MergeFrontMatter(existing, new *FrontMatter) (*FrontMatter, error)

// ValidateMetadataKey ensures metadata keys are valid YAML identifiers
func ValidateMetadataKey(key string) error
```

### 4. Front Matter Merge Logic

**Merge Strategy Implementation:**

```go
// MergeFrontMatter algorithm:
func MergeFrontMatter(existing, new *FrontMatter) (*FrontMatter, error) {
    result := &FrontMatter{
        References: existing.References,
        Metadata:   make(map[string]any),
    }
    
    // Deep copy existing metadata
    for k, v := range existing.Metadata {
        result.Metadata[k] = v
    }
    
    // Append new references (no deduplication per decision log)
    result.References = append(result.References, new.References...)
    
    // Merge metadata with type-aware logic
    for key, newValue := range new.Metadata {
        if existingValue, exists := result.Metadata[key]; exists {
            merged, err := mergeValues(existingValue, newValue)
            if err != nil {
                return nil, fmt.Errorf("merge conflict for key %s: %w", key, err)
            }
            result.Metadata[key] = merged
        } else {
            result.Metadata[key] = newValue
        }
    }
    
    return result, nil
}

// Type-aware value merging:
// - Same types: Arrays append, maps merge recursively, scalars replace
// - Different types: Return error (no automatic conversion)
// - Maximum nesting depth: 3 levels
```

## Data Models

### Extended FrontMatter Usage

The existing `FrontMatter` struct remains unchanged:
```go
type FrontMatter struct {
    References []string       `yaml:"references,omitempty"`
    Metadata   map[string]any `yaml:"metadata,omitempty"`
}
```

### Command Input Validation

**Reference Validation:**
- Accept any string as a reference path (per decision log)
- No file existence checking
- No path normalization

**Metadata Validation:**
- Must follow "key:value" format with validation
- Support nested keys using dot notation: "parent.child:value" (max 3 levels)
- Value parsing rules:
  - Keep values as strings by default (no automatic type inference)
  - Arrays: Only when explicitly using multiple flags with same key
  - Example: `--meta "tags:todo" --meta "tags:urgent"` creates array
  - Single flag with commas stays as string: `--meta "desc:a, b, c"`

**Metadata Parsing Algorithm:**
```go
func ParseMetadataFlags(flags []string) (map[string]any, error) {
    metadata := make(map[string]any)
    
    for _, flag := range flags {
        colonIdx := strings.Index(flag, ":")
        if colonIdx == -1 {
            return nil, fmt.Errorf("invalid format '%s': missing colon", flag)
        }
        
        key := flag[:colonIdx]
        value := flag[colonIdx+1:]
        
        if key == "" {
            return nil, fmt.Errorf("empty key in '%s'", flag)
        }
        
        if err := ValidateMetadataKey(key); err != nil {
            return nil, err
        }
        
        // Handle dot notation for nested keys
        if strings.Contains(key, ".") {
            if err := setNestedValue(metadata, key, value); err != nil {
                return nil, err
            }
        } else {
            // Handle multiple values for same key (creates array)
            if existing, exists := metadata[key]; exists {
                metadata[key] = appendToValue(existing, value)
            } else {
                metadata[key] = value  // Keep as string
            }
        }
    }
    
    return metadata, nil
}
```

## Error Handling and Security

### Input Validation
```go
// Metadata key validation
func ValidateMetadataKey(key string) error {
    // Prevent YAML-breaking characters
    if strings.ContainsAny(key, ":\n\r\t") {
        return fmt.Errorf("invalid characters in metadata key: %s", key)
    }
    // Limit nesting depth for simplicity
    if strings.Count(key, ".") > 2 {  // Max 3 levels
        return fmt.Errorf("metadata nesting too deep (max 3 levels): %s", key)
    }
    // Validate each segment is a valid YAML key
    for _, segment := range strings.Split(key, ".") {
        if !isValidYAMLKey(segment) {
            return fmt.Errorf("invalid YAML key segment: %s", segment)
        }
    }
    return nil
}

// No validation for reference paths - paths are stored as-is
// File existence and path safety are user responsibilities
```

### Resource Limits
- Maximum front matter size: 64KB
- Maximum metadata nesting: 3 levels  
- Maximum number of references: 100
- Maximum metadata entries: 100

### File Operation Safety
```go
// Atomic write implementation
func (tl *TaskList) WriteFileAtomic(filepath string) error {
    // Validate file is managed by rune
    if !strings.HasSuffix(filepath, ".md") {
        return fmt.Errorf("only .md files are supported")
    }
    
    // Write to temporary file first
    tempFile := filepath + ".tmp"
    content := RenderMarkdown(tl)
    
    if err := os.WriteFile(tempFile, content, 0644); err != nil {
        os.Remove(tempFile)
        return fmt.Errorf("failed to write temporary file: %w", err)
    }
    
    // Atomic rename
    if err := os.Rename(tempFile, filepath); err != nil {
        os.Remove(tempFile)
        return fmt.Errorf("failed to save file: %w", err)
    }
    
    return nil
}
```

### Error Types
```go
type FrontMatterError struct {
    Op      string // "parse", "merge", "validate", "write"
    Path    string
    Message string
    Err     error
}

func (e *FrontMatterError) Error() string {
    if e.Path != "" {
        return fmt.Sprintf("%s %s: %s: %v", e.Op, e.Path, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
}
```

## Determining rune Managed Files

A file is considered "managed by rune" when:
1. It has a `.md` extension
2. It exists within the current working directory or subdirectories
3. No additional validation is performed on content (per minimal error handling decision)

## Testing Strategy

### Unit Tests

**Create Command Tests (`cmd/create_test.go`):**
- Test with single reference flag
- Test with multiple reference flags
- Test with single metadata flag
- Test with multiple metadata flags
- Test with combined references and metadata
- Test with invalid metadata format
- Test dry-run mode with front matter

**Add-FrontMatter Command Tests (`cmd/add_frontmatter_test.go`):**
- Test adding references to file without front matter
- Test adding references to file with existing front matter
- Test adding metadata to file without front matter
- Test merging metadata with existing values
- Test array append behavior for metadata
- Test with non-existent file
- Test with invalid file path

**Task Package Tests (`internal/task/operations_test.go`):**
- Test `NewTaskListWithFrontMatter` function
- Test `AddFrontMatterContent` method
- Test `ParseMetadataFlags` helper
- Test merge logic for various data types

### Integration Tests

**End-to-End Scenarios (`cmd/integration_test.go`):**
- Create file with front matter, verify content
- Add front matter to existing file, verify merge
- Complex metadata structures with nested values
- Large front matter content handling
- Atomic write verification
- YAML structure validation testing

**Validation Tests:**
```go
func TestYAMLKeyValidation(t *testing.T) {
    // Test that invalid YAML keys are rejected
}

func TestResourceLimits(t *testing.T) {
    // Test maximum front matter size enforcement
}
```

**Note:** No performance benchmarking is required for this feature. The operations are simple string manipulations and YAML parsing using the existing library.

### Test Data Examples

**Valid Input:**
```bash
# Create with references
rune create tasks.md --title "Project" --reference "doc.md" --reference "spec.md"

# Create with metadata
rune create tasks.md --title "Project" --meta "author:John" --meta "version:1.0"

# Add to existing
rune add-frontmatter tasks.md --reference "new.md" --meta "tags:todo,urgent"
```

**Expected Output:**
```yaml
---
references:
  - doc.md
  - spec.md
metadata:
  author: John
  version: 1.0
  tags: [todo, urgent]
---
# Project

```

## Implementation Considerations

### Performance
- Minimal overhead for files without front matter
- Efficient YAML parsing using existing library
- In-memory operations for merge logic
- No performance benchmarking required - operations are simple string manipulations

### Compatibility
- Backward compatible with existing files
- No changes to existing command behavior without new flags
- Preserves existing front matter structure
- Integration with existing `WriteFile` method through atomic writes

### Safety
- YAML structure validation through key validation
- Resource limits for reasonable operation
- Atomic file operations to prevent corruption
- No file path validation - reference paths are stored as provided

### Success Feedback
Per requirement 2.8, provide clear feedback:
```go
// For create command
fmt.Printf("Created: %s\n", filename)
if len(references) > 0 {
    fmt.Printf("  Added %d reference(s)\n", len(references))
}
if len(metadata) > 0 {
    fmt.Printf("  Added %d metadata field(s)\n", len(metadata))
}

// For add-frontmatter command  
fmt.Printf("Updated: %s\n", filename)
fmt.Printf("  Added %d reference(s)\n", len(newRefs))
fmt.Printf("  Merged %d metadata field(s)\n", len(newMeta))
```

### Future Extensibility
- Generic metadata support allows future front matter additions
- Modular merge logic can be extended for new strategies
- Flag pattern can be reused for other commands
- Consider schema-based validation in future versions

## Design Decisions and Rationales

### Decision: Use StringSlice Flags Instead of Custom Parsing
**Rationale:** Cobra's StringSlice provides robust handling of repeated flags, avoiding shell escaping issues with bracket notation.

### Decision: Separate Commands for Create vs Add
**Rationale:** Maintains single responsibility principle and clear command semantics. Create initializes files, add-frontmatter modifies existing files.

### Decision: Merge Strategy Over Replace/Error
**Rationale:** Per decision log, users prefer additive behavior. Merging prevents data loss and supports incremental updates.

### Decision: Generic Metadata with Type Inference
**Rationale:** Provides flexibility for future use cases without requiring schema changes. Type inference maintains usability.

### Decision: No Reference Deduplication
**Rationale:** Preserves user intent and order. While duplicate strings may seem redundant, deduplication would require additional logic and could interfere with potential future features that attach context to references.

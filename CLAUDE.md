# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

rune is a Go command-line tool for managing hierarchical markdown task lists, optimized for AI agents and developers. It provides consistent markdown formatting, a JSON API for batch operations, and comprehensive task management capabilities.

## Development Commands

### Building and Testing

```bash
# Run complete validation (format, lint, and tests)
make check

# Run unit tests
make test

# Run integration tests (requires INTEGRATION=1 environment variable)
make test-integration

# Run all tests (unit and integration)
make test-all

# Generate test coverage report
make test-coverage

# Run benchmarks
make benchmark

# Run a single test
go test -run TestSpecificFunction ./cmd

# Run single integration test
INTEGRATION=1 go test -run TestIntegrationWorkflows/batch_operations_complex -v ./cmd
```

### Code Quality

```bash
# Format all Go code
make fmt

# Run golangci-lint
make lint

# Apply modernize tool fixes (updates Go patterns to latest standards)
make modernize

# Clean up dependencies
make mod-tidy
```

## Architecture

### Package Structure

- `main.go` - Entry point, calls cmd.Execute()
- `cmd/` - CLI commands using Cobra framework
  - Each command (create, list, add, etc.) has its own file with corresponding test file
  - `root.go` contains global flags and root command setup
- `internal/task/` - Core business logic
  - `task.go` - Task and TaskList structs with validation
  - `parse.go` - Markdown parsing functions
  - `render.go` - Output rendering (markdown, JSON, table)
  - `operations.go` - Task mutations (add, remove, update)
  - `search.go` - Query and filtering capabilities
  - `batch.go` - Batch operation execution

### Key Design Principles

1. **Parser Behavior**: Reports errors without auto-correction for malformed files
2. **Task States**: Supports three states - Pending `[ ]`, InProgress `[-]`, Completed `[x]`
3. **Hierarchical IDs**: Automatic ID management (1, 1.1, 1.2.1) with renumbering on removal
4. **Atomic Batch Operations**: All operations succeed or all fail, no partial updates
5. **Consistent Formatting**: 2-space indentation, standardized markdown output

### Data Flow

1. CLI commands receive user input via Cobra
2. Commands call internal/task functions for business logic
3. Task operations maintain hierarchical structure in memory
4. Rendering functions produce output in requested format (table/markdown/JSON)
5. File operations handle reading/writing with proper validation

## Testing Strategy

- Unit tests alongside source files (*_test.go)
- Integration tests in `cmd/integration_test.go` (run with INTEGRATION=1)
- Test fixtures in `examples/` directory
- Map-based test tables for clear test case names

## Important Constraints

- Maximum file size: 10MB
- Maximum task title: 500 characters
- Task ID pattern: `^\d+(\.\d+)*$` (e.g., "1", "1.2", "1.2.3")
- File paths must be within working directory (security validation)

## Common Development Tasks

### Adding a New Command

1. Create new file in `cmd/` (e.g., `cmd/newcommand.go`)
2. Define cobra.Command with Use, Short, Long, RunE
3. Add flags in init() function
4. Register with rootCmd in init()
5. Create corresponding test file
6. Update integration tests if needed

### Modifying Task Operations

1. Update methods in `internal/task/operations.go`
2. Ensure ID renumbering works correctly after changes
3. Update validation in `task.go` if needed
4. Add/update tests in `operations_test.go`

## Output Formats

The project uses github.com/ArjenSchwarz/go-output/v2 for table rendering. Supported formats:
- `table` - Human-readable table format
- `markdown` - Consistent markdown formatting
- `json` - Structured JSON output for programmatic use
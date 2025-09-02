# go-tasks Development Instructions

**ALWAYS follow these instructions first and only fallback to additional search and context gathering if the information here is incomplete or found to be in error.**

go-tasks is a Go command-line tool for managing hierarchical markdown task lists, optimized for AI agents and developers. It provides consistent markdown formatting, a JSON API for batch operations, and comprehensive task management capabilities.

## Working Effectively

### Environment Setup and Dependencies
- **Go Version Required**: Go 1.25.0+ (validated working)
- **Make**: Available at `/usr/bin/make` (validated working)
- **Missing Tools** (document as failures):
  - `golangci-lint` - NOT available in environment, `make lint` will fail
  - `modernize` - NOT available in environment, `make modernize` will fail

### Bootstrap and Build Process
Run these commands in order to set up the development environment:

```bash
# Navigate to repository root
cd /home/runner/work/go-tasks/go-tasks

# Download dependencies and build (NEVER CANCEL - takes up to 2 minutes)
go mod tidy  # takes ~1 second

# Format code (NEVER CANCEL - takes up to 2 minutes) 
make fmt     # takes ~2 seconds, downloads dependencies first time

# Build the binary (NEVER CANCEL - takes up to 2 minutes)
go build -o go-tasks  # takes ~0.3 seconds

# Add binary to PATH for testing
export PATH=$PATH:/home/runner/work/go-tasks/go-tasks
```

### Testing Commands
**CRITICAL**: Set timeouts of 60+ minutes for all test commands. NEVER CANCEL builds or tests.

```bash
# Run unit tests (NEVER CANCEL - takes up to 15 minutes)
make test  # takes ~13 seconds, some pre-existing failures are normal

# Run integration tests (NEVER CANCEL - requires binary in PATH, takes up to 15 minutes)
export PATH=$PATH:/home/runner/work/go-tasks/go-tasks
make test-integration  # takes ~1 second, some pre-existing failures are normal

# Run all tests (NEVER CANCEL - takes up to 20 minutes)
make test-all  # combines unit and integration tests

# Generate coverage report (NEVER CANCEL - takes up to 10 minutes)
make test-coverage  # takes ~2 seconds, generates coverage.out file

# Run benchmarks (NEVER CANCEL - takes up to 10 minutes)  
make benchmark  # takes ~1 second

# Clean generated files
make clean  # takes <0.1 seconds
```

### Code Quality Commands
```bash
# Format all Go code (NEVER CANCEL - takes up to 2 minutes)
make fmt  # takes ~2 seconds

# Run linting (WILL FAIL - golangci-lint not available)
make lint  # fails immediately with "No such file or directory"

# Run modernize tool (WILL FAIL - modernize not available)  
make modernize  # fails immediately with "No such file or directory"

# Run complete validation (NEVER CANCEL - takes up to 20 minutes)
make check  # runs fmt, lint (fails), and test - takes ~15 seconds total
```

### Development Workflow
```bash
# See all available make targets
make help

# Clean up dependencies
make mod-tidy  # takes ~1 second
```

## Application Usage and Validation

### Core CLI Functionality
Always test your changes by running these validated scenarios:

```bash
# Ensure binary is in PATH
export PATH=$PATH:/home/runner/work/go-tasks/go-tasks

# Test basic functionality
./go-tasks --help
./go-tasks create test-project.md --title "Test Project"
./go-tasks add test-project.md --title "Setup environment" 
./go-tasks add test-project.md --title "Install deps" --parent 1
./go-tasks list test-project.md --format table
./go-tasks list test-project.md --format json
./go-tasks complete test-project.md 1.1
./go-tasks next test-project.md --format table
./go-tasks find test-project.md --pattern "Setup" --format table
```

### VALIDATION SCENARIOS
After making changes, ALWAYS run through these complete end-to-end scenarios:

1. **Basic Task Management Workflow**:
   - Create a new task file: `./go-tasks create validation.md --title "Validation Test"`
   - Add tasks: `./go-tasks add validation.md --title "Task 1"`
   - Add subtasks: `./go-tasks add validation.md --title "Subtask 1.1" --parent 1`
   - List in different formats: table, JSON, markdown
   - Complete tasks and verify auto-completion of parents
   - Use next command to find incomplete tasks

2. **Search and Filter Operations**:
   - Create tasks with different statuses and content
   - Test find command with various patterns and filters
   - Verify search results in multiple output formats

3. **File Format Validation**:
   - Check that generated markdown follows the format rules
   - Verify hierarchical numbering (1, 1.1, 1.2.1)
   - Ensure 2-space indentation and proper status indicators

### Key File Locations

**Primary Source Code**:
- `main.go` - Entry point
- `cmd/` - CLI commands using Cobra framework
  - `root.go` - Global flags and root command setup
  - Individual command files: `add.go`, `create.go`, `list.go`, etc.
- `internal/task/` - Core business logic
  - `task.go` - Task and TaskList structs
  - `parse.go` - Markdown parsing
  - `render.go` - Output rendering (table, JSON, markdown)
  - `operations.go` - Task mutations
  - `batch.go` - Batch operations

**Configuration**:
- `Makefile` - Development commands and targets
- `.golangci.yml` - Linting configuration (tool not available)
- `go.mod` - Dependencies
- `CLAUDE.md` - Detailed development guidance

**Examples and Tests**:
- `examples/` - Sample task files and batch operations
- `*_test.go` files alongside source code
- `cmd/integration_test.go` - Integration tests

## Important Constraints and Behaviors

### Security and Validation
- **Path Validation**: File paths must be within working directory (no path traversal)
- **File Size Limit**: Maximum 10MB per task file
- **Task Title Limit**: Maximum 500 characters per task title
- **Task ID Pattern**: `^\d+(\.\d+)*$` (e.g., "1", "1.2", "1.2.3")

### Parser Behavior
- Reports errors WITHOUT auto-correction for malformed files
- Atomic batch operations: all succeed or all fail
- Automatic ID renumbering when tasks are removed
- Auto-completion of parent tasks when all children are complete

### Task States
- Pending: `[ ]` (status 0)
- In Progress: `[-]` (status 1)  
- Completed: `[x]` (status 2)

## Common Development Issues

### Known Failures (Do Not Fix These)
- `make lint` fails because golangci-lint is not available
- `make modernize` fails because modernize tool is not available
- Some unit tests fail in `TestListCommandPathValidation` - this is pre-existing
- Integration tests fail without the binary in PATH - ensure `export PATH=$PATH:/home/runner/work/go-tasks/go-tasks`

### Build Issues
- If build fails, ensure Go 1.25.0+ is available
- If tests fail to find `go-tasks` executable, ensure binary is built and in PATH
- Path traversal errors occur when using `/tmp` or absolute paths - use relative paths within project

### Timing Expectations
- **Build**: ~0.3 seconds
- **Unit Tests**: ~13 seconds  
- **Integration Tests**: ~1 second
- **Coverage Report**: ~2 seconds
- **Format**: ~2 seconds (first time with dependency download)
- **Mod Tidy**: ~1 second

### Working with the Codebase

**Adding a New Command**:
1. Create `cmd/newcommand.go` with cobra.Command definition
2. Add flags in init() function  
3. Register with rootCmd in init()
4. Create `cmd/newcommand_test.go` with test cases
5. Update integration tests if command involves file operations

**Modifying Task Operations**:
1. Update methods in `internal/task/operations.go`
2. Ensure ID renumbering works after changes
3. Update validation in `task.go` if needed
4. Add tests in `*_test.go` files
5. Test with real CLI scenarios

### Output Formats
The tool supports three output formats:
- `table` - Human-readable table format (uses go-output/v2 library)
- `markdown` - Consistent markdown formatting with 2-space indentation
- `json` - Structured JSON for programmatic use

Always test your changes in all three output formats to ensure consistency.
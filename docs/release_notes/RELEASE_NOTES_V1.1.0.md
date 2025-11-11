# Rune v1.1.0 Release Notes

Release Date: 2025-11-12

This release adds the `renumber` command for fixing task numbering inconsistencies and introduces an official GitHub Action for easy workflow integration.

## New Features

### Renumber Command

Fix task numbering gaps and inconsistencies with the new `renumber` command:

```bash
rune renumber tasks.md
```

**Key Features:**
- Recalculates all task IDs sequentially while preserving hierarchy
- Automatic `.bak` backup creation before operations for safety
- Phase marker preservation with automatic `AfterTaskID` adjustment
- YAML front matter preservation during renumbering
- Atomic file write operations to prevent data corruption
- Supports table, markdown, and JSON output formats

**Use Cases:**
- After manually reordering tasks in markdown files
- Fixing gaps in task numbering (e.g., 1, 2, 5, 7 → 1, 2, 3, 4)
- Cleaning up task IDs after complex editing operations
- Standardizing numbering after merging multiple task sources

**Example:**
```bash
# Renumber tasks and view summary
rune renumber tasks.md

# Get JSON output
rune renumber tasks.md --format json
```

### GitHub Action

Official GitHub Action for installing rune in your workflows:

```yaml
- name: Setup Rune
  uses: ArjenSchwarz/rune/github-action@v1

- name: Manage tasks
  run: rune list tasks.md
```

**Key Features:**
- Cross-platform support: Linux, macOS, and Windows
- Architecture support: amd64 and arm64
- Automatic caching using GitHub Actions tool-cache for fast installations
- Version resolution with "latest" or specific version support
- MD5 checksum verification for integrity
- Outputs for version and installation path

**Example with Specific Version:**
```yaml
- name: Setup Rune
  uses: ArjenSchwarz/rune/github-action@v1
  with:
    version: '1.0.0'
```

See the [GitHub Action documentation](../../github-action/README.md) for more usage examples and configuration options.

## Improvements

### Test Suite Modernization

The test suite has been refactored to follow Go 2025 best practices:

- Converted all slice-based table tests to map-based table tests for better test isolation and clearer test names
- Split monolithic 4,818-line integration test file into focused test files by feature area:
  - `integration_helpers_test.go` - Shared test setup and helpers
  - `integration_batch_test.go` - Batch operations tests
  - `integration_phase_test.go` - Phase-related workflow tests
  - `integration_renumber_test.go` - Renumber command integration tests
  - `integration_requirements_test.go` - Requirements workflow tests
  - Main `integration_test.go` reduced to core workflow tests
- Updated test variable naming from `tt` to `tc` for consistency
- Improved test maintainability with logical file groupings

## Installation

### Go Install

```bash
go install github.com/ArjenSchwarz/rune@v1.1.0
```

### Build from Source

```bash
git clone https://github.com/ArjenSchwarz/rune.git
cd rune
git checkout v1.1.0
make build
```

### GitHub Actions

```yaml
- uses: ArjenSchwarz/rune/github-action@v1
```

## Upgrade Notes

There are no breaking changes in this release. All existing functionality remains compatible with v1.0.0.

### Important Notes for Renumber Command

- Requirement links (`[Req 1.1]`) in task details are NOT updated automatically - these must be manually fixed if they reference renumbered tasks
- The backup file (`.bak`) is always created for safety - review changes and manually delete if not needed
- If interrupted (Ctrl+C), original file remains intact until atomic write completes

## Dependencies

Several dependencies have updates available (not security-critical):
- `github.com/ArjenSchwarz/go-output/v2`: v2.2.0 → v2.6.0
- `github.com/spf13/cobra`: v1.9.1 → v1.10.1
- `github.com/jedib0t/go-pretty/v6`: v6.4.9 → v6.7.1

These may be updated in a future release.

## Full Changelog

See [CHANGELOG.md](../../CHANGELOG.md) for the complete list of changes.

## Links

- **Repository**: https://github.com/ArjenSchwarz/rune
- **Documentation**: [README.md](../../README.md)
- **Issues**: https://github.com/ArjenSchwarz/rune/issues
- **Discussions**: https://github.com/ArjenSchwarz/rune/discussions

## Contributors

Thanks to everyone who contributed to this release through code, testing, feedback, and documentation improvements.

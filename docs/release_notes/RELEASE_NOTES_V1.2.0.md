# Rune v1.2.0 Release Notes

Release Date: 2026-02-06

This release introduces multi-agent orchestration capabilities: task dependencies, work streams, task ownership, and claiming. It also adds stream-aware phase navigation, a batch add-phase operation, and improved branch discovery.

## New Features

### Task Dependencies

Tasks can now declare dependencies on other tasks. A task with unresolved dependencies is considered "blocked" and won't be returned by the `next` command until all dependencies are completed.

```bash
# Create a task that depends on task 1
rune add tasks.md --title "Build API" --blocked-by "1"

# Multiple dependencies
rune add tasks.md --title "Integration tests" --blocked-by "1,2,3"

# Update existing task
rune update tasks.md 4 --blocked-by "1,2"
```

Dependencies are stored using stable IDs (7-character identifiers) that survive task renumbering. Circular dependencies are automatically detected and rejected.

### Work Streams

Streams partition tasks for parallel execution across multiple agents. Each task can be assigned to a numbered stream.

```bash
# Assign to stream 2
rune add tasks.md --title "Build UI" --stream 2

# View stream status
rune streams tasks.md

# Show only streams with ready tasks
rune streams tasks.md --available

# Filter tasks by stream
rune list tasks.md --stream 2
rune next tasks.md --stream 2
```

Dependencies can cross streams, so a task in stream 2 can depend on a task in stream 1.

### Task Ownership and Claiming

Tasks can be claimed by agents to indicate who is working on them.

```bash
# Claim a task
rune update tasks.md 5 --owner "agent-1"

# Release a task
rune update tasks.md 5 --release

# Atomic claim: sets in-progress + owner in one operation
rune next tasks.md --claim "agent-1"

# Claim all ready tasks in a stream
rune next tasks.md --stream 2 --claim "agent-1"
```

### Stream-Aware Phase Navigation

The `next --phase` command now supports stream filtering to find the first phase with ready work in a specific stream:

```bash
# Find first phase with ready stream 2 tasks
rune next tasks.md --phase --stream 2

# Claim ready stream 2 tasks from the appropriate phase
rune next tasks.md --phase --stream 2 --claim "agent-1"
```

Output includes blocking status indicators in all formats (JSON, table, markdown).

### Batch Add-Phase Operation

The batch JSON API now supports creating phase headers programmatically:

```json
{
  "type": "add-phase",
  "phase": "Implementation"
}
```

### Consistent JSON Output Format

All commands now include `success` and `count` fields in JSON responses. Verbose output is directed to stderr when using JSON format, keeping stdout clean for programmatic consumption.

## Changes

### Smart Branch Discovery

Branch prefix stripping now uses the **first** slash instead of the last:
- `feature/auth/oauth` strips to `auth/oauth` (previously `oauth`)
- Full branch path is tried as fallback for backward compatibility

### Default Discovery Template

The default template changed from `{branch}/tasks.md` to `specs/{branch}/tasks.md`. Override in `.rune.yml` or `~/.config/rune/config.yml` if needed.

### Conditional Column Display

The `list` command now shows Stream, BlockedBy, and Owner columns only when relevant data exists in the file.

## Bug Fixes

- Phase marker corruption when removing tasks from phase-based files
- Batch remove operations now preserve phase boundaries and process in reverse order
- Negative `--stream` flag values rejected with a clear error message

## Installation

### Go Install

```bash
go install github.com/ArjenSchwarz/rune@v1.2.0
```

### Build from Source

```bash
git clone https://github.com/ArjenSchwarz/rune.git
cd rune
git checkout v1.2.0
make install
```

### GitHub Actions

```yaml
- uses: ArjenSchwarz/rune/github-action@v1
```

## Upgrade Notes

### Breaking Change: Default Discovery Template

The default discovery template changed from `{branch}/tasks.md` to `specs/{branch}/tasks.md`. If you rely on the previous default, add this to your `.rune.yml`:

```yaml
discovery:
  template: "{branch}/tasks.md"
```

### Breaking Change: Branch Prefix Stripping

Branch prefix stripping now uses the first slash instead of the last. `feature/auth/oauth` now resolves to `auth/oauth` instead of `oauth`. The full branch name is still tried as a fallback.

### Backward Compatibility

- Existing task files without stable IDs, streams, or dependencies continue to work unchanged
- New metadata fields are parsed leniently; invalid formats are ignored without errors
- Legacy files can be mixed with files using the new features

## Dependencies

- Updated `go-output/v2` from v2.2.0 to v2.6.0
- Updated `cobra` from v1.9.1 to v1.10.2

## Full Changelog

See [CHANGELOG.md](../../CHANGELOG.md) for the complete list of changes.

## Links

- **Repository**: https://github.com/ArjenSchwarz/rune
- **Documentation**: [README.md](../../README.md)
- **Issues**: https://github.com/ArjenSchwarz/rune/issues
- **Discussions**: https://github.com/ArjenSchwarz/rune/discussions

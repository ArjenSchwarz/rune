# Rune v1.3.0 Release Notes

Release Date: 2026-02-09

This release adds the `--one` flag for focused single-task navigation and improves batch command ergonomics. It also includes the project's first contributor guide.

## New Features

### Single Path Filter

The `next` command now supports a `--one` (`-1`) flag that shows only the first incomplete subtask at each level, creating a single path from parent to leaf task. This is useful when agents or developers want to focus on exactly one task without being overwhelmed by the full tree.

```bash
# Show only the single next actionable path
rune next tasks.md --one

# Claim the deepest leaf task in the path
rune next tasks.md --one --claim "agent-1"

# JSON output for scripting
rune next tasks.md --one --format json
```

The `--one` flag is supported in all output formats (table, markdown, JSON) and works with `--claim` to atomically claim the deepest leaf task.

### Contributing Guide

Added `CONTRIBUTING.md` with development workflow, code standards, testing conventions, documentation checklist, and commit message format to help new contributors get started.

## Changes

### Batch Command Positional File Argument

The `batch` command now accepts the target task file as a positional argument when `--input` provides the JSON operations, matching the convention used by all other commands:

```bash
# Previously required file in JSON payload
echo '{"operations": [...]}' | rune batch --input -

# Now supports positional file argument
echo '{"operations": [...]}' | rune batch tasks.md --input -
```

If the positional argument conflicts with a `file` field in the JSON payload, a clear error is returned.

## Installation

### Go Install

```bash
go install github.com/ArjenSchwarz/rune@v1.3.0
```

### Build from Source

```bash
git clone https://github.com/ArjenSchwarz/rune.git
cd rune
git checkout v1.3.0
make install
```

### GitHub Actions

```yaml
- uses: ArjenSchwarz/rune/github-action@v1
```

## Upgrade Notes

There are no breaking changes in this release. All existing functionality remains compatible with v1.2.0.

## Full Changelog

See [CHANGELOG.md](../../CHANGELOG.md) for the complete list of changes.

## Links

- **Repository**: https://github.com/ArjenSchwarz/rune
- **Documentation**: [README.md](../../README.md)
- **Issues**: https://github.com/ArjenSchwarz/rune/issues
- **Discussions**: https://github.com/ArjenSchwarz/rune/discussions

## Contributors

Thanks to @paulgear for contributing the single path filter feature (#30).

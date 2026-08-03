# Rune v1.4.0 Release Notes

Release Date: 2026-08-03

This release adds Homebrew as an installation method and delivers a large batch of bug fixes focused on correctness: phase preservation, task claiming, filtered output consistency, and input validation. Release binaries also now report their actual version.

## Breaking Changes

### Stricter Configuration Validation

`.rune.yml` files are now validated strictly:

- A config file that fails to parse produces an error instead of silently falling back to defaults
- Unknown or misspelled fields are rejected instead of being silently ignored

If rune reports a configuration error after upgrading, check your `.rune.yml` (and `~/.config/rune/config.yml`) against the [configuration schema](../../README.md#configuration) and remove any unsupported fields. This change surfaces typos (e.g. `templat:` instead of `template:`) that previously caused settings to be silently dropped.

## New Features

### Homebrew Install

rune can now be installed via Homebrew on macOS and Linux:

```bash
brew install arjenschwarz/rune/rune
```

Every release publishes SHA256 checksums for all platform tarballs and updates the [Homebrew tap](https://github.com/ArjenSchwarz/homebrew-rune) automatically.

## Bug Fixes

This release contains a large number of correctness fixes. Highlights:

### Version Reporting

Release binaries now report their actual version (`rune --version` previously printed `dev` due to a build configuration issue).

### Phase Preservation

Phase headers are no longer lost or misplaced by `renumber`, batch removes, batch adds to earlier phases, filtered output, files with non-sequential task IDs, or content following a horizontal rule after front matter. Files with CRLF (Windows) line endings now parse correctly, including front matter and phase markers.

### Task Claiming and Dependencies

- Blocked tasks can no longer be returned or claimed by `next`
- `next --phase --claim` claims all ready tasks in the next phase instead of a single task
- `next --claim --one` falls back correctly when the deepest task in the path is blocked
- Stable IDs are auto-assigned when tasks gain `blocked_by` references
- Removing a task cleans up blockers that reference it, and tasks with missing blockers are no longer treated as ready

### Filtered Output Consistency

`list` and `find` produce consistent results across table, markdown, and JSON output when filters are applied: non-matching parents and descendants are excluded, phase headers are retained, and stream, owner, and blocked-by metadata is preserved. `--dry-run` now honours the `--format` flag, and `find --parent ""` / `find --include-parent` work as documented.

### Input Validation and Security

- The 500-character title limit is enforced on all code paths and embedded newlines in titles are rejected
- Invalid indented lines and invalid `--filter` values are reported instead of silently ignored
- File paths that escape the working directory through symlinks are rejected

### Batch Operations and Git Discovery

- `batch --input -` reads JSON from stdin
- Batch remove operations no longer reorder across other operation types, and `details`/`references` are validated before any changes are applied
- Task file auto-detection works from subdirectories and merge/rebase states are no longer misclassified

## Installation

### Homebrew (macOS/Linux)

```bash
brew install arjenschwarz/rune/rune
```

### Go Install

```bash
go install github.com/arjenschwarz/rune@v1.4.0
```

### Build from Source

```bash
git clone https://github.com/ArjenSchwarz/rune.git
cd rune
git checkout v1.4.0
make install
```

### GitHub Actions

```yaml
- uses: ArjenSchwarz/rune/github-action@v1
```

## Upgrade Notes

The only change that may require action is the stricter configuration validation described under Breaking Changes: validate your `.rune.yml` files and remove unsupported fields. All other changes are backwards compatible with v1.3.0.

## Full Changelog

See [CHANGELOG.md](../../CHANGELOG.md) for the complete list of changes.

## Links

- **Repository**: https://github.com/ArjenSchwarz/rune
- **Documentation**: [README.md](../../README.md)
- **Issues**: https://github.com/ArjenSchwarz/rune/issues
- **Discussions**: https://github.com/ArjenSchwarz/rune/discussions

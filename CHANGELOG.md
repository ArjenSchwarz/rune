# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Fixed

- **Dry Run**: `next --claim AGENT_ID --dry-run` no longer writes the task file. The command previously claimed for real — setting the task to in-progress and adding an `Owner:` line — silently defeating `--dry-run`. Dry-run claims now preview only: JSON output includes `"dry_run": true` and markdown/table output is headed "Would Claim Tasks (Dry Run)"
- **Security**: `add-phase` now rejects task files outside the working directory. It read and wrote the target file directly, bypassing the path containment check every other mutating command applies
- **Input Validation**: `rune create --title` now rejects titles that produce an unparseable H1 heading — empty or whitespace-only titles (which render as a bare `# `) and titles containing newlines or other control characters (which split the heading) — as well as titles longer than 500 characters, matching the limit already enforced on task titles
- **Batch Command**: `rune batch` now rejects an unsupported `--format` value before any operation is applied, instead of writing the modified task file to disk and only then failing with `unsupported output format`. The check now runs before the batch input is read, so an unsupported format is reported even when the request JSON or the target file is also invalid
- **Renumber Command**: `renumber --dry-run` no longer writes the renumbered file or creates a `.bak` backup. The command previously ignored the global `--dry-run` flag entirely, so a preview silently mutated the task file and left a stray backup behind. Dry runs now print a format-aware summary and report `"dry_run": true` with no backup file in JSON output
- **Input Validation**: Indented content directly beneath a task that is not a `- ` bullet is now reported as a parse error instead of being silently dropped. This covers free text, a bare `-` with no content, `*` and `+` bullets, `1.` numbered sub-lists, indented code fences, blockquotes, tables and standalone HTML comments — all valid Markdown, but not rune's task format. Previously such lines parsed without complaint but were omitted from output and permanently deleted by any command that rewrote the file. The error is now raised on read, so it surfaces on read-only commands such as `list` and `find` as well as on mutations. Add the missing `- ` prefix to any such line to keep its content. Relatedly, an empty or whitespace-only task detail is now rejected on write, since it would render as a bare `- ` line that rune could not read back.
- **Phase Names**: Phase names containing a newline (or other control character) are now rejected instead of being written verbatim into the `## {name}` header, where they could inject fabricated task lines into the file. All phase entry points — `add-phase`, `add --phase` (including `--dry-run`), and the batch `add-phase` and `add` operations — now trim surrounding whitespace and validate identically, so a name like `"Planning\n"` is trimmed and accepted everywhere and no longer creates a duplicate phase header on the batch `add` path. Tab remains allowed inside a phase name
- **Input Validation**: `rune add` (and `add --phase`, `add` with stream/owner/blocked-by options, and the batch JSON `add`, `add-phase` and `update` operations) now rejects an empty task title instead of writing an unparseable bullet (e.g. `- [ ] 1. `) that a subsequent `list` or `find` then fails to read back. A whitespace-only title is now rejected too; that one did parse, round-tripping as a task whose title is blank spaces, so this is a tightening rather than a corruption fix. The check lives in the shared `validateTaskInput` helper used by every title-setting path, matching the equivalent fix already applied to `create` titles. Note that `update` without a title is unaffected: an empty title is its leave-unchanged sentinel and is filtered out before validation
- **Data Integrity**: `add-phase` now writes through the same atomic temp-file-and-rename path used by every other mutating command, instead of overwriting the target file in place. Previously, a write failure partway through (disk full, quota exceeded, a file-size limit) left the task file truncated, destroying its original content; a failed `add-phase` now leaves the file byte-for-byte unchanged. The shared atomic writer now also restores the target's permissions explicitly before renaming, so a group- or other-writable task file keeps its mode instead of being narrowed by the process umask; this corrects the same latent narrowing on every command that writes a task file. Two smaller behaviour changes come with routing through the shared path: a write into a read-only directory now fails rather than succeeding, and a symlinked task file is replaced rather than written through




## [1.4.0] - 2026-08-03

### Added

- **Homebrew Install**: Install rune with `brew install arjenschwarz/rune/rune`. Releases now publish SHA256 checksums for all platform tarballs and update the Homebrew tap automatically

### Changed

- **Configuration Validation**: Invalid `.rune.yml` files now produce an error instead of being silently ignored, and unknown or misspelled fields are rejected. Remove any unsupported fields from your config to resolve
- **Dependencies**: Updated go-output to v2.7.0 and refreshed all transitive dependencies, resolving a known vulnerability (GO-2026-5764) in an indirect AWS SDK dependency

### Fixed

- **Version Reporting**: Release binaries now report their actual version in `rune --version` instead of `dev`
- **Phase Preservation**: Phase headers are no longer lost or misplaced by `renumber`, batch removes, batch adds to earlier phases, filtered output, files with non-sequential task IDs, or content following a horizontal rule after front matter
- **Windows Line Endings**: Files with CRLF line endings now parse correctly, including front matter and phase markers
- **Task Claiming**: Blocked tasks can no longer be returned or claimed by `next`; `next --phase --claim` claims all ready tasks in the next phase; `next --claim --one` falls back correctly when the deepest task in the path is blocked
- **Next Command**: `next` no longer skips incomplete grandchildren, and checks subtasks under completed parents when determining phase completion
- **Filtered Output**: `list` and `find` now produce consistent results across table, markdown, and JSON output when filters are applied — non-matching parents and descendants are excluded, phase headers are retained, and stream, owner, and blocked-by metadata is preserved
- **JSON Output**: Fixed stale parent IDs after promotion, duplicated matches, stale `available` stream IDs, and a pointer-reuse bug that could corrupt task data; `--dry-run` now honours the `--format` flag
- **Task Dependencies**: Stable IDs are auto-assigned when tasks gain `blocked_by` references, removing a task cleans up blockers that reference it, and tasks with missing blockers are no longer treated as ready
- **Input Validation**: The 500-character title limit is enforced on all code paths, embedded newlines in titles are rejected, invalid indented lines are reported instead of silently ignored, and invalid `--filter` values produce an error instead of matching nothing
- **Security**: File paths that escape the working directory through symlinks are now rejected
- **Batch Operations**: `--input -` reads JSON from stdin, remove operations no longer reorder across other operation types, and `details`/`references` are validated before any changes are applied
- **Git Discovery**: Task file auto-detection works from subdirectories, the discovery timeout is enforced, and merge/rebase states are no longer misclassified
- **Find Command**: `--parent ""` filters top-level tasks and `--include-parent` now works as documented
- **Remove Command**: Reports the correct task title when removing a task after earlier deletions in the same file
- **Documentation**: `go install` instructions use the correct lowercase module path

## [1.3.0] - 2026-02-09

### Added

- **Contributing Guide**: Added `CONTRIBUTING.md` with development workflow, code standards, and testing conventions
- **Single Path Filter**: `next --one` (`-1`) flag shows only the first incomplete subtask at each level, creating a single path from parent to leaf task (thanks @paulgear, #30)
  - Works with `--claim` to claim the deepest leaf task in the path
  - Supported in all output formats (table, markdown, JSON)

### Changed

- **Batch Command**: Positional file argument is now treated as the target task file when `--input` flag provides the JSON operations, matching the convention used by all other commands

## [1.2.0] - 2026-02-06

### Added

- **Task Dependencies**: Tasks can declare dependencies on other tasks using `--blocked-by`
  - Dependencies stored using stable IDs that survive task renumbering
  - Automatic cycle detection prevents circular dependencies
  - Dependent references cleaned up when blocking tasks are removed
  - `next` command only returns "ready" tasks (all blockers completed)

- **Work Streams**: Partition tasks for parallel execution across multiple agents
  - `--stream N` flag on `add`, `update`, and `list` commands
  - `streams` command shows ready, blocked, and active task counts per stream
  - `--available` flag filters to streams with ready tasks
  - Cross-stream dependencies supported

- **Task Ownership**: Claim tasks for specific agents
  - `--owner AGENT_ID` flag on `add` and `update` commands
  - `--release` flag on `update` to clear ownership
  - `--owner` filter on `list` command (empty string for unowned tasks)

- **Task Claiming**: Atomic claim operations for multi-agent coordination
  - `next --claim AGENT_ID` claims the next ready task (sets in-progress + owner)
  - `next --stream N --claim AGENT_ID` claims all ready tasks in a stream
  - `next --phase --stream N --claim AGENT_ID` claims ready stream tasks from the appropriate phase

- **Stream-Aware Phase Navigation**: `next --phase --stream N` finds the first phase with ready tasks in the specified stream
  - Blocking status indicators in all output formats (JSON, table, markdown)
  - Backward compatible with existing `--phase` and `--stream` behaviors

- **Batch Add-Phase Operation**: New `add-phase` operation type for the batch JSON API to create phase headers programmatically

- **Streams Command**: `rune streams [file]` displays work stream status
  - `--available` flag shows only streams with ready tasks
  - `--json` flag outputs structured JSON for scripting

- **Consistent Output Format**: All commands now include `success` and `count` fields in JSON output, with verbose output directed to stderr when using JSON format

- **Install Target**: `make install` installs rune binary to `$GOPATH/bin`

### Changed

- **Dependencies**: Updated `go-output/v2` (v2.2.0 → v2.6.0), `cobra` (v1.9.1 → v1.10.2)
- **Smart Branch Discovery**: Branch prefix stripping now uses the first slash instead of the last
  - `feature/auth/oauth` strips to `auth/oauth` (previously `oauth`)
  - Full branch path is tried as fallback for backward compatibility
- **Default Discovery Template**: Changed from `{branch}/tasks.md` to `specs/{branch}/tasks.md`
  - Users can override in `.rune.yml` or `~/.config/rune/config.yml`
- List command conditionally shows Stream, BlockedBy, and Owner columns only when relevant data exists

### Fixed

- Phase marker corruption when removing tasks from phase-based files
- Batch remove operations now preserve phase boundaries and process in reverse order
- Negative `--stream` flag values now rejected with a clear error message

## [1.1.0] - 2025-11-12

### Added

- **Renumber Command**: Fix task numbering and maintain file consistency
  - Recalculates all task IDs sequentially while preserving hierarchy
  - Automatic backup creation with `.bak` extension before operations
  - Phase marker preservation with automatic `AfterTaskID` adjustment
  - YAML front matter preservation during renumbering
  - Atomic file write operations for data safety
  - Supports table, markdown, and JSON output formats

- **GitHub Action**: Official action for installing rune in GitHub workflows
  - Cross-platform support (Linux, macOS, Windows) with amd64/arm64 architectures
  - Version resolution with "latest" or specific version support
  - Automatic caching using GitHub Actions tool-cache
  - MD5 checksum verification for integrity
  - Outputs for version and installation path

### Changed

- **Test Suite Modernization**: Refactored to follow Go 2025 best practices
  - Converted slice-based table tests to map-based for better isolation
  - Split monolithic integration test file into focused test files by feature
  - Improved test maintainability and clarity

## [1.0.0] - 2025-10-08

### Added

- **Task Requirements Linking**: Link tasks to requirement acceptance criteria
  - `--requirements` flag on `add` and `update` commands for comma-separated requirement IDs
  - `--requirements-file` flag to specify requirements document (defaults to "requirements.md")
  - `--clear-requirements` flag to remove all requirements from a task
  - Requirements rendered as markdown links `[ID](file#ID)` in task output
  - Full batch operation support with requirements validation
  - Round-trip preservation in parse-render cycles

- **Task Phases**: Organize tasks under semantic H2 markdown headers
  - `add-phase` command to add phase headers to task files
  - `--phase` flag on `add` and `next` commands
  - `has-phases` command for programmatic phase detection
  - Phase information displayed in all output formats

- **Front Matter Support**: YAML front matter for metadata and references
  - `--reference` and `--meta` flags on `create` command
  - `add-frontmatter` command for existing files
  - References displayed in all output formats

- **Git Branch Discovery**: Automatic task file location based on git branch
  - Default `{branch}/tasks.md` pattern
  - Configuration via `.rune.yml` or `~/.config/rune/config.yml`

- **Next Task Workflow**: Sequential task management
  - `next` command finds first incomplete task via depth-first traversal
  - Task details and references in output

- **Auto-completion**: Parent tasks complete when all children are done
  - Recursive hierarchy checking
  - Visual feedback with 🎯 emoji

- **Position-based Insertion**: `--position` flag for precise task placement
  - Automatic ID renumbering

- **Core Task Management**: Complete CLI for hierarchical markdown task lists
  - CRUD operations: `create`, `add`, `update`, `remove`
  - Status management: `complete`, `progress`, `uncomplete`
  - Query operations: `list`, `find`
  - Batch operations: atomic multi-operation execution

### Changed

- Improved code organization with consolidated helper functions
- Refactored test suites into focused files by functionality
- Simplified ID validation and task parsing logic

### Security

- File size limit (10MB)
- Path traversal protection
- Input sanitization for null bytes and control characters
- Resource limits (10,000 tasks max, 10 levels deep, 1,000 char details)
- Branch name sanitization
# Bugfix Report: Output Format and Context

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1129, T-1159, T-1409, T-1994, T-1995, T-2022

## Description of the Issue

Six commands dropped context from their output, or ignored the selected `--format`.

| Ticket | Command | Observed | Expected |
|--------|---------|----------|----------|
| T-1129 | `list --format json` on a file with `##` phases | No `requirements_file` | Same `requirements_file` as unphased output |
| T-1995 | `find --format json` | Task `requirements` kept, `requirements_file` dropped | Source file's `requirements_file` included |
| T-1994 | `next --format json` / `--format markdown` | Requirements missing | Task requirements, plus the requirements file in JSON |
| T-2022 | `next --format table` | Subtask details and references missing | Details and references of every listed task |
| T-1159 | `add`/`update`/`remove`/`create` (and `add-phase`/`add-frontmatter`) with `--dry-run --format json` | Plain text | Valid JSON (and markdown for `--format markdown`) |
| T-1409 | `streams --format json`, `version --format json` | Table / plain text | JSON |

**Reproduction steps:** the repros from each ticket, run against the PR117 head (80afbaf). All six reproduced as described.

**Impact:** Medium. API consumers and agents that rely on `--format json` received invalid or incomplete output.

## Investigation Summary

- **Code inspected:** `internal/task/render.go` (`RenderJSONWithPhases`), `cmd/find.go` (`outputSearchResultsJSON`), `cmd/next.go` (all next renderers), `cmd/streams.go`, `cmd/version.go`, and the dry-run branches of `cmd/{add,update,remove,create,add_phase,add_frontmatter}.go`, compared with the format-aware dry-run in `cmd/complete.go`.
- **Hypotheses tested:** The likely causes named in each ticket were confirmed.

## Discovered Root Cause

**Defect type:** Output structs and renderers that were not kept in step with their siblings.

- T-1129: `TaskListJSONWithPhases` had no `RequirementsFile` field, although `TaskListJSON` did.
- T-1995: `outputSearchResultsJSON` built a new `TaskList` with only `Title` and `Tasks`.
- T-1994: the custom `next` serializers copied details and references but not `Requirements`, and had no requirements-file field.
- T-2022: the table renderer built its details/references sections from the root task only.
- T-1159: the dry-run branches predate the format-aware output of the normal paths and always printed text.
- T-1409: `streams` and `version` only read their own `--json` / `--output` flags.

## Resolution for the Issue

**Changes made:**
- `internal/task/render.go` - `TaskListJSONWithPhases` gains `requirements_file`, set from the task list.
- `cmd/find.go` - The source file's `RequirementsFile` is passed into the search-result task list.
- `cmd/next.go` - Each task in the JSON output (plain, `--stream`, `--phase`) gets a `requirements` field, and the response gets a top-level `requirements_file`, matching `list`. Markdown output (plain and `--phase`) gets a `Requirements:` line with `[id](file#id)` links, the format used in task files. The table's `Task Details` / `Task References` sections now cover every listed task and have a `Task` column.
- `cmd/next.go` (claim output, review follow-up) - `next --claim` JSON (`ClaimResponse`) gets `requirements` per claimed task and a top-level `requirements_file`; claim markdown gets a `- Requirements:` line with links. This covers plain, `--stream`, `--phase` and `--phase --stream` claims, and their dry runs. Other claim context (details, references, front matter references) is scoped to T-2184.
- `cmd/{add,update,remove,create,add_phase,add_frontmatter}.go` - Dry-run JSON reuses each command's normal response struct with a new `dry_run` field and a "Would ..." message. Dry-run markdown is a `**Dry run:**` line, as in `complete`. Table output is unchanged. In `update`, building the list of changed fields moved above the dry-run branch so both paths can use it.
- `cmd/streams.go`, `cmd/version.go` - JSON is used when either the command flag or the global `--format json` asks for it.

**Approach rationale:** Each fix reuses the structure the command already has for its non-dry-run or sibling output, so consumers see one schema per command.

**Alternatives considered:**
- T-1159/T-1409: reject `--format json` where it is unsupported. Not chosen because every command involved already has a JSON response.
- T-2022: add Details/References columns to the main table. Not chosen because next-task-workflow 5.5 asks for "additional sections", and multi-line details make wide rows.

## Regression Test

**Test file:** `cmd/output_context_regressions_test.go`

| Test | Ticket |
|------|--------|
| `TestListJSONPhasedIncludesRequirementsFile` | T-1129 |
| `TestFindJSONIncludesRequirementsFile` | T-1995 |
| `TestNextJSONIncludesRequirements` (plain, stream, phase) | T-1994 |
| `TestNextMarkdownIncludesRequirements` (plain, phase) | T-1994 |
| `TestNextClaimIncludesRequirements` (4 claim variants × real/dry-run × JSON/markdown) | T-1994 |
| `TestNextTableIncludesSubtaskMetadata` (child and grandchild) | T-2022 |
| `TestStreamsAndVersionHonourGlobalJSONFormat` | T-1409 |
| `TestDryRunHonoursJSONFormat` (6 commands, also checks no file change), `TestDryRunHonoursMarkdownFormat` | T-1159 |

All 8 original tests failed on 80afbaf and pass after the fix. `TestNextClaimIncludesRequirements`, added after review, failed on 9723f37 in all 16 cases and passes after the claim fix.

**Run command:** `go test ./cmd -run 'TestListJSONPhased|TestFindJSONIncludes|TestNextJSONIncludesReq|TestNextMarkdownIncludesReq|TestNextTableIncludesSub|TestStreamsAndVersion|TestDryRunHonours'`

## Verification

**Automated:**
- [x] Regression tests fail before and pass after the fix
- [x] `make check`: lint 0 issues; only `TestCreateBackup/error_handling_for_read_failures` fails (pre-existing, root ignores file permissions)
- [x] `INTEGRATION=1 go test ./...`: only `TestCreateBackup/...` and `TestIntegrationRenumber/renumber_write_failure` fail, both pre-existing for the same reason

**Manual verification:** re-ran each ticket's repro with the fixed build.

## Prevention

- When a command gains a format-aware output, give its dry-run branch the same treatment.
- Copy `RequirementsFile` along whenever a `TaskList` is rebuilt for rendering.

## Related

- Stacked on PR #117, which is stacked on PR #113. Merge order: #113, #117, then this PR.

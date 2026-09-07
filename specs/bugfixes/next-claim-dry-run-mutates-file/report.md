# Bugfix Report: next-claim-dry-run-mutates-file

**Date:** 2026-09-07
**Status:** Fixed
**Ticket:** T-1398 (duplicate: T-1786)

## Description of the Issue

`rune next --claim <owner> --dry-run` claimed the task for real instead of previewing the claim: it set the task's status to in-progress, wrote an `Owner:` line, and persisted both to the task file, despite `--dry-run` being documented (in `cmd/root.go`) as "preview changes without applying them".

**Reproduction steps:**
1. `cp examples/simple.md /tmp/claim-dry-run.md`
2. Record the file's checksum.
3. Run `go run . next /tmp/claim-dry-run.md --claim agent-a --dry-run --format json`
4. Compare the file checksum or inspect the file.

Observed: the command exits 0, prints a successful claim response, and rewrites the file — task 1 is set to `[-]` with `Owner: agent-a` added.

**Impact:** Silently defeats a documented safety flag. Any agent or script relying on `--dry-run` to preview a claim before committing to it instead had the file mutated for real — this can corrupt task-ownership state that automation assumed was still available to claim.

## Investigation Summary

- **Symptoms examined:** File checksum changed and task state was rewritten even with `--dry-run` passed.
- **Code inspected:** `cmd/next.go` (`runNextWithClaim`), `cmd/root.go` (global `dryRun` flag definition), and the dry-run handling in sibling commands (`cmd/complete.go`, `cmd/remove.go`, `cmd/batch.go`) for the established pattern.
- **Hypotheses tested:** Confirmed via `grep -n dryRun cmd/*.go` that `cmd/next.go` never referenced the global `dryRun` variable at all — `runNextWithClaim` claimed the selected tasks and called `taskList.WriteFile(filename)` unconditionally, regardless of the flag.

## Discovered Root Cause

`runNextWithClaim` in `cmd/next.go` never checked the global `dryRun` flag before mutating claimed tasks in memory and calling `taskList.WriteFile(filename)`. Every other mutating command (`complete`, `remove`, `add`, `update`, `batch`, ...) branches on `dryRun` before writing; `next --claim` was missing that branch entirely.

**Defect type:** Missing conditional / incomplete flag wiring — a new mutating code path (`--claim`) was added without honouring the pre-existing global `--dry-run` contract.

**Why it occurred:** `--claim` support was likely added to `next` (which is otherwise a read-only command) without revisiting the shared `--dry-run` contract that other write commands already implement.

**Contributing factors:** `next` is primarily a query command, so there was no existing dry-run branch to extend by analogy within the file itself — the missing guard was easy to overlook.

## Resolution for the Issue

**Changes made:**
- `cmd/next.go:264-282` (`runNextWithClaim`) — guarded the `taskList.WriteFile(filename)` call with `if !dryRun`. The in-memory claim (status → in-progress, owner set) still happens so the existing output functions can render an accurate preview, but nothing is persisted to disk when `--dry-run` is set.
- `cmd/next.go` (`ClaimResponse`, `outputClaimJSON`) — added a `dry_run` field (`omitempty`), following the same convention already used by `internal/task.BatchResponse.DryRun`, so JSON consumers can detect a preview response programmatically.
- `cmd/next.go` (`outputClaimMarkdown`, `outputClaimTable`) — adjusted the markdown header and table title to say "Would Claim Tasks (dry run)" when `dryRun` is set, so human-facing output is unambiguous too.

**Approach rationale:** The minimal, correct fix is to gate the single `WriteFile` call — the root cause is exactly "no dry-run branch before the write". Reusing the existing claim-computation and output code (rather than duplicating a separate preview path) keeps the diff small and guarantees the preview reflects the exact same claim logic (phase/stream/one-flag selection) as the real path.

**Alternatives considered:**
- Compute a separate, unmutated preview list without touching the in-memory `taskPtr` fields — rejected because it would require duplicating the claimed-task-to-output-struct mapping and risks the preview drifting from what a real claim would actually do; gating only the write is simpler and the in-memory mutation is discarded when the process exits.
- Add an early return before task selection when `dryRun` is set — rejected because it would need to duplicate the ready-task selection logic (phase/stream/one-flag combinations) to produce a meaningful preview instead of just gating the write.

## Regression Test

**Test file:** `cmd/next_test.go`
**Test name:** `TestNextCommandClaimDryRun`

**What it verifies:** For `--format json`, `--format markdown`, and `--format table`, running `next --claim agent-a --dry-run` leaves the task file byte-for-byte unchanged (no `Owner:` line, no `[-]` status), while the JSON output still reports the proposed claim (`"success": true`, `"owner": "agent-a"`, `"dry_run": true`).

**Run command:** `go test -run TestNextCommandClaimDryRun -v ./cmd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/next.go` | Gated `taskList.WriteFile` in `runNextWithClaim` behind `!dryRun`; added `DryRun` field to `ClaimResponse`/`outputClaimJSON`; dry-run-aware headers in `outputClaimMarkdown`/`outputClaimTable` |
| `cmd/next_test.go` | Added `TestNextCommandClaimDryRun` regression test (JSON/markdown/table) |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make check`: fmt, lint, test)
- [x] Linters/validators pass (`golangci-lint`: 0 issues)

**Manual verification:**
- Reproduced the exact ticket repro (`go run . next <file> --claim agent-a --dry-run --format json`) before and after the fix: before, the file checksum changed and the file was rewritten with `[-]`/`Owner:`; after, the checksum is identical and the JSON output includes `"dry_run": true`.

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a new mutating code path to an otherwise read-only command, explicitly check whether the global `--dry-run` contract applies and wire it in from the start.
- Consider a lightweight test helper or lint check that flags any `TaskList.WriteFile` call not preceded by a `dryRun` check, given how consistently that pattern is required across `cmd/*.go`.

## Related

- T-1398: `next --claim --dry-run` mutates task files
- T-1786: duplicate of T-1398, closed in favour of it

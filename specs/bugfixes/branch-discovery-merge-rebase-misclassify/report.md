# Bugfix Report: Branch Discovery Merge/Rebase Misclassification

**Date:** 2025-07-16
**Status:** Fixed
**Ticket:** T-716

## Description of the Issue

`isSpecialGitState` in `internal/config/discovery.go` used `strings.Contains` to check whether a branch name contained the substrings "merge" or "rebase". This caused any branch with those words — such as `feature/merge-sort`, `bugfix/rebased-docs`, or `T-123/bugfix-merge-error` — to be falsely classified as a special git state. `DiscoverFileFromBranch` would then refuse auto-discovery with the error "special git state detected".

**Reproduction steps:**
1. Create a branch named `feature/merge-sort`
2. Run any rune command that relies on branch-based file discovery
3. Observe error: "special git state detected: feature/merge-sort"

**Impact:** Any user or agent working on a branch whose name contains "merge" or "rebase" could not use automatic file discovery and was forced to specify the file path explicitly.

## Investigation Summary

- **Symptoms examined:** `isSpecialGitState` returns `true` for ordinary branch names
- **Code inspected:** `internal/config/discovery.go` lines 145-161
- **Hypotheses tested:** The substring check was intended to detect actual git rebase/merge in-progress states, but `git rev-parse --abbrev-ref HEAD` returns the branch name, not a state indicator — actual detached states return "HEAD" or "(no branch)"

## Discovered Root Cause

The `isSpecialGitState` function used broad `strings.Contains` checks for "rebase" and "merge" on the branch name string. Since `git rev-parse --abbrev-ref HEAD` returns the actual branch name (not a state marker), these substring checks produced false positives for any branch containing those common words.

**Defect type:** Logic error — overly broad string matching

**Why it occurred:** The original implementation assumed that branch names containing "merge" or "rebase" indicated an in-progress merge/rebase operation, but that is not how git reports those states.

**Contributing factors:** The existing test suite included test cases like `"branch with rebase in name"` that asserted the incorrect `true` result, with a comment "This might be overly cautious but safer" — masking the bug.

## Resolution for the Issue

**Changes made:**
- `internal/config/discovery.go:145-155` — Removed `strings.Contains` checks for "rebase" and "merge". The function now only checks for exact matches against known detached-state values ("HEAD", "(no branch)").
- `internal/config/discovery_test.go:303-327` — Fixed existing incorrect test assertions and added 13 new regression test cases covering branch names containing "merge" and "rebase" in various positions.

**Approach rationale:** `git rev-parse --abbrev-ref HEAD` never returns a "rebase" or "merge" state string — it returns "HEAD" when detached. Removing the substring checks is the correct minimal fix.

**Alternatives considered:**
- Detecting merge/rebase state via `.git/MERGE_HEAD` or `.git/rebase-merge/` directories — not needed because `git rev-parse --abbrev-ref HEAD` already returns "HEAD" during these operations, which is already handled.

## Regression Test

**Test file:** `internal/config/discovery_test.go`
**Test name:** `TestIsSpecialGitState` (13 new sub-tests)

**What it verifies:** Branch names containing "merge" or "rebase" as substrings, prefixes, suffixes, or exact matches are correctly classified as normal (non-special) branches.

**Run command:** `go test ./internal/config/ -run TestIsSpecialGitState -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/config/discovery.go` | Removed substring checks for "merge"/"rebase" in `isSpecialGitState` |
| `internal/config/discovery_test.go` | Fixed incorrect assertions, added 13 regression test cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Build succeeds

## Prevention

**Recommendations to avoid similar bugs:**
- Detect git operational states via git metadata (`.git/MERGE_HEAD`, `.git/rebase-merge/`) rather than branch name heuristics
- Test with realistic branch names that include common keywords
- Be skeptical of "overly cautious but safer" comments — overly broad checks cause false positives

## Related

- T-716: Branch discovery misclassifies normal branch names containing "merge" or "rebase"

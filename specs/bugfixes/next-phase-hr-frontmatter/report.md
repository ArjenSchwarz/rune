# Bugfix Report: next-phase-hr-frontmatter

**Date:** 2026-04-16
**Status:** Fixed
**Transit:** T-763

## Description of the Issue

`FindNextPhaseTasks` and `FindNextPhaseTasksForStream` use `skipFrontMatter` to strip YAML front matter from task files. When a file contained front matter **and** a later horizontal rule (`---`), the `skipFrontMatter` function re-entered front-matter mode, permanently dropping all subsequent lines from phase extraction.

**Reproduction steps:**
1. Create a task file with YAML front matter (`---` delimited block at start).
2. Add a horizontal rule (`---`) later in the body.
3. Add phase headers and tasks after the horizontal rule.
4. Run `rune next --phase`.

**Impact:** Phases and tasks after a horizontal rule were invisible to the `next --phase` and `next --phase --stream` commands.

## Investigation Summary

- **Symptoms examined:** `skipFrontMatter` returned incomplete line sets when `---` appeared after the closing front matter delimiter.
- **Code inspected:** `internal/task/next.go`, specifically `skipFrontMatter` (lines 128-155).
- **Hypotheses tested:** The counter-based approach (`frontMatterCount`) incorrectly treated every `---` as a front matter delimiter rather than stopping after the first pair.

## Discovered Root Cause

The `skipFrontMatter` function used a `frontMatterCount` integer that incremented on every `---` line. After the closing delimiter (`frontMatterCount == 2`), subsequent `---` lines set `frontMatterCount` to 3 and toggled `inFrontMatter` back to `true`, causing all following lines to be skipped.

**Defect type:** Logic error

**Why it occurred:** The toggle logic did not distinguish "front matter closed" from "another `---` encountered". Once `frontMatterCount > 2`, the `else` branch always set `inFrontMatter = true`.

## Resolution for the Issue

**Changes made:**
- `internal/task/next.go:133-149` — Replaced `frontMatterCount` counter with a `frontMatterClosed` boolean. Once the closing delimiter is found, no further `---` lines are treated as delimiters.

**Approach rationale:** A boolean flag is simpler, more correct, and makes the "only strip the first `---` pair" invariant explicit.

**Alternatives considered:**
- Adding `frontMatterCount <= 2` guard — works but leaves the counter pattern which is conceptually misleading for a two-state toggle.

## Regression Test

**Test file:** `internal/task/next_test.go`
**Test names:**
- `TestSkipFrontMatter_HorizontalRuleAfterFrontMatter` — unit test for `skipFrontMatter` with HR after front matter
- `TestFindNextPhaseTasks_HorizontalRuleAfterFrontMatter` — end-to-end test through `FindNextPhaseTasks`

**What it verifies:** A `---` horizontal rule after YAML front matter is preserved in the output (not stripped), and phases after front matter are correctly found.

**Run command:** `go test -run "TestSkipFrontMatter_HorizontalRule|TestFindNextPhaseTasks_HorizontalRule" ./internal/task/ -v`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/next.go` | Replaced counter logic with boolean flag in `skipFrontMatter` |
| `internal/task/next_test.go` | Added regression tests for HR after front matter |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Code formatted (`make fmt`)

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer boolean state flags over counters when tracking two-state transitions (open/closed).
- Add test cases with horizontal rules whenever front matter stripping is involved.

# Bugfix Report: ParseFileWithPhases Front Matter Skipping After Horizontal Rules

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`ParseFileWithPhases()` has its own front-matter stripping loop (separate from `ParseFrontMatter`) that scans all lines for `---` delimiters. If the file starts with front matter and later contains a horizontal rule (`---`), the loop treats that later `---` as a new front-matter delimiter, setting `inFrontMatter = true` without ever resetting it (unless another `---` appears). This causes lines after the horizontal rule to be dropped from the phase-marker scan, so phase markers after the rule are missed.

**Reproduction steps:**
1. Create a task file with YAML front matter at the top.
2. Add content with a horizontal rule (`---`) later in the file.
3. Add a phase marker (`## Phase Name`) after the horizontal rule.
4. Call `ParseFileWithPhases()` -- the phase marker after `---` is not returned.

**Impact:** Phase markers after horizontal rules in files with front matter would be silently dropped, causing incorrect phase-based task filtering and display. Currently partially masked because `ParseMarkdown` rejects standalone `---` at root level, but the bug is a latent correctness issue in the front-matter stripping logic.

## Investigation Summary

- **Symptoms examined:** Front-matter stripping loop in `ParseFileWithPhases` iterates all lines and counts every `---` occurrence, not just the initial pair.
- **Code inspected:** `internal/task/parse.go` lines 80-101 (the inline front-matter stripping loop), `internal/task/references.go` (`ParseFrontMatter` for comparison), `internal/task/parse.go` (`parseContent` which delegates to `ParseFrontMatter`).
- **Hypotheses tested:** Confirmed the loop continues scanning after finding the closing delimiter. After `frontMatterCount` reaches 2, a 3rd `---` causes `frontMatterCount` to become 3, which hits the `else` branch setting `inFrontMatter = true`.

## Discovered Root Cause

The front-matter stripping loop in `ParseFileWithPhases` did not stop scanning for `---` delimiters after the closing delimiter was found. It continued iterating all remaining lines, and any subsequent `---` line would re-enter the "in front matter" state, causing all following lines to be excluded from the output.

**Defect type:** Logic error -- unbounded delimiter scanning.

**Why it occurred:** The original implementation used a counter-based approach (`frontMatterCount`) to track delimiters but did not break out of the loop or stop matching after finding the second delimiter. The `if frontMatterCount == 2` branch only set `inFrontMatter = false` for that specific iteration, but subsequent `---` lines would increment the counter to 3+ and take the `else` branch.

**Contributing factors:** The front-matter stripping logic was duplicated from `ParseFrontMatter` rather than reusing it. `ParseFrontMatter` uses substring matching (`\n---\n`) which correctly finds only the first pair, but the inline loop in `ParseFileWithPhases` used a different approach that was susceptible to this bug.

## Resolution for the Issue

**Changes made:**
- `internal/task/parse.go` -- Replaced the inline front-matter stripping loop with a new `stripFrontMatterLines` function. The new function finds the first two `---` lines and returns everything after the second one, ignoring any `---` lines that appear later in the file.

**Approach rationale:** The simplest correct approach is to stop scanning after finding the closing delimiter. Instead of tracking state across all lines, the new implementation returns `lines[i+1:]` as soon as the second `---` is found.

**Alternatives considered:**
- Reusing `ParseFrontMatter` directly -- rejected because `ParseFrontMatter` operates on a string and returns remaining content, while `ParseFileWithPhases` needs lines for `ExtractPhaseMarkers`. Converting back and forth would add unnecessary complexity.
- Adding a `break` to the existing loop after `frontMatterCount == 2` -- this would work but still require collecting `newLines` for lines already processed. The slice approach (`lines[i+1:]`) is simpler.

## Regression Test

**Test file:** `internal/task/parse_phases_frontmatter_test.go`
**Test names:** `TestParseFileWithPhases_FrontMatterStripping`, `TestFrontMatterStrippingForPhaseExtraction`

**What it verifies:**
- `TestParseFileWithPhases_FrontMatterStripping`: End-to-end tests through `ParseFileWithPhases` verifying phase markers are correctly extracted from files with front matter.
- `TestFrontMatterStrippingForPhaseExtraction`: Tests the `stripFrontMatterLines` function directly, including the T-458 scenario (front matter + horizontal rule + phase marker after it) and the multiple-horizontal-rules variant.

**Run command:** `go test -run "TestParseFileWithPhases_FrontMatterStripping|TestFrontMatterStrippingForPhaseExtraction" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Replaced inline front-matter stripping loop with `stripFrontMatterLines` function |
| `internal/task/parse_phases_frontmatter_test.go` | New regression tests for front-matter stripping in phase extraction |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Avoid duplicating logic that already exists elsewhere (e.g., `ParseFrontMatter`). When duplication is necessary, keep the logic minimal and test it separately.
- When scanning for delimiters, always bound the scan -- stop after finding the expected number of delimiters rather than continuing through the entire input.
- Extract inline loops into named functions with clear contracts, making them easier to test in isolation.

## Related

- T-458: Fix ParseFileWithPhases front matter skipping after horizontal rules
- Related bugfix: `parse-frontmatter-crlf` (CRLF handling in front matter parsing)

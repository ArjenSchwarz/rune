# Bugfix Report: parser-drops-non-bullet-body-lines

**Date:** 2026-09-07
**Status:** Fixed

## Description of the Issue

Indented content directly beneath a task that was not a valid Markdown bullet was silently ignored by the parser. Any later command that rewrote the file (`add`, `update`, `remove`, batch operations) permanently deleted that content with no error or warning.

**Reproduction steps:**

1. Create a task file with a non-bullet body line at the detail indentation:

   ```markdown
   # Tasks

   - [ ] 1. Keep me
     this malformed body line is not a Markdown bullet
   - [ ] 2. Keep me too
   ```

2. Run `rune list file.md --format json` — exits 0, the body line is absent from the output
3. Run `rune add file.md --title Added --format json` — exits 0 and rewrites the file
4. Inspect the file: the body line is gone

**Impact:** Silent, unrecoverable user data loss. The parser reported success, so nothing signalled that content had been discarded until the file was rewritten.

## Investigation Summary

- **Symptoms examined:** `list --format json` omitted the line; the file lost it after any mutating command; both commands exited 0
- **Code inspected:** `internal/task/parse.go` — `parseDetailsAndChildren`, `parseTasksAtLevel`, `parseDetailLine`, `detailLinePattern`
- **Hypotheses tested:** that `parseDetailLine` returning `""` was treated as "nothing to add" rather than "not a detail line", leaving the line neither stored nor rejected

## Discovered Root Cause

In `parseDetailsAndChildren`, the `indent == expectedIndent` branch treated every line at that indentation as a task detail:

```go
if detail := parseDetailLine(lines[i]); detail != "" {
    items = append(items, detail)
}
```

`parseDetailLine` returns `""` for any line that is not a valid `- <content>` bullet. The branch appended only on a non-empty result and did nothing otherwise, so a non-bullet line parsed "successfully" while being dropped from the in-memory `TaskList`. Because rendering writes from that structure, the next write-back omitted the line.

**Defect type:** Logic error — a sentinel return value ("not a detail line") conflated with a benign empty result.

**Why it occurred:** The guard reads as a nil-check idiom, so the failure path was never given an error branch.

**Contributing factors:** T-674 fixed the identical "silently skip invalid content" defect in `parseTasksAtLevel`'s `default` branch, but did not cover `parseDetailsAndChildren` — a separate function entered only when looking ahead for a task's details and children — which is why this path survived that fix.

## Resolution for the Issue

**Changes made:**

- `internal/task/parse.go` — when `parseDetailLine` returns `""` at the expected detail indentation, return `line %d: unexpected content at this indentation level (missing '- ' bullet?)` instead of continuing silently
- `internal/task/frontmatter_preservation_test.go`, `internal/task/next_test.go` — two pre-existing fixtures had detail/reference lines missing their `- ` prefix and were unknowingly relying on the bug; corrected to well-formed bullets. Neither test asserted on the dropped content, so what they verify is unchanged

**Approach rationale:** Matches the wording and pattern already used by `parseTasksAtLevel`, and keeps to the project principle that the parser reports errors rather than auto-correcting malformed input. The `(missing '- ' bullet?)` hint is appended only at this call site, where a missing bullet is the specific cause — the other two uses of the message cover content at the wrong level, where the hint would mislead.

**Alternatives considered:**

- Preserving the raw line as a plain-text detail — would round-trip the content, but silently reformats input and diverges from the documented file format (`README.md:667-673`), which has always shown detail lines as `- ` bullets
- Emitting a warning and continuing — leaves the data loss in place on write-back and there is no warning channel in the parser

## Regression Test

**Test file:** `internal/task/parse_detail_line_test.go`
**Test names:** `TestParseRejectsNonBulletDetailLines`, `TestParseAllowsValidDetailLines`

**What it verifies:** The exact repro from the ticket, a non-bullet line following a valid detail line, and a bare `-` with no content all produce a parse error (the first also asserts the `(missing '- ' bullet?)` hint); legitimate detail bullets still parse into `Details` unchanged.

**Run command:** `go test -run 'TestParseRejectsNonBulletDetailLines|TestParseAllowsValidDetailLines' -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/parse.go` | Return an error instead of silently dropping a non-bullet line at detail indentation |
| `internal/task/parse_detail_line_test.go` | New regression tests (3 rejection cases, 1 positive case) |
| `internal/task/next_test.go` | Bulleted a fixture's continuation line; renamed the case to `bulleted_continuation_with_description` |
| `internal/task/frontmatter_preservation_test.go` | Bulleted a fixture's detail and reference lines |
| `CHANGELOG.md` | Entry under `[Unreleased]` → `Fixed` |

## Verification

**Automated:**

- [x] Regression tests fail before the fix and pass after
- [x] Full test suite passes (`make check`: fmt, golangci-lint 0 issues, unit tests)
- [x] Integration tests pass (`INTEGRATION=1 go test ./cmd/...`)
- [x] Code formatted (`make fmt`)

## Prevention

**Recommendations to avoid similar bugs:**

- When a parse helper uses an empty/zero return as its "did not match" sentinel, the caller needs an explicit failure branch — `if x != "" { use(x) }` silently discards the failure case
- A parser that reports success must not lose input. When adding a new "skip" path, ask what a later write-back would do with the skipped content
- Fixing a "silently skips invalid content" defect in one function is a prompt to grep for the same shape in sibling parse functions rather than only the reported call site

## Related

- Transit T-2041
- T-674: Report invalid indented lines instead of silently ignoring them (same defect class in `parseTasksAtLevel`; this fix closes the remaining path)

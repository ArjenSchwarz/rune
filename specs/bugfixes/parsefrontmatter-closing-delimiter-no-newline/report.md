# Bugfix Report: ParseFrontMatter Rejects Closing Delimiter Without Trailing Newline

**Date:** 2026-03-09
**Status:** Investigating

## Description of the Issue

`ParseFrontMatter` incorrectly rejects a closing front matter delimiter (`---`) when it appears at the end of the file without a trailing newline. The function returns "unclosed front matter block" even though this is valid YAML front matter.

**Reproduction steps:**
1. Create content with front matter where the closing `---` has no trailing newline (e.g., `"---\nreferences:\n  - ./doc.md\n---"`)
2. Call `ParseFrontMatter` on this content
3. Observe that it returns an "unclosed front matter block" error

**Impact:** Any markdown file where the front matter block is the entire file content (no trailing newline after the closing `---`) fails to parse. This can occur when editors trim trailing newlines or when front matter is the only content.

## Investigation Summary

- **Symptoms examined:** `ParseFrontMatter` returns "unclosed front matter block" for valid front matter at EOF
- **Code inspected:** `internal/task/references.go` (ParseFrontMatter function)
- **Hypotheses tested:** The root cause was immediately apparent from code inspection

## Discovered Root Cause

`ParseFrontMatter` uses patterns that require a trailing newline after the closing delimiter:

1. **Opening delimiter check:** `strings.HasPrefix(content, "---\n")` -- requires `\n` after opening `---` (correct, since content must follow)
2. **Empty front matter check:** `strings.HasPrefix(content[searchStart:], "---\n")` -- requires `\n` after closing `---`, fails when `---` is at EOF
3. **Closing delimiter search:** `endPattern := "\n---\n"` -- requires `\n` after closing `---`, fails when `---` is at EOF

The function never considers the case where `\n---` appears at the very end of the content string (i.e., `strings.HasSuffix(content, "\n---")`).

**Defect type:** Missing boundary condition handling

**Why it occurred:** The delimiter matching assumes content always continues after the closing `---`. This is true for most files but not when the front matter block ends the file.

**Contributing factors:** All existing test cases included content after the closing delimiter, so this edge case was never exercised.

## Resolution for the Issue

<!-- To be filled after fix is implemented -->

## Regression Test

**Test file:** `internal/task/references_test.go`
**Test names:** `TestParseFrontMatter/closing_delimiter_without_trailing_newline`, `TestParseFrontMatter/empty_front_matter_without_trailing_newline`, `TestParseFrontMatter/closing_delimiter_without_trailing_newline_and_CRLF`

**What it verifies:** Front matter parsing succeeds when the closing `---` has no trailing newline, including with CRLF line endings.

**Run command:** `go test -run "trailing_newline" -v ./internal/task/`

## Affected Files

| File | Change |
|------|--------|
| `internal/task/references.go` | Fix delimiter matching to accept EOF after closing `---` |
| `internal/task/references_test.go` | Added 3 regression test cases |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When matching delimiters in text, always consider that the delimiter may appear at the very end of the input
- Include EOF boundary test cases for any text parsing function

## Related

- Transit ticket: T-386
- Related prior fix: T-265 (ParseFrontMatter CRLF handling)

# Parsing Module Notes

## File Structure

- `internal/task/parse.go` - Main parser: `ParseMarkdown`, `ParseFile`, `ParseFileWithPhases`, `parseContent`, line-level parsing
- `internal/task/references.go` - Front matter: `ParseFrontMatter`, `SerializeWithFrontMatter`, `FrontMatter` struct
- `internal/task/references_test.go` - Unit tests for `ParseFrontMatter` and `SerializeWithFrontMatter`
- `internal/task/parse_frontmatter_test.go` - Integration tests for front matter through `ParseMarkdown`
- `internal/task/parse_basic_test.go` - Basic parsing tests
- `internal/task/parse_phases_frontmatter_test.go` - Tests for front-matter stripping in `ParseFileWithPhases` and `stripFrontMatterLines`

## CRLF Handling

CRLF normalization happens at three levels:
1. `ParseFrontMatter` normalizes `\r\n` to `\n` at the start, before any delimiter matching
2. `splitLines` (in parse.go) splits on `\n` and trims trailing `\r` from each line. Used by `parseContent`, `ParseFileWithPhases`, and all phase-related functions in `phase.go`, `next.go`, and `operations.go`. **Always use `splitLines` instead of `strings.Split(content, "\n")` when processing markdown content in the task package.**
3. `ExtractPhaseMarkers` trims `\r` from each line internally as a defensive measure for external callers (like `cmd/has_phases.go`) that pass pre-split lines

`ParseFileWithPhases` uses `stripFrontMatterLines` to remove front matter before extracting phase markers. This function finds the first two `---` lines and returns everything after the closing delimiter, ignoring any later `---` lines (horizontal rules). Fixed in T-458 — the previous inline loop continued scanning all lines and re-entered "front matter" state on any subsequent `---`.

**Gotcha:** Go's RE2 regex `$` matches end-of-text, and `.+` will match `\r`. So patterns like `^## (.+)$` appear to work on CRLF lines, but the captured group includes `\r`. Patterns using specific character classes at the end (like `(\d+)$` in `streamPattern`) will fail outright on untrimmed CRLF lines because `\d` does not match `\r`. Always normalize before matching.

## Front Matter Delimiter Matching

`ParseFrontMatter` uses hardcoded LF patterns (`"---\n"`, `"\n---\n"`) with fixed byte offsets. The CRLF normalization at the top of the function ensures these patterns work regardless of the input's line ending style.

The closing delimiter matching also handles EOF without a trailing newline: `rest == "---"` for empty front matter at EOF, and `strings.HasSuffix(rest, "\n---")` for content front matter at EOF. In both cases, the remaining content is returned as an empty string.

## Title Detection

Title detection in `parseContent()` only considers the **first non-empty line** of the document (after front matter is stripped). If that line is an H1 (`# Title`), it is used as the document title and removed from the line array. If the first non-empty line is anything else (e.g., a task line), no title is set.

A `# ` heading appearing later in the file (after tasks) is NOT treated as a title. It will be rejected by the task parser as unexpected content. This was fixed in T-448 — the previous implementation scanned all lines and took the first `# ` match anywhere.

## CI Notes

The `push.yml` workflow's linter step has pre-existing QF1012 (staticcheck) failures across `cmd/list.go`, `cmd/next.go`, and `internal/task/render.go` for `WriteString(fmt.Sprintf(...))` patterns that should use `fmt.Fprintf(...)`. This affects all branches, not just specific PRs.

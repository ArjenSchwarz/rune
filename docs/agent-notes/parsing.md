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

## UTF-8 BOM

`ParseMarkdown` strips exactly one leading UTF-8 BOM after the size check, and `phaseMarkersFromContent` does the same for `ParseFileWithPhases` (T-2063). `splitLines` deliberately does NOT strip a BOM: it also runs on content `ParseMarkdown` has already stripped, so a second BOM would be removed too. Raw-content phase helpers in `next.go`/`phase.go` don't strip it; that only matters for a BOM file whose front matter has a `## ` YAML comment.

## Front Matter Delimiter Matching

`ParseFrontMatter` uses hardcoded LF patterns (`"---\n"`, `"\n---\n"`) with fixed byte offsets. The CRLF normalization at the top of the function ensures these patterns work regardless of the input's line ending style.

The closing delimiter matching also handles EOF without a trailing newline: `rest == "---"` for empty front matter at EOF, and `strings.HasSuffix(rest, "\n---")` for content front matter at EOF. In both cases, the remaining content is returned as an empty string.

## Unknown Front-Matter Keys

`FrontMatter.Extra` holds top-level keys other than `references`/`metadata` as raw `*yaml.Node` key/value pairs in source order, filled by `FrontMatter.UnmarshalYAML` and appended after the known keys by `MarshalYAML` (T-1845). Nodes, not decoded Go values, are deliberate: decoding into `map[string]any` drops YAML tags (`!vault x` became `x`), and an inline map re-sorts keys alphabetically, which can write an alias before its anchor and produce unparsable YAML. `ParseFrontMatter` does a trial marshal/unmarshal and rejects front matter that would not re-parse (e.g. an unknown key aliasing an anchor on `references`, which is re-encoded without the anchor). Use `FrontMatter.IsEmpty()` to decide whether to write front matter. If you add a field to `FrontMatter`, also add it to `knownFrontMatter`, the skip list in `UnmarshalYAML`, and `deepCopy` (batch.go). Comments are not preserved.

## Title Detection

Title detection in `parseContent()` only considers the **first non-empty line** of the document (after front matter is stripped). If that line is an H1 (`# Title`), it is used as the document title and removed from the line array. If the first non-empty line is anything else (e.g., a task line), no title is set.

A `# ` heading appearing later in the file (after tasks) is NOT treated as a title. It will be rejected by the task parser as unexpected content. This was fixed in T-448 — the previous implementation scanned all lines and took the first `# ` match anywhere.

A bare `# ` line (empty H1) is deliberately rejected by the parser, so both renderers omit the H1 entirely when `TaskList.Title` is empty (T-2062). Don't reintroduce a placeholder heading.

## Detail Lines vs Metadata

`applyDetailLine` in parse.go is the single place that decides whether a `- ` line under a task is read back as metadata (Blocked-by, Stream, Owner, Requirements, References) or as a plain detail. `validateDetails` calls it (through `detailMetadataField`) and rejects any detail that would be read back as metadata, because the renderer writes details as plain `- ` bullets and such a detail would silently turn into data on the next read. When adding a new metadata line type, add it to `applyDetailLine` only; validation picks it up automatically.

Requirements lines are parsed strictly, link by link (T-1848, T-1907): the value must be only `[ID](path#anchor)` links separated by commas, since paths may contain commas. Anything else keeps the whole line as a plain detail, not a partial requirement list. `validateDetails` also rejects details that `parseTaskLine` would read as a task line or as malformed task syntax (T-1999).

References use `\,` for a comma inside a reference and `\\` for a backslash where it would be ambiguous (`formatReferenceList` / `parseReferences`, T-1808). Any other backslash is literal, so Windows paths are written unchanged.

`applyDetailLine` also returns a parse error when a task's requirement links name a different file than `TaskList.RequirementsFile` (or mix files within one line), because the renderer writes every link with that single path (T-1973). `detailMetadataField` ignores that error — it only needs the field name.

## CI Notes

The `push.yml` workflow's linter step has pre-existing QF1012 (staticcheck) failures across `cmd/list.go`, `cmd/next.go`, and `internal/task/render.go` for `WriteString(fmt.Sprintf(...))` patterns that should use `fmt.Fprintf(...)`. This affects all branches, not just specific PRs.

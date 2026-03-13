# Bugfix Report: ParseMarkdown Title Detection Within Task Details

**Ticket:** T-448
**Date:** 2026-03-13
**Status:** Fixed

## Problem Statement

`ParseMarkdown` scans all lines for a `# ` heading and treats the first occurrence as the document title, even if it appears after tasks in the file. This can set an incorrect title and remove the offending line from the parsed content.

Example input:
```markdown
- [ ] 1. First task
- [ ] 2. Second task

# Not a title
```

Parsed result: `Title = "Not a title"`, and the `# Not a title` line is removed from the line array.

## Root Cause

In `parseContent()` (parse.go, lines 128-136), the title detection loop iterates over **all** lines looking for the first `# ` prefix after `TrimSpace`:

```go
for i, line := range lines {
    line = strings.TrimSpace(line)
    if strings.HasPrefix(line, "# ") {
        taskList.Title = strings.TrimSpace(strings.TrimPrefix(line, "#"))
        lines = append(lines[:i], lines[i+1:]...)
        break
    }
}
```

**Why this is wrong:** A title (H1 heading) should only be recognized if it appears at the very beginning of the document, before any task lines. An `# ` line appearing after tasks, between tasks, or later in the file is not a title -- it is either content or a malformed line. The current code has no positional constraint, so any `# ` line anywhere in the file gets treated as the title.

**Five Whys:**
1. Why is the wrong title detected? Because the loop scans all lines, not just leading lines.
2. Why does the loop scan all lines? Because it was written without a stopping condition at the first task or non-blank line.
3. Why is there no stopping condition? The original implementation assumed `# ` headings would only appear at the top of the file.

## Proposed Fix

Change the title detection to only consider lines before the first task line. Specifically: iterate through lines, skip blank lines, and if the first non-blank line starts with `# `, treat it as the title. Stop searching as soon as a non-blank, non-title line is encountered.

## Regression Tests

Added to `internal/task/parse_basic_test.go`:
- `hash_in_detail_not_treated_as_title` - Detail line with `# Note` is not treated as title
- `title_before_tasks_with_hash_detail` - Real title is detected when `# ` also appears in details
- `hash_after_tasks_causes_error` - A `# ` line after tasks is rejected as unexpected content
- `no_title_tasks_only` - File with only tasks has no title
- `title_with_blank_line_before_tasks` - Title followed by blank line and tasks works correctly
- `hash_in_detail_preserved` - Detail line with `# ` content is preserved in task details
- `hash_in_detail_with_title` - Detail with `# ` content preserved alongside a real title

## Resolution

Changed the title detection loop in `parseContent()` to only consider the first non-empty line in the document. The loop now skips blank lines, checks if the first non-blank line is an H1 heading (`# `), and breaks unconditionally after the first non-blank line regardless of whether it was a title.

**File changed:** `internal/task/parse.go` (lines 127-140)

Before: The loop iterated through all lines and selected the first `# ` match anywhere in the file.

After: The loop skips blank lines and breaks after the first non-blank line. If that first non-blank line is an H1, it's used as the title. Otherwise, no title is set and the line is left in place for task parsing.

A `# ` heading appearing after tasks is now correctly rejected as unexpected content by the task parser, rather than being silently consumed as the document title.

# Smart Branch Discovery

## Overview

Enhance the git branch-based file discovery to intelligently strip branch prefixes and try multiple candidate paths. Currently, branches like `specs/my-feature` or `feature/auth` cause incorrect path resolution because the full branch name is used verbatim. This change makes discovery work seamlessly with common branch naming conventions.

## Requirements

- The system MUST strip the prefix before the first `/` from the branch name when constructing candidate paths (e.g., `specs/my-feature` becomes `my-feature`)
- The system MUST try the stripped branch name path first, then fall back to the full branch name path
- The system MUST skip duplicate candidates when the branch has no `/` (e.g., `main` produces only one candidate, not two identical paths)
- The system MUST preserve all existing security validations (branch stripping happens after validation in `DiscoverFileFromBranch`)
- The system MUST maintain backward compatibility: existing files at full branch paths continue to work

## Implementation Approach

**Files to modify:**
- `internal/config/discovery.go` - Update `DiscoverFileFromBranch` to try multiple candidates
- `internal/config/discovery_test.go` - Add tests for new detection logic

**Approach:**

Update `DiscoverFileFromBranch` to try both stripped and full branch names with the template:

```go
func DiscoverFileFromBranch(template string) (string, error) {
    branch, err := getCurrentBranch()
    if err != nil {
        return "", fmt.Errorf("getting git branch: %w", err)
    }

    if isSpecialGitState(branch) {
        return "", fmt.Errorf("special git state detected: %s", branch)
    }

    // Strip prefix before first slash
    strippedBranch := branch
    if _, after, found := strings.Cut(branch, "/"); found {
        strippedBranch = after
    }

    // Try stripped name first, then full name
    candidates := []string{
        strings.ReplaceAll(template, "{branch}", strippedBranch),
    }
    if strippedBranch != branch {
        candidates = append(candidates, strings.ReplaceAll(template, "{branch}", branch))
    }

    for _, path := range candidates {
        if fileExists(path) {
            return path, nil
        }
    }

    return "", fmt.Errorf("task file not found for branch %q (tried: %s)",
        branch, strings.Join(candidates, ", "))
}
```

This works with any template pattern without special-casing.

**Dependencies:**
- Uses existing `getCurrentBranch()` function
- Uses existing `isSpecialGitState()` validation
- Uses existing `fileExists()` function
- Standard library only (`strings.Cut`, `strings.ReplaceAll`, `strings.Join`)

**Out of Scope:**
- Changes to configuration file format
- UPPERCASE filename variants (no evidence of user need)

## Risks and Assumptions

- **Risk**: Branch names with multiple slashes (e.g., `feature/auth/oauth`) strip to `auth/oauth` | **Mitigation**: The full branch path is always tried as fallback; this handles any edge cases where stripping produces the wrong name
- **Risk**: Both stripped and full paths might exist with different content | **Mitigation**: Document that stripped path takes precedence; users should not maintain duplicate task files
- **Assumption**: Specs directories follow the `specs/{name}/tasks.md` convention
- **Assumption**: Users on branches like `specs/my-feature` expect the spec at `specs/my-feature/tasks.md`
- **Prerequisite**: Existing tests for `DiscoverFileFromBranch` will need updating to account for multi-candidate behavior

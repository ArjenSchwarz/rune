# Bugfix Report: Empty Config Disables Discovery

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1065

## Description of the Issue

An empty `.rune.yml`, or one that only sets `discovery.template`, silently disabled branch discovery, although `discovery.enabled` is documented to default to `true`.

**Reproduction steps:**
1. In a git repo with a matching `specs/{branch}/tasks.md`, create an empty `.rune.yml`.
2. Run `rune next`.
3. Observe `Error: no filename specified and git discovery failed or disabled`.

**Impact:** Repos with a placeholder config or a template-only config lost auto-discovery.

## Investigation Summary

- **Code inspected:** `internal/config/config.go` (`loadConfigFile`, `defaultConfig`), `internal/config/config_test.go`.
- The existing test case `"empty file uses defaults"` asserted `wantEnabled: false`, encoding the bug.

## Discovered Root Cause

`loadConfigFile` decoded into a zero-value `Config` and back-filled only `Discovery.Template`. `Discovery.Enabled` stayed at Go's zero value (`false`) whenever the key was absent.

**Defect type:** Missing default.

## Resolution for the Issue

**Changes made:**
- `internal/config/config.go` - Decode YAML on top of `defaultConfig()`, so any omitted key keeps its default. An empty file yields the defaults.

**Approach rationale:** Single source of defaults; explicit `enabled: false` still overrides.

**Alternatives considered:**
- Change `Enabled` to `*bool` - wider type change for no extra benefit.

## Regression Test

**Test file:** `internal/config/config_test.go`
**Test name:** `TestLoadConfigFile` cases `empty file uses defaults` (now expects `true`), `template only keeps discovery enabled`, `explicit enabled false is respected`

**Run command:** `go test ./internal/config -run TestLoadConfigFile`

## Affected Files

| File | Change |
|------|--------|
| `internal/config/config.go` | Decode over defaults |
| `internal/config/config_test.go` | Corrected and added cases |

## Verification

**Automated:**
- [x] Regression tests pass (two failed before the fix)
- [x] Full test suite passes (see T-959 report for pre-existing root-only failures)
- [x] `make lint`: 0 issues

**Manual verification:**
- Empty `.rune.yml` at repo root: `rune next` now finds the branch task file.

## Prevention

- Decode configuration over a defaults struct rather than back-filling individual fields.

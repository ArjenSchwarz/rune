# Bugfix Report: fail-on-invalid-rune-yml

**Date:** 2026-03-27
**Status:** Fixed

## Description of the Issue

When `.rune.yml` exists but contains invalid YAML or unknown fields, rune silently falls back to defaults instead of surfacing the configuration error to the user. This hides config mistakes and can lead to unexpected discovery behavior.

**Reproduction steps:**
1. Create a `.rune.yml` at repo root with invalid YAML (e.g., `template: [broken`)
2. Run any rune command that calls `LoadConfig` (e.g., `rune list`)
3. Observe: no error is surfaced; defaults are silently used

**Impact:** Medium — users with typos in config (e.g., `discovry:` instead of `discovery:`) get no feedback, leading to confusion when discovery doesn't behave as configured.

## Investigation Summary

The config loading system uses a precedence-based search: repo root → CWD → home dir → defaults.

- **Symptoms examined:** `loadConfigUncached()` iterates over config paths and silently skips any that return errors
- **Code inspected:** `internal/config/config.go` — `loadConfigUncached()`, `loadConfigFile()`
- **Hypotheses tested:** The loop `if cfg, err := loadConfigFile(path); err == nil` treats "file not found" and "invalid YAML" identically — both are silently skipped

## Discovered Root Cause

Two defects in `internal/config/config.go`:

1. **Silent error swallowing** (lines 56-60): `loadConfigUncached()` ignores ALL errors from `loadConfigFile()`, including parse errors for files that exist but are invalid. It should only skip "file not found" errors.

2. **No strict YAML decoding** (line 74): `yaml.Unmarshal()` silently ignores unknown fields. A typo like `discovry:` is silently accepted, producing a zero-value config that falls back to defaults.

**Defect type:** Missing error discrimination + missing input validation

**Why it occurred:** The original implementation focused on the fallback chain (try multiple paths) without considering that some errors (parse failures) should halt the search.

**Contributing factors:** `gopkg.in/yaml.v3`'s `Unmarshal()` is permissive by default — strict mode requires explicit opt-in via `Decoder.KnownFields(true)`.

## Resolution for the Issue

_To be filled after fix is implemented._

## Regression Test

**Test file:** `internal/config/config_test.go`
**Test names:**
- `TestLoadConfigUncachedInvalidYAML` — invalid YAML returns error from top-level loader
- `TestLoadConfigUncachedUnknownFields` — unknown fields return error from top-level loader
- `TestLoadConfigFileUnknownFields` — unknown fields rejected at file-level parser
- `TestLoadConfigUncachedMissingFileStillDefaults` — missing files still return defaults (no regression)

**What it verifies:** Config files that exist but are invalid produce errors rather than silent fallback to defaults.

**Run command:** `go test -run "TestLoadConfigUncached|TestLoadConfigFileUnknownFields" -v ./internal/config/`

## Affected Files

| File | Change |
|------|--------|
| `internal/config/config.go` | Distinguish file-not-found from parse errors; use strict YAML decoding |
| `internal/config/config_test.go` | Add regression tests for T-556 |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Create `.rune.yml` with invalid YAML, verify error message
- Create `.rune.yml` with unknown field, verify error message
- Remove `.rune.yml`, verify defaults still work

## Prevention

**Recommendations to avoid similar bugs:**
- Always distinguish between "resource not found" and "resource invalid" when searching multiple paths
- Use strict/known-fields mode when unmarshaling config files
- Test error paths at integration level, not just unit level

## Related

- Transit T-556: Fail on invalid .rune.yml instead of silently defaulting
- T-482: Previous config loading fix (subdirectory support)

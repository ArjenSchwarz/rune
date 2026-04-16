# Bugfix Report: config-tests-write-real-home

**Date:** 2026-04-16
**Status:** Fixed

## Description of the Issue

Config tests in `internal/config/config_test.go` wrote to the developer's real home directory (`~/.config/rune/config.yml`) and the project's working directory (`.rune.yml`), making the test suite not self-contained.

**Reproduction steps:**
1. Run `make test` in a sandboxed environment where `~/.config/rune/` is outside the writable sandbox
2. `TestConfigPrecedence` calls `os.UserHomeDir()` and creates `~/.config/rune/config.yml`
3. Test fails with `operation not permitted`

**Impact:** Tests fail in any sandboxed CI environment (e.g., Codex) unless `HOME` is explicitly overridden. Also risks polluting developer home directories with test artifacts.

## Investigation Summary

- **Symptoms examined:** `open /Users/arjen/.config/rune/config.yml: operation not permitted` during `make test`
- **Code inspected:** `internal/config/config_test.go` — `TestConfigPrecedence`, `TestLoadConfig`, `TestExpandHome`
- **Hypotheses tested:** Confirmed `os.UserHomeDir()` at line 226 resolves to real home; no `t.Setenv("HOME", ...)` isolation present

## Discovered Root Cause

`TestConfigPrecedence` used the real `os.UserHomeDir()` to create a config file under `~/.config/rune/`, writing outside the test sandbox.

`TestLoadConfig` wrote `.rune.yml` to the actual project CWD instead of an isolated temp directory.

`TestExpandHome` referenced the real `HOME` env var, making assertions dependent on the host environment.

**Defect type:** Missing test isolation

**Why it occurred:** Tests were written before sandboxed execution was a requirement.

**Contributing factors:** `os.UserHomeDir()` is easy to call directly; Go's `t.Setenv` + `t.TempDir` pattern for HOME isolation isn't enforced by convention.

## Resolution for the Issue

**Changes made:**
- `internal/config/config_test.go` — `TestConfigPrecedence`: Redirected HOME to `t.TempDir()` via `t.Setenv`, created an isolated git repo in a temp dir, writes config files only within temp dirs
- `internal/config/config_test.go` — `TestLoadConfig`: Same isolation pattern — chdir to temp dir with git repo, fake HOME
- `internal/config/config_test.go` — `TestExpandHome`: Set HOME to a known temp dir for deterministic assertions

**Approach rationale:** `t.Setenv("HOME", ...)` is the standard Go approach — it's automatically restored after the test and works with `os.UserHomeDir()`.

**Alternatives considered:**
- Mocking `os.UserHomeDir` via a package-level var — more invasive, unnecessary when `t.Setenv` works

## Regression Test

The fixed tests themselves serve as regression tests. `TestConfigPrecedence` now proves that both local and home config files are found using only isolated temp directories.

**Run command:** `go test -v -run 'TestLoadConfig$|TestExpandHome|TestConfigPrecedence' ./internal/config/`

## Affected Files

| File | Change |
|------|--------|
| `internal/config/config_test.go` | Isolate HOME and CWD in TestConfigPrecedence, TestLoadConfig, TestExpandHome |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Code formatted (`make fmt`)

## Prevention

**Recommendations to avoid similar bugs:**
- Always use `t.Setenv("HOME", t.TempDir())` in tests that touch home-directory paths
- Never call `os.UserHomeDir()` directly in tests without first isolating HOME
- Tests that write files should operate in `t.TempDir()`, not the project CWD

## Related

- Transit T-812: Config tests write to real home directory

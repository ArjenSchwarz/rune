# Decision Log: Homebrew Install

## Decision 1: Append homebrew job to release.yml instead of a separate workflow

**Date**: 2026-04-19
**Status**: accepted

### Context

The formula-update step depends on the release matrix having uploaded all four platform tarballs. GitHub fires `release: published` once, triggering any workflow subscribed to that event concurrently. The release matrix takes several minutes to finish, so a separate workflow on the same trigger would race against it.

### Decision

Append a new `homebrew` job to the existing `.github/workflows/release.yml` with `needs: releases-matrix`, rather than creating a separate `homebrew.yml`.

### Rationale

`needs:` gives deterministic ordering with zero extra machinery: the formula job only starts once every matrix leg has uploaded its asset. No polling, no `workflow_run`, no race window.

### Alternatives Considered

- **Separate `homebrew.yml` with `workflow_run: release.yml`**: Keeps release.yml untouched, but `workflow_run` does not fire for workflow-triggered workflows, runs on both success and failure (needs gating), and decouples related logic across two files.
- **Separate `homebrew.yml` on `release: published` with polling**: Simple to reason about but racy. A 5-minute poll may not cover a matrix build that can exceed 10 minutes.

### Consequences

**Positive:**
- Deterministic execution order — formula updates run only after all tarballs exist.
- Single place to understand the release pipeline.
- No wasted macOS minutes polling for assets that aren't ready.

**Negative:**
- Modifies release.yml (the spec originally listed this as out-of-scope; that constraint was dropped as arbitrary).
- Homebrew job is tightly coupled to the release workflow — future refactors touch both.

---

## Decision 2: Use a fine-grained PAT (`HOMEBREW_TAP_TOKEN`) for cross-repo push

**Date**: 2026-04-19
**Status**: accepted

### Context

The default `GITHUB_TOKEN` cannot push to a different repository, so the homebrew job needs a cross-repo credential to commit the updated formula to `ArjenSchwarz/homebrew-rune`.

### Decision

Use a fine-grained Personal Access Token scoped to `ArjenSchwarz/homebrew-rune` with `contents:write`, stored as the repo secret `HOMEBREW_TAP_TOKEN`.

### Rationale

Simplest viable credential for a single-maintainer personal project. Fine-grained PATs let us narrow the blast radius to the tap repo only. Expiry risk is acceptable given the maintainer can rotate on a yearly cadence.

### Alternatives Considered

- **Deploy key on the tap repo**: No expiry, but requires SSH-based `git push`, private key in secret, and is still tied to the tap repo only. Marginal benefit over PAT for more operational complexity.
- **GitHub App**: Most robust and rotatable, supports auditable installations. Heavier setup (app creation, installation on both repos, token minting at runtime via `actions/create-github-app-token`). Over-engineered for a personal project with one maintainer.

### Consequences

**Positive:**
- Minimal setup; one secret, one PAT.
- Scope-limited to the tap repo.

**Negative:**
- Max 1-year lifetime; rotation is manual and easy to forget.
- Tied to a personal account — if the account is compromised or disabled, automation breaks.

---

## Decision 3: Bootstrap tap with a placeholder formula, trigger a v1.3.1 release immediately

**Date**: 2026-04-19
**Status**: accepted

### Context

The tap repo must exist (with a `Formula/rune.rb`) before the automation can push to it. There is a window between tap creation and the first automated release during which `brew install` could fail.

### Decision

Seed the tap with a placeholder (non-functional) `Formula/rune.rb`. Immediately after the tap repo and `HOMEBREW_TAP_TOKEN` secret are in place, cut a `v1.3.1` release; the new workflow populates the real formula on first run.

### Rationale

Keeps the bootstrap step mechanical — no manual sha256 computation for the current v1.3.0 tarballs. The broken-install window is tiny (minutes between tap creation and v1.3.1 publish), and no user is relying on the tap before it's announced anyway.

### Alternatives Considered

- **Seed with a real v1.3.0 formula (manual sha256 computation)**: `brew install` works from the moment the tap is public, before any release is cut. Requires computing four sha256s manually for a one-time bootstrap — not worth the effort given how quickly v1.3.1 can be cut.

### Consequences

**Positive:**
- Bootstrap is mechanical; no hand-computed checksums.
- Exercises the automation on the very first release, proving it works.

**Negative:**
- Tap is non-functional between creation and first automated release — must not be announced publicly before v1.3.1 lands.

---

## Decision 4: Validate formula with `brew audit --strict` and `brew test` in CI

**Date**: 2026-04-19
**Status**: accepted

### Context

A broken formula (wrong URL, wrong sha256, missing binary, malformed Ruby) reaches every `brew install` user before anyone notices. Homebrew provides `brew audit` (static checks) and `brew test` (installs + runs the formula's test block) that catch most such problems locally.

### Decision

Run `brew audit --strict --online Formula/rune.rb` and `brew test Formula/rune.rb` on `macos-latest` in the homebrew job, before pushing to the tap repo. Fail the job on any error.

### Rationale

The "MUST pass brew audit" requirement is only meaningful if the workflow actually enforces it. The cost is one `macos-latest` runner for ~2 minutes per release — acceptable for a release-only job.

### Alternatives Considered

- **Skip CI validation, rely on downstream user reports**: Cheaper and faster, but the failure mode (silent broken install for every user) is unacceptable. Would require dropping the audit requirement to be honest.
- **Run audit on `ubuntu-latest` with Homebrew on Linux**: Works but is slower to spin up (Homebrew is slower on Linux runners) and diverges from the platform most users install on.

### Consequences

**Positive:**
- Broken formulas cannot reach the tap repo.
- `brew test` actually installs the real tarball and runs `rune --version`, so URL and sha256 errors surface in CI.

**Negative:**
- macOS runner minutes cost more than Linux minutes.
- `brew audit` warnings can change upstream and cause occasional CI flakes unrelated to rune.

---

## Decision 5: Emit sha256 sidecars from the release matrix and consume them in the homebrew job

**Date**: 2026-04-19
**Status**: accepted

### Context

The homebrew job needs a sha256 digest per platform tarball to write into the formula. The naive implementation downloads each tarball and computes `shasum -a 256` locally. Tarballs are multi-MB; downloading four of them on every release is wasteful when the matrix already has the bytes in hand.

### Decision

Enable `sha256sum: true` on the existing `wangyoucao577/go-release-action` step so each matrix leg publishes a `.sha256` sidecar alongside its tarball. The homebrew job fetches only the sidecars (~80 bytes each) and parses the digest with `awk`.

### Rationale

The action already computes and publishes checksums (`.md5` sidecars exist today). Flipping `sha256sum: true` is a one-line change with no extra scripting, gives end users a published sha256 for manual verification, and avoids several MB of redundant traffic on every release.

### Alternatives Considered

- **Download tarballs in the homebrew job and compute sha256 locally**: Works with zero changes to the matrix step, but redownloads what was just built and uploaded. No user-facing checksum benefit.
- **Add a custom step to each matrix leg that runs `shasum -a 256` and uploads via `gh release upload`**: Needed only if the action didn't support sha256sum natively. Since it does, the custom step is strictly more code.
- **Pass sha256 via matrix job outputs**: Matrix job outputs collapse into a single value per key across legs, making this awkward without per-leg artifact uploads. Sidecars are simpler.

### Consequences

**Positive:**
- One-line change in the release step.
- Homebrew job is fast (fetches four small sidecars, not four multi-MB tarballs).
- End users gain published sha256 sidecars for manual verification.

**Negative:**
- `brew test` still downloads the macOS tarball during install validation — unavoidable without dropping the test gate, and this is genuine validation rather than redundant work.

---

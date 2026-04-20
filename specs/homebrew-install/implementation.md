# Homebrew Install — Implementation Explanation

## Beginner Level

### What Changed

Users can now install rune with `brew install arjenschwarz/rune/rune`. Every time a new rune release is cut on GitHub, a Homebrew formula is automatically updated in a separate "tap" repository so the `brew install` command keeps working against the newest version.

### Why It Matters

Before this change, installing rune required `go install` (assumes the user has Go installed) or building from source. Homebrew is the standard package manager on macOS and popular on Linux, so supporting it lowers the barrier to entry.

### Key Concepts

- **Homebrew formula**: A Ruby file (`Formula/rune.rb`) that tells Homebrew where to download the binary and how to verify it with a sha256 checksum.
- **Tap**: A GitHub repo that hosts Homebrew formulas. `ArjenSchwarz/homebrew-rune` is rune's tap.
- **Release workflow**: The GitHub Actions workflow that runs when a new tag is published. It already builds the binaries; now it also updates the formula.
- **sha256 sidecar**: A tiny companion file (e.g., `rune-v1.3.1-darwin-arm64.tar.gz.sha256`) containing the checksum of a tarball. Used so the formula can verify downloads are intact.

---

## Intermediate Level

### Changes Overview

Three production files changed:

- `.github/workflows/release.yml`: adds `sha256sum: true` to the existing binary-matrix step (so sidecars are published), plus a new `homebrew` job that renders the formula, validates it, and commits it to the tap.
- `README.md`: adds a Homebrew subsection under Installation.
- `CHANGELOG.md`: adds an Unreleased/Added entry.

Plus spec files in `specs/homebrew-install/` (smolspec, tasks, decision log, manual steps).

### Implementation Approach

The `homebrew` job runs on `macos-latest`, gated with `needs: releases-matrix` so it only starts after all four platform tarballs are published. It then:

1. Resolves the tag from `GITHUB_REF_NAME` (release event) or `github.event.inputs.tag` (manual dispatch).
2. Downloads only the four `.sha256` sidecars (~80 bytes each), not the multi-MB tarballs, using `gh release download`.
3. Parses each sidecar with `awk` and validates that each digest is exactly 64 hex chars.
4. Renders `Formula/rune.rb` inline via an unquoted heredoc, substituting the version and four sha256 values. Ruby `#{version}` interpolations pass through untouched because they contain no shell metacharacters.
5. Runs `brew audit --strict --online` and `brew install --formula ./Formula/rune.rb && brew test rune` to validate the formula before it reaches users.
6. Checks out `ArjenSchwarz/homebrew-rune` using a fine-grained PAT (`HOMEBREW_TAP_TOKEN`), copies the formula over, and commits + pushes to `main`. The commit is skipped when the formula is byte-identical (`git diff --cached --quiet`), making re-runs idempotent.

A `concurrency` group serializes all tap writes, and `workflow_dispatch` with a `tag` input allows manual recovery without rebuilding binaries. On manual dispatch, `releases-matrix` is skipped (via `if: github.event_name == 'release'`); the homebrew job's gate `if: always() && result != 'failure' && result != 'cancelled'` allows `'skipped'` through.

### Trade-offs

- **One workflow vs two**: Chose to append the job to `release.yml` rather than use a separate `homebrew.yml` triggered by `workflow_run`. `needs:` gives deterministic ordering; `workflow_run` is racy and doesn't fire for workflow-created workflows. Documented as Decision 1.
- **Fine-grained PAT vs GitHub App**: PAT is simpler for a one-maintainer personal project. Yearly rotation is the downside. Decision 2.
- **Placeholder formula bootstrap**: The tap ships with a non-functional formula so `brew tap` works; the first real `v1.3.1` release populates the genuine one. Avoids hand-computing sha256s for the bootstrap. Decision 3.
- **Sidecars over re-download**: `wangyoucao577/go-release-action` natively supports `sha256sum: true`, so flipping it is one line. Alternative (download tarballs and shasum locally) wastes bandwidth. Decision 5.

---

## Expert Level

### Technical Deep Dive

**Tag resolution dual-mode**: The `tag` step is the load-bearing bridge between `release: published` and `workflow_dispatch`. On a release event, `GITHUB_REF_NAME` is the tag name (e.g., `v1.3.1`). On dispatch, `github.event.inputs.tag` is the user-supplied string. Both are normalized to `version = tag#v` for the formula `version` line, while the literal `v` stays in the URL template so asset filenames match (`rune-v1.3.1-darwin-amd64.tar.gz`).

**Heredoc safety**: The heredoc is unquoted (`<<EOF`) to allow shell expansion of `${VERSION}` and the four digest env vars. Ruby's `#{...}` interpolation syntax contains no shell metacharacters (`$`, backticks, `$(...)`), so it passes through unchanged. This is a fragile invariant — adding any Ruby string with a literal `$` to the template would be silently shell-expanded. Mitigated by the fact that the template is short and lives adjacent to the env var declarations.

**Digest validation**: `[ ${#v} -ne 64 ]` after `awk '{print $1}'` is sufficient because command substitution strips trailing newlines and awk strips field whitespace. A 64-hex-char string is the only valid output; anything else fails loudly.

**Idempotency**: `git diff --cached --quiet` returns 0 when staged changes are empty, which short-circuits the commit. Re-running against the same tag produces byte-identical formula output (version + sha256 are deterministic given the same tag), so the commit step becomes a no-op — meeting the "MUST NOT fail and MUST NOT create empty commits" requirement.

**Concurrency semantics**: `group: homebrew-${{ github.repository }}` with `cancel-in-progress: false` serializes all homebrew job runs across all tags. Back-to-back releases queue rather than race. Trade-off: a stuck workflow blocks subsequent releases, but tap commits must not interleave, so serialization is the correct default.

**Gating subtlety**: On `workflow_dispatch`, `releases-matrix` is skipped (its `if` evaluates false). The needed-job `result` is then `'skipped'`, not `'failure'` or `'cancelled'`, so `always() && result != 'failure' && result != 'cancelled'` passes. This is intentional per the spec (recovery mode against existing assets). If a future maintainer adds another job to `needs:`, the gate won't re-evaluate it — worth a comment but not a bug today.

### Architecture Impact

The homebrew job couples the release pipeline to the tap repo via an external credential. Failure modes:

- **PAT expired**: Fine-grained PAT maximum lifetime is 1 year. Silent expiry manifests as a `git push` auth error that fails the job loudly, but the release itself (binaries uploaded) has already succeeded — so the tap lags the release. Mitigation documented in manual-steps.md.
- **Tap default branch drift**: `git push origin HEAD:main` (explicit) protects against the tap's default branch being renamed. Originally wrote `git push origin HEAD`, corrected during review.
- **`brew audit` flakiness**: Upstream Homebrew audit rules change. A rune release could fail audit for reasons unrelated to the formula. Acceptable cost for the validation it provides (Decision 4).
- **macOS runner quota**: rune is public, so macOS minutes are free. Private-repo use would need reassessment.

### Potential Issues

- **Checkout ref on `workflow_dispatch`**: `actions/checkout@v4` with no `ref:` lands on the branch that triggered the dispatch, not the tag. The current job only reads env vars and downloads sidecars — nothing from the repo working tree — so this is safe today. A future step that reads a file at the tag (e.g., to embed changelog) would silently read the branch copy.
- **Idempotency window**: Between `brew install + test` and `git push`, the formula is installed into the runner's Homebrew but the tap hasn't been updated. If the push fails, the next re-run will still skip the commit if the tap is already ahead (desirable) or overwrite any concurrent edit (serialized by concurrency group, so not a real race).
- **`--version` drift**: The formula's `test do` block runs `rune --version`. If `--version` is ever removed or renamed, `brew test` breaks every subsequent release. Low risk; test gate catches it before tap commit.

---

## Completeness Assessment

### Requirements Coverage (smolspec.md MUST/SHOULD)

| Requirement | Where implemented |
| --- | --- |
| Formula installs on darwin-{arm64,amd64}, linux-{arm64,amd64} (MUST) | `release.yml:115-133` (`on_macos` / `on_linux` with Hardware::CPU.arm? switching) |
| Auto-commit on `release: published` after all tarballs exist (MUST) | `release.yml:47` (`needs: releases-matrix`) |
| Formula passes `brew audit --strict --online` and `brew test` on `macos-latest` (MUST) | `release.yml:145-150` |
| README documents `brew install arjenschwarz/rune/rune` alongside `go install` (MUST) | `README.md:26-30` (before the Go install block) |
| Workflow fails loudly on missing tarballs / bad checksums / audit or push failure (MUST) | Default `set -e` on bash steps; digest length check at `release.yml:90-95`; `git push` exit status propagates |
| Idempotent re-runs — same tag MUST NOT fail and MUST NOT create empty commits (MUST) | `release.yml:167-172` (`git diff --cached --quiet` guard) |
| Uses `HOMEBREW_TAP_TOKEN` repo secret (MUST) | `release.yml:155` (`token: ${{ secrets.HOMEBREW_TAP_TOKEN }}`) |
| `workflow_dispatch` with `tag` input for manual re-run (SHOULD) | `release.yml:7-12` |
| `concurrency` group keyed on repo (SHOULD) | `release.yml:50-52` |
| Inline heredoc rendering (no separate template file) | `release.yml:107-143` |
| Literal `v` in URL template | `release.yml:117, 120, 127, 130` (`v#{version}` throughout) |
| `releases-matrix` skipped on manual dispatch | `release.yml:21` (`if: github.event_name == 'release'`) |

### Fully Implemented

All five tasks in `tasks.md` (sha256 sidecars, workflow_dispatch, formula render + audit + test, tap commit + concurrency, README) map cleanly to the diff. No divergence from smolspec's implementation approach after the `HEAD:main` push correction.

### Partially Implemented

None. Every MUST and SHOULD has a corresponding line in the workflow.

### Not Yet Verified End-to-End

The workflow cannot be exercised until the first real release fires it. `manual-steps.md:19` (task 4) remains open and tracks this:

> v1.3.1 release end-to-end populates the tap with a working formula and `brew install arjenschwarz/rune/rune` installs and runs the binary.

All prerequisites (tap repo, PAT, secret) are marked complete. The next release tag will be the integration test.

### Validation Findings

No gaps found between the spec and implementation. Minor polish items addressed during review:

- `git push origin HEAD` changed to `git push origin HEAD:main` to match spec and protect against tap default-branch drift.
- Garbled `Blocked-by` metadata on `tasks.md` task 3 cleaned up.

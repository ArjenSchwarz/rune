---
references:
    - specs/homebrew-install/smolspec.md
    - specs/homebrew-install/decision_log.md
    - specs/homebrew-install/manual-steps.md
---
# Homebrew Install

- [x] 1. Release workflow publishes .sha256 sidecar files for every platform tarball (sha256sum: true enabled; verified by inspecting release assets of a test tag) <!-- id:h92gaou -->

- [x] 2. Release workflow supports manual re-run via workflow_dispatch with a tag input so the homebrew job can be exercised against an existing release without rebuilding binaries

- [x] 3. Homebrew job renders a fresh Formula/rune.rb from the published sidecars and passes brew audit --strict --online and brew test on macos-latest before attempting any push <!-- id:h92gaov -->
  - Blocked-by: h92gaou (Release workflow publishes .sha256 sidecar files for every platform tarball (sha256sum: true enabled; verified by inspecting release assets of a test tag))

- [x] 4. Homebrew job commits the rendered formula to ArjenSchwarz/homebrew-rune idempotently, with concurrency group preventing parallel runs from clobbering each other <!-- id:h92gaow -->
  - Blocked-by: h92gaov (Homebrew job renders a fresh Formula/rune.rb from the published sidecars and passes brew audit --strict --online and brew test on macos-latest before attempting any push)

- [x] 5. README documents brew install arjenschwarz/rune/rune alongside the existing install instructions

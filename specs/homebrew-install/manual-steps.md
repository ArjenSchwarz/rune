---
references:
    - specs/homebrew-install/smolspec.md
    - specs/homebrew-install/tasks.md
---
# Homebrew Install — Manual Steps

## Before first automated release

- [x] 1. Tap repo ArjenSchwarz/homebrew-rune exists with Formula/rune.rb committed using the placeholder content specified in smolspec.md <!-- id:rcb7ioi -->

- [x] 2. Fine-grained PAT scoped to ArjenSchwarz/homebrew-rune with contents:write is generated; expiry date noted on calendar for rotation <!-- id:rcb7ioj -->

- [x] 3. HOMEBREW_TAP_TOKEN secret is configured on ArjenSchwarz/rune using the PAT <!-- id:rcb7iok -->
  - Blocked-by: rcb7ioi (Tap repo ArjenSchwarz/homebrew-rune exists with Formula/rune.rb committed using the placeholder content specified in smolspec.md), rcb7ioj (Fine-grained PAT scoped to ArjenSchwarz/homebrew-rune with contents:write is generated; expiry date noted on calendar for rotation)

## After implementation lands

- [ ] 4. v1.3.1 release end-to-end populates the tap with a working formula and brew install arjenschwarz/rune/rune installs and runs the binary <!-- id:rcb7iol -->
  - Blocked-by: rcb7iok (HOMEBREW_TAP_TOKEN secret is configured on ArjenSchwarz/rune using the PAT)

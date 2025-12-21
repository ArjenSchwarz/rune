---
references:
    - specs/fix-phase-remove-bug/smolspec.md
---
# Fix Phase Remove Bug

## CLI Remove Fix

- [x] 1. CLI remove command preserves phase boundaries when removing tasks

- [x] 2. Integration test verifies CLI remove with phases matches expected output

## Batch Operations Fix

- [ ] 3. Batch remove operations process in reverse order so users can specify original task IDs

- [ ] 4. Test verifies batch removes work with original task IDs

- [ ] 5. Batch remove operations adjust phase markers after each removal

- [ ] 6. Test verifies batch removes preserve phase boundaries

- [ ] 7. All existing tests pass after changes

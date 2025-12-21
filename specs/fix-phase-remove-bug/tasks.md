---
references:
    - specs/fix-phase-remove-bug/smolspec.md
---
# Fix Phase Remove Bug

## CLI Remove Fix

- [x] 1. CLI remove command preserves phase boundaries when removing tasks

- [x] 2. Integration test verifies CLI remove with phases matches expected output

## Batch Operations Fix

- [x] 3. Batch remove operations process in reverse order so users can specify original task IDs

- [x] 4. Test verifies batch removes work with original task IDs

- [x] 5. Batch remove operations adjust phase markers after each removal

- [x] 6. Test verifies batch removes preserve phase boundaries

- [x] 7. All existing tests pass after changes

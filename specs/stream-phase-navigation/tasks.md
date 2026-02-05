---
references:
    - specs/stream-phase-navigation/requirements.md
    - specs/stream-phase-navigation/design.md
    - specs/stream-phase-navigation/decision_log.md
---
# Stream-Aware Phase Navigation Implementation

## Core Implementation

- [x] 1. Add hasReadyTaskInStream helper function <!-- id:dlv5hgs -->
  - File: internal/task/next.go - Implement helper that checks if any task in a stream is ready (pending + no owner + not blocked)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [3.3](requirements.md#3.3)

- [x] 2. Add unit tests for hasReadyTaskInStream <!-- id:dlv5hgt -->
  - File: internal/task/next_test.go - Test hasReadyTaskInStream with various task states
  - Blocked-by: dlv5hgs (Add hasReadyTaskInStream helper function)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1)

- [x] 3. Implement FindNextPhaseTasksForStream function <!-- id:dlv5hgu -->
  - File: internal/task/next.go - Find first phase with ready stream N tasks and return all stream N tasks from that phase
  - Blocked-by: dlv5hgs (Add hasReadyTaskInStream helper function)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [1.7](requirements.md#1.7), [3.2](requirements.md#3.2)

- [x] 4. Add unit tests for FindNextPhaseTasksForStream <!-- id:dlv5hgv -->
  - File: internal/task/next_test.go - Test phase discovery with various stream and blocking scenarios
  - Blocked-by: dlv5hgu (Implement FindNextPhaseTasksForStream function)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.6](requirements.md#1.6), [1.7](requirements.md#1.7), [3.2](requirements.md#3.2)

## Command Integration

- [x] 5. Modify runNextPhase for stream-aware discovery <!-- id:dlv5hgw -->
  - File: cmd/next.go - Add conditional: if streamFlag > 0 call FindNextPhaseTasksForStream else use existing behavior
  - Blocked-by: dlv5hgu (Implement FindNextPhaseTasksForStream function)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [2.1](requirements.md#2.1)

- [x] 6. Modify runNextWithClaim for phase+stream+claim <!-- id:dlv5hgx -->
  - File: cmd/next.go - Handle --phase --stream --claim: discover phase then claim only ready tasks
  - Blocked-by: dlv5hgw (Modify runNextPhase for stream-aware discovery)
  - Stream: 1
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4)

## Output Enhancements

- [ ] 7. Add blocking status to JSON output <!-- id:dlv5hgy -->
  - File: cmd/next.go - Add blocked boolean and blockedBy array (hierarchical IDs) to JSON output
  - Stream: 2
  - Requirements: [3.6](requirements.md#3.6), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)

- [ ] 8. Add blocking status to table output <!-- id:dlv5hgz -->
  - File: cmd/next.go - Show (ready) or (blocked) indicator in table Status column
  - Blocked-by: dlv5hgy (Add blocking status to JSON output)
  - Stream: 2
  - Requirements: [3.6](requirements.md#3.6), [5.4](requirements.md#5.4)

- [ ] 9. Add blocking status to markdown output <!-- id:dlv5hh0 -->
  - File: cmd/next.go - Add (blocked by: N) notation for blocked tasks in markdown output
  - Blocked-by: dlv5hgy (Add blocking status to JSON output)
  - Stream: 2
  - Requirements: [3.6](requirements.md#3.6), [5.4](requirements.md#5.4)

## Integration Tests

- [ ] 10. Add integration test for stream-aware phase navigation <!-- id:dlv5hh1 -->
  - File: cmd/integration_test.go - Test --phase --stream returns correct phase when earlier phases lack the stream
  - Blocked-by: dlv5hgw (Modify runNextPhase for stream-aware discovery)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [2.1](requirements.md#2.1), [2.2](requirements.md#2.2)

- [ ] 11. Add integration test for blocked tasks in output <!-- id:dlv5hh2 -->
  - File: cmd/integration_test.go - Verify blocked tasks appear in output with blocking status and hierarchical IDs
  - Blocked-by: dlv5hgy (Add blocking status to JSON output), dlv5hgz (Add blocking status to table output), dlv5hh0 (Add blocking status to markdown output)
  - Stream: 2
  - Requirements: [1.3](requirements.md#1.3), [3.6](requirements.md#3.6)

- [ ] 12. Add integration test for all stream tasks blocked skips phase <!-- id:dlv5hh3 -->
  - File: cmd/integration_test.go - Test phase with all stream tasks blocked is skipped
  - Blocked-by: dlv5hgw (Modify runNextPhase for stream-aware discovery)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2)

- [ ] 13. Add integration test for claim with phase and stream <!-- id:dlv5hh4 -->
  - File: cmd/integration_test.go - Test --phase --stream --claim only claims ready tasks
  - Blocked-by: dlv5hgx (Modify runNextWithClaim for phase+stream+claim)
  - Stream: 1
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3)

- [ ] 14. Add integration test for no phases returns error <!-- id:dlv5hh5 -->
  - File: cmd/integration_test.go - Test --phase --stream with no H2 headers returns error
  - Blocked-by: dlv5hgw (Modify runNextPhase for stream-aware discovery)
  - Stream: 1
  - Requirements: [1.7](requirements.md#1.7)

## Backward Compatibility Verification

- [ ] 15. Verify existing next --phase behavior unchanged <!-- id:dlv5hh6 -->
  - File: cmd/integration_test.go - Verify --phase alone returns first phase with any pending tasks
  - Blocked-by: dlv5hgw (Modify runNextPhase for stream-aware discovery)
  - Stream: 1
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3)

- [ ] 16. Verify existing next --stream behavior unchanged <!-- id:dlv5hh7 -->
  - File: cmd/integration_test.go - Verify --stream without --phase returns first ready task ignoring phases
  - Blocked-by: dlv5hgw (Modify runNextPhase for stream-aware discovery)
  - Stream: 1
  - Requirements: [2.2](requirements.md#2.2)

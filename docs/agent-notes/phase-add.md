# Phase-Aware Add

`cmd/add.go` builds a `task.Operation` from every flag and calls `TaskList.AddTaskWithPhases`, the same path batch `add` uses; see `streams-and-phases.md` for how markers are kept in step. Older tickets (T-836, T-973) describe a separate `AddTaskToPhase` call in `cmd/add.go`; that path no longer exists.

`--phase` with `--position` on a top-level task is rejected (T-973). With `--parent`, the phase is ignored and the position applies within the parent.

# Phase-Aware Add

`cmd/add.go` uses a separate path when `--phase` is provided: it calls `task.AddTaskToPhase(filename, addParent, addTitle, addPhase)` instead of building `AddOptions` and calling the normal extended add path.

`AddTaskToPhase` in `internal/task/operations.go` currently accepts only file path, parent ID, title, and phase name. It preserves phase markers and inserts top-level tasks at the end of the target phase, but it does not know about stream, owner, blocked-by, requirements, or requirements-file.

T-836 tracks the resulting bug: `rune add --phase ... --stream/--owner/--blocked-by/--requirements` silently creates the phased task while dropping those extended fields.

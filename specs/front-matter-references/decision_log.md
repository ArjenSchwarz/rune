# Front Matter References Decision Log

## Decision 1: Feature Scope Simplification
**Date:** 2025-09-01  
**Decision:** Limit feature to only adding front matter content, not editing or removing  
**Rationale:** User feedback indicated the initial scope was overly complicated. Only addition capability is needed since existing commands already display front matter.  
**Impact:** Simplified requirements focus on two commands: extending create and adding add-frontmatter

## Decision 2: CLI Flag Design
**Date:** 2025-09-01  
**Decision:** Use standard CLI patterns with --reference and --meta flags instead of bracket notation  
**Rationale:** Bracket notation (e.g., references: [file1, file2]) would cause shell escaping issues and isn't practical for CLI usage. Standard repeatable flags are more user-friendly.  
**Impact:** Commands will use --reference "file.md" (repeatable) and --meta "key:value" (repeatable) syntax

## Decision 3: Duplicate Handling Strategy
**Date:** 2025-09-01  
**Decision:** Use merge/append strategy for duplicate keys  
**Rationale:** User preference for merging rather than overwriting or erroring. References append to arrays, metadata merges with array appending where applicable.  
**Impact:** New content is additive rather than replacing existing front matter

## Decision 4: Error Handling Approach
**Date:** 2025-09-01  
**Decision:** Keep error handling minimal, relying on existing patterns  
**Rationale:** User preference to avoid overcomplicating the feature. Existing go-tasks error handling patterns should be sufficient.  
**Impact:** Requirements focus on core functionality without extensive error handling specifications

## Decision 5: Generic Front Matter Support
**Date:** 2025-09-01  
**Decision:** Support arbitrary key-value metadata in addition to references  
**Rationale:** User requested generic approach to support future front matter additions beyond just references.  
**Impact:** Feature supports both references and arbitrary metadata through separate flags

## Decision 6: No Automatic Type Inference
**Date:** 2025-09-01  
**Decision:** Keep metadata values as strings by default, no automatic type conversion  
**Rationale:** Based on design review feedback, automatic type inference creates ambiguity and edge cases (e.g., "1.0" as float vs string). Explicit is better than implicit.  
**Impact:** All metadata values stored as strings unless multiple flags create arrays

## Decision 7: Simplified Validation Approach
**Date:** 2025-09-01  
**Decision:** Only validate YAML structure integrity, not file paths or security concerns  
**Rationale:** The application doesn't read files from reference paths, only stores them as strings. Path safety is the responsibility of users and external tools that might use these references.  
**Impact:** Validation focuses only on YAML key validity and reasonable resource limits

## Decision 8: Atomic File Operations
**Date:** 2025-09-01  
**Decision:** Use write-to-temp-then-rename pattern for all file modifications  
**Rationale:** Prevents data corruption during concurrent access or system failures.  
**Impact:** All file writes go through WriteFileAtomic method

## Decision 9: Leverage Existing Infrastructure
**Date:** 2025-09-01  
**Decision:** Extend existing NewTaskList and front matter functions rather than creating parallel infrastructure  
**Rationale:** Design review revealed existing ParseFrontMatter and SerializeWithFrontMatter functions that should be reused.  
**Impact:** Minimal new code, better integration with existing codebase
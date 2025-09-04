# Rune Initial Version Decision Log

## Decision 1: In-Progress Status Representation
**Date**: 2025-08-28  
**Decision**: Use `[-]` as the markdown checkbox marker for in-progress tasks  
**Rationale**: Provides clear visual distinction between pending `[ ]`, in-progress `[-]`, and completed `[x]` states  
**Alternatives Considered**: `[~]`, `[*]`, explicit status text  
**Impact**: Simple, consistent with checkbox patterns, easy to parse  

## Decision 2: Error Handling Strategy for Malformed Files
**Date**: 2025-08-28  
**Decision**: Report errors for malformed content without attempting automatic fixes during parsing  
**Rationale**: Prevents silent data corruption, maintains user control over file content  
**Alternatives Considered**: Auto-correction, silent ignoring of errors  
**Impact**: Users must fix malformed files manually, but prevents unexpected modifications  

## Decision 3: Concurrency Requirements Scope
**Date**: 2025-08-28  
**Decision**: Focus on internal concurrency safety within a single process  
**Rationale**: Simpler implementation for initial version, most common use case  
**Alternatives Considered**: Multi-process file locking, database-style transactions  
**Impact**: Multiple processes accessing same file simultaneously not supported in v1  

## Decision 4: CLI Framework Choice
**Date**: 2025-08-28  
**Decision**: Use Cobra framework for CLI implementation  
**Rationale**: User's preferred framework, consistent with existing projects, mature ecosystem  
**Alternatives Considered**: Standard flag package, other CLI frameworks  
**Impact**: More dependencies but better UX and easier maintenance  

## Decision 5: File Extension Requirements
**Date**: 2025-08-28  
**Decision**: Accept any plain-text file regardless of extension  
**Rationale**: Maximum flexibility, content-based validation rather than extension-based  
**Alternatives Considered**: Require .md extension, configurable extensions  
**Impact**: Tool validates content structure instead of relying on file extension  

## Decision 6: ID Collision Handling Strategy
**Date**: 2025-08-28  
**Decision**: Auto-correct duplicate or invalid task IDs during parsing  
**Rationale**: Enables tool to work with manually edited files, reduces user friction  
**Alternatives Considered**: Error on ID conflicts, ignore duplicates  
**Impact**: Tool modifies file structure but maintains usability with hand-edited files  

## Decision 7: go-output/v2 Integration Scope
**Date**: 2025-08-28  
**Decision**: Delegate table format details to go-output/v2 capabilities  
**Rationale**: Avoid over-specifying table formatting, leverage library's strengths  
**Alternatives Considered**: Custom table formatting, detailed format requirements  
**Impact**: Simpler requirements, dependency on external library behavior  

## Decision 8: Batch Operation Limits
**Date**: 2025-08-28  
**Decision**: No specific limits on batch operations for initial version  
**Rationale**: Keep initial implementation simple, add limits based on real-world usage  
**Alternatives Considered**: Maximum operation counts, file size limits  
**Impact**: May need performance tuning later, but avoids premature optimization

## Decision 9: Scope Reduction to MVP
**Date**: 2025-08-28  
**Decision**: Reduce initial version scope to 7 core requirements, defer advanced features to future versions  
**Rationale**: Agent reviews identified scope as too large for initial version, risk of over-engineering  
**Features Included**: Task operations, data structures, parsing/rendering, CLI, JSON API, query/search, file format standardization  
**Features Deferred**: Advanced output formats, performance optimizations, file format flexibility, automatic data correction  
**Alternatives Considered**: Keep full scope, reduce to even smaller MVP  
**Impact**: Faster initial delivery, reduced complexity, ability to iterate based on real usage

## Decision 10: Query/Search Capabilities Addition  
**Date**: 2025-08-28  
**Decision**: Add comprehensive query and search capabilities as requirement 6  
**Rationale**: Critical for AI agents to find tasks efficiently without full file parsing  
**Features**: Search by title/content, filter by status/hierarchy, JSON output, case sensitivity options  
**Alternatives Considered**: Basic filtering only, external search tools  
**Impact**: Enhanced AI agent usability, additional implementation complexity for search algorithms

## Decision 11: Resolution of ID Auto-Correction Conflict
**Date**: 2025-08-28
**Decision**: For MVP, follow Decision #2 strictly - report errors without auto-correction
**Rationale**: Consistency and predictability are more important than convenience for initial version
**Supersedes**: Decision #6 is deferred to future version
**Alternatives Considered**: Configurable strictMode, auto-correction with warnings
**Impact**: Users must fix malformed files manually, clearer error reporting

## Decision 12: Batch Operations Transaction Model
**Date**: 2025-08-28
**Decision**: All batch operations must be atomic - all succeed or all fail
**Rationale**: Prevents partial state and data corruption in multi-operation scenarios
**Implementation**: Deep copy original, validate all ops, apply to copy, atomic write
**Alternatives Considered**: Best-effort with partial success, operation-level rollback
**Impact**: Simpler mental model, potential performance impact for large batches

## Decision 13: Security and Resource Limits
**Date**: 2025-08-28
**Decision**: Implement strict input validation and resource limits
**Rationale**: Prevent security vulnerabilities and DoS attacks
**Limits**: 10MB files, 10K tasks, 10 hierarchy levels, 1MB JSON, 100 batch ops
**Security**: Path traversal protection, input sanitization, atomic writes
**Impact**: Safe for production use, may need tuning based on real usage

## Decision 14: Design Simplification for MVP
**Date**: 2025-08-28
**Decision**: Simplify design to remove unnecessary complexity while preserving all requirements
**Rationale**: Original design was over-engineered with premature abstractions and optimizations
**Changes Made**:
  - Consolidated packages from 7+ to just 2 (cmd/ and internal/task/)
  - Removed interfaces with single implementations
  - Used standard Go error handling instead of complex error types
  - Removed premature optimizations (index maps, complex concurrency)
  - Simplified batch operations (validate-then-apply pattern)
  - Direct recursive parsing instead of complex stack-based approach
  - Kept CLI commands as separate files for clarity (per user preference)
**Alternatives Considered**: Keep complex design, partial simplification
**Impact**: Easier implementation, testing, and maintenance without losing any functionality
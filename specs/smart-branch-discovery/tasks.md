---
references:
    - specs/smart-branch-discovery/smolspec.md
---
# Smart Branch Discovery

- [x] 1. Branch prefix stripping extracts name after first slash
  - Branch specs/my-feature resolves to path with my-feature
  - Branch feature/auth resolves to path with auth

- [x] 2. Multiple candidate paths are tried in order until one exists
  - Stripped path is tried first
  - Full branch path is tried as fallback

- [x] 3. Single-component branches produce only one candidate path
  - Branch main does not try duplicate paths

- [x] 4. Existing tests updated for multi-candidate behavior
  - Tests in discovery_test.go reflect new path resolution logic

- [x] 5. New tests verify stripped vs full branch path precedence
  - Test cases cover prefixed branches finding stripped path
  - Test cases cover prefixed branches falling back to full path
  - Test cases cover branches with multiple slashes

- [x] 6. Error messages list all candidate paths that were tried
  - When no file is found, error shows which paths were attempted

# Malformed Tasks

- [ ] 1. Valid task
  - [ ] 1.1. Valid subtask
    - [ ] This line has incorrect indentation (should cause error)
- [?] 2. Invalid checkbox
- [ ] . Task without number
  - [ ] 3.1. Orphaned subtask (parent doesn't exist)
- [] 4. Missing space in checkbox
- [ ] 5. Normal task
  - Details with no checkbox marker
  - [ ] 5.1. Valid subtask
    - Extra detail line
    - References: test.md
      - [ ] 5.1.1. Too deeply indented (unexpected indentation)
# Parallel Agent Execution Example

- [ ] 1. Initialize project structure <!-- id:init001 -->
  - Stream: 1
  - Set up directories
  - Configure build system

- [ ] 2. Backend API development <!-- id:bknd002 -->
  - Blocked-by: init001 (Initialize project structure)
  - Stream: 1
  - Create REST endpoints
  - Implement business logic

- [ ] 3. Frontend UI development <!-- id:frnt003 -->
  - Blocked-by: init001 (Initialize project structure)
  - Stream: 2
  - Build component library
  - Implement views

- [ ] 4. Database schema design <!-- id:dbs0004 -->
  - Blocked-by: init001 (Initialize project structure)
  - Stream: 1
  - Design tables
  - Set up migrations

- [ ] 5. Frontend-Backend integration <!-- id:intg005 -->
  - Blocked-by: bknd002 (Backend API development), frnt003 (Frontend UI development)
  - Stream: 1
  - Connect API endpoints
  - Handle error states

- [ ] 6. End-to-end testing <!-- id:e2e0006 -->
  - Blocked-by: intg005 (Frontend-Backend integration)
  - Stream: 1
  - Write test scenarios
  - Automate test runs

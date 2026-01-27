# Parallel Agent Execution Example

This example demonstrates how to set up and manage tasks for parallel execution across multiple agents using streams and dependencies.

## Tasks

- [ ] 1. Initialize project structure <!-- id:init001 stream:1 -->
  - Set up directories
  - Configure build system

- [ ] 2. Backend API development <!-- id:backend001 stream:1 -->
  - Blocked-by: init001 (Initialize project structure)
  - Create REST endpoints
  - Implement business logic

- [ ] 3. Frontend UI development <!-- id:frontend001 stream:2 -->
  - Blocked-by: init001 (Initialize project structure)
  - Build component library
  - Implement views

- [ ] 4. Database schema design <!-- id:db001 stream:1 -->
  - Blocked-by: init001 (Initialize project structure)
  - Design tables
  - Set up migrations

- [ ] 5. Frontend-Backend integration <!-- id:integrate001 stream:1 -->
  - Blocked-by: backend001 (Backend API development), frontend001 (Frontend UI development)
  - Connect API endpoints
  - Handle error states

- [ ] 6. End-to-end testing <!-- id:e2e001 stream:1 -->
  - Blocked-by: integrate001 (Frontend-Backend integration)
  - Write test scenarios
  - Automate test runs

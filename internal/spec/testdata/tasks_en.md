# Tasks: User authentication

**Input**: Design documents from `/specs/003-auth/`
**Prerequisites**: plan.md, spec.md

## Phase 1: Setup

- [x] T001 Create project structure per implementation plan
- [x] T002 [P] Add dependencies in `go.mod`
- [x] T003 [P] Configure lint in .golangci.yml

## Phase 2: Foundational

- [x] T004 Create users table migration in db/migrations/003_users.sql
- [~] T005 [P] Implement password hasher in internal/auth/hash.go
- [ ] T006 [P] Implement session store in internal/auth/store.go

## Phase 3: User Story 1 - Log in (Priority: P1)

- [ ] T007 [US1] Add login handler in internal/http/login.go
- [ ] T008 [P] [US1] Add login form in web/src/Login.vue
- [ ] T009 [US1] Wire routes in cmd/server/main.go.

## Phase 4: Polish

- [ ] T010 Update docs/README.md

## Remediation (verify)

- [ ] T011 Reject empty passwords in internal/auth/hash.go (criterion AC-3 MISSING)
- [x] T012 Add rate limit test for login (criterion AC-5 PARTIAL)

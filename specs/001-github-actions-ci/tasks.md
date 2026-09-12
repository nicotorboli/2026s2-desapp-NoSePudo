---
description: "Task list for GitHub Actions CI Pipeline implementation"
---

# Tasks: GitHub Actions CI Pipeline

**Input**: Design documents from `/specs/001-github-actions-ci/`  
**Prerequisites**: [plan.md](file:plan.md), [spec.md](file:spec.md), [research.md](file:research.md), [data-model.md](file:data-model.md), [contracts/](file:contracts/)  
**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [TaskID] [P?] [Story?] Description with file path`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., `[US1]`, `[US2]`, `[US3]`, `[US4]`)
- Include exact file paths in descriptions

## Path Conventions

- **CI Workflows**: `.github/workflows/`
- **Git Hooks**: `.githooks/`
- **Automation Scripts**: `scripts/`
- **Backend Service**: `backend/`
- **Frontend App**: `frontend/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize directory structures for repository hooks, scripts, and GitHub Actions workflows.

- [ ] T001 Create automation scripts directory in scripts/ and git hooks directory in .githooks/
- [ ] T002 [P] Create workflow configuration directory in .github/workflows/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Baseline workflow and backend foundation required before user stories can execute.

**⚠️ CRITICAL**: Base workflow skeleton and runnable backend code must be established so that subsequent jobs can be attached and executed.

- [ ] T003 Initialize base workflow skeleton with push and pull_request triggers for main and dev branches in .github/workflows/ci.yml
- [ ] T004 [P] Create baseline Go server entrypoint using standard library net/http in backend/cmd/server/main.go
- [ ] T005 [P] Initialize Vite+React+TypeScript project in frontend/ with typecheck, lint, and knip scripts per contracts/frontend-quality-gate.md, and remove frontend/.gitkeep

**Checkpoint**: Foundation ready - workflow skeleton, baseline Go code, and frontend project exist. User story implementation can begin.

---

## Phase 3: User Story 1 - Automated Backend Verification and Testing (Priority: P1) 🎯 MVP

**Goal**: Automatically execute Go unit and integration tests and package builds on pushes and PRs targeting `main` or `dev`.

**Independent Test**: Introduce a failing test in `backend/cmd/server/main_test.go`; the `backend` job must report a failure. Revert to passing assertions; the `backend` job must succeed cleanly.

### Tests for User Story 1

- [ ] T006 [P] [US1] Create baseline unit and health check tests in backend/cmd/server/main_test.go

### Implementation for User Story 1

- [ ] T007 [US1] Implement backend verification job with Go 1.26 toolchain setup from backend/go.mod in .github/workflows/ci.yml
- [ ] T008 [US1] Add test execution step running go test -v -race ./... with Docker daemon access in .github/workflows/ci.yml
- [ ] T009 [US1] Add backend compilation step running go build -v ./... in .github/workflows/ci.yml

**Checkpoint**: At this point, User Story 1 provides a functional MVP: pushes and PRs execute Go unit/integration tests and verify compilation in GitHub Actions.

---

## Phase 4: User Story 2 - Precommit Hook Script for Formatting and Static Analysis (Priority: P1)

**Goal**: Provide a shared precommit hook script (`scripts/pre-commit.sh` and `scripts/pre-commit.ps1`) and Git hook entrypoint (`.githooks/pre-commit`) running `gofmt`, `go vet`, and `golangci-lint`, and execute it in CI.

**Independent Test**: Create an intentionally unformatted Go file (`gofmt` violation) or compiler warning; `./scripts/pre-commit.sh` must flag the issue with remediation guidance and exit with code 1. In CI, the precommit step in the `backend` job must fail with the same diagnostic output.

### Implementation for User Story 2

- [ ] T010 [P] [US2] Create static analysis configuration with correctness, security, and context linters in backend/.golangci.yml
- [ ] T011 [P] [US2] Implement POSIX precommit hook script enforcing gofmt, go vet, and golangci-lint in scripts/pre-commit.sh
- [ ] T012 [P] [US2] Implement PowerShell companion precommit hook script for Windows environments in scripts/pre-commit.ps1
- [ ] T013 [P] [US2] Create repository Git hook entrypoint delegating to scripts/pre-commit.sh in .githooks/pre-commit
- [ ] T014 [US2] Add golangci-lint v1.64.5 binary installation step to backend job in .github/workflows/ci.yml
- [ ] T015 [US2] Integrate execution of bash ./scripts/pre-commit.sh into backend job prior to tests in .github/workflows/ci.yml

**Checkpoint**: At this point, User Stories 1 and 2 are active: local development and remote CI share 100% identical formatting and static analysis checks.

---

## Phase 5: User Story 3 - Frontend Linting, Type Checking, and Static Analysis (Priority: P2)

**Goal**: Provide an automated frontend quality gate job executing TypeScript typecheck, ESLint, and Knip static analysis.

**Independent Test**: Introduce a type error (`tsc`) or lint violation (`eslint`) in `frontend/`; the `frontend` job must report a failure.

### Implementation for User Story 3

- [ ] T016 [P] [US3] Document frontend contract scripts and quality gate requirements in frontend/README.md
- [ ] T017 [US3] Implement frontend quality gate job with Node 22 setup and npm ci dependency installation in .github/workflows/ci.yml
- [ ] T018 [US3] Add typecheck, lint, and knip static analysis steps to frontend job in .github/workflows/ci.yml

**Checkpoint**: At this point, User Stories 1, 2, and 3 are complete: both backend and frontend quality gates run in parallel.

---

## Phase 6: User Story 4 - Pipeline Orchestration and Fast Feedback (Priority: P3)

**Goal**: Optimize pipeline execution through workflow-level concurrency cancellation (`cancel-in-progress: true`) and toolchain dependency caching.

**Independent Test**: Push two consecutive commits to the same branch; verify that the earlier run transitions to `cancelled`. Run a subsequent pipeline on unchanged dependencies; verify that `actions/setup-go` and `actions/setup-node` restore dependencies from cache.

### Implementation for User Story 4

- [ ] T019 [US4] Configure workflow concurrency group and in-progress run cancellation in .github/workflows/ci.yml
- [ ] T020 [US4] Enable Go module caching in actions/setup-go referencing backend/go.sum in .github/workflows/ci.yml
- [ ] T021 [US4] Enable npm caching in actions/setup-node referencing frontend/package-lock.json in .github/workflows/ci.yml
- [ ] T022 [US4] Validate independent parallel job execution and isolated PR check reporting in .github/workflows/ci.yml

**Checkpoint**: All user stories are fully implemented and integrated.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Syntax validation and verification against quickstart scenarios.

- [ ] T023 [P] Validate YAML syntax across .github/workflows/ci.yml and backend/.golangci.yml
- [ ] T024 Execute local quickstart validation scenarios from specs/001-github-actions-ci/quickstart.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion. Delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational phase completion. Can be implemented in parallel with US1 or immediately following US1.
- **User Story 3 (Phase 5)**: Depends on Foundational phase completion. Runs in parallel to backend jobs.
- **User Story 4 (Phase 6)**: Depends on US1, US2, and US3 job structures being in place to finalize caching and concurrency settings.
- **Polish (Phase 7)**: Depends on all user story tasks being completed.

### User Story Dependencies

- **User Story 1 (P1)**: Independent of US2 and US3. Establishes the `backend` job in `.github/workflows/ci.yml`.
- **User Story 2 (P1)**: Independent of US3. Produces scripts in `scripts/`, hooks in `.githooks/`, and config in `backend/.golangci.yml`, then integrates into the `backend` job.
- **User Story 3 (P2)**: Independent of US1 and US2. Establishes the `frontend` job in `.github/workflows/ci.yml`.
- **User Story 4 (P3)**: Enhances `.github/workflows/ci.yml` with concurrency and caching across both `backend` and `frontend` jobs.

### Within Each User Story

- Baseline tests before pipeline verification.
- Script and configuration creation before workflow integration.
- Quality gates configured directly for backend and frontend.

### Parallel Opportunities

- **Phase 1**: `T001` and `T002` can run in parallel.
- **Phase 2**: `T004` (server entrypoint) and `T005` (frontend project) can run in parallel.
- **Phase 3**: `T006` (unit test) can run in parallel with backend job setup.
- **Phase 4**: `T010` (.golangci.yml), `T011` (pre-commit.sh), `T012` (pre-commit.ps1), and `T013` (.githooks/pre-commit) can all be created in parallel.
- **Phase 5**: `T016` (frontend README) can be created in parallel with backend tasks.

---

## Parallel Execution Examples

### Parallel Example: User Story 2 (Precommit Infrastructure)

```bash
# Developer A or Agent A can create the linter configuration and shell scripts concurrently:
Task: "T010 [P] [US2] Create static analysis configuration with correctness, security, and context linters in backend/.golangci.yml"
Task: "T011 [P] [US2] Implement POSIX precommit hook script enforcing gofmt, go vet, and golangci-lint in scripts/pre-commit.sh"
Task: "T012 [P] [US2] Implement PowerShell companion precommit hook script for Windows environments in scripts/pre-commit.ps1"
Task: "T013 [P] [US2] Create repository Git hook entrypoint delegating to scripts/pre-commit.sh in .githooks/pre-commit"
```

### Parallel Example: Multi-Story Independent Development

```bash
# Once Foundational (Phase 2) is complete, team members can work on backend and frontend concurrently:
Developer 1: "T007 [US1] Implement backend verification job..." & "T008 [US1] Add test execution step..."
Developer 2: "T011 [P] [US2] Implement POSIX precommit hook script..." & "T010 [P] [US2] Create static analysis config..."
Developer 3: "T017 [US3] Implement frontend quality gate job..." & "T018 [US3] Add typecheck, lint, and knip steps..."
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (`T001`, `T002`)
2. Complete Phase 2: Foundational (`T003`, `T004`, `T005`)
3. Complete Phase 3: User Story 1 (`T006` - `T009`)
4. **STOP and VALIDATE**: Push a test commit or pull request targeting `dev` to verify that Go test execution and build verification pass in GitHub Actions.

### Incremental Delivery

1. **Increment 1 (Foundation + US1 MVP)**: Basic CI pipeline running backend tests on PRs and pushes to `main` and `dev`.
2. **Increment 2 (US2 Quality Gates)**: Precommit hook script enforcement locally via `.githooks` and remotely in CI with `golangci-lint`, `go vet`, and `gofmt`.
3. **Increment 3 (US3 Frontend Gate)**: Isolated `frontend` job executing type checking, linting, and knip static analysis.
4. **Increment 4 (US4 Fast Feedback)**: Concurrency cancellation and dependency caching for Go modules and npm packages.
5. **Increment 5 (Polish)**: Final syntax validations and quickstart verification.

---

## Notes

- All tasks follow strict format: `- [ ] [TaskID] [P?] [Story?] Description with file path`
- File paths are exact and unambiguous.
- Each user story is independently testable per its test criteria.
- Local Git hook activation command: `git config core.hooksPath .githooks`.

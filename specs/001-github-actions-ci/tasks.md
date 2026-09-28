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

- [X] T001 Create automation scripts directory in scripts/ and git hooks directory in .githooks/
- [X] T002 [P] Create workflow configuration directory in .github/workflows/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Baseline workflow and backend foundation required before user stories can execute.

**⚠️ CRITICAL**: Base workflow skeleton and runnable backend code must be established so that subsequent jobs can be attached and executed.

- [X] T003 Initialize base workflow skeleton with push and pull_request triggers for main and dev branches in .github/workflows/ci.yml
- [X] T004 [P] Create baseline Go server entrypoint using standard library net/http in backend/cmd/server/main.go
- [X] T005 [P] Initialize Vite+React+TypeScript project in frontend/ with typecheck, lint, and knip scripts per contracts/frontend-quality-gate.md, and remove frontend/.gitkeep

**Checkpoint**: Foundation ready - workflow skeleton, baseline Go code, and frontend project exist. User story implementation can begin.

---

## Phase 3: User Story 1 - Automated Backend Verification and Testing (Priority: P1) 🎯 MVP

**Goal**: Automatically execute Go unit and integration tests and package builds on pushes and PRs targeting `main` or `dev`.

**Independent Test**: Introduce a failing test in `backend/cmd/server/main_test.go`; the `backend` job must report a failure. Revert to passing assertions; the `backend` job must succeed cleanly.

### Tests for User Story 1

- [X] T006 [P] [US1] Create baseline unit and health check tests in backend/cmd/server/main_test.go

### Implementation for User Story 1

- [X] T007 [US1] Implement backend verification job with Go 1.26 toolchain setup from backend/go.mod in .github/workflows/ci.yml
- [X] T008 [US1] Add test execution step running go test -v -race ./... with Docker daemon access in .github/workflows/ci.yml
- [X] T009 [US1] Add backend compilation step running go build -v ./... in .github/workflows/ci.yml

**Checkpoint**: At this point, User Story 1 provides a functional MVP: pushes and PRs execute Go unit/integration tests and verify compilation in GitHub Actions.

---

## Phase 4: User Story 2 - Precommit Hook Script for Formatting and Static Analysis (Priority: P1)

**Goal**: Provide a shared precommit hook script (`scripts/pre-commit.sh`) and Git hook entrypoint (`.githooks/pre-commit`) running `gofmt`, `go vet`, and `golangci-lint`, and execute it in CI.

> **Note**: `T012` originally added a PowerShell companion (`scripts/pre-commit.ps1`). The 2026-09-21 clarification settled on a single implementation, and `T029` removed it (FR-015). `T012` is kept as a record of what was built, not of the current state.

**Independent Test**: Create an intentionally unformatted Go file (`gofmt` violation) or compiler warning; `./scripts/pre-commit.sh` must flag the issue with remediation guidance and exit with code 1. In CI, the precommit step in the `backend` job must fail with the same diagnostic output.

### Implementation for User Story 2

- [X] T010 [P] [US2] Create static analysis configuration with correctness, security, and context linters in backend/.golangci.yml
- [X] T011 [P] [US2] Implement POSIX precommit hook script enforcing gofmt, go vet, and golangci-lint in scripts/pre-commit.sh
- [X] T012 [P] [US2] Implement PowerShell companion precommit hook script for Windows environments in scripts/pre-commit.ps1
- [X] T013 [P] [US2] Create repository Git hook entrypoint delegating to scripts/pre-commit.sh in .githooks/pre-commit
- [X] T014 [US2] Add golangci-lint v1.64.5 binary installation step to backend job in .github/workflows/ci.yml
- [X] T015 [US2] Integrate execution of bash ./scripts/pre-commit.sh into backend job prior to tests in .github/workflows/ci.yml

**Checkpoint**: At this point, User Stories 1 and 2 are active: local development and remote CI share 100% identical formatting and static analysis checks.

---

## Phase 5: User Story 3 - Frontend Linting, Type Checking, and Static Analysis (Priority: P2)

**Goal**: Provide an automated frontend quality gate job executing TypeScript typecheck, ESLint, and Knip static analysis.

**Independent Test**: Introduce a type error (`tsc`) or lint violation (`eslint`) in `frontend/`; the `frontend` job must report a failure.

### Implementation for User Story 3

- [X] T016 [P] [US3] Document frontend contract scripts and quality gate requirements in frontend/README.md
- [X] T017 [US3] Implement frontend quality gate job with Node 22 setup and npm ci dependency installation in .github/workflows/ci.yml
- [X] T018 [US3] Add typecheck, lint, and knip static analysis steps to frontend job in .github/workflows/ci.yml

**Checkpoint**: At this point, User Stories 1, 2, and 3 are complete: both backend and frontend quality gates run in parallel.

---

## Phase 6: User Story 4 - Pipeline Orchestration and Fast Feedback (Priority: P3)

**Goal**: Optimize pipeline execution through workflow-level concurrency cancellation (`cancel-in-progress: true`) and toolchain dependency caching.

**Independent Test**: Push two consecutive commits to the same branch; verify that the earlier run transitions to `cancelled`. Run a subsequent pipeline on unchanged dependencies; verify that `actions/setup-go` and `actions/setup-node` restore dependencies from cache.

### Implementation for User Story 4

- [X] T019 [US4] Configure workflow concurrency group and in-progress run cancellation in .github/workflows/ci.yml
- [X] T020 [US4] Enable Go module caching in actions/setup-go referencing backend/go.sum in .github/workflows/ci.yml
- [X] T021 [US4] Enable npm caching in actions/setup-node referencing frontend/package-lock.json in .github/workflows/ci.yml
- [X] T022 [US4] Validate independent parallel job execution and isolated PR check reporting in .github/workflows/ci.yml

**Checkpoint**: All user stories are fully implemented and integrated.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Syntax validation and verification against quickstart scenarios.

- [X] T023 [P] Validate YAML syntax across .github/workflows/ci.yml and backend/.golangci.yml
- [X] T024 Execute local quickstart validation scenarios from specs/001-github-actions-ci/quickstart.md

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

---

## Phase 8: Convergence

**Purpose**: Close the gaps between the specification (including the 2026-09-21
clarifications) and the current state of the repository.

- [X] T025 CRITICAL: Add a version-controlled .gitattributes at the repository root declaring `*.go text eol=lf` and renormalize the tracked Go sources, so that gofmt yields identical results on Windows, macOS, and Linux without relying on each developer's core.autocrlf, per FR-011 (missing)
- [X] T026 Replace the local skip-with-warning branch in scripts/pre-commit.sh with a hard failure that exits non-zero and prints the golangci-lint installation command, so the script never reports [SUCCESS] having skipped a configured check, per FR-012 (contradicts)
- [X] T027 Declare the required golangci-lint version once in the repository as the single source of truth (canonical value `v2.12.1`) and make both .github/workflows/ci.yml and scripts/pre-commit.sh resolve the version from that declaration instead of hardcoding it, per FR-013 (missing)
- [X] T028 Add a version verification step to scripts/pre-commit.sh that aborts the commit when the locally installed golangci-lint differs from the declared version, reporting both the expected and the detected version, per FR-014 (missing)
- [X] T029 Delete scripts/pre-commit.ps1 so the repository keeps exactly one precommit implementation (scripts/pre-commit.sh, runnable on Linux, macOS, and Windows through Git Bash), per FR-015 (contradicts)
- [X] T030 Update the documents that still describe a PowerShell companion script to the single-script model: .specify/memory/constitution.md (Code Quality & Linting), specs/001-github-actions-ci/contracts/precommit-hook.md, specs/001-github-actions-ci/quickstart.md, specs/001-github-actions-ci/research.md, and specs/001-github-actions-ci/data-model.md, per FR-015 (partial)
- [X] T031 Add the testcontainers-backed PostgreSQL integration test that the plan's Constitution Check claims as satisfied, or record an explicit deferral, so that `go test -v -race ./...` in the backend job actually exercises a Dockerized database, per plan: testcontainers/PostgreSQL decision, US1/AC1, Constitution II & XIII (partial)
- [X] T032 Review and justify or isolate the SonarQube scan added to the backend job in .github/workflows/ci.yml (and sonar-project.properties): it is not called for by this feature's spec, plan, or tasks, and folding a third gate into the backend check blurs the per-gate status isolation, per FR-010, SC-004 (unrequested)
- [X] T033 Remove backend/.golangci.bck.yml or justify keeping a backup copy alongside the authoritative backend/.golangci.yml, per plan: static analysis configuration (unrequested)

---

## Phase 9: Deuda detectada en ejecucion real de CI

**Purpose**: Hallazgos observados en el run 35663061463 (PR #4), la primera corrida
con la Phase 8 aplicada. No surgen de leer los artefactos sino de ver el pipeline correr.

- [X] T034 Reemplazar `golangci/golangci-lint-action@v8` en el job `backend` de .github/workflows/ci.yml por una instalacion pelada del binario (script oficial o `go install`, pineado a la version de `.golangci-version`), porque el action no solo instala: ejecuta `golangci-lint run` por su cuenta. Hoy eso produce tres defectos: (a) el linter corre dos veces por corrida, una en el action y otra dentro de `scripts/pre-commit.sh`; (b) una violacion de lint aborta en un paso rotulado "Install golangci-lint", diagnostico desorientador que contradice SC-004; (c) al abortar ahi, el paso `Execute precommit hook script` nunca llega a ejecutarse, de modo que el gate que garantiza paridad entre local y CI queda sin correr — incumpliendo FR-004 y la premisa central de US2. Evidencia: run 35663061463, job 106542547468, log del paso "Install golangci-lint" mostrando `Running [...golangci-lint run --path-mode=abs]` seguido del error de `fieldalignment`. (contradicts)

- [X] T035 Cubrir con tests las funciones que el quality gate de Sonar reporto sin cobertura: mover `TestHealthDTO_Conversions` de backend/cmd/server/main_test.go a backend/internal/controller/dto/health_test.go (el test existia pero vivia en `package main`, asi que Go atribuia 0% al paquete `dto`), y agregar backend/internal/persistence/repository/health_test.go cubriendo `NewHealthRepository` y ambas ramas de `Ping`. Se excluye backend/cmd/server/main.go de `sonar.coverage.exclusions`: es pegamento de ciclo de vida del proceso, no logica testeable. Evidencia: SonarCloud PR #4, "38.8% Coverage on New Code (required >= 80%)". (partial)
- [X] T036 Corregir los hallazgos de seguridad de Sonar sobre .github/workflows/ci.yml que bajaban el Security Rating on New Code a C: `npm ci --ignore-scripts` para que las dependencias no ejecuten lifecycle scripts, y `npm run knip` en vez de `npx knip` para no instalar on-demand sin version fijada. El hallazgo bloqueante ("Use full commit SHA hash for this dependency", `golangci/golangci-lint-action@v8`) queda resuelto por T034, que elimina ese action. Evidencia: SonarCloud PR #4, "C Security Rating on New Code (required >= A)". (contradicts)
- [X] T037 Montar el runner de tests del frontend, que no existia: vitest + cobertura v8 en frontend/vite.config.ts, test de frontend/src/api/health.ts (la costura con el backend: prefijo /api, path /health y propagacion del error), y publicacion del lcov al job de sonar como artifact. Se excluye frontend/src/main.tsx de la cobertura por ser el entrypoint (createRoot().render()), mismo criterio que backend/cmd/server/main.go. El directorio generado coverage/ se ignora en .gitignore y en eslint. Nota de alcance: US3 solo pedia lint, typecheck y analisis estatico para el frontend; los tests se agregan porque el Principio XIII los exige para toda feature y el frontend tenia cero. Evidencia: SonarCloud PR #4, 12 lineas sin cubrir sobre codigo nuevo, todas del frontend. (missing)
- [X] T038 Cubrir frontend/src/App.tsx con tests de componente (@testing-library/react + jsdom): hoy es el unico archivo con logica sin tests, con sus dos caminos sin verificar — el happy path que muestra el status devuelto por getHealth y el catch que cae a 'offline'. Requiere agregar el entorno jsdom a vite.config.ts. Queda fuera de T037 porque suma dependencias y configuracion de entorno, no solo un test. (missing)
- [ ] T039 Decidir el quality gate propio en SonarCloud en vez de heredar el "Sonar way" por defecto. El umbral de 80% de cobertura sobre codigo nuevo no lo pide el enunciado ni la constitucion: el Principio XIII define tecnica de diseno de casos (clases de equivalencia, valores limite, tabla de decision) y que se mockea en cada capa, y no menciona cobertura ni una vez. Evaluar conservar las condiciones binarias (seguridad, confiabilidad, mantenibilidad, duplicacion) y reemplazar la de cobertura por una regla alineada con el XIII. Se configura en la web de SonarCloud, no en el repositorio. (unrequested)


# Feature Specification: GitHub Actions CI Pipeline

**Feature Branch**: `001-github-actions-ci`

**Created**: 2026-09-10

**Status**: Draft

**Input**: User description: "I want to create a github actions CI pipeline that runs over the pull requests and push actions to the main and dev branches. I need a job that runs the backend tests and a precommit hook script that runs formatting and static analysis;  and another one that runs the frontend linting with type checking and static analysis."

## Clarifications

### Session 2026-09-21

- Q: ¿Cómo debe garantizar el repositorio que los archivos `.go` tengan los mismos finales de línea en Windows, macOS y Linux, para que `gofmt` dé el mismo resultado local y en CI? → A: Agregar `.gitattributes` con `*.go text eol=lf` y renormalizar el repositorio, de modo que el checkout escriba LF en todos los sistemas operativos sin depender de `core.autocrlf`.
- Q: ¿Qué debe hacer el script de pre-commit cuando `golangci-lint` no está instalado en la máquina del desarrollador? → A: Fallar y abortar el commit, mostrando el comando de instalación en el mensaje de error. El script nunca debe reportar éxito habiéndose salteado un chequeo.
- Q: ¿Dónde debe declararse la versión de `golangci-lint`, y qué pasa si la versión local no coincide con la del CI? → A: Una única versión declarada en el repositorio, tomando como canónica la que ya usa el CI (`v2.12.1`). El hook aborta el commit si la versión instalada localmente no coincide.
- Q: ¿El repositorio debe mantener una o dos implementaciones del script de pre-commit? → A: Una sola, `scripts/pre-commit.sh`, que funciona en Linux, macOS y Windows (vía el Bash que incluye Git para Windows). Se elimina `scripts/pre-commit.ps1` y se actualizan los documentos que lo referencian.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automated Backend Verification and Testing (Priority: P1)

As a developer contributing to the backend, when I push changes or open a pull request targeting `main` or `dev`, an automated job runs backend unit and integration tests. A developer merging changes can trust that logic reaching the shared branches is thoroughly tested and verified.

**Why this priority**: The backend contains core business logic and API persistence. Automatically validating tests on every push and pull request protects system integrity and prevents regressions from reaching shared branches.

**Independent Test**: Push a commit with a deliberately failing backend unit test; the pipeline must report failure on the backend job. Push a commit with passing tests; the backend test job must report success.

**Acceptance Scenarios**:

1. **Given** a commit is pushed to `main` or `dev`, **When** the pipeline runs, **Then** the backend test job executes all unit and integration tests.
2. **Given** a pull request targets `main` or `dev`, **When** the pull request is opened or updated, **Then** the backend test job executes automatically.
3. **Given** any backend test fails, **When** the backend job finishes, **Then** the job is reported as failed with test failure details visible in the logs.
4. **Given** all backend tests pass, **When** the backend job finishes, **Then** the test step is marked as successful.

---

### User Story 2 - Precommit Hook Script for Formatting and Static Analysis (Priority: P1)

As a developer, I want a shared precommit hook script that checks code formatting and runs static analysis, and I want the CI pipeline to run this exact script in the backend verification job. This ensures that local development checks and CI quality gates are identical and reproducible.

**Why this priority**: Formatting inconsistencies and static code defects should be caught early before code review. Having the CI job execute the same precommit script that developers run locally guarantees consistent standards with zero discrepancy between local and remote verification.

**Independent Test**: Introduce misformatted Go code or a static analysis defect (e.g. unhandled error or dead store); running the precommit script locally must fail, and pushing the commit to CI must fail the backend formatting/static analysis step with the same diagnostic output.

**Acceptance Scenarios**:

1. **Given** code with formatting or static analysis violations, **When** the precommit hook script is run locally, **Then** it flags the violations and exits with a non-zero error code.
2. **Given** code is pushed to `main` or `dev`, **When** the backend CI job executes the precommit script, **Then** formatting and static analysis are verified.
3. **Given** code passes formatting and static analysis, **When** the precommit script runs in CI, **Then** the step succeeds without warning or error.
4. **Given** the precommit hook script is maintained in the repository, **When** developers set up their local repository, **Then** they can install or run the script directly as a git pre-commit hook or standalone command.
5. **Given** a required analysis tool is missing from the developer's PATH, **When** the precommit hook script runs locally, **Then** it exits with a non-zero code and reports the installation command, rather than skipping the check and reporting success.

---

### User Story 3 - Frontend Linting, Type Checking, and Static Analysis (Priority: P2)

As a frontend developer, when I push code or open a pull request targeting `main` or `dev`, an automated frontend job validates TypeScript types, runs linting rules, and performs static analysis (such as detecting dead or unused code).

**Why this priority**: Static type safety and linting prevent runtime exceptions, UI defects, and style inconsistencies. This gate ensures high code quality before merging.

**Independent Test**: Introduce a TypeScript type error or a lint violation in the frontend; the frontend CI job must fail and highlight the exact file and line of the violation.

**Acceptance Scenarios**:

1. **Given** a commit is pushed to `main` or `dev`, **When** the frontend CI job runs, **Then** TypeScript type checking executes and reports any type discrepancies as failures.
2. **Given** a commit is pushed to `main` or `dev`, **When** the frontend CI job runs, **Then** linting executes and reports rule violations as failures.
3. **Given** a commit is pushed to `main` or `dev`, **When** the frontend CI job runs, **Then** static analysis executes and flags unused code, dead exports, or dependency issues.

---

### User Story 4 - Pipeline Orchestration and Fast Feedback (Priority: P3)

As a software team member, I want CI jobs to run concurrently where appropriate, reuse dependency caches across runs, and cancel outdated in-flight runs when new commits are pushed, so that feedback on pull requests is quick and CI resource usage is optimized.

**Why this priority**: Fast and reliable feedback encourages developers to submit smaller, more frequent pull requests without waiting on slow, redundant builds.

**Independent Test**: Push two commits to the same branch in rapid succession; verify that the initial run is cancelled in favor of the latest commit, and that dependency caching reduces execution time on the second run.

**Acceptance Scenarios**:

1. **Given** multiple pushes occur on the same branch or pull request, **When** a newer run is triggered, **Then** any running prior execution for that branch is cancelled automatically.
2. **Given** a previous CI run has cached dependencies, **When** a new run executes without dependency changes, **Then** dependency restoration completes significantly faster than a clean install.
3. **Given** a pull request check completes, **When** the developer views the pull request, **Then** clear individual check statuses for backend verification and frontend verification are visible.

---

### Edge Cases

- What happens if backend integration tests require a running database?
  In compliance with project standards, database dependencies are provided as containerized services during test execution. If the database fails to start, the job fails fast with explicit connection diagnostics.
- What happens if a developer runs the precommit hook script on Windows, macOS, or Linux?
  Go source files are normalized to LF line endings via `.gitattributes`, so `gofmt`
  produces identical results on every developer machine and on the Linux CI runner,
  regardless of each developer's local `core.autocrlf` setting.
- What happens if a pull request contains changes only to documentation or non-code files?
  The pipeline triggers appropriately for PR verification, but cache and step evaluations ensure minimal execution overhead.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The pipeline MUST trigger on all pushes to the `main` and `dev` branches.
- **FR-002**: The pipeline MUST trigger on all pull requests targeting the `main` or `dev` branches.
- **FR-003**: The pipeline MUST define a backend job responsible for executing backend unit and integration tests.
- **FR-004**: The pipeline MUST execute a repository-managed precommit hook script that verifies backend code formatting and performs static analysis.
- **FR-005**: The precommit hook script MUST be executable both locally by developers prior to committing and remotely within the CI backend job.
- **FR-006**: The backend job MUST fail if any unit or integration test fails, or if the precommit hook script detects formatting or static analysis violations.
- **FR-007**: The pipeline MUST define a frontend job responsible for running frontend linting, type checking, and static analysis.
- **FR-008**: The pipeline MUST cancel in-progress runs on the same branch or pull request when a newer commit is pushed.
- **FR-009**: The pipeline MUST utilize dependency caching to expedite repeated pipeline executions.
- **FR-010**: The pipeline MUST present clear and isolated status checks for backend and frontend jobs on pull requests, enabling immediate identification of failures.
- **FR-011**: The repository MUST enforce LF line endings for Go source files through a
  version-controlled `.gitattributes`, so that formatting verification yields identical
  results on Windows, macOS, and Linux without per-developer Git configuration.
- **FR-012**: The precommit hook script MUST fail with a non-zero exit code and surface
  the installation command when a required analysis tool is absent from the environment.
  It MUST NOT report success when any configured check was skipped.
- **FR-013**: The required `golangci-lint` version MUST be declared once in the repository
  as the single source of truth, and both the CI workflow and the precommit hook script
  MUST resolve it from that declaration. The canonical version is the one currently used
  by CI (`v2.12.1`).
- **FR-014**: The precommit hook script MUST abort the commit when the locally installed
  `golangci-lint` version differs from the declared version, reporting both the expected
  and the detected version.
- **FR-015**: The repository MUST maintain exactly one implementation of the precommit
  hook script (`scripts/pre-commit.sh`), executable on Linux, macOS, and Windows through
  the Bash runtime bundled with Git. Duplicate per-platform implementations MUST NOT be
  kept, since only the invoked one would be exercised by the hook and by CI.

### Key Entities *(include if feature involves data)*

This feature does not introduce persistent application domain entities. It manages the lifecycle of ephemeral CI workflow entities:

- **Pipeline Run**: An execution instance of the GitHub Actions workflow triggered by a push or pull request event on `main` or `dev`.
- **Backend Quality Gate**: A composite check encompassing backend test execution and precommit hook script validation (formatting and static analysis).
- **Frontend Quality Gate**: A check validating frontend TypeScript types, lint standards, and static analysis.
- **Precommit Hook Script**: A version-controlled automation script callable locally by git hooks/developers and remotely by CI to enforce formatting and static analysis rules.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of pushes and pull requests targeting `main` and `dev` automatically trigger CI quality checks without manual intervention.
- **SC-002**: Automated feedback on pull requests is reported to the developer within 10 minutes of commit publication.
- **SC-003**: 100% of formatting or static analysis issues flagged by the CI backend job can be reproduced and caught locally using the precommit hook script.
- **SC-004**: Developers can determine which specific check failed (backend tests, precommit formatting/analysis, or frontend checks) directly from the GitHub check status interface without re-running tasks locally.
- **SC-005**: Subsequent pipeline runs on unchanged dependencies achieve at least a 40% reduction in setup and run duration compared to a clean run.

## Assumptions

- Backend tests requiring PostgreSQL will run against containerized database instances in CI and test environments, fulfilling project architectural constraints.
- The precommit hook script will reside in the repository (e.g., under a scripts directory) so that developers and CI workflows execute the exact same commands.
- The frontend application exposes standard commands for linting, type checking, and static analysis.
- Pull request status checks can be used as merge requirements for `main` and `dev`.

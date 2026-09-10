# Research: GitHub Actions CI Pipeline

**Feature Branch**: `001-github-actions-ci`  
**Date**: 2026-09-10  
**Status**: Completed  

---

## 1. Pipeline Architecture: Single Workflow with Parallel Quality Gate Jobs

### Decision
Implement a single unified GitHub Actions workflow file `.github/workflows/ci.yml` named `CI Pipeline` that triggers on all `push` and `pull_request` events targeting `main` and `dev`. The workflow orchestrates two isolated, parallel jobs:
1. `backend`: Runs Go setup, executes the repository-managed precommit hook script (code formatting via `gofmt` and static analysis via `go vet` and `golangci-lint`), and executes unit and integration tests (`go test -v -race ./...`).
2. `frontend`: Checks if the frontend application is scaffolded; if not, skips gracefully with an informational notice; if scaffolded, runs dependency installation (`npm ci`), TypeScript type checking (`npm run typecheck`), linting (`npm run lint`), and dead-code static analysis (`npx knip`).

### Rationale
- **Direct Alignment with Requirements**: The user requested "a github actions CI pipeline that runs over the pull requests and push actions to the main and dev branches. I need a job that runs the backend tests and a precommit hook script that runs formatting and static analysis; and another one that runs the frontend linting with type checking and static analysis." A single workflow containing two parallel jobs matches the specification verbatim.
- **Isolated Status Checks on Pull Requests (FR-011, SC-004)**: In GitHub Actions, jobs within the same workflow file appear as distinct checks on pull requests (e.g. `CI Pipeline / Backend Verification & Tests` and `CI Pipeline / Frontend Verification`). Developers can immediately identify whether a backend test, a precommit lint violation, or a frontend type error caused a failure.
- **Independent Execution & Concurrent Feedback**: GitHub Actions schedules the `backend` and `frontend` jobs concurrently on separate `ubuntu-latest` runners. A failure in one job does not prevent the other from completing and reporting its diagnostic findings.
- **Branch Protection Compatibility**: When GitHub branch protection rules require checks to pass before merging into `main` or `dev`, workflow-level triggers without path exclusion guarantee that required check names are always posted, avoiding perpetual pending status on documentation or single-component changes.

### Alternatives Considered
- *Separate workflow files (`backend-ci.yml` and `frontend-ci.yml`)*: Would duplicate workflow-level boilerplate (concurrency declarations, branch triggers) and complicate branch protection rule configuration when PRs touch only documentation.
- *Sequential single job*: Running precommit, tests, and frontend tasks sequentially in a single runner. Rejected because sequential execution multiplies wall-clock duration (violating SC-002, 10-minute feedback goal) and obscures failure attribution.

---

## 2. Repository-Managed Precommit Hook Script Architecture

### Decision
Provide a centralized, version-controlled precommit script located at `scripts/pre-commit.sh` (POSIX bash, executable on Linux, macOS, and Git Bash on Windows) accompanied by a companion PowerShell script `scripts/pre-commit.ps1` for native Windows environments, and integrate it with Git hooks via `.githooks/pre-commit`.
The CI `backend` job directly invokes `./scripts/pre-commit.sh` as an explicit step prior to executing tests.

The script executes the following checks:
1. **Formatting Check**: Verifies that all Go source files conform to standard formatting via `gofmt -l`. If any file requires formatting, the script prints the offending filenames, provides remediation instructions (`gofmt -w .`), and exits with code 1.
2. **Standard Static Analysis**: Runs `go vet ./...` in `backend/` to detect common compiler-level anomalies.
3. **Advanced Static Analysis**: Runs `golangci-lint run ./...` using configuration from `backend/.golangci.yml`. In CI environments (`CI=true`), `golangci-lint` is mandatory; in local developer environments, if `golangci-lint` is not installed, the script emits a clear warning and guidance on installation while passing formatting and `go vet` checks.

### Rationale
- **100% Parity Between Local & CI Quality Gates (FR-004, FR-005, SC-003)**: Running the identical script in both local development (manually or via git pre-commit hooks) and CI guarantees zero discrepancy. Any static analysis or formatting violation flagged in CI can be reproduced and resolved locally before pushing.
- **Cross-Platform Developer Support**: Developers on Linux, macOS, and Windows (via Git Bash or PowerShell) can run checks with a single command (`./scripts/pre-commit.sh` or `pwsh ./scripts/pre-commit.ps1`).
- **Seamless Git Hook Integration**: By maintaining `.githooks/pre-commit` in the repository, developers can enable automated git pre-commit checks with a single command: `git config core.hooksPath .githooks`.

### Alternatives Considered
- *Third-party frameworks like Python `pre-commit`*: Introduces external dependencies (Python, virtualenvs, YAML config files) into a pure Go + TypeScript project. Rejected in favor of lightweight, dependency-free shell and PowerShell scripts.
- *Running linters inline in CI without a local script*: Fails requirement FR-004/FR-005 and leads to "commit-push-wait" debugging cycles where developers push broken code to CI because they cannot execute the exact CI checks locally.

---

## 3. Go Toolchain Resolution & Static Analysis Suite

### Decision
1. **Toolchain Version Resolution**: Use `actions/setup-go@v5` configured with `go-version-file: 'backend/go.mod'`.
2. **Static Analysis Linters (`backend/.golangci.yml`)**:
   - **Correctness**: `errcheck`, `staticcheck`, `govet`, `ineffassign`, `unused`, `nilerr`
   - **Database & Resources (Constitution II, X)**: `sqlclosecheck`, `rowserrcheck`, `bodyclose`, `noctx`
   - **Security & Quality (Constitution VII)**: `gosec`, `gocyclo`, `gocritic`, `revive`, `unconvert`
3. **CI Installation of `golangci-lint`**: In the `backend` job, install `golangci-lint` binary into `$(go env GOPATH)/bin` (pinned version v1.64.5) via the official installer before invoking `./scripts/pre-commit.sh`.

### Rationale
- **Single Source of Truth**: `backend/go.mod` specifies `go 1.26.5`. Deriving the Go version directly from `go.mod` prevents version drift.
- **Constitutional Alignment**: Constitution Principle II mandates direct `database/sql` without ORM; Principle VIII mandates context-bound logging and correlation IDs. The selected linters (`sqlclosecheck`, `rowserrcheck`, `noctx`) catch unclosed database rows, statement leaks, and missing contexts during static analysis before code reaches review.

### Alternatives Considered
- *Hardcoding Go version in workflow YAML*: Prone to drift whenever Go version in `go.mod` is updated.
- *Running `golangci/golangci-lint-action` instead of calling `pre-commit.sh`*: Violates FR-004 ("The pipeline MUST execute a repository-managed precommit hook script"). Installing `golangci-lint` into PATH allows `pre-commit.sh` to run uniformly.

---

## 4. PostgreSQL Integration Testing & Docker Environment

### Decision
Execute unit and integration tests inside the `backend` job with `go test -v -race ./...`. Database integration tests utilize `testcontainers-go` connecting to the runner's native Docker daemon.

### Rationale
- **Constitution Principles II & XIII Compliance**:
  - *"La base de datos debe estar dockerizada en todos los entornos (desarrollo, CI y test)"*
  - *"repository: contra una base real levantada con testcontainers, para no alterar la base de datos real."*
- **Native GitHub Runner Support**: GitHub-hosted `ubuntu-latest` runners provide a pre-installed, active Docker daemon with full socket access (`/var/run/docker.sock`). No service containers, Docker-in-Docker, or privileged flags are required.
- **Isolation & Concurrency**: Testcontainers provisions transient PostgreSQL containers on dynamic ports, ensuring each test run is completely isolated without cross-test data pollution.

### Alternatives Considered
- *Static GitHub Actions `services.postgres` container*: Rejected because Constitution Principle XIII specifically mandates `testcontainers` for repository testing, and static containers do not isolate state between parallel test packages.

---

## 5. Caching Strategy for Go & Frontend Dependencies

### Decision
- **Backend Caching**: Configure `actions/setup-go@v5` with `cache: true` and `cache-dependency-path: 'backend/go.sum'`. If `backend/go.sum` is not yet present, the step falls back gracefully to uncached execution until external dependencies are committed.
- **Frontend Caching**: Configure `actions/setup-node@v4` with `cache: 'npm'` and `cache-dependency-path: 'frontend/package-lock.json'` inside the conditional scaffold block.

### Rationale
- **Performance Optimization (FR-010, SC-005)**: Reusing module caches cuts subsequent pipeline durations by >40%, meeting the fast feedback objective.
- **Zero Configuration Overhead**: Native caching in official actions (`setup-go`, `setup-node`) handles cache keys, lockfile hashing, and fallback invalidation automatically.

### Alternatives Considered
- *Generic `actions/cache` step*: Adds verbose cache path maintenance and hash keys compared to native action-level caching.

---

## 6. Unscaffolded Frontend Handling

### Decision
The `frontend` job executes an initial step `Check scaffold state` that checks for `frontend/package.json`:
```bash
if [ ! -f "frontend/package.json" ]; then
  echo "Frontend is not yet scaffolded (frontend/package.json not found)."
  echo "Skipping frontend verification cleanly."
  echo "scaffolded=false" >> $GITHUB_OUTPUT
else
  echo "scaffolded=true" >> $GITHUB_OUTPUT
fi
```
Subsequent steps in the `frontend` job (Node setup, `npm ci`, typecheck, lint, knip) are conditioned on `if: steps.check-scaffold.outputs.scaffolded == 'true'`.

### Rationale
- **Graceful Skip & Green Verdict (FR-008, Edge Case)**: When `frontend/` contains only `.gitkeep`, the job skips cleanly, emits an informative log, and finishes with status `success` (exit code 0).
- **Merge Block Prevention**: If GitHub branch protection requires the `Frontend Verification` check to pass, an exit code of 0 registers a green passing check rather than failing or hanging the PR.

### Alternatives Considered
- *Job-level `if: hashFiles('frontend/package.json') != ''`*: GitHub Actions marks skipped jobs as "Skipped". Some branch protection configurations treat "Skipped" as pending or failing. A step-level condition that finishes with `success` guarantees compatibility.

---

## 7. Pipeline Concurrency & Outdated Run Cancellation

### Decision
Define workflow-level concurrency in `.github/workflows/ci.yml`:
```yaml
concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

### Rationale
- **Fast Feedback & Resource Efficiency (FR-009, SC-001)**: When a developer pushes a new commit to an open pull request, any running execution for that branch is immediately cancelled, freeing GitHub runner capacity and ensuring developers only wait on feedback for the latest code.
- **Branch Scoping**: Using `${{ github.ref }}` ensures that pushes to separate branches or PRs do not cancel each other.

### Alternatives Considered
- *No concurrency block*: Leads to redundant queued builds, wasting CI runner minutes and causing delayed feedback.

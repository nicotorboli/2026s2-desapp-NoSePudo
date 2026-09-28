# Implementation Plan: GitHub Actions CI Pipeline

**Branch**: `001-github-actions-ci` | **Date**: 2026-09-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-github-actions-ci/spec.md`

## Summary

Implement a unified GitHub Actions CI pipeline (`.github/workflows/ci.yml`) triggering on pushes to `main` and `dev` and pull requests targeting those branches. The pipeline executes two parallel, isolated quality gate jobs:
1. `backend`: Resolves Go 1.26 from `backend/go.mod`, configures module caching, installs `golangci-lint` at the version declared in `.golangci-version`, executes the repository-managed precommit hook script (`scripts/pre-commit.sh`, enforcing `gofmt`, `go vet`, and `golangci-lint`), and executes the test suite (`go test -v -race ./...`) plus a compilation check.
2. `frontend`: Configures Node.js 22 LTS with npm dependency caching, runs `npm ci`, and executes TypeScript compilation check (`tsc --noEmit`), ESLint linting (`eslint .`), and dead-code static analysis (`knip`).

A third job, `sonar`, runs after `backend` and consumes its coverage artifact. It is a
separate job rather than a step of `backend` so that a failing quality gate turns its
own check red instead of the one reporting on tests (FR-010, SC-004).

The workflow incorporates workflow-level concurrency cancellation (`cancel-in-progress: true`) and provides repository-managed git hooks in `.githooks/pre-commit` so that local development gates and remote CI verification remain 100% identical.

## Technical Context

**Language/Version**: Go 1.26.5 (backend, defined in `backend/go.mod`); Node.js 22 LTS / React + TypeScript (frontend)

**Primary Dependencies**: Go standard library (`net/http`); `golangci-lint` (version declared once in `.golangci-version`, currently `v2.12.1`); React + TypeScript (frontend); ESLint; `knip`

**Storage**: none exercised yet. The backend has no `database/sql` usage and `backend/go.mod` declares no dependencies; the repository layer is a stub. When persistence lands, PostgreSQL will be provided inside integration test runs via `testcontainers-go` against the runner's Docker daemon (see Constitution Check II & XIII below).

**Testing**: `go test -v -race ./...` (currently unit tests only); frontend typecheck (`tsc`), linting (`eslint`), and static analysis (`knip`) as quality gates

**Target Platform**: GitHub Actions Linux runner (`ubuntu-latest` with Docker pre-installed)

**Project Type**: Full-stack web application monorepo (Go API in `backend/` + React frontend in `frontend/`)

**Performance Goals**: Pull request feedback within 10 minutes (SC-002); dependency caching achieves ≥40% duration reduction on subsequent runs (SC-005)

**Constraints**: Backend restricted to Go standard library for HTTP and DB access; Go version derived dynamically from `backend/go.mod` (no hardcoding); Precommit hook script runs identically locally and in CI; Frontend job executes standard quality gates (typecheck, lint, dead code analysis)

**Scale/Scope**: Unified CI workflow orchestrating parallel quality gate jobs (`backend`, `frontend`) for a single repository and team

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Requirement | Satisfied by | Status |
|-----------|-------------|--------------|--------|
| I. Backend en Go con net/http | Build and tests must target the Go module in `backend/` | `actions/setup-go` with `go-version-file: backend/go.mod`, running `go test` and `go build` within `backend/` | PASS |
| II. Persistencia en PostgreSQL | PostgreSQL must be dockerized in CI and test | Nothing to verify yet: no persistence code exists. `ubuntu-latest` ships a Docker daemon, so `testcontainers-go` will work with no pipeline change once a feature introduces a database. | DEFERRED |
| III. Arquitectura en capas | Tests and linters must inspect all architectural layers | Static analysis and tests execute over `./...` across controller, service, repository, model, adapters | PASS |
| IV. DTOs en el borde | Model and DTO separation enforced | Validated by unit tests, integration tests, and static analysis | PASS |
| V. Frontend React + TypeScript | Frontend gates must validate React + TypeScript module | TypeScript typecheck (`tsc`), ESLint, and `knip` configured in `frontend` job | PASS |
| VIII. Observabilidad | Structured logging and context propagation | Static analysis includes `noctx` to enforce context propagation; CI logs provide isolated job verdicts | PASS |
| XIII. Testing | Unit and integration tests must run in CI | `backend` job runs `go test -v -race ./...`, which picks up integration tests automatically as they are added. Only unit tests exist today. | PARTIAL |

No constitution violations detected. Principle II is deferred rather than violated:
this feature builds the pipeline, and there is no database code for it to exercise.
See Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/001-github-actions-ci/
├── spec.md                     # Feature specification
├── plan.md                     # This file (/speckit-plan output)
├── research.md                 # Phase 0 output: decisions, rationales, alternatives
├── data-model.md               # Phase 1 output: pipeline entities, jobs, gate verdicts, state machine
├── quickstart.md               # Phase 1 output: local & remote validation scenarios
├── contracts/                  # Phase 1 output: formal interface contracts
│   ├── ci-workflow.md          # Contract for .github/workflows/ci.yml
│   ├── precommit-hook.md       # Contract for scripts/pre-commit.sh
│   └── frontend-quality-gate.md# Contract for frontend quality gate scripts & checks
└── checklists/
    └── requirements.md         # Specification quality checklist
```

### Source Code (repository root)

```text
.github/
└── workflows/
    └── ci.yml                  # Unified GitHub Actions CI pipeline definition

.githooks/
    └── pre-commit              # Git hook entrypoint delegating to scripts/pre-commit.sh

scripts/
└── pre-commit.sh               # Repository-managed precommit hook script (POSIX bash, single implementation)

.gitattributes                  # Forces LF for *.go and *.sh on every OS (FR-011)
.golangci-version               # Single source of truth for the golangci-lint version (FR-013)

backend/
├── go.mod                      # Go version source of truth (go 1.26.5)
└── .golangci.yml               # golangci-lint static analysis configuration

frontend/
├── package.json                # Frontend package configuration & scripts
└── src/                        # React + TypeScript application source
```

**Structure Decision**: 
A single workflow definition `.github/workflows/ci.yml` contains two parallel jobs (`backend` and `frontend`).
- The `backend` job installs `golangci-lint`, executes the version-controlled `./scripts/pre-commit.sh`, and runs the test suite (`go test -v -race ./...`).
- The `frontend` job sets up Node.js 22 LTS with npm caching, runs `npm ci`, and executes `tsc --noEmit`, `eslint .`, and `knip`.
- The `sonar` job runs after `backend`, downloads its coverage artifact, and runs the SonarQube scan as its own PR check.
- The precommit script lives at `scripts/pre-commit.sh` and is the repository's only implementation; on Windows it runs under the Bash bundled with Git. Developers can activate local git pre-commit checks via `git config core.hooksPath .githooks`.

## Pipeline Design

### 1. Unified Workflow Definition (`.github/workflows/ci.yml`)

#### Triggers
- `push`: branches `[main, dev]`
- `pull_request`: branches `[main, dev]`
- Concurrency: `group: ci-${{ github.workflow }}-${{ github.ref }}`, `cancel-in-progress: true`

#### Jobs Summary
| Job ID | Job Name | Runner | Working Dir | Key Steps | Fail Conditions |
|---|---|---|---|---|---|
| `backend` | `Backend Verification & Tests` | `ubuntu-latest` | `backend` | checkout, setup-go (`go-version-file: backend/go.mod`), resolve version from `.golangci-version`, setup-golangci-lint, run `./scripts/pre-commit.sh`, `go test -v -race ./...`, `go build -v ./...`, upload coverage artifact | Misformatted code; vet warning; linter violation; test failure; compile error |
| `frontend` | `Frontend Verification` | `ubuntu-latest` | `frontend` | checkout, setup-node (`node-version: 22` with npm cache), `npm ci`, `tsc --noEmit`, `eslint .`, `npx knip` | Fails on type error, lint violation, or dead-code issue |
| `sonar` | `SonarQube Analysis` | `ubuntu-latest` | repo root | `needs: backend`, checkout (`fetch-depth: 0`), download coverage artifact, SonarQube scan | Fails on SonarQube quality gate (`sonar.qualitygate.wait=true`) |

### 2. Precommit Hook Script (`scripts/pre-commit.sh`)

The precommit script enforces quality gates both locally and in CI. Before any check
runs, it resolves the required `golangci-lint` version from `.golangci-version` and
aborts if the binary is missing or its version differs, reporting expected vs detected
(FR-012, FR-014). It never exits `0` having skipped a configured check, because a
commit that passes locally must fail for the same reasons it would fail in CI.

1. **Formatting**: `gofmt -l .` inside `backend/`. Fails with non-zero exit code and lists files if any unformatted code is detected.
2. **Standard Static Analysis**: `go vet ./...` inside `backend/`. Fails if compiler static analysis detects defects.
3. **Advanced Static Analysis**: `golangci-lint run ./...` inside `backend/` using `backend/.golangci.yml`. Fails on any enabled linter violation.

### 2.1 Line Ending Normalization

`.gitattributes` declares `*.go text eol=lf` and `*.sh text eol=lf`, so the checkout
writes LF on every OS. Without it, a Windows working tree gets CRLF and `gofmt` flags
files that are correct on the Linux runner, and a CRLF `pre-commit.sh` fails on the
runner with a misleading syntax error (FR-011).

### 3. Frontend Quality Gate Suite

The `frontend` job executes quality gates against `frontend/`:
1. **Dependencies**: `npm ci` utilizing npm cache path `frontend/package-lock.json`.
2. **Type Checking**: `npm run typecheck` (`tsc --noEmit`) to verify static types without emitting JavaScript.
3. **Linting**: `npm run lint` (`eslint .`) enforcing React and TypeScript linting standards.
4. **Dead Code Detection**: `npx knip` verifying that no unused files, exports, or dependencies remain.

## Post-Design Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Backend en Go con net/http | PASS | Uses standard Go toolchain (`go build`, `go test`, `go vet`) within `backend/`. Go version sourced from `backend/go.mod`. |
| II. Persistencia en PostgreSQL | DEFERRED | No persistence code exists to test. The CI runner provides a native Docker daemon, so `testcontainers-go` needs no pipeline change when it lands. |
| III. Arquitectura en capas | PASS | Tests and linters execute over all internal packages (`./...`). |
| IV. DTOs en el borde | PASS | Unit/integration tests and static analysis linters validate boundary types. |
| V. Frontend React + TypeScript | PASS | Quality gates (`tsc`, `eslint`, `knip`) enforce React+TS standards. |
| VI. API REST documentada con OpenAPI | PASS | Ready to enforce contract validation tests once OpenAPI specs are merged. |
| VII. Seguridad JWT | PASS | No secret leaks; static analysis includes `gosec` security linter. |
| VIII. Observabilidad | PASS | Step-level clarity, isolated PR check status, and `noctx` linter enforce context propagation. |
| XIII. Testing | PARTIAL | Executes `go test -v -race ./...`; only unit tests exist today. Integration tests are picked up automatically as they are added. |

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle II (PostgreSQL dockerizado) not exercised | This feature delivers the CI pipeline. The backend has no `database/sql` code and `backend/go.mod` has no dependencies, so a `testcontainers-go` test would have no query to run and would exist only to satisfy the checklist. | Writing a placeholder integration test that spins up PostgreSQL and asserts nothing: it would add ~40s per CI run and a dependency, while verifying no application behavior. The obligation moves to the first feature that introduces persistence. |

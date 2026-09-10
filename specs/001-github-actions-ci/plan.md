# Implementation Plan: GitHub Actions CI Pipeline

**Branch**: `001-github-actions-ci` | **Date**: 2026-09-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-github-actions-ci/spec.md`

## Summary

Implement a unified GitHub Actions CI pipeline (`.github/workflows/ci.yml`) triggering on pushes to `main` and `dev` and pull requests targeting those branches. The pipeline executes two parallel, isolated quality gate jobs:
1. `backend`: Resolves Go 1.26 from `backend/go.mod`, configures module caching, installs `golangci-lint`, executes a repository-managed precommit hook script (`scripts/pre-commit.sh` / `scripts/pre-commit.ps1` enforcing `gofmt`, `go vet`, and `golangci-lint`), and executes unit and integration tests (`go test -v -race ./...` utilizing testcontainers against Dockerized PostgreSQL).
2. `frontend`: Detects whether `frontend/package.json` exists; if unscaffolded, cleanly skips with an informational notice (exit code 0); if scaffolded, executes `npm ci` with caching, TypeScript compilation check (`tsc --noEmit`), ESLint linting (`eslint .`), and dead-code static analysis (`knip`).

The workflow incorporates workflow-level concurrency cancellation (`cancel-in-progress: true`) and provides repository-managed git hooks in `.githooks/pre-commit` so that local development gates and remote CI verification remain 100% identical.

## Technical Context

**Language/Version**: Go 1.26.5 (backend, defined in `backend/go.mod`); Node.js 22 LTS / React + TypeScript (frontend)

**Primary Dependencies**: Go standard library (`net/http`, `database/sql`); `golangci-lint` (v1.64.5); React + TypeScript + axios (frontend); ESLint; `knip`

**Storage**: PostgreSQL (provided inside integration test runs via `testcontainers-go` connecting to the runner's Docker daemon, satisfying Constitution Principle II & XIII)

**Testing**: `go test -v -race ./...` (unit and integration tests); frontend typecheck (`tsc`), linting (`eslint`), and static analysis (`knip`) as quality gates

**Target Platform**: GitHub Actions Linux runner (`ubuntu-latest` with Docker pre-installed)

**Project Type**: Full-stack web application monorepo (Go API in `backend/` + React frontend in `frontend/`)

**Performance Goals**: Pull request feedback within 10 minutes (SC-002); dependency caching achieves ≥40% duration reduction on subsequent runs (SC-005)

**Constraints**: Backend restricted to Go standard library for HTTP and DB access; Go version derived dynamically from `backend/go.mod` (no hardcoding); Precommit hook script runs identically locally and in CI; Frontend job skips cleanly (exit 0) when unscaffolded

**Scale/Scope**: Unified CI workflow orchestrating parallel quality gate jobs (`backend`, `frontend`) for a single repository and team

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Requirement | Satisfied by | Status |
|-----------|-------------|--------------|--------|
| I. Backend en Go con net/http | Build and tests must target the Go module in `backend/` | `actions/setup-go` with `go-version-file: backend/go.mod`, running `go test` and `go build` within `backend/` | PASS |
| II. Persistencia en PostgreSQL | PostgreSQL must be dockerized in CI and test | Integration tests utilize `testcontainers-go` connecting to the native Docker daemon on `ubuntu-latest` | PASS |
| III. Arquitectura en capas | Tests and linters must inspect all architectural layers | Static analysis and tests execute over `./...` across controller, service, repository, model, adapters | PASS |
| IV. DTOs en el borde | Model and DTO separation enforced | Validated by unit tests, integration tests, and static analysis | PASS |
| V. Frontend React + TypeScript | Frontend gates must validate React + TypeScript module | TypeScript typecheck (`tsc`), ESLint, and `knip` configured in `frontend` job | PASS |
| VIII. Observabilidad | Structured logging and context propagation | Static analysis includes `noctx` to enforce context propagation; CI logs provide isolated job verdicts | PASS |
| XIII. Testing | Unit and integration tests must run in CI | `backend` job runs `go test -v -race ./...` covering unit and integration tests | PASS |

No constitution violations detected; no complexity-tracking entries required.

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
│   ├── precommit-hook.md       # Contract for scripts/pre-commit.sh & pre-commit.ps1
│   └── frontend-quality-gate.md# Contract for frontend scripts & scaffold guard
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
├── pre-commit.sh               # Repository-managed precommit hook script (POSIX bash)
└── pre-commit.ps1              # Repository-managed precommit hook script (PowerShell)

backend/
├── go.mod                      # Go version source of truth (go 1.26.5)
└── .golangci.yml               # golangci-lint static analysis configuration

frontend/
└── .gitkeep                    # Unscaffolded frontend placeholder
```

**Structure Decision**: 
A single workflow definition `.github/workflows/ci.yml` contains two parallel jobs (`backend` and `frontend`).
- The `backend` job installs `golangci-lint`, executes the version-controlled `./scripts/pre-commit.sh`, and runs all unit and integration tests (`go test -v -race ./...`).
- The `frontend` job verifies whether `frontend/package.json` exists, exiting cleanly (exit code 0) if unscaffolded, or running `npm ci`, `tsc --noEmit`, `eslint .`, and `knip` once scaffolded.
- The precommit script lives at `scripts/pre-commit.sh` and is mirrored for PowerShell at `scripts/pre-commit.ps1`. Developers can activate local git pre-commit checks via `git config core.hooksPath .githooks`.

## Pipeline Design

### 1. Unified Workflow Definition (`.github/workflows/ci.yml`)

#### Triggers
- `push`: branches `[main, dev]`
- `pull_request`: branches `[main, dev]`
- Concurrency: `group: ci-${{ github.workflow }}-${{ github.ref }}`, `cancel-in-progress: true`

#### Jobs Summary
| Job ID | Job Name | Runner | Working Dir | Key Steps | Fail Conditions |
|---|---|---|---|---|---|
| `backend` | `Backend Verification & Tests` | `ubuntu-latest` | `backend` | checkout, setup-go (`go-version-file: backend/go.mod`), setup-golangci-lint, run `./scripts/pre-commit.sh`, `go test -v -race ./...`, `go build -v ./...` | Misformatted code; vet warning; linter violation; test failure; compile error |
| `frontend` | `Frontend Verification` | `ubuntu-latest` | `frontend` | checkout, scaffold guard check, setup-node (`node-version: 22`), `npm ci`, `tsc --noEmit`, `eslint .`, `npx knip` | Skipped cleanly if `frontend/package.json` absent; otherwise fails on type, lint, or dead-code violation |

### 2. Precommit Hook Script (`scripts/pre-commit.sh` & `scripts/pre-commit.ps1`)

The precommit script enforces quality gates both locally and in CI:
1. **Formatting**: `gofmt -l .` inside `backend/`. Fails with non-zero exit code and lists files if any unformatted code is detected.
2. **Standard Static Analysis**: `go vet ./...` inside `backend/`. Fails if compiler static analysis detects defects.
3. **Advanced Static Analysis**: `golangci-lint run ./...` inside `backend/` using `backend/.golangci.yml`. Fails on any enabled linter violation. In CI (`CI=true`), `golangci-lint` must be present. Locally, if not installed, a warning is emitted.

### 3. Frontend Scaffold Guard

In the `frontend` job, the initial step inspects `frontend/package.json`:
- If missing: Emits `Frontend is not yet scaffolded; skipping checks.`, sets step output `scaffolded=false`, and bypasses subsequent steps. The job concludes with `success` (green checkmark).
- If present: Sets `scaffolded=true`, triggering Node.js setup, package installation, type checking, linting, and dead-code detection.

## Post-Design Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Backend en Go con net/http | PASS | Uses standard Go toolchain (`go build`, `go test`, `go vet`) within `backend/`. Go version sourced from `backend/go.mod`. |
| II. Persistencia en PostgreSQL | PASS | CI runner provides native Docker daemon for testcontainers PostgreSQL testing. |
| III. Arquitectura en capas | PASS | Tests and linters execute over all internal packages (`./...`). |
| IV. DTOs en el borde | PASS | Unit/integration tests and static analysis linters validate boundary types. |
| V. Frontend React + TypeScript | PASS | Quality gates (`tsc`, `eslint`, `knip`) enforce React+TS standards once scaffolded. |
| VI. API REST documentada con OpenAPI | PASS | Ready to enforce contract validation tests once OpenAPI specs are merged. |
| VII. Seguridad JWT | PASS | No secret leaks; static analysis includes `gosec` security linter. |
| VIII. Observabilidad | PASS | Step-level clarity, isolated PR check status, and `noctx` linter enforce context propagation. |
| XIII. Testing | PASS | Executes `go test -v -race ./...` running unit and integration tests against real Dockerized PostgreSQL. |

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

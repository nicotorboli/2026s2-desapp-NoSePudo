# Interface Contract: GitHub Actions CI Workflow

**Feature Branch**: `001-github-actions-ci`  
**Contract Version**: 1.1.0  
**Target File**: `.github/workflows/ci.yml`  

---

## 1. Workflow Metadata, Triggers & Concurrency

```yaml
name: CI Pipeline

on:
  push:
    branches:
      - main
      - dev
  pull_request:
    branches:
      - main
      - dev

concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

### Invariants & Guarantees
- **Trigger Invariant (FR-001, FR-002)**: Triggers on all pushes to `main` and `dev` and all pull requests targeting `main` and `dev`.
- **Cancellation Guarantee (FR-009, SC-001)**: Superseded commits on the same branch or PR automatically cancel in-flight CI runs to preserve runner minutes and accelerate feedback.
- **Independent Parallelism (FR-011)**: `backend` and `frontend` execute as independent parallel jobs, reporting isolated status checks to pull requests.

---

## 2. Jobs Specification

### 2.1 `backend` (Backend Verification & Tests)
- **Job Identifier**: `backend`
- **Display Name**: `Backend Verification & Tests`
- **Runner**: `ubuntu-latest`
- **Capabilities Required**: Docker daemon active (standard on GitHub `ubuntu-latest` runner) for `testcontainers-go`.
- **Steps Execution Order**:
  1. **Checkout**: `actions/checkout@v4`
  2. **Setup Go Toolchain**: `actions/setup-go@v5` with:
     - `go-version-file: backend/go.mod`
     - `cache: true`
     - `cache-dependency-path: backend/go.sum`
  3. **Setup golangci-lint**:
     ```bash
     curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.5
     ```
  4. **Execute Precommit Hook Script (FR-004, FR-005, FR-006)**:
     ```bash
     ./scripts/pre-commit.sh
     ```
  5. **Execute Backend Tests (FR-003, FR-006)**:
     ```bash
     cd backend && go test -v -race ./...
     ```
  6. **Compile Backend Packages**:
     ```bash
     cd backend && go build -v ./...
     ```
- **Exit Code Contract**: Returns 0 if and only if formatting, `go vet`, `golangci-lint`, unit tests, integration tests, and build all complete with zero errors. Any failure terminates the job immediately with a non-zero exit code.

---

### 2.2 `frontend` (Frontend Verification)
- **Job Identifier**: `frontend`
- **Display Name**: `Frontend Verification`
- **Runner**: `ubuntu-latest`
- **Steps Execution Order**:
  1. **Checkout**: `actions/checkout@v4`
  2. **Scaffold Guard Check (FR-008)**:
     ```bash
     if [ ! -f "frontend/package.json" ]; then
       echo "Frontend is not yet scaffolded (frontend/package.json not found)."
       echo "Skipping frontend verification cleanly."
       echo "scaffolded=false" >> $GITHUB_OUTPUT
     else
       echo "scaffolded=true" >> $GITHUB_OUTPUT
     fi
     ```
  3. **Setup Node.js Toolchain**:
     - `if: steps.scaffold-guard.outputs.scaffolded == 'true'`
     - `actions/setup-node@v4` with `node-version: 22`, `cache: 'npm'`, `cache-dependency-path: frontend/package-lock.json`
  4. **Install Dependencies**:
     - `if: steps.scaffold-guard.outputs.scaffolded == 'true'`
     - `working-directory: frontend`
     - `run: npm ci`
  5. **Run TypeScript Type Checking (FR-007)**:
     - `if: steps.scaffold-guard.outputs.scaffolded == 'true'`
     - `working-directory: frontend`
     - `run: npm run typecheck`
  6. **Run ESLint Linting (FR-007)**:
     - `if: steps.scaffold-guard.outputs.scaffolded == 'true'`
     - `working-directory: frontend`
     - `run: npm run lint`
  7. **Run Static Analysis / Dead Code Detection (FR-007)**:
     - `if: steps.scaffold-guard.outputs.scaffolded == 'true'`
     - `working-directory: frontend`
     - `run: npx knip`
- **Exit Code Contract**: Returns 0 if `frontend/package.json` is missing (clean pass); if scaffolded, returns 0 only when typecheck, lint, and knip all succeed without violations.

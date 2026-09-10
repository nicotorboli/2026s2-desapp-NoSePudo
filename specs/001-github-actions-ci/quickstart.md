# Quickstart Validation Guide: GitHub Actions CI Pipeline

**Feature Branch**: `001-github-actions-ci`  
**Date**: 2026-09-10  
**Status**: Completed  

This guide provides runnable scenarios to validate the CI pipeline and precommit hook scripts both locally and end-to-end on GitHub Actions.

---

## Prerequisites

1. **Go Toolchain**: Go `1.26.x` installed locally, matching [backend/go.mod](../../backend/go.mod).
2. **Git**: Repository cloned with branch `001-github-actions-ci` checked out.
3. **Docker Engine**: Installed and active (required for `testcontainers-go` database tests).
4. **Shell**: Bash (Linux, macOS, Git Bash) or PowerShell Core (Windows).

---

## Scenario 1: Validate Workflow & Linter Configuration Syntax

### Objective
Ensure `.github/workflows/ci.yml` and `backend/.golangci.yml` are syntactically valid.

### Validation Steps
From repository root:
```bash
python -c "import yaml; [yaml.safe_load(open(f)) for f in ['.github/workflows/ci.yml', 'backend/.golangci.yml']]; print('All YAML configurations are valid!')"
```
Or with PowerShell:
```powershell
python -c "import yaml; [yaml.safe_load(open(f)) for f in ['.github/workflows/ci.yml', 'backend/.golangci.yml']]; print('All YAML configurations are valid!')"
```

### Expected Outcome
Outputs `All YAML configurations are valid!` without parse exceptions.

---

## Scenario 2: Validate Precommit Hook Script Locally

### Objective
Verify that `scripts/pre-commit.sh` and `scripts/pre-commit.ps1` enforce Go formatting (`gofmt`), static analysis (`go vet`), and linter rules (`golangci-lint`) with exact diagnostic reporting.

### Validation Steps

#### 2.1 Happy Path
Execute the precommit script from the repository root:
```bash
# Bash / Git Bash:
./scripts/pre-commit.sh

# Or PowerShell:
pwsh ./scripts/pre-commit.ps1
```

**Expected Outcome**:
- Scans `backend/` files with `gofmt -l`.
- Runs `go vet ./...`.
- Runs `golangci-lint run ./...` (if installed).
- Exits with status code `0` and prints:
  ```text
  [SUCCESS] All backend pre-commit checks passed.
  ```

#### 2.2 Negative Path: Formatting Violation (SC-003)
1. Introduce an intentionally misformatted Go file:
   ```bash
   echo "package main; func main() { println(\"test\") }" > backend/cmd/unformatted.go
   ```
2. Run the script:
   ```bash
   ./scripts/pre-commit.sh
   ```
3. Observe output:
   - Identifies `backend/cmd/unformatted.go` as misformatted.
   - Recommends `gofmt -w backend/`.
   - Exits with non-zero exit code (`1`).
4. Cleanup:
   ```bash
   rm backend/cmd/unformatted.go
   ```

---

## Scenario 3: Verify Git Hook Integration

### Objective
Ensure developers can activate git pre-commit checks locally via repository-managed hooks.

### Validation Steps
1. Configure git hooks path:
   ```bash
   git config core.hooksPath .githooks
   ```
2. Stage a misformatted Go file and attempt to commit:
   ```bash
   echo "package main; func dummy() {}" > backend/cmd/dummy.go
   git add backend/cmd/dummy.go
   git commit -m "test commit"
   ```

### Expected Outcome
Git aborts the commit immediately, displaying diagnostic output from `./scripts/pre-commit.sh`. Cleaning up `git reset HEAD backend/cmd/dummy.go && rm backend/cmd/dummy.go` restores repository cleanliness.

---

## Scenario 4: Validate Backend Tests with Testcontainers Locally

### Objective
Verify that unit and integration tests run cleanly against containerized PostgreSQL.

### Validation Steps
From repository root:
```bash
cd backend
go test -v -race ./...
```

### Expected Outcome
`go test` compiles test packages, provisions transient PostgreSQL containers via `testcontainers-go`, runs all test assertions, and exits with code `0`.

---

## Scenario 5: Validate Frontend Scaffold Guard Locally

### Objective
Ensure the frontend quality gate cleanly skips when `frontend/package.json` is absent (FR-008).

### Validation Steps
From repository root:
```bash
if [ ! -f "frontend/package.json" ]; then
  echo "Frontend is not yet scaffolded (frontend/package.json not found)."
  echo "Skipping frontend verification cleanly."
  exit 0
fi
```

### Expected Outcome
Emits `Frontend is not yet scaffolded; skipping checks.` and exits immediately with code `0`.

---

## Scenario 6: End-to-End Pipeline Execution on GitHub Actions

### Objective
Validate workflow execution, job parallelism, concurrency cancellation, and pull request status checks.

### Validation Steps
1. Push branch `001-github-actions-ci` to origin:
   ```bash
   git push origin 001-github-actions-ci
   ```
2. Open a Pull Request targeting `dev` on GitHub.
3. Observe the **Checks** section on the PR:
   - `CI Pipeline / Backend Verification & Tests` executes setup-go, precommit hook script, and tests.
   - `CI Pipeline / Frontend Verification` executes scaffold guard and concludes with green success.
   - Both jobs run concurrently.
   - Wall-clock time completes within 10 minutes (SC-002).
4. **Cancellation Test (FR-009)**:
   - While the workflow is running, push another commit to `001-github-actions-ci`.
   - Verify that the previous run immediately transitions to `cancelled`.
5. **Negative Test (FR-006, SC-004)**:
   - Push a commit containing an intentional lint or test error.
   - Verify that `Backend Verification & Tests` turns red with detailed logs, while `Frontend Verification` remains green.
   - Revert the bad commit before merging.

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
4. **Shell**: Bash (Linux, macOS, or the Bash bundled with Git for Windows).
5. **golangci-lint**: installed at exactly the version declared in [.golangci-version](../../.golangci-version). The precommit script aborts otherwise.

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
Verify that `scripts/pre-commit.sh` enforces Go formatting (`gofmt`), static analysis (`go vet`), and linter rules (`golangci-lint`) with exact diagnostic reporting, and that it never reports success having skipped a check.

### Validation Steps

#### 2.1 Happy Path
Execute the precommit script from the repository root:
```bash
# Bash, on Linux, macOS, or Git Bash on Windows:
./scripts/pre-commit.sh
```

**Expected Outcome**:
- Resolves the required linter version from `.golangci-version`.
- Scans `backend/` files with `gofmt -l`.
- Runs `go vet ./...`.
- Runs `golangci-lint run ./...`.
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

#### 2.3 Negative Path: Linter Absent (FR-012)
1. Run the script with a PATH that does not contain `golangci-lint`:
   ```bash
   env PATH="/usr/bin:/bin" ./scripts/pre-commit.sh
   ```
2. Observe output:
   - Reports that `golangci-lint` is required and was not found.
   - Prints the `go install ...@<declared version>` command.
   - Exits with non-zero exit code (`1`). It must **not** print `[SUCCESS]`.

#### 2.4 Negative Path: Linter Version Mismatch (FR-014)
1. Temporarily declare a different version:
   ```bash
   cp .golangci-version /tmp/gv.bak && printf 'v2.9.0
' > .golangci-version
   ```
2. Run the script:
   ```bash
   ./scripts/pre-commit.sh
   ```
3. Observe output:
   - Reports the version mismatch with both `expected:` and `detected:` values.
   - Exits with non-zero exit code (`1`), before running any check.
4. Cleanup:
   ```bash
   cp /tmp/gv.bak .golangci-version && rm /tmp/gv.bak
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

## Scenario 4: Validate Backend Tests Locally

### Objective
Verify that the backend test suite runs cleanly under the race detector.

### Validation Steps
From repository root:
```bash
cd backend
go test -v -race ./...
```

### Expected Outcome
`go test` compiles test packages, runs all test assertions, and exits with code `0`.

> **Note on PostgreSQL**: the backend does not talk to a database yet — the repository
> layer is still a stub with no `database/sql` usage and `backend/go.mod` has no
> dependencies. The `testcontainers-go` integration test described in the plan is
> therefore **deferred** until a feature actually introduces persistence; there is no
> query to exercise. The CI runner already provides a Docker daemon, so no pipeline
> change will be required when that test lands. See plan.md, Constitution Check II/XIII.

---

## Scenario 5: End-to-End Pipeline Execution on GitHub Actions

### Objective
Validate workflow execution, job parallelism, concurrency cancellation, and pull request status checks.

### Validation Steps
1. Push branch `001-github-actions-ci` to origin:
   ```bash
   git push origin 001-github-actions-ci
   ```
2. Open a Pull Request targeting `dev` on GitHub.
3. Observe the **Checks** section on the PR:
   - `CI Pipeline / Backend Verification & Tests` executes setup-go, precommit hook script, tests, and build.
   - `CI Pipeline / Frontend Verification` executes setup-node, npm ci, typecheck, lint, and knip.
   - `CI Pipeline / SonarQube Analysis` runs after the backend job, consuming its coverage artifact.
   - `backend` and `frontend` run concurrently; `sonar` waits only on `backend`.
   - Each gate reports its own check, so a red status names the gate that failed (FR-010, SC-004).
   - Wall-clock time completes within 10 minutes (SC-002).
4. **Cancellation Test (FR-008)**:
   - While the workflow is running, push another commit to `001-github-actions-ci`.
   - Verify that the previous run immediately transitions to `cancelled`.
5. **Negative Test (FR-006, SC-004)**:
   - Push a commit containing an intentional lint or test error.
   - Verify that `Backend Verification & Tests` turns red with detailed logs, while `Frontend Verification` remains green and `SonarQube Analysis` is skipped.
   - Revert the bad commit before merging.

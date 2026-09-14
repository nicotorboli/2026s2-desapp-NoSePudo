#!/usr/bin/env bash
set -eo pipefail

# Locate repository root dynamically
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BACKEND_DIR="${REPO_ROOT}/backend"

if [ ! -d "${BACKEND_DIR}" ]; then
  echo "[ERROR] Backend directory not found at ${BACKEND_DIR}"
  exit 1
fi

cd "${BACKEND_DIR}"

echo "[INFO] Running backend pre-commit checks in ${BACKEND_DIR}..."

# Step 1: Check Go Formatting via gofmt -l
echo "[INFO] Checking Go formatting..."
UNFORMATTED=$(gofmt -l .)
if [ -n "${UNFORMATTED}" ]; then
  echo "[ERROR] The following Go files are not formatted according to gofmt:"
  echo "${UNFORMATTED}"
  echo "To fix, run: gofmt -w backend/"
  exit 1
fi

# Step 2: Run go vet ./...
echo "[INFO] Running go vet ./..."
if ! go vet ./...; then
  echo "[ERROR] go vet reported compiler/analysis defects. Review diagnostic output above."
  exit 1
fi

# Step 3: Run golangci-lint run ./...
echo "[INFO] Running golangci-lint..."
if command -v golangci-lint >/dev/null 2>&1; then
  if ! golangci-lint run ./...; then
    echo "[ERROR] golangci-lint reported rule violations according to backend/.golangci.yml."
    exit 1
  fi
else
  if [ "${CI}" = "true" ]; then
    echo "[ERROR] golangci-lint is required in CI environment but was not found in PATH."
    exit 1
  else
    echo "[WARNING] golangci-lint is not installed locally. Skipping advanced static analysis."
    echo "To install golangci-lint: https://golangci-lint.run/welcome/install/"
  fi
fi

echo "[SUCCESS] All backend pre-commit checks passed."
exit 0

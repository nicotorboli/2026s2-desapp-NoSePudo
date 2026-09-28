#!/usr/bin/env bash
set -eo pipefail

# Locate repository root dynamically
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BACKEND_DIR="${REPO_ROOT}/backend"
GOLANGCI_VERSION_FILE="${REPO_ROOT}/.golangci-version"

# read_required_golangci_version prints the single source of truth for the
# golangci-lint version. CI and this script both resolve it from here, so a
# version bump never has to be applied in two places.
read_required_golangci_version() {
  if [ ! -f "${GOLANGCI_VERSION_FILE}" ]; then
    echo "[ERROR] Version declaration not found at ${GOLANGCI_VERSION_FILE}" >&2
    echo "This file is the single source of truth for the golangci-lint version." >&2
    exit 1
  fi

  local version
  version="$(tr -d '[:space:]' < "${GOLANGCI_VERSION_FILE}")"
  if [ -z "${version}" ]; then
    echo "[ERROR] ${GOLANGCI_VERSION_FILE} is empty; expected a version such as v2.12.1" >&2
    exit 1
  fi

  echo "${version}"
}

# detect_installed_golangci_version prints the semantic version reported by the
# golangci-lint binary on PATH, without the leading "v".
detect_installed_golangci_version() {
  golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
}

# require_golangci_lint aborts unless golangci-lint is installed at the exact
# declared version. A skipped check must never be reported as success: a commit
# that passes locally has to fail for the same reasons it would fail in CI.
require_golangci_lint() {
  local required="$1"
  local required_bare="${required#v}"

  if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "[ERROR] golangci-lint is required but was not found in PATH." >&2
    echo "Install the declared version with:" >&2
    echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${required}" >&2
    echo "Other installation methods: https://golangci-lint.run/welcome/install/" >&2
    exit 1
  fi

  local detected
  detected="$(detect_installed_golangci_version)"
  if [ -z "${detected}" ]; then
    echo "[ERROR] Unable to determine the installed golangci-lint version." >&2
    echo "  expected: ${required_bare}" >&2
    echo "  'golangci-lint --version' produced no recognizable version string." >&2
    exit 1
  fi

  if [ "${detected}" != "${required_bare}" ]; then
    echo "[ERROR] golangci-lint version mismatch: local checks would not match CI." >&2
    echo "  expected: ${required_bare} (declared in .golangci-version)" >&2
    echo "  detected: ${detected}" >&2
    echo "Install the declared version with:" >&2
    echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${required}" >&2
    exit 1
  fi
}

if [ ! -d "${BACKEND_DIR}" ]; then
  echo "[ERROR] Backend directory not found at ${BACKEND_DIR}"
  exit 1
fi

REQUIRED_GOLANGCI_VERSION="$(read_required_golangci_version)"
require_golangci_lint "${REQUIRED_GOLANGCI_VERSION}"

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
echo "[INFO] Running golangci-lint ${REQUIRED_GOLANGCI_VERSION}..."
if ! golangci-lint run ./...; then
  echo "[ERROR] golangci-lint reported rule violations according to backend/.golangci.yml."
  exit 1
fi

echo "[SUCCESS] All backend pre-commit checks passed."
exit 0

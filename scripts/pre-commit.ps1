#!/usr/bin/env pwsh
$ErrorActionPreference = 'Stop'

# Locate repository root dynamically
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$BackendDir = Join-Path $RepoRoot "backend"

if (-not (Test-Path $BackendDir -PathType Container)) {
    Write-Error "[ERROR] Backend directory not found at $BackendDir"
    exit 1
}

Set-Location $BackendDir

Write-Output "[INFO] Running backend pre-commit checks in $BackendDir..."

# Step 1: Check Go Formatting via gofmt -l
Write-Output "[INFO] Checking Go formatting..."
$Unformatted = & gofmt -l .
if ($LASTEXITCODE -ne 0) {
    Write-Error "[ERROR] gofmt command failed."
    exit 1
}

if ($Unformatted) {
    Write-Output "[ERROR] The following Go files are not formatted according to gofmt:"
    $Unformatted | ForEach-Object { Write-Output $_ }
    Write-Output "To fix, run: gofmt -w backend/"
    exit 1
}

# Step 2: Run go vet ./...
Write-Output "[INFO] Running go vet ./..."
& go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Output "[ERROR] go vet reported compiler/analysis defects. Review diagnostic output above."
    exit 1
}

# Step 3: Run golangci-lint run ./...
Write-Output "[INFO] Running golangci-lint..."
$LinterCmd = Get-Command golangci-lint -ErrorAction SilentlyContinue

if ($null -ne $LinterCmd) {
    & golangci-lint run ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Output "[ERROR] golangci-lint reported rule violations according to backend/.golangci.yml."
        exit 1
    }
} else {
    if ($env:CI -eq "true") {
        Write-Output "[ERROR] golangci-lint is required in CI environment but was not found in PATH."
        exit 1
    } else {
        Write-Output "[WARNING] golangci-lint is not installed locally. Skipping advanced static analysis."
        Write-Output "To install golangci-lint: https://golangci-lint.run/welcome/install/"
    }
}

Write-Output "[SUCCESS] All backend pre-commit checks passed."
exit 0

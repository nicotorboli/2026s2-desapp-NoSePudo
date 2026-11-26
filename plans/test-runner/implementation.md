# Implementation: backend test runner

Date: 2026-11-18 · Branch: `rf/reestructurar-agents-workflow` · Language: English

## Goal

Separate unit tests from Docker/testcontainers integration tests without editing tests or
adding build tags, and produce readable output plus coverage statistics.

## Decisions

| Decision | Choice |
| --- | --- |
| Scope | Backend only; no root Makefile, no frontend targets |
| Location | `backend/Makefile` (extends the existing file) |
| Classification | Dynamic via `go list -test`; no tags, no test edits |
| Formatter | Shell + `jq` over `go test -json` |
| CI | Unchanged; `make coverage-ci` reproduces its exact flags |

## Files

| File | Change |
| --- | --- |
| `backend/Makefile` | Rewritten: `help`, `run`, `test`, `test-integration`, `test-e2e`, `test-all`, `test-packages`, `coverage-unit`, `coverage`, `coverage-ci`, `clean-coverage`, `require-jq`, `require-docker` |
| `backend/scripts/test-report.sh` | New: formats `go test -json` from stdin |
| `backend/scripts/coverage-report.sh` | New: merges profiles, prints table, optional HTML and threshold |

No test files, frontend files, CI files, `go.mod`, or `.gitignore` changed.

## Classification

`TESTED` is every package with test files; `INTEGRATION` is every tested package whose
`TestImports` or `XTestImports` include `internal/testsupport` or `testcontainers`;
`UNIT` is the difference. Current split: 12 unit, 4 integration (`persistence/dao`,
`persistence/repository`, `service`, `test/e2e`). `make test-packages` prints both lists.

## Behavior

- `make test` runs unit packages with `-race -count=1`; no Docker.
- Integration targets run `docker info` first and exit 2 with an actionable message.
- `make test` changed meaning from "everything" to "unit only"; `make test-all` is the old
  full run with formatted output.
- `coverage-unit` and `coverage` use `-covermode=atomic -coverpkg=./...`, write profiles under
  `coverage/`, merge them, and print a worst-first table plus `TOTAL`; `--html` renders
  `coverage/index.html`, `--min` gates the total.
- `coverage-ci` runs the untouched CI command and reports on `coverage.out`.
- `RACE=0` drops `-race`; `COUNT`, `SLOWEST`, `COVER_MIN`, `NO_COLOR` are honored.

## Verification

| Check | Result |
| --- | --- |
| `bash -n` on both scripts | pass |
| `make help` | pass |
| `make test-packages` | 12 unit / 4 integration, correct |
| `make test` | 12 packages pass, 240 tests |
| Synthetic build failure and failing subtest through `test-report.sh` | correct output, exit 1 |
| `make test UNIT=... RACE=0` | pass, no race flags |
| `make coverage-unit UNIT="./internal/model ./internal/dto"` | table, merged profile, HTML, exit 0 |
| `coverage-report.sh` merge, `--min`, mode mismatch, empty profile | correct; exit 0/1/2 as expected |
| `make test-integration`, `make test-e2e` without Docker | exit 2 with the Docker message |
| `PATH` without `jq` on `test-report.sh` | exit 2 with install hint |
| `./scripts/pre-commit.sh` | 0 issues |

Docker-backed runs on 2026-11-25:

| Check | Result |
| --- | --- |
| `make test-all` | 16 packages pass: 12 unit + 4 integration |
| `make coverage` | merged report generated |
| `make coverage-ci` | verified; one-shot total matches the merged `make coverage` total |

## Known expected failure

None. The four P2P routes were added to `backend/internal/server/routes_test.go` on 2026-11-25,
so `internal/server` and `make test` are green. `make test-all`, `make coverage`, and
`make coverage-ci` were also run and passed.

## Notes

- `README.md:93-95` still claims tests skip without Docker; they fail loudly in `TestMain`.
  Left untouched because existing Spanish docs are frozen.
- `backend/Makefile~` is untracked and stale; left in place.
- Coverage artifacts (`backend/coverage/`, `backend/coverage.out`) are already ignored.

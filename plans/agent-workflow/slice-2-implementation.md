# Implementation: design workflow slice 2

Date: 2026-11-25 · Branch: `rf/reestructurar-agents-workflow` · Language: English

## Summary

Finished slice 2 (context docs, ADRs, design skills), added the four P2P routes to the route
test, and calibrated both design skills on real tasks. The backend test runner was already
implemented; its record was corrected now that `make test` is green.

## Modifications

| File | Change |
| --- | --- |
| `backend/internal/server/routes_test.go` | Added `P2P: controller.NewP2PController(nil)` and the four `authenticated` P2P routes to the expected map. Only test edit, explicitly authorized. |
| `docs/architecture-backend.md` | New: layers, P2P walkthrough, change placement, package map, asymmetries. |
| `docs/domain.md` | New: credits, ledger, positions, orders/trades, quotes/scores, P2P machine, invariants. |
| `docs/errors.md` | New: sentinel ownership, translation rule, status tables, known gaps. |
| `docs/ui.md` | New: tokens, BEM, 15 components, 7 pages, routing, 768px, states, rules with exceptions. |
| `docs/adr/0001-layered-flow.md` … `0006-dto-package-split.md` | New: six one-page decisions. |
| `.opencode/skills/design-backend/SKILL.md` + `references/decision-template.md` | New skill and template. |
| `.opencode/skills/design-frontend/SKILL.md` + `references/decision-template.md` | New skill and template. |
| `AGENTS.md` | Two Agent workflow rows for the design skills. |
| `docs/designs/2026-11-25-p2p-price-units.md` | Calibration output for `design-backend`. |
| `docs/designs/2026-11-25-p2p-create-states.md` | Calibration output for `design-frontend`. |
| `plans/agent-workflow/handoff.md` | Rewritten to the current branch, work, blockers, and backlog. |
| `plans/test-runner/implementation.md` | Removed the stale "known expected failure"; `make test` now passes. |

No production code, frontend source, OpenAPI, CI, `specs/`, or `.specify/` changes beyond the
authorized test edit.

## Verification

| Check | Result |
| --- | --- |
| `go test ./internal/server/` | pass |
| `make test` | 12 packages, 240 tests, pass |
| `./scripts/pre-commit.sh` | 0 issues |
| Architecture inventory | 126 non-test files, matches the doc |
| Domain inventory | 26 non-test model files, matches the doc |
| Errors inventory | 77 `errors.New` sites, matches the doc |
| UI inventory | 15 components, 7 pages, one 768px breakpoint, axios only in `api/`, four inline styles |
| ADR count | 6 |
| Skill discovery | both new skills listed by OpenCode |
| Calibration docs | backend 68 lines and frontend 73 lines above the checklist, within the template target |
| `git status` | only the expected paths; nothing committed |

Not verified: Docker-backed integration tests, merged coverage, and the P2P data check. Docker is
unavailable; `plans/agent-workflow/handoff.md` records the exact commands.

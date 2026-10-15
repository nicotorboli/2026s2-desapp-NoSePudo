# Implementation Plan: Interactive API Documentation (Swagger/OpenAPI)

**Branch**: `004-documentacion-swagger` | **Date**: 2026-10-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-documentacion-swagger/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Serve a Swagger UI page from the Go backend that renders the existing, maintained
`backend/api/openapi.yaml` as-is — no generation, no second copy. The UI's static
assets (HTML/CSS/JS) are vendored and embedded into the binary so the page works
with no outbound internet access; the YAML contract itself is read from disk on
every request so editing it and reloading the page shows the change with no
rebuild. "Try it out" talks to the real backend for real, including mutations,
through the exact same request pipeline every other client goes through — the
documentation delivery is a new, separate concern that sits beside the existing
access-controlled route table, not inside it, so the existing contract-sync test
and routing invariants are not touched or weakened.

## Technical Context

**Language/Version**: Go 1.26.5 (per `backend/go.mod`)

**Primary Dependencies**: `net/http` only (Principle I forbids external web
frameworks). No new Go module dependency: Swagger UI's static dist assets
(`swagger-ui-dist`, HTML/CSS/JS) are vendored into the repo and served with
stdlib `embed.FS` + `http.FileServer`, matching the project's already-minimal
dependency list.

**Storage**: N/A — no new persistence. The feature reads an existing file
(`backend/api/openapi.yaml`) from disk per request; no database involved.

**Testing**: `go test` (table-driven, `net/http/httptest`), same conventions
already used in `internal/server`.

**Target Platform**: Wherever the backend already runs — `go run ./cmd/server`
from `backend/` on a dev machine or CI (Linux); Postgres is dockerized, the Go
process is not (pre-existing state, unchanged by this feature).

**Project Type**: Web service (backend only). The frontend (React) is out of
scope per the spec's Assumptions.

**Performance Goals**: None specific — a documentation page for a handful of
concurrent developers, not a hot path.

**Constraints**: Must work with zero outbound internet access (FR-004/SC-004) →
no CDN references anywhere in the served page. Try-it-out requests must land on
whichever host/port is actually serving the docs page, not a value hardcoded in
the contract (edge case: Docker vs. local) → the contract's `servers` entry
becomes relative (`/`) so Swagger UI issues same-origin requests. The existing
route-table/contract-sync test and the "no `Handle` outside `buildMux`" AST
invariant (`internal/server/routes_test.go`) must keep passing unmodified
(FR-008) → the new routes are deliberately not part of `routes()`/`buildMux`.

**Scale/Scope**: Two new HTTP entry points (`/docs/` and `/openapi.yaml`), one
new internal subpackage, ~6 vendored static files, no schema/model changes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Result |
|---|---|---|
| I. Backend en Go con net/http | No external web framework added; Swagger UI is vendored static assets served with stdlib `embed.FS` + `http.FileServer`. | PASS |
| III. Arquitectura en capas | New code lives in a subpackage of the existing `server` layer (`internal/server/docs`), the same way `persistence` already has `dao`/`repository` subpackages. No new top-level layer. | PASS |
| IV. Inyección de dependencias | The contract file path is passed into the new package's constructor (same pattern `contract_test.go` already uses), not read from a global; `cmd/server` is still the only place that wires concrete pieces together. | PASS |
| V. DTOs en el borde | N/A — no domain model crosses this border; the YAML is served byte-for-byte and the UI is static assets. | PASS |
| VII. API REST documentada con OpenAPI | This *is* the principle's UI. The contract itself is edited only to make `servers` relative (see research.md); no second copy of the contract is introduced. | PASS |
| VIII. Seguridad JWT | Documentation page itself is unauthenticated by design (Assumptions), but every documented endpoint still enforces its declared access level when invoked for real through try-it-out — the new routes carry no access level because they are not part of the access-controlled route table at all. | PASS |
| IX. Observabilidad | The combined handler (docs + business router) is wrapped by the existing `CorrelationID`/`RequestLogger` middleware chain before being handed to `http.Server`, so docs requests are logged and correlated like any other request. | PASS |
| XIII. Testing | New package gets its own unit tests (httptest); a small addition at the `server` level verifies composition; see Phase 1 quickstart for the e2e smoke check. | PASS |

No violations to justify — Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/004-documentacion-swagger/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
backend/
├── api/
│   └── openapi.yaml                 # unchanged file, now also rendered by /docs/
├── cmd/server/main.go               # wires docs.Handler(contractPath) into NewServer
└── internal/
    └── server/
        ├── server.go                # composes docs handler + business mux, unchanged test files
        ├── routes.go                # UNCHANGED — docs routes are deliberately not here
        ├── routes_test.go           # UNCHANGED
        ├── contract_test.go         # UNCHANGED
        └── docs/                    # NEW subpackage, own package docs, own tests
            ├── docs.go              # Mount(contractPath, apiRouter) http.Handler
            ├── docs_test.go
            ├── embed.go             # //go:embed dist
            └── dist/                # vendored swagger-ui-dist static assets (committed)
                ├── swagger-initializer.js   # customized: url "/openapi.yaml", persistAuthorization
                ├── index.html
                ├── swagger-ui.css
                ├── swagger-ui-bundle.js
                ├── swagger-ui-standalone-preset.js
                └── favicon-32x32.png

frontend/   # untouched — out of scope per spec Assumptions
```

**Structure Decision**: Single existing project (Go backend), no new top-level
layer. The feature lives entirely in a new `internal/server/docs` subpackage
(mirroring the existing `persistence/dao` + `persistence/repository` pattern of
one layer with purpose-specific subpackages) plus a small composition change in
`server.go` and a one-line wiring change in `cmd/server/main.go`. `routes.go`
and both of its guarding test files are not touched, by design: the docs
routes are not business endpoints and must not enter the
access-level/contract-sync machinery that governs `routes()`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations — this section is intentionally empty.

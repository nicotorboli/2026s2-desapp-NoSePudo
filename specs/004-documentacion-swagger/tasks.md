---

description: "Task list template for feature implementation"
---

# Tasks: Interactive API Documentation (Swagger/OpenAPI)

**Input**: Design documents from `/specs/004-documentacion-swagger/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/docs-endpoints.md](./contracts/docs-endpoints.md), [quickstart.md](./quickstart.md)

**Tests**: Included — `plan.md`'s Technical Context specifies `go test` (table-driven, `net/http/httptest`) as the project's existing convention for this layer, and `routes_test.go`/`contract_test.go` already protect the invariants this feature must not weaken (FR-008).

**Organization**: Tasks are grouped by user story (US1–US4 from `spec.md`) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Single existing project (Go backend only — `frontend/` is untouched, per plan.md). All paths are relative to the repository root unless noted.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the new subpackage's skeleton and pull in the vendored UI assets it will embed.

- [X] T001 Create the `backend/internal/server/docs` package skeleton: empty `docs.go` (package `docs`), empty `embed.go`, empty `docs_test.go`, and an empty `dist/` subdirectory, per the structure in [plan.md](./plan.md#project-structure)
- [X] T002 [P] Vendor the official `swagger-ui-dist` static build into `backend/internal/server/docs/dist/`: commit `index.html`, `swagger-ui.css`, `swagger-ui-bundle.js`, `swagger-ui-standalone-preset.js`, `favicon-32x32.png` and `swagger-initializer.js` as-is from the upstream package (no CDN references anywhere in them — FR-004), per [research.md §1](./research.md#1-how-to-serve-a-usable-offline-capable-swagger-ui-without-a-web-framework)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Wire the new package into the real HTTP pipeline so `/docs/` and `/openapi.yaml` are reachable at all, without yet implementing their full response bodies. No user story is demoable until this phase and the next are both done.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 [P] Customize `backend/internal/server/docs/dist/swagger-initializer.js`: set `url: "/openapi.yaml"` (relative, not `http://localhost:8080`) and `persistAuthorization: true`, per [research.md §3](./research.md#3-making-try-it-out-target-the-right-backend-regardless-of-hostport) (depends on T002)
- [X] T004 [P] Implement `backend/internal/server/docs/embed.go`: `//go:embed dist` into an `embed.FS` package variable (depends on T001, T002)
- [X] T005 Implement `Mount(contractPath string, next http.Handler) http.Handler` in `backend/internal/server/docs/docs.go`: dispatch on request path — exact match `/openapi.yaml` and prefix match `/docs/` go to this package's own (stub) handlers, everything else falls through to `next` unchanged, per [contracts/docs-endpoints.md](./contracts/docs-endpoints.md#composition-with-the-existing-business-router) (depends on T004)
- [X] T006 Compose `docs.Mount` into `backend/internal/server/server.go#NewServer`: add a `contractPath string` parameter, wrap `s.router` with `docs.Mount(contractPath, s.router)` before the existing `middleware.Chain(...)` call — `routes.go` and its test files stay untouched (FR-008) (depends on T005)
- [X] T007 Pass the contract path into `backend/cmd/server/main.go`'s `server.NewServer(...)` call, using the same fixed-relative-path convention `internal/server/contract_test.go` already uses (`filepath.Join("api", "openapi.yaml")`) (depends on T006)

**Checkpoint**: `/docs/` and `/openapi.yaml` are routed to the new package instead of 404ing against the business router. No story is testable yet — handler bodies are still stubs.

---

## Phase 3: User Story 1 - Browse the complete API documentation in a browser (Priority: P1) 🎯 MVP

**Goal**: A developer opens `/docs/` and sees every endpoint from `backend/api/openapi.yaml`, grouped by tag, with parameters/schemas/responses — with zero manual duplication and zero outbound network requests.

**Independent Test**: With no other story's work done, open the documentation page and confirm every endpoint declared in the contract appears, grouped by tag, with its schema (per [quickstart.md § US1](./quickstart.md#us1--browse-the-complete-documentation-p1)).

### Tests for User Story 1

- [X] T008 [P] [US1] Table-driven `httptest` cases in `backend/internal/server/docs/docs_test.go`: `GET /docs/` → `200` with `index.html`'s content type; `GET /docs/swagger-ui.css` → `200`; `GET /docs/does-not-exist` → `404`; `GET /openapi.yaml` against a fixture file → `200`, `Content-Type: application/yaml`, body matches the fixture byte-for-byte; `GET /openapi.yaml` against a missing path → `500`, `Content-Type: text/plain` (write these first, confirm they fail against the stub from T005)

### Implementation for User Story 1

- [X] T009 [US1] Implement the `/docs/` branch of `docs.Mount` in `backend/internal/server/docs/docs.go`: serve the embedded `dist/` tree with `http.FileServer` over `http.FS`, `/docs/` resolving to `index.html`, per [contracts/docs-endpoints.md](./contracts/docs-endpoints.md#get-docs-and-any-path-prefixed-docs) (depends on T004, T006)
- [X] T010 [US1] Implement the `/openapi.yaml` branch of `docs.Mount` in `backend/internal/server/docs/docs.go`: read `contractPath` from disk fresh on every request, respond `200`/`application/yaml` with the raw bytes on success; on a read error, log it via the request logger and respond `500`/`text/plain` with a short message (e.g. `"openapi contract unavailable"`) — never panic (FR-012), per [contracts/docs-endpoints.md](./contracts/docs-endpoints.md#get-openapiyaml) (depends on T006)
- [X] T011 [US1] Run T008's tests to green against T009/T010's real implementation; fix until all pass
- [X] T012 [US1] Manual validation per [quickstart.md § US1](./quickstart.md#us1--browse-the-complete-documentation-p1): run the backend, open `http://localhost:8080/docs/`, confirm all 7 operations are visible grouped under `auth`/`players` tags with each one's declared access level (FR-009), confirm zero requests leave the machine in the Network tab, confirm the UI text is English (FR-010a)

**Checkpoint**: User Story 1 is fully functional and independently testable — the documentation page browses the live contract.

---

## Phase 4: User Story 2 - Try out a request directly from the documentation (Priority: P1)

**Goal**: From the documentation page alone, execute a real request (including mutations) against the running backend and see the actual response.

**Independent Test**: From the documentation page, execute a request to an anonymous endpoint (account registration) and confirm a real HTTP response renders on the page (per [quickstart.md § US2](./quickstart.md#us2--try-it-out-for-real-p1)).

### Implementation for User Story 2

- [X] T013 [US2] Edit `backend/api/openapi.yaml`: change the single `servers` entry from `http://localhost:8080` to the relative URL `/`, per [research.md §3](./research.md#3-making-try-it-out-target-the-right-backend-regardless-of-hostport) — this makes every try-it-out call same-origin with whatever actually served `/docs/`, and (together with T003's `persistAuthorization` and the contract's existing `accessToken`/`refreshToken` bearer schemes) makes token reuse (FR-006) work with no custom code
- [X] T014 [US2] Run `go test ./internal/server/...` from `backend/` and confirm `TestPublishedContractMatchesTheRouteTable`/`TestDeclaredLevelsMatchTheSpecification` still pass after T013's edit (the `servers` field is outside what they check, but verify no regression)
- [X] T015 [US2] Manual validation per [quickstart.md § US2](./quickstart.md#us2--try-it-out-for-real-p1): try-it-out `POST /auth/register` with a fresh body, confirm the real status/body renders; Authorize with a token obtained from `POST /auth/login`, confirm `GET /players` succeeds with no extra header set (token reused automatically); clear the Authorize field and confirm the real `401` is shown, not a client-side message (FR-005, FR-006, FR-007, FR-007a)

**Checkpoint**: User Stories 1 AND 2 both work independently — the page is browsable and interactive.

---

## Phase 5: User Story 3 - Documentation always matches the deployed API (Priority: P2)

**Goal**: The documentation page reflects whatever the contract currently says, with no stale/aspirational drift, and the existing contract-sync check keeps protecting that.

**Independent Test**: Modify an endpoint's declared access level in the contract and confirm the documentation page reflects the change on next load, with no separate manual step (per [quickstart.md § US3](./quickstart.md#us3--documentation-tracks-the-live-contract-p2)).

### Tests for User Story 3

- [X] T016 [P] [US3] Regression test in `backend/internal/server/docs/docs_test.go`: write a temp file, request it through the `/openapi.yaml` handler, overwrite the temp file's contents, request again, and assert the second response reflects the new contents with no restart/cache (FR-003) (depends on T010)

### Implementation for User Story 3

- [X] T017 [US3] Manual validation per [quickstart.md § US3](./quickstart.md#us3--documentation-tracks-the-live-contract-p2): with the backend running, edit one operation's `summary` in `backend/api/openapi.yaml`, reload `/docs/` with no restart, confirm the new summary is visible, then revert; run `go test ./internal/server/...` from `backend/` and confirm `TestPublishedContractMatchesTheRouteTable` and `TestNoRouteIsRegisteredOutsideBuildMux` still pass completely unmodified (FR-008)

**Checkpoint**: All three P1/P2 stories work independently; the contract-sync safety net is confirmed intact.

---

## Phase 6: User Story 4 - Documentation is easy to find (Priority: P3)

**Goal**: A new contributor finds the documentation path without insider knowledge.

**Independent Test**: Standalone — confirm the README names the documentation path, and that path resolves with the backend running (per [quickstart.md § US4](./quickstart.md#us4--easy-to-find-p3)).

### Implementation for User Story 4

- [X] T018 [US4] Add a line under the "La API" section of `README.md` naming `/docs/` as the path where the interactive documentation is served (FR-011)
- [X] T019 [US4] Manual validation per [quickstart.md § US4](./quickstart.md#us4--easy-to-find-p3): re-read `README.md`'s "La API" section, confirm `/docs/` is named; with the backend running, request `http://localhost:8080/docs/` and confirm it resolves `200`

**Checkpoint**: All four user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final repo-wide checks that span all stories.

- [X] T020 [P] Edge-case check per [quickstart.md § Edge cases](./quickstart.md#edge-cases): temporarily rename `backend/api/openapi.yaml`, request `http://localhost:8080/openapi.yaml` directly, confirm a clear `500` and that `/players` etc. still serve normally (FR-012); restore the file afterward
- [X] T021 [P] Run `./scripts/pre-commit.sh` from the repository root (gofmt, `go vet`, golangci-lint) and `go test ./...` from `backend/`, confirming no regressions anywhere in the suite, including `-race` where a C compiler is available

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational completion
- **User Story 2 (Phase 4)**: Depends on Foundational completion; independent of US1's own tasks, but exercised more meaningfully once US1's page is browsable
- **User Story 3 (Phase 5)**: Depends on Foundational completion and on US1's `/openapi.yaml` handler existing (T010) to regression-test
- **User Story 4 (Phase 6)**: Depends on Foundational completion only (README + a single request)
- **Polish (Phase 7)**: Depends on all four user stories being complete

### User Story Dependencies

- **US1 (P1)**: No dependencies on other stories
- **US2 (P1)**: No dependencies on other stories (T013 edits a different part of the same contract file US1 reads, but not a code dependency)
- **US3 (P2)**: Regression-tests US1's handler (T010); otherwise independent
- **US4 (P3)**: No dependencies on other stories

### Within Each User Story

- US1: tests (T008) before implementation (T009, T010), both before running tests green (T011) and manual validation (T012)
- US2: contract edit (T013) before its own regression check (T014) and manual validation (T015)
- US3: regression test (T016) alongside manual validation (T017)
- US4: README edit (T018) before manual validation (T019)

### Parallel Opportunities

- T002 (vendoring) can run in parallel with T001 (skeleton creation) — different files
- T003 and T004 can run in parallel once T001/T002 land — different files (`swagger-initializer.js` vs. `embed.go`)
- T008 (US1 tests) can be written in parallel with T020 (edge-case check is independent); within Phase 3, T008 precedes T009/T010 by design (tests first)
- T016 (US3 test) can be written in parallel with US2's or US4's phase tasks since it touches a different concern, though it depends on T010 already existing
- T020 and T021 (Polish) can run in parallel — independent checks

---

## Parallel Example: Setup + Foundational

```bash
# Launch Setup tasks together:
Task: "Create backend/internal/server/docs package skeleton"
Task: "Vendor swagger-ui-dist static assets into backend/internal/server/docs/dist/"

# Once those land, launch these Foundational tasks together:
Task: "Customize dist/swagger-initializer.js (relative url, persistAuthorization)"
Task: "Implement embed.go (//go:embed dist)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: browse `/docs/`, confirm all 7 operations and their schemas render, confirm no outbound requests
5. This alone is already a usable, demoable increment — a read-only, always-current documentation page

### Incremental Delivery

1. Setup + Foundational → routes reachable, nothing visible yet
2. Add US1 → browsable documentation page (MVP!)
3. Add US2 → the page becomes interactive (try-it-out, token reuse)
4. Add US3 → confirms the page can never drift, and the existing safety net still holds
5. Add US4 → discoverable from a fresh clone
6. Polish → edge cases and full-suite confirmation

### Notes

- This feature is deliberately small: two new routes, one new subpackage, zero new domain models. Foundational work (Phase 2) is genuinely minimal — just enough wiring for Phase 3 onward to have something real to attach to.
- US1 and US2 are both P1 per `spec.md`; they are ordered US1 → US2 here only because a browsable page is a prerequisite for a human to *use* try-it-out manually, not because of a code dependency.
- Commit after each task or logical group, per the "un paso, un commit" convention already in use on this branch.

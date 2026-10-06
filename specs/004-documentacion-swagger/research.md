# Research: Interactive API Documentation (Swagger/OpenAPI)

No items were marked `NEEDS CLARIFICATION` in Technical Context — the two
ambiguities the spec itself flagged were already resolved in its Clarifications
section (try-it-out executes for real including mutations; UI text is English).
What follows are the technical decisions the plan depends on.

## 1. How to serve a usable, offline-capable Swagger UI without a web framework

- **Decision**: Vendor the official `swagger-ui-dist` static build (HTML/CSS/JS)
  into `backend/internal/server/docs/dist/`, committed to the repo, embedded at
  compile time with `//go:embed`, and served with stdlib `http.FileServer` over
  `http.FS`.
- **Rationale**: Principle I bans external web frameworks; a static asset
  bundle is not a framework — it adds no routing, no middleware, no request
  lifecycle of its own, same category as serving a favicon. Vendoring (vs. a
  Go module like `swaggo/http-swagger`) adds **zero** new entries to `go.mod`,
  matching the project's already short, curated dependency list, and gives full
  control over the one file that needs customizing
  (`swagger-initializer.js` — see §3). Embedding via `go:embed` (vs. reading
  the dist files from disk at runtime) means the binary is self-contained and
  nothing about serving the UI shell depends on the process's working
  directory, unlike the contract file itself (see §2).
- **Alternatives considered**:
  - `swaggo/http-swagger` + `swaggo/files` — rejected: it's built around
    generating the spec from Go annotations, which is explicitly out of scope
    (Assumptions: "not a second source of truth"); using it only for its asset
    embedding would add a dependency and its transitive tree for a feature it
    isn't really designed to provide, against the grain of this repo's minimal
    `go.mod`.
  - Loading Swagger UI from a CDN (`unpkg`/`jsdelivr`) — rejected outright:
    directly violates FR-004/SC-004 (must work with no outbound internet).
  - Redoc / Stoplight Elements instead of Swagger UI — rejected: neither ships
    "try it out" as directly and simply as Swagger UI does, which is the whole
    point of User Story 2.

## 2. Where the OpenAPI contract is read from at request time

- **Decision**: `GET /openapi.yaml` reads `backend/api/openapi.yaml` from disk
  on every request (not embedded), using the same fixed-relative-path
  convention `internal/server/contract_test.go` already uses
  (`filepath.Join("api", "openapi.yaml")` from the backend module root). The
  path is passed as a constructor parameter, not read from a package-level
  global, so tests can point it at a fixture.
- **Rationale**: FR-003 is explicit — "reloading the page after the contract
  changes MUST show the updated content" with "no separate generation or build
  step." Reading from disk per request satisfies this literally, with no
  restart needed at all (stricter than SC-003's "no step beyond restarting the
  backend," which this also happens to satisfy). The contract file is already
  the single source of truth (Assumptions) and is already read this way by
  `contract_test.go`, so this isn't a new pattern.
- **Alternatives considered**: Embedding `openapi.yaml` alongside the UI
  assets — rejected: would require a restart (effectively a rebuild, since the
  project's documented start command is `go run ./cmd/server`) to see a
  contract edit, which is allowed by SC-003 but not by the more literal FR-003,
  and reading from disk is no more complex.

## 3. Making try-it-out target the right backend regardless of host/port

- **Decision**: Change the contract's single `servers` entry from
  `http://localhost:8080` to a relative URL (`/`), and set
  `persistAuthorization: true` in the vendored `swagger-initializer.js`.
- **Rationale**: Swagger UI builds try-it-out request URLs from the first
  `servers` entry. An absolute `http://localhost:8080` breaks the Docker-vs-local
  edge case the spec calls out: the docs page could be viewed through a
  different host/port than the one literally written in the YAML (e.g. a
  forwarded container port). A relative server URL makes every try-it-out call
  same-origin with whatever actually served the `/docs/` page, which is always
  correct because the same backend process serves both. This is also exactly
  the mechanism that makes FR-006 (reuse a bearer token across authenticated,
  renewal and superuser calls) work for free: the contract already declares
  two named `securitySchemes` (`accessToken`, `refreshToken`, both HTTP
  bearer); Swagger UI's native "Authorize" dialog captures one value per
  scheme and attaches it to every operation that references that scheme — no
  custom code needed. `persistAuthorization: true` extends that reuse across a
  page reload, which the spec doesn't require but costs nothing and avoids a
  surprising "lost my token" moment during manual exploration.
- **Alternatives considered**: Leaving `servers` absolute and telling users to
  pick the right entry from Swagger UI's server dropdown — rejected: the edge
  case requires this to work automatically, not via a manual step a first-time
  visitor wouldn't know to take.

## 4. Where the new routes live relative to the existing route table

- **Decision**: `/docs/` and `/openapi.yaml` are **not** added to
  `routes()`/`buildMux` in `internal/server/routes.go`. Instead, `server.go`
  composes a single handler: try the docs package's handler first (prefix
  match on `/docs/` and exact match on `/openapi.yaml`), fall back to the
  existing business `s.router` otherwise. Both ship inside
  `internal/server/docs`, a sibling *subpackage* so that
  `TestNoRouteIsRegisteredOutsideBuildMux` (which walks only the files
  directly inside `internal/server/`, skipping subdirectories) never inspects
  it, and `TestDeclaredLevelsMatchTheSpecification` /
  `TestPublishedContractMatchesTheRouteTable` (which only ever look at
  `s.routes()`) never see these two paths at all.
- **Rationale**: FR-008 forbids weakening or bypassing the existing
  route-table/contract-sync check. Folding `/docs/` and `/openapi.yaml` into
  `routes()` would force one of two bad choices: (a) also add them to
  `openapi.yaml`'s `paths`, which is a category error — OpenAPI path
  templating has no concept of "everything under this prefix," so a static
  asset tree (`index.html`, `.css`, `.js`, favicons) can't be expressed as a
  resource with meaningful parameters/schemas worth documenting, or (b) special-case
  the sync test to skip certain patterns, which is a textbook *weakening* of
  it. Keeping the docs delivery mechanism structurally outside the business
  route table sidesteps both: the existing tests are not touched at all, and
  every route that *is* in `routes()` continues to be 100% covered by the
  sync check, with no exceptions carved out.
- **Alternatives considered**: Documenting `/docs/{file}` and `/openapi.yaml`
  as real OpenAPI operations with `x-access-level: anonymous` — rejected per
  above (no way to express a static-asset catch-all cleanly, and it would
  misrepresent "every endpoint in the contract" as including the contract's
  own delivery mechanism). Putting the new package under a brand new top-level
  `internal/docs` instead of `internal/server/docs` — considered equally
  valid; `internal/server/docs` was picked only because the constitution
  already has precedent for a layer growing purpose-specific subpackages
  (`persistence/dao`, `persistence/repository`), avoiding a Constitution Check
  question about introducing an entirely new layer.

## 5. Discoverability (FR-010, FR-011) and resilience (FR-012)

- **Decision**: Stable path `/docs/` (trailing slash, so it matches as a
  subtree); README gets one new line under "La API" pointing at it. If
  `openapi.yaml` is missing, `GET /openapi.yaml` responds `500` with a short,
  clear plaintext body and logs the error — it does not panic, so the backend
  process keeps serving every other route. If the file exists but is not valid
  YAML, the backend doesn't parse it at all (it's served byte-for-byte); the
  *browser-side* Swagger UI is the one that attempts to parse it, and it
  already renders its own clear "Failed to load API definition" banner instead
  of a blank page — this is Swagger UI's existing behavior, not something this
  feature needs to build.
- **Rationale**: Matches FR-012's "clear error rather than crashing the backend
  or serving a broken page" with the least code: the failure mode that's
  actually ours to handle (file not found/unreadable) gets an explicit check;
  the failure mode that's inherently a client-side concern (malformed YAML)
  is already handled by the vendored UI for free.

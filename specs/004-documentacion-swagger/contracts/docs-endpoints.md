# Contract: Documentation delivery endpoints

These two endpoints are the interface this feature itself exposes. They are
**deliberately not entries in `backend/api/openapi.yaml`** — see
[research.md §4](../research.md#4-where-the-new-routes-live-relative-to-the-existing-route-table)
for why: they are not business endpoints, and OpenAPI's path templating has no
clean way to express a static-asset catch-all. They also carry no
`x-access-level`, by design (Assumptions: the documentation page itself is
reachable without authentication) and are not part of `routes()`/`buildMux`,
so `internal/server/routes_test.go` and `internal/server/contract_test.go` are
unaffected by them, and neither test file nor `routes.go` is modified by this
feature.

## `GET /docs/` (and any path prefixed `/docs/`)

- **Purpose**: Serves the vendored, embedded Swagger UI shell and its static
  assets (`swagger-initializer.js`, `swagger-ui.css`,
  `swagger-ui-bundle.js`, `swagger-ui-standalone-preset.js`, favicons).
- **Auth**: None.
- **Request**: No parameters. `GET /docs/` serves `index.html`;
  `GET /docs/<asset>` serves that embedded file.
- **Response**:
  - `200` with the asset's bytes and correct `Content-Type`, for every file
    that exists in the embedded `dist/`.
  - `404` (plain, stdlib `http.FileServer` default) for a path under `/docs/`
    that isn't one of the embedded files.
- **Notes**: Always succeeds for `/docs/` itself regardless of the state of
  `backend/api/openapi.yaml` — the UI shell is embedded in the binary and
  independent of the contract file. Any failure to load the *contract* surfaces
  inside the page (see `/openapi.yaml` below and FR-012), not as a failure of
  this route.

## `GET /openapi.yaml`

- **Purpose**: Returns the current contents of `backend/api/openapi.yaml`,
  unmodified, so the Swagger UI (and any other client — curl, Postman) always
  sees exactly what's on disk.
- **Auth**: None.
- **Request**: No parameters.
- **Response**:
  - `200`, `Content-Type: application/yaml`, body = the file's raw bytes. Read
    fresh from disk on every call — no caching, no embedding, so an edit is
    visible on the very next request with no restart (FR-003).
  - `500`, `Content-Type: text/plain`, a short human-readable message (e.g.
    `"openapi contract unavailable"`), logged via the request logger with the
    underlying `os` error — if the file is missing or unreadable (FR-012).
    Never panics.
- **Not this endpoint's job**: Validating that the file is well-formed YAML or
  a valid OpenAPI document. It is served byte-for-byte; if it's malformed, the
  browser-side Swagger UI that fetches it renders its own "Failed to load API
  definition" error — existing Swagger UI behavior, not custom logic here.

## Composition with the existing business router

`server.go#NewServer` builds the final `http.Handler` as:

```text
combined := docs.Mount(contractPath, businessRouter)   // businessRouter = buildMux(routes(), ...)
handler  := middleware.Chain(combined, CorrelationID, CORS, RequestLogger)
```

`docs.Mount` checks the request path: `/openapi.yaml` exact match or `/docs/`
prefix match go to the docs package's own handler; everything else falls
through to `businessRouter` unchanged. Every request — docs or business — still
passes through `CorrelationID`/`CORS`/`RequestLogger`, so Principle IX
(observability) covers these routes too even though Principle VIII's
access-level machinery deliberately does not apply to them.

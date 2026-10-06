# Data Model: Interactive API Documentation (Swagger/OpenAPI)

No new domain entities, no new persistence, no schema change. This feature
adds a rendering of an already-existing artifact plus a pass-through for real
HTTP requests; nothing it touches is stored.

The spec's Key Entities are conceptual, not persisted — recorded here only to
make explicit that none of them become a model/DTO/table:

| Entity | What it actually is | Where it lives |
|---|---|---|
| OpenAPI contract | The existing file `backend/api/openapi.yaml`. Read from disk, byte-for-byte, on every `GET /openapi.yaml` request. Never parsed, copied, or cached by the backend. | Filesystem, unchanged location. |
| Documentation page | The vendored, embedded Swagger UI static assets (`internal/server/docs/dist/*`), served as-is. Not a template rendered with data — the browser-side JS fetches `/openapi.yaml` itself. | Compiled into the binary via `go:embed`. |
| Try-it-out request | An ordinary HTTP request from the browser to the running backend, through the exact same `routes()`/`buildMux` pipeline every other client uses. Nothing about it is recorded, mocked, or special-cased; the backend cannot distinguish it from a `curl` call. | Not persisted — ephemeral, like any request. |

No validation rules, no state transitions, no relationships to document.

# Feature Specification: Interactive API Documentation (Swagger/OpenAPI)

**Feature Branch**: `004-documentacion-swagger`

**Created**: 2026-10-05

**Status**: Draft

**Input**: User description: "Implementar documentación Swagger/OpenAPI para la API backend del proyecto, de forma que los endpoints existentes queden documentados automáticamente y explorables vía UI interactiva."

## Clarifications

### Session 2026-10-05

- Q: ¿"Try it out" debe poder ejecutar operaciones reales que mutan datos (crear cuentas, disparar sincronizaciones) contra el backend en vivo, o deben quedar excluidas/solo documentadas sin ejecución? → A: Ejecuta todo de verdad, incluidas mutaciones, sin restricción — consistente con el supuesto ya registrado de que la API corre solo en localhost/Docker, no en internet público.
- Q: ¿La documentación interactiva (textos de la interfaz, no el contrato OpenAPI en sí) debe mostrarse en español o en inglés? → A: Todo en inglés (UI de Swagger + contrato), como ya está hoy el `openapi.yaml`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse the complete API documentation in a browser (Priority: P1)

As a developer or technical stakeholder, I open a single page in my browser and see every endpoint of the API organized by resource, with its parameters, request body schema and response schemas, so I can understand what the API offers without reading the raw OpenAPI file or the Go route table.

**Why this priority**: This is the baseline the feature exists to deliver. Without a browsable page, nothing else here has value.

**Independent Test**: With no other story implemented, open the documentation page and confirm every endpoint declared in the project's OpenAPI contract appears, grouped by tag, with its schema.

**Acceptance Scenarios**:

1. **Given** the backend running, **When** a user navigates to the documentation page, **Then** it loads and lists all endpoints grouped by tag (`auth`, `players`) with their summaries.
2. **Given** an endpoint entry on the page, **When** the user expands it, **Then** its parameters, request body schema and possible response codes with their schemas are shown.
3. **Given** the OpenAPI contract declares N endpoints, **When** the documentation page is inspected, **Then** exactly N endpoints are visible — no more, no fewer.

---

### User Story 2 - Try out a request directly from the documentation (Priority: P1)

As a developer validating the API, I fill in parameters on the documentation page and execute a real request against the running backend, and I see the actual response, so I can verify endpoint behavior without a separate HTTP client.

**Why this priority**: "Explorable via an interactive UI" is what distinguishes this feature from simply opening the existing YAML file. Without execution, the page is read-only documentation, not an interactive one.

**Independent Test**: From the documentation page alone, execute a request to an anonymous endpoint (e.g. account registration) and confirm a real HTTP response is rendered on the page.

**Acceptance Scenarios**:

1. **Given** an endpoint declared `anonymous`, **When** the user supplies a valid body and executes it from the documentation page, **Then** the backend processes it for real and the actual response (status and body) is displayed on the page.
2. **Given** an endpoint that requires a credential, **When** the user supplies a bearer token once through the documentation page, **Then** that token is reused on subsequent try-it-out requests to authenticated, renewal or superuser endpoints.
3. **Given** invalid input supplied through the page, **When** the request is executed, **Then** the page shows the exact error response the backend returns, not a generic client-side message invented by the documentation tool.

---

### User Story 3 - Documentation always matches the deployed API (Priority: P2)

As anyone relying on the documentation, I need it to reflect exactly the endpoints and access levels currently exposed by the backend, so the documentation is never a stale or aspirational description.

**Why this priority**: It protects the value delivered by User Stories 1 and 2 over time. Without it, the page could silently drift from the real API and become actively misleading — worse than no documentation at all.

**Independent Test**: Modify an endpoint's declared access level in the OpenAPI contract and confirm the documentation page reflects the change the next time it is loaded, with no separate manual step.

**Acceptance Scenarios**:

1. **Given** the OpenAPI contract file changes, **When** the documentation page is reloaded, **Then** it shows the updated content with no separate build or copy step.
2. **Given** the existing automated check that fails the build when an endpoint's route table disagrees with the OpenAPI contract, **When** this feature is added, **Then** that check still runs and still fails under the same conditions as before.

---

### User Story 4 - Documentation is easy to find (Priority: P3)

As a new contributor or external integrator, I want a clearly discoverable location for the documentation, so I don't need insider knowledge to find it.

**Why this priority**: Lowest priority — a nice usability improvement, but the documentation already delivers its value once published at any known URL.

**Independent Test**: Standalone — confirm the project's README names the path where the documentation is served, and that path resolves successfully with the backend running.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** a contributor reads the README, **Then** they find the path where the interactive documentation is served.
2. **Given** the backend running locally, **When** that documented path is requested, **Then** the documentation page loads successfully.

---

### Edge Cases

- What happens when the OpenAPI contract file is missing or fails to parse at startup? The documentation page must report a clear error rather than crashing the backend or serving a blank page.
- What happens when a try-it-out request targets a protected endpoint with no credential supplied? The real refusal (same as any other client would get) must be shown, not a client-side guess.
- What happens when the machine serving or viewing the documentation has no outbound internet access? The page must still load and function fully.
- What happens when two endpoints share the same path but different HTTP methods? Both must appear as distinct entries.
- What happens when the backend is reached through a different host/port than the one declared in the contract (e.g. inside Docker vs. locally)? Try-it-out requests must still target the backend instance actually serving the documentation page.

## Requirements *(mandatory)*

### Functional Requirements

**Documentation availability**

- **FR-001**: System MUST expose an interactive documentation page that renders the project's existing OpenAPI contract (`backend/api/openapi.yaml`).
- **FR-002**: The documentation page MUST list every endpoint declared in the OpenAPI contract, grouped by tag, together with its summary, parameters, request body schema and response schemas.
- **FR-003**: The documentation page MUST reflect the current contents of the OpenAPI contract file without a separate generation or build step: reloading the page after the contract changes MUST show the updated content.
- **FR-004**: The documentation page MUST remain fully usable on a machine with no outbound internet access.

**Interactive exploration**

- **FR-005**: The documentation page MUST let a user execute a real HTTP request against the running backend for any documented endpoint, supplying path/query parameters, headers and a request body as applicable, and MUST display the backend's actual response status and body.
- **FR-006**: The documentation page MUST let a user supply a bearer credential once and reuse it across subsequent try-it-out requests to authenticated, renewal or superuser endpoints.
- **FR-007**: A try-it-out request MUST go through the exact same request pipeline (middleware, validation, business logic) as a request from any other client; the documentation page MUST NOT special-case or bypass any server behavior.
- **FR-007a**: Try-it-out MUST be able to execute every documented endpoint for real, including state-changing operations (e.g. account creation, triggering a sync). No endpoint is excluded or made read-only within the documentation page.

**Consistency with the existing contract**

- **FR-008**: This feature MUST NOT weaken or bypass the existing automated check that fails the build when an endpoint's route table disagrees with the OpenAPI contract.
- **FR-009**: The documentation page MUST surface each endpoint's declared access level (anonymous, authenticated, renewal, superuser) exactly as declared in the OpenAPI contract.

**Discoverability & resilience**

- **FR-010**: System MUST expose the documentation at a stable, predictable URL path.
- **FR-010a**: The documentation page's interface text MUST be in English, matching the language the OpenAPI contract is already written in; this feature introduces no Spanish-language UI strings.
- **FR-011**: The project's README MUST reference the documentation's path.
- **FR-012**: If the OpenAPI contract file is missing or fails to parse, the documentation endpoint MUST report a clear error rather than crashing the backend or serving a broken page.

### Key Entities

- **OpenAPI contract**: The existing, maintained YAML document (`backend/api/openapi.yaml`) describing every endpoint, its schemas and its declared access level. The single source of truth this feature renders — never a second copy of it.
- **Documentation page**: The interactive page a person visits in a browser. It is a rendering of the contract, not a separate description that could drift from it.
- **Try-it-out request**: A real HTTP request issued from the documentation page against the running backend; indistinguishable by the server from a request issued by any other client.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of endpoints declared in the OpenAPI contract are visible on the documentation page, with zero manually duplicated descriptions.
- **SC-002**: A developer unfamiliar with the codebase can find an endpoint's required fields and execute a successful request against it from the documentation page alone, in under 2 minutes.
- **SC-003**: Changing the OpenAPI contract and reloading the documentation page reflects the change with no additional build or deploy step beyond restarting the backend.
- **SC-004**: The documentation page loads and fully functions on a machine with no outbound internet access.
- **SC-005**: 100% of try-it-out requests produce exactly the response a request from curl or Postman would produce for the same input.

## Assumptions

- **Scope is rendering the existing contract, not generating a new one.** `backend/api/openapi.yaml` is already the maintained source of truth under Principle VII, kept honest by an existing automated check. This feature adds a human-facing, interactive rendering of that file; it does not introduce a second source of truth or an auto-generation pipeline from Go code/comments.
- **The documentation page itself is reachable without authentication.** This matches the project's current deployment assumption (localhost / Docker, not public internet) already recorded for the API itself. Each documented *endpoint* still enforces its own declared access level when invoked through try-it-out — the page being open does not open the API.
- **Try-it-out targets the same backend instance serving the page.** The documentation's configured server URL matches the environment it is viewed from (`http://localhost:8080`, per the contract's existing `servers` entry).
- **UI assets are bundled with the backend rather than loaded from a CDN**, so the page keeps working offline (FR-004 / SC-004). This is a technical decision left to the plan, recorded here because it is directly observable by users.
- **Out of scope**: auto-regenerating the OpenAPI YAML from Go annotations or comments, versioning multiple API revisions in the documentation, and a documentation UI for the frontend (React) application.

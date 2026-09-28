# Feature Specification: JWT Authentication and Authorization

**Feature Branch**: `003-autenticacion-jwt`

**Created**: 2026-09-27

**Status**: Draft

**Input**: User description: "Autenticación y autorización con JWT para la API del backend, según el Principio VIII de la constitución: los usuarios se crean y se autentican, reciben una credencial firmada, y esa credencial es lo que da acceso al resto de los endpoints. Alcance limitado a la primera entrega: creación de usuario, login, catálogo de jugadores protegido, renovación de la sesión con refresh token rotado, y los dos niveles de privilegio (usuario común y superusuario) representados en la cuenta y transportados en la credencial. Toda entrada de datos se valida y las entradas inválidas se rechazan sin ejecutar lógica de negocio. Las operaciones autenticadas deben poder identificar al actor para los logs, y nunca se registran credenciales ni datos personales."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create an account to operate in the market (Priority: P1)

As a prospective market participant, I create an account with an identifier and a password, so that I can then sign in and reach the endpoints the platform exposes.

**Why this priority**: Nothing else in the feature is reachable without an account. It is the first half of the delivery's "creación de usuario y credencial para acceder al resto de los endpoints".

**Independent Test**: Create an account with valid data and confirm it exists as a common user. Attempt to create a second account with the same identifier and confirm it is refused. Both are testable with no other story implemented.

**Acceptance Scenarios**:

1. **Given** valid registration data, **When** an account is created, **Then** it exists as a common user and can be used to sign in.
2. **Given** an identifier already in use, **When** account creation is attempted, **Then** the system refuses it and the existing account is left unchanged.
3. **Given** registration data that fails validation, **When** it is submitted, **Then** the system rejects it and creates nothing.
4. **Given** a registration request that also carries a privilege level, **When** it is submitted, **Then** the request is rejected as carrying an unexpected field, and under no circumstance is an account created with a privilege level taken from the request.
5. **Given** a created account, **When** its stored record is inspected, **Then** the original password cannot be recovered from it.

---

### User Story 2 - Sign in and receive a session credential (Priority: P1)

As an account holder, I submit my credentials once and receive a session credential that proves who I am and what privilege level I hold, so that I do not resend my password on every request.

**Why this priority**: It is the other half of the delivery requirement and the thing every later story consumes. Without it there is no credential to enforce.

**Independent Test**: With a seeded account, submit correct credentials and confirm a session credential is returned carrying the account's identity, its privilege level and an expiry. Submit wrong credentials and confirm nothing is issued.

**Acceptance Scenarios**:

1. **Given** an existing account and its correct password, **When** they are submitted, **Then** the system returns a session credential together with the instant at which it expires.
2. **Given** an existing account, **When** a wrong password is submitted, **Then** the system refuses the attempt and issues no credential.
3. **Given** an identifier that matches no account, **When** credentials are submitted, **Then** the system responds exactly as it does for a wrong password, revealing nothing about whether the account exists.
4. **Given** a sign-in request with a missing, empty or malformed field, **When** it is received, **Then** the system rejects it as an invalid input without looking up or verifying any credential.
5. **Given** a returned session credential, **When** its contents are read by anyone holding it, **Then** it contains no password and no value that would be damaging to disclose.

---

### User Story 3 - The player catalog is unreachable without a valid credential (Priority: P1)

As the platform owner, I need the player catalog — the only other endpoint the first delivery exposes — to require a valid session credential, so that the credential actually governs access to the API rather than merely existing.

**Why this priority**: This is the protection the feature exists to deliver. Issuing credentials that no endpoint demands would satisfy nothing.

**Independent Test**: Call the catalog with no credential, with a malformed one, with an expired one and with one whose contents were altered; all four must be refused. Call it with a valid credential and confirm the catalog is returned.

**Acceptance Scenarios**:

1. **Given** the catalog endpoint, **When** it is requested with no credential, **Then** the system refuses it as unauthenticated and executes no business logic.
2. **Given** the catalog endpoint, **When** it is requested with an expired credential, **Then** the system refuses it as unauthenticated.
3. **Given** the catalog endpoint, **When** it is requested with a credential whose contents or signature were altered, **Then** the system refuses it and trusts no value carried inside it.
4. **Given** the catalog endpoint, **When** it is requested with a credential presented in an unexpected shape (no scheme, empty value, unknown scheme), **Then** the system refuses it rather than attempting to interpret it.
5. **Given** a valid credential, **When** the catalog is requested, **Then** the catalog is returned and the requesting account is available to the operation as the acting party.

---

### User Story 4 - Every endpoint declares whether it requires a credential (Priority: P1)

As a developer adding an endpoint in a later delivery, I need the access level of every endpoint to be declared explicitly and checked automatically, so that forgetting to protect a new endpoint fails the build instead of silently shipping an open door.

**Why this priority**: In this stack nothing is protected implicitly — an endpoint left unwrapped is simply public, and no compiler, linter or ordinary test reports it. Without this story the protection delivered by US3 decays the moment the API grows.

**Independent Test**: Add an endpoint without declaring its access level and confirm the automated check fails. Declare it and confirm the check passes. This is testable in isolation from the credential logic itself.

**Acceptance Scenarios**:

1. **Given** the set of endpoints the system exposes, **When** the automated check runs, **Then** every endpoint has a declared access level of anonymous, authenticated, renewal or superuser.
2. **Given** a new endpoint added without a declared access level, **When** the automated check runs, **Then** the check fails and names the endpoint.
3. **Given** an endpoint declared as authenticated, **When** it is requested without a valid credential, **Then** it is refused — the declaration and the enforcement agree.
4. **Given** the endpoints of this delivery, **When** their declarations are inspected, **Then** account creation and sign-in are anonymous, the player catalog and sign-out are authenticated, and renewal is declared as renewal.

---

### User Story 5 - The account's privilege level travels in the credential (Priority: P2)

As the platform owner, I hold the single superuser account, and I need the distinction between a common user and the superuser to be a property of the account and to be carried by the credential, so that the operations reserved to the superuser in later deliveries can be enforced without changing how credentials are issued.

**Why this priority**: The two operations the constitution reserves to the superuser — triggering the recalculation job by hand and changing the valuation rules — do not exist yet in this delivery. What must exist now is the account data and the credential shape, because changing those later is a migration; the enforcement point is built together with the first endpoint that needs it.

**Independent Test**: Sign in as a common user and as the superuser, and confirm each credential states the corresponding privilege level. Confirm the superuser account cannot be produced by registration.

**Acceptance Scenarios**:

1. **Given** the system, **When** accounts are inspected, **Then** each one holds exactly one of two privilege levels: common user or superuser.
2. **Given** a common user signing in, **When** the credential is issued, **Then** it states the common privilege level.
3. **Given** the superuser signing in, **When** the credential is issued, **Then** it states the superuser privilege level.
4. **Given** the system after provisioning, **When** superuser accounts are counted, **Then** there is exactly one, and it was not created through account registration.
5. **Given** a credential presented to a protected operation, **When** the privilege level inside it is read, **Then** it is read only after the credential has been verified, and an absent or unrecognised level is treated as insufficient rather than as superuser.

---

### User Story 6 - Authenticated operations name their actor in the logs (Priority: P3)

As someone reviewing what happened in the system, I need every authenticated operation to record which account performed it, and I need certainty that no password, credential or personal datum ever appears in a log event.

**Why this priority**: It closes the loop with the observability obligations and it is the groundwork for the audit trail of later deliveries, but the system is already protected without it.

**Independent Test**: Run a full flow — create an account, sign in, request the catalog, fail a sign-in — then inspect the emitted log events: each authenticated operation names its actor, and a search for the password, the credential and the account's personal data returns nothing.

**Acceptance Scenarios**:

1. **Given** an authenticated operation, **When** its log events are emitted, **Then** they carry the acting account's identity next to the correlation identifier already present on every request.
2. **Given** any request at all, **When** its log events are emitted, **Then** no password, session credential or signing secret appears in any of them, error events included.
3. **Given** a failed sign-in or a refused authorization, **When** it is logged, **Then** the event records the reason for the refusal, without the submitted credential and without personal data such as the submitted identifier.
4. **Given** a failed sign-in for an identifier that matches no account, **When** it is logged, **Then** the event is still emitted with its reason and correlation identifier, simply without a subject identity.
5. **Given** the log events of a complete run, **When** they are read by a tool, **Then** each event is machine-parseable, so that the two preceding scenarios can be verified automatically rather than by eye.

---

### Edge Cases

- A credential is presented one moment before, exactly at, and one moment after its expiry instant: the boundary is defined and behaves consistently.
- A credential arrives correctly signed but with an algorithm the system does not accept: refused, rather than verified on the credential's own terms.
- A credential carries no privilege level, or one the system does not recognise: treated as insufficient privilege, never as superuser.
- An account's privilege level changes while one of its credentials is still unexpired: that credential keeps the level it was issued with until it expires.
- The same account signs in twice: both credentials work independently, and neither invalidates the other.
- A request carries both a valid credential and a malformed body: the credential is checked first, and the body is rejected as invalid input only once the requester is known.
- A request carries an unexpected extra field: refused as an invalid input rather than silently ignored.
- The identifier used to sign in differs from the stored one only by letter case or surrounding whitespace: it resolves to the same account, because both are normalised the same way at registration and at sign-in.
- The signing secret is absent or unusable at startup: the system refuses to start instead of serving requests with credentials nobody can verify.
- An authentication failure originates deep in the system and is passed upward through several layers: it still reaches the caller as an authentication refusal and not as an internal error.

### User Story 7 - Staying signed in without resending the password (Priority: P3)

As an account holder using the application, I do not want to be sent back to the sign-in screen while I am working, nor when I come back a day or two later, but I do want a way for the platform to cut a session off if something goes wrong.

**Why this priority**: The access credential is deliberately short-lived, which alone would sign a working user out every few minutes. This story is what makes that short lifetime livable. It ranks last because the delivery is already complete and demonstrable without it: sign-in and the protected catalog work, they are simply less comfortable.

**Independent Test**: Sign in, wait for the access credential to expire, and confirm the application continues working without a new sign-in. Present a renewal credential twice and confirm the second attempt is refused and the account's other renewal credentials are cut off.

**Acceptance Scenarios**:

1. **Given** a successful sign-in, **When** it completes, **Then** the account holder receives both a short-lived access credential and a longer-lived renewal credential.
2. **Given** an expired access credential and a valid renewal credential, **When** renewal is requested, **Then** a new access credential is issued and the account holder never resends the password.
3. **Given** a renewal request, **When** it succeeds, **Then** the renewal credential used is invalidated and a new one is issued in its place.
4. **Given** a renewal credential that was already used, **When** it is presented again, **Then** the system refuses it and invalidates every renewal credential of that account, on the assumption that one of them was stolen.
5. **Given** an expired renewal credential, **When** renewal is requested, **Then** the system refuses it and the account holder signs in again.
6. **Given** a signed-in account holder, **When** they sign out, **Then** their renewal credential is invalidated and cannot be used again.
7. **Given** a renewal credential, **When** it is presented to any endpoint other than renewal, **Then** it is refused — a renewal credential never grants access to a resource.
8. **Given** an access credential, **When** it is presented to the renewal endpoint, **Then** it is refused — the two kinds of credential are not interchangeable.

---

## Requirements *(mandatory)*

### Functional Requirements

**Credential issuance**

- **FR-001**: System MUST let an account holder authenticate by submitting an account identifier and a password, and MUST issue a signed session credential when they match a stored account.
- **FR-002**: The session credential MUST carry the acting account's identity, its privilege level and an expiry instant, and MUST be verifiable by the system without consulting the client or any client-supplied value.
- **FR-003**: System MUST respond to a failed authentication with a single outcome that does not distinguish a wrong password from an account that does not exist.
- **FR-004**: System MUST store passwords in a form from which the original cannot be recovered, and MUST never include a password in any response.
- **FR-005**: System MUST expire session credentials after a bounded lifetime and MUST refuse an expired credential.
- **FR-006**: The sign-in response MUST be a structured object with named fields rather than a bare credential string, so that later deliveries can add fields to it without breaking existing clients.
- **FR-007**: System MUST NOT place any value in the credential that would be damaging to disclose, on the understanding that anyone holding a credential can read its contents.

**Authentication enforcement**

- **FR-008**: System MUST refuse any request to a protected endpoint that presents no credential, an unverifiable credential, an altered credential or an expired credential, and MUST do so before executing business logic and before touching persistence.
- **FR-009**: System MUST treat the contents of a credential as trustworthy only after verifying its integrity, and MUST accept only signing algorithms from an explicit allowlist.
- **FR-010**: System MUST make the acting account's identity available to the operation being served, and every operation MUST take its actor from the credential and never from a client-supplied field.
- **FR-011**: System MUST distinguish a refusal for missing or invalid authentication from a refusal for insufficient privilege, so that a caller can tell "you are not signed in" from "you are signed in but not allowed".
- **FR-012**: An authentication or authorization refusal MUST reach the caller as such even when it originates in an inner layer and is propagated upward through intermediate layers.

**Endpoint access classification**

- **FR-013**: Every endpoint the system exposes MUST have an explicitly declared access level: anonymous, authenticated, renewal, or superuser. The renewal level exists because that endpoint requires a valid renewal credential and must refuse an access credential, so neither anonymous nor authenticated describes it truthfully — and a declaration that does not describe an endpoint truthfully defeats the purpose of declaring it.
- **FR-014**: System MUST include an automated check that fails when an exposed endpoint has no declared access level, so that an unprotected new endpoint cannot ship unnoticed.
- **FR-015**: In this delivery, account creation and sign-in MUST be declared anonymous, the player catalog and sign-out MUST be declared authenticated, and renewal MUST be declared renewal.

**Privilege levels**

- **FR-016**: System MUST distinguish exactly two privilege levels — common user and superuser — and MUST derive an account's level from the account itself, never from client input.
- **FR-017**: The session credential MUST carry the acting account's privilege level.
- **FR-018**: System MUST treat an absent or unrecognised privilege level as insufficient privilege.
- **FR-019**: System MUST provision exactly one superuser account, and that account MUST NOT be obtainable through account creation.

**Account creation**

- **FR-020**: Users MUST be able to create an account that can then be used to sign in.
- **FR-021**: System MUST refuse account creation when the chosen identifier is already in use, leaving the existing account untouched.
- **FR-022**: System MUST create every self-registered account as a common user, and the shape accepted at registration MUST NOT include a privilege level at all.

**Input validation**

- **FR-023**: System MUST validate every input of every operation introduced by this feature, and MUST reject an invalid input without executing business logic and without persisting anything.
- **FR-024**: System MUST reject a request that carries fields it does not expect, rather than ignoring them silently.
- **FR-025**: System MUST report a validation failure in a way that names what was wrong with the input without echoing back any submitted secret.

**Traceability**

- **FR-026**: System MUST include the acting account's identity in the log events of every authenticated operation, alongside the correlation identifier every request already carries.
- **FR-027**: System MUST NOT write passwords, session credentials or signing secrets to log events, error messages or responses.
- **FR-028**: System MUST NOT write personal data to log events; a subject is identified by its account identity, not by the personal data used to sign in.
- **FR-029**: System MUST log every refused authentication and authorization attempt with the reason for the refusal, and MUST still log the attempt when no account could be identified.
- **FR-030**: Log events MUST be machine-parseable, so that the two preceding requirements can be verified automatically over the events of a complete run.

**Configuration**

- **FR-031**: System MUST obtain the credential signing secret, the credential lifetime and the password hashing cost from injected configuration, never from values embedded in the code.
- **FR-032**: System MUST refuse to start when the signing secret is absent or unusable, rather than serving requests whose credentials cannot be verified.

**Session renewal**

- **FR-033**: System MUST issue, on a successful sign-in, both a short-lived access credential and a longer-lived renewal credential, and MUST state the lifetime of each.
- **FR-034**: System MUST let an account holder exchange a valid renewal credential for a new access credential without resending the password.
- **FR-035**: System MUST persist every renewal credential it issues, so that a renewal credential can be invalidated before its expiry. A renewal credential the system cannot invalidate is not acceptable.
- **FR-036**: System MUST invalidate a renewal credential when it is used, and issue a new one in its place.
- **FR-037**: System MUST refuse a renewal credential that was already used or already invalidated, and MUST invalidate every renewal credential of that account when it happens, treating the reuse as evidence of theft.
- **FR-038**: System MUST distinguish the two kinds of credential, and MUST refuse an access credential presented for renewal and a renewal credential presented to any other endpoint.
- **FR-039**: Users MUST be able to sign out, which invalidates their renewal credential. The access credential is not invalidated and expires on its own.
- **FR-040**: System MUST obtain both credential lifetimes from injected configuration.

### Key Entities *(include if data involved)*

- **Account**: A participant in the market. Holds the identifier used to sign in, a non-recoverable form of the password, a privilege level, and whether it is currently allowed to operate. It is the subject that later deliveries will point orders, positions and audit records at.
- **Privilege level**: The two-valued distinction between a common user and the superuser. A property of the account, and the basis on which superuser-only operations will be permitted or refused in later deliveries.
- **Session credential**: The proof of identity a client presents on each request. Carries the acting account's identity, its privilege level and an expiry instant; its contents are readable by whoever holds it and protected against alteration.
- **Actor**: The account identity attributed to an operation, taken from the credential and carried into the log events of that operation.
- **Renewal credential**: The longer-lived credential an account holder exchanges for a new access credential without resending their password. Unlike the access credential it is persisted, so it can be invalidated before expiring; each one is single-use and is replaced when used.
- **Endpoint access level**: The declared requirement attached to each exposed endpoint — anonymous, authenticated, renewal or superuser — against which the automated classification check runs. It is what the system uses to apply the control, so it cannot describe an endpoint inaccurately without the control itself being wrong.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of requests to an authenticated endpoint without a valid credential are refused, and the system's state is identical before and after every such attempt.
- **SC-002**: Adding an endpoint without declaring its access level fails the automated check, demonstrated by deliberately adding one.
- **SC-003**: 100% of invalid inputs to account creation and sign-in are rejected without creating, modifying or deleting any data.
- **SC-004**: A scan of the log events of a complete run finds zero occurrences of any password, session credential, signing secret or personal datum.
- **SC-005**: 100% of authenticated operations in that run have their acting account identifiable from the log events, and 100% of refused attempts have their reason recorded.
- **SC-006**: A new participant can go from having no account to a successful authenticated request against the catalog in under two minutes, unassisted.
- **SC-007**: Signing in answers fast enough to feel immediate, with 95% of attempts answered in under one second under the expected demonstration load.
- **SC-008**: Every refusal is distinguishable as either "not authenticated" or "not privileged", with no case where the two are indistinguishable to the caller.
- **SC-009**: Started without a usable signing secret, the system refuses to start and reports why, in 100% of attempts.
- **SC-010**: An authentication refusal raised in an inner layer reaches the caller as an authentication refusal in 100% of cases, never as an internal error.
- **SC-011**: An account holder who keeps using the application is never asked to sign in again while their renewal credential is valid, and is asked exactly once when it expires.
- **SC-012**: Reusing an already-used renewal credential is refused in 100% of attempts, and leaves zero usable renewal credentials for that account.
- **SC-013**: After signing out, 100% of renewal attempts with the invalidated credential are refused.

## Assumptions

- **Scope is the first delivery only.** The delivery requires user creation, a credential that gives access to the rest of the endpoints, and the player catalog. The market itself does not exist yet, so orders, portfolio, transaction history and the two superuser-only operations are not part of this specification. The privilege level is modelled and carried now because changing the account record and the credential shape later is a migration; the point that refuses a common user is built together with the first endpoint that needs it.
- **JWT is a constitutional constraint, not a decision made here.** Principle VIII mandates JWT for authentication and authorization. This specification therefore describes the behaviour of a signed, self-contained session credential and leaves the token format, the algorithm, the claim names, the lifetime and the transport header to the plan.
- **The "ApiKEY" of the delivery checklist and the JWT are the same requirement.** The checklist names the technology in one section and the visible behaviour in another. One mechanism is implemented, not two.
- **The account identifier is an email address, and it is normalised.** Surrounding whitespace is trimmed and the address is lowercased before it is stored and before it is looked up, so one address cannot become two accounts. The local part of an address is technically case-sensitive, but no provider treats it that way and normalising is what users expect. Being personal data, the email never appears in a log event: a subject is identified there by its account identity (FR-029). Federated sign-in and multi-factor authentication are out of scope.
- **The catalog is protected in this delivery, and that is reversible.** It is the only endpoint besides account creation and sign-in, so leaving it anonymous would mean the credential governs nothing and the delivery has nothing to demonstrate. Because FR-013 makes each endpoint's access level an explicit declaration, opening the catalog to anonymous requests in a later delivery is a recorded decision rather than a regression.
- **There is exactly one superuser**, per the Vision Document, where it is the initial owner of every player's tokens. It is assumed to be provisioned at bootstrap from injected configuration.
- **The scope is the backend API boundary.** The sign-in screen, credential storage in the browser and session handling in the user interface belong to a frontend feature that consumes what this one exposes.
- **An access credential is trusted until it expires; a renewal credential is revocable.** Verifying an access credential checks its signature and expiry and nothing else: no per-request lookup confirms that the account still exists or that its privilege level has not changed. That is the point of a self-contained credential, and what limits the window is its short lifetime. The renewal credential is the opposite by design — it is persisted precisely so that it can be cut off before expiring, which is what makes sign-out and theft response possible at all.
- **Lifetimes**: the access credential is measured in minutes (15 assumed) and the renewal credential in days (7 assumed). Both come from configuration. The short access lifetime is what makes it acceptable not to check it against persistence on every request; the long renewal lifetime is what lets an account holder come back after a day or two without signing in again.
- **Transport security is assumed, not built here.** A credential in an `Authorization` header is readable by anything that can see the traffic, so it depends on the connection being encrypted. This delivery runs on localhost and in Docker, where that is not in place; serving the API over TLS belongs to deployment, not to this feature. The risk is recorded rather than solved.
- **No rate limiting, and the cost is accepted.** Hashing a password deliberately takes hundreds of milliseconds, so unlimited sign-in attempts are a way to burn CPU. This delivery accepts that: attempts are refused and logged individually, with no lockout and no throttling. Refused attempts are logged with their reason (FR-030), which is what will show whether it ever becomes a real problem.
- **Out of scope**: password reset and recovery, identity verification, account deletion, changing an account's privilege level through the API, lockout after repeated failed sign-ins, and request rate limiting.
- **Two existing gaps in the codebase are prerequisites, not part of this feature's behaviour.** First, the HTTP error wrapper currently recognises a domain error's intended status code only when the error reaches it undecorated; refusals raised in an inner layer and propagated upward therefore surface as internal errors, which FR-012 and SC-010 forbid. Second, log events are currently emitted as human-readable console output rather than in a machine-parseable form, which makes FR-030, SC-004 and SC-005 unverifiable. Both must be addressed for this feature's criteria to hold.

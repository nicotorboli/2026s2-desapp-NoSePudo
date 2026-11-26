# Backend decision template

Write the decision to `docs/designs/<YYYY-MM-DD>-<topic>.md` using the sections below, in this
order. Do not add or rename sections.

## Size and citation rules

- **One page plus checklist.** The document above the `File checklist` section must fit one page
  (about 60 rendered lines at the repository's default width). The checklist is the only part
  allowed to overflow.
- **Cite `path:line`.** Every claim about current behavior, every invariant, and every error
  mapping names a file and verifies the line before writing. If a fact cannot be verified, say so
  explicitly instead of inventing a citation.
- English prose. Product strings and existing Spanish identifiers are quoted as they are.

```markdown
# <YYYY-MM-DD> — <topic>

- Status: proposed
- Area: auth | player | sync | scoring | market | p2p
- Request: one sentence.

## Invariants

What must still hold after the change. Cite the model or DB rule that owns each one.
- <invariant> — `path:line`

## Data and constraints

New or changed columns, indexes, checks, and FKs in `backend/db/init.sql`. State the type and the
unit. If the schema is untouched, write `None.` and say why.

## Transactions and lock order

Which repository method owns the transaction, which rows are locked, and in what order. Name the
rollback path. If the change is read-only, write `None.`

## HTTP contract delta

| Method | Path | Access level | Request DTO | Response DTO | Units |
| --- | --- | --- | --- | --- | --- |
| <METHOD> | </path> | anonymous \| authenticated \| renewal \| superuser | <type> | <type> | <unit> |

Name the `api/openapi.yaml` operation and the DTO packages (`internal/dto` or
`internal/controller/dto`). Money is a decimal string; quantities are integers. If the contract is
unchanged, write `None.` with one line of justification.

## Authorization matrix

| Actor | Allowed? | Enforced at |
| --- | --- | --- |
| anonymous | <yes/no> | route `<path>:<line>` |
| authenticated owner | <yes/no> | service `<path>:<line>` |
| authenticated non-owner | <yes/no> | service `<path>:<line>` |
| superuser | <yes/no> | middleware `<path>:<line>` |

Route level and domain level are separate. Say which one covers each row.

## Error mapping

| Condition | Sentinel | HTTP status | Mapping site |
| --- | --- | --- | --- |
| <condition> | `<package.ErrName>` | <status> | `<path>:<line>` |

State the default for unmatched errors. If a sentinel is new, say which layer owns it and which
layer translates it. See `docs/errors.md`.

## Concurrency and idempotency

What two simultaneous requests can do, which constraint or lock decides the winner, and whether a
retry is safe. If neither applies, write `None.`

## Migration and rollback

Order of operations for an existing database (schema first, backfill, deploy), and how to roll
back without losing data. `db/init.sql` only runs on an empty volume; say what an existing
environment needs. If no migration is required, write `None.`

## Observability

New log lines and their fields, the correlation ID path, and what must never be logged
(credentials, tokens, PII). If nothing is added, write `None.`

## Rejected options

Two or three alternatives with the repo-specific reason each was rejected: layers touched, cost,
risk, client breakage. This section is cut first if the page is too long.

## File checklist

One line per file, create or modify, with the reason:

- [ ] `backend/internal/...` — <reason>
- [ ] `backend/api/openapi.yaml` — <operation and access level>
- [ ] `backend/internal/server/routes.go` — <route and access level>
- [ ] `docs/errors.md` — <only if a sentinel or mapping changed>

## Out of scope

What this decision deliberately does not change. One line each.
```

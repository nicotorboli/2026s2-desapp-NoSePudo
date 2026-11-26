# 0001 — Layered flow: controller → service → repository → dao → adapters

- Status: accepted
- Date: 2026-11-25

## Context

The core is stdlib-only HTTP and SQL (`AGENTS.md`, Architecture). Request parsing, business rules,
transaction boundaries, and SQL must stay at separate seams.

## Decision

Routes call controllers, controllers call one service, services call repositories, repositories
own transactions, DAOs run one statement each
(`backend/internal/server/routes.go:19-181`, `backend/internal/controller/p2p.go:51-57`,
`backend/internal/service/p2p.go:84`, `backend/internal/persistence/repository/p2p.go:59-118`,
`backend/internal/persistence/dao/p2p_session_sql.go:47-55`). Adapters are wired at the root
(`backend/internal/adapters/container.go:30-36`).

## Consequences

A new endpoint usually touches route, controller, service, repository, DAO, and OpenAPI.
Transactions live only in repositories; some services still take concrete repositories
(`backend/internal/service/market.go:37-40`).

## Alternatives rejected

Handlers calling DAOs; one store package; an ORM.

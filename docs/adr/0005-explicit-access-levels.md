# 0005 — Explicit access levels and JWT middleware

- Status: accepted
- Date: 2026-11-25

## Context

An endpoint that forgets its authorization must fail closed, not become public. The contract must
state each operation's requirement.

## Decision

`AccessLevel` has zero value `AccessUndeclared` (`backend/internal/server/access.go:7-15`); every
route declares one (`backend/internal/server/routes.go:19-181`) and `chainFor` panics otherwise
(`backend/internal/server/routes.go:194-213`). The JWT middleware verifies the token, checks its
kind, and injects the `Actor` (`backend/internal/middleware/authentication.go:50-83`,
`backend/internal/middleware/context.go:12-21`); superuser adds a privilege check
(`backend/internal/middleware/authorization.go:25-56`). Ownership is checked in services
(`backend/internal/service/portfolio.go:33-39`, `backend/internal/service/p2p.go:114-122`).

## Consequences

A missing level fails at startup and in the contract test
(`backend/internal/server/contract_test.go:58`). Handlers can assume an actor only on protected
routes; `actorOf` still returns `401` when it is absent
(`backend/internal/controller/auth.go:108-119`).

## Alternatives rejected

Public-by-default routes; per-handler auth checks; role checks in controllers.

# 0006 — DTO package split: `internal/dto` and `internal/controller/dto`

- Status: accepted
- Date: 2026-11-25

## Context

Two DTO packages exist: `backend/internal/dto` (13 production files) and
`backend/internal/controller/dto` (3 files). The split is historical, not a design target.

## Decision

Document reality. `internal/dto` owns player, scoring, sync, ranking, and pagination shapes plus
`CreditsADecimal` (`backend/internal/dto/credits.go:9`); player, sync, and score services build
them (`backend/internal/service/player.go:75`, `backend/internal/service/sync_run.go:93`,
`backend/internal/service/score.go:114`). `internal/controller/dto` owns auth, market, and P2P
shapes plus `CreditsDesdeDecimal` (`backend/internal/controller/dto/market.go:42`); those
controllers build them at the edge (`backend/internal/controller/auth.go:44`,
`backend/internal/controller/market.go:75`, `backend/internal/controller/p2p.go:64`).
`CreditsADecimal` exists in both packages (`backend/internal/dto/credits.go:9`,
`backend/internal/controller/dto/market.go:24`).

## Consequences

Dependencies point opposite ways: `internal/dto` imports `net/http`
(`backend/internal/dto/player_filter.go:22`), while `controller/dto` imports `service`
(`backend/internal/controller/dto/market.go:12`). A formatter change may need two edits, and
consumers must know which package owns a shape.

## Alternatives rejected

Consolidating in this slice (behavior-neutral refactor, out of scope); all DTOs in `model`
(violates ADR-0003); all DTOs in `controller/dto` (forces `net/http` into service-built shapes).

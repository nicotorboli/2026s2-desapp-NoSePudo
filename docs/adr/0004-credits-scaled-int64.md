# 0004 — Credits as scaled int64, decimal string on the wire

- Status: accepted
- Date: 2026-11-25

## Context

Balances, prices, and totals must be exact; floating point would let rounding errors move money.
Go, Postgres, and JSON each need one unambiguous representation.

## Decision

`model.Credits` is an `int64` scaled by 10,000 (`backend/internal/model/credits.go:3-5`).
Arithmetic goes through `Times`, `Plus`, `Minus`, and `DividedBy`, which truncates and guards
zero (`backend/internal/model/credits.go:23-28`). Totals are `Price.Times(Quantity)`
(`backend/internal/model/p2p_session.go:49-51`). Storage is `BIGINT`
(`backend/db/init.sql:227`, `backend/db/init.sql:258-260`). The wire format is a decimal string
with four fraction digits (`backend/internal/dto/credits.go:9-25`); input parsing rejects more
than four decimals or overflow (`backend/internal/controller/dto/market.go:42-76`).

## Consequences

Every boundary converts and the unit is part of the contract: amounts are strings, quantities are
integers. Four decimals is the maximum; finer input is rejected, never rounded. Services take
`model.Credits` (`backend/internal/service/market.go:22-30`).

## Alternatives rejected

`float64`; a decimal dependency; raw integers on the wire (the historical 10,000x undercharge).

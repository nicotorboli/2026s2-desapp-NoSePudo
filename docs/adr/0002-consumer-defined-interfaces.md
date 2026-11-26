# 0002 — Consumer-defined interfaces

- Status: accepted
- Date: 2026-11-25

## Context

Constructor injection is mandatory and there is no global state (`AGENTS.md`, Architecture).
Tests replace external services and, for controllers, services themselves.

## Decision

The consumer declares the interface. Controllers declare service interfaces
(`backend/internal/controller/p2p.go:15-20`, `backend/internal/controller/market.go:16-26`);
services declare repository and adapter interfaces (`backend/internal/service/p2p.go:20-30`,
`backend/internal/service/score.go:26-56`, `backend/internal/service/auth.go:16-45`).
Constructors accept those interfaces (`backend/internal/service/p2p.go:46-55`).

## Consequences

A stub can replace a provider without importing it. Some interfaces are declared more than once,
and market and portfolio still take concrete repositories
(`backend/internal/service/market.go:37-40`, `backend/internal/service/portfolio.go:25-31`).

## Alternatives rejected

Provider-owned interfaces; one broad interface per domain; concrete types everywhere.

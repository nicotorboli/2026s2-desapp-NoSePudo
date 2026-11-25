# 0003 — DTO boundary and no JSON tags on models

- Status: accepted
- Date: 2026-11-25

## Context

Models are shared by services and pure tests; JSON is a transport concern. One decode path is
needed so unknown fields, duplicate values, and validation cannot be skipped by a handler.

## Decision

Models carry no JSON tags (`grep -rn 'json:"' backend/internal/model/` is empty). DTOs carry the
tags (`backend/internal/controller/dto/p2p.go:18-22`) and convert through `DesdeModelo` /
`AModelo` (`backend/internal/dto/player_detail.go:26`,
`backend/internal/controller/dto/p2p.go:75`). All JSON goes through `httphandler`
(`backend/internal/httphandler/utils.go:24-31`,
`backend/internal/httphandler/utils.go:40-67`).

## Consequences

Every field needs an explicit mapping, and nullable model fields become pointers on the wire. The
contract test ties DTO, route, and access-level changes together
(`backend/internal/server/contract_test.go:58`). Two DTO packages exist; see ADR-0006.

## Alternatives rejected

Tags on models; marshalling models directly in handlers; per-controller encoders.

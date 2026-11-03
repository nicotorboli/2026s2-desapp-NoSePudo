# Implementation Plan: Transacción P2P de Tokens

**Branch**: `ft/p2p-token-session` | **Date**: 2026-11-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-transaccion-p2p-tokens/spec.md`

---

## Summary

Permitir que un usuario (vendedor) inicie una oferta de venta directa de tokens de un jugador,
fijando cantidad y precio, y comparta un enlace único (`?token=<uuid>`) con otro usuario (comprador).
El comprador abre el enlace y ve los emails de ambos participantes (vendedor y comprador), la información del jugador, cantidad, precio y un countdown de 5 minutos.
Al aceptar:
- Si la oferta ya fue tomada por otro usuario: error 409 con mensaje claro ("la oferta ya fue tomada por otro usuario").
- Si la oferta expiró: error 410 ("la oferta venció y ya no puede aceptarse"), y la UI muestra un panel estético de "Oferta Vencida".
- Si es válida y tiene fondos suficientes: la transacción se liquida atómicamente: tokens al comprador, créditos al vendedor y se registra en el ledger.
Las sesiones no aceptadas expiran automáticamente a través de un job en segundo plano y liberan los tokens en escrow del vendedor.

Técnicamente: nueva tabla `p2p_sessions`, columna `p2p_session_id` en `market_trades`, nueva capa DAO/Repository/Service/Controller,
reutilización del mecanismo de escrow de `user_positions` y del patrón de settlement de `market_trades`. Los usuarios se identifican por su email ya existente en la base de datos (sin campo `name`).

---

## Technical Context

**Language/Version**: Go 1.24 (declarado en `backend/go.mod`)

**Primary Dependencies**: `net/http` (stdlib), `database/sql` (stdlib), `github.com/lib/pq`, `github.com/golang-jwt/jwt/v5`, `crypto/rand` (stdlib UUID)

**Storage**: PostgreSQL 16 — nueva tabla `p2p_sessions`, columna `p2p_session_id` nullable en `market_trades`

**Testing**: `testing` stdlib + `testcontainers-go` (integración Postgres). Mocks sólo para clock (`time.Now` inyectado).

**Target Platform**: Linux server (Docker). Frontend: React + TypeScript + Vite en `frontend/`.

**Project Type**: Web service (REST API) + SPA

**Performance Goals**: Creación de sesión < 500 ms p95. Expiración dentro de 30 s del vencimiento.

**Constraints**: Stdlib only (`net/http`, `database/sql`). Sin frameworks externos. Sin WebSockets.

**Scale/Scope**: Misma escala que el mercado abierto existente.

---

## Constitution Check

| Principio | Estado | Observación |
|-----------|--------|-------------|
| I – Backend Go net/http | ✅ | Nuevos endpoints via `net/http` |
| II – PostgreSQL database/sql | ✅ | Nueva tabla, sin ORM |
| III – Arquitectura en capas | ✅ | DAO → Repository → Service → Controller |
| IV – Inyección de dependencias | ✅ | `NewP2PRepository`, `NewP2PService`, `NewRestP2PController` |
| V – DTOs en el borde | ✅ | `dto/p2p.go` con `DesdeModelo`/`AModelo` |
| VI – Frontend React + TypeScript | ✅ | `src/api/p2p.ts`, BEM CSS |
| VII – OpenAPI documentado | ✅ | Paths P2P en `api/openapi.yaml` + contract test |
| VIII – JWT auth explícita | ✅ | Todos los endpoints declaran `AccessAuthenticated` |
| IX – Observabilidad | ✅ | Logs con `correlation_id`, `actor` en cada operación |
| X – Auditoría append-only | ✅ | `player_audit_logs` para cada cambio de estado de sesión |
| XI – Transacciones | ✅ | Creación, aceptación y expiración en `db.BeginTx` |
| XII – Scheduler | ✅ | `P2PExpirationJob` con ticker 30 s, controlado por `runCtx` |
| XIII – Testing | ✅ | Tests de integración con testcontainers; mocks sólo para clock |
| XIV – Strings congelados | ✅ | Sin modificar strings existentes |

---

## Project Structure

### Documentation (this feature)

```text
specs/006-transaccion-p2p-tokens/
├── plan.md              ← este archivo
├── research.md          ← decisiones de investigación
├── data-model.md        ← entidades, SQL, capas
├── quickstart.md        ← guía de validación manual
├── contracts/
│   └── p2p-api.yaml     ← contrato OpenAPI de los 4 endpoints
└── tasks.md             ← generado por /speckit-tasks (pendiente)
```

### Source Code (repository root)

```text
backend/
├── db/
│   └── init.sql                              ← +p2p_sessions, +market_trades.p2p_session_id
├── api/
│   └── openapi.yaml                          ← +paths P2P, +schemas P2P
├── internal/
│   ├── model/
│   │   ├── p2p_session.go                    ← nuevo: P2PSession, estados, métodos, errores
│   │   └── trade.go                          ← modificar: añadir P2PSessionID *int64
│   ├── persistence/
│   │   └── dao/
│   │       ├── p2p_session_sql.go            ← nuevo: P2PSessionSql
│   │       ├── p2p_session_sql_test.go       ← nuevo
│   │       ├── container.go                  ← +P2PSession *P2PSessionSql
│   │       ├── trade_sql.go                  ← modificar: soporte P2PSessionID
│   │       └── order_sql.go                  ← sin cambios
│   ├── persistence/
│   │   └── repository/
│   │       ├── p2p.go                        ← nuevo: P2PRepository
│   │       ├── p2p_test.go                   ← nuevo
│   │       └── container.go                  ← +P2P *P2PRepository
│   ├── service/
│   │   ├── p2p.go                            ← nuevo: P2PService
│   │   ├── p2p_test.go                       ← nuevo
│   │   ├── p2p_expiration.go                 ← nuevo: P2PExpirationJob
│   │   ├── p2p_expiration_test.go            ← nuevo
│   │   └── container.go                      ← +P2P *P2PService, +P2PExpiration *P2PExpirationJob
│   ├── controller/
│   │   ├── p2p.go                            ← nuevo: RestP2PController
│   │   ├── p2p_test.go                       ← nuevo
│   │   ├── dto/
│   │   │   └── p2p.go                        ← nuevo: request/response DTOs + conversiones
│   │   └── container.go                      ← +P2P *RestP2PController
│   └── server/
│       ├── routes.go                         ← +4 rutas P2P
│       └── contract_test.go                  ← pasa automáticamente al actualizar openapi.yaml

frontend/
├── src/
│   ├── api/
│   │   └── p2p.ts                            ← nuevo: 4 funciones tipadas
│   ├── pages/
│   │   ├── P2PCreatePage.tsx                 ← nuevo
│   │   ├── P2PCreatePage.css                 ← nuevo (BEM)
│   │   ├── P2PSessionPage.tsx                ← nuevo (con countdown)
│   │   └── P2PSessionPage.css                ← nuevo (BEM)
│   └── App.tsx                               ← +rutas /p2p/create y /p2p/sessions/:token
```

**Structure Decision**: Web application (backend Go + frontend React). Se sigue la estructura existente del proyecto sin añadir nuevas capas ni proyectos.

---

## Complexity Tracking

> No hay violaciones a la Constitución. Sección omitida.

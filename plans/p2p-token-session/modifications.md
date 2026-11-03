# Resumen de Modificaciones: Transacción P2P de Tokens

**Fecha**: 2026-11-03
**Rama**: `ft/p2p-token-session`
**Feature**: Transacción P2P de Tokens (Ofertas directas personalizadas con link e invitación)

---

## Modificaciones Backend

### 1. Base de Datos & Esquema (`backend/db/init.sql`)
- Creación de tabla `p2p_sessions` con claves foráneas hacia `users(id)` y `players(id)`, constraints de estado y valores positivos, e índices para búsquedas por vendedor, token y sesiones pendientes por vencer.
- Inclusión de columna nullable `p2p_session_id BIGINT REFERENCES p2p_sessions(id)` e índice `idx_trades_p2p` en `market_trades`.
- Ajuste de `buy_order_id` y `sell_order_id` en `market_trades` como opcionales/nullables para soportar transacciones directas P2P (que no originan órdenes de mercado convencionales).
- Registro de `p2p_sessions` en `expectedTables` de `backend/cmd/server/main.go`.

### 2. Modelo de Dominio (`backend/internal/model/`)
- `p2p_session.go`: Creación de struct `P2PSession` (ordenado por `fieldalignment`), estados `PENDING`, `COMPLETED`, `CANCELLED`, `EXPIRED`, métodos de transición `Accept`, `Cancel`, `Expire`, y errores sentinel (`ErrP2PSessionNotFound`, `ErrP2PSessionNotLive`, `ErrP2PSessionExpired`, `ErrP2PSessionAlreadyTaken`, `ErrP2PSessionSelfTrade`).
- `trade.go`: Incorporación del campo `P2PSessionID *int64` al struct `Trade`.

### 3. Capa DAO (`backend/internal/persistence/dao/`)
- `p2p_session_sql.go`: DAO `P2PSessionSql` con métodos `Insert`, `GetByToken` (con JOIN a users para emails), `LockByToken` (`SELECT ... FOR UPDATE`), `UpdateStatus` y `ListPendingExpired`.
- `trade_sql.go`: Adaptación de `Insert` y `ListByParticipant` para persistir y leer `p2p_session_id`, así como manejo de `buy_order_id` y `sell_order_id` nullables.
- `container.go`: Registro e instanciación de `P2PSessionSql`.

### 4. Capa de Repositorio (`backend/internal/persistence/repository/`)
- `p2p.go`: Implementación de `P2PRepository` con transacciones atómicas `db.BeginTx`:
  - `Create`: Reserva tokens en `user_positions` y registra auditoría.
  - `Accept`: Bloqueo pesimista, transferencia atómica de tokens y saldo, generación de trade vinculado y asientos de doble partida en ledger.
  - `Cancel`: Cancelación por el vendedor y liberación inmediata de tokens reservados.
  - `ExpirePending`: Proceso en lote con `SKIP LOCKED` que expira sesiones vencidas y libera tokens.
- `container.go`: Inyección de dependencias de `P2PRepository`.

### 5. Capa de Servicio (`backend/internal/service/`)
- `p2p.go`: `P2PService` con validaciones de negocio, cálculo de 5 minutos de expiración, validación de estado y autorización de cancelación.
- `p2p_expiration.go`: `P2PExpirationJob` que ejecuta un ticker cada 30 segundos gobernado por el contexto de ejecución.
- `container.go`: Registro e instanciación de `P2PService` y `P2PExpirationJob`.
- `backend/cmd/server/main.go`: Arranque y detención coordinada de `P2PExpirationJob`.

### 6. Capa de Controlador, DTO y Rutas (`backend/internal/controller/`, `backend/internal/server/`)
- `dto/p2p.go`: `CreateP2PSessionRequest`, `P2PSessionResponse`, `P2PTradeResponse`, y métodos de conversión `DesdeModelo`.
- `controller/p2p.go`: `RestP2PController` exponiendo `Create`, `Get`, `Accept` y `Cancel` con mapeo de errores HTTP (400, 403, 404, 409, 410, 422).
- `controller/container.go`: Registro de `RestP2PController`.
- `server/routes.go`: Enrutamiento de `POST /p2p/sessions`, `GET /p2p/sessions/{token}`, `POST /p2p/sessions/{token}/accept` y `POST /p2p/sessions/{token}/cancel` con nivel `AccessAuthenticated`.
- `backend/api/openapi.yaml`: Actualización del contrato OpenAPI con las 4 rutas y esquemas, validado mediante `contract_test.go`.

---

## Modificaciones Frontend

### 1. API Client (`frontend/src/api/p2p.ts`)
- Módulo tipado con funciones `createP2PSession`, `getP2PSession`, `acceptP2PSession` y `cancelP2PSession`.

### 2. Páginas y Componentes (`frontend/src/pages/`)
- `P2PCreatePage/`: Formulario para elegir jugador del portfolio, cantidad y precio, con cálculo de total en vivo y generación del link copiable `/?token=<uuid>`.
- `P2PSessionPage/`: Vista completa de la sesión que muestra los emails de vendedor y comprador, detalles del jugador, countdown visual en tiempo real de 5 minutos, y paneles estilizados específicos para estados terminales:
  - *"¡Transacción Completada!"* al aceptar.
  - *"Oferta Vencida"* cuando se agotan los 5 minutos o la API responde 410.
  - *"Oferta Ya Aceptada"* cuando otro usuario ya la tomó (409).
  - *"Oferta Cancelada"* si el vendedor la canceló.
- `Navbar/Navbar.tsx`: Enlace a "Venta P2P" para usuarios autenticados.
- `App.tsx`: Detección automática del query param `?token=` en la URL para abrir directamente la sesión compartida.

# Research: Transacción P2P de Tokens

**Feature**: `006-transaccion-p2p-tokens`
**Fecha**: 2026-11-03

---

## 1. Mecanismo de expiración automática

**Decision**: Scheduler background con ticker de 30 segundos en el `SyncScheduler` existente, inyección de `time.Now` como dependencia para tests deterministas.

**Rationale**: El proyecto ya tiene `SyncScheduler` con patrón de goroutine controlada por contexto (`runCtx`). La expiración a los 5 minutos no requiere precisión de milisegundos; un ticker de 30 s es suficiente y respeta el criterio SC-003. No se introduce ninguna dependencia externa.

**Alternatives considered**:
- Cron externo (pg_cron): viola la restricción de stdlib/PostgreSQL-only; requiere extensión DB.
- TTL en Redis: no existe Redis en la arquitectura.
- Expiración lazy (sólo al intentar aceptar): no libera los tokens del vendedor mientras nadie accede al link; puede dejar escrow atrapado indefinidamente.

---

## 2. Identificador único de sesión P2P

**Decision**: UUID v4 generado en el servicio con `crypto/rand` (stdlib). Almacenado como columna `TEXT` / `UUID` en Postgres.

**Rationale**: El proyecto ya usa UUIDs para `refresh_tokens` (ver `db/init.sql`). El UUID v4 es opaco e impredecible por diseño, lo que satisface el requisito de "enlace semiprivado". No se introduce dependencia externa; `crypto/rand` es stdlib.

**Alternatives considered**:
- ID secuencial: predecible, permite enumeración.
- Token firmado (JWT): complejidad innecesaria; la sesión ya está en base de datos.

---

## 3. Integración con escrow de tokens existente

**Decision**: Reutilizar `Position.ReserveTokens` / `Position.ReleaseReservedTokens` / `Position.DeliverReservedTokens` del modelo de dominio ya existente. Los campos `available_quantity` / `reserved_quantity` de `user_positions` ya implementan el escrow.

**Rationale**: El modelo `Position` ya tiene los métodos de ciclo de vida de reserva. El constraint de DB `chk_positions_available_non_negative` garantiza integridad. No hay que añadir ninguna tabla auxiliar de escrow; la referencia a la sesión P2P vive en la tabla de sesiones y apunta a la posición de forma implícita (user_id + player_id).

**Alternatives considered**:
- Tabla `token_escrow` separada: redundante; duplica la información ya en `user_positions`.

---

## 4. Modelo de transacción completada (settlement)

**Decision**: Al aceptar, emitir un `Trade` análogo al del mercado abierto, usando una referencia opcional `p2p_session_id` en `market_trades` o creando entradas equivalentes en el ledger. Se crea la sesión P2P con sus propios DAOs; el settlement reutiliza `MarketSession.SettleExecution` con un trade P2P marcado por la referencia de sesión.

**Rationale**: Reutilizar el settlement existente garantiza que ledger y auditoría queden completos con el mínimo código nuevo. La única diferencia es el origen (P2P vs. open market); un campo `p2p_session_id BIGINT REFERENCES p2p_sessions(id)` nullable en `market_trades` distingue el origen sin duplicar la lógica de liquidación.

**Alternatives considered**:
- Tabla de transacciones P2P separada: fragmenta el historial; `GET /users/{id}/transactions` debería unir dos tablas.

---

## 5. Acceso al link por usuario no autenticado

**Decision**: Redirigir al login. El endpoint `GET /p2p/sessions/{token}` requiere `AccessAuthenticated`. El frontend detecta 401 y redirige a `/login?redirect=/p2p/sessions/{token}`.

**Rationale**: Consistente con la política del proyecto: ausencia de declaración de acceso es error; no existe concepto de "vista pública". El flujo de redirect post-login es estándar en SPAs.

---

## 6. Concurrencia: doble aceptación simultánea

**Decision**: `SELECT ... FOR UPDATE` en la fila de sesión P2P al intentar aceptar, dentro de la misma transacción de DB que el settlement. El primer `UPDATE status = COMPLETED` gana; el segundo lee el estado terminal y devuelve 409/410.

**Rationale**: Patrón ya usado en `MarketSession.LockAccount` y `LockPosition`. El bloqueo pesimista a nivel fila es la solución estándar del proyecto para race conditions financieras.

---

## 7. Frontend: countdown en tiempo real

**Decision**: Calcular el tiempo restante en el cliente con `setInterval(1000)` usando `expires_at` del response JSON (timestamp UTC). No se usa WebSocket ni SSE.

**Rationale**: El intervalo de polling es 1 segundo, suficiente para un countdown visual. WebSocket/SSE añadiría complejidad de infraestructura no justificada para este caso.

---

## 8. Scheduling de expiración: integración con SyncScheduler

**Decision**: Añadir un segundo ticker en el `SyncScheduler` existente (o crear un `P2PExpirationJob` independiente con el mismo patrón de goroutine controlada por `runCtx`).

**Rationale**: El `SyncScheduler` actual gestiona `playerSync` y `statsSync` con tickers independientes. El patrón `select { case <-ticker.C: ... case <-ctx.Done(): ... }` es idiomático y ya testeado en el proyecto.

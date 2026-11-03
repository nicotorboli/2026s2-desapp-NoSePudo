# Data Model: Transacción P2P de Tokens

**Feature**: `006-transaccion-p2p-tokens`
**Fecha**: 2026-11-03

---

## Entidades de dominio nuevas

### `P2PSession` (model)

Representa una oferta de venta directa iniciada por un vendedor.

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `ID` | `int64` | PK |
| `Token` | `string` | UUID v4 opaco (urlsafe); identificador del enlace |
| `SellerID` | `int64` | FK → `users.id` |
| `BuyerID` | `*int64` | FK → `users.id`; nil hasta que alguien acepta |
| `PlayerID` | `int64` | FK → `players.id` |
| `Quantity` | `int64` | Cantidad de tokens ofertados |
| `Price` | `Credits` | Precio por token en la escala interna (`model.Credits`) |
| `Status` | `P2PSessionStatus` | `PENDING` / `COMPLETED` / `CANCELLED` / `EXPIRED` |
| `CreatedAt` | `time.Time` | Timestamp de creación |
| `ExpiresAt` | `time.Time` | `CreatedAt + 5 min`; calculado en el servicio |
| `CompletedAt` | `*time.Time` | Timestamp de cierre (cualquier estado terminal) |
| `SellerEmail` | `string` | Email del vendedor resuelto en la lectura |
| `BuyerEmail` | `string` | Email del comprador resuelto en la lectura (vacío si aún no hay comprador) |

**Estado de transición**:

```
PENDING ──accept──► COMPLETED
PENDING ──cancel (vendedor)──► CANCELLED
PENDING ──timeout──► EXPIRED
COMPLETED/CANCELLED/EXPIRED → (terminal; sin transición válida)
```

**Reglas de dominio**:
- `Quantity > 0`, `Price > 0`
- `SellerID != BuyerID` (validado en el servicio al aceptar)
- Sólo `PENDING` puede transicionar
- Si ya está `COMPLETED`, rechaza con `ErrP2PSessionAlreadyTaken`
- Si ya está `EXPIRED` o `now > expiresAt`, rechaza con `ErrP2PSessionExpired`

**Métodos de modelo**:
```go
func (s *P2PSession) IsLive() bool
func (s *P2PSession) IsExpired(now time.Time) bool
func (s *P2PSession) Accept(buyerID int64, now time.Time) error   // → COMPLETED
func (s *P2PSession) Cancel(now time.Time) error                  // → CANCELLED
func (s *P2PSession) Expire(now time.Time) error                  // → EXPIRED
func (s *P2PSession) TotalAmount() Credits                        // Quantity * Price
```

**Errores sentinel**:
```go
var ErrP2PSessionNotFound     = errors.New("no existe una sesión P2P con ese token")
var ErrP2PSessionNotLive      = errors.New("la sesión P2P ya no está activa")
var ErrP2PSessionExpired      = errors.New("la oferta venció y ya no puede aceptarse")
var ErrP2PSessionAlreadyTaken = errors.New("la oferta ya fue tomada por otro usuario")
var ErrP2PSessionSelfTrade    = errors.New("el vendedor no puede aceptar su propia oferta")
```

---

## Cambios en entidades existentes

### `market_trades` (SQL)

Añadir columna nullable para vincular trades P2P:

```sql
p2p_session_id BIGINT REFERENCES p2p_sessions(id) ON DELETE RESTRICT
```

### `model.Trade`

Añadir campo:
```go
P2PSessionID *int64
```

---

## Esquema SQL nuevo

```sql
CREATE TABLE IF NOT EXISTS p2p_sessions (
    id           BIGSERIAL    PRIMARY KEY,
    token        UUID         NOT NULL UNIQUE,
    seller_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    buyer_id     BIGINT                REFERENCES users(id) ON DELETE RESTRICT,
    player_id    BIGINT       NOT NULL REFERENCES players(id) ON DELETE RESTRICT,
    quantity     BIGINT       NOT NULL,
    price        BIGINT       NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ  NOT NULL,
    completed_at TIMESTAMPTZ,

    CONSTRAINT chk_p2p_status   CHECK (status IN ('PENDING','COMPLETED','CANCELLED','EXPIRED')),
    CONSTRAINT chk_p2p_quantity CHECK (quantity > 0),
    CONSTRAINT chk_p2p_price    CHECK (price > 0)
);

CREATE INDEX IF NOT EXISTS idx_p2p_sessions_seller   ON p2p_sessions (seller_id, status);
CREATE INDEX IF NOT EXISTS idx_p2p_sessions_token    ON p2p_sessions (token);
CREATE INDEX IF NOT EXISTS idx_p2p_sessions_pending  ON p2p_sessions (status, expires_at)
    WHERE status = 'PENDING';
```

Columna adicional en tabla existente:

```sql
ALTER TABLE market_trades
    ADD COLUMN IF NOT EXISTS p2p_session_id BIGINT REFERENCES p2p_sessions(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_trades_p2p ON market_trades (p2p_session_id)
    WHERE p2p_session_id IS NOT NULL;
```

---

## Capas de acceso a datos nuevas

### DAO: `P2PSessionSql`

| Método | SQL |
|--------|-----|
| `Insert(ctx, tx, session)` | `INSERT INTO p2p_sessions ...` |
| `GetByToken(ctx, tx, token)` | `SELECT ... WHERE token = $1` |
| `LockByToken(ctx, tx, token)` | `SELECT ... WHERE token = $1 FOR UPDATE` |
| `Update(ctx, tx, session)` | `UPDATE p2p_sessions SET status, buyer_id, completed_at WHERE id = $1` |
| `ListPendingExpired(ctx, now)` | `SELECT ... WHERE status='PENDING' AND expires_at <= $1` |
| `ListBySellerLive(ctx, sellerID)` | `SELECT ... WHERE seller_id=$1 AND status='PENDING'` |

### Repository: `P2PRepository`

Coordina `P2PSessionSql` + `PositionSql` + `AccountSql` + `AuditDAO` + `TradeSql` + `LedgerSql` dentro de transacciones de DB.

Operaciones principales:

| Método | Descripción |
|--------|-------------|
| `Create(ctx, session, sellerPosition)` | Crea sesión + reserva tokens vendedor en una tx |
| `Accept(ctx, token, buyerID, buyerAccount, now)` | Lock sesión + settle (tokens+créditos) + trade + ledger + audit |
| `Cancel(ctx, token, actorID, now)` | Lock sesión + libera reserva + audit |
| `ExpirePending(ctx, now)` | Lote de sesiones expiradas: libera reservas + audit |
| `GetByToken(ctx, token)` | Lectura sin lock (para vista pública de la sesión) |

---

## Capas de servicio y controller nuevas

### `P2PService`

Depende de interfaz `P2PRepositoryInterface` (definida junto al servicio). Métodos:

```go
Create(ctx, sellerID int64, playerID int64, quantity int64, price Credits) (P2PSession, error)
Get(ctx, token string) (P2PSession, error)
Accept(ctx, token string, buyerID int64) (Trade, error)
Cancel(ctx, token string, actorID int64) error
```

### `P2PExpirationJob`

Goroutine controlada por contexto, ticker 30 s, llama a `P2PRepository.ExpirePending`.

### `RestP2PController`

| Endpoint | Handler |
|----------|---------|
| `POST /p2p/sessions` | `Create()` |
| `GET /p2p/sessions/{token}` | `GetByToken()` |
| `POST /p2p/sessions/{token}/accept` | `Accept()` |
| `POST /p2p/sessions/{token}/cancel` | `Cancel()` |

---

## Frontend: componentes nuevos

| Archivo | Responsabilidad |
|---------|----------------|
| `src/api/p2p.ts` | Funciones tipadas para los 4 endpoints P2P |
| `src/pages/P2PCreatePage.tsx` | Formulario de creación (jugador, cantidad, precio) + link resultante |
| `src/pages/P2PSessionPage.tsx` | Vista de sesión: ambos usuarios, countdown, botones Aceptar/Cancelar |
| `src/pages/P2PCreatePage.css` | Estilos BEM de la página de creación |
| `src/pages/P2PSessionPage.css` | Estilos BEM de la página de sesión |

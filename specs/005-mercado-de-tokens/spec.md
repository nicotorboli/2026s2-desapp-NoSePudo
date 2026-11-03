# Feature Specification: Player Token Market (Mercado de tokens)

**Feature Branch**: `005-mercado-de-tokens`

**Created**: 2026-10-17

**Status**: Ready for Planning

**Input**: User description: "resolver el 3.3 Mercado de tokens. Para simplificar el dominio, existirá un único superusuario que será el dueño inicial de todos los tokens de los jugadores. Cada jugador tendrá un total de 100 tokens emitidos. En el momento cero, cada token tendrá un valor inicial de 1 crédito. A partir de ese momento, la cotización de los tokens irá cambiando según la estrategia de valuación configurada en el sistema y la evolución del valor del jugador. Los usuarios podrán comprar y vender tokens de jugadores. En una operación de compra, el sistema deberá validar que exista disponibilidad de tokens, utilizar la cotización vigente del jugador, actualizar la posición del usuario y registrar la operación. En una operación de venta, el sistema deberá validar que el usuario posea la cantidad de tokens a vender, actualizar su saldo, ajustar su posición y registrar la operación. Las operaciones de compra de los usuarios deberán realizarse inicialmente contra el superusuario, quien concentra la tenencia inicial de todos los tokens. El sistema deberá permitir el registro y creación de nuevos usuarios para operar en el mercado."

## Clarifications

### Session 2026-10-17 (round 1)

- Q: ¿Con qué saldo de créditos inicial deben arrancar los nuevos usuarios registrados para operar en el mercado? → A: Comienzan con 1000 créditos al registrarse (consistente con los montos de ejemplo del enunciado de visión), y el valor debe ser configurable en el entorno (`NSP_INITIAL_USER_BALANCE`, por defecto 1000).
- Q: ¿Cómo determinamos la cotización vigente del jugador para este módulo (considerando que el momento cero es 1 crédito)? → A: Se crea la tabla de cotizaciones (`player_quotes`) con valor inicial de 1 crédito por token para cada jugador en momento cero (dejando la estructura persistida y lista para ser consumida o recalculada por las estrategias de valuación del punto 3.2).
- Q: ¿Cuál es la contraparte en la venta de tokens? → A: En el modelo inicial del mercado, el superusuario actúa como creador de mercado institucional (*market maker*). Al vender, el usuario transfiere sus tokens de regreso a la tenencia del superusuario a la cotización vigente, recibiendo los créditos correspondientes en su saldo.
- Q: ¿Quién puede consultar el portfolio e historial de transacciones de un usuario? → A: Cada usuario autenticado puede consultar exclusivamente su propio portfolio e historial (`/users/{id}/portfolio`, `/users/{id}/transactions`), mientras que el superusuario puede consultar el de cualquier usuario (o consultar su propia tenencia institucional).

### Session 2026-10-17 (round 2)

- Q: ¿Quién fija el precio de ejecución de una operación? → A: El precio es opcional en el request. Si se omite, la operación usa la cotización vigente del jugador (`player_quotes`), que es lo que pide el punto 3.3 del enunciado. Si el usuario envía un precio, su orden es una orden con límite de precio real y puede quedar sin cruzar.
- Q: Una orden que no puede cruzarse por completo, ¿deja su remanente vivo en el libro o se rechaza? → A: Queda viva, con ejecuciones parciales, reserva de fondos/tokens (*escrow*) y cancelación. El usuario controla el comportamiento del remanente con `timeInForce`: `GTC` (el remanente descansa hasta que se cancele), `IOC` (se ejecuta lo posible y el remanente se cancela liberando la reserva) y `FOK` (se ejecuta entera o se rechaza sin dejar estado). Por defecto: una orden sin precio es `FOK`, una orden con precio es `GTC`.
- Q: ¿Qué endpoints expone el mercado? → A: `POST /orders/buy` y `POST /orders/sell`, exactamente como los nombra la tabla de APIs del punto 4 del enunciado. El `side` queda implícito en la ruta. Se agregan `GET /orders` y `DELETE /orders/{id}`, que son consecuencia necesaria de que existan órdenes vivas: sin ellos la reserva de una orden que nunca cruza quedaría retenida de forma indefinida.
- Q: ¿Cómo se audita cada operación financiera? → A: Con los dos mecanismos. El registro de auditoría con autor y estado anterior/posterior es el que cumple el punto 5.2 del enunciado y reutiliza la entidad de auditoría que ya existe en el sistema; los asientos de partida doble (debe/haber con monto firmado) aportan el invariante contable de que todo movimiento de dinero suma cero.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Buy Player Tokens (Priority: P1)

As an authenticated user with available credits, I want to buy tokens of a football player, so that I can hold a position in that player's value.

**Why this priority**: Core value of the token market. Without purchasing, no positions exist, no portfolio exists and the market has no transactions.

**Independent Test**: Register an account, confirm the initial balance of 1000 credits, and send a buy order for 10 tokens of a player whose active quote is 1 credit. Verify the order is reported as fully executed, the balance drops to 990, the user position is 10 tokens, the superuser position drops from 100 to 90 and both an audit record and a transaction record exist.

**Acceptance Scenarios**:

1. **Given** an authenticated user with 1000 credits and a player whose active quote is 1 credit with the market maker holding 100 tokens, **When** the user sends `POST /orders/buy` with `{"playerId": 1, "quantity": 10}`, **Then** the system applies the active quote as the price, defaults the order to `FOK`, executes it in full against the market maker, moves 10 credits from buyer to seller, moves 10 tokens from seller to buyer, and returns HTTP 201 with the order in state `FILLED` and its executed trades.
2. **Given** the same user, **When** the user sends a buy order whose `quantity * price` exceeds the available balance, **Then** the request is rejected with HTTP 422 (`insufficient funds`), no balance, position or order state changes, and nothing is recorded.
3. **Given** a player for which only 5 tokens are offered at or below the order price, **When** an authenticated user sends a buy order for 10 tokens without a price (therefore `FOK`), **Then** the request is rejected with HTTP 409 (`insufficient liquidity`), no partial execution occurs and no reservation is retained.
4. **Given** the same situation, **When** the user sends a buy order for 10 tokens with an explicit price (therefore `GTC`), **Then** 5 tokens execute immediately, the order is reported as `PARTIALLY_FILLED` with 5 remaining, the credits for those 5 remaining tokens stay reserved, and the remainder stays available in the market for later sellers.
5. **Given** the same situation, **When** the user sends a buy order for 10 tokens with an explicit price and `"timeInForce": "IOC"`, **Then** 5 tokens execute, the remaining 5 are cancelled, the reservation for the cancelled remainder is released back to the available balance, and the order is reported as `CANCELLED` with 5 executed.
6. **Given** a resting sell order priced below the buyer's limit price, **When** a buy order crosses it, **Then** the execution price is the resting order's price (the price the market already offered), and the difference between the reserved amount and the executed amount is released back to the buyer's available balance.
7. **Given** two resting sell orders of the same player at different prices, **When** a buy order large enough to consume both crosses the book, **Then** the cheapest one executes first; at equal prices the older one executes first; the buy order ends `FILLED` with two separate trades.
8. **Given** an order with `quantity <= 0`, a non-integer quantity, a `price <= 0` or an unknown `timeInForce`, **When** submitted, **Then** the request is rejected with HTTP 400 and no state changes.
9. **Given** a `playerId` that does not exist or whose player is inactive, **When** an order is submitted, **Then** the request is rejected with HTTP 404 and no state changes.

---

### User Story 2 - Sell Player Tokens (Priority: P1)

As an authenticated user holding tokens of a player, I want to sell some or all of them, so that I can liquidate my position and recover credits.

**Why this priority**: The other half of market operations. Without selling, positions cannot be closed and the market is not operational.

**Independent Test**: With a user holding 10 tokens, submit a sell order for 4 tokens without a price. Verify the position drops to 6, the balance rises by `4 * active quote`, the market maker position rises by 4, and the recorded cost basis of the remaining position is consistent with the average purchase price.

**Acceptance Scenarios**:

1. **Given** an authenticated user holding 10 tokens of a player whose active quote is 1 credit, **When** the user sends `POST /orders/sell` with `{"playerId": 1, "quantity": 4}`, **Then** the order executes in full against the market maker at the active quote, the user receives 4 credits, the user position drops to 6, the market maker position rises by 4, and HTTP 201 is returned with the order `FILLED`.
2. **Given** an authenticated user holding fewer tokens than requested (including zero), **When** attempting to sell, **Then** the request is rejected with HTTP 422 (`insufficient token holding`) and no state changes.
3. **Given** a user holding 10 tokens who already has a resting sell order reserving 6 of them, **When** the user attempts to sell 6 more, **Then** the request is rejected with HTTP 422: reserved tokens are not available for a second order.
4. **Given** a user who sells part of a position bought at different prices, **When** the sale executes, **Then** the cost basis of the position is reduced in proportion to the quantity sold at the position's average purchase price, so that the average purchase price of the remaining tokens is unchanged by the sale.
5. **Given** a sell order with an explicit price above every resting buy order, **When** submitted with the default `GTC`, **Then** no execution occurs, the order rests in state `PENDING`, and the tokens are reserved and no longer available to sell again.

---

### User Story 3 - Inspect and Cancel Open Orders (Priority: P1)

As an authenticated user with resting orders, I want to see my open orders and cancel them, so that credits or tokens reserved by an order that is not crossing are returned to me.

**Why this priority**: A direct consequence of orders being allowed to rest. Without cancellation the reservation of an order that never crosses is retained indefinitely, which is a defect, not a feature gap.

**Independent Test**: Place a `GTC` buy order that cannot cross, confirm it appears in `GET /orders` with its remaining quantity, cancel it with `DELETE /orders/{id}`, and verify the full reservation returns to the available balance.

**Acceptance Scenarios**:

1. **Given** an authenticated user with two resting orders and one fully executed order, **When** calling `GET /orders`, **Then** only the two resting orders are returned (states `PENDING` and `PARTIALLY_FILLED`), each with its original quantity, executed quantity, remaining quantity, price, side and creation time.
2. **Given** an authenticated user, **When** calling `GET /orders`, **Then** only that user's own orders are returned, never another user's.
3. **Given** a resting buy order with 30 of 80 tokens remaining, **When** the owner calls `DELETE /orders/{id}`, **Then** the order moves to `CANCELLED`, the reservation covering the 30 remaining tokens returns to the available balance, the 50 already executed are untouched, and HTTP 200 is returned.
4. **Given** a resting sell order, **When** the owner cancels it, **Then** the reserved tokens return to the available position.
5. **Given** an order that is already `FILLED` or `CANCELLED`, **When** cancellation is attempted, **Then** HTTP 409 is returned and no state changes.
6. **Given** an order belonging to another user, **When** cancellation is attempted by a non-owner, **Then** HTTP 403 is returned.

---

### User Story 4 - View Portfolio and Performance (Priority: P2)

As an authenticated user, I want to see my position: tokens per player, average purchase price, current value and profit or loss, so that I can assess my investment.

**Why this priority**: Explicitly required by section 3.4 of the vision document.

**Independent Test**: After buying 10 tokens at 1 credit and 10 more at 2 credits, with the active quote at 2 credits, call `GET /users/{id}/portfolio` and verify quantity 20, average purchase price 1.5, current value 40 and profit 10.

**Acceptance Scenarios**:

1. **Given** the situation above, **When** requesting the portfolio, **Then** the response reports per position: player, quantity, average purchase price, current quote, current valuation and unrealized profit or loss; plus the user's cash balance, the total token valuation and the total portfolio value.
2. **Given** a user with tokens reserved by a resting sell order, **When** requesting the portfolio, **Then** the position quantity includes the reserved tokens, because they are still owned, and the reserved cash is reported separately from the available cash.
3. **Given** a user with no positions, **When** requesting the portfolio, **Then** an empty position list is returned with the cash balance, not an error.
4. **Given** user A requesting the portfolio of user B, where A is not B and A is not the superuser, **When** the request is made, **Then** HTTP 403 is returned.
5. **Given** the superuser requesting any user's portfolio, **When** the request is made, **Then** that user's portfolio is returned.

---

### User Story 5 - View Transaction History (Priority: P2)

As an authenticated user, I want to see my executed operations in chronological order, so that I have a record of everything I traded.

**Why this priority**: Required by sections 3.4 and 4 of the vision document.

**Independent Test**: Execute one purchase and one sale, then call `GET /users/{id}/transactions` and verify both appear with their side, player, quantity, unit price, total and timestamp.

**Acceptance Scenarios**:

1. **Given** a user with executed operations, **When** calling `GET /users/{id}/transactions`, **Then** the executed trades are returned newest first with pagination, each reporting side (`BUY` or `SELL`) from that user's perspective, player, quantity, execution price, total amount and execution time.
2. **Given** a user whose order executed against two counterparties at different prices, **When** listing transactions, **Then** each execution appears as its own entry with its own price, not merged into one.
3. **Given** user A requesting user B's transactions without superuser privilege, **When** the request is made, **Then** HTTP 403 is returned.

---

### User Story 6 - Market Bootstrap at Moment Zero (Priority: P2)

As the platform, every player in the catalog must have exactly 100 issued tokens held by the superuser and an initial quote of 1 credit, offered to the market, so that users can trade from the first minute without any manual setup.

**Why this priority**: Domain precondition stated by the enunciado, and the condition that makes the market demonstrable during evaluation.

**Independent Test**: Boot the server on an empty database, run a player synchronization, and verify that every player has exactly one initial quote of 1 credit, that the superuser holds exactly 100 tokens of each, and that one sell order of the remaining market maker inventory rests in the book. Then boot again and verify nothing was duplicated.

**Acceptance Scenarios**:

1. **Given** a player newly added to the catalog, **When** the market bootstrap runs, **Then** the player gets an initial quote of 1 credit, the superuser gets 100 tokens, and a resting sell order for those tokens at the active quote is placed on the superuser's behalf.
2. **Given** a player already initialized, **When** the bootstrap runs again (server restart or a new synchronization), **Then** no additional tokens are issued, no additional initial quote is recorded and no duplicate market maker order is placed. Existing positions, quotes and orders are preserved.
3. **Given** a player whose market maker tokens were partly sold to users, **When** the bootstrap runs again, **Then** the already sold tokens are not reissued: total issued tokens per player stay at 100.
4. **Given** a new quote recorded for a player (by the valuation strategies of section 3.2), **When** the market maker's resting order is priced at the previous quote, **Then** that resting order is re-priced to the new quote, so that the market maker never keeps offering tokens at a stale price. Orders belonging to regular users are never re-priced.

---

### User Story 7 - Registration with Initial Balance (Priority: P3)

As a new visitor, I want my account to start with a credit balance, so that I can operate in the market without any external payment.

**Why this priority**: Precondition for any user to trade, and for the evaluation scenario that creates four users.

**Independent Test**: Register a user via `POST /auth/register` and verify the available balance equals `NSP_INITIAL_USER_BALANCE` (1000 by default).

**Acceptance Scenarios**:

1. **Given** valid registration credentials, **When** `POST /auth/register` is called, **Then** the account is created with an available balance equal to the configured initial balance and zero reserved balance.
2. **Given** a configured initial balance different from the default, **When** a user registers, **Then** the configured value is applied.
3. **Given** the superuser account created at boot, **When** the market bootstrap runs, **Then** the superuser also has an account, so that it can receive credits for the tokens it sells.

---

### Edge Cases

- **Invalid input**: `quantity <= 0`, non-integer quantity, `price <= 0`, unknown `timeInForce`, unknown player: HTTP 400 or 404, with no state change.
- **Concurrent orders over the last tokens**: two simultaneous buy orders for the last available token must be serialized; one executes and the other is rejected for lack of liquidity or rests, depending on its `timeInForce`. Neither may oversell the position nor overspend the balance.
- **Concurrent orders by the same user**: two orders submitted simultaneously by one user may not both reserve the same credits or the same tokens.
- **Self-crossing**: an order that would cross against the same user's own resting order must not execute against itself; it skips that order in the book.
- **Market maker selling to itself**: the superuser is not expected to trade with itself; if it submits an order that would cross its own resting order, the same self-crossing rule applies.
- **Rounding**: all credit amounts use fixed-point arithmetic, never binary floating point, so that the average purchase price and the profit or loss are exact to four decimal places.
- **Zero liquidity**: a player with no resting sell orders accepts buy orders only under `GTC`; `FOK` and `IOC` orders are rejected or cancelled with a clear error, never silently accepted.
- **Reservation never retained silently**: every path that ends without full execution (rejection, partial execution under `IOC`, cancellation) must leave the reservation released. No path may leave credits or tokens reserved by an order that is no longer live.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST issue exactly 100 tokens per player, initially held by the single superuser, and MUST NOT reissue them on subsequent runs.
- **FR-002**: The system MUST record an initial quote of 1 credit per token for every player at moment zero, in a dedicated quote history that also records which valuation strategy and strategy version produced each quote.
- **FR-003**: The system MUST credit every newly registered user with the configured initial balance (`NSP_INITIAL_USER_BALANCE`, default 1000) and MUST maintain, per user, the available balance separately from the balance reserved by live orders.
- **FR-004**: The system MUST expose `POST /orders/buy` (authenticated), accepting `playerId`, `quantity`, an optional `price` and an optional `timeInForce`, and MUST:
  - Reject an unknown or inactive player, a non-positive quantity and a non-positive price.
  - Use the player's active quote as the order price when `price` is absent.
  - Default `timeInForce` to `FOK` when `price` is absent and to `GTC` when `price` is present.
  - Reject the order when the available balance is below `quantity * price`.
  - Reserve `quantity * price` from the available balance before attempting to cross.
  - Cross against resting sell orders of that player priced at or below the order price, cheapest first and, at equal price, oldest first, skipping the submitter's own orders.
  - Execute each match at the resting order's price and release the difference between the reserved amount and the executed amount back to the available balance.
  - Apply the remainder rule of the order's `timeInForce`: `GTC` leaves the remainder resting with its reservation held, `IOC` cancels the remainder and releases its reservation, `FOK` rejects the entire order with no state change at all when it cannot execute in full.
  - Return the resulting order with its state, executed quantity, remaining quantity and executed trades.
- **FR-005**: The system MUST expose `POST /orders/sell` (authenticated) with the symmetric behavior of FR-004: it reserves tokens instead of credits, requires the available position to cover the quantity, and crosses against resting buy orders priced at or above the order price, most expensive first and, at equal price, oldest first.
- **FR-006**: The system MUST expose `GET /orders` (authenticated), returning only the caller's live orders (`PENDING` and `PARTIALLY_FILLED`), optionally filtered by player, each with side, price, original quantity, executed quantity, remaining quantity, state and creation time.
- **FR-007**: The system MUST expose `DELETE /orders/{id}` (authenticated), which cancels a live order of the caller, releases the reservation covering its remaining quantity, leaves already executed quantity untouched, rejects cancellation of an order that is already final with HTTP 409, and rejects a non-owner with HTTP 403.
- **FR-008**: The system MUST expose `GET /users/{id}/portfolio` (authenticated), returning the available and reserved cash balance, and per position: player identity, quantity owned including reserved tokens, average purchase price, active quote, current valuation and unrealized profit or loss; plus total token valuation and total portfolio value. A regular user may only read their own portfolio; the superuser may read any.
- **FR-009**: The system MUST expose `GET /users/{id}/transactions` (authenticated), returning the caller's executed trades newest first with pagination, one entry per execution, each with side from that user's perspective, player, quantity, execution price, total amount and execution time. A regular user may only read their own history; the superuser may read any.
- **FR-010**: The system MUST reduce a position's cost basis in proportion to the quantity sold, valued at the position's average purchase price, so that selling part of a position does not change the average purchase price of the remainder.
- **FR-011**: The system MUST keep the market maker's resting orders priced at the player's active quote, re-pricing them when a new quote is recorded, and MUST NOT re-price orders belonging to regular users.
- **FR-012**: The system MUST record, for every change to a balance, a position or an order state, an immutable audit entry with the actor who caused it, the previous state, the resulting state and the timestamp, as required by section 5.2 of the vision document.
- **FR-013**: The system MUST record, for every monetary movement, immutable double-entry accounting entries whose signed amounts sum to zero per settlement, including the initial balance granted at registration.
- **FR-014**: The system MUST apply every order submission, execution and cancellation as a single atomic unit: either every balance, position, order, trade, accounting entry and audit entry of that operation is persisted, or none is.
- **FR-015**: The system MUST serialize concurrent operations over the same player so that no sequence of concurrent orders can overspend a balance, oversell a position or issue more than 100 tokens of a player.
- **FR-016**: The system MUST document every new endpoint in `backend/api/openapi.yaml` with its explicit access level, and MUST keep the published contract consistent with the registered routes.

### Key Entities

- **Account**: A user's credit balance, split into the available balance and the balance reserved by live orders.
- **Position**: The tokens a user holds of a player, split into available and reserved tokens, plus the accumulated cost basis used to derive the average purchase price.
- **Order**: A user's intent to buy or sell a quantity of a player's tokens at a price, with a time in force governing its remainder, and a lifecycle from live (`PENDING`, `PARTIALLY_FILLED`) to final (`FILLED`, `CANCELLED`). An order that executes nothing and does not rest is rejected and never recorded, so the order history is a record of what happened rather than of what was attempted.
- **Trade**: The immutable record of one execution between a buy order and a sell order: player, quantity, execution price, both parties and the execution time.
- **PlayerQuote**: The valuation of a player's token at a point in time, with the strategy and strategy version that produced it.
- **LedgerEntry**: One signed accounting entry of a monetary movement, tied to the settlement that caused it.
- **AuditEntry**: The actor, previous state, resulting state and timestamp of a change to a balance, a position or an order.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Under concurrent load over the same player, no sequence of orders results in a negative balance, a negative position, more than 100 tokens issued per player, or credits reserved by an order that is no longer live.
- **SC-002**: Average purchase price, current valuation and profit or loss are exact to four decimal places, with no accumulated rounding error after a sequence of purchases and sales at different prices.
- **SC-003**: Every registered route is declared in the published contract with its access level, verified by the existing contract test.
- **SC-004**: The evaluation scenario of section 8.3 of the vision document runs as an automated end-to-end test: four users are created, each buys tokens of five players, and the current valuation of their positions is reported after the quotes move.
- **SC-005**: Every state that a resting order can reach is covered by tests, including cancellation of a partially executed order, a remainder that crosses later, and the release of a reservation under every non-executing path.
- **SC-006**: The existing test suite of features 001 to 004 keeps passing unchanged, with no test modified or relaxed.
- **SC-007**: `golangci-lint run` passes with zero findings.

## Assumptions

- The single superuser created from `NSP_SUPERUSER_EMAIL` and `NSP_SUPERUSER_PASSWORD` is the initial holder of every player's tokens and acts as the market maker.
- The active quote of a player is the most recent entry in its quote history; at moment zero it is 1 credit.
- Credits are internal platform currency. Deposits and withdrawals against real money are out of scope.
- A user is never hard deleted, so trades and accounting entries keep referring to existing users.

## Out of Scope

- Quote recalculation and the valuation strategies themselves, the player ranking and the quote history endpoints: those belong to section 3.2 of the vision document. This feature only reads the active quote and defines where quotes are stored.
- The job scheduler and the cache layer required by section 5.4.
- The market screens of the frontend required by section 9. This feature delivers the backend contract they will consume.

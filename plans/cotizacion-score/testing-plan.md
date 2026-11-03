# Plan: tests de cotización de jugadores por score

Tests agregados para cubrir la feature de cotización (`precio = base +
score × escala`), el registro de cotizaciones en cada recálculo y la
reconstrucción del precio a una fecha.

Solo se tocaron archivos de test: no se modificó implementación.

## Archivos nuevos

- `backend/internal/model/player_price_test.go`
- `backend/internal/dto/credits_test.go`
- `backend/internal/dto/player_price_test.go`
- `backend/test/e2e/cotizacion_test.go`

## Archivos modificados

- `backend/internal/persistence/dao/quote_sql_test.go`
  - Seed de un `player_score` (`seedPlayerScore`).
  - `Insert` persiste `score_id` / lo deja nulo.
  - `ListByPlayer` ordena más-reciente-primero y devuelve vacío sin quotes.
- `backend/internal/persistence/repository/player_score_test.go`
  - `Save` completa los IDs (`RETURNING id`).
- `backend/internal/persistence/repository/league_coefficient_test.go`
  - `AtOrBefore` vigente, límite inclusive y sin coeficiente.
- `backend/internal/persistence/repository/market_test.go`
  - `LatestPrices` y `QuoteHistory` (round-trip y vacío).
- `backend/internal/service/market_maker_test.go`
  - `RecordScoreQuote` graba el precio calculado, re-precía la oferta del
    creador y con score 0 cobra la base.
- `backend/internal/service/score_test.go`
  - Fixture extendido con `repos` y `marketMaker`; variante
    `newScoreServiceQuoteFixture` con quote reader/writer.
  - `recalculate` graba una cotización por score y no duplica al recalcular.
  - `GetPriceAt` feliz, coeficiente histórico, sin score y fecha futura.
  - `GetQuoteHistory` feliz, vacío y jugador inexistente.
  - `ListRanking` incluye el precio.
- `backend/internal/service/player_test.go`
  - `GetPlayerByID` incluye/omite el precio según haya cotización.
- `backend/internal/controller/scoring_test.go`
  - Mock `QuoteService`; endpoints `GetQuoteHistory` y `GetPriceAt` con casos
    200, 400 (id/fecha) y 404.

## Cobertura por capa

1. Unitarios (sin Docker): fórmula de precio, `CreditsADecimal` y DTOs.
2. Persistencia (testcontainers): `score_id`, `RETURNING id`, `AtOrBefore`,
   `LatestPrices`, `QuoteHistory`.
3. Servicio (testcontainers): registro/reprecio de cotizaciones, idempotencia,
   `GetPriceAt`, `GetQuoteHistory`, precio en ranking/detalle.
4. Controller (sin Docker): endpoints y errores de validación.
5. e2e (`cotizacion_test.go`): superusuario → bootstrap → activar estrategia →
   score, cotización, precio, historial, oferta del creador y ranking.

## Verificación

- `go vet ./...`: OK.
- `golangci-lint run ./...` (v2.14.0): 0 issues.
- Tests sin Docker (model, dto, controller, server, httphandler, middleware,
  configuration, adapters): OK.
- Tests con testcontainers (dao, repository, service, e2e): no ejecutados por
  falta de Docker en el entorno.

## Nota

`PricePolicy` redondea a la diezmilésima de crédito (la precisión de `Credits`),
no al crédito entero; el test de redondeo refleja ese comportamiento.

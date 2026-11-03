# Plan: cotización de jugadores por score

Implementación de la cotización en créditos de los tokens de jugador a partir de
su score, con historial de cotizaciones y reconstrucción del precio a una fecha.

## Fórmula

`precio = base + score × escala`, con `base = 1 crédito`, `escala = 100 créditos`,
piso en `base` y sin techo. El `score` es el final:
`percentil × coeficiente de liga × descuento por posición`.

El precio se recalcula en cada recálculo de score (post-sync de estadísticas y
activación de estrategia) y se persiste una cotización por cada score nuevo. El
market-maker re-precía su orden de venta con esa cotización.

## Modificaciones

### Dominio
- `backend/internal/model/player_price.go` (nuevo):
  - `DefaultPriceBaseUnits = 1`, `DefaultPriceScaleUnits = 100`.
  - `PricePolicy{Base, Scale Credits}` con `Price(score float64) Credits`
    (redondeo al crédito entero y piso en `Base`).
  - `DefaultPricePolicy()`.
- `backend/internal/model/quote.go`: `PlayerQuote` suma `ScoreID *int64`.

### Schema
- `backend/db/init.sql`: `player_quotes` suma
  `score_id BIGINT NULL REFERENCES player_scores (id) ON DELETE SET NULL`.
  Requiere recrear el volumen (`docker compose down -v && docker compose up -d`).

### Persistencia
- `backend/internal/persistence/dao/player_score_sql.go`: `Insert` usa
  `RETURNING id` y completa `scores[i].ID`, para poder asociar la cotización al
  score que la originó.
- `backend/internal/persistence/dao/quote_sql.go`:
  - `Insert` persiste `score_id`.
  - Nuevo `ListByPlayer` (historial, más reciente primero).
- `backend/internal/persistence/dao/league_coefficient_sql.go`: nuevo
  `AtOrBefore(leagueCode, at)` (coeficiente vigente a una fecha).
- `backend/internal/persistence/repository/league_coefficient.go`: `AtOrBefore`
  en la interfaz `LeagueCoefficientDAO` y en el repositorio.
- `backend/internal/persistence/repository/market.go`: nuevos `LatestPrices` y
  `QuoteHistory`.

### Servicio
- `backend/internal/service/market_maker.go`:
  - `RecordQuote` delega en un `recordQuote` privado (misma firma pública).
  - Nuevo `RecordScoreQuote`, que calcula el precio con `DefaultPricePolicy` y
    persiste la cotización con `ScoreID`.
- `backend/internal/service/score.go`:
  - Interfaces `QuoteReader` y `QuoteWriter`.
  - Opciones `WithScoreQuoteReader` y `WithScoreQuoteWriter`.
  - `LeagueCoefficientRepository` suma `AtOrBefore`.
  - `recalculate` arma `playerByID` y llama a `recordQuotes`, que graba una
    cotización por score nuevo.
  - `GetPriceAt`: reconstruye el precio a una fecha desde el score vigente y el
    coeficiente vigente a esa fecha.
  - `GetQuoteHistory`: historial materializado de cotizaciones.
  - `rankingItems`: incorpora el precio vigente a cada ítem del ranking.
- `backend/internal/service/player.go`: opción `WithQuoteReader`; el detalle del
  jugador incorpora el precio vigente.
- `backend/internal/service/container.go`: crea el market-maker antes del score
  service e inyecta `repos.Market` como `QuoteReader` y el market-maker como
  `QuoteWriter`.

### DTO
- `backend/internal/dto/credits.go` (nuevo): `CreditsADecimal`.
- `backend/internal/dto/player_price.go` (nuevo): `PlayerPriceResponse`,
  `PlayerQuoteResponse` y sus conversiones.
- `backend/internal/dto/ranking.go`: `PlayerRankingItemResponse.Price *string`.
- `backend/internal/dto/player_detail.go`: `PlayerDetailResponse.Price *string`.

### Controller / rutas / contrato
- `backend/internal/controller/scoring.go`: interfaz `QuoteService`, opción
  `WithQuoteService`, endpoints `GetQuoteHistory` y `GetPriceAt` (con `date`
  opcional, RFC3339 o fecha sola).
- `backend/internal/controller/container.go`: inyecta `services.Score` como
  `QuoteService`.
- `backend/internal/server/routes.go`: `GET /players/{id}/quotes` y
  `GET /players/{id}/price` (anónimas).
- `backend/internal/server/routes_test.go`: agrega las dos rutas al mapa
  `expected` (registro de rutas; autorizado explícitamente).
- `backend/api/openapi.yaml`: documenta los dos endpoints y agrega los schemas
  `PlayerPrice` y `PlayerQuoteHistoryItem`; agrega `price` a `PlayerDetail` y
  `PlayerRankingItem`.

## Verificación

- `go build ./...`: OK.
- `go vet ./...`: OK.
- `golangci-lint run ./...` (v2.14.0): 0 issues.
- Tests sin Docker (adapters, configuration, httphandler, logger, middleware,
  server, model, dto, controller): OK.
- Tests con testcontainers (dao, repository, service, e2e): no ejecutados por
  falta de Docker en el entorno.

## Pendiente de decisión

- La `escala` quedó como constante en código; `PricePolicy` está preparada para
  configurarse por operador más adelante.
- La fecha sola (`YYYY-MM-DD`) se interpreta como medianoche UTC.

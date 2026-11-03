# Plan: sistema de scoring (esqueleto de arquitectura)

Implementación de la arquitectura de scoring acordada: score relativo por
(posición × liga) vía histograma, coeficiente de liga aplicado on-read, y
estrategias de cálculo como abstracción en código (sin implementaciones ni
datos concretos en esta entrega).

## Modificaciones

### Schema
- `backend/db/init.sql`: nuevas tablas `scoring_strategy_activations`,
  `scoring_histograms`, `player_scores` (UNIQUE por jugador+estrategia+stats),
  `league_coefficients`, con sus índices.

### Model
- `backend/internal/model/player_statistics.go`: agregado `ID int64`.
- `backend/internal/model/player_score.go`: `PlayerScore`.
- `backend/internal/model/score_histogram.go`: `ScoreScope`, `ScoreHistogram`.
- `backend/internal/model/strategy_activation.go`: `StrategyActivation`.
- `backend/internal/model/league_coefficient.go`: `LeagueCoefficient`.

### DAO
- `backend/internal/persistence/dao/player_statistics_sql.go`: `ListByPlayerID`
  ahora selecciona `id` y ordena por `created_at DESC, id DESC`.
- `backend/internal/persistence/dao/player_score_sql.go`: `Insert`,
  `ExistsFor`, `ListByPlayer`.
- `backend/internal/persistence/dao/score_histogram_sql.go`: `Save`, `GetByScope`.
- `backend/internal/persistence/dao/strategy_activation_sql.go`: `Insert`, `Latest`.
- `backend/internal/persistence/dao/league_coefficient_sql.go`: `Insert`, `CurrentFor`.
- `backend/internal/persistence/dao/container.go`: +4 DAOs.

### Repository
- `backend/internal/persistence/repository/player_score.go`,
  `score_histogram.go`, `strategy_activation.go`, `league_coefficient.go`:
  interfaces DAO del lado consumidor + `*Impl` con transacción donde aplica.
- `backend/internal/persistence/repository/container.go`: +4 repos.

### Service
- `backend/internal/service/scoring_strategy.go`: interfaz `ScoringStrategy`
  (`Name`, `Weights`, `Score`).
- `backend/internal/service/histogram.go`: `buildHistogram`, `percentile`
  (funciones puras, 1000 buckets).
- `backend/internal/service/score.go`: `ScoreService` + interfaces consumidas
  (`ScoreRepository`, `HistogramRepository`, `StrategyActivationRepository`,
  `LeagueCoefficientRepository`); recálculo con idempotencia
  `(player, strategy, statistics_id)`, activación de estrategia, y lecturas con
  coef aplicado on-read.
- `backend/internal/service/player_statistics.go`: dependencia `*ScoreService`;
  `SyncStatistics` invoca `RecalculateAfterSync` por liga.
- `backend/internal/service/container.go`: wiring del `ScoreService` (registry
  de estrategias vacío) y del `PlayerStats`.

### DTO
- `backend/internal/dto/score.go`: `PlayerScoreResponse` + `PlayerScoreDesdeModelo`.
- `backend/internal/dto/scoring.go`: `ScoringStrategyResponse`,
  `ActiveStrategyResponse`, `LeagueCoefficientResponse`, `ActivateStrategyRequest`,
  `SetLeagueCoefficientRequest`.

### Controller
- `backend/internal/controller/scoring.go`: `RestScoringController` + interfaz
  `ScoringService` local.
- `backend/internal/controller/container.go`: +`Scoring`.

### Server / API / arranque
- `backend/internal/server/routes.go`: 7 rutas nuevas (lecturas `authenticated`,
  mutaciones `superuser`).
- `backend/api/openapi.yaml`: 7 operaciones con `x-access-level` + schemas.
- `backend/cmd/server/main.go`: 4 tablas nuevas en `expectedTables`.

### Tests (compile-required)
- `backend/internal/service/player_statistics_sync_test.go` y
  `player_statistics_test.go`: ajuste del constructor `NewPlayerStatisticsService`.
- `backend/internal/server/routes_test.go` y `server_test.go`: stub de scoring +
  mapa de rutas esperadas.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l .` | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| Contract test + tests de rutas | pasan |
| Suite no-Docker (model, adapters, middleware, httphandler, configuration, controller, server, dao, service mocks, repository mocks) | pasa |
| Tests con Docker (service estadísticas, repository, e2e) | a cargo del usuario |

## Notas
- El registry de estrategias está vacío: no hay `ScoringStrategy` concretas, así
  que `GET /scoring/strategies` devuelve vacío y el recálculo es no-op mientras
  no haya estrategia activa.
- `league_coefficients` es append-only; el coef se aplica on-read y cambiarlo no
  dispara recálculo. La reconstrucción histórica del coef (método `At`) queda
  como evolución futura.
- Las tablas nuevas requieren `docker compose down -v && docker compose up -d`
  para aplicarse en un volumen existente.

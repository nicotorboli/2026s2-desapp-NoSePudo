# Plan: scoring en el frontend (ranking + score en catálogo y detalle)

Implementación de la capa de lectura de scoring en el backend (endpoint de
ranking) y en el frontend (página de ranking, columna de score ordenable en el
catálogo y score con historial en el detalle). El cálculo del score sigue
siendo on-read en el backend.

## Modificaciones

### Backend — endpoint de ranking

- `backend/internal/dto/ranking.go` (nuevo): `PlayerRankingItemResponse`
  (`score` nullable) y `PlayerRankingPageResponse` (`items`, `offset`, `limit`,
  `total`).
- `backend/internal/persistence/dao/player_score_sql.go`: `LatestPercentiles`,
  con `DISTINCT ON (player_id)` y `ORDER BY player_id, computed_at DESC, id DESC`
  para traer el último percentil por jugador en una sola consulta.
- `backend/internal/persistence/repository/player_score.go`: `LatestPercentiles`
  en la interfaz `PlayerScoreDAO` y en `PlayerScoreRepositoryImpl` (guarda para
  lista vacía).
- `backend/internal/service/score.go`:
  - `LatestPercentiles` agregado a la interfaz `ScoreRepository`.
  - `ListRanking(ctx, filter, sortOrder, offset)`:
    - `sort=name`: paginación server-side por nombre; adjunta el score de la
      página.
    - `sort=score_asc|score_desc`: lista todos los que matchean, calcula
      `percentil × coeficiente × descuento` en Go, ordena (nulos al final) y
      devuelve el chunk `[offset, offset+limit)`.
    - Cachea los coeficientes por liga dentro del request.
  - Constante `rankingMaxPlayers` para el tope de listado.
- `backend/internal/controller/scoring.go`: handler `GetPlayersRanking` (parsea
  `sort`, `offset` y filtros) y `ListRanking` en la interfaz local
  `ScoringService`.
- `backend/internal/server/routes.go`: ruta `GET /players/ranking`
  (`AccessAnonymous`).
- `backend/api/openapi.yaml`: operación `getPlayersRanking` con
  `x-access-level: anonymous`, parámetros (`offset`, `limit`, `sort`, filtros) y
  schemas `PlayerRankingItem` / `PlayerRankingPage`.
- Tests: stubs de `routes_test.go` y `server_test.go` con `ListRanking`; test de
  `LatestPercentiles` en `repository/player_score_test.go`; tests de `ListRanking`
  (orden y nulos al final) en `service/score_test.go`.

### Frontend

- `frontend/src/api/players.ts`: DTOs `PlayerScoreDTO`, `PlayerRankingItemDTO`,
  `PlayerRankingPageDTO`, `RankingSort`; funciones `getPlayerScore`,
  `getPlayerScores`, `getPlayersRanking`. Se retiran `getPlayers` y
  `PaginatedPlayersDTO` (el catálogo migra al ranking).
- `frontend/src/components/ScoreBar/` (nuevo): porcentaje (2 decimales) + barra;
  `N/R` cuando no hay score.
- `frontend/src/pages/RankingPage/` (nuevo): ranking por score descendente con
  paginado mixto (chunk de 100 por fetch + 25 por página cliente-side), puesto
  `#N`, y toggle ascendente/descendente.
- `frontend/src/components/Navbar/Navbar.tsx` + `Navbar.css`: link "Ranking" y
  "Catálogo".
- `frontend/src/App.tsx`: estado de vista `catalog | detail | ranking`.
- `frontend/src/pages/PlayerListPage/PlayerListPage.tsx`: consume
  `getPlayersRanking` con `offset`/`limit`, filtros y `sort`.
- `frontend/src/components/PlayerTable/PlayerTable.tsx` + `PlayerTable.css`:
  columna de score y header ordenable (nombre default, score asc/desc).
- `frontend/src/components/PlayerScore/` (nuevo): score actual + desglose
  (percentil, coeficiente, estrategia) + historial.
- `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.tsx`: obtiene y muestra
  el score y el historial del jugador.
- Tests: `api/players.test.ts`, `PlayerListPage.test.tsx`,
  `PlayerDetailPage.test.tsx`, `App.test.tsx`, `ScoreBar.test.tsx` (nuevo) y
  `RankingPage.test.tsx` (nuevo).

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l .` | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| Backend no-Docker (`dto`, `controller`, `server`, `model`) | pasa |
| Binarios de test `service` / `repository` | compilan |
| Frontend `npm run typecheck` | pasa |
| Frontend `npm run lint` | pasa |
| Frontend `npm run knip` | pasa |
| Frontend `npm run test` | 20 tests en 6 archivos, pasan |
| Tests con Docker (`service`, `repository`, `e2e`) | no ejecutados (Docker no disponible en el entorno) |

## Deuda técnica

- **Cache del ranking (rendimiento).** Cada fetch de chunk vuelve a ordenar todo
  en Go (el score es on-read). Se resuelve con una caché en memoria del ranking
  ordenado, invalidada en `ActivateStrategy`, `RecalculateAfterSync` y
  `SetLeagueCoefficient` (los tres puntos que cambian percentiles o coeficientes).
  Advertencias: riesgo de staleness si se agrega un punto de recálculo sin
  invalidar, y necesidad de sincronización por concurrencia. Redis solo se
  justificaría ante múltiples réplicas del backend (caché compartida) o para
  sobrevivir reinicios; para una instancia y ~miles de jugadores, una caché en
  memoria alcanza. No implementado en esta entrega.
- **`rankingMaxPlayers`** acota el listado para el orden por score; si el catálogo
  supera ese tope, el ranking se trunca en silencio.

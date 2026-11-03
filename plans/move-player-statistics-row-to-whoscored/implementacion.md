# Plan: mover `PlayerStatisticsRow` a `adapters/whoscored`

Refactor pedido explícitamente. `PlayerStatisticsRow` es la salida normalizada
del scraper de WhoScored (nunca se persiste, sin invariantes), así que no
pertenece a `model`. Se mueve al paquete del adaptador que lo produce.

## Modificaciones

### `backend/internal/model/player_statistics.go`
- Eliminado el tipo `PlayerStatisticsRow` (y su comentario). Queda solo
  `PlayerStatistics`.

### `backend/internal/adapters/whoscored/dto.go`
- Agregado `PlayerStatisticsRow` (con su comentario original verbatim).
- `mergeFeeds` devuelve `[]PlayerStatisticsRow` y usa literales del tipo local.
- Eliminado el import de `model` (quedó sin uso).

### `backend/internal/adapters/whoscored/client.go`
- `FetchLeagueStatistics` devuelve `[]PlayerStatisticsRow`.
- Eliminado el import de `model` (era su única referencia).

### `backend/internal/service/dependencies.go`
- `WhoscoredClient.FetchLeagueStatistics` devuelve
  `[]whoscored.PlayerStatisticsRow`.
- Agregado el import de `whoscored`.

### Tests
- `backend/internal/adapters/whoscored/client_test.go`: `model.PlayerStatisticsRow`
  → `whoscored.PlayerStatisticsRow`; eliminado el import de `model`.
- `backend/internal/service/player_statistics_sync_test.go`: idem.

### `backend/internal/service/player_statistics.go`
- Sin cambios: `syncLeague` usa el tipo por inferencia y ya importa `whoscored`.

## Sin cambios
- Comportamiento, queries, matcheo y DTOs.
- `model.PlayerStatistics` y la persistencia.

## Verificación

| Check | Resultado |
|---|---|
| Búsqueda de `model.PlayerStatisticsRow` | sin coincidencias |
| `gofmt -l` sobre los paquetes tocados | sin salida |
| `go build ./...` | pasa |
| `go vet ./internal/adapters/whoscored/... ./internal/service/... ./internal/model/...` | pasa |
| Tests de `whoscored` / servicio (Docker) | a cargo del usuario |

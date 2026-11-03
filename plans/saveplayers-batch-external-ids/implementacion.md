# Plan: eliminar el N+1 de `SavePlayers`

Refactor #2 del relevamiento. `SavePlayers` consultaba `GetPlayerByExternalID`
por cada jugador dentro del loop para saber si ya existía (y así decidir el
audit `INSERT` vs `UPDATE` y calcular el diff). Pasa a una única consulta batch
+ lookup en memoria. El audit asincrónico se descartó explícitamente.

## Modificaciones

### `backend/internal/persistence/dao/player_sql.go`
- Eliminado `GetPlayerByExternalID`.
- Nuevo `GetPlayersByExternalIDs(ctx, externalIDs []int64) ([]model.Player, error)`:
  una sola query `WHERE external_id = ANY($1)` con `pq.Array`, reusando
  `playerColumns` y `scanPlayer`.

### `backend/internal/persistence/repository/player.go`
- Interfaz `PlayerDAO`: `GetPlayerByExternalID` reemplazado por
  `GetPlayersByExternalIDs`.
- `SavePlayers`: `activeExternalIDs` se arma antes del loop; se hace un único
  `GetPlayersByExternalIDs` y se construye `map[externalID]model.Player`; en el
  loop `isNew` se reemplaza por `exists := existingByExternalID[...]` y el audit
  `UPDATE` compara contra `existingPlayer` del map.

### Tests (compile-required)
- `backend/internal/persistence/dao/player_sql_test.go`: el caso nil-db usa
  `GetPlayersByExternalIDs(ctx, []int64{1})`.
- `backend/internal/persistence/repository/player_test.go`: en `mockPlayerDAO`,
  se quita `getPlayerByExternalIDFn`/`GetPlayerByExternalID` y se agrega
  `getPlayersByExternalIDsFn`/`GetPlayersByExternalIDs`.

## Sin cambios
- Semántica, atomicidad y contenido del audit (`INSERT`/`UPDATE`/`DEACTIVATE`)
  y la transacción (`BeginTx`/`Commit`/`Rollback`).
- `computePlayerDiff`, `DeactivateMissingPlayers`, `UpsertPlayer`.
- Schema y API.

## Verificación

| Check | Resultado |
|---|---|
| Búsqueda de `GetPlayerByExternalID`/`getPlayerByExternalIDFn` | sin coincidencias |
| `gofmt -l` sobre los paquetes tocados | sin salida |
| `go build ./...` | pasa |
| `go vet ./internal/persistence/...` (compila tests, no los ejecuta) | pasa |
| Tests (nil-db de dao, mock de repository, integración con Docker) | no ejecutados (a cargo del usuario) |

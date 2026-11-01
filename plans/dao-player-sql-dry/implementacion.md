# Plan: centralizar columnas y scan de `players` en el DAO

Refactor pedido explícitamente: eliminar la triple duplicación de la proyección
de la tabla `players` (lista de 13 columnas del `SELECT` y `Scan` de esas
columnas a `model.Player`) en `dao/player_sql.go`.

## Modificaciones

### `backend/internal/persistence/dao/player_sql.go`
- Nueva constante `playerColumns`: única fuente de verdad de la proyección de
  `players`.
- Nueva interfaz `rowScanner` (`Scan(dest ...any) error`), satisfecha tanto por
  `*sql.Rows` como por `*sql.Row`.
- Nuevo helper `scanPlayer(rowScanner) (model.Player, error)` que escanea las 13
  columnas en el orden de `playerColumns`.
- `ListPlayers`: `dataQuery` compone el `SELECT` con `playerColumns`; el loop de
  filas usa `scanPlayer`.
- `GetPlayerByID` y `GetPlayersByExternalIDs`: usan `playerColumns` + `scanPlayer`.
  (`GetPlayersByExternalIDs` llegó después, en el refactor del N+1, y reusa el
  mismo helper.)

## Sin cambios
- Comportamiento, queries resultantes, orden de columnas y mensajes de error.
- Mapeo `sql.ErrNoRows` → `dao.ErrNotFound`.
- `model`, `repository`, `service` y `db/init.sql`.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l internal/persistence/dao` | sin salida |
| `go build ./...` | pasa |
| `go vet ./internal/persistence/...` | pasa |
| Tests de `dao` | no ejecutados (a cargo del usuario) |

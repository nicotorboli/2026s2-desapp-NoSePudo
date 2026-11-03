# Plan: interfaces definidas junto al consumidor en `service`

Reordenamiento pedido explícitamente: cada interfaz de la capa `service` se
declara en el archivo del struct que la consume. La única compartida
(`PlayerRepository`, usada por `PlayerServiceImpl` y
`PlayerStatisticsServiceImpl`) se mueve a `shared_dependencies.go`.

## Modificaciones

### `backend/internal/service/player.go`
- Agregada la interfaz `FootballDataClient` (la consume `PlayerServiceImpl`).
- Agregado el import `model` (la firma usa `model.Player`).

### `backend/internal/service/player_statistics.go`
- Agregadas `PlayerStatisticsRepository` y `WhoscoredClient` (las consume
  `PlayerStatisticsServiceImpl`). Sin cambios de imports (ya tenía `model` y
  `whoscored`).

### `backend/internal/service/dependencies.go` → `backend/internal/service/shared_dependencies.go`
- Renombrado el archivo; queda **solo** `PlayerRepository`.
- Se quitaron `PlayerStatisticsRepository`, `FootballDataClient` y
  `WhoscoredClient`.
- Se quitó el import `whoscored` de este archivo.

### `backend/internal/service/auth.go`
- Sin cambios: `UserRepository`, `PasswordHasher`, `TokenIssuer`,
  `RefreshTokenRepository` y `Clock` ya estaban inline.

## Sin cambios
- `service/container.go` (usa constructores y tipos concretos).
- Tests (los mocks implementan métodos, no referencian los tipos de interfaz).
- Las demás capas (`repository`, `controller`, `middleware`) ya declaran sus
  interfaces del lado del consumidor.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l internal/service` | sin salida |
| `go build ./...` | pasa |
| `go vet ./internal/service/...` | pasa (compila tests) |
| Tests de `service` | no ejecutados (a cargo del usuario) |

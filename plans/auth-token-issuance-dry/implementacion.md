# Plan: deduplicar la emisión de credenciales en `service/auth.go`

Refactor #1 del relevamiento. `Login` (vía `openSession`) y `Refresh` repetían
la emisión del par de credenciales y el armado de `model.RefreshToken` /
`model.Session`. Se extraen a helpers únicos.

## Modificaciones

### `backend/internal/service/auth.go`
- Nuevo tipo privado `tokenPair` (claims de acceso y renovación + tokens).
- Nuevo helper `issuePair(user, sessionID) (tokenPair, error)`: emite
  `IssueAccess` + `IssueRefresh` con los mismos dos mensajes de error que antes.
- Nuevo helper `refreshTokenRecord(user, sessionID, claims) model.RefreshToken`.
- Nuevo helper `sessionFromPair(pair) model.Session`.
- `openSession`: reemplaza la emisión y el armado de structs por `issuePair` +
  `refreshTokenRecord` (para el `Insert`) + `sessionFromPair`.
- `Refresh`: reemplaza la cola (emisión + `replacement` + `model.Session`) por
  `issuePair` + `refreshTokenRecord` (para el `Rotate`) + `sessionFromPair`. El
  arranque (GetByID + checks de expiración/reuso/revocación) no cambia.

## Sin cambios
- Comportamiento y strings de error: idénticos.
- Orden de operaciones: emitir access → emitir refresh → persistir/rotar →
  devolver sesión.
- `Login`, `Register`, `EnsureSuperuser`, `Logout`, `respondToTheft`, interfaces
  y constructores.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l internal/service` | sin salida |
| `go build ./...` | pasa |
| `go vet ./internal/service/...` (compila tests, no los ejecuta) | pasa |
| Tests de `service` | no ejecutados (a cargo del usuario) |

# Plan: adaptar los tests e2e al catálogo público

Decisión de producto: el catálogo (`GET /players`, `GET /players/{id}`,
`GET /players/{id}/statistics`) es **público** (`AccessAnonymous`), tal como ya
declaran `routes.go` y `openapi.yaml`. Los tests e2e asumían que requería
credencial (401). Se ajustaron para reflejar el comportamiento real.

## Modificaciones

### `backend/test/e2e/catalog_test.go`
- Eliminados `TestCatalogIsRefusedWithoutAUsableCredential`,
  `TestCatalogIsRefusedWithAnExpiredCredential` y
  `TestRefusedRequestsLeaveNoTrace` (el catálogo ya no rechaza).
- `TestCatalogIsServedWithAValidCredential` → `TestCatalogIsServed`: pide
  `/players` sin credencial y verifica que se sirve con los jugadores sembrados.
- Eliminado el helper `accessTokenFor` (quedó sin uso).
- Imports: quitados `strings` y `time` (sin uso tras los borrados).

### `backend/test/e2e/session_test.go`
- `TestTheTwoCredentialsAreNotInterchangeable`: quitado el check contra
  `/players`; se conserva el de `/auth/logout` (credencial de renovación en un
  endpoint que no es el suyo).
- `TestRefreshKeepsTheSessionAliveAfterTheAccessCredentialExpires`: los probes
  de autenticación usan `/auth/logout` (endpoint autenticado) en vez de
  `/players`.
- `TestRefreshIssuesANewSessionAndConsumesTheOldCredential`: quitado el check
  del catálogo; se ajustó `newAccessToken` a `_`.
- `TestSignOutInvalidatesTheRefreshCredential`: quitado el check de que la
  credencial de acceso sigue abriendo el catálogo.

### `backend/test/e2e/logging_test.go`
- `runCompleteFlow`: el rechazo de una credencial inválida se dispara con
  `POST /auth/logout` en vez de `/players`, para que se registre
  `authentication refused`.
- `TestRefusalsWithNoIdentifiableAccountAreStillLogged`: la petición sin
  credencial se hace contra `/auth/logout` (endpoint autenticado).

## Sin cambios
- `routes.go`, `openapi.yaml` y código de producción: el catálogo ya era
  anónimo.
- `TestCredentialIsCheckedBeforeTheBody` y `TestAnonymousEndpointsNeedNoCredential`.

## Verificación

| Check | Resultado |
|---|---|
| `grep` de `accessTokenFor` / tests borrados | sin coincidencias |
| `gofmt -l test/e2e` | sin salida |
| `go vet ./test/e2e/...` (compila tests, no los ejecuta) | pasa |
| Tests e2e (testcontainers) | a cargo del usuario |

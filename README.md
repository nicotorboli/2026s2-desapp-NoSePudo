# NoSePudo

Monorepo con el backend en Go (`backend/`) y el frontend en React + TypeScript +
Vite (`frontend/`).

`.specify/memory/constitution.md` es el documento normativo del proyecto y la
fuente de verdad de las convenciones: capas, inyección de dependencias, DTOs,
testing y linters. Conviene leerlo antes de escribir código, porque hay partes
del repositorio que todavía no las cumplen.

## Levantar el backend

La base corre en Docker y el servidor desde el host.

```bash
cd backend
docker compose up -d          # Postgres en el puerto 5433
go run ./cmd/server
```

El servidor se niega a arrancar si le falta el secreto de firma o si a la base
le falta alguna tabla, y en los dos casos dice qué hacer. Ver
[Configuración](#configuración).

## Configuración

Todo entra por variables de entorno. Las tres primeras ya existían; el resto las
trajo la autenticación.

| Variable | ¿Obligatoria? | Default | Para qué |
|---|---|---|---|
| `NSPPSQLDS` | sí | — | Cadena de conexión a Postgres |
| `NSPHOST` | no | `127.0.0.1` | Interfaz donde escucha el servidor |
| `NSPPORT` | no | `8080` | Puerto donde escucha el servidor |
| `NSP_JWT_SECRET` | **sí** | — | Clave de firma HS256, **mínimo 32 bytes** |
| `NSP_ACCESS_TTL` | no | `15m` | Vida de la credencial de acceso |
| `NSP_REFRESH_TTL` | no | `168h` | Vida de la credencial de renovación (7 días) |
| `NSP_BCRYPT_COST` | no | `12` | Factor de costo de bcrypt (entre 4 y 31) |
| `NSP_SUPERUSER_EMAIL` | no | — | Superusuario, aprovisionado al arrancar |
| `NSP_SUPERUSER_PASSWORD` | no | — | Su contraseña |

Un ejemplo para desarrollo:

```bash
export NSPPSQLDS="postgres://devuser:devpassword@localhost:5433/nsp_db?sslmode=disable"
export NSP_JWT_SECRET="$(openssl rand -base64 48)"
export NSP_SUPERUSER_EMAIL=admin@nosepudo.ar
export NSP_SUPERUSER_PASSWORD='cambiar-esto-en-serio'
```

Dos cosas que conviene saber de antemano:

- **Sin `NSP_JWT_SECRET`, o con uno de menos de 32 bytes, el servidor no
  arranca.** Es a propósito: emitir credenciales que después nadie puede
  verificar es peor que no arrancar.
- **El superusuario se aprovisiona de forma idempotente.** Si la cuenta ya
  existe no se toca, ni se le cambia la contraseña, así que un reinicio no le
  deshace una que el dueño haya cambiado. Si las dos variables faltan, el paso
  se saltea sin ruido.

### Si cambió el esquema de la base

`db/init.sql` sólo se ejecuta cuando el volumen de Postgres está vacío, así que
editarlo no cambia nada hasta recrear el volumen. El servidor lo detecta al
arrancar, nombra las tablas que faltan y pide esto:

```bash
cd backend
docker compose down -v && docker compose up -d
```

## La API

[`backend/api/openapi.yaml`](backend/api/openapi.yaml) es el documento
mantenido. Cada operación declara su nivel de acceso en `x-access-level`, y hay
un test que compara esas declaraciones contra la tabla de rutas: un endpoint que
se agregue sin documentar, o documentado con un nivel que el código no aplica,
rompe el build.

## Verificación

Backend, desde `backend/`:

```bash
go build ./...
go test -race ./...
```

Los tests de repositorio y los e2e levantan un Postgres real con
testcontainers, así que **necesitan Docker corriendo**. Sin Docker no fallan:
se saltean, y el paquete informa `ok` sin haber probado nada.

`go test -race` necesita cgo, y por lo tanto un compilador de C. En Windows sin
uno instalado hay que correr los tests sin `-race`; el CI los corre con `-race`
en Linux.

La verificación completa del backend, idéntica a la del CI, es
`./scripts/pre-commit.sh` desde la raíz: gofmt, `go vet` y golangci-lint en la
versión exacta que declara `.golangci-version`.

Frontend, desde `frontend/`:

```bash
npm run typecheck
npm run lint
npm run knip
npm run test
npm run dev
```

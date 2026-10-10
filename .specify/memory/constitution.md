
# NoSePudo Constitution

## Core Principles

### I. Backend en Go con net/http

- Implementar el backend en Go exponiendo la API REST exclusivamente mediante `net/http` de la librería estándar.
- Prohibir terminantemente el uso de frameworks web de terceros (e.g., Gin, Echo, Chi, Fiber).
- Proveer soporte de ejecución dual obligatorio: ejecución local y contenedor Docker (`Dockerfile`).
- Centralizar el código del backend bajo el módulo `backend/`, siendo su `go.mod` la única fuente de verdad sobre versiones de dependencias.

### II. Persistencia en PostgreSQL

- Emplear PostgreSQL como el único motor de base de datos relacional del sistema.
- Conectar y operar exclusivamente mediante la librería estándar `database/sql`. Prohibir el uso de ORMs (e.g., GORM, Ent).

### III. Arquitectura en capas

Organizar el backend bajo `cmd/` e `internal/` con responsabilidades delimitadas y flujo unidireccional:

- `cmd/`: Punto de entrada (`main.go`), responsable único de instanciar implementaciones concretas, ensamblar el grafo de dependencias y arrancar el servidor HTTP.
- `internal/server`: Inicialización del servidor HTTP (`NewServer`) y registro de rutas (`routes`).
- `internal/middleware`: Interceptores HTTP para lógica transversal (autenticación, autorización, tracing y métricas).
- `internal/httphandler`: Utilidades HTTP comunes (codificación y decodificación JSON con `Encode`/`Decode` y wrappers de endpoints).
- `internal/logger`: Inicialización y configuración centralizada de logging (`log/slog` con handler de Zerolog).
- `internal/controller`: Manejo de endpoints HTTP, validación inicial de DTOs y mapeo de respuestas delegando en `service`.
- `internal/service`: Orquestación de casos de uso y lógica de negocio. Interactúa con `model`, `repository` y `adapters`. Prohibido acceder directamente a DAOs o base de datos.
- `internal/model`: Entidades y reglas puras de dominio. Prohibido contener dependencias de infraestructura o tags de serialización.
- `internal/persistence`: Acceso a datos subdividido en:
  - `repository`: Abstracción por entidad/concepto de dominio (retorna y recibe modelos). Coordina múltiples DAOs si una operación abarca varias tablas.
  - `dao`: Acceso granular a nivel SQL por tabla. Son públicos para permitir su inyección desde `cmd/`.
- `internal/adapters`: Integración con dependencias externas (servicios de red, APIs de terceros, librerías complejas).
- `internal/dto`: Estructuras de datos dedicadas para transferencia entre límites del sistema y capas.
- **Propagación de Contexto**: Propagar obligatoriamente `context.Context` como primer parámetro en toda función o método que realice I/O (controller, service, repository, dao, adapters) para garantizar cancelación oportuna, timeouts y trazabilidad.

### IV. Inyección de dependencias

- Recibir todas las dependencias explícitamente vía constructor (`New...`). Prohibir la auto-instanciación dentro de tipos o el acceso a estado global.
- Depender de interfaces y nunca de implementaciones concretas: los `services` consumen interfaces de `repository` y `adapters`; los `repositories` reciben DAOs (o sus interfaces) vía constructor.
- Declarar interfaces junto a su consumidor o abstracción correspondiente para desacoplar implementaciones y habilitar testing.
- Ensamblar el grafo de dependencias e instanciar implementaciones concretas exclusivamente en `cmd/`.

### V. DTOs en el borde

- Prohibir la exposición de modelos de dominio en el borde HTTP. Los structs de `model` no deben contener tags de serialización (`json`, `xml`).
- Diseñar DTOs independientes y desacoplados para request y response en cada operación; prohibido reutilizar DTOs añadiendo campos opcionales para contextos disímiles.
- Implementar la conversión bidireccional exclusivamente en el paquete `dto` mediante métodos explícitos:
  - `DesdeModelo(m Modelo) DTO`: construye el DTO a partir del modelo.
  - `AModelo(deps...) Modelo`: construye el modelo recibiendo sus colaboradores externos ya resueltos (prohibido consultar persistencia desde el DTO).
- Prohibir conversiones manuales ad-hoc en los controllers.
- Canalizar toda codificación y decodificación JSON HTTP mediante `internal/httphandler`.

### VI. Frontend React + TypeScript

- Implementar la interfaz en `frontend/` utilizando React con TypeScript.
- Encapsular todas las llamadas HTTP en un módulo de abstracción dedicado utilizando exclusivamente `axios`:
  - Mantener llamadas de axios y métodos HTTP privados dentro del módulo.
  - Exportar una función tipada por cada endpoint consumido. Prohibir importar `axios` fuera de este módulo.
  - Espejar estrictamente en los tipos TypeScript los DTOs de la API REST, nunca modelos internos de backend.
- Organizar el proyecto modularmente en carpetas de páginas y componentes.
- Estilizar mediante archivos CSS por página/componente utilizando nombres de clases bajo convención BEM.

### VII. API REST documentada con OpenAPI

- Mantener una especificación OpenAPI (v3+) como contrato obligatorio y fuente de verdad de la API REST.
- Sincronizar el artefacto OpenAPI en el mismo commit ante cualquier cambio, agregado o remoción de endpoints, DTOs o códigos de estado.
- Validar la conformidad de los controllers frente a la especificación OpenAPI (rutas, esquemas y códigos de error).

### VIII. Seguridad: autenticación y autorización JWT

- Autenticar mediante tokens JWT firmados exclusivamente con algoritmo HS256 y fecha de expiración obligatoria.
- Almacenar contraseñas protegidas mediante hashing con `bcrypt`.
- Implementar control de acceso y autenticación mediante `internal/middleware`:
  - Extraer e inyectar el ID de usuario y nivel de privilegios (usuario común vs. superusuario) en el `context.Context`.
  - Rechazar inmediatamente solicitudes con tokens inválidos, expirados o ausentes en rutas protegidas.
  - Exigir nivel de acceso declarado explícitamente en cada endpoint; la ausencia de declaración es un error, nunca un endpoint público.
- Validar todas las entradas de datos en la frontera (`controller`/`middleware`), descartando datos malformados antes de ejecutar lógica de negocio.
- Resolver controles de autorización basados en datos de dominio en `internal/service`: contrastar el ID del `context` contra el recurso requerido en la URL o body.

### IX. Observabilidad

- Emitir logs estructurados en JSON (un evento por línea) vía `log/slog` con handler de Zerolog centralizado en `internal/logger`.
- Incluir en cada log los campos estándar mínimos: `timestamp` (UTC), `level`, `correlation_id`, `operation` y `actor` (en operaciones autenticadas).
- Generar un `correlation_id` en el borde HTTP si el cliente no lo envía, propagarlo por `context.Context` e incluirlo en todos los eventos (incluyendo adapters).
- Registrar de forma obligatoria:
  - Peticiones HTTP: método, ruta, status code y latencia.
  - Errores: mensaje, causa raíz y contexto de falla.
  - Integraciones externas: operación, duración y resultado.
- Prohibir terminantemente el registro de credenciales, contraseñas, tokens JWT o datos personales (PII).
- Exponer un endpoint de health check y métricas operativas de latencia y tasa de error.

### X. Auditoría transaccional

- Registrar todo cambio de estado relevante en un log/tabla de auditoría inmutable (exclusivamente operaciones append-only; prohibidos updates y deletes).
- Registrar por cada evento: identificador del autor/actor, timestamp GMT-3, entidad afectada, tipo de operación y diferencia exacta (diff/payload) entre estado anterior y posterior.

### XI. Integridad de datos

- Ejecutar toda operación de persistencia dentro de una transacción (`db.BeginTx`) por defecto; omitir una transacción requiere justificación y documentación explícita.
- Definir índices sobre todas las claves foráneas, columnas de búsqueda y filtros frecuentes.
- Forzar integridad referencial mediante constraints de clave primaria, unicidad y foreign keys en base de datos.

### XII. Resiliencia: caché y scheduler

- Intermediar lecturas a datos y servicios externos mediante una capa de caché con TTL.
- Proveer degradación elegante: responder con datos locales o cacheados cuando la fuente externa no esté disponible.
- Ejecutar procesos en lote (batch) y sincronizaciones periódicas únicamente mediante un scheduler automatizado, prohibiendo ejecuciones manuales ad-hoc.

### XIII. Testing

- Cubrir cada funcionalidad con tests unitarios, de integración y end-to-end (e2e).
- Aislar el alcance de pruebas según la capa:
  - `model`: tests unitarios puros en memoria, sin base de datos ni dependencias de I/O.
  - `service`: tests unitarios con mocks de interfaces de `repository` y `adapters`.
  - `repository`: tests de integración contra una instancia real de PostgreSQL orquestada con `testcontainers`.
  - `adapters`: tests contra stubs/fixtures grabados para servicios externos (sin llamadas reales a la red); inyección de reloj y fuentes de aleatoriedad para pruebas deterministas.
  - `controller`: tests de integración HTTP validando status codes, payloads y contrato OpenAPI.
- Diseñar casos de prueba mediante técnicas formales: clases de equivalencia (casos válidos e inválidos), análisis de valores límite (anterior, exacto y posterior) y tablas de decisión ante lógica multicondicional.
- Automatizar los escenarios requeridos por el enunciado como suites e2e ejecutables en CI.

### XIV. Semánticas

- Nombrar el directorio de cada especificación (`.specify/specs/<branch-name>`) de forma idéntica al nombre de la branch activa de Git.
- Prohibir desarrollar o modificar especificaciones cuyo identificador no coincida unívocamente con la rama activa.
- Garantizar la trazabilidad 1:1 entre rama de Git, especificación, tareas (`tasks.md`), commits y Pull Requests.

## Technical Constraints

- **Backend**: Go (versión declarada en `backend/go.mod`). Uso exclusivo de `net/http` y `database/sql` de la librería estándar.
- **Base de datos**: PostgreSQL ejecutado en contenedor Docker aislado en desarrollo, CI y test.
- **Docker**: `Dockerfile` en `backend/` y `docker-compose.yml` en la raíz para orquestación de servicios locales y testing.
- **Seguridad**: `github.com/golang-jwt/jwt/v5` para manejo de tokens JWT y `golang.org/x/crypto/bcrypt` para hashing de contraseñas.
- **Logging**: `log/slog` estándar configurado con handler de Zerolog (`github.com/rs/zerolog`).
- **Frontend**: TypeScript y React en `frontend/`. `axios` restringido al módulo de abstracción HTTP. Estilos modulares con convención BEM.
- **Testing**: `testing` estándar de Go y `testcontainers-go` para integración de PostgreSQL.

## Code Quality & Linting

El backend Go se valida con `golangci-lint` en la versión fijada en `.golangci-version` (consistente entre local y CI). Pull Requests con fallas de linter tienen el merge bloqueado.

- **Configuración central**: `backend/.golangci.yml` es la única fuente de verdad.
- **Linters mandatorios**:
  - `bodyclose`: Cierre obligatorio de response bodies HTTP.
  - `gocritic`: Detección de patrones de código sospechosos.
  - `gocyclo`: Complejidad ciclomática máxima de 15 por función.
  - `gosec`: Detección de vulnerabilidades de seguridad (excluyendo G104, delegado en `errcheck`).
  - `nilerr`: Prohibición de retornar `nil` tras comprobar un error no nulo.
  - `noctx`: Peticiones HTTP obligadas a incluir contexto explícito.
  - `rowserrcheck`: Verificación de `Err()` en result sets de base de datos.
  - `sqlclosecheck`: Cierre obligatorio de statements y rows SQL.
  - `unconvert`: Eliminación de conversiones de tipo redundantes.
  - `govet`: Ejecución con `enable-all: true` (incluyendo `fieldalignment` para orden de structs por tamaño de tipos).
  - Linters por defecto: `errcheck`, `ineffassign`, `staticcheck`, `unused`.
- **Verificación local**:
  ```bash
  cd backend
  golangci-lint run
  ```
- **Git Hook de pre-commit**:
  - Entrypoint en `.githooks/pre-commit` ejecutando `scripts/pre-commit.sh`.
  - Configuración inicial por clon:
    ```bash
    git config core.hooksPath .githooks
    ```
  - Instalación de la versión requerida:
    ```bash
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(cat .golangci-version)
    ```

## Governance

Esta constitución es el documento normativo supremo del proyecto. Toda práctica o implementación que la contradiga debe corregirse de inmediato.

- **Procedimiento de enmienda**:
  - Requiere documentación explícita de la motivación y alcance en el commit y descripción del PR.
  - Aplica versionado semántico según el tipo de cambio:
    - **MAJOR**: Remoción, redefinición o cambios incompatibles en los principios.
    - **MINOR**: Incorporación de un nuevo principio o ampliación material de directivas.
    - **PATCH**: Correcciones, reducción de prosa, eliminación de redundancias o refinamientos no semánticos.
- **Cumplimiento**: Todo Pull Request y code review debe validar conformidad estricta con esta constitución; la introducción de complejidad no prevista debe justificarse formalmente.

**Version**: 2.1.1 | **Ratified**: 2026-09-05 | **Last Amended**: 2026-10-09

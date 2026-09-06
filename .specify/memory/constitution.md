<!--
Sync Impact Report
==================
Version change: (scaffold sin versionar) → 1.0.0
Bump rationale: Ratificación inicial de la constitución; primer documento
normativo del proyecto que reemplaza todos los placeholders del scaffold.
Principios modificados: Ninguno (no existía constitución previa).
Secciones añadidas: Core Principles (12), Technical Constraints,
Security, Observability & Data Integrity, Governance.
Secciones eliminadas: Ninguna.
TODOs pendientes: Ninguno.
-->

# NoSePudo Constitution

## Core Principles

### I. Backend en Go con net/http

El backend se implementa en Go y expone la API REST únicamente con la librería
estándar `net/http`. Se prohíbe el uso de frameworks web externos; toda
funcionalidad HTTP debe construirse sobre la librería estándar.

### II. Persistencia en PostgreSQL

PostgreSQL es la única base de datos del sistema y se conecta mediante la
librería estándar `database/sql`, sin ORM. La base de datos debe estar
dockerizada en todos los entornos (desarrollo, CI y test).

### III. Arquitectura en capas

El código se organiza en `cmd` (main que levanta el servidor) e `internal`,
separado en capas:

- controller: sirve los endpoints REST y traduce los datos externos al
  servicio y los resultados del servicio al cliente.
- service: orquesta el modelo de negocio con la persistencia y los adapters.
- repository: capa de persistencia que se comunica con la base de datos y no
  tiene conocimiento del modelo de negocio.
- model (dominio): contiene la lógica del negocio.
- adapters: integración con servicios externos.

### IV. Frontend React + TypeScript

El frontend se implementa en React con TypeScript y se comunica con el backend
por HTTP/REST usando axios como único cliente HTTP. Toda comunicación pasa por
un módulo de abstracción: los métodos HTTP se representan como funciones
privadas y se exporta una función por cada endpoint consumido, desacoplando la
comunicación del cliente HTTP. El proyecto se organiza por páginas y
componentes; los estilos se manejan con archivos CSS por página/componente,
usando clases y siguiendo el arquetipo BEM.

### V. API REST documentada con OpenAPI

La API REST se documenta con el estándar OpenAPI. La especificación es un
artefacto obligatorio y debe mantenerse sincronizada con la implementación en
cada cambio de endpoints.

### VI. Seguridad: autenticación y autorización JWT

La autenticación y autorización se implementan con JWT, diferenciando los
privilegios entre usuario común y superusuario al gestionar el acceso a los
recursos. Todas las entradas de datos deben validarse; las entradas inválidas
se rechazan sin ejecutar lógica de negocio.

### VII. Observabilidad

Toda operación relevante emite logs estructurados de análisis y propaga un
correlation ID para trazabilidad de principio a fin. El sistema expone un
health check y métricas de latencia y tasa de error para monitoreo continuo.

### VIII. Auditoría transaccional

Se mantiene un registro de auditoría de las transacciones que incluye autor,
timestamp, cambios realizados y la diferencia entre el estado anterior y el
posterior. Todo cambio de estado relevante debe quedar auditado.

### IX. Integridad de datos

Todas las operaciones de base de datos son transaccionales por defecto; omitir
una transacción solo se permite cuando se indica y justifica explícitamente.
El esquema se optimiza con indexación sobre las claves de búsqueda y consultas
frecuentes.

### X. Resiliencia: caché y scheduler

Se implementa una caché para consultas frecuentes hacia la API externa, con
funcionamiento local ante indisponibilidad de la misma, mitigando fallas y
latencia. Un scheduler ejecuta procesos batch automáticos para la
actualización de datos de jugadores y la revalorización.

### XI. Backoffice de administración

Existe un backoffice desde el cual se disparan trabajos automatizados y se
modifican las reglas de evaluación de la aplicación. El acceso al backoffice
es exclusivo de superusuarios.

### XII. Testing

Toda feature incluye test unitarios, de integración y e2e. Los tests que
requieren base de datos usan testcontainers para no alterar la base de datos
real.

## Technical Constraints

- Backend: Go con `net/http` y `database/sql` de la librería estándar; el
  módulo vive en `backend/` y su `go.mod` es fuente de verdad de la versión.
- Capas: el repository no conoce el modelo; el service es el único orquestador
  de modelo, persistencia y adapters; el controller solo traduce datos.
- Frontend: vive en `frontend/`, organizado por páginas y componentes, con CSS
  por página/componente siguiendo BEM; axios se usa únicamente dentro del
  módulo de abstracción HTTP.
- Base de datos: PostgreSQL dockerizada de forma obligatoria.

## Security, Observability & Data Integrity

- Privilegios: usuario común vs superusuario vía JWT; el backoffice, el
  disparo de jobs y la modificación de reglas de evaluación requieren
  superusuario.
- Validación de entradas: obligatoria en todos los endpoints.
- Observabilidad: logs estructurados, correlation ID, health check y métricas
  de latencia y tasa de error.
- Auditoría: registro con autor, timestamp, cambios y diff entre estado
  anterior y posterior.
- Datos: operaciones transaccionales por defecto e indexación del esquema.

## Governance

Esta constitución es el documento normativo supremo del proyecto; cualquier
práctica o implementación que la contradiga debe corregirse.

- Enmiendas: requieren documentación del cambio, justificación y ajuste de
  versión según semver (MAJOR: remoción o redefinición de principios; MINOR:
  nuevo principio o expansión material; PATCH: clarificaciones y correcciones
  no semánticas).
- Cumplimiento: toda PR o revisión verifica la conformidad con esta
  constitución; la complejidad adicional debe justificarse.

**Version**: 1.0.0 | **Ratified**: 2026-09-05 | **Last Amended**: 2026-09-05

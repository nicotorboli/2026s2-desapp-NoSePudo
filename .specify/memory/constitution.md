<!--
Sync Impact Report
==================
Version change: 1.0.0 → 1.1.0
Bump rationale: MINOR. Se agrega un principio nuevo y se expande
materialmente otro; no se remueve ni se redefine ningún principio previo. La
renumeración en romanos es consecuencia de la inserción.
Principios añadidos: IV. DTOs en el borde.
Principios modificados: VIII. Observabilidad (ex VII) — se fija zerolog como
única librería de logging, el formato JSON y los campos mínimos obligatorios,
el ciclo del correlation ID, los eventos de registro obligatorio y qué no se
registra. Se elimina la expresión "logs de análisis", que se leía como una
traza del análisis de negocio y no como lo que pide el TP: logs estructurados
que faciliten el análisis de los propios logs.
V. Frontend React + TypeScript (ex IV) — los tipos TypeScript del módulo de
abstracción HTTP espejan los DTOs de la API, no el modelo de dominio.
IX. Auditoría transaccional (ex VIII) — el registro es inmutable: las filas se
agregan y nunca se modifican ni se borran.
XIII. Testing (ex XII) — se explicita qué se testea y qué se mockea en cada
capa, los casos obligatorios por caso de uso y las técnicas de diseño de
casos; los escenarios de prueba del enunciado pasan a ser tests e2e.
Principios renumerados sin cambio de contenido: VI a XII (ex V a XI).
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

### IV. DTOs en el borde

Los tipos que cruzan el borde HTTP son DTOs y nunca son el modelo de dominio,
aunque los campos coincidan. El modelo de dominio no lleva tags de
serialización. Request y respuesta son DTOs distintos, y un mismo modelo puede
tener varios DTOs según el contexto en que se lo devuelve.

La conversión es explícita y vive en el DTO: `DesdeModelo` construye el DTO a
partir del modelo y `AModelo` hace el camino inverso. `AModelo` recibe por
parámetro los colaboradores ya resueltos, de modo que el DTO nunca consulta la
persistencia. No se arman conversiones a mano en el controller.

### V. Frontend React + TypeScript

El frontend se implementa en React con TypeScript y se comunica con el backend
por HTTP/REST usando axios como único cliente HTTP. Toda comunicación pasa por
un módulo de abstracción: los métodos HTTP se representan como funciones
privadas y se exporta una función por cada endpoint consumido, desacoplando la
comunicación del cliente HTTP. Los tipos TypeScript de ese módulo espejan los
DTOs de la API, no el modelo de dominio del backend. El proyecto se organiza
por páginas y componentes; los estilos se manejan con archivos CSS por
página/componente, usando clases y siguiendo el arquetipo BEM.

### VI. API REST documentada con OpenAPI

La API REST se documenta con el estándar OpenAPI. La especificación es un
artefacto obligatorio y debe mantenerse sincronizada con la implementación en
cada cambio de endpoints.

### VII. Seguridad: autenticación y autorización JWT

La autenticación y autorización se implementan con JWT, diferenciando los
privilegios entre usuario común y superusuario al gestionar el acceso a los
recursos. Todas las entradas de datos deben validarse; las entradas inválidas
se rechazan sin ejecutar lógica de negocio.

### VIII. Observabilidad

Los logs se emiten con zerolog como única librería de logging, son
estructurados en JSON, un evento por línea, y todo evento incluye como mínimo
timestamp, nivel, correlation ID, operación y, en las operaciones
autenticadas, el actor. Los nombres de los campos son comunes a todo el
sistema: un log que no se puede filtrar no facilita ningún análisis.

El correlation ID se genera en el borde HTTP cuando el cliente no lo provee,
viaja por el `context` y aparece en todos los eventos de la operación,
incluidos los de los adapters, de modo que filtrar por un ID muestre la
solicitud completa de principio a fin.

Se registran obligatoriamente cada request con su latencia y su status, todo
error con su causa, y toda llamada a un servicio externo con su resultado y su
duración. No se registran credenciales, tokens ni datos personales. El sistema
expone un health check y métricas de latencia y tasa de error.

### IX. Auditoría transaccional

Se mantiene un registro de auditoría de las transacciones que incluye autor,
timestamp, cambios realizados y la diferencia entre el estado anterior y el
posterior. El registro es inmutable: las filas se agregan y nunca se
modifican ni se borran. Todo cambio de estado relevante debe quedar auditado.

### X. Integridad de datos

Todas las operaciones de base de datos son transaccionales por defecto; omitir
una transacción solo se permite cuando se indica y justifica explícitamente.
El esquema se optimiza con indexación sobre las claves de búsqueda y consultas
frecuentes.

### XI. Resiliencia: caché y scheduler

Se implementa una caché para consultas frecuentes hacia la API externa, con
funcionamiento local ante indisponibilidad de la misma, mitigando fallas y
latencia. Un scheduler ejecuta procesos batch automáticos para la
actualización de datos de jugadores y la revalorización.

### XII. Backoffice de administración

Existe un backoffice desde el cual se disparan trabajos automatizados y se
modifican las reglas de evaluación de la aplicación. El acceso al backoffice
es exclusivo de superusuarios.

### XIII. Testing

Toda feature incluye tests unitarios, de integración y e2e, y cada capa define
qué se mockea:

- model (dominio): sin base de datos ni red; las estrategias de valuación se
  testean como funciones que reciben métricas y devuelven un score.
- service: con el repository y los adapters mockeados.
- repository: contra una base real levantada con testcontainers, para no
  alterar la base de datos real.
- adapters: contra respuestas guardadas; ningún test consulta el sitio o la
  API externa.
- controller: contra el contrato OpenAPI, incluidos los códigos de error.

Cada caso de uso se cubre con su camino feliz, sus casos negativos y sus casos
borde. Los casos no se eligen por intuición: se usan clases de equivalencia,
un caso por clase de entrada; valores límite, probando el valor anterior, el
exacto y el posterior en todo umbral; y tabla de decisión cuando una regla
combina dos o más condiciones.

Los escenarios de prueba exigidos por el enunciado se implementan como tests
e2e, no como una demostración manual.

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

**Version**: 1.1.0 | **Ratified**: 2026-09-05 | **Last Amended**: 2026-09-06

# NoSePudo Constitution

## Core Principles

### I. Backend en Go con net/http

El backend se implementa en Go y expone la API REST únicamente con la librería
estándar `net/http`. Se prohíbe el uso de frameworks web externos; toda
funcionalidad HTTP debe construirse sobre la librería estándar. Se debe incluir
una versión dockerizada de la aplicación junto a la ejecución local.

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
- repository: capa de persistencia; traduce entre la base de datos y el
  modelo de dominio, y no conoce reglas de negocio ni HTTP.
- model (dominio): contiene la lógica del negocio.
- adapters: integración con servicios externos.

Hay un repository por concepto del dominio —jugador, cotización, usuario,
orden—, no uno por tabla, y lo que cruza su borde son modelos, nunca filas.
Adentro delega en DAOs, uno por tabla, que ejecutan el SQL y viven en el
paquete del repository sin exportarse. Si una operación abarca varias tablas
las coordina el repository; el service nunca usa un DAO.

### IV. Inyección de dependencias

Ninguna capa construye sus propias dependencias: las recibe por constructor.
Cada repository y cada adapter expone una interfaz y el service depende de
ella, nunca del tipo concreto. La interfaz se declara junto a su
implementación, en el paquete de su capa.

El único lugar donde se instancian implementaciones concretas y se arma el
grafo de dependencias es `cmd`. Sin eso el service no se puede testear con el
repository mockeado, como exige el principio de Testing.

### V. DTOs en el borde

Los tipos que cruzan el borde HTTP son DTOs y nunca son el modelo de dominio,
aunque los campos coincidan. El modelo de dominio no lleva tags de
serialización. Request y respuesta son DTOs distintos, y cada contexto en que
se devuelve un modelo tiene su propio DTO: no se reusa uno agregándole campos
opcionales.

La conversión es explícita y vive en el DTO: `DesdeModelo` construye el DTO a
partir del modelo y `AModelo` hace el camino inverso. El DTO nunca consulta la
persistencia: `AModelo` recibe por parámetro los colaboradores ya resueltos.
No se arman conversiones a mano en el controller.

### VI. Frontend React + TypeScript

El frontend se implementa en React con TypeScript y se comunica con el backend
por HTTP/REST usando axios como único cliente HTTP. Toda comunicación pasa por
un módulo de abstracción: los métodos HTTP se representan como funciones
privadas y se exporta una función por cada endpoint consumido; fuera de ese
módulo nadie importa axios. Los tipos TypeScript de ese módulo espejan los
DTOs de la API, no el modelo de dominio del backend. El proyecto se organiza
por páginas y componentes; los estilos se manejan con archivos CSS por
página/componente, usando clases y siguiendo el arquetipo BEM.

### VII. API REST documentada con OpenAPI

La API REST se documenta con el estándar OpenAPI. La especificación es un
artefacto obligatorio y debe mantenerse sincronizada con la implementación en
cada cambio de endpoints.

### VIII. Seguridad: autenticación y autorización JWT

La autenticación y autorización se implementan con JWT, diferenciando los
privilegios entre usuario común y superusuario al gestionar el acceso a los
recursos. Todas las entradas de datos deben validarse; las entradas inválidas
se rechazan sin ejecutar lógica de negocio.

### IX. Observabilidad

Los logs se emiten con zerolog como única librería de logging, son
estructurados en JSON, un evento por línea, y todo evento incluye como mínimo
timestamp, nivel, correlation ID, operación y, en las operaciones
autenticadas, el actor. Los nombres de los campos son comunes a todo el
sistema: un log que no se puede filtrar no facilita ningún análisis.

El correlation ID se genera en el borde HTTP cuando el cliente no lo provee,
viaja por el `context` y aparece en todos los eventos de la operación,
incluidos los de los adapters: filtrar por un ID devuelve la solicitud
completa de principio a fin.

Se registran obligatoriamente cada request con su latencia y su status, todo
error con su causa, y toda llamada a un servicio externo con su resultado y su
duración. No se registran credenciales, tokens ni datos personales. El sistema
expone un health check y métricas de latencia y tasa de error.

### X. Auditoría transaccional

Se mantiene un registro de auditoría de las transacciones que incluye autor,
timestamp, cambios realizados y la diferencia entre el estado anterior y el
posterior. El registro es inmutable: las filas se agregan y nunca se
modifican ni se borran. Todo cambio de estado relevante debe quedar auditado.

### XI. Integridad de datos

Todas las operaciones de base de datos son transaccionales por defecto; omitir
una transacción solo se permite cuando se indica y justifica explícitamente.
El esquema se optimiza con indexación sobre las claves de búsqueda y consultas
frecuentes.

### XII. Resiliencia: caché y scheduler

Toda lectura de datos externos pasa por una caché, y el sistema responde con
datos locales cuando la fuente externa no está disponible. Los procesos batch
se ejecutan desde un scheduler, nunca a mano.

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

### XIV. Semánticas

Toda especificación sobre la que se trabaje debe llevar exactamente el nombre
de la branch de GitHub sobre la cual se está programando. Queda prohibido
desarrollar o modificar una especificación cuyo identificador o directorio no
coincida con el nombre de la rama activa en el repositorio. Esta
correspondencia unívoca asegura la trazabilidad estricta entre la
especificación de requisitos, las tareas asociadas, los commits y las
revisiones en los Pull Requests.

## Technical Constraints

- Backend: Go con `net/http` y `database/sql` de la librería estándar; el
  módulo vive en `backend/` y su `go.mod` es fuente de verdad de la versión.
  Se debe incluir una versión de la aplicación dockerizada para permitir tanto
  la ejecución local como por medio de un contenedor.
- Capas: el repository traduce entre la base de datos y el modelo delegando
  en DAOs; el service es el único orquestador de modelo, persistencia y
  adapters y depende de interfaces, no de implementaciones; el controller
  solo traduce datos; el grafo de dependencias se arma en `cmd`.
- Frontend: vive en `frontend/`, organizado por páginas y componentes, con CSS
  por página/componente siguiendo BEM; axios se usa únicamente dentro del
  módulo de abstracción HTTP.
- Base de datos: PostgreSQL dockerizada de forma obligatoria y separada del contenedor de backend.

## Security, Observability & Data Integrity

- Privilegios: usuario común vs superusuario vía JWT; disparar un job a mano
  y modificar las reglas de valuación requieren superusuario.
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
  no semánticas). La documentación de cada enmienda se escribe en el mensaje
  de su commit, y el PR la repite para quien revisa. Este archivo no lleva
  historial de cambios ni Sync Impact Report: describe únicamente lo que rige
  hoy.
- Cumplimiento: toda PR o revisión verifica la conformidad con esta
  constitución; la complejidad adicional debe justificarse.

**Version**: 1.3.0 | **Ratified**: 2026-09-05 | **Last Amended**: 2026-09-15

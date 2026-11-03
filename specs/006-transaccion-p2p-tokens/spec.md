# Feature Specification: Transacción P2P de Tokens (Invitación Directa)

**Feature Branch**: `ft/p2p-token-session`

**Created**: 2026-11-03

**Status**: Draft

**Input**: User description: "poder generar una compra y venta entre 2 usuarios. Yo como vendedor puedo iniciar una transacción personalizada, poner la cantidad de tokens a vender y el precio, generar un link. Ese link lo abre otro usuario y en esa vista se ven los 2 usuarios y tienen el botón de aceptar o cancelar. La pantalla tiene un tiempo límite de 5 minutos y se cancela automáticamente, y se genera la transacción."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Crear Oferta de Venta P2P (Priority: P1)

Como usuario autenticado que posee tokens de un jugador, quiero crear una oferta de venta directa especificando la cantidad de tokens y el precio, y obtener un enlace único para compartir con el comprador, de modo que pueda acordar operaciones fuera del mercado abierto con personas concretas.

**Why this priority**: Es la puerta de entrada de toda la funcionalidad P2P. Sin poder crear la oferta no existe ningún flujo posterior.

**Independent Test**: Iniciar sesión con un usuario que posea tokens, crear una oferta P2P por 5 tokens a 3 créditos cada uno y verificar que el sistema responde con un identificador de sesión único, una URL compartible y que los 5 tokens quedan reservados (no disponibles para otras operaciones) en el balance del vendedor.

**Acceptance Scenarios**:

1. **Given** un usuario autenticado con 10 tokens del jugador X, **When** crea una oferta P2P por 5 tokens a precio 3 créditos, **Then** el sistema genera un identificador único de sesión, devuelve la URL de invitación, reserva los 5 tokens del vendedor y la sesión queda en estado `PENDING` con expiración a 5 minutos.
2. **Given** un usuario autenticado, **When** intenta crear una oferta P2P por más tokens de los que posee, **Then** el sistema rechaza la solicitud con error de fondos insuficientes, no se reserva nada y no se genera enlace.
3. **Given** un usuario autenticado, **When** intenta crear una oferta con cantidad ≤ 0 o precio ≤ 0, **Then** el sistema rechaza la solicitud con error de validación (HTTP 400) y no se genera enlace.
4. **Given** un usuario autenticado, **When** crea una oferta P2P válida, **Then** el vendedor puede ver su propia oferta pendiente en su listado de sesiones activas.

---

### User Story 2 - Visualizar y Aceptar la Sesión P2P como Comprador (Priority: P1)

Como usuario autenticado que recibió un enlace de oferta P2P, quiero ver los detalles de la operación propuesta (vendedor, jugador, cantidad y precio) y decidir si acepto o cancelo, de modo que pueda participar de la transacción directa de forma informada.

**Why this priority**: Es la otra mitad imprescindible del flujo. Sin aceptación del comprador la transacción no puede completarse.

**Independent Test**: Abrir el enlace con un segundo usuario autenticado que tenga créditos suficientes, verificar que la pantalla muestra el nombre de ambos usuarios, el jugador, la cantidad y el precio, pulsar "Aceptar" y confirmar que los tokens pasan al comprador, los créditos al vendedor, y la sesión queda en estado `COMPLETED`.

**Acceptance Scenarios**:

1. **Given** un enlace de sesión P2P en estado `PENDING` y un usuario autenticado con créditos suficientes, **When** el comprador accede al enlace, **Then** la vista muestra: email del vendedor, email del comprador (usuario en sesión), jugador, cantidad de tokens y precio total; además del tiempo restante y los botones "Aceptar" y "Cancelar".
2. **Given** la misma sesión, **When** el comprador pulsa "Aceptar", **Then** el sistema valida que el comprador tiene créditos suficientes, transfiere los tokens al comprador, transfiere los créditos al vendedor, libera la reserva del vendedor, registra la transacción y marca la sesión como `COMPLETED`; cualquier intento posterior de otro usuario de aceptar esa misma sesión es rechazado.
3. **Given** la misma sesión, **When** el comprador no tiene créditos suficientes y pulsa "Aceptar", **Then** el sistema rechaza la operación con error de fondos insuficientes, la sesión permanece en `PENDING` (si aún no expiró) y no se transfiere nada.
4. **Given** la misma sesión, **When** el comprador pulsa "Cancelar", **Then** la sesión pasa a `CANCELLED`, los tokens reservados del vendedor quedan liberados y el comprador ve confirmación de cancelación.
5. **Given** un enlace válido, **When** lo accede el mismo usuario que creó la oferta (el vendedor), **Then** el sistema no le permite actuar como comprador; puede ver los detalles pero no puede aceptar su propia oferta.
6. **Given** una sesión en estado `COMPLETED`, **When** cualquier segundo usuario intenta aceptarla, **Then** el sistema responde con un error claro indicando que la oferta ya fue tomada por otro usuario (HTTP 409).
7. **Given** una sesión en estado `EXPIRED`, **When** cualquier usuario intenta aceptarla, **Then** el sistema responde con un mensaje de error amigable indicando que la oferta venció (HTTP 410), y la vista frontend muestra el mismo mensaje en lugar del formulario de aceptación.

---

### User Story 3 - Expiración Automática de la Sesión P2P (Priority: P2)

Como sistema, debo cancelar automáticamente toda sesión P2P que no sea aceptada dentro de los 5 minutos posteriores a su creación, de modo que los tokens del vendedor no queden reservados indefinidamente.

**Why this priority**: Garantiza la integridad del balance del vendedor y la consistencia del sistema sin requerir acción manual.

**Independent Test**: Crear una sesión P2P y esperar sin interacción a que superen los 5 minutos. Verificar que la sesión pasa a `EXPIRED`, los tokens del vendedor quedan liberados y cualquier intento de aceptar el enlace es rechazado.

**Acceptance Scenarios**:

1. **Given** una sesión P2P en estado `PENDING`, **When** transcurren 5 minutos desde su creación sin que el comprador acepte ni cancele, **Then** la sesión pasa automáticamente a `EXPIRED` y los tokens del vendedor quedan liberados.
2. **Given** una sesión en estado `EXPIRED`, **When** cualquier usuario intenta aceptarla, **Then** el sistema responde con error indicando que la sesión ya no está disponible (HTTP 410 Gone).
3. **Given** la vista de sesión abierta en el navegador del comprador, **When** el tiempo llega a cero, **Then** la interfaz muestra un indicador de tiempo expirado e impide pulsar "Aceptar".

---

### User Story 4 - Cancelar la Sesión como Vendedor (Priority: P2)

Como vendedor que creó una oferta P2P activa, quiero poder cancelarla antes de que expire o sea aceptada, de modo que pueda liberar mis tokens si cambio de opinión.

**Why this priority**: Control operacional del vendedor sobre sus propias ofertas activas.

**Independent Test**: Crear una sesión P2P y, antes de que expire o sea aceptada, cancelarla desde la cuenta del vendedor. Verificar que la sesión pasa a `CANCELLED` y los tokens quedan inmediatamente disponibles.

**Acceptance Scenarios**:

1. **Given** una sesión en estado `PENDING`, **When** el vendedor solicita su cancelación, **Then** la sesión pasa a `CANCELLED`, los tokens reservados quedan liberados y se registra quién inició la cancelación.
2. **Given** una sesión en estado `COMPLETED` o `EXPIRED`, **When** el vendedor intenta cancelarla, **Then** el sistema rechaza la solicitud indicando que la sesión ya está en estado terminal.

---

### Edge Cases

- ¿Qué ocurre si el vendedor vende sus tokens desde el mercado abierto mientras tiene una sesión P2P activa con esos mismos tokens? → Los tokens deben estar reservados (en escrow) desde la creación de la sesión P2P, por lo que no estarán disponibles para otras operaciones hasta que la sesión termine.
- ¿Dos usuarios distintos pueden intentar aceptar la misma sesión simultáneamente? → La sesión tiene bloqueo pesimista al aceptar; sólo el primero en confirmar gana. El segundo recibe un error claro ("la oferta ya fue tomada").
- ¿Puede el mismo comprador abrir el enlace más de una vez simultáneamente? → La sesión sólo acepta una única operación de aceptación; el estado `COMPLETED` es terminal e idempotente para intentos duplicados.
- ¿Puede un usuario no autenticado acceder al enlace? → El sistema debe requerir autenticación para ver los detalles y para aceptar. Un usuario no autenticado es redirigido al login.
- ¿Qué ocurre si el enlace contiene un identificador de sesión inexistente o malformado? → HTTP 404.
- ¿Puede haber múltiples sesiones P2P activas simultáneamente para el mismo vendedor? → Sí, siempre que el vendedor tenga tokens suficientes para cubrir la reserva de todas ellas; cada sesión tiene su propia reserva independiente.
- ¿Qué ven los usuarios cuando la sesión ya expiró? → La vista del frontend muestra un panel de "oferta vencida" en lugar del formulario de aceptación; el backend devuelve HTTP 410 con un mensaje de error amigable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema DEBE permitir a un usuario autenticado crear una sesión de oferta P2P especificando: jugador, cantidad de tokens y precio por token.
- **FR-002**: Al crear una sesión P2P, el sistema DEBE reservar inmediatamente la cantidad de tokens indicada del balance del vendedor, impidiendo su uso en otras operaciones.
- **FR-003**: El sistema DEBE generar un identificador único e impredecible para cada sesión P2P y exponer una URL compartible que lo contenga.
- **FR-004**: El sistema DEBE establecer un tiempo de expiración de 5 minutos desde la creación de la sesión; pasado ese tiempo, la sesión pasa automáticamente a `EXPIRED` y se liberan los tokens reservados.
- **FR-005**: La vista de sesión P2P DEBE mostrar: email del vendedor, email del usuario en sesión (comprador), jugador, cantidad de tokens, precio por token, precio total y tiempo restante hasta expiración.
- **FR-006**: El sistema DEBE impedir que el creador de la oferta (vendedor) actúe como comprador de su propia sesión.
- **FR-007**: El sistema DEBE validar, al momento de aceptar, que el comprador dispone de créditos suficientes para cubrir el precio total antes de ejecutar cualquier transferencia.
- **FR-008**: Al aceptar una sesión, el sistema DEBE: transferir los tokens al comprador, transferir los créditos al vendedor, liberar la reserva del vendedor y registrar la transacción con todos sus datos.
- **FR-009**: El sistema DEBE registrar un evento de auditoría para cada cambio de estado de la sesión (creación, aceptación, cancelación, expiración) incluyendo el actor responsable.
- **FR-010**: El vendedor DEBE poder cancelar una sesión en estado `PENDING`, liberando los tokens reservados.
- **FR-011**: Una sesión en estado terminal (`COMPLETED`, `CANCELLED`, `EXPIRED`) NO DEBE poder ser aceptada ni cancelada nuevamente.
- **FR-012**: El sistema DEBE requerir autenticación para acceder a los detalles de la sesión y para realizar cualquier acción sobre ella.
- **FR-013**: Si una sesión ya fue aceptada por otro usuario (`COMPLETED`), el sistema DEBE responder con HTTP 409 y un mensaje que indique que la oferta ya fue tomada; no debe ser posible que un segundo comprador la tome.
- **FR-014**: Si una sesión está en estado `EXPIRED`, el sistema DEBE responder con HTTP 410 y un mensaje de error amigable; la vista del frontend DEBE mostrar un panel de "oferta vencida" en lugar del formulario de aceptación.

### Key Entities

- **Sesión P2P** (`p2p_session`): Representa la oferta de venta directa entre dos usuarios. Atributos: identificador único, vendedor, jugador, cantidad de tokens, precio por token, estado (`PENDING`, `COMPLETED`, `CANCELLED`, `EXPIRED`), marca de tiempo de creación, marca de tiempo de expiración, comprador (puede estar vacío hasta que alguien acepte).
- **Reserva de tokens** (`token_escrow`): Tokens del vendedor puestos en reserva durante una sesión activa; se liberan al finalizar la sesión por cualquier vía.
- **Transacción P2P**: Registro de la operación ejecutada al aceptar la sesión, análogo a las transacciones del mercado abierto, con referencia a la sesión que la originó.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un vendedor puede crear una oferta P2P y obtener el enlace compartible en menos de 3 segundos.
- **SC-002**: Un comprador puede ver todos los detalles de la oferta y completar la aceptación en menos de 5 pasos desde que abre el enlace.
- **SC-003**: La expiración automática ocurre dentro de los 30 segundos posteriores al vencimiento del plazo de 5 minutos (tolerancia operativa aceptable).
- **SC-004**: El 100% de las sesiones que alcanzan un estado terminal liberan correctamente los tokens reservados del vendedor; no puede haber tokens atrapados en escrow de forma permanente.
- **SC-005**: Ninguna sesión permite que dos usuarios actúen simultáneamente como comprador y vendedor de la misma oferta (no hay doble ejecución).
- **SC-006**: Toda transacción completada queda trazable en el historial de operaciones de ambos usuarios involucrados.

## Assumptions

- Los tokens del vendedor se reservan en escrow al crear la sesión; el mercado abierto no puede consumir tokens reservados.
- El enlace es semiprivado (protegido por un UUID opaco): cualquier usuario autenticado que conozca la URL puede acceder a la vista, pero sólo puede aceptar si es un usuario distinto al vendedor y tiene fondos suficientes.
- El tiempo de expiración de 5 minutos es fijo en esta versión y no es configurable por el usuario.
- La operación de aceptación es atómica: o se completa totalmente o no se modifica ningún balance.
- El sistema de autenticación existente (JWT) se reutiliza; no se introduce ningún mecanismo de autenticación nuevo.
- El frontend mostrará un contador de cuenta regresiva en tiempo real; si el tiempo expira mientras la pantalla está abierta, se bloquea la acción de aceptar y se muestra un mensaje de sesión expirada.
- El vendedor puede tener múltiples sesiones P2P activas simultáneamente siempre que tenga tokens suficientes para cubrir todas las reservas.
- La vista P2P es accesible desde la web (no se requiere soporte móvil nativo en esta versión).

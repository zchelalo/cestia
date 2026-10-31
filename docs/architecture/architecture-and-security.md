# Arquitectura y seguridad

## Forma general

Cestia es un monolito modular:

```text
apps/api   Go + net/http + SQLC/pgx + PostgreSQL
apps/web   React + Vite + TypeScript
apps/web-e2e Playwright
packages/api-contract OpenAPI
.docker    PostgreSQL y Mailpit
```

El API es un proceso. El worker de correo corre como goroutine supervisada dentro
del mismo proceso y reclama trabajos de PostgreSQL. No hay Redis, broker,
microservicios, WebSockets ni almacenamiento de objetos en el MVP.

```text
root
├── apps/api
│   ├── cmd/api
│   ├── cmd/migrate
│   └── internal
│       ├── api          transporte HTTP, cookies, CSRF, envelope
│       ├── apperr       errores tipados y seguros
│       ├── config       configuración y validación de producción
│       ├── data         Store, transacciones, modelos y mapeos SQLC
│       ├── db/sqlc      Código generado; nunca editar a mano
│       ├── domain       decimales y cantidades
│       ├── mailer       SMTP y plantillas embebidas
│       ├── migrations   migraciones versionadas
│       ├── modules
│       │   ├── identity
│       │   ├── groups
│       │   ├── shopping
│       │   └── notifications
│       └── secure       tokens, hashes y derivación HMAC
├── apps/api/queries     Consultas SQLC por dominio
├── apps/api/sqlc.yaml   Configuración SQLC
├── apps/web/src
│   ├── app              router y tokens
│   ├── components       primitivas y shell
│   ├── pages            pantallas
│   └── shared           HTTP, tipos, queries, i18n, dinero
└── packages/api-contract
```

### Límites

- `identity`: cuentas, contraseñas, sesiones y acciones de correo.
- `groups`: grupos, propiedad, membresías e invitaciones.
- `shopping`: listas, artículos, compras, gastos e historial.
- `notifications`: worker, SMTP y plantillas.
- `data`: adaptadores de persistencia, transacciones y mapeos sobre el cliente
  SQLC generado.
- `queries` + `internal/db/sqlc`: todas las consultas de aplicación, tests,
  idempotencia, rate limiting y metadatos de migraciones viven en SQL y se
  ejecutan mediante SQLC. El único uso directo de `Exec` es aplicar archivos DDL
  de migración.
- `scripts/check_no_raw_sql.sh` ejecuta un analizador AST que falla ante SQL
  inline o llamadas directas a `Query`, `QueryRow` o `Exec` fuera del código
  generado.

La autorización vive en los servicios y se revalida dentro de las transacciones.
El frontend solo oculta acciones por usabilidad.

## Identidad y sesiones

- Contraseñas con bcrypt, mínimo 12 caracteres y máximo 72 bytes UTF-8.
- Sesión opaca de 32 bytes; se guarda SHA-256, nunca el token original.
- Cookie `HttpOnly`, `SameSite=Lax`, `Secure` en producción.
- Producción usa `__Host-cestia_session`, sin `Domain`.
- Token CSRF separado, cookie legible y cabecera `X-CSRF-Token`.
- Toda mutación autenticada exige CSRF. Peticiones con `Origin` deben coincidir
  con `APP_ORIGIN`.
- Caducidad absoluta inicial de 14 días y revocación persistida.
- Cambio/recuperación de contraseña y desactivación revocan sesiones.
- Registro y recuperación no revelan existencia de cuentas.
- Rate limiting persistido para login, registro e invitaciones.
- Producción falla al arrancar con secreto débil, cookies inseguras, origen HTTP
  o SMTP sin cifrado.

## Roles

La propiedad canónica es `groups.owner_user_id`. `group_members.role` solo
contiene `admin` o `member`; el rol efectivo `owner` se deriva.

| Operación | Owner | Admin | Member |
|---|:---:|:---:|:---:|
| Consultar listas/compras/gastos/historial | Sí | Sí | Sí |
| Crear/editar/archivar listas | Sí | Sí | No |
| Agregar/editar artículos | Sí | Sí | Sí |
| Eliminar artículos | Sí | Sí | No |
| Registrar compras | Sí | Sí | Sí |
| Corregir/revertir cualquier compra | Sí | Sí | No |
| Corregir/revertir compras propias | Sí | Sí | Sí |
| Invitar/cancelar/reenviar invitaciones | Sí | Sí | No |
| Eliminar members | Sí | Sí | No |
| Nombrar/remover admins | Sí | No | No |
| Editar configuración del grupo | Sí | No | No |
| Archivar/restaurar/transferir | Sí | No | No |
| Abandonar | Tras transferir | Sí | Sí |

Un grupo nunca puede quedar sin propietario activo: triggers diferidos lo
verifican al confirmar. La desactivación de cuenta también se bloquea si el
usuario posee grupos.

## Autorización e IDOR

- Todo acceso a lista incluye la condición personal o la membresía vigente en
  SQL; no se autoriza solo por el ID recibido.
- Recurso ausente o ajeno: `404`. Acción no permitida sobre recurso visible:
  `403`.
- `actor_id`, `recorded_by` y `created_by` se derivan de la sesión.
- Los integrantes retirados pierden acceso en la siguiente petición.
- Las compras conservan atribuciones de exintegrantes, pero no se pueden asignar
  nuevos responsables fuera de la membresía activa.
- El pagador no obtiene permisos adicionales.
- Las listas archivadas y grupos archivados son de solo lectura, salvo
  restauración, transferencia o abandono.

## Concurrencia

- `version BIGINT` en grupos, listas, artículos, invitaciones y compras.
- Las mutaciones requieren `expected_version`; una versión distinta devuelve
  `409 version_conflict`.
- Orden de bloqueo en compras grupales: grupo `SHARE`, lista `SHARE`, artículo
  `UPDATE`, compra/registro idempotente.
- Crear, editar o eliminar artículos bloquea la lista en modo exclusivo porque
  actualiza su versión; así se evita un deadlock por promoción de bloqueo.
- Archivar/eliminar miembros/transferir toman bloqueo exclusivo del grupo.
- Las mutaciones de grupo bloquean primero la fila del grupo y **releen la
  membresía en otra sentencia**; el rol de un join anterior a la espera no se
  usa para autorizar.
- Invitaciones siguen el orden grupo → invitación → entregas, también al
  reenviar o cancelar.
- Crear o transferir un grupo bloquea al propietario destinatario antes del
  grupo; una cuenta deshabilitada no puede recibir propiedad.
- Login, cambio de contraseña, desactivación y acciones de correo bloquean al
  usuario antes de revalidar. Las acciones de correo se emiten bajo ese bloqueo,
  por lo que solo una queda vigente por usuario y propósito.
- Los resúmenes multi-consulta usan una transacción `REPEATABLE READ` de solo
  lectura para no mezclar dos estados.
- El migrador toma un advisory lock transaccional y aplica todo el lote en una
  sola transacción; dos procesos no pueden crear el mismo esquema a la vez.
- Una sola compra activa por artículo mediante índice único parcial.
- `Idempotency-Key` obligatoria en compras, correcciones, reversiones,
  invitaciones y transferencia. La huella incluye operación, recurso y comando;
  se persiste también `resource_id`. Reutilizar una clave en otro artículo o
  compra devuelve `409 idempotency_key_reused`.
- Un advisory lock transaccional serializa la misma clave antes de mutar.
- Al reutilizar una clave vencida se reemplazan huella, respuesta y
  `resource_id`. Los registros anteriores sin `resource_id` se reconocen por su
  huella antigua, se validan contra el recurso de la respuesta y se actualizan
  al formato estricto.

## Correo

- Puerto SMTP con TLS/STARTTLS; Mailpit en local.
- Plantillas HTML y texto embebidas.
- Trabajos persistidos con estado, intentos, backoff, lease y `SKIP LOCKED`.
- Un fallo SMTP no cancela la invitación.
- Máximo cinco intentos, con backoff aproximado de 1, 5, 30 y 120 minutos.
- `failed` es terminal: no se reclama otra vez automáticamente. Un reenvío
  explícito crea un trabajo nuevo.
- Entrega **al menos una vez**: un correo duplicado es posible, una membresía
  duplicada no.
- Tokens de invitación derivados con HMAC y generación; solo se persiste el hash.
- Reenviar invalida el enlace anterior.
- La aceptación se hace con `POST`; el enlace nunca muta por sí solo.
- El correo de la sesión debe coincidir con el invitado.

## Sincronización

El MVP usa polling, no SSE ni WebSockets:

- Lista compartida visible: 15 s.
- Resumen/actividad grupal: 30 s.
- Refetch al enfocar; sin escritura offline ni service worker que almacene datos
  privados.
- Las mutaciones financieras esperan confirmación del servidor; no hay updates
  optimistas de gastos.

## Riesgos residuales

- La persistencia compartida en `data` puede crecer; conviene dividirla por
  módulo cuando aparezcan más invariantes o equipos.
- No hay un generador OpenAPI conectado al cliente TypeScript; tipos y cliente
  son manuales.
- Playwright cubre autenticación, compra personal en escritorio y móvil,
  corrección/reversión y un flujo grupal con dos usuarios.
- La entrega SMTP no es exactamente una vez.

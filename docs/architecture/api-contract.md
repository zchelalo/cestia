# Contrato API

El contrato canónico está en
[`packages/api-contract/openapi.yaml`](../../packages/api-contract/openapi.yaml).

## Transporte

- REST/JSON bajo `/api/v1`.
- `net/http` y `ServeMux` del API Go.
- Mismo origen en producción; Vite proxyfica `/api` en desarrollo.
- Fechas RFC 3339 y UUID.
- Importes como strings decimales.

No se usa ConnectRPC: el proyecto de referencia implementa HTTP/JSON y Cestia no
necesita otra cadena de herramientas.

## Envelope

Éxito:

```json
{
  "data": {},
  "meta": { "next_cursor": "…", "limit": 30 },
  "request_id": "01a…"
}
```

Error:

```json
{
  "error": {
    "code": "version_conflict",
    "message": "El recurso cambió. Actualiza e intenta de nuevo.",
    "details": {}
  },
  "request_id": "01a…"
}
```

Los errores internos se registran con causa y `request_id`; nunca se serializa
`err.Error()` al cliente.

## Autenticación

| Operación | Método y ruta |
|---|---|
| Registro | `POST /auth/register` |
| Verificación | `POST /auth/verify-email` |
| Reenvío | `POST /auth/resend-verification` |
| Login | `POST /auth/login` |
| Logout | `POST /auth/logout` |
| Recuperación | `POST /auth/forgot-password` |
| Nueva contraseña | `POST /auth/reset-password` |
| Cambio autenticado | `POST /auth/change-password` |
| Perfil | `GET/PATCH /me` |
| Sesiones | `GET /me/sessions`, `DELETE /me/sessions/{id}` |
| Desactivación | `POST /me/deactivate` |

Login devuelve también `csrf_token`; la cookie CSRF es legible y debe enviarse en
`X-CSRF-Token` para mutaciones autenticadas.

## Grupos e invitaciones

- `GET/POST /groups`
- `GET/PATCH /groups/{id}`
- `POST /groups/{id}/archive|restore|transfer-ownership|leave`
- `GET /groups/{id}/members`
- `PATCH/DELETE /groups/{id}/members/{userId}`
- `GET/POST /groups/{id}/invitations`
- `POST /groups/{id}/invitations/{invitationId}/resend`
- `DELETE /groups/{id}/invitations/{invitationId}`
- `POST /invitations/preview|accept|reject` con token en el cuerpo

`preview`, `accept` y `reject` requieren sesión verificada y correo coincidente.
La aceptación se ejecuta en una transacción con bloqueo de grupo e invitación.

## Listas y artículos

- `GET/POST /shopping-lists`
- `GET/PATCH /shopping-lists/{id}`
- `POST /shopping-lists/{id}/archive|restore`
- `GET/POST /shopping-lists/{id}/items`
- `PATCH/DELETE /items/{id}`

`group_id` ausente significa espacio personal. `include_archived` controla la
vista operativa.

## Compras

- `POST /items/{id}/purchases`
- `GET/PATCH /purchases/{id}`
- `POST /purchases/{id}/reverse`

Las tres mutaciones requieren `Idempotency-Key`.

`PurchaseInput`:

```json
{
  "expected_version": 1,
  "actual_unit_price": "42.50",
  "purchased_by": "uuid",
  "paid_by": "uuid | null",
  "purchased_at": "2026-10-30T16:00:00-06:00",
  "notes": ""
}
```

`actual_unit_price` vacío/nulo deja el precio desconocido. El cliente no envía el
total: lo calcula el servidor desde cantidad y precio.

## Gastos e historial

- `GET /expenses/summary`
- `GET /expenses/by-list`
- `GET /purchases`
- `GET /activity`

Filtros comunes: `group_id`, `from`, `to`. `include_archived` se aplica de forma
explícita y por defecto es `true` en gastos.

`GET /purchases` usa cursor opaco y límite 1–100 (30 por defecto). Las
agregaciones cubren todas las filas filtradas, no solo la página.

## Códigos de error

| Status | Uso |
|---|---|
| 400 | JSON, formato, cursor o ID inválido |
| 401 | Sesión ausente, inválida o expirada |
| 403 | Rol insuficiente, CSRF u origen inválido |
| 404 | Recurso inexistente o inaccesible |
| 409 | Versión, estado, duplicado o clave idempotente incompatible |
| 410 | Invitación/enlace vencido o reemplazado |
| 422 | Regla de negocio/validación |
| 429 | Rate limit con `Retry-After` |
| 500 | Error interno redactado |

## Validaciones y límites

- Nombres 1–120; descripciones ≤ 2,000; notas ≤ 4,000; motivos ≤ 500.
- Cuerpo máximo 64 KiB; se rechazan campos desconocidos.
- Cantidad `0 < q ≤ 999999.999`, hasta 3 decimales.
- Precio `0 ≤ p ≤ 99999999.9999`, hasta 4 decimales.
- Fecha de compra no futura salvo tolerancia técnica de 5 minutos.
- Pagador nulo permitido; comprador obligatorio.
- No hay endpoints ni campos de deuda, saldo, división o liquidación.

## Cliente TypeScript

El frontend usa un cliente tipado manual en
`apps/web/src/shared/http.ts` y tipos en `apps/web/src/shared/types.ts`.

**Pendiente / desviación:** el contrato OpenAPI todavía no genera esos tipos ni
un cliente en CI. Antes de ampliar la superficie pública conviene conectar
`openapi-typescript` y una comprobación de regeneración.

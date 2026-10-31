# Datos, compras y gastos

## Principio central

Cestia separa cuatro cosas que suelen confundirse:

```text
artículo planificado
compra registrada
atribuciones de la compra
historial de cambios
```

Un artículo no tiene un estado `purchased` mutable. **Está comprado si existe una
compra no revertida.** Así no puede haber un artículo marcado como comprado sin
compra, ni una compra activa en un artículo pendiente.

## Tablas

| Tabla | Propósito |
|---|---|
| `users` | Cuenta, credenciales, verificación, perfil y desactivación |
| `sessions` | Sesiones opacas revocables y hash CSRF |
| `email_actions` | Verificación de correo y recuperación de contraseña |
| `groups` | Grupo y propietario canónico |
| `group_members` | Membresía histórica con rol base y retiro lógico |
| `group_invitations` | Invitaciones con hash, generación, expiración y estado |
| `shopping_lists` | Lista personal **o** de grupo, nunca ambas |
| `shopping_list_items` | Artículo planificado, búsqueda de compra activa |
| `purchases` | Compra, snapshots, importes, atribuciones y reversión |
| `activity_events` | Historial append-only de negocio |
| `email_deliveries` | Cola y estado de entrega SMTP |
| `idempotency_requests` | Repetición segura de mutaciones |
| `rate_limit_buckets` | Límites persistentes por ventana |

### Restricciones esenciales

- `shopping_lists` usa XOR entre `personal_owner_id` y `group_id`.
- Membresía única por `(group_id, user_id)`.
- Invitación pendiente única por `(group_id, lower(email))`.
- Una compra activa única por artículo:
  `UNIQUE (item_id) WHERE reversed_at IS NULL`.
- Un grupo conserva su propietario activo mediante trigger diferido.
- Las FK de compra e historial no usan cascadas destructivas.
- Listas, grupos y artículos se archivan/eliminan lógicamente; los gastos no se
  borran.

### Índices principales

- Email normalizado de usuario.
- Sesiones activas por usuario/expiración.
- Membresías activas por usuario y grupo.
- Invitaciones por hash, grupo y expiración.
- Listas por propietario/grupo y archivo.
- Artículos activos por lista.
- Compras por artículo, fecha y pagador.
- Actividad por alcance/fecha y recurso.
- Cola por estado y próxima ejecución.
- Idempotencia por actor/operación/clave.

## Dinero

| Dato | PostgreSQL |
|---|---|
| Cantidad | `NUMERIC(12,3)` |
| Precio unitario | `NUMERIC(14,4)` |
| Total de línea | `NUMERIC(18,2)` |
| Moneda | `MXN` |

- La API expone importes como **strings**.
- Go usa `shopspring/decimal`; el navegador usa `decimal.js`.
- La base calcula `total_amount` como columna generada:
  `round(quantity * actual_unit_price, 2)`.
- Se redondea cada línea antes de sumar (`ROUND_HALF_UP` para no negativos).
- No se acepta un total independiente del precio unitario y la cantidad.
- `actual_unit_price = NULL` significa precio desconocido.
- `actual_unit_price = 0` significa compra conocida sin costo.
- `estimated_unit_price` no participa en el gasto real.
- No hay conversión de moneda.

Ejemplos verificados:

```text
3 × 0.10      = 0.30
0.125 × 19.99 = 2.50
2 × 42.50     = 85.00
```

## Compra completa del MVP

Cada compra captura la cantidad completa del artículo en ese momento. No hay
compras parciales ni múltiples compras activas.

Para cambiar cantidad de un artículo comprado:

1. Revertir la compra con motivo.
2. Editar el artículo.
3. Registrar una compra nueva.

La compra revertida se conserva y deja de contribuir a totales.

## Atribuciones

| Campo | Significado |
|---|---|
| `item.created_by` | Agregó el artículo |
| `purchase.recorded_by` | Registró la compra en Cestia |
| `purchase.purchased_by` | Realizó la compra |
| `purchase.paid_by` | Aportó el dinero; puede ser nulo |

`paid_by` representa dinero pagado, no dinero adeudado. Los resúmenes agrupan por
`paid_by`; los nulos aparecen como “No identificado”.

Los snapshots de nombre/unidad y los nombres visibles de comprador/pagador
conservan interpretación histórica aunque el integrante salga del grupo o cambie
su perfil.

## Corrección y reversión

### Corrección

Permite cambiar precio, comprador, pagador, fecha y notas. Exige
`expected_version` y motivo. No cambia artículo, cantidad ni unidad. Un miembro
solo puede corregir compras que registró.

### Reversión

Marca `reversed_at`, `reversed_by` y motivo. La compra no se borra ni se
reactiva. El artículo vuelve a pendiente y el gasto deja de contarse.

### Historial

`activity_events` guarda actor, alcance, lista/artículo/compra, acción,
`payload_version`, valores anteriores/nuevos y fecha. Si falla la inserción del
evento, la mutación no confirma.

Acciones registradas incluyen:

- `list.created`, `list.updated`, `list.archived`, `list.restored`
- `item.created`, `item.updated`, `item.deleted`
- `purchase.recorded`, `purchase.corrected`, `purchase.reversed`
- `group.created`, `group.updated`, `group.archived`, `group.restored`,
  `group.owner_transferred`
- `member.role_changed`, `member.removed`, `member.left`
- `invitation.created`, `invitation.resent`, `invitation.canceled`,
  `invitation.accepted`
- Eventos de cuenta y seguridad (`account.*`)

## Totales

Se calculan desde las fuentes; no hay agregados persistidos.

Cada resumen devuelve:

- `estimated_known`: valor planificado conocido.
- `estimated_unknown_items`: artículos sin estimación.
- `pending_estimated`: planificado conocido de artículos sin compra activa.
- `actual_known`: suma de compras activas con precio.
- `purchases_without_price`: compras activas sin precio real.
- `purchase_count`, `pending_count`, `purchased_count`.
- `by_payer`: importes pagados agrupados por `paid_by`.
- `unidentified_paid`: subconjunto sin pagador identificado.

Una compra sin precio no se convierte en cero. Los totales reales se muestran
como “conocidos” cuando existe al menos una compra sin precio.

## Fechas

- Persistencia `timestamptz`.
- API RFC 3339.
- Periodos `[from, to)`.
- La UI usa la zona horaria del perfil (`America/Mexico_City` por defecto).
- Los filtros de periodo afectan gasto real e historial de compras.
- Pendientes y planificado reflejan el estado actual, no un presupuesto
  histórico.

## Archivo

- Las vistas operativas de listas excluyen archivadas por defecto.
- Gastos e historial las incluyen por defecto y devuelven `include_archived`.
- Archivar no borra artículos, compras ni eventos.
- Una lista/grupo archivado es de solo lectura.

## Migraciones

- Archivos `NNNNNN_name.up.sql` y `.down.sql` embebidos.
- El runner crea/verifica el diario mediante SQLC y aplica en orden los archivos
  DDL, registrando cada versión con la consulta generada.
- `000000_schema_migrations` crea el diario de forma idempotente.
- Todas las consultas de metadatos usan SQLC; la ejecución del cuerpo DDL es la
  única excepción documentada.
- No se reescribe una migración desplegada.
- La corrección del trigger de propiedad se añadió como `000003`, no editando
  `000002`.
- Las migraciones se ejecutan explícitamente con `pnpm nx run api:migrate`.
- En producción deben ejecutarse una vez antes de arrancar las instancias, no
  concurrentemente desde cada instancia.

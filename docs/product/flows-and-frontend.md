# Flujos y frontend

## Rutas

| Ruta | Función |
|---|---|
| `/sign-in`, `/sign-up` | Acceso y registro |
| `/forgot-password`, `/reset-password` | Recuperación |
| `/verify-email` | Verificación |
| `/` | Dashboard |
| `/personal` | Listas personales |
| `/groups` | Grupos |
| `/groups/:groupId` | Grupo, listas, resumen y actividad |
| `/groups/:groupId/members` | Integrantes e invitaciones |
| `/lists/:listId` | Artículos y compras |
| `/expenses` | Gastos por alcance y periodo |
| `/history` | Compras y actividad |
| `/settings` | Perfil, seguridad y sesiones |
| `/invitations/accept` | Aceptar/rechazar invitación |

## Flujos

### Registro y verificación

1. La persona se registra sin iniciar sesión.
2. Se encola un correo con enlace de un solo uso.
3. El enlace se consume con `POST`; no muta por GET.
4. Al verificar puede iniciar sesión.
5. Una cuenta no verificada no accede a recursos privados.

### Compra personal

1. Crear lista.
2. Agregar artículos con cantidad, unidad y estimado opcional.
3. Registrar compra: precio real opcional, comprador, pagador, fecha y notas.
4. La API calcula el total y actualiza el resumen.
5. Corregir o revertir con motivo conserva historial.

### Grupo

1. Crear grupo; el creador es propietario y miembro.
2. Invitar por correo; la invitación se envía por SMTP.
3. El destinatario inicia sesión con ese correo.
4. Previsualiza, acepta o rechaza.
5. La aceptación crea membresía `member` en una transacción.
6. Owner/admin crean listas compartidas.
7. Cualquier integrante agrega artículos y registra compras.
8. Correcciones y reversiones respetan rol y `recorded_by`.

### Gastos

1. Elegir espacio personal o un grupo.
2. Aplicar periodo opcional.
3. Ver planificado, pendiente y gastado real por separado.
4. Ver desglose por `paid_by`.
5. Ver compras sin precio y pagador no identificado.
6. No hay balances, deudas ni liquidaciones.

## Pantallas y componentes

Componentes principales:

- `AppShell`: navegación lateral, cabecera y nav móvil.
- `AuthShell`: acceso/registro responsive.
- `PageHeader`, `SummaryCard`, `States`, `Modal`, `Field`, `Button`.
- `ActivityTimeline`.
- Formularios de artículo, compra, corrección, reversión, lista y grupo.

La lista muestra:

- Estado derivado (pendiente/comprado).
- Cantidad y unidad.
- Estimado.
- Comprador y pagador.
- Importe real o “sin precio”.
- Acciones según permisos y archivo.

## Responsive

Verificado en 1440×900 y 390×844; CSS contempla desde 320 px.

- Móvil: una columna, nav inferior, acciones táctiles de ≥44 px, sin overflow
  horizontal en las pantallas probadas.
- Inputs a 16 px en móvil para evitar zoom.
- Formularios de compra en modal tipo sheet.
- Tablas/listas convertidas en filas legibles.
- Foco visible, labels y `prefers-reduced-motion`.

## Estado

- TanStack Query para estado servidor.
- TanStack Router para rutas.
- React Hook Form + Zod en formularios.
- Invalidación de listas, items, resumen, historial, actividad y dashboard tras
  mutaciones.
- Limpieza total de caché al cerrar sesión/cambiar cuenta.
- Sin updates optimistas de compras.
- Polling de listas compartidas cada 15 s y resumen grupal cada 30 s; refetch al
  enfocar.
- No hay offline ni service worker que guarde datos privados.
- E2E cubre autenticación, registro/verificación, compra personal en escritorio y móvil, corrección/reversión y colaboración grupal.

## Estados de interfaz

- Carga, error con reintento y vacío.
- Solo lectura para archivo.
- Errores de versión sin destruir el formulario.
- Confirmación antes de eliminar integrante, revertir o salir.
- Avisos explícitos de que los pagos no son deudas.
- Badges de rol/propiedad y estado de invitación.
- `gasto real conocido` y contadores de datos faltantes.

## Accesibilidad

- Estructura semántica, headings y labels.
- Botones táctiles y foco visible.
- Diálogo con `aria-modal` y cierre por Escape.
- Contraste sobre tokens claro/oscuro.
- No se depende solo del color para estado: hay texto.

## i18n

i18next con español inicial. La infraestructura permite añadir locales, aunque
algunos textos secundarios de la implementación actual siguen en español directo
y deberían moverse al catálogo al añadir un segundo idioma.

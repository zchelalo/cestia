# Plan MVP y estado de implementación

Fecha: 2026-10-30.

## Fases

| Fase | Estado | Resultado |
|---|---|---|
| 0. Investigación y diseño | Completada | Referencias inspeccionadas; arquitectura, datos, API, flujos y plan documentados |
| 1. Fundación | Implementada | Nx/pnpm, API Go, SPA, PostgreSQL, Mailpit, migraciones, auth, sesiones, correo, CSRF, tests |
| 2. Compras personales | Implementada | Listas, artículos, compras, decimales, idempotencia, corrección, reversión, gastos e historial |
| 3. Grupos y colaboración | Implementada | Roles, membresías, invitaciones, entrega, aceptación, listas compartidas, IDOR y concurrencia |
| 4. Gastos y UX | Implementada (MVP) | Resúmenes, desglose por pagador/lista, filtros, responsive, polling, estados |
| 5. Opcionales | No iniciada | Sin notificaciones push, PWA offline, recibos, códigos de barras, compras parciales ni app móvil |

## Tareas y validación

### Fundación

- [x] Monorepo Nx + pnpm.
- [x] API modular Go con configuración tipada.
- [x] SPA React 19 + Vite + TypeScript.
- [x] PostgreSQL 17 y Mailpit en Compose.
- [x] Migraciones versionadas y runner de arranque.
- [x] SQLC v1.29.0 para todas las consultas, incluido tests y metadatos de migración.
- [x] Analizador AST que prohíbe SQL inline y llamadas directas fuera del código generado.
- [x] Bloqueo exclusivo de lista en mutaciones de artículos y prueba de idempotencia por recurso.
- [x] Renovación de claves vencidas con `resource_id` actualizado y migración de registros idempotentes anteriores.
- [x] Bloqueo y relectura de credenciales, membresías y propiedad; emisión serializada de acciones de correo.
- [x] Resúmenes sobre una instantánea de lectura y migrador serializado con advisory lock.
- [x] Pruebas concurrentes de membresía, transferencia/desactivación, snapshot y migradores.
- [x] Entregas de correo `failed` terminales con prueba de no reclamación.
- [x] Registro, verificación, login, logout, recuperación y cambio de contraseña.
- [x] Sesiones opacas, cookies HttpOnly, CSRF, revocación y rate limiting.
- [x] Cola de correo con plantillas, leases y reintentos.
- [x] Errores tipados y envelope.
- [x] Tests unitarios de dinero y tokens; integración PostgreSQL.
- [x] E2E de registro, verificación por Mailpit y compra personal.
- [ ] E2E de recuperación de contraseña.

### Compras personales

- [x] CRUD de listas y artículos.
- [x] Cantidad `NUMERIC(12,3)`, precio `NUMERIC(14,4)`, total `NUMERIC(18,2)`.
- [x] Cálculo decimal exacto y redondeo por línea.
- [x] Registro de compra completa con precio desconocido/cero.
- [x] Comprador y pagador independientes.
- [x] Idempotencia y compra activa única.
- [x] Corrección/reversión con motivo e historial.
- [x] Totales y filtros.
- [x] Prueba de `3 × 0.10 = 0.30` y `0.125 × 19.99 = 2.50`.
- [x] E2E que registra, verifica, agrega artículo y compra con total `$0.30`.
- [ ] Pruebas de servicio con repositorios falsos para cada caso de uso.
- [ ] Concurrencia automatizada de dos registros simultáneos en CI.

### Grupos

- [x] Creación con propietario y membresía atómica.
- [x] Matriz Owner/Admin/Member aplicada en backend.
- [x] Trigger diferido: el grupo conserva propietario activo.
- [x] Invitaciones con hash, HMAC, generación y expiración.
- [x] Reenvío invalida enlace anterior; cancelación.
- [x] Aceptación/rechazo con correo coincidente.
- [x] Una sola aceptación efectiva.
- [x] Listas compartidas, compras y atribuciones.
- [x] Retiro pierde acceso; transferencia y abandono.
- [x] IDOR verificado: lista personal devuelve 404 a otra cuenta.
- [x] E2E multiusuario con dos contextos de navegador: invitación, aceptación, compra y desglose por pagador.
- [ ] Prueba automatizada de compra vs. retiro concurrente.

### Gastos y experiencia

- [x] Resumen personal/grupal.
- [x] Totales planificado, pendiente y real separados.
- [x] Importes por pagador y sin pagador identificado.
- [x] Gasto por lista.
- [x] Historial y actividad con filtros.
- [x] Diseño responsive y revisión visual 1440×900 / 390×844.
- [x] Polling 15/30 s.
- [x] Errores de versión, archivo y solo lectura.
- [ ] Optimización de índices basada en planes con volumen sintético.
- [x] E2E de corrección, pagador desconocido y reversión con historial visible.

## Criterios de aceptación

### Verificados en esta implementación

| Criterio | Evidencia |
|---|---|
| Registro + correo + verificación + login | Flujo ejecutado contra Mailpit y API |
| Sesión revocable y CSRF | Login, mutaciones autenticadas y pruebas manuales |
| Lista personal privada | Bob recibió 404 sobre lista de Ana |
| Compra exacta | `3 × 0.10 = 0.30` vía API |
| Idempotencia | Reintento no creó otra compra ni evento |
| Comprador/pagador distintos | Bob compró; Ana pagó; resumen atribuyó a Ana |
| Corrección con motivo | Precio 42.50 → 43.00, total 85 → 86 |
| Grupo + invitación + aceptación | Ana invitó; Bob aceptó; membresía confirmada |
| Invitación de un solo uso | Segunda aceptación devolvió 409 |
| Owner no abandona | `409 owner_must_transfer` |
| Transferencia a cuenta deshabilitada | `409 owner_unavailable` |
| Member no elimina | `403` |
| Miembro eliminado pierde acceso | Bob recibió 404 tras retiro |
| Montos decimales en PostgreSQL | Prueba de integración `2.50` |
| Compra activa única | Prueba de integración de índice parcial |
| Propietario activo obligatorio | Prueba de integración del trigger diferido |
| Sin overflow móvil | Playwright 390×844 |
| Sin deudas/balances | No existen tablas, endpoints ni pantallas |

### Pendientes antes de considerar E2E completo

- Aceptación de invitación concurrente automatizada.
- Compra duplicada concurrente automatizada (hoy cubierta por índice e
  idempotencia, probada manualmente).
- Fallo SMTP y reintento con worker real en CI.
- Reversión visible en la UI E2E.
- Flujo completo de recuperación de contraseña en navegador.
- Revisión de accesibilidad automatizada (axe).

## Desviaciones del contrato aprobado

1. **Persistencia:** todas las consultas pasan por SQLC v1.29.0 y
   `internal/db/sqlc`. `pgx` se limita al pool, transacciones y ejecución de DDL
   de migraciones. Un control de CI prohíbe SQL inline y llamadas directas a
   `Query`, `QueryRow` o `Exec` fuera del código generado.
2. **Contrato TypeScript:** el OpenAPI existe, pero el cliente y los tipos del
   frontend son manuales. No hay comprobación de regeneración en CI.
3. **E2E:** Playwright verifica autenticación, responsive, compra personal en
   escritorio y móvil, corrección/reversión y colaboración grupal con dos
   contextos de navegador. Queda pendiente automatizar concurrencia y retiro de
   acceso.
4. **`expenses/by-list`:** se añadió a la API y a la UI, pero no estaba en el
   listado original de mensajes del plan.
5. **Tests de servicio:** no se han creado dobles de repositorio para cada caso de
   uso; dominio, restricciones PostgreSQL y flujos principales están cubiertos.
6. **i18n:** la estructura existe; algunos textos secundarios permanecen
   directamente en español.

No hay desviación en el límite de producto: no existen deudas, balances,
divisiones, liquidaciones, reembolsos, transferencias ni pasarelas.

## Validación ejecutada

```text
go test ./...                                              OK
CESTIA_TEST_DATABASE_URL=… go test -race -run Integration  OK
pnpm nx run api:test-integration                           OK (con URL)
go test -race ./...                                        OK
pnpm nx run web:typecheck                                  OK
pnpm nx run web:build                                      OK
pnpm nx run web:test                                       3 tests OK
pnpm nx run web-e2e:e2e                                    8 passed, 2 skipped
pnpm nx run api:sqlc-check                                 compile + vet + diff OK
scripts/check_no_raw_sql.sh (AST)                          OK
go test -race -run Integration (delivery/idempotencia)      OK
Integración concurrente (bloqueos/snapshot/migración)      OK
Migraciones PostgreSQL 17                                   OK
Flujos API: registro, verificación, login, listas,
artículos, compra idempotente, resumen, grupos,
invitaciones, aceptación, roles, IDOR, retiro            OK
Playwright visual: 1440×900 y 390×844, sin overflow       OK
```

## Validación no ejecutada

- E2E de retiro de integrante y pérdida de acceso.
- Worker SMTP con caída/recuperación real en CI.
- Pruebas de carga y planes de índices con volumen.
- Axe/automatización de accesibilidad.
- Despliegue productivo y migraciones en un entorno real.

## Siguientes tareas recomendadas

1. Automatizar el retiro de integrante y la pérdida de acceso en E2E.
2. Añadir pruebas de concurrencia con barreras, sin sleeps.
3. Conectar generación OpenAPI → TypeScript y verificar en CI.
4. Dividir los archivos de consultas si crecen; mantener SQLC como única vía de acceso.
5. Añadir observabilidad de cola y métricas de entregas fallidas.
6. Completar i18n de textos secundarios.
7. Revisar índices de gastos con datos sintéticos.

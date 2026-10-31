# Documentación de Cestia

Cestia organiza compras personales y compartidas. Registra lo que se planea,
lo que efectivamente se compró y quién aportó el dinero. **No calcula deudas,
balances, divisiones, reembolsos ni liquidaciones.**

## Mapa

| Documento | Contenido |
|---|---|
| [research/reference-projects.md](research/reference-projects.md) | Hallazgos de Comledge y Neuraclinic, patrones reutilizados y descartados |
| [architecture/architecture-and-security.md](architecture/architecture-and-security.md) | Monolito modular, sesiones, CSRF, permisos, concurrencia y correo |
| [architecture/data-and-purchases.md](architecture/data-and-purchases.md) | Esquema, precisión monetaria, compras, correcciones e historial |
| [architecture/api-contract.md](architecture/api-contract.md) | REST/JSON, envelope, errores, idempotencia y operaciones |
| [product/flows-and-frontend.md](product/flows-and-frontend.md) | Flujos, pantallas, responsive, estados y sincronización |
| [plans/mvp-implementation.md](plans/mvp-implementation.md) | Fases, estado real, validación y trabajo pendiente |

Todas las consultas de persistencia se generan con SQLC v1.29.0 desde
`apps/api/queries/`; `scripts/check_no_raw_sql.sh` usa análisis AST para impedir SQL manual.

El contrato OpenAPI está en
[`packages/api-contract/openapi.yaml`](../packages/api-contract/openapi.yaml).

## Reglas de producto

1. Un artículo planificado no es una compra.
2. Una compra registrada no es una deuda.
3. El precio estimado nunca cuenta como gasto real.
4. Un precio real desconocido no es cero.
5. `recorded_by`, `purchased_by` y `paid_by` son identidades distintas.
6. Archivar conserva historia y gastos.
7. Los permisos se aplican en el backend y por recurso.
8. Los tokens originales no se persisten.
9. Una mutación financiera y su historial confirman en la misma transacción.
10. Si una operación no puede garantizar estas reglas, falla de forma cerrada.

## Estado

El MVP implementado cubre autenticación, espacio personal, grupos, invitaciones,
compras, correcciones, reversiones, gastos e historial. Las diferencias entre el
diseño aprobado y el código actual están registradas en
[`plans/mvp-implementation.md`](plans/mvp-implementation.md).

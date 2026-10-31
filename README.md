# Cestia

Compras personales y compartidas, con registro de gastos reales. Cestia no
calcula deudas, saldos, divisiones ni liquidaciones.

## Requisitos

- Go 1.27.1
- Node.js 24
- pnpm 10.15
- Docker Compose

## Inicio local

```bash
pnpm install
pnpm dev
```

Ese comando copia `apps/api/.env` si no existe, construye y levanta PostgreSQL,
Mailpit y la API con Air, aplica migraciones y arranca Vite. La API escucha en
`http://localhost:8000`, la SPA en `http://localhost:5173` y Mailpit en
`http://localhost:8025`.

Guardar archivos Go, migraciones o plantillas de correo recompila y reinicia la
API dentro del contenedor.

Comandos útiles:

```bash
pnpm nx run dev:env       # copiar envs faltantes sin sobrescribir
pnpm nx run dev:infra     # solo PostgreSQL y Mailpit
pnpm nx run dev:up        # API + infraestructura, sin Vite
pnpm nx run dev:status    # estado de contenedores
pnpm nx run dev:logs      # logs del stack
pnpm nx run dev:down      # detener sin borrar datos
pnpm nx run api:restart   # recrear la API tras cambiar el env
```

La API también puede ejecutarse sin Docker con `pnpm nx run api:serve-local`.
Los correos de verificación, recuperación e invitaciones se capturan en
Mailpit.

## SQLC

Todas las consultas de la API se generan desde `apps/api/queries/`:

```bash
pnpm nx run api:sqlc
pnpm nx run api:sqlc-check
pnpm nx run api:check-no-raw-sql
```

No se acepta SQL inline en Go. El único `Exec` directo permitido es el que aplica
archivos DDL del runner de migraciones.

## Validación

```bash
pnpm nx run api:test
pnpm nx run web:test
pnpm nx run-many -t lint,typecheck,build
```

La base de integración debe ser desechable:

```bash
CESTIA_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:55432/cestia_test?sslmode=disable \
  pnpm nx run api:test-integration
```

E2E requiere API, SPA y Mailpit activos:

```bash
CESTIA_E2E_BASE_URL=http://localhost:5173 \
CESTIA_CHROMIUM_PATH=/usr/bin/chromium-browser \
  pnpm nx run web-e2e:e2e
```

Los documentos de arquitectura y producto están en [`docs/`](docs/).

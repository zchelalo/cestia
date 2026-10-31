# Proyectos de referencia

Fecha de inspección: 2026-10-30. No se modificaron Comledge ni Neuraclinic.

## Comledge

Ruta: `/home/chelalo/projects/personal/comledge`.

### Arquitectura observada

- Nx como interfaz de tareas y pnpm para el frontend.
- API Go en `apps/api`, SPA React/Vite en `apps/web` y Playwright en
  `apps/web-e2e`.
- HTTP/JSON con `net/http`, no ConnectRPC.
- PostgreSQL 17, `pgx/v5`, SQL explícito y `sqlc` por módulo.
- Migraciones numeradas con `golang-migrate`.
- `internal/modules/{auth,user,article,audit}` sin importaciones entre módulos.
- Casos de uso por paquete, puertos tipados y adaptadores de persistencia/HTTP.
- Transacciones con contexto y auditoría append-only en la misma transacción.
- Frontend con TanStack Router/Query, React Hook Form, Zod, i18next y CSS Modules.
- Tokens de acceso/refresh JWT con cookies para web.

### Reutilizado

- Forma del monorepo y targets Nx.
- Separación HTTP, casos de uso, dominio y persistencia.
- Envelope `{data, meta, error, request_id}`.
- PostgreSQL, transacciones explícitas, errores seguros y pruebas de integración
  contra una base desechable.
- React, Vite, TanStack Router/Query, RHF, Zod, i18next, CSS Modules y tokens
  visuales.
- Invalidación de consultas después de mutaciones y limpieza de caché al cambiar
  de sesión.

### Descartado

- ACL globales, permisos por artículo, versiones documentales y workflows.
- TipTap, PDF, S3/MinIO y adjuntos.
- JWT de acceso/refresh: Cestia usa sesiones opacas revocables.
- `--passWithNoTests`, porque podía ocultar pruebas frontend ausentes.
- Terraform y despliegue específicos de Comledge.

## Neuraclinic

Ruta: `/home/chelalo/projects/personal/neuraclinic/neuraclinic`.

### Correo observado

- Puerto `Mailer` independiente del proveedor.
- SMTP con TLS/STARTTLS, timeout y autenticación opcional.
- Mensajes multipart HTML/texto con `html/template` y `text/template`.
- Plantillas embebidas por tipo e idioma.
- Cola persistida con estado, intentos, backoff, leases y `SKIP LOCKED`.
- Tokens derivados con HMAC; solo hashes en persistencia; cargas de cola sin
  tokens originales.
- ADR 0005: Amazon SES mediante SMTP, DKIM/SPF/DMARC y supresión.

### Reutilizado conceptualmente

- Puerto de correo, adaptador SMTP y plantillas embebidas.
- Entrega asíncrona con estado persistido y reintentos.
- Derivación HMAC de tokens y almacenamiento exclusivo de hashes.
- Mensajes asíncronos con identificadores, nunca secretos.
- `FOR UPDATE SKIP LOCKED` para reclamar trabajo.

### Adaptado

- La derivación de Neuraclinic depende del ID. Cestia incluye una **generación**
  para invalidar enlaces al reenviar.
- La cola y el worker viven en el mismo proceso y PostgreSQL; no se usan
  RabbitMQ, gRPC entre servicios ni resolutores remotos.
- Mailpit es el transporte local y de pruebas.

### Descartado

- Infraestructura clínica, multitenencia, gRPC y microservicios.
- RabbitMQ, outbox distribuido y servicios de notificaciones separados.
- Branding y componentes clínicos.

## Toolchain verificada

```text
Go 1.27.1
Node v24.14.0
pnpm 10.15.0 (invocado mediante el Node indicado por el usuario)
```

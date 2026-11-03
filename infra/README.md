# Infraestructura

Cestia producción vive en una sola instancia Amazon Lightsail reemplazable.
La información no depende de esa instancia:

- **PostgreSQL 17** se ejecuta en un **disco Lightsail adjunto** (`cestia-data`),
  que sobrevive al borrado de la instancia.
- **Los respaldos de PostgreSQL** viven en un bucket S3 privado, versionado y
  cifrado.
- La **contraseña de la base de datos** y el **secreto de sesiones** también
  viven en el disco adjunto, para que una instancia nueva reutilice la misma
  identidad.

El subdominio `cestia.chelalo.me` y los registros de SES se crean en la hosted
zone existente `chelalo.me` de Route 53. La región es `us-east-1` y Terraform
usa el perfil `default` de AWS.

## Topología

```text
Internet
   |
   +--> cestia.chelalo.me --(Route 53 A)--> IP estática Lightsail
                                               |
                                               +-> Caddy :443
                                                     +-> /api/*  -> API Go :8000 (systemd)
                                                     +-> resto   -> SPA /opt/cestia/current/web
                                               |
                                               +-> PostgreSQL 17 en /mnt/cestia-data (disco adjunto)
                                               +-> SES SMTP -> correos transaccionales
                                               +-> S3: respaldos pg_dump diarios
```

El disco y el bucket tienen `prevent_destroy`. La instancia se puede reemplazar.

## Costo estimado

| Recurso | Estimado |
|---|---|
| Lightsail `nano_3_0` (512 MB, IPv4 público) | USD 5/mes |
| Disco 8 GB (`data_disk_size_gb = 8`) | USD 0.80/mes |
| S3 respaldos (uso pequeño) | < USD 0.50/mes |
| SES (hasta 3 000 mensajes/mes desde EC2/Lightsail) | USD 0 |
| Route 53 (hosted zone ya existente) | USD 0.50/mes |

Total aproximado: **USD 6.80/mes** antes de impuestos. Revisa el precio vigente
de la región antes de aplicar. AWS Budget solo alerta; no limita el gasto.

## Cuenta y perfil

1. Usa la cuenta personal de AWS con el perfil **`default`**
   (`AWS_PROFILE=default`, `aws sts get-caller-identity`).
2. `expected_aws_account_id` (12 dígitos) hace fallar el `plan` si las
   credenciales apuntan a otra cuenta.
3. Tras el primer `apply`, activa en **Billing > Cost allocation tags** las
   claves `Organization`, `Project`, `Environment`, `CostCenter` y `ManagedBy`.

Todas las etiquetas comunes se definen en `local.common_tags`:
`Organization=chelalo`, `Project=cestia`, `Environment=production`,
`CostCenter=personal-cestia`, `ManagedBy=terraform`.

## Primer despliegue

Detalle completo en
[`docs/runbooks/production-lightsail.md`](../docs/runbooks/production-lightsail.md).

```bash
# 1. Backend remoto de Terraform
cd infra/bootstrap
cp terraform.tfvars.example terraform.tfvars   # ajusta cuenta y bucket
terraform init -backend=false
terraform plan -out=tfplan
terraform apply tfplan
cp backend.s3.tf.example backend.tf
terraform init -migrate-state -backend-config=...

# 2. Infraestructura de producción
cd ../lightsail
cp terraform.tfvars.example terraform.tfvars   # ajusta cuenta, AZ, key pair, email
terraform init -backend-config=...
terraform fmt -check
terraform validate
terraform plan -out=tfplan
terraform apply tfplan
```

3. Configura el registro `A` (lo crea Terraform cuando `manage_dns = true`).
4. Copia el repositorio a la instancia y ejecuta `bootstrap_command`.
5. Crea las access keys de `runtime_iam_user` y `ses_smtp_user`; rellena
   `/etc/cestia/aws.env` y `/etc/cestia/smtp.env` en la instancia (nunca en Git).
6. Los despliegues posteriores los hace GitHub Actions al integrar en `main`.

## GitHub Actions

Configura en el repositorio:

| Secreto | Valor |
|---|---|
| `AWS_DEPLOY_ROLE_ARN` | output `github_deploy_role_arn` |
| `SSH_PRIVATE_KEY` | llave privada del key pair `cestia-deploy` |

Opcional: variable `LIGHTSAIL_INSTANCE_NAME` (default `cestia-app`).

El workflow abre el puerto 22 solo para la IP del runner, despliega y lo cierra
en un paso `if: always()`.

## Consultas locales

```bash
terraform -chdir=infra/bootstrap fmt -check
terraform -chdir=infra/bootstrap init -backend=false && terraform -chdir=infra/bootstrap validate
terraform -chdir=infra/lightsail fmt -check
terraform -chdir=infra/lightsail init -backend=false && terraform -chdir=infra/lightsail validate
pnpm nx run infra:lint
pnpm nx run infra:test   # requiere Docker
```

## Límites y migración futura

- El disco protege contra borrar la instancia, pero PostgreSQL se detiene hasta
  que una instancia nueva lo reanexa. Los respaldos en S3 cubren la pérdida del
  disco con un RPO diario.
- `deploy-release.sh` no revierte migraciones de base de datos. Toda migración
  debe ser compatible con el release anterior.
- La API lee `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASS`,
  `DB_SSLMODE` y `DB_SSLROOTCERT`; `DATABASE_URL` sigue siendo un override
  explícito. Para migrar a RDS solo cambia `/etc/cestia/db.env` a
  `DB_SSLMODE=verify-full` y `DB_SSLROOTCERT=/etc/cestia/rds-ca.pem`.
- El disco y el bucket tienen `prevent_destroy`: un `terraform destroy` falla a
  propósito.

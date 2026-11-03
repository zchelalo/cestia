# Producción en Lightsail

Runbook operativo de Cestia en una sola instancia Lightsail con PostgreSQL en
disco adjunto, respaldos en S3, correo por SES y despliegues desde GitHub
Actions. La topología y el costo están en [`infra/README.md`](../../infra/README.md).

## 1. Requisitos previos

- Cuenta AWS personal con el perfil **`default`** (`AWS_PROFILE=default`).
- Permisos para crear Lightsail, S3, IAM, SES, Route 53 y Budgets.
- `terraform >= 1.10`, `aws`, `pnpm`, `node`, `go` y un par de llaves Lightsail.
- La hosted zone `chelalo.me` ya existe en Route 53.

Acceso SSH: crea el key pair `cestia-deploy` en Lightsail antes del `apply` y
pasa su nombre en `key_pair_name`. La instancia no abre el puerto 22 por defecto
(`ssh_allowed_cidrs = []`); el workflow de despliegue lo abre solo para la IP del
runner y lo cierra siempre al terminar.

Comprueba la cuenta antes de tocar Terraform:

```bash
export AWS_PROFILE=default
aws sts get-caller-identity
```

## 2. Backend de estado

```bash
cd infra/bootstrap
cp terraform.tfvars.example terraform.tfvars
# expected_aws_account_id y state_bucket_name reales
terraform init -backend=false
terraform plan -out=tfplan
terraform apply tfplan

cp backend.s3.tf.example backend.tf
terraform init -migrate-state \
  -backend-config="bucket=<state_bucket>" \
  -backend-config="key=bootstrap/terraform.tfstate" \
  -backend-config="region=us-east-1" \
  -backend-config="encrypt=true"
terraform state list
```

`use_lockfile = true` guarda el bloqueo como `<key>.tflock` en el mismo bucket
versionado. No uses `dynamodb_table`.

## 3. Infraestructura de producción

```bash
cd infra/lightsail
cp terraform.tfvars.example terraform.tfvars
# expected_aws_account_id, key_pair_name, availability_zone, alert_email
terraform init \
  -backend-config="bucket=<state_bucket>" \
  -backend-config="key=production/lightsail/terraform.tfstate" \
  -backend-config="region=us-east-1" \
  -backend-config="encrypt=true"
terraform fmt -check
terraform validate
terraform plan -out=tfplan
terraform apply tfplan
```

Revisa en el `plan` que el disco y el bucket tengan `prevent_destroy` y que solo
se abran 80/443. Toma nota de:

- `static_ip` → registro `A` de `cestia.chelalo.me` (Terraform lo crea).
- `backups_bucket`.
- `runtime_iam_user` y `ses_smtp_user`.
- `github_deploy_role_arn`.

## 4. Preparar la instancia

El bootstrap es idempotente y no borra datos. Descarga los scripts en `/tmp` y
ejecuta el comando con el entorno de `terraform output bootstrap_command`,
sustituyendo la ruta del backup y la del script:

```bash
scp -i <key>.pem \
  infra/lightsail/scripts/bootstrap-instance.sh \
  infra/lightsail/runtime/scripts/backup-postgres.sh \
  ubuntu@<static_ip>:/tmp/
ssh -i <key>.pem ubuntu@<static_ip>

# En la instancia:
ls /tmp/bootstrap-instance.sh /tmp/backup-postgres.sh

CESTIA_APP_DOMAIN='cestia.chelalo.me' \
CESTIA_ENVIRONMENT='production' \
CESTIA_AWS_REGION='us-east-1' \
CESTIA_BACKUP_BUCKET='<backups_bucket>' \
CESTIA_TLS_EMAIL='<tls_email>' \
CESTIA_DATA_DISK_DEVICE='/dev/xvdf' \
CESTIA_BACKUP_SCRIPT='/tmp/backup-postgres.sh' \
  sudo -E bash /tmp/bootstrap-instance.sh
```

El script:

1. crea 1 GiB de swap (la instancia más barata tiene 512 MB de RAM);
2. instala PostgreSQL 17, Caddy y AWS CLI;
3. monta el disco por etiqueta `cestia-data` y lo agrega a `/etc/fstab`;
4. niega el arranque de PostgreSQL si el disco no está montado;
5. crea la base `cestia` y los roles `cestia_migrate` y `cestia_app`;
6. genera contraseñas y `TOKEN_SECRET` **en el disco**, y escribe
   `/etc/cestia/{api,db,smtp,aws,backup}.env`;
7. habilita `cestia-postgres`, `caddy`, `cestia-api` y el timer diario.

Verifica:

```bash
systemctl is-active cestia-postgres caddy
sudo -u postgres psql -d cestia -c "SELECT version FROM schema_migrations;"
```

## 5. Credenciales de S3 y SES

Terraform crea los usuarios IAM pero nunca sus access keys.

```bash
aws iam create-access-key --user-name cestia-production-runtime
aws iam create-access-key --user-name cestia-production-smtp
```

En la instancia rellena `/etc/cestia/aws.env` con la access key de runtime y
`/etc/cestia/smtp.env` con las credenciales SMTP de SES, y reinicia:

```bash
sudo systemctl restart cestia-api
sudo systemctl start cestia-pg-backup.timer
```

Las credenciales SMTP de SES **no** son la access key y el secret directamente:
usa la consola de SES (SMTP settings) o el algoritmo documentado por AWS para
derivarlas. Nunca las guardes en Git ni en Terraform.

### Salir del sandbox de SES

Las cuentas nuevas solo envían a direcciones verificadas. Verifica los registros
DKIM y MAIL FROM en la consola de SES y solicita producción:

```bash
aws sesv2 put-account-details \
  --mail-type TRANSACTIONAL \
  --website-url https://cestia.chelalo.me \
  --contact-language EN \
  --production-access-enabled
```

## 6. Configurar GitHub Actions

En el repositorio, crea el environment **`production`** y los secretos:

| Secreto | Valor |
|---|---|
| `AWS_DEPLOY_ROLE_ARN` | output `github_deploy_role_arn` |
| `SSH_PRIVATE_KEY` | llave privada del key pair `cestia-deploy` |

Variables opcionales: `APP_DOMAIN` (`cestia.chelalo.me`),
`LIGHTSAIL_INSTANCE_NAME` (`cestia-app`) y `LIGHTSAIL_STATIC_IP_NAME`
(`cestia-app-ip`).

El workflow `Deploy` se ejecuta al integrar en `main` o a mano. Compila los
binarios y la SPA en el runner, asume el rol por OIDC, abre el 22 solo para la
IP del runner, despliega y cierra el 22 en un paso `if: always()`.

## 7. Desplegar un release manualmente

Si necesitas desplegar sin Actions:

```bash
pnpm nx run api:build
pnpm nx run web:build
release_id=$(git rev-parse --short=12 HEAD)
staging=$(mktemp -d)
cp apps/api/bin/cestia-api apps/api/bin/cestia-migrate "$staging/"
cp infra/lightsail/runtime/scripts/restore-postgres.sh "$staging/"
cp infra/lightsail/runtime/scripts/backup-postgres.sh "$staging/"
cp -r apps/web/dist "$staging/web"
tar -czf "/tmp/cestia-${release_id}.tar.gz" -C "$staging" .
rm -rf "$staging"
```

Abre el 22 temporalmente desde tu IP, sube el tar y el script, y ejecuta
`deploy-release.sh`. Cierra el 22 al terminar.

`deploy-release.sh` ejecuta migraciones, cambia el symlink `current`, reinicia
la API, comprueba `/api/health` y revierte al release anterior si falla. **Las
migraciones no se revierten automáticamente**: deben ser compatibles con el
release anterior.

## 8. Verificación en línea

```bash
curl -fsS https://cestia.chelalo.me/api/health
curl -fsSI https://cestia.chelalo.me/
```

Comprueba en el navegador un registro, el correo de verificación en SES y una
compra personal.

### Diagnóstico TLS

```bash
sudo systemctl status caddy --no-pager
sudo journalctl -u caddy --since "2 hours ago" --no-pager | grep -iE 'acme|tls|certificate|error'
```

Si el certificado ya existe pero Let's Encrypt respondió de forma transitoria,
fuerza un reintento con `systemctl reload caddy`; no reinicies en bucle.

## 9. Respaldos y restauración

El timer `cestia-pg-backup.timer` ejecuta el respaldo a las 04:15 UTC y sube
`s3://<backups_bucket>/postgres/cestia-<timestamp>.dump` con su manifiesto JSON.
El ciclo de vida expira los objetos después de `backup_retention_days`.

```bash
systemctl list-timers cestia-pg-backup.timer
sudo systemctl start cestia-pg-backup.service
aws s3 ls s3://<backups_bucket>/postgres/ --human-readable
```

**No hay alarma externa.** Revisa a diario que el timer siga activo, que el
último servicio terminó en éxito y que existe un objeto reciente:

```bash
systemctl is-active cestia-pg-backup.timer
systemctl show cestia-pg-backup.service -p Result
journalctl -u cestia-pg-backup.service --since '2 days ago' --no-pager | tail
```

Validar un respaldo sin modificar nada:

```bash
sudo /usr/local/sbin/cestia-restore-postgres \
  s3://<backups_bucket>/postgres/cestia-<timestamp>.dump
```

Reemplazar la base productiva (recuperación real):

```bash
sudo /usr/local/sbin/cestia-restore-postgres --replace-production \
  s3://<backups_bucket>/postgres/cestia-<timestamp>.dump
```

Tras verificar, revisa `cestia_previous` y elimínala:

```bash
sudo -u postgres psql -c 'DROP DATABASE cestia_previous'
```

El script mantiene el lock `/run/lock/cestia-maintenance.lock` compartido con el
deploy, de modo que una restauración nunca corre a la vez que una migración o un
respaldo.

## 10. Recrear la instancia

1. `terraform apply` recrea la instancia, conserva la IP y mantiene el disco.
2. Reaplica el bootstrap; reutiliza contraseñas y `TOKEN_SECRET` del disco.
3. Despliega el último release.
4. Rellena `/etc/cestia/aws.env` y `/etc/cestia/smtp.env` con credenciales nuevas
   y rota las anteriores.
5. Verifica `/api/health` y un login.

## 11. Migrar PostgreSQL a RDS (futuro)

La API admite TLS verificado y no requiere cambios de código:

1. Crea una instancia RDS PostgreSQL privada con `publicly_accessible = false`
   y `deletion_protection`.
2. Abre 5432 solo desde la instancia Lightsail.
3. En una ventana de mantenimiento, detén escrituras
   (`systemctl stop cestia-api`) y genera un dump con `pg_dump -Fc`.
4. Restaura el dump en RDS con los roles `cestia_migrate` y `cestia_app`.
5. Cambia `/etc/cestia/db.env` a `DB_HOST=<rds-endpoint>`,
   `DB_SSLMODE=verify-full` y `DB_SSLROOTCERT=/etc/cestia/rds-ca.pem`.
6. Reinicia la API y verifica `/api/health` y un flujo real. **Conserva el disco
   y el dump** hasta confirmar el cambio.

`DATABASE_URL` sigue siendo un override explícito (lo usa el migrador); el resto
de la aplicación lee las variables `DB_*`.

## 12. Costos y cuenta

- El Budget avisa al 50/80/100 %. No bloquea el gasto.
- Activa las etiquetas de costos en Billing después del primer `apply`.
- Revisa el costo total (Lightsail + disco + S3 + transferencia), no solo la VM.

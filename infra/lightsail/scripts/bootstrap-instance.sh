#!/usr/bin/env bash
# Cestia Lightsail bootstrapper — idempotent first-run setup (run as root).
#
# PostgreSQL data and the generated database and session secrets live on the
# attached Lightsail data disk, never on the instance root volume. The instance
# can be destroyed and recreated; this script reattaches the existing disk and
# reuses the persisted secrets.
#
# Required environment (see `terraform output bootstrap_command`):
#   CESTIA_APP_DOMAIN       production hostname, e.g. cestia.chelalo.me
#   CESTIA_ENVIRONMENT      e.g. production
#   CESTIA_AWS_REGION       e.g. us-east-1
#   CESTIA_BACKUP_BUCKET    PostgreSQL backups bucket
# Optional:
#   CESTIA_TLS_EMAIL        ACME contact email
#   CESTIA_DATA_DISK_DEVICE defaults to /dev/xvdf
#   CESTIA_SMTP_HOST        defaults to email-smtp.<region>.amazonaws.com
#   CESTIA_SMTP_PORT        defaults to 587 (STARTTLS)
#   CESTIA_SMTP_FROM        defaults to Cestia <no-reply@<app_domain>>

if [ -z "${BASH_VERSION:-}" ]; then
  exec /usr/bin/env bash "$0" "$@"
fi

set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "bootstrap-instance.sh must run as root" >&2
  exit 1
fi

: "${CESTIA_APP_DOMAIN:?missing CESTIA_APP_DOMAIN}"
: "${CESTIA_ENVIRONMENT:?missing CESTIA_ENVIRONMENT}"
: "${CESTIA_AWS_REGION:?missing CESTIA_AWS_REGION}"
: "${CESTIA_BACKUP_BUCKET:?missing CESTIA_BACKUP_BUCKET}"

# The backup script ships with the repository next to this bootstrap script, or
# is named by CESTIA_BACKUP_SCRIPT. Resolve it before any package, PostgreSQL or
# systemd change so a missing file fails without touching the instance.
backup_source="${CESTIA_BACKUP_SCRIPT:-$(dirname "$(readlink -f "$0")")/../runtime/scripts/backup-postgres.sh}"
if [ ! -f "${backup_source}" ]; then
  echo "backup-postgres.sh not found at ${backup_source}" >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

PG_VERSION="17"
PG_BIN="/usr/lib/postgresql/${PG_VERSION}/bin"
DATA_MOUNT="/mnt/cestia-data"
DATA_ROOT="${DATA_MOUNT}/cestia"
PGDATA="${DATA_ROOT}/postgres"
PG_CONF="${DATA_ROOT}/postgresql.conf"
PG_HBA="${DATA_ROOT}/pg_hba.conf"
SECRETS_DIR="${DATA_ROOT}/secrets"
DB_NAME="cestia"
MIGRATE_ROLE="cestia_migrate"
APP_ROLE="cestia_app"
DISK_DEVICE="${CESTIA_DATA_DISK_DEVICE:-/dev/xvdf}"
DISK_LABEL="cestia-data"
SMTP_HOST="${CESTIA_SMTP_HOST:-email-smtp.${CESTIA_AWS_REGION}.amazonaws.com}"
SMTP_PORT="${CESTIA_SMTP_PORT:-587}"
SMTP_FROM="${CESTIA_SMTP_FROM:-Cestia <no-reply@${CESTIA_APP_DOMAIN}>}"

log() { printf '[cestia-bootstrap] %s\n' "$*"; }

# ---------------------------------------------------------------------------
# 1. Swap first: the cheapest Lightsail bundle has 512 MB of RAM and both
#    PostgreSQL and Caddy need headroom during dumps and TLS renewals.
# ---------------------------------------------------------------------------
if ! swapon --show=NAME --noheadings 2>/dev/null | grep -q '/swapfile'; then
  if [ ! -f /swapfile ]; then
    log "creating a 1 GiB swap file on the root volume"
    if ! fallocate -l 1G /swapfile 2>/dev/null; then
      dd if=/dev/zero of=/swapfile bs=1M count=1024 status=none
    fi
    chmod 0600 /swapfile
    mkswap /swapfile >/dev/null
  fi
  swapon /swapfile
fi
if ! grep -q '^/swapfile ' /etc/fstab; then
  echo "/swapfile none swap sw 0 0" >>/etc/fstab
fi
cat >/etc/sysctl.d/99-cestia.conf <<'EOF'
vm.swappiness = 10
EOF
sysctl --system >/dev/null 2>&1 || true

# ---------------------------------------------------------------------------
# 2. Packages
# ---------------------------------------------------------------------------
apt-get update -qq
apt-get install -y -qq \
  apt-transport-https \
  ca-certificates \
  curl \
  e2fsprogs \
  gnupg \
  jq \
  openssl \
  unzip \
  xz-utils

if ! command -v aws >/dev/null 2>&1; then
  curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" -o /tmp/awscliv2.zip
  unzip -q /tmp/awscliv2.zip -d /tmp/awscliv2
  /tmp/awscliv2/aws/install --update
  rm -rf /tmp/awscliv2 /tmp/awscliv2.zip
fi

if [ ! -x "${PG_BIN}/postgres" ]; then
  install -d -m 0755 /usr/share/postgresql-common/pgdg
  curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
    -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc
  # shellcheck disable=SC1091
  . /etc/os-release
  echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt ${VERSION_CODENAME}-pgdg main" \
    >/etc/apt/sources.list.d/pgdg.list
  apt-get update -qq
  # The server package pulls the matching client.
  apt-get install -y -qq "postgresql-${PG_VERSION}"
fi

if ! command -v caddy >/dev/null 2>&1; then
  apt-get install -y -qq debian-keyring debian-archive-keyring
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' |
    gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' |
    tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
  apt-get update -qq
  apt-get install -y -qq caddy
fi

# The distribution cluster would own port 5432 on the root disk; Cestia runs its
# own cluster on the data disk instead.
systemctl disable --now postgresql >/dev/null 2>&1 || true
if command -v pg_dropcluster >/dev/null 2>&1; then
  pg_dropcluster --stop "${PG_VERSION}" main >/dev/null 2>&1 || true
fi

# ---------------------------------------------------------------------------
# 3. Data disk: persistent PostgreSQL and secrets
# ---------------------------------------------------------------------------
# Lightsail may expose the attached disk as NVMe (e.g. /dev/nvme1n1) instead of
# the requested /dev/xvdf, so discover the non-root block device when the
# configured path does not exist.
resolve_data_disk() {
  local candidate root_source root_disk
  if [ -b "${DISK_DEVICE}" ]; then
    printf '%s' "${DISK_DEVICE}"
    return 0
  fi
  root_source="$(findmnt -n -o SOURCE / 2>/dev/null || true)"
  root_disk="$(lsblk -no PKNAME "${root_source}" 2>/dev/null | head -1 || true)"
  for candidate in /dev/nvme1n1 /dev/nvme2n1 /dev/nvme3n1 /dev/xvdf /dev/xvdb /dev/sdf; do
    [ -b "${candidate}" ] || continue
    [ "$(basename "${candidate}")" = "${root_disk}" ] && continue
    [ "$(lsblk -no TYPE "${candidate}" 2>/dev/null)" = "disk" ] || continue
    printf '%s' "${candidate}"
    return 0
  done
  return 1
}

if ! blkid -L "${DISK_LABEL}" >/dev/null 2>&1; then
  data_device="$(resolve_data_disk)" || {
    echo "could not locate the attached data disk and no ${DISK_LABEL} filesystem exists" >&2
    exit 1
  }
  if blkid "${data_device}" >/dev/null 2>&1; then
    echo "refusing to format ${data_device}: it already contains a filesystem without the ${DISK_LABEL} label" >&2
    exit 1
  fi
  log "formatting ${data_device} as ext4 label ${DISK_LABEL}"
  mkfs.ext4 -F -L "${DISK_LABEL}" "${data_device}"
fi

install -d -m 0755 "${DATA_MOUNT}"
if ! mountpoint -q "${DATA_MOUNT}"; then
  mount -L "${DISK_LABEL}" "${DATA_MOUNT}"
fi
if ! grep -q "LABEL=${DISK_LABEL}" /etc/fstab; then
  echo "LABEL=${DISK_LABEL} ${DATA_MOUNT} ext4 defaults,nofail 0 2" >>/etc/fstab
fi

if ! getent group postgres >/dev/null; then
  groupadd --system postgres
fi
if ! id postgres >/dev/null 2>&1; then
  useradd --system --gid postgres --home-dir /var/lib/postgresql --shell /usr/sbin/nologin postgres
fi
if ! getent group cestia >/dev/null; then
  groupadd --system cestia
fi
if ! id cestia >/dev/null 2>&1; then
  useradd --system --gid cestia --home-dir /opt/cestia --shell /usr/sbin/nologin cestia
fi

install -d -o root -g root -m 0755 "${DATA_ROOT}"
install -d -o postgres -g postgres -m 0700 "${PGDATA}"
install -d -o root -g root -m 0700 "${SECRETS_DIR}"

# ---------------------------------------------------------------------------
# 4. PostgreSQL cluster on the data disk
# ---------------------------------------------------------------------------
if [ ! -s "${PGDATA}/PG_VERSION" ]; then
  log "initializing PostgreSQL ${PG_VERSION} cluster on the data disk"
  runuser -u postgres -- "${PG_BIN}/initdb" \
    -D "${PGDATA}" \
    --auth-local=peer \
    --auth-host=scram-sha-256 \
    --encoding=UTF8 \
    --locale=C
fi

cat >"${PG_CONF}" <<EOF
# Managed by infra/lightsail/scripts/bootstrap-instance.sh. Do not edit by hand.
listen_addresses = 'localhost'
port = 5432
unix_socket_directories = '/var/run/postgresql'
hba_file = '${PG_HBA}'
password_encryption = 'scram-sha-256'
shared_buffers = '64MB'
effective_cache_size = '192MB'
maintenance_work_mem = '32MB'
work_mem = '2MB'
max_connections = 25
wal_level = 'replica'
EOF

cat >"${PG_HBA}" <<EOF
# Managed by infra/lightsail/scripts/bootstrap-instance.sh.
local   all             postgres                                peer
local   all             all                                     peer
host    all             ${MIGRATE_ROLE}         127.0.0.1/32    scram-sha-256
host    all             ${APP_ROLE}             127.0.0.1/32    scram-sha-256
host    all             all                     127.0.0.1/32    reject
host    all             all                     ::1/128         reject
EOF
chown postgres:postgres "${PG_CONF}" "${PG_HBA}"
chmod 0600 "${PG_CONF}" "${PG_HBA}"

cat >/usr/local/sbin/cestia-postgres-guard.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if ! mountpoint -q /mnt/cestia-data; then
  echo "cestia data disk is not mounted at /mnt/cestia-data; refusing to start PostgreSQL" >&2
  exit 1
fi
if [ ! -s /mnt/cestia-data/cestia/postgres/PG_VERSION ]; then
  echo "PostgreSQL data directory is missing on the data disk; refusing to start an empty cluster" >&2
  exit 1
fi
EOF
chmod 0755 /usr/local/sbin/cestia-postgres-guard.sh

cat >/etc/systemd/system/cestia-postgres.service <<EOF
[Unit]
Description=Cestia PostgreSQL ${PG_VERSION}
After=network-online.target
Wants=network-online.target
RequiresMountsFor=${DATA_MOUNT}

[Service]
Type=notify
User=postgres
Group=postgres
RuntimeDirectory=postgresql
RuntimeDirectoryMode=2775
ExecStartPre=/usr/local/sbin/cestia-postgres-guard.sh
ExecStart=${PG_BIN}/postgres -D ${PGDATA} -c config_file=${PG_CONF}
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=mixed
KillSignal=SIGINT
TimeoutStopSec=300

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now cestia-postgres.service

for _ in $(seq 1 60); do
  if runuser -u postgres -- "${PG_BIN}/pg_isready" -h /var/run/postgresql -q; then
    break
  fi
  sleep 1
done
if ! runuser -u postgres -- "${PG_BIN}/pg_isready" -h /var/run/postgresql -q; then
  echo "PostgreSQL did not become ready" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# 5. Generated secrets (persisted on the data disk)
# ---------------------------------------------------------------------------
SECRETS_FILE="${SECRETS_DIR}/postgres.env"
if [ ! -f "${SECRETS_FILE}" ]; then
  log "generating database passwords on the data disk"
  umask 077
  cat >"${SECRETS_FILE}" <<EOF
CESTIA_MIGRATE_PASSWORD=$(openssl rand -hex 24)
CESTIA_APP_PASSWORD=$(openssl rand -hex 24)
EOF
fi
chmod 0600 "${SECRETS_FILE}"
# shellcheck disable=SC1090
. "${SECRETS_FILE}"

APP_SECRETS_FILE="${SECRETS_DIR}/app.env"
if [ ! -f "${APP_SECRETS_FILE}" ]; then
  log "generating the session secret on the data disk"
  umask 077
  cat >"${APP_SECRETS_FILE}" <<EOF
TOKEN_SECRET=$(openssl rand -hex 32)
EOF
fi
chmod 0600 "${APP_SECRETS_FILE}"
# shellcheck disable=SC1090
. "${APP_SECRETS_FILE}"
: "${TOKEN_SECRET:?missing TOKEN_SECRET on the data disk}"

# ---------------------------------------------------------------------------
# 6. Roles, database and grants
# ---------------------------------------------------------------------------
psql_postgres() {
  runuser -u postgres -- "${PG_BIN}/psql" -v ON_ERROR_STOP=1 -q "$@"
}

psql_postgres <<EOF
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${MIGRATE_ROLE}') THEN
    CREATE ROLE ${MIGRATE_ROLE} LOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${APP_ROLE}') THEN
    CREATE ROLE ${APP_ROLE} LOGIN;
  END IF;
END
\$\$;
ALTER ROLE ${MIGRATE_ROLE} PASSWORD '${CESTIA_MIGRATE_PASSWORD}';
ALTER ROLE ${APP_ROLE} PASSWORD '${CESTIA_APP_PASSWORD}';
EOF

if ! psql_postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}'" | grep -q 1; then
  psql_postgres -c "CREATE DATABASE ${DB_NAME} OWNER ${MIGRATE_ROLE}"
fi

psql_postgres -d "${DB_NAME}" <<EOF
ALTER DATABASE ${DB_NAME} OWNER TO ${MIGRATE_ROLE};
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT CONNECT ON DATABASE ${DB_NAME} TO ${APP_ROLE};
GRANT ALL ON SCHEMA public TO ${MIGRATE_ROLE};
GRANT USAGE ON SCHEMA public TO ${APP_ROLE};
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ${APP_ROLE};
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ${APP_ROLE};
ALTER DEFAULT PRIVILEGES FOR ROLE ${MIGRATE_ROLE} IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ${APP_ROLE};
ALTER DEFAULT PRIVILEGES FOR ROLE ${MIGRATE_ROLE} IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO ${APP_ROLE};
EOF

# ---------------------------------------------------------------------------
# 7. Runtime directories and configuration
# ---------------------------------------------------------------------------
install -d -o cestia -g cestia -m 0755 /opt/cestia
install -d -o cestia -g cestia -m 0755 /opt/cestia/releases
install -d -o root -g root -m 0755 /etc/cestia
install -d -o caddy -g caddy -m 0750 /var/log/caddy

if [ ! -f /etc/cestia/api.env ]; then
  cat >/etc/cestia/api.env <<EOF
APP_ENV=${CESTIA_ENVIRONMENT}
HTTP_ADDR=127.0.0.1:8000
APP_ORIGIN=https://${CESTIA_APP_DOMAIN}
COOKIE_SECURE=true
SESSION_TTL=336h
TOKEN_SECRET=${TOKEN_SECRET}
REGISTER_RATE_LIMIT=10
EOF
fi
chown root:cestia /etc/cestia/api.env
chmod 0640 /etc/cestia/api.env

cat >/etc/cestia/db.env <<EOF
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=${DB_NAME}
DB_USER=${APP_ROLE}
DB_PASS=${CESTIA_APP_PASSWORD}
DB_SSLMODE=disable
EOF
chown root:cestia /etc/cestia/db.env
chmod 0640 /etc/cestia/db.env

if [ ! -f /etc/cestia/smtp.env ]; then
  cat >/etc/cestia/smtp.env <<EOF
# Create an SMTP password for the ${CESTIA_ENVIRONMENT} SES IAM user, then fill
# both SMTP credentials and restart the API. SES requires STARTTLS on 587.
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM="${SMTP_FROM}"
SMTP_TLS=false
SMTP_STARTTLS=true
EOF
fi
chown root:cestia /etc/cestia/smtp.env
chmod 0640 /etc/cestia/smtp.env

if [ ! -f /etc/cestia/aws.env ]; then
  cat >/etc/cestia/aws.env <<'EOF'
# Create the access key for the runtime IAM user after `terraform apply` and
# fill both pairs below, then restart the API and the backup timer.
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
EOF
fi
chown root:cestia /etc/cestia/aws.env
chmod 0640 /etc/cestia/aws.env

if [ ! -f /etc/cestia/backup.env ]; then
  cat >/etc/cestia/backup.env <<EOF
BACKUP_BUCKET=${CESTIA_BACKUP_BUCKET}
AWS_REGION=${CESTIA_AWS_REGION}
EOF
fi
chown root:cestia /etc/cestia/backup.env
chmod 0640 /etc/cestia/backup.env

# ---------------------------------------------------------------------------
# 8. API service
# ---------------------------------------------------------------------------
cat >/etc/systemd/system/cestia-api.service <<'EOF'
[Unit]
Description=Cestia API
After=network-online.target cestia-postgres.service
Wants=network-online.target
Requires=cestia-postgres.service

[Service]
Type=simple
User=cestia
Group=cestia
WorkingDirectory=/opt/cestia/current
EnvironmentFile=/etc/cestia/api.env
EnvironmentFile=-/etc/cestia/db.env
EnvironmentFile=-/etc/cestia/smtp.env
EnvironmentFile=-/etc/cestia/aws.env
ExecStart=/opt/cestia/current/cestia-api
Restart=always
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

# ---------------------------------------------------------------------------
# 9. PostgreSQL backup script and timer
# ---------------------------------------------------------------------------
install -m 0750 "${backup_source}" /usr/local/sbin/cestia-pg-backup.sh

cat >/etc/systemd/system/cestia-pg-backup.service <<'EOF'
[Unit]
Description=Cestia PostgreSQL backup to S3
After=cestia-postgres.service
Requires=cestia-postgres.service

[Service]
Type=oneshot
ExecStart=/usr/local/sbin/cestia-pg-backup.sh
EOF

cat >/etc/systemd/system/cestia-pg-backup.timer <<'EOF'
[Unit]
Description=Daily Cestia PostgreSQL backup to S3

[Timer]
OnCalendar=*-*-* 04:15:00 UTC
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
EOF

# ---------------------------------------------------------------------------
# 10. Caddy reverse proxy
# ---------------------------------------------------------------------------
{
  echo "CESTIA_APP_DOMAIN=${CESTIA_APP_DOMAIN}"
  echo "CESTIA_TLS_EMAIL=${CESTIA_TLS_EMAIL:-}"
} >/etc/caddy/caddy.env
chmod 0644 /etc/caddy/caddy.env

if [ -n "${CESTIA_TLS_EMAIL:-}" ]; then
  cat >/etc/caddy/Caddyfile <<'EOF'
{
	email {$CESTIA_TLS_EMAIL}
}

{$CESTIA_APP_DOMAIN} {
	encode zstd gzip

	header {
		-Server
		X-Content-Type-Options "nosniff"
		Referrer-Policy "strict-origin-when-cross-origin"
	}

	handle /api/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle /assets/* {
		root * /opt/cestia/current/web
		header Cache-Control "public, max-age=31536000, immutable"
		file_server
	}

	handle {
		root * /opt/cestia/current/web
		try_files {path} /index.html
		header Cache-Control "no-cache"
		file_server
	}

	log {
		output file /var/log/caddy/cestia-access.log {
			roll_size 25MiB
			roll_keep 10
			roll_keep_for 720h
		}
		format json
	}
}
EOF
else
  cat >/etc/caddy/Caddyfile <<'EOF'
{$CESTIA_APP_DOMAIN} {
	encode zstd gzip

	header {
		-Server
		X-Content-Type-Options "nosniff"
		Referrer-Policy "strict-origin-when-cross-origin"
	}

	handle /api/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle /assets/* {
		root * /opt/cestia/current/web
		header Cache-Control "public, max-age=31536000, immutable"
		file_server
	}

	handle {
		root * /opt/cestia/current/web
		try_files {path} /index.html
		header Cache-Control "no-cache"
		file_server
	}

	log {
		output file /var/log/caddy/cestia-access.log {
			roll_size 25MiB
			roll_keep 10
			roll_keep_for 720h
		}
		format json
	}
}
EOF
fi

install -d -m 0755 /etc/systemd/system/caddy.service.d
cat >/etc/systemd/system/caddy.service.d/cestia.conf <<'EOF'
[Service]
EnvironmentFile=/etc/caddy/caddy.env
EOF

systemctl daemon-reload
systemctl enable caddy cestia-api.service cestia-pg-backup.timer >/dev/null
systemctl restart caddy
# Start the timer now, not only at boot, so a recreated instance resumes backups
# without waiting for a reboot. The first run needs AWS credentials in
# /etc/cestia/aws.env; the runbook covers the verification backup.
systemctl start cestia-pg-backup.timer

log "bootstrap complete"
log "next: fill /etc/cestia/aws.env and /etc/cestia/smtp.env, then restart cestia-api"
log "then deploy a release with infra/lightsail/runtime/scripts/deploy-release.sh"

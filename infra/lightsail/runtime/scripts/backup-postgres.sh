#!/usr/bin/env bash
# Cestia PostgreSQL backup to S3 (run as root, normally from a systemd timer).
#
# Writes a custom-format dump and a JSON manifest (size, latest schema migration
# version and SHA-256) next to it, under the same S3 prefix and lifecycle. The
# restore script uses the manifest to validate the dump and estimate its
# restored size.
#
# Paths and helpers can be overridden for testing; production uses the defaults.
set -Eeuo pipefail

if [[ "${EUID}" -ne 0 ]] && [ -z "${CESTIA_TEST_ALLOW_NONROOT:-}" ]; then
  echo "cestia-pg-backup.sh must run as root" >&2
  exit 1
fi

PG_BIN="${CESTIA_PG_BIN:-/usr/lib/postgresql/17/bin}"
ENV_DIR="${CESTIA_ENV_DIR:-/etc/cestia}"
DB_NAME="${CESTIA_DB_NAME:-cestia}"
PG_RUNAS="${CESTIA_PG_RUNAS-postgres}"
LOCK_FILE="${CESTIA_LOCK_FILE:-/run/lock/cestia-maintenance.lock}"
LOCK_WAIT="${CESTIA_LOCK_WAIT:-300}"

set -a
# shellcheck disable=SC1091
. "${ENV_DIR}/backup.env"
if [ -f "${ENV_DIR}/aws.env" ]; then
  # shellcheck disable=SC1091
  . "${ENV_DIR}/aws.env"
fi
set +a

: "${BACKUP_BUCKET:?missing BACKUP_BUCKET}"
: "${AWS_REGION:?missing AWS_REGION}"

# Coordinate with deploy and restore so a backup never runs while the database is
# being renamed. Wait a bounded time, then fail loudly (the daily check in the
# runbook will surface it).
install -d -m 0755 "$(dirname "${LOCK_FILE}")"
exec 7>"${LOCK_FILE}"
if ! flock -w "${LOCK_WAIT}" 7; then
  echo "another maintenance operation is running; backup skipped" >&2
  exit 1
fi

pg_run() {
  if [ -n "${PG_RUNAS}" ]; then
    runuser -u "${PG_RUNAS}" -- "$@"
  else
    "$@"
  fi
}

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
work="$(mktemp -d)"
if [ -z "${CESTIA_TEST_ALLOW_NONROOT:-}" ]; then
  chown "${PG_RUNAS}:${PG_RUNAS}" "${work}"
fi
trap 'rm -rf "${work}"' EXIT

dump_file="${work}/${DB_NAME}-${stamp}.dump"
pg_run "${PG_BIN}/pg_dump" \
  -Fc --no-owner --no-privileges -d "${DB_NAME}" -f "${dump_file}"

# A manifest lets the restore estimate the restored size without trusting the
# compressed dump size, and verify the dump it downloads. Cestia records one
# text row per applied migration, so the manifest keeps the latest one.
sha256="$(sha256sum "${dump_file}" | awk '{print $1}')"
database_size="$(pg_run "${PG_BIN}/psql" -tAc "SELECT pg_database_size('${DB_NAME}')" -d "${DB_NAME}")"
migrations_table="$(pg_run "${PG_BIN}/psql" -tAc "SELECT to_regclass('public.schema_migrations')" -d "${DB_NAME}")"
if [ -n "${migrations_table}" ]; then
  migration_version="$(pg_run "${PG_BIN}/psql" -tAc \
    "SELECT COALESCE((SELECT version FROM schema_migrations WHERE version <> '000000_schema_migrations' ORDER BY version DESC LIMIT 1), '000000_schema_migrations')" \
    -d "${DB_NAME}")"
else
  # The timer may run once between bootstrap and the first deployment.
  migration_version="none"
fi
jq -n \
  --arg dump "${DB_NAME}-${stamp}.dump" \
  --argjson database_size_bytes "${database_size}" \
  --arg schema_migration_version "${migration_version}" \
  --arg sha256 "${sha256}" \
  --arg created_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '{dump: $dump, database_size_bytes: $database_size_bytes, schema_migration_version: $schema_migration_version, sha256: $sha256, created_at: $created_at}' \
  >"${work}/manifest.json"

aws s3 cp \
  "${dump_file}" \
  "s3://${BACKUP_BUCKET}/postgres/${DB_NAME}-${stamp}.dump" \
  --region "${AWS_REGION}" \
  --only-show-errors

aws s3 cp \
  "${work}/manifest.json" \
  "s3://${BACKUP_BUCKET}/postgres/${DB_NAME}-${stamp}.json" \
  --region "${AWS_REGION}" \
  --only-show-errors

echo "uploaded s3://${BACKUP_BUCKET}/postgres/${DB_NAME}-${stamp}.dump"

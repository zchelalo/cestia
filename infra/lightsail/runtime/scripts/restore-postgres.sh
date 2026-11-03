#!/usr/bin/env bash
# Cestia PostgreSQL restore (run as root on the Lightsail instance).
#
# Usage: restore-postgres.sh [--replace-production] <s3-uri|/path/to/cestia-*.dump>
#
# Without --replace-production the script only validates the dump, its manifest
# and the available space, then exits without touching any database. This is the
# safe "restore drill". Replacing the live database requires the explicit flag,
# because the backup may predate the latest writes (RPO) and the previous
# database is kept for inspection.
#
# It restores into a temporary database first, while the API keeps running. Only
# after the temporary database verifies does it stop the API and swap database
# names. Any failure from that point rolls back to the original database and is
# verified by database OID, application-role access and API health.
#
# Paths and helpers can be overridden for testing; production uses the defaults.
set -Eeuo pipefail

replace_production=0
source_ref=""
for arg in "$@"; do
  case "${arg}" in
    --replace-production) replace_production=1 ;;
    -*) echo "unknown option: ${arg}" >&2; exit 2 ;;
    *)
      if [ -z "${source_ref}" ]; then
        source_ref="${arg}"
      else
        echo "unexpected argument: ${arg}" >&2
        exit 2
      fi
      ;;
  esac
done
: "${source_ref:?backup s3 URI or local path is required}"

if [[ "${EUID}" -ne 0 ]] && [ -z "${CESTIA_TEST_ALLOW_NONROOT:-}" ]; then
  echo "restore-postgres.sh must run as root" >&2
  exit 1
fi

PG_BIN="${CESTIA_PG_BIN:-/usr/lib/postgresql/17/bin}"
DATA_MOUNT="${CESTIA_DATA_MOUNT:-/mnt/cestia-data}"
ENV_DIR="${CESTIA_ENV_DIR:-/etc/cestia}"
API_SERVICE="${CESTIA_API_SERVICE:-cestia-api.service}"
SYSTEMCTL="${CESTIA_SYSTEMCTL:-systemctl}"
PG_RUNAS="${CESTIA_PG_RUNAS-postgres}"
LOCK_FILE="${CESTIA_LOCK_FILE:-/run/lock/cestia-maintenance.lock}"
HEALTH_URL="${CESTIA_HEALTH_URL:-http://127.0.0.1:8000/api/health}"

DB_NAME="cestia"
PREVIOUS_DB="cestia_previous"
STAMP="$(date -u +%Y%m%d%H%M%S)"
RESTORE_DB="cestia_restore_${STAMP}"

set -a
. "${ENV_DIR}/backup.env"
if [ -f "${ENV_DIR}/aws.env" ]; then
  . "${ENV_DIR}/aws.env"
fi
set +a
. "${DATA_MOUNT}/cestia/secrets/postgres.env"

: "${CESTIA_MIGRATE_PASSWORD:?missing migrate password on the data disk}"

# A single maintenance lock is shared with deploy-release.sh and the backup
# script so a restore never races another database or release operation.
install -d -m 0755 "$(dirname "${LOCK_FILE}")"
exec 8>"${LOCK_FILE}" || {
  echo "cannot open the maintenance lock ${LOCK_FILE}" >&2
  exit 1
}
if ! flock -n 8; then
  echo "another maintenance operation (deploy, restore or backup) is running" >&2
  exit 1
fi

pg_as_postgres() {
  if [ -n "${PG_RUNAS}" ]; then
    runuser -u "${PG_RUNAS}" -- "$@"
  else
    "$@"
  fi
}

psql_postgres() {
  pg_as_postgres "${PG_BIN}/psql" -v ON_ERROR_STOP=1 -q "$@"
}

# Prints present|absent|error and returns 0|1|2. Never treats a query failure as
# an absent database.
db_state() {
  local out
  if ! out="$(pg_as_postgres "${PG_BIN}/psql" -tAc \
    "SELECT 1 FROM pg_database WHERE datname = '$1'" 2>/dev/null)"; then
    printf 'error'
    return 2
  fi
  if [ "${out}" = "1" ]; then
    printf 'present'
    return 0
  fi
  printf 'absent'
  return 1
}

db_present() {
  db_state "$1" >/dev/null 2>&1
  [ $? -eq 0 ]
}

db_oid() {
  pg_as_postgres "${PG_BIN}/psql" -tAc \
    "SELECT oid FROM pg_database WHERE datname = '$1'"
}

terminate_connections() {
  psql_postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
    WHERE datname IN ('${DB_NAME}', '${RESTORE_DB}', '${PREVIOUS_DB}')
      AND pid <> pg_backend_pid()"
}

verify_app_role() {
  set -a
  # shellcheck disable=SC1091
  . "${ENV_DIR}/db.env"
  set +a
  PGPASSWORD="${DB_PASS}" "${PG_BIN}/psql" -h 127.0.0.1 -U cestia_app \
    -d "$1" -tAc "SELECT count(*) FROM users" >/dev/null
}

# Test-only failure injection. Unset in production.
test_fail() {
  if [ -n "${CESTIA_TEST_FAIL_STAGE:-}" ] && [ "${CESTIA_TEST_FAIL_STAGE}" = "$1" ]; then
    echo "test hook: failing at ${1}" >&2
    return 1
  fi
}

# Test-only signal injection. Unset in production.
test_signal() {
  if [ -n "${CESTIA_TEST_SIGNAL_STAGE:-}" ] && [ "${CESTIA_TEST_SIGNAL_STAGE}" = "$1" ]; then
    echo "test hook: sending SIGTERM at ${1}" >&2
    kill -TERM $$
  fi
}

# Drops the temporary database of this attempt only. Returns non-zero when it
# cannot confirm the database is gone (including when PostgreSQL cannot be
# queried), so callers never report a cleanup they did not achieve.
drop_temp_db() {
  local db="$1"
  set +e
  local attempt state rc
  for attempt in $(seq 1 15); do
    state="$(db_state "${db}")"
    rc=$?
    if [ "${rc}" -eq 1 ]; then
      set -e
      return 0
    fi
    if [ "${rc}" -eq 2 ]; then
      set -e
      return 1
    fi
    psql_postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
      WHERE datname = '${db}' AND pid <> pg_backend_pid()" >/dev/null 2>&1
    psql_postgres -c "DROP DATABASE IF EXISTS ${db} WITH (FORCE)" >/dev/null 2>&1
    sleep 1
  done
  state="$(db_state "${db}")"
  rc=$?
  set -e
  [ "${rc}" -eq 1 ]
}

# Best-effort recovery of the original database from whatever swap state was
# reached. It never deletes data, verifies the original OID, and reports whether
# recovery actually succeeded.
rollback_swap() {
  set +e
  trap - ERR INT TERM HUP
  echo "restore failed after the API was stopped; attempting rollback" >&2
  "${SYSTEMCTL}" stop "${API_SERVICE}" 2>/dev/null || true
  terminate_connections 2>/dev/null || true
  if db_present "${PREVIOUS_DB}"; then
    if db_present "${DB_NAME}"; then
      psql_postgres -c "ALTER DATABASE ${DB_NAME} RENAME TO ${RESTORE_DB}" || true
    fi
    psql_postgres -c "ALTER DATABASE ${PREVIOUS_DB} RENAME TO ${DB_NAME}" || true
  fi
  "${SYSTEMCTL}" start "${API_SERVICE}" 2>/dev/null || true

  local ok=1
  if [ -z "${original_oid}" ] || [ "$(db_oid "${DB_NAME}")" != "${original_oid}" ]; then
    ok=0
  fi
  verify_app_role "${DB_NAME}" || ok=0
  curl -fsS --max-time 3 "${HEALTH_URL}" >/dev/null 2>&1 || ok=0

  if [ "${ok}" -eq 1 ]; then
    echo "rollback verified: ${DB_NAME} is the original database and the API is healthy" >&2
  else
    echo "ROLLBACK COULD NOT BE VERIFIED. Databases (name/oid):" >&2
    pg_as_postgres "${PG_BIN}/psql" -tAc \
      "SELECT datname || '/' || oid FROM pg_database WHERE datname LIKE 'cestia%'" >&2
    echo "do not drop any database; recover manually before continuing" >&2
  fi

  # Never delete data during recovery, but never leave a temporary database
  # silently either: report it so the operator can inspect and drop it.
  leftovers="$(pg_as_postgres "${PG_BIN}/psql" -tAc \
    "SELECT datname FROM pg_database WHERE datname LIKE 'cestia_restore%'" | tr '\n' ' ')"
  if [ -n "${leftovers// /}" ]; then
    echo "leftover temporary database(s) kept for inspection: ${leftovers}" >&2
    echo "drop them after verifying: runuser -u postgres -- psql -c 'DROP DATABASE <name>'" >&2
  fi
}

temp_created=0
critical_started=0
original_oid=""
pg_pid=""
aborting=0
on_abort() {
  local status="${1:-$?}"
  if [ "${aborting}" -eq 1 ]; then
    return
  fi
  aborting=1
  set +e
  trap - ERR INT TERM HUP
  # Stop a running pg_restore and wait for it before touching databases.
  if [ -n "${pg_pid}" ] && kill -0 "${pg_pid}" 2>/dev/null; then
    kill "${pg_pid}" 2>/dev/null || true
    wait "${pg_pid}" 2>/dev/null || true
  fi
  if [ "${critical_started}" -eq 1 ]; then
    rollback_swap
  elif [ "${temp_created}" -eq 1 ]; then
    if drop_temp_db "${RESTORE_DB}"; then
      echo "temporary database ${RESTORE_DB} removed" >&2
    else
      echo "could not confirm removal of temporary database ${RESTORE_DB}; remove it manually" >&2
    fi
  fi
  exit "${status}"
}
trap 'on_abort $?' ERR
trap 'on_abort 130' INT
trap 'on_abort 143' TERM
trap 'on_abort 129' HUP

# Private working directory: the dump holds user data and password hashes.
work="$(mktemp -d)"
chmod 0700 "${work}"
trap 'rm -rf "${work}"' EXIT

dump_path="${work}/restore.dump"
case "${source_ref}" in
  s3://*)
    : "${AWS_REGION:?missing AWS_REGION}"
    aws s3 cp "${source_ref}" "${dump_path}" --region "${AWS_REGION}" --only-show-errors
    ;;
  *)
    cp "${source_ref}" "${dump_path}"
    ;;
esac
chmod 0600 "${dump_path}"

# Validate the archive as root: the dump is intentionally unreadable by postgres.
if ! "${PG_BIN}/pg_restore" --list "${dump_path}" >/dev/null 2>&1; then
  if [ ! -r "${dump_path}" ]; then
    echo "the dump file could not be read; aborting" >&2
  else
    echo "the provided file is not a valid pg_restore archive; aborting" >&2
  fi
  exit 1
fi

# A previous rollback copy must be reviewed and removed before a new attempt, so
# this run never deletes the only recovery point.
previous_state="$(db_state "${PREVIOUS_DB}")" || previous_rc=$?
previous_rc="${previous_rc:-0}"
if [ "${previous_rc}" -eq 2 ]; then
  echo "could not query PostgreSQL for ${PREVIOUS_DB}; aborting" >&2
  exit 1
fi
if [ "${previous_state}" = "present" ]; then
  echo "${PREVIOUS_DB} already exists from an earlier restore; review it and drop it" >&2
  echo "  runuser -u postgres -- psql -c 'DROP DATABASE ${PREVIOUS_DB}'" >&2
  exit 1
fi

# The compressed dump size is not a valid estimate of the restored database.
# A manifest, when present, must match this dump: reject a present-but-invalid
# manifest instead of silently falling back. Only old backups without a manifest
# use the conservative live-size estimate.
manifest_ref="${source_ref%.dump}.json"
manifest_path="${work}/manifest.json"
manifest_found=0
case "${manifest_ref}" in
  s3://*)
    manifest_target="${manifest_ref#s3://}"
    manifest_bucket="${manifest_target%%/*}"
    manifest_key="${manifest_target#*/}"
    head_ok=0
    head_err="$(aws s3api head-object \
      --bucket "${manifest_bucket}" --key "${manifest_key}" \
      --region "${AWS_REGION}" 2>&1 >/dev/null)" && head_ok=1
    if [ "${head_ok}" -eq 1 ]; then
      # The manifest exists, so a download failure is an error, not a fallback.
      if ! aws s3 cp "${manifest_ref}" "${manifest_path}" --region "${AWS_REGION}" --only-show-errors; then
        echo "the backup manifest exists but could not be downloaded; refusing to restore" >&2
        exit 1
      fi
      manifest_found=1
    else
      case "${head_err}" in
        *404*|*"Not Found"*|*NoSuchKey*)
          : # legacy backup without a manifest
          ;;
        *)
          echo "could not confirm whether a backup manifest exists; refusing to restore" >&2
          echo "${head_err}" >&2
          exit 1
          ;;
      esac
    fi
    ;;
  *)
    if [ -f "${manifest_ref}" ]; then
      if ! cp "${manifest_ref}" "${manifest_path}"; then
        echo "the backup manifest exists but could not be read; refusing to restore" >&2
        exit 1
      fi
      manifest_found=1
    fi
    ;;
esac

restored_kb=""
if [ "${manifest_found}" -eq 1 ]; then
  if ! command -v jq >/dev/null 2>&1; then
    echo "the backup manifest requires jq, which is not installed; refusing to restore" >&2
    exit 1
  fi
  expected_name="$(jq -r '.dump // empty' "${manifest_path}" 2>/dev/null)"
  expected_size="$(jq -r '.database_size_bytes // empty' "${manifest_path}" 2>/dev/null)"
  expected_sha="$(jq -r '.sha256 // empty' "${manifest_path}" 2>/dev/null)"
  if [ -z "${expected_name}" ] || [ -z "${expected_size}" ] || [ -z "${expected_sha}" ]; then
    echo "the backup manifest is incomplete; refusing to restore" >&2
    exit 1
  fi
  if [ "${expected_name}" != "$(basename "${source_ref}")" ]; then
    echo "the backup manifest does not describe this dump; refusing to restore" >&2
    exit 1
  fi
  actual_sha="$(sha256sum "${dump_path}" | awk '{print $1}')"
  if [ "${actual_sha}" != "${expected_sha}" ]; then
    echo "the dump does not match its manifest checksum; refusing to restore" >&2
    exit 1
  fi
  restored_kb=$((expected_size / 1024))
else
  echo "no backup manifest; using the live database size as a conservative estimate" >&2
  restored_kb="$(pg_as_postgres "${PG_BIN}/psql" -tAc "SELECT pg_database_size('${DB_NAME}') / 1024")"
fi

disk_total_kb="$(df -Pk "${DATA_MOUNT}" | awk 'NR==2 {print $2}')"
reserve_kb=$((1024 * 1024))
if [ $((disk_total_kb / 5)) -gt "${reserve_kb}" ]; then
  reserve_kb=$((disk_total_kb / 5))
fi
available_kb="$(df -Pk "${DATA_MOUNT}" | awk 'NR==2 {print $4}')"
required_kb=$((restored_kb + reserve_kb))
if [ "${available_kb}" -lt "${required_kb}" ]; then
  echo "not enough free space on ${DATA_MOUNT} for a side-by-side restore" >&2
  echo "available_kb=${available_kb} required_kb=${required_kb}" >&2
  exit 1
fi

if [ "${replace_production}" -eq 0 ]; then
  echo "validation only: dump, manifest and free space are valid"
  echo "re-run with --replace-production to replace the live database (the backup may predate the latest writes)"
  exit 0
fi

echo "restoring dump into ${RESTORE_DB}"
psql_postgres -c "CREATE DATABASE ${RESTORE_DB} OWNER cestia_migrate"
temp_created=1

# Run pg_restore while watching free space so the data disk cannot fill up.
PGPASSWORD="${CESTIA_MIGRATE_PASSWORD}" "${PG_BIN}/pg_restore" \
  -h 127.0.0.1 -U cestia_migrate --no-owner --no-privileges \
  -d "${RESTORE_DB}" "${dump_path}" &
pg_pid=$!
test_signal during_restore
watchdog_failed=0
while kill -0 "${pg_pid}" 2>/dev/null; do
  free_kb="$(df -Pk "${DATA_MOUNT}" | awk 'NR==2 {print $4}')"
  if [ "${free_kb}" -lt "${reserve_kb}" ]; then
    echo "free space dropped below the reserve while restoring; aborting" >&2
    watchdog_failed=1
    kill "${pg_pid}" 2>/dev/null || true
    break
  fi
  sleep 5
done
wait "${pg_pid}" 2>/dev/null || pg_failed=1
pg_failed="${pg_failed:-0}"

if [ "${pg_failed}" -eq 1 ] || [ "${watchdog_failed}" -eq 1 ]; then
  echo "restore into ${RESTORE_DB} did not finish; removing it" >&2
  if drop_temp_db "${RESTORE_DB}"; then
    temp_created=0
  else
    echo "could not confirm removal of ${RESTORE_DB}; remove it manually before retrying" >&2
  fi
  exit 1
fi

echo "verifying the restored database"
version="$(pg_as_postgres "${PG_BIN}/psql" -tAc "SELECT count(*) FROM schema_migrations" -d "${RESTORE_DB}")"
if [ -z "${version}" ] || [ "${version}" = "0" ]; then
  echo "the restored database has no schema_migrations rows; removing it" >&2
  if drop_temp_db "${RESTORE_DB}"; then
    temp_created=0
  fi
  exit 1
fi

# Apply grants and confirm the application role can read before the swap.
psql_postgres -d "${RESTORE_DB}" <<EOF
GRANT CONNECT ON DATABASE ${RESTORE_DB} TO cestia_app;
GRANT ALL ON SCHEMA public TO cestia_migrate;
GRANT USAGE ON SCHEMA public TO cestia_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cestia_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cestia_app;
ALTER DEFAULT PRIVILEGES FOR ROLE cestia_migrate IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO cestia_app;
ALTER DEFAULT PRIVILEGES FOR ROLE cestia_migrate IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO cestia_app;
EOF
if ! verify_app_role "${RESTORE_DB}"; then
  echo "the application role cannot read the restored database; removing it" >&2
  if drop_temp_db "${RESTORE_DB}"; then
    temp_created=0
  fi
  exit 1
fi

# Remember the original database OID: it survives a rename and lets rollback
# prove it restored the original, not the restored copy.
original_oid="$(db_oid "${DB_NAME}")"
if [ -z "${original_oid}" ]; then
  echo "could not read the original database OID; aborting before any swap" >&2
  exit 1
fi

test_fail before_swap
critical_started=1

echo "stopping API and swapping databases"
"${SYSTEMCTL}" stop "${API_SERVICE}" 2>/dev/null || true
terminate_connections
psql_postgres -c "ALTER DATABASE ${DB_NAME} RENAME TO ${PREVIOUS_DB}"
test_fail after_first_rename
test_signal after_first_rename
psql_postgres -c "ALTER DATABASE ${RESTORE_DB} RENAME TO ${DB_NAME}"
temp_created=0
test_fail after_swap
test_signal after_swap

psql_postgres -d "${DB_NAME}" <<EOF
GRANT CONNECT ON DATABASE ${DB_NAME} TO cestia_app;
GRANT ALL ON SCHEMA public TO cestia_migrate;
GRANT USAGE ON SCHEMA public TO cestia_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cestia_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cestia_app;
EOF

echo "starting API"
"${SYSTEMCTL}" start "${API_SERVICE}"
test_fail after_start

for _ in $(seq 1 30); do
  if curl -fsS --max-time 3 "${HEALTH_URL}" >/dev/null; then
    break
  fi
  sleep 2
done
curl -fsS --max-time 3 "${HEALTH_URL}" >/dev/null
verify_app_role "${DB_NAME}"

echo "restore complete and verified"
echo "the previous database is kept as ${PREVIOUS_DB}; review it against the backup RPO, then drop it:"
echo "  runuser -u postgres -- psql -c 'DROP DATABASE ${PREVIOUS_DB}'"
exit 0

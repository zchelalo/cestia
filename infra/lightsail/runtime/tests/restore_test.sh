#!/usr/bin/env bash
# Exercises restore-postgres.sh against a disposable PostgreSQL 17 in Docker.
#
# The live database and the backup use different sentinel rows, so a restore
# that silently leaves the wrong database active fails the checks. It also
# tracks the original database OID, which survives a rename, and injects
# failures through CESTIA_TEST_FAIL_STAGE. It never touches production.
set -Eeuo pipefail

IMAGE="postgres:17"
CONTAINER="cestia-restore-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESTORE_SCRIPT="${SCRIPT_DIR}/../scripts/restore-postgres.sh"
DATA_MOUNT="/mnt/cestia-data"
MIGRATE_PASSWORD="migrate-secret"
APP_PASSWORD="app-secret"

failures=0
last_output="$(mktemp)"
cleanup() {
  docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
  rm -f "${last_output}"
}
trap cleanup EXIT

check() {
  local label="$1"
  shift
  if "$@"; then
    echo "ok   - ${label}"
  else
    echo "FAIL - ${label}" >&2
    failures=$((failures + 1))
  fi
}

sql() { docker exec -i -u postgres "${CONTAINER}" psql -v ON_ERROR_STOP=1 -q "$@"; }
sql_t() {
  local query="$1"
  shift
  docker exec -i -u postgres "${CONTAINER}" psql -tAc "${query}" "$@"
}

live_intact() {
  [ "$(sql_t "SELECT count(*) FROM users WHERE id = 1" -d cestia 2>/dev/null)" = "1" ]
}
live_is_restored() {
  [ "$(sql_t "SELECT count(*) FROM users WHERE id = 2" -d cestia 2>/dev/null)" = "1" ]
}
previous_is_live() {
  [ "$(sql_t "SELECT count(*) FROM users WHERE id = 1" -d cestia_previous 2>/dev/null)" = "1" ]
}
no_temp_left() {
  [ "$(sql_t "SELECT count(*) FROM pg_database WHERE datname LIKE 'cestia_restore%'")" = "0" ]
}
cestia_oid() {
  sql_t "SELECT oid FROM pg_database WHERE datname = 'cestia'"
}

run_restore() {
  set +e
  docker exec -e CESTIA_PG_RUNAS=postgres -e CESTIA_TEST_FAIL_STAGE="${1:-}" \
    "${CONTAINER}" /usr/local/sbin/cestia-restore-postgres \
    --replace-production /tmp/cestia.dump 2>&1 | tee "${last_output}"
  local status="${PIPESTATUS[0]}"
  set -e
  return "${status}"
}

run_restore_signal() {
  set +e
  docker exec -e CESTIA_PG_RUNAS=postgres -e CESTIA_TEST_SIGNAL_STAGE="${1:-}" \
    "${CONTAINER}" /usr/local/sbin/cestia-restore-postgres \
    --replace-production /tmp/cestia.dump 2>&1 | tee "${last_output}"
  local status="${PIPESTATUS[0]}"
  set -e
  return "${status}"
}

run_restore_validate() {
  set +e
  docker exec -e CESTIA_PG_RUNAS=postgres \
    "${CONTAINER}" /usr/local/sbin/cestia-restore-postgres \
    /tmp/cestia.dump 2>&1 | tee "${last_output}"
  local status="${PIPESTATUS[0]}"
  set -e
  return "${status}"
}

run_restore_s3() {
  set +e
  docker exec -e CESTIA_PG_RUNAS=postgres -e CESTIA_TEST_FAIL_STAGE="${1:-}" \
    "${CONTAINER}" /usr/local/sbin/cestia-restore-postgres \
    --replace-production s3://bucket/cestia.dump 2>&1 | tee "${last_output}"
  local status="${PIPESTATUS[0]}"
  set -e
  return "${status}"
}

# After a rollback the script keeps data and reports any leftover temporary
# database instead of deleting it; accept either absence or that report.
temp_clean_or_reported() {
  if [ "$(sql_t "SELECT count(*) FROM pg_database WHERE datname LIKE 'cestia_restore%'")" = "0" ]; then
    return 0
  fi
  grep -q "leftover temporary database" "${last_output}"
}

seed_db() {
  docker exec -i -u postgres "${CONTAINER}" psql -v ON_ERROR_STOP=1 -q -d "$1" <<'SQL'
CREATE TABLE schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());
INSERT INTO schema_migrations (version) VALUES ('000000_schema_migrations'), ('000004_concurrency_guards');
CREATE TABLE users (id int);
SQL
}

reset_state() {
  sql -c "DROP DATABASE IF EXISTS cestia_previous"
  sql -c "DROP DATABASE IF EXISTS cestia_source"
  sql -c "DROP DATABASE IF EXISTS cestia"
  # Rollback scenarios intentionally keep a temporary database; clear any before
  # the next scenario so its checks start from a clean slate.
  sql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname LIKE 'cestia_restore%'" >/dev/null
  for leftover in $(sql_t "SELECT datname FROM pg_database WHERE datname LIKE 'cestia_restore%'"); do
    sql -c "DROP DATABASE IF EXISTS ${leftover}" >/dev/null
  done

  sql -c "CREATE DATABASE cestia_source OWNER cestia_migrate"
  seed_db cestia_source
  sql -d cestia_source -c "INSERT INTO users (id) VALUES (2)"
  sql -d cestia_source -c "GRANT USAGE ON SCHEMA public TO cestia_app"
  sql -d cestia_source -c "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cestia_app"
  docker exec -u postgres "${CONTAINER}" pg_dump -Fc -d cestia_source -f /tmp/cestia.dump
  docker exec "${CONTAINER}" bash -c 'rm -f /tmp/cestia.json'
  sql -c "DROP DATABASE cestia_source"

  sql -c "CREATE DATABASE cestia OWNER cestia_migrate"
  seed_db cestia
  sql -d cestia -c "INSERT INTO users (id) VALUES (1)"
  sql -d cestia -c "GRANT USAGE ON SCHEMA public TO cestia_app"
  sql -d cestia -c "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cestia_app"
}

write_manifest() {
  # write_manifest <size_bytes> [sha_override]
  local size="$1"
  local sha="${2:-}"
  if [ -z "${sha}" ]; then
    sha="$(docker exec "${CONTAINER}" sha256sum /tmp/cestia.dump | awk '{print $1}')"
  fi
  docker exec "${CONTAINER}" bash -c "cat > /tmp/cestia.json <<JSON
{\"dump\":\"cestia.dump\",\"database_size_bytes\":${size},\"schema_migration_version\":\"000004_concurrency_guards\",\"sha256\":\"${sha}\",\"created_at\":\"2026-01-01T00:00:00Z\"}
JSON"
}

# The restore script sizes its reserve from the data volume. Report a fixed 8 GiB
# volume for the data mount so the check does not depend on the CI runner disk;
# every other df call is delegated to the real binary.
install_df_stub() {
  local real_df stub
  real_df="$(docker exec "${CONTAINER}" sh -c 'command -v df')"
  if [ -z "${real_df}" ]; then
    echo "could not locate df in the test container" >&2
    exit 1
  fi
  stub="$(mktemp)"
  cat >"${stub}" <<STUB
#!/bin/sh
case "\$*" in
  *${DATA_MOUNT}*) ;;
  *) exec ${real_df} "\$@" ;;
esac
total=8388608
avail=6291456
if [ -f /tmp/df-watchdog ]; then
  count=0
  [ -f /tmp/df-calls ] && count=\$(cat /tmp/df-calls)
  count=\$((count + 1))
  echo "\$count" > /tmp/df-calls
  if [ "\$count" -ge 3 ]; then
    echo 'Filesystem 1024-blocks Used Available Capacity Mounted on'
    echo 'overlay 8388608 8388607 1 100% ${DATA_MOUNT}'
    exit 0
  fi
fi
echo 'Filesystem 1024-blocks Used Available Capacity Mounted on'
echo "overlay \${total} \$((total - avail)) \${avail} 25% ${DATA_MOUNT}"
STUB
  docker cp "${stub}" "${CONTAINER}:/usr/local/bin/df" >/dev/null
  docker exec "${CONTAINER}" chmod 0755 /usr/local/bin/df
  rm -f "${stub}"
}

# Make the data mount report almost no free space from the third df call on, so
# the preflight passes but the restore watchdog must abort.
enable_watchdog_df() {
  docker exec "${CONTAINER}" bash -c 'rm -f /tmp/df-calls; touch /tmp/df-watchdog'
}
disable_watchdog_df() {
  docker exec "${CONTAINER}" bash -c 'rm -f /tmp/df-watchdog /tmp/df-calls'
}

enable_aws_stub() {
  docker exec "${CONTAINER}" bash -c 'cat > /usr/local/bin/aws <<"STUB"
#!/bin/sh
case "$1" in
  s3api)
    mode=ok
    [ -f /tmp/aws-head-mode ] && mode=$(cat /tmp/aws-head-mode)
    if [ "$mode" = "404" ]; then
      echo "An error occurred (404) when calling the HeadObject operation: Not Found" >&2
      exit 254
    fi
    if [ "$mode" = "403" ]; then
      echo "An error occurred (AccessDenied) when calling the HeadObject operation: Access Denied" >&2
      exit 254
    fi
    exit 0
    ;;
  s3)
    src="$3"
    dest="$4"
    case "$src" in
      *.json)
        if [ -f /tmp/aws-manifest-fail ]; then
          echo "download failed" >&2
          exit 1
        fi
        cp /tmp/cestia.json "$dest"
        ;;
      *)
        cp /tmp/cestia.dump "$dest"
        ;;
    esac
    exit 0
    ;;
esac
exit 0
STUB
chmod +x /usr/local/bin/aws'
}

echo "starting disposable PostgreSQL"
docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER}" \
  -e POSTGRES_PASSWORD=test -e POSTGRES_HOST_AUTH_METHOD=scram-sha-256 \
  "${IMAGE}" >/dev/null
# The entrypoint starts, initializes, then restarts the server; wait until two
# consecutive queries succeed so setup does not race the restart.
ready=0
for _ in $(seq 1 90); do
  if docker exec "${CONTAINER}" psql -U postgres -tAc "SELECT 1" >/dev/null 2>&1; then
    sleep 3
    if docker exec "${CONTAINER}" psql -U postgres -tAc "SELECT 1" >/dev/null 2>&1; then
      ready=1
      break
    fi
  fi
  sleep 1
done
if [ "${ready}" -ne 1 ]; then
  echo "PostgreSQL container did not become ready" >&2
  exit 1
fi

sql -c "CREATE ROLE cestia_migrate LOGIN PASSWORD '${MIGRATE_PASSWORD}'"
sql -c "CREATE ROLE cestia_app LOGIN PASSWORD '${APP_PASSWORD}'"

# The production VM installs jq during bootstrap; the disposable container needs
# it too so manifest parsing behaves like production.
if ! docker exec "${CONTAINER}" sh -c 'command -v jq' >/dev/null 2>&1; then
  docker exec "${CONTAINER}" bash -c 'apt-get update -qq && apt-get install -y -qq jq' >/dev/null 2>&1
fi
if ! docker exec "${CONTAINER}" sh -c 'command -v jq' >/dev/null 2>&1; then
  echo "could not install jq in the test container" >&2
  exit 1
fi

docker exec "${CONTAINER}" bash -c '
  set -e
  mkdir -p /etc/cestia /mnt/cestia-data/cestia/secrets /usr/local/sbin
  printf "%s\n" "AWS_REGION=us-east-1" "BACKUP_BUCKET=bucket" > /etc/cestia/backup.env
  : > /etc/cestia/aws.env
  printf "%s\n" "CESTIA_MIGRATE_PASSWORD=migrate-secret" > /mnt/cestia-data/cestia/secrets/postgres.env
  printf "%s\n" "DB_PASS=app-secret" > /etc/cestia/db.env
  printf "#!/bin/sh\nexit 0\n" > /usr/local/bin/systemctl
  printf "#!/bin/sh\nexit 0\n" > /usr/local/bin/curl
  chmod +x /usr/local/bin/systemctl /usr/local/bin/curl
'
docker cp "${RESTORE_SCRIPT}" "${CONTAINER}:/usr/local/sbin/cestia-restore-postgres"
docker exec "${CONTAINER}" chmod 0755 /usr/local/sbin/cestia-restore-postgres
install_df_stub

echo "scenario: valid restore with manifest"
reset_state
write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
live_oid="$(cestia_oid)"
check "valid restore succeeds" run_restore ""
check "restored database is active" live_is_restored
check "previous database holds the original data" previous_is_live
check "previous database keeps the original OID" \
  bash -c "[ \"$(sql_t "SELECT oid FROM pg_database WHERE datname='cestia_previous'")\" = \"${live_oid}\" ]"
sql -c "DROP DATABASE IF EXISTS cestia_previous"

echo "scenario: corrupt dump"
reset_state
docker exec "${CONTAINER}" bash -c 'printf "not a dump" > /tmp/bad.dump'
if docker exec -e CESTIA_PG_RUNAS=postgres "${CONTAINER}" \
  /usr/local/sbin/cestia-restore-postgres /tmp/bad.dump; then
  check "corrupt dump is rejected" false
else
  check "corrupt dump is rejected" true
fi
check "live database intact after corrupt dump" live_intact
check "no temp database after corrupt dump" no_temp_left

echo "scenario: previous database pending"
reset_state
sql -c "CREATE DATABASE cestia_previous" >/dev/null
if run_restore ""; then
  check "pending previous is refused" false
else
  check "pending previous is refused" true
fi
check "live database intact when previous pending" live_intact
sql -c "DROP DATABASE IF EXISTS cestia_previous"

echo "scenario: manifest checksum mismatch"
reset_state
write_manifest "$(sql_t "SELECT pg_database_size('cestia')")" "deadbeef"
live_oid="$(cestia_oid)"
if run_restore ""; then
  check "mismatched manifest is refused" false
else
  check "mismatched manifest is refused" true
fi
check "live database intact after manifest mismatch" live_intact
check "original OID kept after manifest mismatch" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "no temp database after manifest mismatch" no_temp_left

echo "scenario: not enough space"
reset_state
write_manifest "109951162777600"
live_oid="$(cestia_oid)"
if run_restore ""; then
  check "insufficient space is refused" false
else
  check "insufficient space is refused" true
fi
check "live database intact when space is short" live_intact
check "original OID kept when space is short" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "no temp database when space is short" no_temp_left

echo "scenario: old backup without manifest"
reset_state
check "restore without manifest still succeeds" run_restore ""
check "restored database is active without manifest" live_is_restored
sql -c "DROP DATABASE IF EXISTS cestia_previous"

for stage in after_first_rename after_swap after_start; do
  echo "scenario: failure at ${stage}"
  reset_state
  write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
  live_oid="$(cestia_oid)"
  if run_restore "${stage}"; then
    check "injected ${stage} failure aborts" false
  else
    check "injected ${stage} failure aborts" true
  fi
  check "live database intact after ${stage}" live_intact
  check "original OID restored after ${stage}" \
    bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
  check "no leftover temp or it is reported after ${stage}" temp_clean_or_reported
done

echo "scenario: disk drops below reserve during restore"
reset_state
write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
live_oid="$(cestia_oid)"
enable_watchdog_df
if run_restore ""; then
  check "watchdog abort fails the restore" false
else
  check "watchdog abort fails the restore" true
fi
check "live database intact after watchdog abort" live_intact
check "original OID kept after watchdog abort" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "temp removed or reported after watchdog abort" temp_clean_or_reported
disable_watchdog_df

echo "scenario: S3 backup without a manifest (legacy)"
reset_state
enable_aws_stub
docker exec "${CONTAINER}" bash -c 'echo 404 > /tmp/aws-head-mode; rm -f /tmp/aws-manifest-fail'
check "legacy S3 restore succeeds" run_restore_s3 ""
check "restored database is active for legacy S3" live_is_restored
sql -c "DROP DATABASE IF EXISTS cestia_previous"

echo "scenario: S3 manifest access denied"
reset_state
live_oid="$(cestia_oid)"
docker exec "${CONTAINER}" bash -c 'echo 403 > /tmp/aws-head-mode'
if run_restore_s3 ""; then
  check "S3 access denied aborts" false
else
  check "S3 access denied aborts" true
fi
check "live database intact after S3 access denied" live_intact
check "original OID kept after S3 access denied" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "no temp database after S3 access denied" no_temp_left

echo "scenario: S3 manifest download failure"
reset_state
live_oid="$(cestia_oid)"
docker exec "${CONTAINER}" bash -c 'echo ok > /tmp/aws-head-mode; touch /tmp/aws-manifest-fail'
if run_restore_s3 ""; then
  check "S3 manifest download failure aborts" false
else
  check "S3 manifest download failure aborts" true
fi
check "live database intact after S3 manifest download failure" live_intact
check "original OID kept after S3 manifest download failure" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "no temp database after S3 manifest download failure" no_temp_left

echo "scenario: S3 manifest present and valid"
reset_state
docker exec "${CONTAINER}" bash -c 'echo ok > /tmp/aws-head-mode; rm -f /tmp/aws-manifest-fail'
write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
check "valid S3 restore succeeds" run_restore_s3 ""
check "restored database is active for valid S3" live_is_restored
sql -c "DROP DATABASE IF EXISTS cestia_previous"
docker exec "${CONTAINER}" bash -c 'rm -f /usr/local/bin/aws /tmp/aws-head-mode /tmp/aws-manifest-fail'

for stage in during_restore after_first_rename after_swap; do
  echo "scenario: SIGTERM at ${stage}"
  reset_state
  write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
  live_oid="$(cestia_oid)"
  if run_restore_signal "${stage}"; then
    check "signal at ${stage} aborts" false
  else
    check "signal at ${stage} aborts" true
  fi
  check "live database intact after signal at ${stage}" live_intact
  check "original OID restored after signal at ${stage}" \
    bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
  check "temp clean or reported after signal at ${stage}" temp_clean_or_reported
done

echo "scenario: validation without --replace-production"
reset_state
write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
live_oid="$(cestia_oid)"
check "validation-only run succeeds" run_restore_validate ""
check "validation-only keeps the live database" live_intact
check "validation-only keeps the original OID" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "validation-only creates no previous database" \
  bash -c "[ -z \"$(sql_t "SELECT 1 FROM pg_database WHERE datname='cestia_previous'")\" ]"
check "validation-only creates no temp database" no_temp_left

echo "scenario: concurrent restore is refused"
reset_state
write_manifest "$(sql_t "SELECT pg_database_size('cestia')")"
live_oid="$(cestia_oid)"
docker exec -d "${CONTAINER}" sh -c \
  'exec 9>/run/lock/cestia-maintenance.lock; flock -n 9; sleep 6'
sleep 1
if run_restore ""; then
  check "concurrent restore is refused" false
else
  check "concurrent restore is refused" true
fi
check "live database intact while locked" live_intact
check "original OID kept while locked" \
  bash -c "[ \"$(cestia_oid)\" = \"${live_oid}\" ]"
check "no temp database while locked" no_temp_left
sleep 6

if [ "${failures}" -ne 0 ]; then
  echo "${failures} check(s) failed" >&2
  exit 1
fi
echo "all restore scenarios passed"

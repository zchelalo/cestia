#!/usr/bin/env bash
# Exercises the packaged backup-postgres.sh against a disposable PostgreSQL 17
# with a stubbed AWS CLI. It checks the produced dump and manifest, restores that
# dump into another database, and verifies the shared maintenance lock. It never
# touches AWS or production.
set -Eeuo pipefail

IMAGE="postgres:17"
CONTAINER="cestia-backup-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_SCRIPT="${SCRIPT_DIR}/../scripts/backup-postgres.sh"
MIGRATE_PASSWORD="migrate-secret"
LOCK="/tmp/maintenance.lock"

failures=0
cleanup() {
  docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
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

run_backup() {
  docker exec -e CESTIA_PG_RUNAS=postgres -e CESTIA_LOCK_FILE="${LOCK}" \
    -e CESTIA_LOCK_WAIT="${1:-300}" \
    "${CONTAINER}" /usr/local/sbin/cestia-pg-backup.sh
}

echo "starting disposable PostgreSQL"
docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER}" \
  -e POSTGRES_PASSWORD=test -e POSTGRES_HOST_AUTH_METHOD=scram-sha-256 \
  "${IMAGE}" >/dev/null
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

if ! docker exec "${CONTAINER}" sh -c 'command -v jq' >/dev/null 2>&1; then
  docker exec "${CONTAINER}" bash -c 'apt-get update -qq && apt-get install -y -qq jq' >/dev/null 2>&1
fi
if ! docker exec "${CONTAINER}" sh -c 'command -v jq' >/dev/null 2>&1; then
  echo "could not install jq in the test container" >&2
  exit 1
fi

sql -c "CREATE ROLE cestia_migrate LOGIN PASSWORD '${MIGRATE_PASSWORD}'"
sql -c "CREATE DATABASE cestia OWNER cestia_migrate"
docker exec -i -u postgres "${CONTAINER}" psql -v ON_ERROR_STOP=1 -q -d cestia <<'SQL'
CREATE TABLE schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());
INSERT INTO schema_migrations (version) VALUES ('000000_schema_migrations'), ('000004_concurrency_guards');
CREATE TABLE users (id int);
INSERT INTO users (id) VALUES (1);
SQL

# AWS CLI stub: writes to a local directory instead of S3.
docker exec "${CONTAINER}" bash -c 'cat > /usr/local/bin/aws <<"STUB"
#!/bin/sh
src="$3"
dest="$4"
case "$src" in
  *.json)
    if [ -f /tmp/backup-manifest-fail ]; then
      echo "upload failed" >&2
      exit 1
    fi
    ;;
esac
mkdir -p /tmp/s3
cp "$src" "/tmp/s3/$(basename "$dest")"
STUB
chmod +x /usr/local/bin/aws'

docker exec "${CONTAINER}" bash -c '
  set -e
  mkdir -p /etc/cestia /mnt/cestia-data/cestia/secrets
  printf "%s\n" "AWS_REGION=us-east-1" "BACKUP_BUCKET=bucket" > /etc/cestia/backup.env
  : > /etc/cestia/aws.env
'
docker cp "${BACKUP_SCRIPT}" "${CONTAINER}:/usr/local/sbin/cestia-pg-backup.sh"
docker exec "${CONTAINER}" chmod 0750 /usr/local/sbin/cestia-pg-backup.sh

echo "scenario: successful backup produces dump and manifest"
docker exec "${CONTAINER}" bash -c 'rm -rf /tmp/s3 /tmp/backup-manifest-fail'
check "backup succeeds" run_backup
check "dump uploaded" bash -c "[ -n \"$(docker exec "${CONTAINER}" sh -c 'ls /tmp/s3/*.dump 2>/dev/null')\" ]"
check "manifest uploaded" bash -c "[ -n \"$(docker exec "${CONTAINER}" sh -c 'ls /tmp/s3/*.json 2>/dev/null')\" ]"

manifest_sha="$(docker exec "${CONTAINER}" sh -c 'jq -r .sha256 /tmp/s3/*.json')"
dump_sha="$(docker exec "${CONTAINER}" sh -c 'sha256sum /tmp/s3/*.dump | awk "{print \$1}"')"
check "manifest checksum matches the dump" test "${manifest_sha}" = "${dump_sha}"
check "manifest records the latest migration" \
  bash -c "[ \"$(docker exec "${CONTAINER}" sh -c 'jq -r .schema_migration_version /tmp/s3/*.json')\" = \"000004_concurrency_guards\" ]"

echo "scenario: the produced dump restores into another database"
docker exec -u postgres "${CONTAINER}" bash -c '
  set -e
  psql -q -c "CREATE DATABASE cestia_verify"
  pg_restore --no-owner --no-privileges -d cestia_verify /tmp/s3/*.dump'
check "restored dump has the sentinel row" \
  bash -c "[ \"$(docker exec -u postgres "${CONTAINER}" psql -tAc "SELECT count(*) FROM users WHERE id = 1" -d cestia_verify)\" = \"1\" ]"

echo "scenario: backup waits and fails when the maintenance lock is held"
docker exec -d "${CONTAINER}" sh -c "exec 9>${LOCK}; flock -n 9; sleep 6"
sleep 1
if run_backup 2; then
  check "locked backup fails" false
else
  check "locked backup fails" true
fi
sleep 6

echo "scenario: manifest upload failure fails the backup"
docker exec "${CONTAINER}" bash -c 'touch /tmp/backup-manifest-fail'
if run_backup; then
  check "manifest upload failure fails" false
else
  check "manifest upload failure fails" true
fi

if [ "${failures}" -ne 0 ]; then
  echo "${failures} check(s) failed" >&2
  exit 1
fi
echo "all backup scenarios passed"

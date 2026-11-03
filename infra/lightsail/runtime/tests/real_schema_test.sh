#!/usr/bin/env bash
# End-to-end backup/restore against the real Cestia schema.
#
# Requires Docker and the Go toolchain (it builds the same `cestia-migrate`
# binary the release ships). The fast harnesses (backup_test.sh,
# restore_test.sh) use tiny fixtures to inject failures quickly. This one closes
# the gap they cannot see: it applies every real migration with the release's
# migrator, produces the dump with the packaged backup script and restores it
# with the packaged restore script, then reads a product row through the
# application role. It uses only disposable PostgreSQL and a local AWS stub.
set -Eeuo pipefail

IMAGE="postgres:17"
CONTAINER="cestia-real-schema-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
API_DIR="${REPO_ROOT}/apps/api"
MIGRATIONS_DIR="${API_DIR}/internal/migrations"
BACKUP_SCRIPT="${SCRIPT_DIR}/../scripts/backup-postgres.sh"
RESTORE_SCRIPT="${SCRIPT_DIR}/../scripts/restore-postgres.sh"
DATA_MOUNT="/mnt/cestia-data"
MIGRATE_PASSWORD="migrate-secret"
APP_PASSWORD="app-secret"
USER_ID="11111111-1111-1111-1111-111111111111"
USER_EMAIL="real-schema@example.com"

for tool in docker go; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "${tool} is required to run the real-schema test" >&2
    exit 1
  fi
done

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
  docker exec -u postgres "${CONTAINER}" psql -tAc "${query}" "$@"
}

# Reads through the application role over TCP, exactly like production.
app_query() {
  docker exec -e PGPASSWORD="${APP_PASSWORD}" "${CONTAINER}" \
    psql -h 127.0.0.1 -U cestia_app -d cestia -tAc "$1"
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
echo 'Filesystem 1024-blocks Used Available Capacity Mounted on'
echo 'overlay 8388608 2097152 6291456 25% ${DATA_MOUNT}'
STUB
  docker cp "${stub}" "${CONTAINER}:/usr/local/bin/df" >/dev/null
  docker exec "${CONTAINER}" chmod 0755 /usr/local/bin/df
  rm -f "${stub}"
}

latest_migration_version() {
  local latest
  latest="$(for file in "${MIGRATIONS_DIR}"/*.up.sql; do
    name="$(basename "${file}" .up.sql)"
    [ "${name}" = "000000_schema_migrations" ] && continue
    printf '%s\n' "${name}"
  done | sort | tail -1)"
  if [ -z "${latest}" ]; then
    return 1
  fi
  printf '%s' "${latest}"
}

# Builds the same migrator the release ships, for the disposable image's
# architecture, and prints its host path.
build_migrate() {
  local arch tmp
  case "$(docker exec "${CONTAINER}" uname -m)" in
    x86_64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *)
      echo "unsupported container architecture" >&2
      return 1
      ;;
  esac
  tmp="$(mktemp -d)"
  if ! (cd "${API_DIR}" && CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" \
    go build -buildvcs=false -o "${tmp}/cestia-migrate" ./cmd/migrate) >/dev/null 2>&1; then
    echo "could not build cestia-migrate" >&2
    return 1
  fi
  printf '%s' "${tmp}/cestia-migrate"
}

# Applies the real embedded migrations with the release's migrator and prints the
# latest version it recorded.
run_migrations() {
  local latest migrate_bin actual
  latest="$(latest_migration_version)" || {
    echo "could not determine the latest migration version" >&2
    exit 1
  }
  migrate_bin="$(build_migrate)" || exit 1
  docker cp "${migrate_bin}" "${CONTAINER}:/usr/local/bin/cestia-migrate"
  docker exec "${CONTAINER}" chmod 0755 /usr/local/bin/cestia-migrate
  rm -rf "$(dirname "${migrate_bin}")"

  if ! docker exec -e DATABASE_URL="postgres://cestia_migrate:${MIGRATE_PASSWORD}@127.0.0.1:5432/cestia?sslmode=disable" \
    "${CONTAINER}" /usr/local/bin/cestia-migrate >&2; then
    echo "cestia-migrate failed to apply the real migrations" >&2
    exit 1
  fi

  actual="$(sql_t "SELECT version FROM schema_migrations WHERE version <> '000000_schema_migrations' ORDER BY version DESC LIMIT 1" -d cestia)"
  if [ "${actual}" != "${latest}" ]; then
    echo "cestia-migrate recorded an unexpected schema_migrations state: got '${actual}', want '${latest}'" >&2
    exit 1
  fi
  printf '%s' "${latest}"
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

sql -c "CREATE ROLE cestia_migrate LOGIN PASSWORD '${MIGRATE_PASSWORD}'"
sql -c "CREATE ROLE cestia_app LOGIN PASSWORD '${APP_PASSWORD}'"
sql -c "CREATE DATABASE cestia OWNER cestia_migrate"

if ! docker exec "${CONTAINER}" sh -c 'command -v jq' >/dev/null 2>&1; then
  docker exec "${CONTAINER}" bash -c 'apt-get update -qq && apt-get install -y -qq jq' >/dev/null 2>&1
fi
if ! docker exec "${CONTAINER}" sh -c 'command -v jq' >/dev/null 2>&1; then
  echo "could not install jq in the test container" >&2
  exit 1
fi

schema_version="$(run_migrations)"
echo "applied real migrations up to version ${schema_version}"

# A product row the restore must preserve and that only the application role can
# read once restore-postgres.sh applies its grants.
docker exec -e PGPASSWORD="${MIGRATE_PASSWORD}" \
  -e PGHOST=127.0.0.1 -e PGUSER=cestia_migrate -e PGDATABASE=cestia \
  "${CONTAINER}" psql -v ON_ERROR_STOP=1 -q \
  -c "INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at) VALUES ('${USER_ID}', '${USER_EMAIL}', 'Real Schema', 'not-a-real-hash', now(), now())"

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
  rm -rf /tmp/s3'

# AWS CLI stub: writes to a local directory instead of S3.
docker exec "${CONTAINER}" bash -c 'cat > /usr/local/bin/aws <<"STUB"
#!/bin/sh
src="$3"
dest="$4"
mkdir -p /tmp/s3
cp "$src" "/tmp/s3/$(basename "$dest")"
STUB
chmod +x /usr/local/bin/aws'

docker cp "${BACKUP_SCRIPT}" "${CONTAINER}:/usr/local/sbin/cestia-pg-backup.sh"
docker cp "${RESTORE_SCRIPT}" "${CONTAINER}:/usr/local/sbin/cestia-restore-postgres"
docker exec "${CONTAINER}" chmod 0750 /usr/local/sbin/cestia-pg-backup.sh
docker exec "${CONTAINER}" chmod 0755 /usr/local/sbin/cestia-restore-postgres
install_df_stub

echo "scenario: real-schema dump is produced by the packaged backup script"
if docker exec -e CESTIA_PG_RUNAS=postgres -e CESTIA_LOCK_FILE=/tmp/maintenance.lock \
  "${CONTAINER}" /usr/local/sbin/cestia-pg-backup.sh; then
  check "backup succeeds against the real schema" true
else
  check "backup succeeds against the real schema" false
fi
dump_ref="$(docker exec "${CONTAINER}" sh -c 'ls /tmp/s3/*.dump 2>/dev/null' | head -1)"
manifest_ref="$(docker exec "${CONTAINER}" sh -c 'ls /tmp/s3/*.json 2>/dev/null' | head -1)"
check "dump uploaded" bash -c "[ -n \"${dump_ref}\" ]"
check "manifest uploaded" bash -c "[ -n \"${manifest_ref}\" ]"
check "manifest records the real schema version" \
  bash -c "[ \"$(docker exec "${CONTAINER}" sh -c 'jq -r .schema_migration_version /tmp/s3/*.json')\" = \"${schema_version}\" ]"

echo "scenario: the real-schema dump restores and stays readable through cestia_app"
set +e
docker exec -e CESTIA_PG_RUNAS=postgres \
  "${CONTAINER}" /usr/local/sbin/cestia-restore-postgres \
  --replace-production "${dump_ref}" 2>&1 | tee "${last_output}"
restore_status="${PIPESTATUS[0]}"
set -e
check "restore of the real-schema dump succeeds" test "${restore_status}" -eq 0
check "schema_migrations version preserved" \
  bash -c "[ \"$(app_query "SELECT version FROM schema_migrations WHERE version <> '000000_schema_migrations' ORDER BY version DESC LIMIT 1")\" = \"${schema_version}\" ]"
check "public schema has the product tables" \
  bash -c "[ \"$(app_query "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'")\" -ge 5 ]"
check "foreign keys survived the restore" \
  bash -c "[ \"$(app_query "SELECT count(*) FROM pg_constraint WHERE contype='f'")\" -ge 3 ]"
check "cestia_app reads the restored product row" \
  bash -c "[ \"$(app_query "SELECT email FROM users WHERE id = '${USER_ID}'")\" = \"${USER_EMAIL}\" ]"
check "previous database keeps the product row for rollback review" \
  bash -c "[ \"$(sql_t "SELECT email FROM users WHERE id = '${USER_ID}'" -d cestia_previous 2>/dev/null)\" = \"${USER_EMAIL}\" ]"

if [ "${failures}" -ne 0 ]; then
  echo "${failures} check(s) failed" >&2
  exit 1
fi
echo "all real-schema scenarios passed"

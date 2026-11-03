#!/usr/bin/env bash
# Cestia release deployment (run as root on the Lightsail instance).
#
# Usage: deploy-release.sh <release.tar.gz> <release-id>
#
# The archive must contain, at its top level:
#   cestia-api     compiled API binary
#   cestia-migrate compiled migration runner
#   restore-postgres.sh
#   backup-postgres.sh
#   web/           built SPA (index.html + assets)
#
# The script runs pending migrations before switching the `current` symlink and
# keeps the previous release available for rollback.
set -Eeuo pipefail

archive="${1:?release archive is required}"
release_id="${2:?release id is required}"

if [[ "${EUID}" -ne 0 ]] && [ -z "${CESTIA_TEST_ALLOW_NONROOT:-}" ]; then
  echo "deploy-release.sh must run as root" >&2
  exit 1
fi
if [[ ! "${release_id}" =~ ^[a-f0-9]{7,64}$ ]]; then
  echo "release id must be a lowercase git SHA" >&2
  exit 1
fi
if [[ ! -f "${archive}" ]]; then
  echo "release archive not found: ${archive}" >&2
  exit 1
fi

root="${CESTIA_ROOT:-/opt/cestia}"
release_dir="${root}/releases/${release_id}"
staging="${root}/releases/.staging-${release_id}"
attempt_dir="${root}/releases/.attempt-${release_id}"
current_link="${root}/current"
restore_path="${CESTIA_RESTORE_PATH:-/usr/local/sbin/cestia-restore-postgres}"
backup_path="${CESTIA_BACKUP_PATH:-/usr/local/sbin/cestia-pg-backup.sh}"
api_service="${CESTIA_API_SERVICE:-cestia-api.service}"
health_url="${CESTIA_HEALTH_URL:-http://127.0.0.1:8000/api/health}"
env_dir="${CESTIA_ENV_DIR:-/etc/cestia}"
SYSTEMCTL="${CESTIA_SYSTEMCTL:-systemctl}"
LOCK_FILE="${CESTIA_LOCK_FILE:-/run/lock/cestia-maintenance.lock}"
secrets_file="${CESTIA_SECRETS_FILE:-/mnt/cestia-data/cestia/secrets/postgres.env}"
backup_timer="${CESTIA_BACKUP_TIMER:-cestia-pg-backup.timer}"
backup_service="${CESTIA_BACKUP_SERVICE:-cestia-pg-backup.service}"
previous_release=""

# Track what this attempt actually created so cleanup never removes a release,
# helper or staging directory owned by another run.
staging_created=0
release_created=0
restore_replaced=0
backup_replaced=0
keep_attempt=0
deploy_completed=0
mutation_started=0
backup_timer_was_active=0
backup_timer_restore_attempted=0
backup_timer_restored=0

# Restores the stable helpers this attempt replaced. Only clears a flag after the
# copy succeeds, and reports failure so a rollback never starts with the wrong
# helper or discards the only backup.
restore_previous_helpers() {
  local ok=0
  if [ "${restore_replaced}" -eq 1 ]; then
    if [ -f "${attempt_dir}/restore-postgres.sh" ] &&
      install -m 0755 "${attempt_dir}/restore-postgres.sh" "${restore_path}" 2>/dev/null; then
      restore_replaced=0
    else
      echo "could not restore the previous cestia-restore-postgres" >&2
      ok=1
    fi
  fi
  if [ "${backup_replaced}" -eq 1 ]; then
    if [ -f "${attempt_dir}/backup-postgres.sh" ] &&
      install -m 0750 "${attempt_dir}/backup-postgres.sh" "${backup_path}" 2>/dev/null; then
      backup_replaced=0
    else
      echo "could not restore the previous cestia-pg-backup.sh" >&2
      ok=1
    fi
  fi
  return "${ok}"
}

# Prints the ActiveState of a unit, failing when systemd cannot report it.
# A missing or unreadable unit must never be mistaken for "inactive".
unit_active_state() {
  local state
  if ! state="$("${SYSTEMCTL}" show -p ActiveState --value "$1" 2>/dev/null)"; then
    return 1
  fi
  if [ -z "${state}" ]; then
    return 1
  fi
  printf '%s' "${state}"
}

# Restores the backup timer this attempt stopped and confirms it is active, so a
# failure to resume backups is never silent. It runs at most once: a failure is
# reported instead of being retried, and it is only marked done after the timer
# is confirmed active.
restore_backup_timer() {
  if [ "${backup_timer_was_active}" -eq 0 ] || [ "${backup_timer_restored}" -eq 1 ]; then
    return 0
  fi
  if [ "${backup_timer_restore_attempted}" -eq 1 ]; then
    return 1
  fi
  backup_timer_restore_attempted=1
  local state=""
  if "${SYSTEMCTL}" start "${backup_timer}" 2>/dev/null &&
    state="$(unit_active_state "${backup_timer}")" &&
    [ "${state}" = "active" ]; then
    backup_timer_restored=1
    return 0
  fi
  return 1
}

cleanup_on_exit() {
  local status=$?
  if [ "${staging_created}" -eq 1 ]; then
    rm -rf "${staging}"
  fi
  # Once the new release is active and healthy, a later failure (prune or timer
  # restart) must not trigger the rollback path.
  if [ "${status}" -ne 0 ] && [ "${deploy_completed}" -eq 0 ]; then
    if [ "${release_created}" -eq 1 ] && [ -d "${release_dir}" ]; then
      local active
      active="$(readlink -f "${current_link}" 2>/dev/null || true)"
      if [ "$(readlink -f "${release_dir}")" != "${active}" ]; then
        rm -rf "${release_dir}"
      fi
    fi
    # Restore helpers this attempt replaced unless a rollback already did it.
    if ! restore_previous_helpers; then
      echo "previous helpers kept in ${attempt_dir} for manual recovery" >&2
      keep_attempt=1
    fi
    # Keep the tarball when the preflight aborted before any mutation, so the
    # operator can retry once the running backup finishes.
    if [ "${mutation_started}" -eq 1 ]; then
      rm -f "${archive}"
    fi
  fi
  if [ "${keep_attempt}" -eq 0 ]; then
    rm -rf "${attempt_dir}"
  fi
  if ! restore_backup_timer; then
    echo "the PostgreSQL backup timer could not be restored; start it manually" >&2
    if [ "${status}" -eq 0 ]; then
      status=1
    fi
  fi
  # `exit` (not `return`) so a timer-restore failure on an otherwise successful
  # path actually changes the script's exit status.
  exit "${status}"
}

run_migrations() {
  local dir="$1"
  if [ ! -f "${secrets_file}" ]; then
    echo "migration secrets not found at ${secrets_file}" >&2
    return 1
  fi
  if [ ! -f "${env_dir}/db.env" ]; then
    echo "database environment not found at ${env_dir}/db.env" >&2
    return 1
  fi

  # Production validation in config.Load reads the same runtime environment.
  # The migrator reuses DB_HOST/DB_PORT/DB_NAME/DB_SSLMODE/DB_SSLROOTCERT from
  # db.env, so moving the cluster to RDS needs no change here; only the role is
  # swapped to the DDL role and DATABASE_URL is cleared to force DB_* assembly.
  (
    set -a
    # shellcheck disable=SC1090
    . "${secrets_file}"
    # shellcheck disable=SC1090
    . "${env_dir}/db.env"
    if [ -f "${env_dir}/api.env" ]; then
      # shellcheck disable=SC1090
      . "${env_dir}/api.env"
    fi
    if [ -f "${env_dir}/smtp.env" ]; then
      # shellcheck disable=SC1090
      . "${env_dir}/smtp.env"
    fi
    set +a
    : "${CESTIA_MIGRATE_PASSWORD:?missing migrate password on the data disk}"
    export DB_USER="cestia_migrate"
    export DB_PASS="${CESTIA_MIGRATE_PASSWORD}"
    unset DATABASE_URL
    "${dir}/cestia-migrate"
  )
}

validate_release() {
  local dir="$1"
  local required=(cestia-api cestia-migrate restore-postgres.sh backup-postgres.sh web/index.html)
  local file
  for file in "${required[@]}"; do
    if [ ! -e "${dir}/${file}" ]; then
      echo "release is missing ${file}" >&2
      exit 1
    fi
  done
}

install_release() {
  local dir="$1"
  if [ -z "${CESTIA_TEST_ALLOW_NONROOT:-}" ]; then
    chown -R root:root "${dir}"
  fi
  find "${dir}" -type d -exec chmod 0755 {} +
  find "${dir}" -type f -exec chmod 0644 {} +
  chmod 0755 "${dir}/cestia-api" "${dir}/cestia-migrate"
  # Stable paths outside the release symlink so rollback keeps working and the
  # runbook always finds the right tools.
  install -d -m 0755 "$(dirname "${restore_path}")"
  install -m 0755 "${dir}/restore-postgres.sh" "${restore_path}"
  restore_replaced=1
  install -m 0750 "${dir}/backup-postgres.sh" "${backup_path}"
  backup_replaced=1
}

switch_current() {
  local target="$1"
  ln -sfn "${target}" "${current_link}.next"
  mv -Tf "${current_link}.next" "${current_link}"
}

wait_for_health() {
  local attempt
  for attempt in $(seq 1 30); do
    if curl -fsS --max-time 3 "${health_url}" >/dev/null; then
      return 0
    fi
    sleep 2
  done
  return 1
}

# Switches the active release and verifies it. Any failure (start, Caddy reload,
# or health) returns non-zero so the caller can roll back.
activate_release() {
  local target="$1"
  "${SYSTEMCTL}" stop "${api_service}" 2>/dev/null || true
  switch_current "${target}"
  "${SYSTEMCTL}" start "${api_service}" || return 1
  "${SYSTEMCTL}" reload caddy 2>/dev/null || "${SYSTEMCTL}" restart caddy || return 1
  wait_for_health || return 1
}

# Keeps only the active release and the previous one for rollback, identified by
# their resolved paths rather than by age, and removes stale staging directories.
prune_releases() {
  local keep_current keep_previous candidate resolved
  keep_current="$(readlink -f "${current_link}" 2>/dev/null || true)"
  keep_previous=""
  if [ -n "${previous_release}" ] && [ -d "${previous_release}" ]; then
    keep_previous="$(readlink -f "${previous_release}")"
  fi

  while read -r candidate; do
    resolved="$(readlink -f "${candidate}")"
    if [ "${resolved}" = "${keep_current}" ] || [ "${resolved}" = "${keep_previous}" ]; then
      continue
    fi
    rm -rf "${candidate}"
  done < <(find "${root}/releases" -mindepth 1 -maxdepth 1 -type d ! -name '.*' -print)

  find "${root}/releases" -mindepth 1 -maxdepth 1 -type d -name '.staging-*' -exec rm -rf {} +
}

# Reject an existing release and take the shared maintenance lock before
# installing the trap, so a failed or concurrent attempt never deletes a
# preexisting release.
if [ -e "${release_dir}" ]; then
  echo "release ${release_id} already exists at ${release_dir}" >&2
  exit 1
fi

install -d -m 0755 "$(dirname "${LOCK_FILE}")"
exec 9>"${LOCK_FILE}"
if ! flock -n 9; then
  echo "another maintenance operation (deploy, restore or backup) is running" >&2
  exit 1
fi

trap cleanup_on_exit EXIT

# A backup started earlier must never write a dump while the migration or the
# release swap runs. The preflight fails closed on any state it cannot confirm.
timer_state=""
if ! timer_state="$(unit_active_state "${backup_timer}")"; then
  echo "could not read the state of ${backup_timer}; aborting before touching the release" >&2
  exit 1
fi
if [ "${timer_state}" != "active" ]; then
  echo "${backup_timer} is ${timer_state}, not active; repair backups before deploying" >&2
  exit 1
fi
backup_timer_was_active=1

if ! "${SYSTEMCTL}" stop "${backup_timer}" 2>/dev/null; then
  echo "could not stop ${backup_timer}; aborting before touching the release" >&2
  exit 1
fi
if ! timer_state="$(unit_active_state "${backup_timer}")"; then
  echo "could not confirm ${backup_timer} stopped; aborting before touching the release" >&2
  exit 1
fi
if [ "${timer_state}" != "inactive" ]; then
  echo "${backup_timer} is still ${timer_state} after stop; aborting before touching the release" >&2
  exit 1
fi

service_state=""
if ! service_state="$(unit_active_state "${backup_service}")"; then
  echo "could not read the state of ${backup_service}; aborting before touching the release" >&2
  exit 1
fi
case "${service_state}" in
  inactive | failed) ;;
  *)
    echo "a PostgreSQL backup is ${service_state}; wait for it to finish before deploying" >&2
    exit 1
    ;;
esac

# Past the preflight: any later failure may have changed the release and the
# uploaded tarball can be cleaned up.
mutation_started=1

# Back up the stable helpers this attempt will replace, under a unique path.
rm -rf "${attempt_dir}"
mkdir -p "${attempt_dir}"
if [ -f "${restore_path}" ]; then
  cp -p "${restore_path}" "${attempt_dir}/restore-postgres.sh"
fi
if [ -f "${backup_path}" ]; then
  cp -p "${backup_path}" "${attempt_dir}/backup-postgres.sh"
fi

rm -rf "${staging}"
mkdir -p "${staging}"
staging_created=1
tar -xzf "${archive}" -C "${staging}"
validate_release "${staging}"
install_release "${staging}"
mv -Tf "${staging}" "${release_dir}"
staging_created=0
release_created=1

previous_release="$(readlink -f "${current_link}" 2>/dev/null || true)"

run_migrations "${release_dir}"

if ! activate_release "${release_dir}"; then
  echo "activation of ${release_id} failed" >&2
  if [ -n "${previous_release}" ] && [ -d "${previous_release}" ]; then
    if restore_previous_helpers; then
      if activate_release "${previous_release}"; then
        echo "rolled back to ${previous_release}" >&2
      else
        echo "manual intervention required: neither release is healthy" >&2
      fi
    else
      echo "manual intervention required: the previous helper could not be restored" >&2
    fi
  fi
  exit 1
fi

prune_releases
rm -f "${archive}"
deploy_completed=1

if ! restore_backup_timer; then
  echo "release ${release_id} is deployed, but the backup timer could not be restarted" >&2
  exit 1
fi

echo "release ${release_id} deployed and healthy"

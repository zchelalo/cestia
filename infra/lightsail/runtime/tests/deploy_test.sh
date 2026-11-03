#!/usr/bin/env bash
# Exercises deploy-release.sh with isolated paths and stubbed systemctl/curl.
#
# It checks that a failed or concurrent attempt never damages the active release
# or the rollback, and that the stable helpers are restored. It never touches
# production.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/../scripts/deploy-release.sh"

failures=0
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

root="$(mktemp -d)"
bin="$(mktemp -d)"
trap 'rm -rf "${root}" "${bin}"' EXIT

restore="${root}/bin/restore-postgres.sh"
backup="${root}/bin/pg-backup.sh"
lock="${root}/maintenance.lock"
secrets="${root}/secrets.env"
migrate_fail="${root}/migrate-fail"

mkdir -p "${root}/releases" "${root}/bin" "${root}/web" "${root}/env"
printf 'CESTIA_MIGRATE_PASSWORD=secret\n' >"${secrets}"
printf 'APP_ENV=test\n' >"${root}/env/api.env"
: >"${root}/env/smtp.env"
printf '%s\n' 'DB_HOST=127.0.0.1' 'DB_PORT=5432' 'DB_NAME=cestia' 'DB_USER=cestia_app' 'DB_PASS=app' 'DB_SSLMODE=disable' >"${root}/env/db.env"

# Stubs. The systemctl stub tracks timer/service ActiveState in files and exposes
# failure switches (read/stop/start), so the preflight and its cleanup are
# exercised with the same commands production runs. Unknown subcommands fail
# instead of being silently treated as success.
state_dir="${root}/systemctl-state"
mkdir -p "${state_dir}"
cat >"${bin}/systemctl" <<STUB
#!/bin/sh
state="${state_dir}"
unit=""
for arg in "\$@"; do
  case "\$arg" in
    -*) ;;
    *) unit="\$arg" ;;
  esac
done
case "\$1" in
  show)
    case "\$unit" in
      *pg-backup.timer)
        [ -f "\$state/timer-read-fail" ] && { echo "timer read failed" >&2; exit 1; }
        if [ -f "\$state/timer-state" ]; then cat "\$state/timer-state"; else echo inactive; fi
        ;;
      *pg-backup.service)
        [ -f "\$state/service-read-fail" ] && { echo "service read failed" >&2; exit 1; }
        if [ -f "\$state/service-state" ]; then cat "\$state/service-state"; else echo inactive; fi
        ;;
      *) echo inactive ;;
    esac
    exit 0
    ;;
  stop)
    case "\$unit" in
      *pg-backup.timer)
        [ -f "\$state/timer-stop-fail" ] && { echo "timer stop failed" >&2; exit 1; }
        if [ ! -f "\$state/timer-stop-ineffective" ]; then echo inactive >"\$state/timer-state"; fi
        ;;
      *pg-backup.service)
        echo inactive >"\$state/service-state"
        ;;
    esac
    exit 0
    ;;
  start)
    if [ "\$unit" = "cestia-api.service" ] &&
      [ -f "\$state/reject-new-release" ] &&
      grep -q 'new-release' "${backup}" 2>/dev/null; then
      exit 1
    fi
    case "\$unit" in
      *pg-backup.timer)
        [ -f "\$state/timer-start-fail" ] && { echo "timer start failed" >&2; exit 1; }
        echo active >"\$state/timer-state"
        ;;
      *pg-backup.service)
        echo active >"\$state/service-state"
        ;;
    esac
    exit 0
    ;;
  reload | restart)
    exit 0
    ;;
  *)
    echo "systemctl stub: unexpected subcommand \$1" >&2
    exit 1
    ;;
esac
STUB
printf '#!/bin/sh\nexit 0\n' >"${bin}/curl"
chmod +x "${bin}/systemctl" "${bin}/curl"
export PATH="${bin}:${PATH}"

clear_systemctl_state() {
  rm -f \
    "${state_dir}/timer-state" "${state_dir}/service-state" \
    "${state_dir}/timer-read-fail" "${state_dir}/service-read-fail" \
    "${state_dir}/timer-stop-fail" "${state_dir}/timer-stop-ineffective" \
    "${state_dir}/timer-start-fail" "${state_dir}/reject-new-release"
}

timer_active() { [ "$(cat "${state_dir}/timer-state" 2>/dev/null)" = "active" ]; }

# Every preflight abort must leave the installed helpers, the release link and
# the candidate release exactly as they were.
no_mutation_checks() {
  local label="$1" id="$2"
  check "${label}: restore helper untouched" \
    bash -c "[ \"$(cat "${restore}")\" = \"old-restore\" ]"
  check "${label}: backup script untouched" \
    bash -c "[ \"$(cat "${backup}")\" = \"old-backup\" ]"
  check "${label}: current untouched" \
    bash -c "[ \"$(readlink -f "${root}/current")\" = \"${root}/releases/${existing_id}\" ]"
  check "${label}: no release created" \
    bash -c "[ ! -d \"${root}/releases/${id}\" ]"
}

make_release() {
  local archive="$1"
  local work
  work="$(mktemp -d)"
  printf '#!/bin/sh\nexit 0\n' >"${work}/cestia-api"
  printf '#!/bin/sh\n[ -f "%s" ] && exit 1\nexit 0\n' "${migrate_fail}" >"${work}/cestia-migrate"
  printf '#!/bin/sh\n# new-release\nexit 0\n' >"${work}/restore-postgres.sh"
  printf '#!/bin/sh\n# new-release\nexit 0\n' >"${work}/backup-postgres.sh"
  mkdir -p "${work}/web"
  printf '<html></html>' >"${work}/web/index.html"
  chmod +x "${work}/cestia-api" "${work}/cestia-migrate" "${work}/restore-postgres.sh" "${work}/backup-postgres.sh"
  tar -czf "${archive}" -C "${work}" .
  rm -rf "${work}"
}

run_deploy() {
  local archive="$1"
  local id="$2"
  CESTIA_ROOT="${root}" \
    CESTIA_RESTORE_PATH="${restore}" \
    CESTIA_BACKUP_PATH="${backup}" \
    CESTIA_SECRETS_FILE="${secrets}" \
    CESTIA_ENV_DIR="${root}/env" \
    CESTIA_SYSTEMCTL="${bin}/systemctl" \
    CESTIA_LOCK_FILE="${lock}" \
    CESTIA_TEST_ALLOW_NONROOT=1 \
    bash "${DEPLOY_SCRIPT}" "${archive}" "${id}"
}

# Captures deploy output and status without aborting the test on failure.
deploy_output=""
deploy_status=0
run_deploy_capture() {
  set +e
  deploy_output="$(run_deploy "$1" "$2" 2>&1)"
  deploy_status=$?
  set -e
}

# An existing release and helpers we must not lose.
existing_id="aaaaaaa1"
mkdir -p "${root}/releases/${existing_id}"
printf 'existing' >"${root}/releases/${existing_id}/marker"
printf 'old-restore' >"${restore}"
printf 'old-backup' >"${backup}"
ln -sfn "${root}/releases/${existing_id}" "${root}/current"
: >"${root}/releases/${existing_id}/cestia-api"

archive="${root}/release.tar.gz"
make_release "${archive}"

echo "scenario: existing release id is preserved"
if run_deploy "${archive}" "${existing_id}"; then
  check "existing release is refused" false
else
  check "existing release is refused" true
fi
check "existing release marker kept" test -f "${root}/releases/${existing_id}/marker"
check "current still points to the existing release" \
  bash -c "[ \"$(readlink -f "${root}/current")\" = \"${root}/releases/${existing_id}\" ]"

echo "scenario: concurrent deployment is refused"
exec 8>"${lock}"
flock -n 8
if run_deploy "${archive}" "bbbbbbb2"; then
  check "locked deploy is refused" false
else
  check "locked deploy is refused" true
fi
check "no candidate release created while locked" \
  bash -c "[ ! -d \"${root}/releases/bbbbbbb2\" ]"
check "existing release marker kept while locked" \
  test -f "${root}/releases/${existing_id}/marker"
exec 8>&-

# Runs a deploy expected to be refused by the preflight, then asserts nothing was
# mutated and the tarball survives for a retry.
preflight_refused() {
  local id="$1" label="$2"
  make_release "${archive}"
  if run_deploy "${archive}" "${id}"; then
    check "${label}: deploy refused" false
  else
    check "${label}: deploy refused" true
  fi
  no_mutation_checks "${label}" "${id}"
  check "${label}: tarball kept for retry" test -f "${archive}"
}

timer_not_active() { [ "$(cat "${state_dir}/timer-state" 2>/dev/null)" != "active" ]; }
timer_failure_reported() { printf '%s' "${deploy_output}" | grep -q 'backup timer'; }
success_not_reported() { ! printf '%s' "${deploy_output}" | grep -q 'deployed and healthy'; }

echo "scenario: a running backup aborts the deploy before any change"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
echo active >"${state_dir}/service-state"
preflight_refused "eeeeeee1" "running backup"
check "running backup: timer restored active" timer_active

echo "scenario: an inactive timer aborts the deploy"
clear_systemctl_state
preflight_refused "eeeeeee2" "inactive timer"
check "inactive timer: timer left inactive" timer_not_active

echo "scenario: an unreadable timer aborts the deploy"
clear_systemctl_state
touch "${state_dir}/timer-read-fail"
preflight_refused "eeeeeee3" "unreadable timer"
check "unreadable timer: timer left inactive" timer_not_active

echo "scenario: a failed timer stop aborts the deploy"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
touch "${state_dir}/timer-stop-fail"
preflight_refused "eeeeeee4" "timer stop failure"
check "timer stop failure: timer restored active" timer_active

echo "scenario: a timer that stays active after stop aborts the deploy"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
touch "${state_dir}/timer-stop-ineffective"
preflight_refused "eeeeeee5" "timer stop ineffective"
check "timer stop ineffective: timer restored active" timer_active

echo "scenario: an unreadable backup service aborts the deploy"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
touch "${state_dir}/service-read-fail"
preflight_refused "eeeeeee6" "unreadable backup service"
check "unreadable backup service: timer restored active" timer_active

echo "scenario: an activating backup service aborts the deploy"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
echo activating >"${state_dir}/service-state"
preflight_refused "eeeeeee7" "activating backup service"
check "activating backup service: timer restored active" timer_active

echo "scenario: a deactivating backup service aborts the deploy"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
echo deactivating >"${state_dir}/service-state"
preflight_refused "eeeeeee8" "deactivating backup service"
check "deactivating backup service: timer restored active" timer_active

echo "scenario: migration failure restores the helpers and removes the release"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
: >"${migrate_fail}"
make_release "${archive}"
if run_deploy "${archive}" "ccccccc3"; then
  check "migration failure aborts" false
else
  check "migration failure aborts" true
fi
check "restore helper restored after migration failure" \
  bash -c "[ \"$(cat "${restore}")\" = \"old-restore\" ]"
check "backup script restored after migration failure" \
  bash -c "[ \"$(cat "${backup}")\" = \"old-backup\" ]"
check "failed release removed after migration failure" \
  bash -c "[ ! -d \"${root}/releases/ccccccc3\" ]"
check "no staging left after migration failure" \
  bash -c "[ -z \"$(find "${root}/releases" -maxdepth 1 -name '.staging-*')\" ]"
check "current untouched after migration failure" \
  bash -c "[ \"$(readlink -f "${root}/current")\" = \"${root}/releases/${existing_id}\" ]"
check "backup timer restored after migration failure" timer_active
rm -f "${migrate_fail}"

echo "scenario: a failed migration plus a failed timer restart is reported"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
touch "${state_dir}/timer-start-fail"
: >"${migrate_fail}"
make_release "${archive}"
run_deploy_capture "${archive}" "ccccccc9"
check "failed migration plus timer restart fails" test "${deploy_status}" -ne 0
check "failed migration plus timer restart reports the timer" timer_failure_reported
check "restore helper restored after failed migration plus timer restart" \
  bash -c "[ \"$(cat "${restore}")\" = \"old-restore\" ]"
check "timer left inactive after failed restart" timer_not_active
rm -f "${migrate_fail}" "${state_dir}/timer-start-fail"

echo "scenario: activation failure rolls back to the previous release"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
touch "${state_dir}/reject-new-release"
make_release "${archive}"
if run_deploy "${archive}" "ddddddd4"; then
  check "activation failure aborts" false
else
  check "activation failure aborts" true
fi
check "restore helper restored after activation failure" \
  bash -c "[ \"$(cat "${restore}")\" = \"old-restore\" ]"
check "rollback points to the previous release" \
  bash -c "[ \"$(readlink -f "${root}/current")\" = \"${root}/releases/${existing_id}\" ]"
check "failed release removed after activation failure" \
  bash -c "[ ! -d \"${root}/releases/ddddddd4\" ]"
check "backup timer restored after activation failure" timer_active

echo "scenario: successful deployment restores the backup timer"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
echo failed >"${state_dir}/service-state"
make_release "${archive}"
if run_deploy "${archive}" "fffffff6"; then
  check "successful deployment succeeds" true
else
  check "successful deployment succeeds" false
fi
check "current points to the new release" \
  bash -c "[ \"$(readlink -f "${root}/current")\" = \"${root}/releases/fffffff6\" ]"
check "new restore helper installed after success" \
  bash -c "grep -q 'new-release' "${restore}""
check "new backup script installed after success" \
  bash -c "grep -q 'new-release' "${backup}""
check "backup timer restored after success" timer_active

echo "scenario: a healthy deploy with a failed timer restart is not reported as success"
clear_systemctl_state
echo active >"${state_dir}/timer-state"
touch "${state_dir}/timer-start-fail"
make_release "${archive}"
run_deploy_capture "${archive}" "fffffff7"
check "healthy deploy plus timer restart fails" test "${deploy_status}" -ne 0
check "healthy deploy plus timer restart does not claim success" success_not_reported
check "healthy deploy plus timer restart reports the timer" timer_failure_reported
check "new release stays active after timer restart failure" \
  bash -c "[ \"$(readlink -f "${root}/current")\" = \"${root}/releases/fffffff7\" ]"
check "new restore helper stays installed after timer restart failure" \
  bash -c "grep -q 'new-release' "${restore}""
check "timer left inactive after healthy deploy restart failure" timer_not_active

if [ "${failures}" -ne 0 ]; then
  echo "${failures} check(s) failed" >&2
  exit 1
fi
echo "all deploy scenarios passed"

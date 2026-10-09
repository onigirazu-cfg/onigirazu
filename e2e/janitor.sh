#!/usr/bin/env bash
# Deletes e2e VMs left behind by cancelled or crashed runs.
# Only touches VMs directly inside the e2e folder whose name starts with the
# e2e prefix: with GH_TOKEN and GITHUB_REPOSITORY set, VMs of finished runs
# are removed at once and those of runs in progress kept; VMs whose run
# cannot be asked about are removed after TTL_HOURS. Also removes work
# directories of killed runs on this runner.
#
# Environment: GOVC_URL GOVC_USERNAME GOVC_PASSWORD GOVC_INSECURE,
#   E2E_DATACENTER, E2E_FOLDER; optional TTL_HOURS (default 3), DRY_RUN=1;
#   MIKROTIK_API_* for the DHCP leases (dhcp-leases.sh)
set -euo pipefail

PREFIX="tmp-e2e-onigirazu-"
TTL_HOURS="${TTL_HOURS:-3}"
: "${E2E_DATACENTER:?}" "${E2E_FOLDER:?}"
folder="/$E2E_DATACENTER/vm/$E2E_FOLDER"

cutoff="$(date -u -d "-$TTL_HOURS hours" +%s 2>/dev/null || date -u -v-"$TTL_HOURS"H +%s)"
found=0 deleted=0

while IFS=$'\t' read -r name created note; do
  [ -n "$name" ] || continue
  case "$name" in "$PREFIX"*) ;; *) continue ;; esac
  found=$((found + 1))
  ts="$(date -u -d "$created" +%s 2>/dev/null || date -u -j -f '%Y-%m-%dT%H:%M:%S' "${created%%.*}" +%s)"
  # tmp-e2e-onigirazu-<run id>-<attempt>-...: a VM of a finished run (a
  # cancelled one gets no cleanup) goes at once, whatever its age; the TTL
  # is for VMs whose run cannot be asked about
  run_id="${name#"$PREFIX"}" run_id="${run_id%%-*}"
  status=""
  if [ -n "${GH_TOKEN:-}" ] && [ -n "${GITHUB_REPOSITORY:-}" ] && [[ "$run_id" =~ ^[0-9]+$ ]]; then
    status="$(curl -fsS -H "Authorization: Bearer $GH_TOKEN" \
      "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runs/$run_id" 2>/dev/null | jq -r '.status // empty')"
  fi
  case "$status" in
  completed)
    # keep_vms marks its VMs: they get the TTL like the rest
    if [[ "$note" == "e2e keep_vms"* ]] && [ "$ts" -gt "$cutoff" ]; then
      echo "keep   $name (kept by its run, created $created)"
      continue
    fi
    echo "delete $name (run $run_id finished)" ;;
  "")
    if [ "$ts" -gt "$cutoff" ]; then
      echo "keep   $name (created $created)"
      continue
    fi
    echo "delete $name (created $created)" ;;
  *)
    echo "keep   $name (run $run_id $status)"
    continue ;;
  esac
  if [ -z "${DRY_RUN:-}" ]; then
    govc vm.power -off -force "$folder/$name" >/dev/null 2>&1 || true
    # two janitors may run at once (schedule + dispatch): the other one won
    govc vm.destroy "$folder/$name" 2>/dev/null || echo "       already gone"
    deleted=$((deleted + 1))
  fi
done < <(govc vm.info -json "$folder/*" 2>/dev/null | jq -r '
  (.virtualMachines // .VirtualMachines // [])[] | [.name, (.config.createDate // ""), (.config.annotation // "")] | @tsv')

echo "e2e VMs found: $found, deleted: $deleted"

# base templates: the newest 2 per image key stay (image/build.sh keeps as
# many); older ones that a running linked clone kept alive go now
for key in $(govc find "$folder" -type m -name "e2e-base-*" 2>/dev/null | sed -E 's|.*/e2e-base-([a-z0-9]+)-.*|\1|' | sort -u); do
  govc find "$folder" -type m -name "e2e-base-$key-*" | awk '{print substr($0, length($0) - 12) " " $0}' | sort | cut -d' ' -f2- | head -n -2 | while read -r old; do
    echo "delete ${old##*/} (older base template)"
    [ -n "${DRY_RUN:-}" ] || govc vm.destroy "$old" 2>/dev/null || echo "       still in use by a linked clone"
  done
done

# DHCP leases of VMs that no longer exist (killed runs, hard power-offs)
[ -n "${DRY_RUN:-}" ] || "$(dirname "$0")/dhcp-leases.sh" sweep

# Work directories of runs that were killed (a cancelled job gets no EXIT
# trap): they hold the run's private key. At least 3 hours old even when
# purging VMs, so runs in progress keep theirs.
dir_ttl=$(( TTL_HOURS > 3 ? TTL_HOURS : 3 ))
stale="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -type d -name 'onigirazu-e2e.*' -user "$(id -un)" -mmin +$((dir_ttl * 60)) 2>/dev/null)"
if [ -n "$stale" ]; then
  echo "removing $(wc -l <<<"$stale" | tr -d ' ') stale work dir(s)"
  [ -n "${DRY_RUN:-}" ] || xargs rm -rf <<<"$stale"
fi

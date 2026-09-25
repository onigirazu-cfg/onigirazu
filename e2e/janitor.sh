#!/usr/bin/env bash
# Deletes e2e VMs left behind by cancelled or crashed runs.
# Only touches VMs directly inside the e2e folder whose name starts with the
# e2e prefix and that were created more than TTL_HOURS ago.
#
# Environment: GOVC_URL GOVC_USERNAME GOVC_PASSWORD GOVC_INSECURE,
#   E2E_DATACENTER, E2E_FOLDER; optional TTL_HOURS (default 3), DRY_RUN=1
set -euo pipefail

PREFIX="tmp-e2e-onigirazu-"
TTL_HOURS="${TTL_HOURS:-3}"
: "${E2E_DATACENTER:?}" "${E2E_FOLDER:?}"
folder="/$E2E_DATACENTER/vm/$E2E_FOLDER"

cutoff="$(date -u -d "-$TTL_HOURS hours" +%s 2>/dev/null || date -u -v-"$TTL_HOURS"H +%s)"
found=0 deleted=0

while IFS=$'\t' read -r name created; do
  [ -n "$name" ] || continue
  case "$name" in "$PREFIX"*) ;; *) continue ;; esac
  found=$((found + 1))
  ts="$(date -u -d "$created" +%s 2>/dev/null || date -u -j -f '%Y-%m-%dT%H:%M:%S' "${created%%.*}" +%s)"
  if [ "$ts" -gt "$cutoff" ]; then
    echo "keep   $name (created $created)"
    continue
  fi
  echo "delete $name (created $created)"
  if [ -z "${DRY_RUN:-}" ]; then
    govc vm.power -off -force "$folder/$name" >/dev/null 2>&1 || true
    govc vm.destroy "$folder/$name"
    deleted=$((deleted + 1))
  fi
done < <(govc vm.info -json "$folder/*" 2>/dev/null | jq -r '
  (.virtualMachines // .VirtualMachines // [])[] | [.name, (.config.createDate // "")] | @tsv')

echo "e2e VMs found: $found, deleted: $deleted"

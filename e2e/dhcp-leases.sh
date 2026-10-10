#!/usr/bin/env bash
# DHCP leases of e2e VMs on the MikroTik that serves the e2e network: the VMs
# are powered off hard and destroyed, so nothing releases their leases and the
# pool ran out. The runner removes them by MAC through the REST API.
#
#   dhcp-leases.sh macs VM_PATH...      print the MACs of VMs (before destroy)
#   dhcp-leases.sh remove MAC...        remove the dynamic leases of these MACs
#   dhcp-leases.sh sweep                remove dynamic leases of VMware MACs
#                                       (00:50:56) that no VM in vCenter has
#
# Environment: MIKROTIK_API_URL (https://router), MIKROTIK_API_USER,
# MIKROTIK_API_PASSWORD; MIKROTIK_API_INSECURE=1 for a self-signed
# certificate. Without them the script says so and does nothing. `sweep` and
# `macs` need govc (GOVC_* set). Only dynamic leases are ever touched.
set -euo pipefail

cmd="${1:-}"; shift || true

macs() {  # VM paths -> MACs, one per line, upper case
  [ $# -gt 0 ] || return 0
  govc vm.info -json "$@" 2>/dev/null |
    jq -r '(.virtualMachines // .VirtualMachines // [])[].config.hardware.device[]? | .macAddress? // empty' |
    tr 'a-f' 'A-F' | sort -u
}

api() {  # METHOD PATH
  local insecure=()
  [ "${MIKROTIK_API_INSECURE:-}" = 1 ] && insecure=(-k)
  curl -fsS "${insecure[@]}" -u "$MIKROTIK_API_USER:$MIKROTIK_API_PASSWORD" -X "$1" \
    -H 'Content-Type: application/json' "${MIKROTIK_API_URL%/}/rest$2"
}

need_api() {
  if [ -z "${MIKROTIK_API_URL:-}" ] || [ -z "${MIKROTIK_API_USER:-}" ] || [ -z "${MIKROTIK_API_PASSWORD:-}" ]; then
    echo "dhcp leases: MIKROTIK_API_URL/USER/PASSWORD not set, leases are left to expire"
    exit 0
  fi
}

# remove_leases reads "id<TAB>mac<TAB>address" lines and deletes each lease
remove_leases() {
  local n=0 id mac addr
  while IFS=$'\t' read -r id mac addr; do
    [ -n "$id" ] || continue
    if api DELETE "/ip/dhcp-server/lease/$id" >/dev/null; then
      echo "released $addr ($mac)"; n=$((n + 1))
    else
      echo "could not remove the lease of $mac ($addr)"
    fi
  done
  echo "dhcp leases removed: $n"
}

case "$cmd" in
macs)
  macs "$@"
  ;;
remove)
  need_api
  [ $# -gt 0 ] || { echo "dhcp leases: no MACs"; exit 0; }
  for mac in "$@"; do
    api GET "/ip/dhcp-server/lease?mac-address=$(tr 'a-f' 'A-F' <<<"$mac")" |
      jq -r '.[] | select(.dynamic == "true") | [.[".id"], .["mac-address"], .address] | @tsv'
  done | remove_leases
  ;;
sweep)
  need_api
  # every MAC of every VM vCenter shows us; an empty list means the listing
  # failed, and nothing is removed on a guess
  known="$(govc find / -type m 2>/dev/null | xargs -r govc vm.info -json 2>/dev/null |
    jq -r '(.virtualMachines // .VirtualMachines // [])[].config.hardware.device[]? | .macAddress? // empty' | tr 'a-f' 'A-F' | sort -u)"
  [ -n "$known" ] || { echo "dhcp leases: cannot list the VMs in vCenter, nothing swept"; exit 0; }
  api GET "/ip/dhcp-server/lease" |
    jq -r '.[] | select(.dynamic == "true" and (.["mac-address"] | startswith("00:50:56"))) | [.[".id"], .["mac-address"], .address] | @tsv' |
    grep -vF -f <(printf '%s\n' "$known" | sed 's/.*/\t&\t/') | remove_leases
  ;;
*)
  echo "usage: $0 macs VM...|remove MAC...|sweep" >&2; exit 2
  ;;
esac

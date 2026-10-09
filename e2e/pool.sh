#!/usr/bin/env bash
# shellcheck disable=SC2154 # TF_VAR_* come from the caller, images_json from images.sh
# Pre-warmed e2e VMs: pool-e2e-<id>-<key>, linked clones of the current e2e
# base template, powered on and waiting. A run claims one per image key by
# renaming it (atomic in vCenter, so parallel shards never take the same one)
# and sets its own key through guestinfo, which the base image applies within
# seconds (e2e-access-refresh timer, image/prepare.sh).
#
#   pool.sh fill                          create VMs up to POOL_SIZE per key
#   pool.sh claim KEY NAME PUBKEY HOST    rename one, give it the key; prints its IP
#   pool.sh purge                         remove VMs of other templates or without an address
#
# Environment: as run.sh (TF_VAR_*, E2E_IMAGES); POOL_SIZE (default 2);
# only e2e base templates built with the refresh timer are pooled.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
cmd="${1:-}"; shift || true
POOL_SIZE="${POOL_SIZE:-2}"
E2E_IMAGES="${E2E_IMAGES:-u2404=ubuntu-24.04 u2604=ubuntu-26.04}"
export GOVC_URL="$TF_VAR_vsphere_server" GOVC_USERNAME="$TF_VAR_vsphere_user" GOVC_PASSWORD="$TF_VAR_vsphere_password" GOVC_INSECURE=1
export GOVC_DATACENTER="$TF_VAR_datacenter"
FOLDER="/$TF_VAR_datacenter/vm/$TF_VAR_folder"
# shellcheck source=e2e/images.sh
. "$HERE/images.sh"
E2E_BASE=1 resolve_images >/dev/null

vm_json() { govc vm.info -json -e "$1" 2>/dev/null | jq -c '(.virtualMachines // .VirtualMachines // [])[0] // {}'; }
# guest_ipv4 waits up to a minute for the address of the VM's NIC as the
# tools report it (vm.ip answered with something that was not an address)
guest_ipv4() {
  local i ip
  for i in $(seq 30); do
    # the virtual NIC only (deviceConfigId >= 0): the tools also report the
    # guest's docker bridge, which came first
    ip="$(govc vm.info -json "$1" 2>/dev/null | jq -r '(.virtualMachines // .VirtualMachines // [])[0].guest.net[]? | select((.deviceConfigId // -1) >= 0) | .ipAddress[]? // empty' |
      grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | grep -vE '^(127\.|169\.254\.)' | head -1)"
    [ -n "$ip" ] && { echo "$ip"; return 0; }
    [ "$i" = 30 ] || sleep 2
  done
  return 1
}
template_of() { jq -r '[.config.extraConfig[]? | select(.key == "guestinfo.e2e_template") | .value] | first // ""' <<<"$1"; }
pooled_template() {  # key -> the template the pool clones, "" when none qualifies
  local tpl
  tpl="$(jq -r --arg k "$1" '.[$k] // ""' <<<"$images_json")"
  case "$tpl" in e2e-base-*) ;; *) return ;; esac
  govc vm.info -json "$FOLDER/$tpl" 2>/dev/null | jq -r '(.virtualMachines // .VirtualMachines // [])[0].config.annotation // ""' | grep -q e2e-access-refresh && echo "$tpl"
}
pool_vms() { govc find "$FOLDER" -type m -name "pool-e2e-*-$1" 2>/dev/null; }

case "$cmd" in
claim)
  key="$1" name="$2" pub="$3" host="$4"
  # the address is the only thing on stdout: govc prints task progress there
  exec 3>&1 1>&2
  tpl="$(pooled_template "$key")"
  [ -n "$tpl" ] || exit 0
  while IFS= read -r vm; do
    [ -n "$vm" ] || continue
    info="$(vm_json "$vm")"
    [ "$(template_of "$info")" = "$tpl" ] || continue
    [ "$(jq -r .runtime.powerState <<<"$info")" = poweredOn ] || continue
    # the rename is the claim: it fails when another shard renamed it first
    govc object.rename "$vm" "$name" >/dev/null 2>&1 || continue
    new="$FOLDER/$name"
    govc vm.change -vm "$new" -e "guestinfo.e2e_authorized_key=$pub" -e "guestinfo.e2e_hostname=$host" \
      -annotation "e2e run: ${RUN_URL:-local}; was ${vm##*/}" >/dev/null
    if ip="$(guest_ipv4 "$new")"; then echo "$ip" >&3; exit 0; fi
    echo "pool: ${vm##*/} reports no IPv4 address, removed" >&2
    govc vm.destroy "$new" >/dev/null 2>&1 || true
  done < <(pool_vms "$key" | sort -R)
  ;;
fill)
  command -v terraform >/dev/null || { echo "pool: terraform not found"; exit 0; }
  for key in $(jq -r 'keys[]' <<<"$images_json"); do
    tpl="$(pooled_template "$key")"
    [ -n "$tpl" ] || { echo "pool: $key: no pooled template"; continue; }
    have=0
    while IFS= read -r vm; do
      [ -n "$vm" ] && [ "$(template_of "$(vm_json "$vm")")" = "$tpl" ] && have=$((have + 1))
    done < <(pool_vms "$key")
    echo "pool: $key: $have of $POOL_SIZE ready"
    for _ in $(seq "$((POOL_SIZE - have))"); do
      ( work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT
        cp -r "$HERE/terraform/." "$work/"
        id="$(date +%s | tail -c 7)$RANDOM"
        ssh-keygen -q -t ed25519 -N '' -f "$work/key"   # replaced by the run's key at claim
        jq -n --arg run_id "p$id" --arg run_url "${RUN_URL:-pool}" --argjson images "{\"$key\":\"$tpl\"}" \
          --arg public_key "$(cat "$work/key.pub")" \
          '{run_id: $run_id, run_url: $run_url, images: $images, public_key: $public_key, name_prefix: "pool-e2e-"}' > "$work/vars.json"
        terraform -chdir="$work" init -input=false >/dev/null 2>&1 &&
          terraform -chdir="$work" apply -auto-approve -input=false -var-file="$work/vars.json" >/dev/null 2>"$work/err" &&
          echo "pool: $key: created pool-e2e-p$id-$key" || echo "pool: $key: creation failed: $(tail -2 "$work/err" | paste -sd' ' -)"
      ) &
    done
  done
  wait
  ;;
purge)
  for key in $(jq -r 'keys[]' <<<"$images_json"); do
    tpl="$(pooled_template "$key")"
    while IFS= read -r vm; do
      [ -n "$vm" ] || continue
      info="$(vm_json "$vm")"
      if [ -z "$tpl" ] || [ "$(template_of "$info")" != "$tpl" ] || [ "$(jq -r .runtime.powerState <<<"$info")" != poweredOn ]; then
        macs="$("$HERE/dhcp-leases.sh" macs "$vm" | tr '\n' ' ')"
        echo "pool: removing ${vm##*/} (template $(template_of "$info"))"
        govc vm.power -off -force "$vm" >/dev/null 2>&1 || true
        # shellcheck disable=SC2086
        govc vm.destroy "$vm" && "$HERE/dhcp-leases.sh" remove $macs
      fi
    done < <(pool_vms "$key")
  done
  ;;
*) echo "usage: $0 fill|claim KEY NAME PUBKEY HOST|purge" >&2; exit 2 ;;
esac

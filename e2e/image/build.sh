#!/usr/bin/env bash
# Builds the e2e base templates: clones each current golden [latest] image,
# preinstalls what the cases need (prepare.sh), seals the VM (seal.sh) and turns
# it into template e2e-base-<key>-<golden item>-<time> in the e2e folder. run.sh
# with E2E_BASE=1 clones it while it matches the current golden item; the two
# newest per key are kept.
# Environment: as run.sh (TF_VAR_*, optional E2E_IMAGES, RUN_ID, RUN_URL).
# shellcheck disable=SC2154
set -euo pipefail

HERE="$(cd "$(dirname "$0")/.." && pwd)"
# the prefix lets the janitor find what a killed (cancelled) run left behind
WORK="$(mktemp -d -t onigirazu-e2e.XXXXXX)"
TF_DIR="$WORK/terraform"
KEY="$WORK/id_e2e"
TFVARS="$WORK/run.tfvars.json"
RUN_ID="${RUN_ID:-local-$(date -u +%m%d%H%M)}-img"  # the janitor reads the run id first
RUN_URL="${RUN_URL:-local run}"
E2E_IMAGES="${E2E_IMAGES:-u2404=ubuntu-24.04 u2604=ubuntu-26.04}"
KEEP=2

log() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

cleanup() {
  local rc=$?
  # VMs that did not become templates are still in the state: destroy them
  if [ -f "$TF_DIR/terraform.tfstate" ] && [ -n "$(terraform -chdir="$TF_DIR" state list 2>/dev/null)" ]; then
    log "Destroying leftover build VMs"
    terraform -chdir="$TF_DIR" destroy -auto-approve -input=false -var-file="$TFVARS" >/dev/null ||
      echo "destroy failed; the janitor will remove the VMs"
  fi
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT

for v in vsphere_server vsphere_user vsphere_password datacenter cluster host datastore network folder library; do
  n="TF_VAR_$v"
  [ -n "${!n:-}" ] || die "$n is not set"
done
export GOVC_URL="$TF_VAR_vsphere_server" GOVC_USERNAME="$TF_VAR_vsphere_user" GOVC_PASSWORD="$TF_VAR_vsphere_password" GOVC_INSECURE=1
export GOVC_DATACENTER="$TF_VAR_datacenter"
FOLDER="/$TF_VAR_datacenter/vm/$TF_VAR_folder"

# shellcheck source=e2e/images.sh
. "$HERE/images.sh"
E2E_BASE=0 resolve_images || exit 1

log "Creating build VMs (run $RUN_ID)"
cp -R "$HERE/terraform" "$TF_DIR"
rm -rf "$TF_DIR/.terraform" "$TF_DIR"/terraform.tfstate*
ssh-keygen -q -t ed25519 -N '' -C "onigirazu-e2e-$RUN_ID" -f "$KEY"
for try in 1 2 3; do
  terraform -chdir="$TF_DIR" init -input=false >/dev/null && break
  [ "$try" = 3 ] && exit 1
  sleep 15
done
jq -n --arg run_id "$RUN_ID" --arg run_url "$RUN_URL" --argjson images "$images_json" \
  --arg public_key "$(cat "$KEY.pub")" \
  '{run_id: $run_id, run_url: $run_url, images: $images, public_key: $public_key}' > "$TFVARS"
terraform -chdir="$TF_DIR" apply -auto-approve -input=false -var-file="$TFVARS" >/dev/null
hosts_json="$(terraform -chdir="$TF_DIR" output -json hosts)"
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  for ip in $(jq -r '.[]' <<<"$hosts_json"); do echo "::add-mask::$ip"; done
fi

SSH_OPTS=(-i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -o LogLevel=ERROR)
on_host() { local ip; ip="$(jq -r --arg h "$1" '.[$h]' <<<"$hosts_json")"; shift; timeout 1800 ssh "${SSH_OPTS[@]}" "e2e@$ip" "$@"; }

stamp="$(date -u +%Y%m%d-%H%M)"
for key in $(jq -r 'keys[]' <<<"$hosts_json"); do
  golden="$(jq -r --arg k "$key" '.[$k]' <<<"$golden_json")"
  vm="$FOLDER/tmp-e2e-onigirazu-$RUN_ID-$key"
  log "$key: preparing (from $golden)"
  for _ in $(seq 60); do on_host "$key" true 2>/dev/null && break; sleep 5; done
  cat "$HERE/setup-lib.sh" "$HERE/image/prepare.sh" | on_host "$key" 'sudo -n bash -s'
  on_host "$key" 'sudo -n tee /run/e2e-seal.sh >/dev/null' < "$HERE/image/seal.sh"
  on_host "$key" 'sudo -n systemd-run --unit e2e-seal --quiet /bin/bash /run/e2e-seal.sh'

  log "$key: waiting for the sealed VM to power off"
  for _ in $(seq 120); do
    [ "$(govc vm.info -json "$vm" | jq -r '(.virtualMachines // .VirtualMachines)[0].runtime.powerState')" = poweredOff ] && break
    sleep 5
  done
  [ "$(govc vm.info -json "$vm" | jq -r '(.virtualMachines // .VirtualMachines)[0].runtime.powerState')" = poweredOff ] ||
    die "$key: the VM did not power off"
  # shellcheck disable=SC2046
  "$HERE/dhcp-leases.sh" remove $("$HERE/dhcp-leases.sh" macs "$vm")

  name="e2e-base-$key-$golden-$stamp"
  terraform -chdir="$TF_DIR" state rm "vsphere_virtual_machine.vm[\"$key\"]" >/dev/null
  govc vm.change -vm "$vm" -e guestinfo.e2e_authorized_key= \
    -annotation "onigirazu e2e base template: $golden with the packages the e2e cases install. Built by $RUN_URL. Replaced by the next build; e2e/image/build.sh."
  govc object.rename "$vm" "$name"
  # e2e VMs are linked clones of this snapshot: a delta disk, no full copy
  govc snapshot.create -vm "$FOLDER/$name" -m=false -q=false base >/dev/null
  # a powered-off VM clones as well; a template cannot be started by mistake
  if govc vm.markastemplate "$FOLDER/$name"; then
    echo "$key: template $name"
  else
    echo "$key: $name kept as a powered-off VM (VirtualMachine.Provisioning.MarkAsTemplate missing?)"
  fi

  # keep the newest $KEEP of this key: newest by the build stamp at the end
  # of the name, not by the golden image's stamp in the middle (the janitor
  # retries what a running linked clone keeps alive)
  govc find "$FOLDER" -type m -name "e2e-base-$key-*" | awk '{print substr($0, length($0) - 12) " " $0}' | sort | cut -d' ' -f2- | head -n -"$KEEP" | while read -r old; do
    echo "$key: removing $old"
    old_macs="$("$HERE/dhcp-leases.sh" macs "$old" | tr '\n' ' ')"
    # a template object is turned back into a VM first (vm.destroy refuses
    # templates); a running linked clone keeps it, the janitor retries
    govc vm.markasvm -pool "/$TF_VAR_datacenter/host/$TF_VAR_cluster/Resources" "$old" >/dev/null 2>&1 || true
    # shellcheck disable=SC2086
    govc vm.destroy "$old" && "$HERE/dhcp-leases.sh" remove $old_macs || echo "$key: $old not removed (linked clones still running?)"
  done
done
log "Done"

#!/usr/bin/env bash
# End-to-end tests of the Windows modules on a disposable Windows VM.
#
# Clones the [latest] Windows golden image (its first boot creates the local
# administrator e2e with a password generated here and passed in guestinfo),
# runs every case in e2e/cases-windows over WinRM and destroys the VM on exit.
#
# Required environment: as e2e/run.sh (TF_VAR_vsphere_* and the estate names).
# Optional: E2E_WINDOWS_IMAGES (default "win2025=windows-2025"), E2E_CASES,
#   RUN_ID, RUN_URL, KEEP_VMS=1
# A case: playbook.yml (hosts: windows, or https for the https listener),
# optional verify.yml (must pass after the apply), NOT_IDEMPOTENT marker.
# Steps: apply, verify, apply again (nothing changed), verify again.
# shellcheck disable=SC2154 # TF_VAR_* come from the caller
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
TF_DIR="$HERE/terraform-windows"
WORK="$(mktemp -d -t onigirazu-e2e.XXXXXX)"
BIN="$WORK/onigirazu"
INVENTORY="$WORK/inventory.yml"
KEY="$WORK/id_e2e"
RESULTS="$WORK/results.tsv"
TFVARS="$WORK/run.tfvars.json"
touch "$RESULTS"

RUN_ID="${RUN_ID:-local-$(date -u +%m%d%H%M)}-w"
RUN_URL="${RUN_URL:-local run}"
# shellcheck disable=SC2034 # read by images.sh
E2E_IMAGES="${E2E_WINDOWS_IMAGES:-win2025=windows-2025}"
cases="${E2E_CASES:-$(cd "$HERE/cases-windows" && ls)}"

log() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

cleanup() {
  local rc=$?
  if [ -z "${KEEP_VMS:-}" ] && [ -f "$TF_DIR/terraform.tfstate" ]; then
    log "Destroying VMs"
    terraform -chdir="$TF_DIR" destroy -auto-approve -input=false -var-file="$TFVARS" >/dev/null ||
      echo "destroy failed; the janitor will remove the VMs"
  fi
  if [ -n "${KEEP_VMS:-}" ] && [ -f "$INVENTORY" ]; then
    # kept VMs are reachable only with this run's password; it stays on the runner
    local keep="$HOME/.cache/onigirazu-e2e/$RUN_ID"
    mkdir -p "$keep" && chmod 700 "$keep" && cp "$INVENTORY" "$keep/"
    echo "kept VMs: inventory in $keep on the runner"
  fi
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT

for v in vsphere_server vsphere_user vsphere_password datacenter cluster host datastore network folder library; do
  n="TF_VAR_$v"
  [ -n "${!n:-}" ] || die "$n is not set"
done
command -v govc >/dev/null || die "govc not found"
command -v terraform >/dev/null || die "terraform not found"

export GOVC_URL="$TF_VAR_vsphere_server" GOVC_USERNAME="$TF_VAR_vsphere_user" GOVC_PASSWORD="$TF_VAR_vsphere_password" GOVC_INSECURE=1
export GOVC_DATACENTER="$TF_VAR_datacenter"
# shellcheck source=e2e/images.sh
. "$HERE/images.sh"
resolve_images || exit 1

log "Building onigirazu"
(cd "$ROOT" && go build -o "$BIN" ./cmd/onigirazu)

# One-time password: random, with the four character classes Windows wants;
# it lives only in this run's 0700 work directory
umask 077
PASSWORD="$(head -c 48 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24)Aa1-"
[ "${#PASSWORD}" -ge 20 ] || die "could not generate a password"
ssh-keygen -q -t ed25519 -N '' -C "onigirazu-e2e-$RUN_ID" -f "$KEY"

log "Creating the Windows VM (run $RUN_ID)"
for try in 1 2 3; do
  terraform -chdir="$TF_DIR" init -input=false >/dev/null && break
  [ "$try" = 3 ] && exit 1
  sleep 15
done
jq -n --arg run_id "$RUN_ID" --arg run_url "$RUN_URL" --argjson images "$images_json" --arg pw "$PASSWORD" \
  --arg key "$(cat "$KEY.pub")" \
  '{run_id: $run_id, run_url: $run_url, images: $images, e2e_password: $pw, public_key: $key}' > "$TFVARS"
terraform -chdir="$TF_DIR" apply -auto-approve -input=false -var-file="$TFVARS" >/dev/null || die "terraform apply failed"
hosts_json="$(terraform -chdir="$TF_DIR" output -json hosts)"
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  for ip in $(jq -r '.[]' <<<"$hosts_json"); do echo "::add-mask::$ip"; done
fi

# windows: NTLM with message encryption over http, as ClanRed connects;
# https: the same host over the https listener; ssh: over OpenSSH
export ONIGIRAZU_SSH_KNOWN_HOSTS_FILE="$WORK/known_hosts"
{
  echo "all:"
  echo "  vars:"
  echo "    ansible_connection: winrm"
  echo "    ansible_user: e2e"
  echo "    ansible_password: \"$PASSWORD\""
  echo "    ansible_winrm_transport: ntlm"
  echo "  children:"
  echo "    windows:"
  echo "      hosts:"
  jq -r 'to_entries[] | "        \(.key):\n          ansible_host: \(.value)\n          ansible_port: 5985\n          ansible_winrm_scheme: http\n          ansible_winrm_message_encryption: always"' <<<"$hosts_json"
  echo "    https:"
  echo "      hosts:"
  jq -r 'to_entries[] | "        \(.key)-https:\n          ansible_host: \(.value)\n          ansible_port: 5986\n          ansible_winrm_scheme: https\n          ansible_winrm_server_cert_validation: ignore"' <<<"$hosts_json"
  echo "    ssh:"
  echo "      hosts:"
  jq -r --arg key "$KEY" 'to_entries[] | "        \(.key)-ssh:\n          ansible_host: \(.value)\n          ansible_connection: ssh\n          ansible_shell_type: powershell\n          ansible_port: 22\n          ansible_ssh_private_key_file: \($key)"' <<<"$hosts_json"
} > "$INVENTORY"

log "Waiting for WinRM (first boot creates the e2e account)"
cat > "$WORK/ping.yml" <<'EOF'
- hosts: windows:https
  gather_facts: false
  tasks:
    - win_ping:
EOF
for _ in $(seq 90); do
  "$BIN" apply "$WORK/ping.yml" -i "$INVENTORY" --state "$WORK/ping-state" --no-color >"$WORK/ping.log" 2>&1 && break
  sleep 10
done
"$BIN" apply "$WORK/ping.yml" -i "$INVENTORY" --state "$WORK/ping-state" --no-color --log-format json >"$WORK/ping.log" 2>&1 || {
  # the task errors say why (refused, 401, encryption)
  jq -R -r 'split("{\"timestamp\"")[1:][] | ("{\"timestamp\"" + .) | sub("}[^}]*$"; "}") | fromjson? |
    select(.fields.type == "task_end" and .fields.success != true) | "\(.fields.host): \(.fields.msg // .message)"' "$WORK/ping.log" |
    sed -E 's/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/<ip>/g' | cut -c1-1500 | sort -u
  die "no WinRM access as e2e"
}
echo "WinRM ready (http with NTLM encryption, https, ssh)"

records() {
  jq -c -R 'split("{\"timestamp\"")[1:][] | ("{\"timestamp\"" + .) | sub("}[^}]*$"; "}") | fromjson?' "$WORK/apply.log"
}
apply() {  # dir playbook -> task_end events in $WORK/events.jsonl
  APPLY_RC=0
  (cd "$1" && timeout 1800 "$BIN" apply "$2" -i "$INVENTORY" --state "$WORK/state-$(basename "$1")" \
    --log-format json --no-color >"$WORK/apply.log" 2>&1) || APPLY_RC=$?
  records | jq -c 'select(.fields.type == "task_end") | .fields' > "$WORK/events.jsonl" || true
}
apply_errors() {
  records | jq -r 'select(.level == "ERROR" or .level == "WARN") | "      \(.level): \(.message)"' | cut -c1-400 | head -6
}
record() { printf '%s\t%s\t%s\n' "$1" "$2" "$3" >> "$RESULTS"; echo "  [$2] $1 ${3:+- $3}"; }
failed_tasks() {
  local f
  f="$(jq -r 'select(.success != true and .ignored != true) | "\(.host): \(.task)"' "$WORK/events.jsonl")"
  [ "${APPLY_RC:-0}" = 0 ] || f="$f${f:+, }exit code $APPLY_RC"
  echo "$f"
}

for c in $cases; do
  src="$HERE/cases-windows/$c"
  [ -f "$src/playbook.yml" ] || continue
  dir="$WORK/cases/$c"
  mkdir -p "$WORK/cases" && cp -R "$src" "$dir"
  log "Case $c"

  apply "$dir" playbook.yml
  if [ -n "$(failed_tasks)" ]; then record "$c" FAIL "apply: $(failed_tasks | paste -sd, -)"; apply_errors; continue; fi
  if [ -f "$dir/verify.yml" ]; then
    apply "$dir" verify.yml
    if [ -n "$(failed_tasks)" ]; then record "$c" FAIL "verify: $(failed_tasks | paste -sd, -)"; apply_errors; continue; fi
  fi
  record "$c" PASS "apply+verify"
  [ -f "$dir/NOT_IDEMPOTENT" ] && continue

  apply "$dir" playbook.yml
  changed="$(jq -r 'select(.changed == true) | "\(.host): \(.task)\(if (.msg // "") != "" then " (\(.msg))" else "" end)"' "$WORK/events.jsonl")"
  if [ -n "$(failed_tasks)" ]; then record "$c" FAIL "second apply: $(failed_tasks | paste -sd, -)"; apply_errors
  elif [ -n "$changed" ]; then record "$c" FAIL "not idempotent: $(echo "$changed" | paste -sd, -)"
  else record "$c" PASS "idempotent"; fi
done

log "Summary"
total="$(wc -l < "$RESULTS" | tr -d ' ')"
fails="$(grep -c $'\tFAIL\t' "$RESULTS" || true)"
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  { echo "| case | result | detail |"; echo "|---|---|---|"
    awk -F'\t' '{printf "| %s | %s | %s |\n", $1, $2, $3}' "$RESULTS"; } >> "$GITHUB_STEP_SUMMARY"
fi
echo "$((total - fails))/$total checks passed"
[ "$total" -gt 0 ] && [ "$fails" = 0 ]

#!/usr/bin/env bash
# End-to-end tests on disposable vSphere VMs.
#
# Creates one VM per image, runs every case in e2e/cases against all of them,
# and destroys the VMs on exit, whatever happens.
#
# Required environment:
#   TF_VAR_vsphere_server TF_VAR_vsphere_user TF_VAR_vsphere_password
#   TF_VAR_datacenter TF_VAR_cluster TF_VAR_host TF_VAR_datastore
#   TF_VAR_network TF_VAR_folder TF_VAR_library
# Optional: E2E_IMAGES (default "u2404=ubuntu-24.04 u2604=ubuntu-26.04"),
#   E2E_CASES (case directory names, default all), RUN_ID, RUN_URL, KEEP_VMS=1
# TF_VAR_* come from the environment and are checked below
# shellcheck disable=SC2154
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
TF_DIR="$HERE/terraform"
WORK="$(mktemp -d)"
BIN="$WORK/onigirazu"
KEY="$WORK/id_e2e"
RESULTS="$WORK/results.tsv"
TFVARS="$WORK/run.tfvars.json"

RUN_ID="${RUN_ID:-local-$(date -u +%m%d%H%M)}"
RUN_URL="${RUN_URL:-local run}"
E2E_IMAGES="${E2E_IMAGES:-u2404=ubuntu-24.04 u2604=ubuntu-26.04}"

log() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

cleanup() {
  local rc=$?
  if [ -z "${KEEP_VMS:-}" ] && [ -f "$TF_DIR/terraform.tfstate" ]; then
    log "Destroying VMs"
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
command -v govc >/dev/null || die "govc not found"
command -v terraform >/dev/null || die "terraform not found"

# --- resolve the current [latest] item of each image family ------------------
export GOVC_URL="$TF_VAR_vsphere_server" GOVC_USERNAME="$TF_VAR_vsphere_user" GOVC_PASSWORD="$TF_VAR_vsphere_password" GOVC_INSECURE=1
items="$(govc library.info -json "/$TF_VAR_library/*")"
images_json="{"
for pair in $E2E_IMAGES; do
  key="${pair%%=*}" family="${pair#*=}"
  name="$(jq -r --arg f "$family-" '
    [ (if type == "array" then . else [.] end)[]
      | select(.name | startswith($f)) | select((.description // "") | contains("[latest]")) | .name ] | first // empty' <<<"$items")"
  [ -n "$name" ] || die "no [latest] item for $family in $TF_VAR_library"
  echo "$key: $name"
  images_json+="\"$key\":\"$name\","
done
images_json="${images_json%,}}"

# --- build onigirazu and a one-time key ---------------------------------------
log "Building onigirazu"
(cd "$ROOT" && go build -o "$BIN" ./cmd/onigirazu)
ssh-keygen -q -t ed25519 -N '' -C "onigirazu-e2e-$RUN_ID" -f "$KEY"

# --- create the VMs ------------------------------------------------------------
log "Creating VMs (run $RUN_ID)"
terraform -chdir="$TF_DIR" init -input=false >/dev/null
# One vars file for apply and destroy
jq -n --arg run_id "$RUN_ID" --arg run_url "$RUN_URL" --argjson images "$images_json" \
  --arg public_key "$(cat "$KEY.pub")" \
  '{run_id: $run_id, run_url: $run_url, images: $images, public_key: $public_key}' > "$TFVARS"
terraform -chdir="$TF_DIR" apply -auto-approve -input=false -var-file="$TFVARS" >/dev/null
hosts_json="$(terraform -chdir="$TF_DIR" output -json hosts)"
# Actions logs of a public repository are public: keep internal addresses out
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  for ip in $(jq -r '.[]' <<<"$hosts_json"); do echo "::add-mask::$ip"; done
fi
echo "$hosts_json" | jq -r 'to_entries[] | "\(.key)\t\(.value)"'

INVENTORY="$WORK/inventory.yml"
{
  echo "groups:"
  echo "  e2e:"
  echo "    hosts:"
  jq -r 'to_entries[] | "      \(.key):\n        onigirazu_host: \(.value)\n        onigirazu_user: e2e\n        onigirazu_port: 22\n        onigirazu_ssh_private_key_file: KEYFILE\n        onigirazu_connection: ssh"' <<<"$hosts_json" \
    | sed "s#KEYFILE#$KEY#"
} > "$INVENTORY"

SSH_OPTS=(-i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -o LogLevel=ERROR)
host_ip() { jq -r --arg h "$1" '.[$h]' <<<"$hosts_json"; }
on_host() { local ip; ip="$(host_ip "$1")"; shift; ssh "${SSH_OPTS[@]}" "e2e@$ip" "$@"; }

log "Waiting for SSH"
for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
  for _ in $(seq 60); do on_host "$h" true 2>/dev/null && break; sleep 5; done
  on_host "$h" 'sudo -n true' || die "$h: no ssh/sudo access as e2e"
  echo "$h ready: $(on_host "$h" '. /etc/os-release; echo $PRETTY_NAME')"
done

# --- run the cases -------------------------------------------------------------
# Each case: playbook.yml, verify.sh (run on the host with sudo), optional
# NOT_IDEMPOTENT marker. Steps: apply, verify, apply again (nothing changed),
# verify again.
apply() {  # case_dir label -> writes task_end events to $WORK/events.jsonl
  local dir="$1" state
  state="$WORK/state-$(basename "$1")"
  (cd "$WORK" && "$BIN" apply "$dir/playbook.yml" -i "$INVENTORY" --state "$state" \
    --log-format json --no-color >"$WORK/apply.log" 2>&1) || true
  grep '"type":"task_end"' "$WORK/apply.log" | jq -c '.fields' > "$WORK/events.jsonl" || true
}

record() { printf '%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" >> "$RESULTS"; echo "  [$3] $1 / $2 ${4:+- $4}"; }

cases="${E2E_CASES:-$(cd "$HERE/cases" && ls)}"
for c in $cases; do
  dir="$HERE/cases/$c"
  [ -f "$dir/playbook.yml" ] || continue
  log "Case $c"

  apply "$dir"
  for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
    failed="$(jq -r --arg h "$h" 'select(.host == $h and .success != true) | .task' "$WORK/events.jsonl")"
    ran="$(jq -r --arg h "$h" 'select(.host == $h) | .task' "$WORK/events.jsonl" | wc -l | tr -d ' ')"
    if [ "$ran" = 0 ]; then
      record "$c" "$h" FAIL "no task ran: $(grep -m1 -iE 'error|failed' "$WORK/apply.log" | jq -r '.message // .' 2>/dev/null | cut -c1-200)"
      continue
    fi
    [ -z "$failed" ] || { record "$c" "$h" FAIL "apply failed: $(echo "$failed" | paste -sd, -)"; continue; }
    if [ -f "$dir/verify.sh" ] && ! out="$(on_host "$h" 'sudo -n bash -s' < "$dir/verify.sh" 2>&1)"; then
      record "$c" "$h" FAIL "verify: $(echo "$out" | tail -1)"; continue
    fi
    record "$c" "$h" PASS "apply+verify"
  done

  [ -f "$dir/NOT_IDEMPOTENT" ] && continue
  apply "$dir"
  for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
    changed="$(jq -r --arg h "$h" 'select(.host == $h and .changed == true) | .task' "$WORK/events.jsonl")"
    failed="$(jq -r --arg h "$h" 'select(.host == $h and .success != true) | .task' "$WORK/events.jsonl")"
    if [ -n "$failed" ]; then record "$c" "$h" FAIL "second apply failed: $(echo "$failed" | paste -sd, -)"
    elif [ -n "$changed" ]; then record "$c" "$h" FAIL "not idempotent: $(echo "$changed" | paste -sd, -)"
    else record "$c" "$h" PASS "idempotent"; fi
  done
done

# --- summary -------------------------------------------------------------------
log "Summary"
total="$(wc -l < "$RESULTS" | tr -d ' ')"
fails="$(grep -c $'\tFAIL\t' "$RESULTS" || true)"
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  { echo "| case | host | result | detail |"; echo "|---|---|---|---|"
    awk -F'\t' '{printf "| %s | %s | %s | %s |\n", $1, $2, $3, $4}' "$RESULTS"; } >> "$GITHUB_STEP_SUMMARY"
fi
echo "$((total - fails))/$total checks passed"
[ "$fails" = 0 ]

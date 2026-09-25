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
#   E2E_CASES (case directory names, default all), RUN_ID, RUN_URL, KEEP_VMS=1,
#   E2E_SHARD=i/n (run every n-th case starting with the i-th, on own VMs)
# TF_VAR_* come from the environment and are checked below
# shellcheck disable=SC2154
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
TF_DIR="$HERE/terraform"
WORK="$(mktemp -d)"
BIN="$WORK/onigirazu"
KEY="$WORK/id_e2e"
INVENTORY="$WORK/inventory.yml"
RESULTS="$WORK/results.tsv"
TFVARS="$WORK/run.tfvars.json"

RUN_ID="${RUN_ID:-local-$(date -u +%m%d%H%M)}"

# Cases of this run; a shard takes every n-th of them on its own VMs
cases="${E2E_CASES:-$(cd "$HERE/cases" && ls)}"
if [ -n "${E2E_SHARD:-}" ]; then
  shard="${E2E_SHARD%/*}" shards="${E2E_SHARD#*/}"
  # shellcheck disable=SC2086 # one case per word
  cases="$(printf '%s\n' $cases | sort | awk -v i="$shard" -v n="$shards" '(NR - 1) % n == i - 1')"
  RUN_ID="$RUN_ID-s$shard"
fi
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
  if [ -n "${KEEP_VMS:-}" ] && [ -f "$KEY" ]; then
    # Kept VMs are only reachable with this run's key; it stays on the runner
    local keep="$HOME/.cache/onigirazu-e2e/$RUN_ID"
    mkdir -p "$keep" && cp "$KEY" "$INVENTORY" "$keep/" 2>/dev/null && chmod 700 "$keep"
    echo "kept VMs: key and inventory in $keep on the runner"
  fi
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT

[ -n "${cases//[[:space:]]/}" ] || { echo "no cases for shard ${E2E_SHARD:-}"; exit 0; }
echo "cases: ${cases//$'\n'/ }"

for v in vsphere_server vsphere_user vsphere_password datacenter cluster host datastore network folder library; do
  n="TF_VAR_$v"
  [ -n "${!n:-}" ] || die "$n is not set"
done
command -v govc >/dev/null || die "govc not found"
command -v terraform >/dev/null || die "terraform not found"

# --- resolve the current [latest] item of each image family ------------------
export GOVC_URL="$TF_VAR_vsphere_server" GOVC_USERNAME="$TF_VAR_vsphere_user" GOVC_PASSWORD="$TF_VAR_vsphere_password" GOVC_INSECURE=1
export GOVC_DATACENTER="$TF_VAR_datacenter"
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

# The REST deploy call answers a bare 403; the SOAP clone of the same template
# names the missing privilege and the object it was checked on.
diagnose_permissions() {
  log "Permission diagnosis (SOAP clone of the template)"
  local first tpl name
  first="$(jq -r 'to_entries[0].value' <<<"$images_json")"
  tpl="$(govc find "/$TF_VAR_datacenter/vm" -type m -name "$first" | head -1)"
  [ -n "$tpl" ] || { echo "template VM $first not found"; return; }
  echo "template: $tpl"
  local ref
  for ref in $(govc vm.info -json "$tpl" | jq -r '(.virtualMachines // .VirtualMachines)[0] | (.network[]?, .datastore[]?) | "\(.type):\(.value)"'); do
    echo "template uses $ref: $(govc ls -L "$ref" 2>/dev/null)"
  done
  name="tmp-e2e-onigirazu-$RUN_ID-diag"
  govc vm.clone -vm "$tpl" -on=false -folder "/$TF_VAR_datacenter/vm/$TF_VAR_folder" \
    -pool "/$TF_VAR_datacenter/host/$TF_VAR_cluster/Resources" -host "$TF_VAR_host" \
    -ds "$TF_VAR_datastore" "$name" 2>&1 | tail -5 || true
  govc vm.destroy "/$TF_VAR_datacenter/vm/$TF_VAR_folder/$name" >/dev/null 2>&1 || true
}

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
if ! terraform -chdir="$TF_DIR" apply -auto-approve -input=false -var-file="$TFVARS" >/dev/null; then
  diagnose_permissions
  die "terraform apply failed"
fi
hosts_json="$(terraform -chdir="$TF_DIR" output -json hosts)"
# Actions logs of a public repository are public: keep internal addresses out
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  for ip in $(jq -r '.[]' <<<"$hosts_json"); do echo "::add-mask::$ip"; done
fi
echo "$hosts_json" | jq -r 'to_entries[] | "\(.key)\t\(.value)"'

{
  echo "groups:"
  echo "  e2e:"
  echo "    hosts:"
  jq -r 'to_entries[] | "      \(.key):\n        onigirazu_host: \(.value)\n        onigirazu_user: e2e\n        onigirazu_port: 22\n        onigirazu_ssh_private_key_file: KEYFILE\n        onigirazu_connection: ssh"' <<<"$hosts_json" \
    | sed "s#KEYFILE#$KEY#"
} > "$INVENTORY"

SSH_OPTS=(-i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -o LogLevel=ERROR)
host_ip() { jq -r --arg h "$1" '.[$h]' <<<"$hosts_json"; }
# Every remote call is bounded: one stuck case must not hold the whole run
on_host() { local ip; ip="$(host_ip "$1")"; shift; timeout 600 ssh "${SSH_OPTS[@]}" "e2e@$ip" "$@"; }

log "Waiting for SSH"
for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
  for _ in $(seq 60); do on_host "$h" true 2>/dev/null && break; sleep 5; done
  on_host "$h" 'sudo -n true' || die "$h: no ssh/sudo access as e2e"
  echo "$h ready: $(on_host "$h" '. /etc/os-release; echo $PRETTY_NAME')"
done

# --- run the cases -------------------------------------------------------------
# Each case: playbook.yml, verify.sh (run on the host with sudo), optional
# setup.sh (run on the host with sudo before the first apply) and
# NOT_IDEMPOTENT marker. Steps: apply, verify, apply again (nothing changed),
# verify again.
# JSON log records of the last apply. The progress bar shares stdout and can
# sit between two records on one line, so split at every record start and cut
# what follows the closing brace.
records() {
  jq -c -R 'split("{\"timestamp\"")[1:][] | ("{\"timestamp\"" + .) | sub("}[^}]*$"; "}") | fromjson?' "$WORK/apply.log"
}

apply() {  # case_dir -> writes task_end events to $WORK/events.jsonl
  local dir="$1" state
  state="$WORK/state-$(basename "$1")"
  # Run from the case's own directory so relative paths (src, script) work
  (cd "$dir" && timeout 1200 "$BIN" apply playbook.yml -i "$INVENTORY" --state "$state" \
    --log-format json --no-color >"$WORK/apply.log" 2>&1) || true
  records | jq -c 'select(.fields.type == "task_end") | .fields' > "$WORK/events.jsonl" || true
}

# First error lines of the last apply, for the job log
apply_errors() {
  records | jq -r 'select(.level == "ERROR" or .level == "WARN") | "      \(.level): \(.message)"' |
    cut -c1-400 | head -"${1:-4}"
}

record() { printf '%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" >> "$RESULTS"; echo "  [$3] $1 / $2 ${4:+- $4}"; }

for c in $cases; do
  [ -f "$HERE/cases/$c/playbook.yml" ] || continue
  # Work on a copy: cases may write next to their playbook (fetch)
  dir="$WORK/cases/$c"
  mkdir -p "$WORK/cases" && cp -R "$HERE/cases/$c" "$dir"
  log "Case $c"

  if [ -f "$dir/setup.sh" ]; then
    for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
      on_host "$h" 'sudo -n bash -s' < "$dir/setup.sh" >/dev/null 2>&1 || echo "  setup failed on $h"
    done
  fi

  apply "$dir"

  if [ -f "$dir/EXPECT_FAIL" ]; then
    for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
      if jq -e --arg h "$h" 'select(.host == $h and .success != true)' "$WORK/events.jsonl" >/dev/null; then
        record "$c" "$h" PASS "failed as expected"
      else
        record "$c" "$h" FAIL "expected the apply to fail"
      fi
    done
    continue
  fi

  passed=""
  for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
    failed="$(jq -r --arg h "$h" 'select(.host == $h and .success != true) | .task' "$WORK/events.jsonl")"
    ran="$(jq -r --arg h "$h" 'select(.host == $h) | .task' "$WORK/events.jsonl" | wc -l | tr -d ' ')"
    if [ "$ran" = 0 ]; then
      record "$c" "$h" FAIL "no task ran: $(records | jq -r 'select(.level == "ERROR") | .message' | head -1 | cut -c1-200)"
      continue
    fi
    if [ -n "$failed" ]; then
      record "$c" "$h" FAIL "apply failed: $(echo "$failed" | paste -sd, -)"
      apply_errors
      continue
    fi
    if [ -f "$dir/verify.sh" ] && ! out="$(on_host "$h" 'sudo -n bash -s' < "$dir/verify.sh" 2>&1)"; then
      record "$c" "$h" FAIL "verify: $(echo "$out" | tail -1)"; continue
    fi
    if [ -f "$dir/verify-local.sh" ] && ! out="$(cd "$dir" && HOST="$h" bash verify-local.sh 2>&1)"; then
      record "$c" "$h" FAIL "verify-local: $(echo "$out" | tail -1)"; continue
    fi
    record "$c" "$h" PASS "apply+verify"
    passed="$passed $h"
  done

  [ -f "$dir/NOT_IDEMPOTENT" ] && continue
  [ -n "$passed" ] || continue
  apply "$dir"
  for h in $passed; do
    ran="$(jq -r --arg h "$h" 'select(.host == $h) | .task' "$WORK/events.jsonl" | wc -l | tr -d ' ')"
    [ "$ran" != 0 ] || { record "$c" "$h" FAIL "second apply: no task ran"; continue; }
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

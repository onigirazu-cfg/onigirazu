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
#   E2E_BASE=1 (clone the e2e base templates of image/build.sh when current),
#   E2E_CASES (case directory names, default all), RUN_ID, RUN_URL, KEEP_VMS=1,
#   E2E_SHARD=i/n (the i-th of n shards balanced by case-seconds.tsv, on own VMs)
# TF_VAR_* come from the environment and are checked below
# shellcheck disable=SC2154
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
TF_DIR="$HERE/terraform"
# the prefix lets the janitor find what a killed (cancelled) run left behind
WORK="$(mktemp -d -t onigirazu-e2e.XXXXXX)"
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
  # Longest cases first, each onto the shard with the least time so far
  # (case-seconds.tsv; an unknown case counts 10 s)
  # shellcheck disable=SC2086 # one case per word
  cases="$(printf '%s\n' $cases | sort -u | awk -v i="$shard" -v n="$shards" -F'\t' '
    NR == FNR { if ($0 !~ /^#/) secs[$1] = $2; next }
    { c[++m] = $1; w[m] = ($1 in secs) ? secs[$1] : 10 }
    END {
      for (a = 1; a <= m; a++) for (b = a + 1; b <= m; b++)
        if (w[b] > w[a] || (w[b] == w[a] && c[b] < c[a])) { t = w[a]; w[a] = w[b]; w[b] = t; t = c[a]; c[a] = c[b]; c[b] = t }
      for (a = 1; a <= m; a++) {
        best = 1; for (k = 2; k <= n; k++) if (load[k] < load[best]) best = k
        load[best] += w[a]; if (best == i) print c[a]
      }
    }' "$HERE/case-seconds.tsv" - | sort)"
  RUN_ID="$RUN_ID-s$shard"
fi
RUN_URL="${RUN_URL:-local run}"
E2E_IMAGES="${E2E_IMAGES:-u2404=ubuntu-24.04 u2604=ubuntu-26.04}"

log() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

# a failed run: what vCenter knows about each VM (power, guest IP, tools,
# last events) and a console screenshot into $E2E_DIAG (uploaded by CI)
diagnose() {
  local vm dir="${E2E_DIAG:-$WORK/diag}"
  mkdir -p "$dir"
  log "VM state"
  while IFS= read -r vm; do
    [ -n "$vm" ] || continue
    govc vm.info -json "$vm" 2>/dev/null | jq -r '(.virtualMachines // .VirtualMachines)[0] |
      "\(.name): power=\(.runtime.powerState) guest=\(.guest.guestState) tools=\(.guest.toolsRunningStatus) ip=\(.guest.ipAddress // "-") boot=\(.runtime.bootTime // "-")"'
    govc events -n 8 "$vm" 2>/dev/null | sed 's/^/    /'
    govc vm.console -capture "$dir/${vm##*/}.png" "$vm" >/dev/null 2>&1 && echo "    console: ${vm##*/}.png"
  done < <(govc find "/$TF_VAR_datacenter/vm/$TF_VAR_folder" -type m -name "tmp-e2e-onigirazu-$RUN_ID-*" 2>/dev/null)
}

cleanup() {
  local rc=$?
  if [ "$rc" != 0 ] && { [ -f "$TF_DIR/terraform.tfstate" ] || [ -f "$WORK/claimed" ]; }; then diagnose || true; fi
  if [ -z "${KEEP_VMS:-}" ] && { [ -f "$TF_DIR/terraform.tfstate" ] || [ -f "$WORK/claimed" ]; }; then
    log "Destroying VMs"
    # throwaway VMs: power them off hard first; terraform would wait for a
    # clean guest shutdown (over a minute with the databases running)
    local vm n=0 t0=$SECONDS
    # shellcheck disable=SC2046
    [ -n "${hosts_json:-}" ] && "$HERE/dhcp-release.sh" "$KEY" $(jq -r '.[]' <<<"$hosts_json")
    # the folder path may have spaces: one VM per line
    local macs=""
    while IFS= read -r vm; do
      [ -n "$vm" ] || continue
      macs="$macs $("$HERE/dhcp-leases.sh" macs "$vm" | tr '\n' ' ')"
      govc vm.power -off -force "$vm" >/dev/null 2>&1 &
      n=$((n + 1))
    done < <(govc find "/$TF_VAR_datacenter/vm/$TF_VAR_folder" -type m -name "tmp-e2e-onigirazu-$RUN_ID-*" 2>/dev/null)
    wait
    echo "powered off $n VM(s) in $((SECONDS - t0)) s"
    t0=$SECONDS
    if [ -f "$TF_DIR/terraform.tfstate" ]; then
      terraform -chdir="$TF_DIR" destroy -auto-approve -input=false -var-file="$TFVARS" >/dev/null ||
        echo "destroy failed; the janitor will remove the VMs"
    fi
    # pre-warmed VMs this run claimed are not in the terraform state
    while IFS= read -r key; do
      govc vm.destroy "/$TF_VAR_datacenter/vm/$TF_VAR_folder/tmp-e2e-onigirazu-$RUN_ID-$key" >/dev/null 2>&1 ||
        echo "claimed VM $key not destroyed; the janitor will"
    done < <(cat "$WORK/claimed" 2>/dev/null)
    echo "destroyed in $((SECONDS - t0)) s"
    # the VMs' DHCP leases go with them
    # shellcheck disable=SC2086
    "$HERE/dhcp-leases.sh" remove $macs
  fi
  if [ -n "${KEEP_VMS:-}" ] && [ -f "$KEY" ]; then
    # Kept VMs are only reachable with this run's key; it stays on the runner
    local keep="$HOME/.cache/onigirazu-e2e/$RUN_ID"
    mkdir -p "$keep" && cp "$KEY" "$INVENTORY" "$keep/" 2>/dev/null && chmod 700 "$keep"
    # the janitor removes VMs of finished runs at once, kept ones after its TTL
    while IFS= read -r vm; do
      [ -n "$vm" ] && govc vm.change -vm "$vm" -annotation "e2e keep_vms: $RUN_URL" >/dev/null 2>&1
    done < <(govc find "/$TF_VAR_datacenter/vm/$TF_VAR_folder" -type m -name "tmp-e2e-onigirazu-$RUN_ID-*" 2>/dev/null)
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
# shellcheck source=e2e/images.sh
. "$HERE/images.sh"
resolve_images || exit 1

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
# with the agents built in, as a release has them (remote_server auto)
(cd "$ROOT" && go generate ./internal/agentbin && go build -o "$BIN" ./cmd/onigirazu)
ssh-keygen -q -t ed25519 -N '' -C "onigirazu-e2e-$RUN_ID" -f "$KEY"

# --- pre-warmed VMs, then the rest --------------------------------------------
# a pool VM (e2e/pool.sh) is claimed by renaming it to this run's name and
# given this run's key; what the pool lacks is created with terraform
claimed_json="{}"
if [ "${E2E_POOL:-1}" = 1 ]; then
  log "Claiming pre-warmed VMs"
  for key in $(jq -r 'keys[]' <<<"$images_json"); do
    ip="$("$HERE/pool.sh" claim "$key" "tmp-e2e-onigirazu-$RUN_ID-$key" "$(cat "$KEY.pub")" "e2e-$key" || true)"
    [ -n "$ip" ] || { echo "$key: none in the pool"; continue; }
    [[ "$ip" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "$key: the pool claim answered with something that is not an address"
    echo "$key: pre-warmed"
    claimed_json="$(jq -c --arg k "$key" --arg ip "$ip" '. + {($k): $ip}' <<<"$claimed_json")"
    echo "$key" >> "$WORK/claimed"
  done
fi
to_create="$(jq -c --argjson c "$claimed_json" 'with_entries(select(.key as $k | $c[$k] == null))' <<<"$images_json")"
hosts_json="$claimed_json"
if [ "$to_create" != "{}" ]; then
  log "Creating VMs (run $RUN_ID)"
  # the provider registry drops connections now and then
  for try in 1 2 3; do
    terraform -chdir="$TF_DIR" init -input=false >/dev/null && break
    [ "$try" = 3 ] && exit 1
    sleep 15
  done
  # One vars file for apply and destroy
  jq -n --arg run_id "$RUN_ID" --arg run_url "$RUN_URL" --argjson images "$to_create" \
    --arg public_key "$(cat "$KEY.pub")" \
    '{run_id: $run_id, run_url: $run_url, images: $images, public_key: $public_key}' > "$TFVARS"
  if ! terraform -chdir="$TF_DIR" apply -auto-approve -input=false -var-file="$TFVARS" >/dev/null; then
    diagnose_permissions
    die "terraform apply failed"
  fi
  hosts_json="$(jq -s '.[0] + .[1]' <<<"$claimed_json
$(terraform -chdir="$TF_DIR" output -json hosts)")"
fi
# Actions logs of a public repository are public: keep internal addresses out
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  for ip in $(jq -r '.[]' <<<"$hosts_json"); do echo "::add-mask::$ip"; done
fi
echo "$hosts_json" | jq -r 'to_entries[] | "\(.key)\t\(.value)"'

# The VMs are new, their addresses are not: a key a previous run recorded for
# the same address would be a changed key. Each run checks host keys against
# its own file, and stale entries of these addresses leave the runner's file
# (older binaries read ~/.ssh/known_hosts whatever the configuration says).
export ONIGIRAZU_SSH_KNOWN_HOSTS_FILE="$WORK/known_hosts"
for ip in $(jq -r '.[]' <<<"$hosts_json"); do
  ssh-keygen -R "$ip" >/dev/null 2>&1 || true
  ssh-keygen -R "[$ip]:22" >/dev/null 2>&1 || true
done

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

# where the creation time went: vCenter events of each VM (clone,
# customization, power on), oldest first
while IFS= read -r vm; do
  [ -n "$vm" ] || continue
  echo "events of ${vm##*/}:"
  govc events -n 30 "$vm" 2>/dev/null | cat |
    sed -E 's/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/<ip>/g' | tail -12 || true
done < <(govc find "/$TF_VAR_datacenter/vm/$TF_VAR_folder" -type m -name "tmp-e2e-onigirazu-$RUN_ID-*" 2>/dev/null)
log "Waiting for SSH"
for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
  for _ in $(seq 60); do on_host "$h" true 2>/dev/null && break; sleep 5; done
  # say how the host refuses, not just that it does (key not applied, sshd down, no route)
  on_host "$h" true 2>/dev/null || echo "$h: $(timeout 20 ssh "${SSH_OPTS[@]}" -v "e2e@$(host_ip "$h")" true 2>&1 | grep -E 'Permission denied|Connection refused|timed out|No route|Authentications that can continue|Offering|Server accepts' | tail -3 | paste -sd' | ' -)"
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
  APPLY_RC=0
  (cd "$dir" && timeout 1200 "$BIN" apply playbook.yml -i "$INVENTORY" --state "$state" \
    --log-format json --no-color >"$WORK/apply.log" 2>&1) || APPLY_RC=$?
  records | jq -c 'select(.fields.type == "task_end") | .fields' > "$WORK/events.jsonl" || true
}

# First error lines of the last apply, for the job log
apply_errors() {
  # the failed tasks' own messages first (the module's reason), then the log
  jq -r 'select(.success == false and .msg != null and .msg != "") | "      \(.host) \(.task): \(.msg)"' "$WORK/events.jsonl" 2>/dev/null | cut -c1-400 | head -4
  records | jq -r 'select(.level == "ERROR" or .level == "WARN") | "      \(.level): \(.message)"' |
    cut -c1-400 | head -"${1:-4}"
}

# Drops task names listed in the case's EXPECTED_FAILED_TASKS (failures a
# rescue or ignore_errors handles)
expected_failures_out() {
  if [ -f "$1/EXPECTED_FAILED_TASKS" ]; then grep -vxF -f "$1/EXPECTED_FAILED_TASKS" || true; else cat; fi
}

# timeout(1) ended the last apply: its tasks failed because the run was stopped
timed_out() { [ "${APPLY_RC:-0}" = 124 ] && echo " (apply timed out after 1200s)" || true; }

record() { printf '%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" >> "$RESULTS"; echo "  [$3] $1 / $2 ${4:+- $4}"; }

for c in $cases; do
  [ -f "$HERE/cases/$c/playbook.yml" ] || continue
  # Work on a copy: cases may write next to their playbook (fetch)
  dir="$WORK/cases/$c"
  mkdir -p "$WORK/cases" && cp -R "$HERE/cases/$c" "$dir"
  log "Case $c"

  if [ -f "$dir/setup.sh" ]; then
    for h in $(jq -r 'keys[]' <<<"$hosts_json"); do
      out="$(cat "$HERE/setup-lib.sh" "$dir/setup.sh" | on_host "$h" 'sudo -n bash -s' 2>&1)" ||
        echo "  setup failed on $h: $(echo "$out" | tail -3 | paste -sd' ' -)"
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
    failed="$(jq -r --arg h "$h" 'select(.host == $h and .success != true) | .task' "$WORK/events.jsonl" | expected_failures_out "$dir")"
    ran="$(jq -r --arg h "$h" 'select(.host == $h) | .task' "$WORK/events.jsonl" | wc -l | tr -d ' ')"
    if [ "$ran" = 0 ]; then
      record "$c" "$h" FAIL "no task ran: $(records | jq -r 'select(.level == "ERROR") | .message' | head -1 | cut -c1-200)"
      continue
    fi
    if [ -n "$failed" ]; then
      record "$c" "$h" FAIL "apply failed: $(echo "$failed" | paste -sd, -)$(timed_out)"
      apply_errors
      continue
    fi
    if [ -f "$dir/verify.sh" ] && ! out="$(on_host "$h" 'sudo -n bash -s' < "$dir/verify.sh" 2>&1)"; then
      record "$c" "$h" FAIL "verify: $(echo "$out" | grep -v '^$' | tail -2 | paste -sd' ' - | cut -c1-300)"; continue
    fi
    if [ -f "$dir/verify-local.sh" ] && ! out="$(cd "$dir" && HOST="$h" HOST_IP="$(host_ip "$h")" KEY="$KEY" BIN="$BIN" INVENTORY="$INVENTORY" bash verify-local.sh 2>&1)"; then
      record "$c" "$h" FAIL "verify-local: $(echo "$out" | grep -v '^$' | tail -2 | paste -sd' ' - | cut -c1-300)"; continue
    fi
    record "$c" "$h" PASS "apply+verify"
    # "note: ..." lines of a passing verify-local are shown (timings and such)
    grep '^note: ' <<<"${out:-}" | sed "s/^/      $h /" || true
    out=""
    passed="$passed $h"
  done

  [ -f "$dir/NOT_IDEMPOTENT" ] && continue
  [ -n "$passed" ] || continue
  apply "$dir"
  for h in $passed; do
    ran="$(jq -r --arg h "$h" 'select(.host == $h) | .task' "$WORK/events.jsonl" | wc -l | tr -d ' ')"
    [ "$ran" != 0 ] || { record "$c" "$h" FAIL "second apply: no task ran"; continue; }
    changed="$(jq -r --arg h "$h" 'select(.host == $h and .changed == true) | .task' "$WORK/events.jsonl")"
    failed="$(jq -r --arg h "$h" 'select(.host == $h and .success != true) | .task' "$WORK/events.jsonl" | expected_failures_out "$dir")"
    if [ -n "$failed" ]; then record "$c" "$h" FAIL "second apply failed: $(echo "$failed" | paste -sd, -)$(timed_out)"; apply_errors
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

#!/usr/bin/env bash
# Load test: onigirazu and ansible-playbook run bench/site.yml on fresh
# disposable vSphere VMs, for each host count. Per tool and host count: new
# VMs, converge, a second run (must change nothing), check mode; wall time,
# CPU time and peak memory of the control side are recorded.
#
# Environment: as e2e/run.sh (TF_VAR_* estate, govc). Optional:
#   BENCH_HOSTS   host counts, default "1 5 10"
#   BENCH_TOOLS   default "onigirazu ansible"
#   BENCH_VARS    extra vars for both tools, JSON (e.g. {"bench_files": 500})
#   ANSIBLE_PLAYBOOK (default ansible-playbook from PATH), RUN_ID, RUN_URL
# Results: $BENCH_OUT (default ./bench-results) results.tsv + summary.md
# shellcheck disable=SC2154 # TF_VAR_* come from the caller
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
TF_DIR="$ROOT/e2e/terraform"
WORK="$(mktemp -d -t onigirazu-e2e.XXXXXX)"
OUT="${BENCH_OUT:-$PWD/bench-results}"
BIN="$WORK/onigirazu"
KEY="$WORK/id_e2e"
RUN_ID="${RUN_ID:-local-$(date -u +%m%d%H%M)}-b"
RUN_URL="${RUN_URL:-local run}"
HOSTS="${BENCH_HOSTS:-1 5 10}"
TOOLS="${BENCH_TOOLS:-onigirazu ansible}"
VARS="${BENCH_VARS:-{\}}"
ANSIBLE="${ANSIBLE_PLAYBOOK:-ansible-playbook}"
mkdir -p "$OUT"
RESULTS="$OUT/results.tsv"
printf 'tool\thosts\tphase\tseconds\tcpu_user\tcpu_sys\tpeak_rss_mb\tchanged\tfailed\trc\n' > "$RESULTS"

log() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

TFSTATE="$WORK/tf.tfstate"
TFVARS="$WORK/run.tfvars.json"
destroy() {
  [ -f "$TFSTATE" ] || return 0
  terraform -chdir="$TF_DIR" destroy -auto-approve -input=false -state="$TFSTATE" -var-file="$TFVARS" >/dev/null ||
    echo "destroy failed; the janitor will remove the VMs"
  rm -f "$TFSTATE"
}
cleanup() { local rc=$?; destroy; rm -rf "$WORK"; exit "$rc"; }
trap cleanup EXIT

for v in vsphere_server vsphere_user vsphere_password datacenter cluster host datastore network folder library; do
  n="TF_VAR_$v"; [ -n "${!n:-}" ] || die "$n is not set"
done
command -v "$ANSIBLE" >/dev/null || die "$ANSIBLE not found"

export GOVC_URL="$TF_VAR_vsphere_server" GOVC_USERNAME="$TF_VAR_vsphere_user" GOVC_PASSWORD="$TF_VAR_vsphere_password" GOVC_INSECURE=1
export GOVC_DATACENTER="$TF_VAR_datacenter"
# the e2e base of Ubuntu 24.04: docker, databases and images are in it
# shellcheck disable=SC2034 # read by images.sh
E2E_IMAGES="u2404=ubuntu-24.04" E2E_BASE=1
# shellcheck source=e2e/images.sh
. "$ROOT/e2e/images.sh"
resolve_images || exit 1
template="$(jq -r '.u2404' <<<"$images_json")"

log "Building onigirazu"
(cd "$ROOT" && go build -o "$BIN" ./cmd/onigirazu)
ssh-keygen -q -t ed25519 -N '' -C "onigirazu-bench-$RUN_ID" -f "$KEY"
for try in 1 2 3; do
  terraform -chdir="$TF_DIR" init -input=false >/dev/null && break
  [ "$try" = 3 ] && exit 1; sleep 15
done

# measure TOOL HOSTS PHASE command...: wall/CPU/peak memory of the command
# and its children; the log is kept for the counts
measure() {
  local tool="$1" hosts="$2" phase="$3"; shift 3
  local logf="$OUT/$tool-$hosts-$phase.log"
  (cd "$HERE" && python3 - "$@" > "$WORK/m.json" 2> "$logf") <<'PY'
import json, resource, subprocess, sys, time
t = time.time()
rc = subprocess.call(sys.argv[1:], stdout=sys.stderr, stderr=sys.stderr)
r = resource.getrusage(resource.RUSAGE_CHILDREN)
print(json.dumps({"seconds": round(time.time() - t, 2), "user": round(r.ru_utime, 2),
                  "sys": round(r.ru_stime, 2), "rss": round(r.ru_maxrss / 1024, 1), "rc": rc}))
PY
  local changed failed
  if [ "$tool" = ansible ]; then
    changed="$( (grep -oE 'changed=[0-9]+' "$logf" || true) | awk -F= '{s+=$2} END {print s+0}')"
    failed="$( (grep -oE '(failed|unreachable)=[0-9]+' "$logf" || true) | awk -F= '{s+=$2} END {print s+0}')"
  else
    changed="$( (grep -oE 'CHANGED:[0-9]+' "$logf" || true) | awk -F: '{s+=$2} END {print s+0}')"
    failed="$( (grep -oE 'FAILED:[0-9]+' "$logf" || true) | awk -F: '{s+=$2} END {print s+0}')"
  fi
  jq -r --arg t "$tool" --arg h "$hosts" --arg p "$phase" --arg c "$changed" --arg f "$failed" \
    '[$t, $h, $p, .seconds, .user, .sys, .rss, $c, $f, .rc] | @tsv' "$WORK/m.json" | tee -a "$RESULTS"
}

round=0
for n in $HOSTS; do
  # alternate which tool goes first
  order="$TOOLS"
  [ $((round % 2)) = 1 ] && order="$(tr ' ' '\n' <<<"$TOOLS" | tac | paste -sd' ' -)"
  round=$((round + 1))
  for tool in $order; do
    log "$tool on $n host(s): creating VMs"
    jq -n --arg run_id "$RUN_ID-$tool-$n" --arg run_url "$RUN_URL" --arg tpl "$template" --arg key "$(cat "$KEY.pub")" --argjson n "$n" \
      '{run_id: ($run_id | ascii_downcase | gsub("[^a-z0-9-]"; "") | .[0:24]), run_url: $run_url, public_key: $key,
        images: ([range(1; $n + 1)] | map({key: "h\(.)", value: $tpl}) | from_entries)}' > "$TFVARS"
    terraform -chdir="$TF_DIR" apply -auto-approve -input=false -no-color -state="$TFSTATE" -var-file="$TFVARS" > "$WORK/tf.log" 2>&1 ||
      { grep -vE '^\s*$' "$WORK/tf.log" | sed -E 's/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/<ip>/g' | tail -20; die "terraform apply failed"; }
    hosts_json="$(terraform -chdir="$TF_DIR" output -state="$TFSTATE" -json hosts)"
    if [ -n "${GITHUB_ACTIONS:-}" ]; then for ip in $(jq -r '.[]' <<<"$hosts_json"); do echo "::add-mask::$ip"; done; fi

    inv="$WORK/inventory.ini"
    { echo "[all]"; jq -r 'to_entries[] | "\(.key) ansible_host=\(.value)"' <<<"$hosts_json"
      echo "[all:vars]"; echo "ansible_user=e2e"; echo "ansible_ssh_private_key_file=$KEY"; } > "$inv"
    for ip in $(jq -r '.[]' <<<"$hosts_json"); do
      for _ in $(seq 60); do
        ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 \
          -o LogLevel=ERROR "e2e@$ip" 'sudo -n true' 2>/dev/null && break
        sleep 5
      done
    done
    rm -rf "$HERE/fetched"

    if [ "$tool" = ansible ]; then
      cat > "$WORK/ansible.cfg" <<EOF
[defaults]
forks = $n
host_key_checking = False
retry_files_enabled = False
interpreter_python = auto_silent
[ssh_connection]
pipelining = True
ssh_args = -o ControlMaster=auto -o ControlPersist=120s -o UserKnownHostsFile=/dev/null
EOF
      export ANSIBLE_CONFIG="$WORK/ansible.cfg"
      cmd=("$ANSIBLE" -i "$inv" site.yml -e "$VARS")
      measure ansible "$n" converge "${cmd[@]}"
      measure ansible "$n" second "${cmd[@]}"
      measure ansible "$n" check "${cmd[@]}" --check
      unset ANSIBLE_CONFIG
    else
      export ONIGIRAZU_SSH_KNOWN_HOSTS_FILE="$WORK/known_hosts" ONIGIRAZU_MAX_CONCURRENCY="$n"
      cmd=("$BIN" apply site.yml -i "$inv" -e "$VARS" --state "$WORK/state-$n" --no-color)
      measure onigirazu "$n" converge "${cmd[@]}"
      measure onigirazu "$n" second "${cmd[@]}"
      measure onigirazu "$n" check "${cmd[@]}" --check
      # where the time goes: one more noop run with per-task durations
      (cd "$HERE" && "${cmd[@]}" -o json 2>/dev/null > "$WORK/tasks.out") || true
      # the report follows the progress bar on stdout
      python3 - "$WORK/tasks.out" > "$OUT/onigirazu-$n-tasks.json" <<'PY' || true
import json, re, sys
data = open(sys.argv[1], errors="replace").read()
for m in re.finditer(r'\{\s*"execution_id"', data):
    try:
        print(json.dumps(json.JSONDecoder().raw_decode(data[m.start():])[0]))
        break
    except ValueError:
        pass
PY
      rm -f "$WORK/known_hosts"
    fi
    log "$tool on $n host(s): destroying VMs"
    destroy
  done
done

# --- summary -------------------------------------------------------------------
python3 - "$RESULTS" > "$OUT/summary.md" <<'PY'
import csv, sys
rows = list(csv.DictReader(open(sys.argv[1]), delimiter="\t"))
by = {(r["tool"], r["hosts"], r["phase"]): r for r in rows}
hosts = sorted({int(r["hosts"]) for r in rows})
print("| hosts | phase | onigirazu s | ansible s | ratio | onigirazu CPU s | ansible CPU s | onigirazu MB | ansible MB | changed o/a | failed o/a |")
print("|---|---|---|---|---|---|---|---|---|---|---|")
for h in hosts:
    for p in ("converge", "second", "check"):
        o, a = by.get(("onigirazu", str(h), p)), by.get(("ansible", str(h), p))
        if not o or not a:
            continue
        cpu = lambda r: float(r["cpu_user"]) + float(r["cpu_sys"])
        ratio = float(a["seconds"]) / max(float(o["seconds"]), 0.01)
        print(f'| {h} | {p} | {o["seconds"]} | {a["seconds"]} | {ratio:.1f}x | {cpu(o):.1f} | {cpu(a):.1f} | '
              f'{o["peak_rss_mb"]} | {a["peak_rss_mb"]} | {o["changed"]}/{a["changed"]} | {o["failed"]}/{a["failed"]} |')
PY
for f in "$OUT"/onigirazu-*-tasks.json; do
  [ -s "$f" ] || continue
  python3 - "$f" >> "$OUT/summary.md" <<'PY' || true
import json, os, sys
d = json.load(open(sys.argv[1]))
tasks = sorted(d.get("tasks", []), key=lambda t: -t.get("duration", 0))
total = sum(t.get("duration", 0) for t in tasks) or 1
print(f"\n#### {os.path.basename(sys.argv[1])[:-5]}: slowest tasks of a noop run\n")
print("| task | s | share |")
print("|---|---|---|")
for t in tasks[:15]:
    print(f'| {t["name"]} | {t["duration"] / 1e9:.2f} | {100 * t["duration"] / total:.0f}% |')
PY
done
cat "$OUT/summary.md"
[ -n "${GITHUB_STEP_SUMMARY:-}" ] && cat "$OUT/summary.md" >> "$GITHUB_STEP_SUMMARY"
# a failed task or a second run that changed something makes the numbers moot
awk -F'\t' 'NR > 1 && ($9 > 0 || $10 != 0 || ($3 == "second" && $8 > 0)) {bad = 1; print "not clean: " $0} END {exit bad}' "$RESULTS"

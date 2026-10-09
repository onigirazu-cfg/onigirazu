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
GOSS_RESULTS="$OUT/goss.tsv"
printf 'tool\thosts\tphase\thost\tchecks\tfailed\n' > "$GOSS_RESULTS"
# goss checks the state the playbook leaves on every host after each run
GOSS_VERSION=0.4.10
GOSS_SHA256=26e365428946294bcec0c61d867bb3c8349f39feb3d0e6f59084e98632785cc7

log() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

TFSTATE="$WORK/tf.tfstate"
TFVARS="$WORK/run.tfvars.json"
destroy() {
  [ -f "$TFSTATE" ] || return 0
  # the leases of this run's VMs are removed after them (names as terraform makes them)
  local prefix macs
  prefix="tmp-e2e-onigirazu-$(printf '%s' "$RUN_ID" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9-')"
  # shellcheck disable=SC2046
  macs="$("$ROOT/e2e/dhcp-leases.sh" macs $(govc find "/$TF_VAR_datacenter/vm/$TF_VAR_folder" -type m -name "$prefix*" 2>/dev/null) | tr '\n' ' ')"
  # shellcheck disable=SC2046
  "$ROOT/e2e/dhcp-release.sh" "$KEY" $(terraform -chdir="$TF_DIR" output -state="$TFSTATE" -json hosts 2>/dev/null | jq -r '.[]?')
  terraform -chdir="$TF_DIR" destroy -auto-approve -input=false -state="$TFSTATE" -var-file="$TFVARS" >/dev/null ||
    echo "destroy failed; the janitor will remove the VMs"
  rm -f "$TFSTATE"
  # shellcheck disable=SC2086
  "$ROOT/e2e/dhcp-leases.sh" remove $macs
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

log "Fetching goss $GOSS_VERSION"
curl -fsSL -o "$WORK/goss.tgz" "https://github.com/goss-org/goss/releases/download/v$GOSS_VERSION/goss_${GOSS_VERSION}_linux_x86_64.tar.gz"
echo "$GOSS_SHA256  $WORK/goss.tgz" | sha256sum -c --quiet - || die "goss checksum mismatch"
tar xzf "$WORK/goss.tgz" -C "$WORK" goss
python3 "$HERE/goss.py" "$VARS" > "$WORK/goss.json"

log "Building onigirazu"
# the agents built into onigirazu, as in a release
(cd "$ROOT" && go generate ./internal/agentbin)
(cd "$ROOT" && go build -o "$BIN" ./cmd/onigirazu)
ssh-keygen -q -t ed25519 -N '' -C "onigirazu-bench-$RUN_ID" -f "$KEY"
for try in 1 2 3; do
  terraform -chdir="$TF_DIR" init -input=false >/dev/null && break
  [ "$try" = 3 ] && exit 1; sleep 15
done

# goss_check TOOL HOSTS PHASE: validates every host; a failed check is
# shown and fails the bench at the end
goss_check() {
  local tool="$1" n="$2" phase="$3" ip name out
  while read -r name ip; do
    out="$(ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR "e2e@$ip" \
      'sudo -n /tmp/goss -g /tmp/goss.json validate --format tap --no-color' 2>&1)" || true
    local total failed
    total="$( (grep -cE '^(ok|not ok) ' <<<"$out") || true)"
    failed="$( (grep -cE '^not ok ' <<<"$out") || true)"
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$tool" "$n" "$phase" "$name" "$total" "$failed" >> "$GOSS_RESULTS"
    if [ "$total" = 0 ] || [ "$failed" != 0 ]; then
      echo "goss: $tool $phase on $name: $failed of $total checks failed"
      (grep -E '^not ok ' <<<"$out" || tail -5 <<<"$out") | head -20
    fi
  done < <(jq -r 'to_entries[] | "\(.key) \(.value)"' <<<"$hosts_json")
}

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
    for ip in $(jq -r '.[]' <<<"$hosts_json"); do
      scp -q -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
        "$WORK/goss" "$WORK/goss.json" "e2e@$ip:/tmp/" || die "cannot copy goss to a VM"
    done
    # fresh package lists before the timed runs: the base image may come
    # with none, and cache_valid_time would then skip the update
    for ip in $(jq -r '.[]' <<<"$hosts_json"); do
      ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR "e2e@$ip" \
        'sudo -n apt-get -o DPkg::Lock::Timeout=300 update -qq >/dev/null' || die "apt-get update failed on a VM"
    done

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
      goss_check ansible "$n" converge
      measure ansible "$n" second "${cmd[@]}"
      goss_check ansible "$n" second
      measure ansible "$n" check "${cmd[@]}" --check
      unset ANSIBLE_CONFIG
    else
      export ONIGIRAZU_SSH_KNOWN_HOSTS_FILE="$WORK/known_hosts" ONIGIRAZU_MAX_CONCURRENCY="$n"
      cmd=("$BIN" apply site.yml -i "$inv" -e "$VARS" --state "$WORK/state-$n" --no-color)
      measure onigirazu "$n" converge "${cmd[@]}"
      # the run's record (per-task wall times): where the converge's time goes
      cp "$HOME/.onigirazu/cache/executions/current.json" "$OUT/onigirazu-$n-converge.json" 2>/dev/null || true
      goss_check onigirazu "$n" converge
      measure onigirazu "$n" second "${cmd[@]}"
      goss_check onigirazu "$n" second
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
printf '\nonigirazu ran with remote_server %s.\n' "${ONIGIRAZU_REMOTE_SERVER:-auto}" >> "$OUT/summary.md"
for f in "$OUT"/onigirazu-*-converge.json "$OUT"/onigirazu-*-tasks.json; do
  [ -s "$f" ] || continue
  python3 - "$f" >> "$OUT/summary.md" <<'PY' || true
import datetime, json, os, sys
d = json.load(open(sys.argv[1]))
# a loop reports no duration of its own: the wall time of a task is the gap
# to the start of the next one
at = lambda s: datetime.datetime.fromisoformat(s.rstrip("Z")[:26])
tasks = [t for t in d.get("tasks", []) if not t["start_time"].startswith("0001")]
ends = [t["start_time"] for t in tasks[1:]] + [d["end_time"]]
rows = sorted(((at(e) - at(t["start_time"])).total_seconds(), t["name"], t["total"]) for t, e in zip(tasks, ends))[::-1]
total = (at(d["end_time"]) - at(d["start_time"])).total_seconds() or 1
kind = "the converge" if sys.argv[1].endswith("-converge.json") else "a noop run"
print(f"\n#### {os.path.basename(sys.argv[1])[:-5]}: slowest tasks of {kind} ({total:.1f} s)\n")
print("| task | items | s | share |")
print("|---|---|---|---|")
for s, name, n in rows[:15]:
    print(f"| {name.replace(' (item 1)', '')} | {n} | {s:.2f} | {100 * s / total:.0f}% |")
PY
done
python3 - "$GOSS_RESULTS" >> "$OUT/summary.md" <<'PY'
import csv, sys
rows = list(csv.DictReader(open(sys.argv[1]), delimiter="\t"))
if rows:
    print("\n#### State checks (goss): every host after each run\n")
    print("| tool | hosts | phase | hosts passed | checks per host | failed checks |")
    print("|---|---|---|---|---|---|")
    groups = {}
    for r in rows:
        groups.setdefault((r["tool"], int(r["hosts"]), r["phase"]), []).append(r)
    for (tool, n, phase), rs in sorted(groups.items(), key=lambda x: (x[0][1], x[0][0], x[0][2])):
        ok = sum(1 for r in rs if r["failed"] == "0" and r["checks"] != "0")
        print(f'| {tool} | {n} | {phase} | {ok}/{len(rs)} | {max(int(r["checks"]) for r in rs)} | {sum(int(r["failed"]) for r in rs)} |')
PY
cat "$OUT/summary.md"
[ -n "${GITHUB_STEP_SUMMARY:-}" ] && cat "$OUT/summary.md" >> "$GITHUB_STEP_SUMMARY"
# a failed task or a second run that changed something makes the numbers moot
awk -F'\t' 'NR > 1 && ($9 > 0 || $10 != 0 || ($3 == "second" && $8 > 0)) {bad = 1; print "not clean: " $0} END {exit bad}' "$RESULTS"
awk -F'\t' 'NR > 1 && ($5 == 0 || $6 != 0) {bad = 1; print "state check failed: " $0} END {exit bad}' "$GOSS_RESULTS"

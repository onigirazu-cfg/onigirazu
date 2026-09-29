#!/usr/bin/env bash
# Runs each playbook in compat/cases with ansible-playbook and with onigirazu,
# each on its own fresh container, twice, and compares per task status and
# message, and the files the playbook left under /root/compat.
# Needs docker, ansible-playbook (ansible-core), go and the rig image
# (make docker-up RIG=ubuntu2404).
# Usage: compat/run.sh [case.yml ...]   COMPAT_IMAGE=onigirazu-ubuntu2404
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
IMAGE="${COMPAT_IMAGE:-onigirazu-ubuntu2404}"
KEY="$ROOT/docker/ssh/id_rsa"
WORK="$(mktemp -d)"
containers=()
cleanup() { [ "${#containers[@]}" = 0 ] || docker rm -f "${containers[@]}" >/dev/null 2>&1; rm -rf "$WORK"; }
trap cleanup EXIT

[ -f "$KEY" ] || { echo "no $KEY: run make docker-up first" >&2; exit 1; }
docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "no image $IMAGE: make docker-up RIG=ubuntu2404" >&2; exit 1; }
command -v ansible-playbook >/dev/null || { echo "ansible-playbook not found" >&2; exit 1; }
(cd "$ROOT" && go build -o "$WORK/onigirazu" ./cmd/onigirazu)

start() {  # prints the host port of a fresh container's sshd
  local id port
  id="$(docker run -d -p 127.0.0.1::22 "$IMAGE")"
  containers+=("$id")
  port="$(docker port "$id" 22/tcp | head -1 | sed 's/.*://')"
  for _ in $(seq 30); do
    ssh -i "$KEY" -p "$port" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
      -o LogLevel=ERROR root@127.0.0.1 true 2>/dev/null && { echo "$id $port"; return; }
    sleep 1
  done
  echo "container $id: no ssh" >&2; return 1
}

# onigirazu reports each loop item ("name (item N)"), Ansible the whole loop:
# items are folded into one task, changed if any changed, skipped if all were
normalize() {
  jq -s -c 'reduce .[] as $r ([];
      ($r.task | sub(" \\(item [0-9]+\\)$"; "")) as $t
      | if ($r.task | test(" \\(item [0-9]+\\)$")) and ($r.task | test(" \\(item 1\\)$") | not)
           and length > 0 and .[-1].task == $t and .[-1].loop then
          .[-1] |= (.changed = (.changed or $r.changed) | .skipped = (.skipped and $r.skipped)
                    | .success = (.success and $r.success) | .ignored = (.ignored or $r.ignored))
        elif ($r.task | test(" \\(item [0-9]+\\)$")) then . + [$r + {task: $t, loop: true, msg: (if $r.module == "debug" then "All items completed" else "" end)}]
        else . + [$r] end) | .[]' "$1" |
  jq -r '(if .ignored then "ignored" elif .success == false then "failed" elif .skipped then "skipped"
     elif .changed then "changed" else "ok" end) as $st
    | (.msg // "" | tostring) as $raw
    | ($raw | if . == "True" then "true" elif . == "False" then "false" else . end
       | . as $s | try (fromjson | tojson) catch $s) as $msg
    | [.task, $st, (if ($st == "failed" or $st == "ignored") and .module != "fail" then "" else $msg end)]
    | join(" | ")'
}
files() {  # container -> checksums of /root/compat
  docker exec "$1" sh -c 'cd /root/compat 2>/dev/null && find . -type f | sort | xargs -r md5sum' || true
}

# hosts: "h" alone, or h1..hN for "# compat-hosts: N" in the case, in groups
# odd and even
host_names() {
  if [ "$1" = 1 ]; then echo h; else seq -f 'h%g' 1 "$1"; fi
}

inventory() {  # file name:port...
  local file="$1" odd="" even="" n=0 entry
  shift
  for entry in "$@"; do
    n=$((n + 1))
    local line="        ${entry%%:*}: {ansible_host: 127.0.0.1, ansible_port: ${entry#*:}}"$'\n'
    if [ $((n % 2)) = 1 ]; then odd+="$line"; else even+="$line"; fi
  done
  {
    echo "all:"
    echo "  vars:"
    echo "    ansible_user: root"
    echo "    ansible_ssh_private_key_file: $KEY"
    echo "    ansible_ssh_common_args: \"-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null\""
    echo "  children:"
    echo "    odd:"
    echo "      hosts:"
    printf '%s' "$odd"
    if [ -n "$even" ]; then
      echo "    even:"
      echo "      hosts:"
      printf '%s' "$even"
    fi
  } > "$file"
}

# the task results of one host
of_host() { jq -c --arg h "$2" 'select(.host == $h)' "$1"; }

for f in start inventory normalize files of_host; do
  declare -F "$f" >/dev/null || { echo "compat/run.sh: $f is not defined" >&2; exit 2; }
done
if [ $# -gt 0 ]; then playbooks=("$@"); else playbooks=("$HERE"/cases/*.yml); fi
pass=0 fail=0
for pb in "${playbooks[@]}"; do
  pb="$(cd "$(dirname "$pb")" && pwd)/$(basename "$pb")"
  name="$(basename "$pb" .yml)"
  count="$(sed -n 's/^# compat-hosts: *\([0-9]*\).*/\1/p' "$pb" | head -1)"
  count="${count:-1}"
  hosts="$(host_names "$count")"
  a_ids=() b_ids=() a_entries=() b_entries=()
  for h in $hosts; do
    read -r id port < <(start); a_ids+=("$id"); a_entries+=("$h:$port")
    read -r id port < <(start); b_ids+=("$id"); b_entries+=("$h:$port")
  done
  inventory "$WORK/a.yml" "${a_entries[@]}"
  inventory "$WORK/b.yml" "${b_entries[@]}"
  report=""
  for run in 1 2; do
    (cd "$(dirname "$pb")" && ANSIBLE_STDOUT_CALLBACK=compat_results ANSIBLE_CALLBACK_PLUGINS="$HERE/callback_plugins" \
      ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_NOCOLOR=1 \
      ansible-playbook -i "$WORK/a.yml" "$pb" 2>/dev/null | grep '^{' > "$WORK/a$run.jsonl") || true
    (cd "$(dirname "$pb")" && ONIGIRAZU_SSH_KNOWN_HOSTS_FILE="$WORK/known_hosts" \
      "$WORK/onigirazu" apply "$pb" -i "$WORK/b.yml" --log-format json --no-color --state "$WORK/state-$name" 2>&1 |
      grep -o '{"timestamp.*' | jq -c 'select(.fields.type == "task_end") | .fields' > "$WORK/b$run.jsonl") || true
    # hosts run in parallel: each host's tasks are compared on their own
    for h in $hosts; do
      of_host "$WORK/a$run.jsonl" "$h" > "$WORK/ah.jsonl"
      of_host "$WORK/b$run.jsonl" "$h" > "$WORK/bh.jsonl"
      label="run $run"
      [ "$count" = 1 ] || label="run $run, $h"
      if ! d="$(diff <(normalize "$WORK/ah.jsonl") <(normalize "$WORK/bh.jsonl"))"; then
        report+=$'\n'"  $label tasks (< ansible, > onigirazu):"$'\n'"$(sed 's/^/    /' <<<"$d")"
      fi
    done
  done
  i=0
  for h in $hosts; do
    if ! d="$(diff <(files "${a_ids[$i]}") <(files "${b_ids[$i]}"))"; then
      report+=$'\n'"  /root/compat on $h (< ansible, > onigirazu):"$'\n'"$(sed 's/^/    /' <<<"$d")"
    fi
    i=$((i + 1))
  done
  docker rm -f "${a_ids[@]}" "${b_ids[@]}" >/dev/null
  if [ ! -s "$WORK/a1.jsonl" ]; then
    echo "ERROR $name: ansible-playbook ran no task (ansible-playbook --syntax-check $pb)"; fail=$((fail + 1)); continue
  fi
  if [ -z "$report" ]; then
    echo "PASS $name ($(wc -l < "$WORK/a1.jsonl" | tr -d ' ') task results)"; pass=$((pass + 1))
  else
    echo "DIFF $name$report"; fail=$((fail + 1))
  fi
done
echo "$pass passed, $fail differ"
[ "$fail" = 0 ]

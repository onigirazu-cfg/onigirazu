#!/usr/bin/env bash
# Scale test: N small sshd containers on one docker host, a short playbook,
# onigirazu at several concurrencies; wall time, CPU and peak memory of the
# control side. Runs on the docker host (the lab), never on production.
#
#   bench/scale.sh BIN N "CONCURRENCIES"     e.g. bench/scale.sh ./onigirazu 500 "50 100 500"
#   SERVER=sh|python|auto (default auto); KEEP=1 leaves the containers up
set -euo pipefail
BIN=$(realpath "${1:?onigirazu binary}")
N=${2:-300}
CONC=${3:-"50 100 $N"}
SERVER=${SERVER:-auto}
work=$(mktemp -d)
image=onigirazu-scale-sshd
label=onigirazu-scale

cleanup() {
  if [ "${KEEP:-0}" = 1 ]; then echo "kept: $work"; return; fi
  docker ps -aq --filter "label=$label" | xargs -r docker rm -f >/dev/null
  rm -rf "$work"
}
trap cleanup EXIT

ssh-keygen -q -t ed25519 -N '' -f "$work/key"
cat > "$work/Dockerfile" <<'EOF'
FROM alpine:3.20
RUN apk add --no-cache openssh-server && ssh-keygen -A && \
    sed -i 's/^#\?MaxSessions.*/MaxSessions 30/' /etc/ssh/sshd_config && \
    sed -i 's/^root:[^:]*:/root:*:/' /etc/shadow
COPY key.pub /root/.ssh/authorized_keys
CMD ["/usr/sbin/sshd", "-D", "-e"]
EOF
docker build -q -t "$image" "$work" >/dev/null

docker ps -aq --filter "label=$label" | xargs -r docker rm -f >/dev/null
echo "==> starting $N containers"
for i in $(seq 1 "$N"); do echo "$i"; done |
  xargs -P 16 -I{} docker run -d --label "$label" --name "scale-{}" --memory 64m "$image" >/dev/null

{
  echo "all:"
  echo "  vars:"
  echo "    ansible_user: root"
  echo "    ansible_ssh_private_key_file: $work/key"
  echo "    ansible_ssh_common_args: \"-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null\""
  echo "  hosts:"
  docker ps --filter "label=$label" --format '{{.Names}}' | sort -V | while read -r n; do
    echo "    $n: {ansible_host: $(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$n")}"
  done
} > "$work/inventory.yml"

cat > "$work/site.yml" <<'EOF'
- hosts: all
  gather_facts: true
  tasks:
    - file: {path: /etc/scale, state: directory, mode: "0755"}
    - copy: {dest: "/etc/scale/{{ item }}.conf", content: "name={{ item }} host={{ inventory_hostname }}\n", mode: "0644"}
      loop: [a, b, c, d, e]
    - lineinfile: {path: /etc/scale/a.conf, line: "extra=1"}
    - command: cat /etc/scale/a.conf
      register: out
      changed_when: false
    - assert: {that: ["'extra=1' in out.stdout"]}
EOF

run() {  # label, concurrency
  local log="$work/$1.log" t="$work/$1.time"
  /usr/bin/time -f '%e %U %S %M' -o "$t" env ONIGIRAZU_REMOTE_SERVER="$SERVER" ONIGIRAZU_MAX_CONCURRENCY="$2" \
    "$BIN" apply "$work/site.yml" -i "$work/inventory.yml" --state "$work/state-$1" --no-color > "$log" 2>&1 || { tail -20 "$log"; return 1; }
  read -r wall user sys rss < "$t"
  printf '%-14s %5s %8.1f %8.1f %8.0f\n' "$1" "$2" "$wall" "$(echo "$user + $sys" | bc)" "$((rss / 1024))"
}

printf '%-14s %5s %8s %8s %8s\n' run conc wall_s cpu_s rss_MB
for c in $CONC; do
  docker ps -q --filter "label=$label" | xargs -r -P 16 -I{} docker exec {} rm -rf /etc/scale /root/.onigirazu
  run "converge-$c" "$c"
  run "again-$c" "$c"
done

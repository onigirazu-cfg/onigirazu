#!/usr/bin/env bash
# an inventory without all: is read; the group and its vars are there
set -euo pipefail
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
printf 'sensors:\n  hosts:\n    %s: {ansible_host: %s, ansible_user: e2e, ansible_ssh_private_key_file: %s, ansible_ssh_common_args: "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"}\n  vars: {pilot_group: sensors}\n' "$HOST" "$HOST_IP" "$KEY" > "$work/hosts.yml"
"$BIN" inventory --list --json -i "$work/hosts.yml" > "$work/inv.json"
python3 -c "
import json; d=json.load(open('$work/inv.json')); assert '$HOST' in d['sensors']['hosts'], d; assert d['_meta']['hostvars']['$HOST']['pilot_group']=='sensors', d"
echo "note: inventory without all: read, group sensors with $HOST"

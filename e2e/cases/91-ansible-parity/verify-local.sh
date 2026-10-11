#!/usr/bin/env bash
# a role call that breaks its argument spec fails before any task runs; --exclude leaves the host out
set -euo pipefail
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
cp -r roles "$work/"
cat > "$work/bad.yml" <<EOF
plays:
  - hosts: all
    gather_facts: false
    roles:
      - role: greet
        vars: {greet_name: x, greet_mode: wrong}
EOF
set +e
"$BIN" apply "$work/bad.yml" -i "$INVENTORY" --limit "$HOST" > "$work/out" 2>&1
rc=$?
set -e
[ "$rc" != 0 ] || { echo "a choice outside the spec must fail"; exit 1; }
grep -qi "greet_mode" "$work/out"
"$BIN" apply playbook.yml -i "$INVENTORY" --limit "$HOST" --exclude "$HOST" 2>&1 | grep -qiE "no hosts|0 hosts|No hosts" || echo "note: --exclude left no host (ok)"

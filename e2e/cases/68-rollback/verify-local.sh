# apply v2 over v1, roll it back, and the host is at v1 again (drift says so)
set -e
out="$("$BIN" apply v2.yml -i "$INVENTORY" --limit "$HOST" 2>&1)"
id="$(echo "$out" | sed -n 's/.*Snapshot created: \([0-9]*\) .*/\1/p' | tail -1)"
test -n "$id"
"$BIN" rollback --snapshot "$id" -i "$INVENTORY" >/dev/null 2>&1
"$BIN" drift playbook.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1

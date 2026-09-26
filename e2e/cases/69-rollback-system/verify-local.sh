# packages, accounts and a service changed by a run come back with rollback
set -e
out="$("$BIN" apply changes.yml -i "$INVENTORY" --limit "$HOST" 2>&1)"
id="$(echo "$out" | sed -n 's/.*Snapshot created: \([0-9]*\) .*/\1/p' | tail -1)"
test -n "$id"
set +e; "$BIN" drift playbook.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1; rc=$?; set -e
test "$rc" = 2
"$BIN" rollback --snapshot "$id" -i "$INVENTORY" >/dev/null 2>&1
"$BIN" drift playbook.yml -i "$INVENTORY" --limit "$HOST"

# an unhealthy batch is rolled back and the rollout stops (exit 5); the
# healthy batch before it keeps its changes
set -e
"$BIN" apply start.yml -i "$INVENTORY" >/dev/null 2>&1
rc=0; "$BIN" apply rollout.yml -i "$INVENTORY" >out 2>&1 || rc=$?
test "$rc" = 5
grep -q "rolled back: 2 change(s) undone, healthy again" out
"$BIN" drift expected.yml -i "$INVENTORY" >/dev/null 2>&1

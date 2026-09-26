# drift: 0 in sync, 2 after a manual change (one task), --fix restores
set -e
d() { "$BIN" drift playbook.yml -i "$INVENTORY" --limit "$HOST" "$@" 2>/dev/null; }
d >/dev/null
"$BIN" apply tamper.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1
set +e; out="$(d --format json)"; rc=$?; set -e
test "$rc" = 2
test "$(echo "$out" | jq -r --arg h "$HOST" '.drift[$h] | map(.task) | join(",")')" = "Managed file"
echo "$out" | jq -r --arg h "$HOST" '.drift[$h][0].diff' | grep -qx -- '-edited by hand'
echo "$out" | jq -r --arg h "$HOST" '.drift[$h][0].diff' | grep -qx -- '+managed'
"$BIN" apply playbook.yml -i "$INVENTORY" --limit "$HOST" --check --diff 2>/dev/null | grep -qx -- '+managed'
# a second check keeps the first one's time as "since"
since1="$(echo "$out" | jq -r --arg h "$HOST" '.drift[$h][0].since')"
since2="$(set +e; d --format json | jq -r --arg h "$HOST" '.drift[$h][0].since')"
test -n "$since1" && test "$since1" != null && test "$since1" = "$since2"
d --fix >/dev/null
d >/dev/null
"$BIN" drift playbook.yml --history | grep -q 'CHECKED'

# `onigirazu verify` runs only the checks: all pass; a playbook with a failing
# check exits 1 and names it
set -e
out="$("$BIN" verify playbook.yml -i "$INVENTORY" --limit "$HOST" 2>&1)" || { echo "verify failed: $(echo "$out" | tail -4 | paste -sd' | ' -)"; exit 1; }
echo "$out" | grep -q "All 15 check(s) passed on 1 host(s)"
rc=0; out="$("$BIN" verify failing.yml -i "$INVENTORY" --limit "$HOST" 2>&1)" || rc=$?
test "$rc" = 1
echo "$out" | grep -q "✗ port 59998: not listening"
echo "$out" | grep -q "✗ file /etc/onigirazu-e2e-verify.conf: contains lacks"
echo "note: 15 checks pass, a failing playbook exits 1 with the checks named"

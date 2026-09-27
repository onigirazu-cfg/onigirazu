# the security policy stops run as it stops apply
set -e
! "$BIN" run "$HOST" -m command -a "echo onigirazu-e2e-forbidden" -i "$INVENTORY" --security-policy policy.json >out 2>&1
grep -q "blocked pattern" out
! "$BIN" run "$HOST" -m file -a "path=/tmp/x state=touch" -i "$INVENTORY" --security-policy policy.json >out 2>&1
grep -q "not in allowed modules" out
"$BIN" run "$HOST" -m command -a "true" -i "$INVENTORY" --security-policy policy.json >/dev/null 2>&1

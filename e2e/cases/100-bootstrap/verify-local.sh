#!/usr/bin/env bash
# bootstrap through the e2e user (sudo) creates oni-deploy with a fresh key
# and passwordless sudo, proves the login, prints the inventory line
set -euo pipefail
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
ssh-keygen -q -t ed25519 -N '' -f "$work/key"
"$BIN" bootstrap "$HOST_IP" -u e2e --key "$KEY" --new-user oni-deploy --pubkey "$work/key.pub" > "$work/out" 2>&1 || { tail -5 "$work/out"; exit 1; }
grep -q "^bootstrap: done" "$work/out"
grep -q "ansible_user=oni-deploy" "$work/out"
got=$(ssh -i "$work/key" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR "oni-deploy@$HOST_IP" 'id -un; sudo -n id -un' | paste -sd' ' -)
test "$got" = "oni-deploy root"
# a second bootstrap changes nothing
"$BIN" bootstrap "$HOST_IP" -u e2e --key "$KEY" --new-user oni-deploy --pubkey "$work/key.pub" --check > "$work/out2" 2>&1
! grep -qE "CHANGED:[1-9]" "$work/out2" || { echo "the second bootstrap must be idempotent"; grep -E "CHANGED" "$work/out2"; exit 1; }
echo "note: $(grep '^bootstrap: done' "$work/out")"

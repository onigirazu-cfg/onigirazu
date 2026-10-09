# packages, accounts and a service changed by a run come back with rollback
set -e
out="$("$BIN" apply changes.yml -i "$INVENTORY" --limit "$HOST" 2>&1)"
id="$(echo "$out" | sed -n 's/.*Snapshot created: \([0-9]*\) .*/\1/p' | tail -1)"
[ -n "$id" ] || { echo "no snapshot id in the apply output: $(echo "$out" | grep -v '^$' | tail -3 | paste -sd' ' -)"; exit 1; }
set +e; "$BIN" drift playbook.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1; rc=$?; set -e
test "$rc" = 2
out="$("$BIN" rollback --snapshot "$id" -i "$INVENTORY" 2>&1)" || { echo "rollback failed: $(echo "$out" | grep -v '^$' | tail -3 | paste -sd' ' -)"; exit 1; }
# a drift after the rollback: say what the rollback did and what the host holds
if ! "$BIN" drift playbook.yml -i "$INVENTORY" --limit "$HOST"; then
  echo "rollback output: $(echo "$out" | grep -iE 'rolling back|rolled back|failed|error|package' | paste -sd' | ' - | cut -c1-600)"
  echo "note: $(ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "e2e@$HOST_IP" 'dpkg-query -W -f="tree: ${Status}\n" tree 2>&1; sudo -n fuser /var/lib/dpkg/lock-frontend 2>&1; systemctl is-active apt-daily.service apt-daily-upgrade.service unattended-upgrades.service 2>&1 | paste -sd, -' 2>&1 | paste -sd' ' -)"
  exit 1
fi

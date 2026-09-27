# --timeout ends the run, the state is still saved
set -e
rm -f timeout-state
! "$BIN" apply slow.yml -i "$INVENTORY" --limit "$HOST" --timeout 3s --state timeout-state >out 2>&1
grep -q "deadline exceeded" out
! grep -q "Failed to save state" out
test -s timeout-state

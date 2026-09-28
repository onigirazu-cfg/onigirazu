# import writes a playbook for the host; planned against the same host it
# has nothing to change (exit 0), and it holds what the playbook made
set -e
out="import-$HOST"
"$BIN" import "$HOST" -i "$INVENTORY" -o "$out" --force > log 2>&1 || {
  # the tasks that still change, on the last line (the harness shows that one)
  (sed -n '/^## Check/,/^## [^C]/p' "$out/IMPORT_REPORT.md" | grep '^- ' || tail -3 log) | head -6 | cut -c1-200 | paste -sd';' -
  exit 1; }
grep -q "nothing to change" log
grep -rq "File /etc/onigirazu-import.conf" "$out/roles"
grep -rq "User e2eimport" "$out/roles"
test -f "$out/IMPORT_REPORT.md"

# all hosts at once, one time: shared roles, still nothing to change
if mkdir all.lock 2>/dev/null; then
  "$BIN" import all -i "$INVENTORY" -o import-all --force > log-all 2>&1 || { tail -30 log-all; exit 1; }
  grep -q "nothing to change" log-all
  grep -q "^Roles: common" log-all
  grep -rq "File /etc/onigirazu-import.conf" import-all/roles/common
fi

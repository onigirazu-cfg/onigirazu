# import writes a playbook for the host; planned against the same host it
# has nothing to change (exit 0), and it holds what the playbook made
set -e
out="import-$HOST"
"$BIN" import "$HOST" -i "$INVENTORY" -o "$out" --force > log 2>&1 || {
  # the tasks that still change, on the last line (the harness shows that one)
  (sed -n '/^## Check/,/^## [^C]/p' "$out/IMPORT_REPORT.md" | grep '^- ' || tail -3 log) | head -6 | cut -c1-200 | paste -sd';' -
  exit 1; }
grep -q "nothing to change" log
tasks="$out/roles/host_$(echo "$HOST" | tr -c 'A-Za-z0-9_\n' '_')/tasks/main.yml"
grep -q "File /etc/onigirazu-import.conf" "$tasks"
grep -q "User e2eimport" "$tasks"
test -f "$out/IMPORT_REPORT.md"

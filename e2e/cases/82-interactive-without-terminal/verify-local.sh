# --interactive without a terminal runs with the normal output instead of
# waiting forever for a dashboard that cannot start
set -e
timeout 120 "$BIN" apply playbook.yml -i "$INVENTORY" --limit "$HOST" --interactive </dev/null >out 2>&1
grep -q "Interactive mode needs a terminal" out
grep -q "Per-Host Results" out

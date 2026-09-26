# --tags selects a task whose tags are one comma-separated string
set -e
"$BIN" apply tagged.yml -i "$INVENTORY" --limit "$HOST" --tags kw-setup >/dev/null 2>&1
"$BIN" drift expected.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1

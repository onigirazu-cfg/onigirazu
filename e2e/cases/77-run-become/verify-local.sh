# run -b writes into /root, with {{ }} and -e rendered per host
set -e
"$BIN" run "$HOST" -m copy -a 'content="{{ inventory_hostname }} v{{ version }}" dest=/root/onigirazu-e2e-run mode=0600' \
  -i "$INVENTORY" -b -e '{"version": 7}' >/dev/null 2>&1
"$BIN" drift expected.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1

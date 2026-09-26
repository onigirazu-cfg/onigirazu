# The same host reached through -i user@addr:port, (a host list) and through
# an Ansible inventory script; drift through the regular inventory then
# finds the markers both left
set -e
block="$(grep -A5 "^      $HOST:\$" "$INVENTORY")"
addr="$(echo "$block" | sed -n 's/.*onigirazu_host: //p')"
port="$(echo "$block" | sed -n 's/.*onigirazu_port: //p')"
key="$(sed -n 's/.*onigirazu_ssh_private_key_file: //p' "$INVENTORY" | head -1)"
test -n "$addr" -a -n "$key"

"$BIN" apply marker-list.yml -i "e2e@$addr:$port," --private-key "$key" >/dev/null 2>&1

script="$PWD/inv-script"
cat > "$script" <<SCRIPT
#!/bin/sh
[ "\$1" = "--list" ] || exit 1
echo '{"dyn": {"hosts": ["$HOST"]}, "_meta": {"hostvars": {"$HOST": {"ansible_host": "$addr", "ansible_port": $port, "ansible_user": "e2e", "ansible_ssh_private_key_file": "$key"}}}}'
SCRIPT
chmod +x "$script"
"$BIN" apply marker-script.yml -i "$script" >/dev/null 2>&1

"$BIN" drift expected.yml -i "$INVENTORY" --limit "$HOST" >/dev/null 2>&1

# run -o json: stdout is only the JSON
"$BIN" run all -m ping -i "e2e@$addr:$port," -k "$key" -o json 2>/dev/null | jq -e '.results | length == 1' >/dev/null

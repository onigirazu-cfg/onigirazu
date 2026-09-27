# user, port and key given only as group variables (all.vars), as many
# Ansible inventories do
set -e
block="$(grep -A5 "^      $HOST:\$" "$INVENTORY")"
addr="$(echo "$block" | sed -n 's/.*onigirazu_host: //p')"
port="$(echo "$block" | sed -n 's/.*onigirazu_port: //p')"
key="$(sed -n 's/.*onigirazu_ssh_private_key_file: //p' "$INVENTORY" | head -1)"
cat > group-vars-inv.yml <<INV
all:
  vars:
    ansible_user: e2e
    ansible_port: "$port"
    ansible_ssh_private_key_file: $key
  children:
    web:
      hosts:
        $HOST:
          ansible_host: $addr
INV
"$BIN" run all -m command -a "id -un" -i group-vars-inv.yml -o json 2>/dev/null | jq -e '.results[0].status == "success"' >/dev/null

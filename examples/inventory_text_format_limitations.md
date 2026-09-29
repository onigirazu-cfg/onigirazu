# Plain Host List Limitations

A plain host list file has one `[user@]host[:port]` per line; `#` starts a comment.
It is not the INI format: a `.ini` file (Ansible style, with groups and variables) is read
as INI. See [INVENTORY_FORMATS.md](../INVENTORY_FORMATS.md).

```text
192.168.1.10
deploy@192.168.1.20:2222
server.example.com
```

A plain list holds only address, port and user. Every host is in the group `all`, gets
port 22 and the local user unless given, and is named by its address.

It cannot hold:

- groups, group variables or host variables
- a key file, password or become password
- host key settings (`insecure_ignore_host_key`, `ansible_ssh_common_args`)

Key file and user for all hosts can still be given on the command line (`run -u USER
-k KEY`, `apply -u USER --private-key KEY`) or with `-e` (`-e ansible_user=deploy`).
Variables can also come from `group_vars/all.yml` and `host_vars/<address>.yml` next to
the file.

For anything else use INI or YAML:

```ini
[dev]
server-01 ansible_host=192.168.1.10
server-02 ansible_host=192.168.1.11 ansible_port=2222 ansible_user=deploy

[dev:vars]
ansible_ssh_private_key_file=~/.ssh/dev_key
```

Host key settings per format: [README_insecure_ignore_host_key.md](./README_insecure_ignore_host_key.md).

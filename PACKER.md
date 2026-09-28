# Packer

Packer has no onigirazu provisioner; `shell-local` runs onigirazu from the
build machine against the VM being built, the way the `ansible` provisioner
runs ansible-playbook.

```hcl
provisioner "shell-local" {
  environment_vars = [
    "BUILD_PASSWORD=${build.Password}",
  ]
  inline_shebang = "/bin/bash -e"
  inline = [
    # requirements.yml: roles and git collections, as ansible-galaxy would
    "onigirazu galaxy install -r ${path.root}/ansible/requirements.yml",
    # the password goes through a 0600 file, not the command line
    "umask 077; ev=$(mktemp); trap 'rm -f $ev' EXIT",
    "python3 -c 'import json,os; p=os.environ[\"BUILD_PASSWORD\"]; print(json.dumps({\"ansible_password\": p, \"ansible_become_password\": p}))' > $ev",
    "ANSIBLE_CONFIG=${path.root}/ansible/ansible.cfg onigirazu apply ${path.root}/ansible/playbook.yml -i '${build.Host}:${build.Port},' -u ${build.User} -e @$ev -e '{\"ansible_ssh_common_args\": \"-o StrictHostKeyChecking=no\"}'",
  ]
}
```

- `-i 'host:port,'` is a host list, as in Ansible; `-u` sets the user.
- With a key instead of a password, pass `--private-key` (e.g. the file the
  builder's `ssh_private_key_file` names) and drop the extra-vars file.
- `ANSIBLE_CONFIG` is read for `roles_path` and `collections_path`.
- The `ansible` provisioner's `ansible_python_interpreter` and
  `ANSIBLE_REMOTE_TMP` are not needed: onigirazu runs no Python on the host and
  keeps its work files in `~/.onigirazu/tmp`.
- Windows builds (WinRM) stay on the `ansible` provisioner.

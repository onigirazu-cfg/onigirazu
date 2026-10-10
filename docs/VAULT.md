# Ansible Vault

Onigirazu reads Ansible Vault data (format 1.1 and 1.2, AES256) and writes it in the same
format, so files work with both tools.

## What is decrypted

- Whole encrypted files: playbooks, `vars_files`, `include_vars`, inventories,
  `group_vars`/`host_vars`, role `defaults`/`vars`/tasks, `-e @file`
- `!vault |` values inside any of these files
- `copy` and `template` source files (`decrypt: false` on `copy` copies them as they are)
- SOPS-encrypted YAML/JSON files are opened the same way, through the `sops` binary (not `copy`/`template` sources): see [Secrets — SOPS](BITWARDEN_INTEGRATION.md#sops)

## Passwords

```bash
onigirazu apply site.yml -i hosts.ini --vault-password-file ~/.vault_pass
onigirazu apply site.yml -i hosts.ini --ask-vault-pass            # or -J
onigirazu apply site.yml -i hosts.ini --vault-id prod@~/.vault_prod --vault-id dev@prompt
```

Sources, all combined as in Ansible: `--vault-id label@file|prompt`, `--vault-password-file`,
`--ask-vault-pass`, `ANSIBLE_VAULT_IDENTITY_LIST`, `ANSIBLE_VAULT_PASSWORD_FILE`, and
`vault_identity_list` / `vault_password_file` in the `[defaults]` of `ansible.cfg`
(`ANSIBLE_CONFIG`, `./ansible.cfg`, `~/.ansible.cfg`, `/etc/ansible/ansible.cfg`).

A password file is read as text (trailing newline removed); an executable one is run and its
output is the password. Passwords are asked for only when encrypted data is met. Data encrypted
with a vault id (format 1.2) tries the password with that label first.

## The vault command

```bash
onigirazu vault encrypt group_vars/prod/secrets.yml --vault-password-file ~/.vault_pass
onigirazu vault decrypt secrets.yml
onigirazu vault view secrets.yml
onigirazu vault encrypt_string --name db_password 's3cret'     # prints a !vault value
```

Encryption uses the first password given (`--vault-id prod@file` writes format 1.2 with the
label); without one it asks twice.

Values decrypted in a run are plain variables: mark tasks that show them with `no_log: true`.

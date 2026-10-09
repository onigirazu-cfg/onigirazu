# Secrets: Bitwarden, HashiCorp Vault and SOPS

Templates, task arguments and variables read secrets when they are rendered, on the control
machine. Nothing is written to disk; a value is reused within a run for `secrets.cache_ttl`.
Mark tasks that show secrets with `no_log: true`.

## Bitwarden (and Vaultwarden)

Onigirazu uses the `bw` CLI and the session of an unlocked vault:

```bash
bw config server https://vault.example.com   # self-hosted Vaultwarden, once
bw login
export BW_SESSION=$(bw unlock --raw)
onigirazu apply site.yml -i hosts.ini
```

```yaml
db_password: "{{ bitwarden('app-db') }}"                 # the password
db_user: "{{ bitwarden('app-db', 'username') }}"
db_port: "{{ bitwarden('app-db', 'port') }}"             # a custom field
# as Ansible's community.general.bitwarden lookup
api_key: "{{ lookup('community.general.bitwarden', 'api', field='key') }}"
```

The item is a name or an id. Fields: `password` (`pass`), `username` (`user`), `totp` (the stored
TOTP secret, not a code), `notes` (`note`), or the name of a custom field (case-insensitive). The session reaches `bw` in its environment, never on a command line, and the
vault stays unlocked after the run.

## HashiCorp Vault

Secrets are read from a KV version 2 engine. The token comes from `VAULT_TOKEN` or
`~/.vault-token` (`vault login`); the address from `secrets.vault.address` or `VAULT_ADDR`, the
namespace from `secrets.vault.namespace` or `VAULT_NAMESPACE`. `VAULT_SKIP_VERIFY=true` skips the
token check at start-up. The `secrets:` block of `onigirazu.yml` is read by `apply`, `plan`, `drift`,
`verify` and `pull`; other commands use the environment only.

```yaml
# onigirazu.yml
secrets:
  cache_ttl: 5m
  vault:
    address: https://vault.example.com:8200
    mount: secret        # the KV v2 engine, default "secret"
    namespace: team-a    # optional
```

```yaml
db_password: "{{ vault('app/db', 'password') }}"
all_of_it: "{{ vault('app/db') }}"          # the whole secret as JSON
# as Ansible's community.hashi_vault lookups: `<mount>/data/<path>` or a path inside secrets.vault.mount
db_password: "{{ lookup('community.hashi_vault.hashi_vault', 'secret=secret/data/app/db:password') }}"
db: "{{ lookup('community.hashi_vault.vault_kv2_get', 'app/db').secret }}"   # a dict
```

The lookups' `url`, `token`, `namespace`, `auth_method` and `engine_mount_point` options are accepted
and ignored: the server, the mount and the credentials always come from the configuration and the
environment.

Without a token, an AppRole logs in: `VAULT_ROLE_ID` (or `secrets.vault.role_id`) and `VAULT_SECRET_ID`
(environment only, never in a file); `secrets.vault.auth_mount` names the auth method's mount (default
`approle`). The login's token is used for the run.

## Any provider

`secret(provider, item, field)`: `secret('bitwarden', 'app-db', 'username')`,
`secret('vault', 'app/db', 'password')`.

A provider is set up the first time a template uses it: runs without secrets need neither
`bw` nor Vault. Errors say what is missing (a locked vault, no token, an unknown field).

## SOPS

A YAML or JSON file encrypted with [SOPS](https://github.com/getsops/sops) (age, PGP, KMS; not the
dotenv, INI or binary formats) is opened wherever a whole Ansible Vault file is accepted (see
[VAULT.md](VAULT.md)): playbooks, `vars_files`, inventories, `group_vars/` and `host_vars/`, role
files, `-e @file`, `include_vars`, `community.sops.load_vars`. The `sops` binary does the decryption
with the keys it finds (`SOPS_AGE_KEY_FILE`, the PGP keyring, cloud credentials); a file is decrypted
once per run.

```yaml
- hosts: db
  vars_files: [vars/secrets.sops.yml]
  tasks:
    - community.sops.load_vars: {file: vars/more.sops.yml}
    - debug: {msg: "{{ lookup('community.sops.sops', 'files/token.sops.yml') }}"}   # the plain text
```

`lookup('community.sops.sops', file, rstrip=False)` keeps the trailing newline (`base64`,
`input_type` and `output_type` are accepted and ignored); a file that is not SOPS-encrypted fails the
lookup. Without `sops` installed the run fails with "the file is encrypted with SOPS but sops is not
installed"; without a key for the file, with sops's own message.

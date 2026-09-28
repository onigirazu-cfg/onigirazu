# Secrets: Bitwarden and HashiCorp Vault

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

The item is a name or an id. Fields: `password`, `username`, `totp`, `notes`, or the name of a
custom field. The session reaches `bw` in its environment, never on a command line, and the
vault stays unlocked after the run.

## HashiCorp Vault

Secrets are read from a KV version 2 engine. The token comes from `VAULT_TOKEN` or
`~/.vault-token` (`vault login`); the address from `secrets.vault.address` or `VAULT_ADDR`.

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
```

## Any provider

`secret(provider, item, field)`: `secret('bitwarden', 'app-db', 'username')`,
`secret('vault', 'app/db', 'password')`.

A provider is set up the first time a template uses it: runs without secrets need neither
`bw` nor Vault. Errors say what is missing (a locked vault, no token, an unknown field).

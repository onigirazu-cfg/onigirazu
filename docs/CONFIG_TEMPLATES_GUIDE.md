# Configuration Templates

Four example `onigirazu.yml` files ship with Onigirazu. They list many keys, but most of them
have no effect and produce a warning at start. The keys that do take effect are described in
[CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md).

## Where they are

| Template | Repository / release archive | Linux package |
|----------|------------------------------|---------------|
| default | `examples/onigirazu.default.yml` | `/usr/share/onigirazu/onigirazu.default.yml` |
| minimal | `examples/onigirazu.minimal.yml` | `/usr/share/onigirazu/examples/onigirazu.minimal.yml` |
| production | `examples/onigirazu.production.yml` | `/usr/share/onigirazu/examples/onigirazu.production.yml` |
| docker | `examples/onigirazu.docker.yml` | `/usr/share/onigirazu/examples/onigirazu.docker.yml` |

On first install the Linux package copies the default template to `/etc/onigirazu/onigirazu.yml`,
so it applies to every `apply` that has no `-c` and no `onigirazu.yml` next to the playbook.
The container image contains no templates.

## What each template changes

Only keys that take effect are listed.

`onigirazu.default.yml` (installed as `/etc/onigirazu/onigirazu.yml` by the packages) sets
nothing: every key is commented out, so the built-in defaults apply.

| Key | minimal | production | docker |
|-----|---------|------------|--------|
| `max_concurrency` | 10 | 5 | 10 |
| `log_level` | info | info | info |
| `log_format` | - | json | json |
| `output_format` | - | json | json |
| `show_diff` | - | true | true |
| `color_output` | - | true | false |
| `ssh_timeout` | - | 60s | 30s |
| `ssh_strict_host_key` | - | true | true |
| `ssh_known_hosts_file` | - | `~/.ssh/known_hosts` | `~/.ssh/known_hosts` |
| `remote_server` | - | - | - |
| `enable_metrics` | - | true (port 9090) | false |

Things to know before using one:

- `default_timeout` is commented out in every template: it limits the whole `apply` run, not
  a single task (as `apply -t`).
- `output_format: json` (production, docker) prints the result as JSON on stdout, as `apply -o json`.
- `retry_attempts`, `retry_delay`, `allow_shell_commands`, `blocked_commands`, `enable_caching`,
  `vault_*`, `progress_bar` and the other keys listed under "accepted but have no effect" in the
  reference do nothing. Restrict commands and paths with a
  [security policy](SECURITY_POLICY_GUIDE.md); use `secrets.vault` for Vault.

## Using a template

```bash
# one run
onigirazu apply site.yml -i inventory.yml -c /usr/share/onigirazu/examples/onigirazu.production.yml

# for a project: place it next to the playbook
cp examples/onigirazu.minimal.yml ./onigirazu.yml
```

A short file with only the keys you need is usually clearer than a copied template; see the
example in [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md#example).

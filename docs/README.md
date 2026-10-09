# Onigirazu Documentation

Start with the [Quick Start](QUICK_START_CONFIGURATION.md). The project
[README](../README.md) lists all commands.

## Running playbooks

- [Quick Start](QUICK_START_CONFIGURATION.md) — inventory, playbook, `plan` and `apply`, common flags
- [Plan, drift, diff and rollback](DRIFT_AND_ROLLBACK.md) — preview changes, detect and fix drift, undo a run
- [Import](IMPORT.md) — write a playbook from running hosts
- [Managed state](MANAGED_STATE.md) — resources each playbook manages; orphans of removed tasks
- [Tag filtering](TAG_FILTERING.md) — `--tags` and `--skip-tags`
- [Listing tags and tasks](LIST_TAGS_TASKS_GUIDE.md) — `--list-tags` and `--list-tasks`
- [Interactive mode](INTERACTIVE_MODE.md) — `apply --interactive`
- [Ad-hoc commands](ADHOC_GUIDE.md) — `onigirazu run` without a playbook
- [Verify](VERIFY.md) — `verify:` checks of a play and `onigirazu verify`
- [Safe apply](SAFE_APPLY.md) — `serial`, canary batches, health checks, automatic rollback, `strategy: free`, `throttle`
- [Pull mode](PULL.md) — `onigirazu pull`: a host converges itself from git, on a timer, with notifications and metrics
- [Plan in a pull request](PLAN_IN_PR.md) — `plan --github-comment` as a sticky PR comment
- [Machine-readable results](MACHINE_OUTPUT.md) — `-o json|yaml`, exit codes

## Inventory

- [Inventory formats](INVENTORY_FORMATS.md) — YAML, JSON, TOML, INI, host lists, dynamic scripts, `group_vars`/`host_vars`
- [Ansible inventories](ANSIBLE_INVENTORY_QUICK_START.md) — using existing Ansible INI and YAML inventories
- [Inventory plugins](INVENTORY_PLUGINS.md) — hosts from NetBox, vSphere, Proxmox and NetBird

## Writing playbooks

- [Playbook examples](examples/README.md)
- [Variables cheat sheet](VARIABLES_CHEATSHEET.md)
- [Loops](LOOPS_GUIDE.md)
- [Handlers](HANDLERS_GUIDE.md) and [handler examples](HANDLERS_EXAMPLES.md)
- [run_once, delegate_to, local_action](RUN_ONCE_DELEGATE_TO_GUIDE.md)
- [Filters](FILTERS_GUIDE.md) — filters, tests and lookups in expressions

## Modules

- [Module reference](modules/README.md)
- [systemd, cron and firewall modules](MODULES_SYSTEMD_CRON_FIREWALL.md)
- Windows hosts: the `win_*` modules in the [module reference](modules/README.md), over WinRM or OpenSSH ([inventory variables](INVENTORY_FORMATS.md#connection-variables))
- [Ansible bridge](ANSIBLE_BRIDGE.md) — modules Onigirazu lacks run through ansible-core

## Configuration and security

- [Configuration reference](CONFIGURATION_REFERENCE.md) — `onigirazu.yml` and `ONIGIRAZU_*` variables
- [Ansible Vault](VAULT.md) — encrypted files and `!vault` values, passwords, `onigirazu vault`
- [Secrets](BITWARDEN_INTEGRATION.md) — Bitwarden/Vaultwarden, HashiCorp Vault (token or AppRole), SOPS-encrypted vars files; the Ansible lookups
- [Security policy](SECURITY_POLICY_GUIDE.md) — restricting hosts, modules, paths and commands
- [Troubleshooting](TROUBLESHOOTING_CONFIG.md)

## Extending Onigirazu

- [Plugins](PLUGIN_INTEGRATION.md) — filter, module and callback plugins
- [Callback plugins](CALLBACKS_GUIDE.md)
- [Module development](MODULE_DEVELOPMENT_GUIDE.md) and [module scaffolding](MODULE_SCAFFOLDING_GUIDE.md)
- [Testing roles](TESTING_ROLES.md) — `onigirazu test` runs Molecule scenarios

## Project

- [Supported platforms](PLATFORMS.md)
- [Packer](PACKER.md) — onigirazu as the Packer provisioner
- [CI/CD](ci-cd.md)
- [Release process](RELEASE_PROCESS.md)

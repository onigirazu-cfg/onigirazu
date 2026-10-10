# Onigirazu

[![Release](https://img.shields.io/github/v/release/onigirazu-cfg/onigirazu)](https://github.com/onigirazu-cfg/onigirazu/releases/latest)

Onigirazu is a configuration management tool in a single Go binary. It runs Ansible playbooks,
roles and inventories over SSH (and WinRM for Windows), with nothing to install on the hosts
beforehand (no Python; a small agent is uploaded on the first connection, with a shell fallback), and
adds what Ansible leaves to other tools: a plan before every change, drift checks, rollback of a
run, Terraform-like managed state, canary rollouts with health checks, and import of running hosts
into a playbook.

- **Ansible-compatible**: playbooks, roles, collections from git, `requirements.yml`,
  `ansible.cfg`, INI/YAML/JSON inventories, dynamic inventory scripts, `group_vars`/`host_vars`,
  Jinja expressions and filters, facts and magic variables, FQCN module names
- **80+ built-in modules** in Go: files, packages, services, users, containers, databases,
  firewall, HTTP and more
- **See before you change**: `plan` shows per host what `apply` would change, with file diffs
- **Undo**: every run keeps a snapshot; `rollback` restores files, packages, services and accounts
- **Managed state**: a task removed from the playbook has its resources cleaned up on the next run
- **Safe rollouts**: `serial` batches, `--canary`, health checks, automatic rollback of an
  unhealthy batch
- **Drift detection**: scheduled checks with history, HTML reports and webhook notifications
- **Import**: turns a running host into a playbook that recreates it
- **Fast**: one SSH connection and one shell per host, commands framed over it
  (see [Performance](#performance))
- **Terminal dashboard**, ad-hoc commands (including plain English), JSON/YAML output for scripts

## Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Coming from Ansible](#coming-from-ansible)
- [Playbooks](#playbooks)
- [Seeing and controlling changes](#seeing-and-controlling-changes)
- [Modules](#modules)
- [Command reference](#command-reference)
- [Ad-hoc commands](#ad-hoc-commands)
- [Interactive mode](#interactive-mode)
- [Automation and CI](#automation-and-ci)
- [Configuration and security](#configuration-and-security)
- [Performance](#performance)
- [Documentation](#documentation)
- [Development](#development)

## Installation

### Pre-built binaries

```bash
# Linux x86_64 (also arm64, armv6, armv7, i386)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Apple Silicon; onigirazu_Darwin_x86_64.tar.gz for Intel)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

Every [release](https://github.com/onigirazu-cfg/onigirazu/releases) has archives for Linux,
macOS, Windows, FreeBSD, OpenBSD and NetBSD, and `.deb`, `.rpm`, `.apk` and Arch
(`.pkg.tar.zst`) packages.

### Container image

```bash
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest version
```

Images for `linux/amd64` and `linux/arm64`, tagged `latest`, `X.Y.Z`, `X.Y` and `X`.

### From source

```bash
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@latest
# or
git clone https://github.com/onigirazu-cfg/onigirazu.git && cd onigirazu && make build   # bin/onigirazu
```

See [INSTALLATION.md](INSTALLATION.md) for details.

## Quick start

An inventory, in Ansible's format:

```ini
# hosts.ini
[web]
web1 ansible_host=192.168.1.10
web2 ansible_host=192.168.1.11

[web:vars]
ansible_user=ubuntu
```

A playbook:

```yaml
# site.yml
- name: Web servers
  hosts: web
  become: true
  tasks:
    - name: Nginx installed
      ansible.builtin.package:
        name: nginx
        state: present

    - name: Site configuration
      ansible.builtin.template:
        src: site.conf.j2
        dest: /etc/nginx/conf.d/site.conf
      notify: Reload nginx

    - name: Nginx running
      ansible.builtin.service:
        name: nginx
        state: started
        enabled: true

  handlers:
    - name: Reload nginx
      ansible.builtin.service:
        name: nginx
        state: reloaded
```

See what it would change, then apply it:

```bash
onigirazu plan site.yml -i hosts.ini
onigirazu apply site.yml -i hosts.ini
```

A host list works without an inventory file: `-i "ubuntu@web1,ubuntu@web2:2222"`.

Onigirazu also has its own playbook format with a top-level `plays:` list and `vars:`; both
formats can be used side by side. Examples: [docs/examples](docs/examples/README.md).

## Coming from Ansible

Existing Ansible content runs as it is in most cases. What is supported:

YAML is read as Ansible reads it: unquoted `yes`/`no`/`on`/`off` values are booleans (YAML 1.1); quote them to keep text.

- **Playbooks**: plays with `pre_tasks`, `tasks`, `post_tasks`, `handlers`, `roles`,
  `vars`, `vars_files`, `environment`, `gather_facts`, `serial`, `strategy` (`linear`, `free`),
  `throttle`, `max_fail_percentage`, `any_errors_fatal`, `force_handlers`, `verify:` checks
  ([docs/VERIFY.md](docs/VERIFY.md)); `import_playbook`, `include_tasks`/`import_tasks` (with `vars`, `when`,
  `tags` and `loop`; a loop runs each included task over the items in turn, not the whole file per
  item), `include_role`/`import_role`, `block`/`rescue`/`always`
- **Task keywords**: `when`, `loop` and `with_*` (items, list, dict, sequence, nested,
  together, subelements, indexed_items, file, fileglob, first_found, lines, ...), `loop_control`, `register`, `until`/`retries`/
  `delay`, `changed_when`, `failed_when`, `ignore_errors`, `notify`/`listen`, `tags`,
  `become`/`become_user`, `delegate_to`, `local_action`, `run_once`, `throttle`, `no_log`,
  `check_mode`, `diff`, `vars`, `environment`, `action` (also with a templated module name),
  `async` with `poll` (the task fails after `async` seconds); `poll: 0` starts it in the
  background and `async_status` reports on it (jobs live in the onigirazu process, which waits
  for unfinished ones at the end of the run)
- **Short forms**: `command: make install chdir=/src`, `file: path=/etc/app state=directory`,
  `args:`; argument aliases such as `apt: pkg:`, `file: dest:`, `systemd: unit:`
- **Module names**: `ansible.builtin.*`, `ansible.legacy.*` and the collection modules
  Onigirazu implements (`ansible.posix.sysctl`/`mount`/`authorized_key`,
  `community.general.ufw`/`ini_file`/`timezone`/`archive`,
  `community.docker.docker_container`/`docker_image`/`docker_compose`/`docker_compose_v2`/`docker_host_info`,
  `community.mysql.mysql_db`/`mysql_user`, `community.postgresql.postgresql_db`/`postgresql_user`,
  `community.sops.load_vars`, `ansible.windows.*`, `community.windows.*`,
  `chocolatey.chocolatey.win_chocolatey`); `dnf`/`dnf5` run the `yum` module
- **Roles and collections**: `roles/` next to the playbook, `roles_path` and `collections_path`
  from `ansible.cfg`, `ANSIBLE_ROLES_PATH`, `ANSIBLE_COLLECTIONS_PATH`,
  `namespace.collection.role`; `onigirazu galaxy install -r requirements.yml` installs roles
  (git or Ansible Galaxy) and git collections
- **Inventories**: INI and YAML, JSON, executable scripts (`--list`), directories, host lists,
  `group_vars`/`host_vars` next to the inventory and the playbook, `ANSIBLE_INVENTORY`;
  `onigirazu inventory --list --json` prints what `ansible-inventory --list` prints
- **Connection variables**: `ansible_host`, `ansible_port`, `ansible_user`,
  `ansible_password`, `ansible_ssh_private_key_file`, `ansible_become_password`,
  `ansible_connection: winrm` with `ansible_winrm_*` (NTLM, message encryption, https),
  `ansible_ssh_common_args`/`ansible_ssh_extra_args` (`ConnectTimeout`, `ProxyJump`/`-J`,
  `ProxyCommand`, `StrictHostKeyChecking=no`), `ansible_connection` `local`, `docker`, `podman`;
  templated values and `-e` overrides
- **Jinja**: filters (`default`, `map`, `select`/`selectattr`, `combine`, `regex_*`,
  `to_json`/`from_yaml`, `ternary`, set operations, `password_hash`, ...), tests (`is defined`,
  `is version`, `is success`/`failed`/`changed`/`skipped`, `is match`/`search`, ...), `~`,
  inline `if`, `omit`, Python string/dict/list methods (`.split()`, `.get()`), lookups
  (`env`, `file`, `pipe`, `template`, `fileglob`, `first_found`, `community.general.bitwarden`,
  ...), `{% set %}`
- **Facts and variables**: `ansible_facts` and the `ansible_*` names (distribution, OS family,
  hostname, IP, memory, virtualization, `ansible_pkg_mgr`, `ansible_local` from
  `/etc/ansible/facts.d`), `setup`, `hostvars`, `groups`, `group_names`, `inventory_hostname`,
  `ansible_check_mode`, `ansible_play_hosts`, `ansible_limit`; templated variables are
  rendered per host when used, as in Ansible
- **Command line**: `-i`, `-e` (also `@file`), `--limit`, `--tags`/`--skip-tags`, `--check`,
  `--diff`, `-b`/`--become-user`, `-u`, `--private-key`, `--start-at-task`, `--list-hosts`,
  `--list-tasks`, `--list-tags`, `--syntax-check`
- **Ansible Vault**: encrypted files and `!vault` values anywhere variables are read, `copy` and
  `template` sources; `--vault-password-file`, `--vault-id`, `--ask-vault-pass`, the
  `ANSIBLE_VAULT_*` variables and `ansible.cfg`; `onigirazu vault encrypt|decrypt|view|encrypt_string`.
  See [docs/VAULT.md](docs/VAULT.md)

Windows hosts are managed with the `win_*` modules over WinRM (`ansible_connection: winrm`) or
OpenSSH (`ansible_shell_type: powershell` or `cmd`; the agent runs there too).
Not supported: Python modules and plugins from collections (Onigirazu has its own modules;
`validate` names any module it lacks). A task argument its module does not have fails the task,
as in Ansible.
For Packer builds see [docs/PACKER.md](docs/PACKER.md).

## Playbooks

### Variables and templates

```yaml
- hosts: web
  vars:
    app_dir: "/opt/{{ app_name }}"
    backends: "{{ groups['app'] }}"
  tasks:
    - template:
        src: haproxy.cfg.j2
        dest: /etc/haproxy/haproxy.cfg
```

```jinja
{% for h in backends %}
server {{ h }} {{ hostvars[h].ansible_default_ipv4.address }}:{{ app_port | default(8080) }} check
{% endfor %}
```

An undefined variable fails the task; `default()` and `is defined` handle optional ones. An
argument that is exactly `{{ var }}` keeps a list or map as it is. `-e` (`key=value`, JSON/YAML
or `@file`) overrides everything. Precedence follows Ansible, from role defaults (lowest) to
`-e` (highest).

Guides: [variables](docs/VARIABLES_CHEATSHEET.md), [filters, tests and lookups](docs/FILTERS_GUIDE.md).

### Conditions, loops and retries

```yaml
- name: Wait for the app
  uri:
    url: http://127.0.0.1:8080/health
  register: health
  until: health.status == 200
  retries: 10
  delay: 3

- name: Packages of this OS
  package:
    name: "{{ item }}"
  loop: "{{ packages[ansible_os_family] }}"
  when: ansible_os_family in packages
```

`loop` takes a list or an expression. The `with_*` forms work as in Ansible: `with_items`,
`with_list`, `with_dict`, `with_sequence`, and `with_<lookup>` for `nested`, `together`,
`subelements`, `indexed_items`, `random_choice`, `file`, `fileglob`, `first_found`, `lines`,
`pipe`, `env`, `template`; an unknown `with_*` is an error. `loop_control` takes `loop_var`
and `index_var` (`label` is accepted and ignored). See [docs/LOOPS_GUIDE.md](docs/LOOPS_GUIDE.md).

### Roles, includes and imports

```yaml
- import_playbook: base.yml

- hosts: web
  roles:
    - common
    - role: web
      web_port: 9090
  tasks:
    - include_role:
        name: web
        tasks_from: upgrade
      when: upgrade | bool
```

A role's `tasks/`, `handlers/`, `defaults/`, `vars/`, `templates/`, `files/` and
`meta/main.yml` dependencies are used as in Ansible.

### Blocks, handlers and failures

`block` with `rescue` (`ansible_failed_task`, `ansible_failed_result`) and `always`. Handlers run
once per host at the end of each section and at `meta: flush_handlers`; `meta: end_host` and
`meta: end_play` stop hosts. A host whose task fails leaves the run and the others go on; the play
stops with `any_errors_fatal`, above `max_fail_percentage`, or when no host is left. See
[docs/HANDLERS_GUIDE.md](docs/HANDLERS_GUIDE.md).

### Rollouts

```yaml
- hosts: web
  serial: [1, "30%"]          # batches: a number, a percentage, or a list
  health_check:               # after every batch
    - uri: {url: "http://127.0.0.1:8080/health"}
      retries: 10
      delay: 3
  tasks:
    - name: Upload to the shared service, two hosts at a time
      command: ./upload.sh
      throttle: 2
```

`strategy: free` lets every host run the play's tasks at its own pace instead of waiting for the
slowest host at every task (handlers still run at the end, for the hosts that notified them); a
play-level `throttle` applies to every task without its own. An unhealthy batch is rolled back and
the rollout stops (exit code 5). `--canary 1 --canary-pause
5m` runs one host first and checks it again after a soak; `--auto-rollback` rolls back a batch
whose tasks fail. See [docs/SAFE_APPLY.md](docs/SAFE_APPLY.md).

### Check mode and diffs

`--check` runs every module that can predict its result without changing anything
(files, packages, services, users, containers, ...); the others are skipped with a note.
`--diff` shows unified diffs of files. `check_mode: false` on a task runs it anyway (a probe
later tasks need).

## Seeing and controlling changes

### Plan

```bash
onigirazu plan site.yml -i hosts.ini                 # per host: what would change, with diffs
onigirazu plan site.yml -i hosts.ini --format html --output plan.html
```

`plan` runs the playbook in check mode and also lists resources that left the playbook (see
Managed state). Formats: text, json, html, markdown; `--github-comment` posts the plan on the pull
request of the workflow run ([docs/PLAN_IN_PR.md](docs/PLAN_IN_PR.md)).

### Drift

```bash
onigirazu drift site.yml -i hosts.ini                 # report; exit code 2 when a host drifted
onigirazu drift site.yml -i hosts.ini --fix           # apply when drift is found
onigirazu drift site.yml -i hosts.ini --notify "$WEBHOOK" --format html --output drift.html
onigirazu drift site.yml --history                    # past checks
```

```
Drift: 1 task(s) on 1 of 3 host(s) differ from site.yml

web1
  ~ Config file (copy)
      --- before: /etc/app.conf
      +++ after: /etc/app.conf
      -port = 8081
      +port = 8080

In sync: web2, db1
```

Every check is kept, so a drifting task shows since when it drifts. Exit codes: 0 in sync,
2 drift, 1 a task could not be checked. `--notify` posts to Slack or
Mattermost style webhooks; `--metrics-file` (node_exporter textfile collector), `--metrics-push URL`
(VictoriaMetrics import, Pushgateway) and `--metrics-label k=v` export the result as Prometheus
metrics. A systemd timer or CI schedule running `drift` watches a fleet.

### Rollback

Every `apply` keeps a snapshot of what it changed: file contents (text up to 1 MiB), modes and
owners, packages installed or removed, service states, users and groups it created.

```bash
onigirazu rollback --list
onigirazu rollback --last --dry-run                   # what would be restored
onigirazu rollback --snapshot <id> -i hosts.ini
onigirazu rollback --cleanup --max-age 30d
```

What cannot be undone (upgrades, changes to existing accounts, large or binary files) is listed.
See [docs/DRIFT_AND_ROLLBACK.md](docs/DRIFT_AND_ROLLBACK.md).

### Managed state

`apply` records the files, packages, services, users and groups each task manages, in
`.onigirazu/<playbook>.state.json` or an S3 bucket shared by the team, with a lock against
concurrent runs. When a task leaves the playbook, `plan` lists its resources and the next `apply`
cleans up like `terraform apply`: what Onigirazu created is removed, what it took over is put back
as it was. `--no-destroy` keeps them; `prevent_destroy: true` on a task only forgets them.

```bash
onigirazu state resources site.yml       # what the playbook manages on each host
onigirazu state rm site.yml web1 file /etc/app.conf   # forget one; the host is not touched
onigirazu state unlock site.yml <lock-id>            # the lock of a run that is gone
```

See [docs/MANAGED_STATE.md](docs/MANAGED_STATE.md).

### Import

```bash
onigirazu import web1 -i hosts.ini -o imported/ --baseline fresh1
```

Writes a playbook (roles shared between hosts, templates, `group_vars`) that recreates what the
host has beyond a fresh install: packages installed by hand, services, accounts, configuration
files, repositories. It then plans the playbook against the host; a faithful import has nothing
to change. Secrets become variables with an example file. See [docs/IMPORT.md](docs/IMPORT.md).

## Modules

| Area | Modules |
|------|---------|
| Commands | `command`, `shell`, `script` |
| Files | `file`, `copy`, `template`, `lineinfile`, `blockinfile`, `replace`, `ini_file`, `fetch`, `slurp`, `stat`, `find`, `archive`, `unarchive`, `get_url`, `config` (JSON/YAML/TOML keys) |
| Packages | `package`, `apt`, `yum` (also `dnf`), `apt_repository`, `apt_key`, `pip` |
| Services and system | `service`, `systemd`, `cron`, `sysctl`, `mount`, `hostname`, `timezone`, `reboot`, `user`, `group`, `authorized_key`, `getent` |
| Network and firewall | `uri`, `wait_for`, `firewall` (ufw, firewalld, iptables), `ufw` |
| Containers | `docker_container`, `docker_image`, `docker_compose` (v1 and v2), `docker_host_info`, `podman` |
| Databases | `mysql_db`, `mysql_user`, `postgresql_db`, `postgresql_user`, `mongodb` |
| Source control | `git` |
| Windows (WinRM or SSH) | `win_ping`, `win_command`, `win_shell`, `win_powershell`, `win_regedit`, `win_file`, `win_copy`, `win_service`, `win_timezone`, `win_firewall_rule`, `win_firewall`, `win_group_membership`, `win_feature`, `win_reboot`, `win_scheduled_task`, `win_chocolatey`, `win_optional_feature`, `win_disk_facts`, `win_initialize_disk`, `win_partition`, `win_format`; facts (`ansible_os_family: Windows`, ...) |
| Flow and data | `debug`, `assert`, `fail`, `set_fact`, `include_vars`, `add_host`, `group_by`, `setup`/`gather_facts`, `pause`, `ping`, `meta`, `async_status`, `include_role`/`import_role`, `verify` |

Any other Ansible module (a collection module, or a `win_*` module Onigirazu does not implement) runs through an installed
ansible-core when `ansible_bridge` in `onigirazu.yml` allows it: see
[docs/ANSIBLE_BRIDGE.md](docs/ANSIBLE_BRIDGE.md).

Modules compare the host with the task first and report `changed` only when they changed
something (`command` and `shell` always do, unless `changed_when`, `creates` or `removes` say
otherwise). Arguments,
return values and examples: [docs/modules/README.md](docs/modules/README.md). New modules:
[docs/MODULE_DEVELOPMENT_GUIDE.md](docs/MODULE_DEVELOPMENT_GUIDE.md) and the generator in
`scripts/module_scaffold`.

## Command reference

| Command | Purpose |
|---------|---------|
| `apply PLAYBOOK` | Run a playbook |
| `plan PLAYBOOK` | Show what `apply` would change |
| `drift PLAYBOOK` | Check that hosts still match a playbook; `--fix` applies |
| `verify PLAYBOOK` | Run the plays' `verify:` checks (files, packages, services, ports, http, ...) and report them ([docs/VERIFY.md](docs/VERIFY.md)) |
| `comply --profile PROFILE` | Check hosts against a compliance profile (bundled `linux-baseline`, `ssh`, or your own CIS-style controls) and score them; `comply list`, `comply show` ([docs/COMPLY.md](docs/COMPLY.md)) |
| `pull --repo URL` | This host converges itself from a git repository: once, every `--interval`, or `pull install` (systemd timer); `--drift-only`, `--only-on-change`, `--notify`, `--metrics-*` ([pull mode](docs/PULL.md)) |
| `serve -f serve.yml` | Fleet server: scheduled drift/apply/verify/comply jobs, results per host, run history, web page and REST API behind a token or a reverse proxy's identity ([docs/SERVE.md](docs/SERVE.md)) |
| `listen -f listen.yml` | Run playbooks on events: Alertmanager/vmalert alerts, GitHub webhooks, Mattermost commands, any JSON POST; `listen test`, `listen install` ([docs/LISTEN.md](docs/LISTEN.md)) |
| `plugin list` | Command plugins found (`onigirazu-NAME` executables) |
| `diff PLAYBOOK` | Compare a playbook with the last recorded run |
| `rollback` | List, inspect and restore snapshots of runs |
| `state` | Managed state: `resources`, `rm`, `unlock`; `list`, `show` |
| `import HOST...` | Write a playbook from running hosts |
| `run PATTERN ...` | Ad-hoc commands |
| `validate PLAYBOOK` | Parse the playbook and check every module name, roles included |
| `lint`, `fmt` | Best-practice checks (arguments of bridged modules against ansible-doc too); YAML formatting |
| `doc MODULE` | A module's arguments: built in, or through the Ansible bridge |
| `graph PLAYBOOK` | Plays, tasks, handlers and variables as ASCII, DOT or Mermaid |
| `inventory` | `--list`, `--host`, `--graph`, `--json` (as `ansible-inventory`); sources: YAML/JSON/TOML/INI, scripts, [plugins](docs/INVENTORY_FORMATS.md#inventory-plugins) (NetBox, vSphere, Proxmox, NetBird, Terraform/OpenTofu state) |
| `galaxy install -r FILE` | Install roles and collections from a requirements file |
| `vault` | `encrypt`, `decrypt`, `view`, `encrypt_string` of Ansible Vault data |
| `healthcheck` | Reachability, disk, memory, CPU and services of the inventory hosts |
| `audit` | History of runs: `list`, `show`, `host`, `stats`, `export`, `clear` |
| `list-executions`, `show-execution`, `show-last-execution` | Results of `apply --background` runs |
| `completion`, `version` | Shell completion; version |

Frequently used `apply` flags:

```bash
onigirazu apply site.yml -i hosts.ini \
  --limit 'web:!web3' --tags deploy -e version=1.4 -e @prod.yml \
  -b -u deploy --private-key ~/.ssh/deploy \
  --check --diff -f 20
```

`--list-hosts`, `--list-tasks`, `--list-tags` and `--syntax-check` print and exit;
`--start-at-task` skips to a task; `-o json|yaml` prints a machine-readable result.
Every command has `--help`. Global flags: `-i`, `-c` (config file), `-s` (state file), `-v`, `--show-debug`,
`--no-color`, `--security-policy`.

## Ad-hoc commands

```bash
onigirazu run web -m package -a "name=nginx state=present" -b -i hosts.ini   # Ansible style
onigirazu run web "install nginx" -i hosts.ini                             # plain English
onigirazu run web "package:name=nginx,state=present" -i hosts.ini          # module:args
onigirazu run web '{"module": "service", "args": {"name": "nginx", "state": "restarted"}}' -i hosts.ini
onigirazu run all -m ping -i "ubuntu@web1,ubuntu@web2" -o table
```

Output: text, json, yaml, table. See [docs/ADHOC_GUIDE.md](docs/ADHOC_GUIDE.md).

## Interactive mode

`apply --interactive` shows a terminal dashboard while the playbook runs: a live log, progress,
per-host counts and a browser for task results.

| Key | Action |
|-----|--------|
| **P** | Pause / resume (running tasks finish first) |
| **G** | Stop gracefully: no new task starts |
| **R** | Task results; **Enter** shows one (error, message, stdout, stderr, diff) |
| **L** | Timeline per host |
| **B** | Rollout batches: health, rollback, what was undone |
| **N** / **V** / **D** | Detail levels |
| **F**, **/** | Filter, search |
| **S**, **H** | Statistics, help |
| **X** | After the run: run again on the failed hosts |
| **Q**, **Ctrl+C** | Close; during a run it asks before stopping |

Without a terminal the normal output is used. See [docs/INTERACTIVE_MODE.md](docs/INTERACTIVE_MODE.md).

## Automation and CI

- `apply -o json` writes one document to stdout (status, totals, every task per host); logs go
  to stderr. Exit codes: 0 success, 1 failure, 5 a batch was rolled back, 130 interrupted.
- `drift` and `plan` have JSON, HTML and Markdown reports; `--github-comment` keeps the plan as one
  comment on the pull request ([docs/PLAN_IN_PR.md](docs/PLAN_IN_PR.md)); `drift --notify` posts to
  webhooks and `--metrics-*` exports Prometheus metrics.
- `pull` runs a playbook from git on the host itself, once or on a systemd timer, with the same
  notifications and metrics ([docs/PULL.md](docs/PULL.md)).
- `listen` runs playbooks on events — an alert, a push, a chat command — with per-rule throttling
  ([docs/LISTEN.md](docs/LISTEN.md)).
- `serve` is the fleet server: scheduled jobs, their results per host, the run history, a web page
  and an API with viewer/operator roles ([docs/SERVE.md](docs/SERVE.md)).
- `verify` runs the plays' `verify:` checks and reports them, also as JSON ([docs/VERIFY.md](docs/VERIFY.md)).
- `verify` runs the plays' `verify:` checks and reports them, also as JSON ([docs/VERIFY.md](docs/VERIFY.md));
  `comply` scores hosts against compliance profiles, with Markdown/HTML reports and metrics ([docs/COMPLY.md](docs/COMPLY.md)).
- `apply --background` returns at once; `show-execution` reads the result later.
- `audit` keeps the history of runs with per-host statistics.
- `onigirazu test` runs a role's Molecule scenarios (docker/podman instances, converge,
  idempotence, verify) without Molecule or Python: see [docs/TESTING_ROLES.md](docs/TESTING_ROLES.md).
- Command plugins: `onigirazu NAME` runs the executable `onigirazu-NAME` (from
  `~/.onigirazu/plugins`, `ONIGIRAZU_PLUGIN_PATH` or `PATH`); callback, filter and module plugins
  (Go plugins) hook into runs. See [docs/PLUGIN_INTEGRATION.md](docs/PLUGIN_INTEGRATION.md).

## Configuration and security

- Settings live in `onigirazu.yml` (path from `-c`, next to the playbook, or
  `/etc/onigirazu/onigirazu.yml`), overridable by `ONIGIRAZU_*` variables: parallelism,
  timeouts, `roles_path`, the managed state backend, logging. See
  [docs/CONFIGURATION_REFERENCE.md](docs/CONFIGURATION_REFERENCE.md).
- SSH host keys are checked against `~/.ssh/known_hosts`: a new host is added, a changed key
  fails the connection. Per host, `insecure_ignore_host_key` (or
  `ansible_ssh_common_args: -o StrictHostKeyChecking=no`) turns the check off. Jump hosts
  (`-J`/`ProxyJump`, aliases from `~/.ssh/config`) are checked the same way.
- Passwords (`ansible_password`, `ansible_become_password`) never appear on command lines;
  sudo reads them on stdin. `no_log: true` keeps a task's values out of logs, state and output.
- Secrets come from Bitwarden/Vaultwarden (`bw` CLI, `BW_SESSION`), HashiCorp Vault (token or
  AppRole) or SOPS-encrypted vars files when a template uses them: `{{ bitwarden('app-db') }}`,
  `{{ vault('app/db', 'password') }}`, `lookup('community.hashi_vault.hashi_vault', ...)`,
  `lookup('community.sops.sops', ...)`. See [docs/BITWARDEN_INTEGRATION.md](docs/BITWARDEN_INTEGRATION.md).
- An optional security policy restricts modules, hosts, paths and commands:
  [docs/SECURITY_POLICY_GUIDE.md](docs/SECURITY_POLICY_GUIDE.md).
- Work files on the hosts live in `~/.onigirazu/tmp` of the connecting user.

## Performance

One SSH connection per host and, on it, a small command server (a static Go agent uploaded
once; a Python or POSIX sh fallback) that runs commands, probes and writes files without a new
process or session per task; the file modules reuse one capture of the target. Measured with
[`bench/`](bench/README.md) (a ~40-module playbook, the same for both tools, on fresh VMs):

| 1 host, 10 hosts | converge | second (unchanged) run | check mode |
|---|---|---|---|
| Ansible (forks=10, pipelining, ControlPersist) | 168 s, 181 s | 167 s, 172 s | 168 s, 168 s |
| Onigirazu | 10.3 s, 13.8 s | 1.6 s, 2.0 s | 1.1 s, 1.3 s |
| control-side CPU, 10 hosts | 2.0 s vs 182 s | 1.4 s vs 163 s | 1.1 s vs 159 s |

Both tools leave the hosts in the same state (about 1100 goss checks per host pass after each run).
Without the agent (the POSIX sh server, no Python on the hosts) a second run takes 3.1 s and 4.0 s.

On 500 hosts (containers, `bench/scale.sh`) a short playbook converges in 7.3–7.8 s and a second run
takes 5.5–5.8 s with about 6 s of CPU and 230–285 MB on the control side, at concurrency 50 to 500.

## Documentation

All guides are listed in [docs/README.md](docs/README.md). Start with:

- [Quick start](docs/QUICK_START_CONFIGURATION.md) - inventory, playbook, plan, apply
- [Plan, drift, diff and rollback](docs/DRIFT_AND_ROLLBACK.md)
- [Safe apply](docs/SAFE_APPLY.md), [Managed state](docs/MANAGED_STATE.md), [Import](docs/IMPORT.md)
- [Modules](docs/modules/README.md), [Playbook examples](docs/examples/README.md)
- [Inventory formats](docs/INVENTORY_FORMATS.md), [Variables](docs/VARIABLES_CHEATSHEET.md),
  [Filters and lookups](docs/FILTERS_GUIDE.md), [Loops](docs/LOOPS_GUIDE.md),
  [Handlers](docs/HANDLERS_GUIDE.md), [Tags](docs/LIST_TAGS_TASKS_GUIDE.md)
- [Packer](docs/PACKER.md), [Configuration](docs/CONFIGURATION_REFERENCE.md),
  [Security policy](docs/SECURITY_POLICY_GUIDE.md)

## Development

```bash
make build          # bin/onigirazu
make test           # unit tests
make test-race      # with the race detector
make coverage       # coverage report
make lint           # golangci-lint
make run-example    # a playbook against localhost
```

End-to-end tests (`e2e/`) run every case playbook twice against fresh Ubuntu 24.04 and 26.04
virtual machines, check the result on the hosts and that the second run changes nothing; CI runs unit tests, lint, CodeQL and a coverage gate.

```
cmd/onigirazu/     entry point
pkg/types/         playbook, task and inventory types, Ansible short forms and aliases
internal/
  cli/             commands
  parser/          playbooks, roles, includes, imports
  engine/          plays, batches, handlers, loops, facts, variables
  expression/      Jinja expressions, filters and tests
  template/        template rendering
  modules/         built-in modules
  inventory/       inventory formats and variables
  ssh/, executor/  connections, the remote shell, become
  managed/         managed state (local and S3)
  rollback/, drift/, state/, audit/   snapshots, drift reports, run history
  importer/        import of running hosts
  galaxy/          requirements installation
e2e/               end-to-end cases
docs/              guides
```

Contributions: see [CONTRIBUTING.md](CONTRIBUTING.md). License: MIT, see [LICENSE](LICENSE).
Issues: [GitHub Issues](https://github.com/onigirazu-cfg/onigirazu/issues).

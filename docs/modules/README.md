# Core Modules Documentation

Reference for the built-in modules of Onigirazu: 63 module names, 61 modules
(`include_role`/`import_role` and `setup`/`gather_facts` are one module each under two names).
For a one-line summary of each module see the [Alphabetical Index](INDEX.md).

## Table of Contents

- [Module Overview](#module-overview)
- [System Modules](#system-modules)
- [File System Modules](#file-system-modules)
- [Configuration Modules](#configuration-modules)
- [Service Modules](#service-modules)
- [System Control Modules](#system-control-modules)
- [Package Modules](#package-modules)
- [Network Modules](#network-modules)
- [Security Modules](#security-modules)
- [System Connectivity](#system-connectivity)
- [Version Control](#version-control)
- [Scheduled Jobs](#scheduled-jobs)
- [Security & Firewall](#security--firewall)
- [Container Management](#container-management)
- [Database Management](#database-management)
- [Utility Modules](#utility-modules)
- [Complete Module List](#complete-module-list)

## Module Overview

Each task calls one module with its arguments. Arguments a module does not read are ignored.

### Module Structure

```yaml
- name: "Task Name"
  module_name:
    parameter1: value1
    parameter2: value2
  when: "condition"
  register: "variable_name"
```

### Task Keywords

Every task accepts these keywords next to its module:

| Keyword | Description |
|---------|-------------|
| `when` | Run only when the expression is true |
| `register` | Store the result under this name (also for failed or skipped tasks) |
| `loop`, `with_items`, `with_list`, `with_dict`, `with_sequence`, `loop_control` | Repeat the task |
| `until`, `retries`, `delay` | Repeat until the expression is true |
| `changed_when`, `failed_when` | Override the changed/failed status |
| `ignore_errors` | Continue on failure |
| `notify`, `listen` | Handlers |
| `become`, `become_user`, `become_method` | Privilege escalation |
| `delegate_to`, `run_once`, `local_action` | Where and how often the task runs |
| `block`, `rescue`, `always` | Task groups with error handling |
| `check_mode` | `false` runs the task for real in a check run |
| `environment`, `vars`, `tags`, `no_log`, `timeout` | Task environment, variables, tags, hidden output, time limit |

`retries` must be a number and `delay`/`timeout` a number of seconds or a duration such as `10s`. `retries` and `delay` can also be templates (`retries: "{{ n }}"`), rendered per host.

A registered result that has `stdout`/`stderr` also gets `stdout_lines`/`stderr_lines`.

### Short Forms

Ansible's short forms work:

```yaml
- command: echo hi chdir=/tmp       # free form; inline options: chdir, creates, removes, executable
- file: path=/tmp/x state=touch     # key=value pairs; \n, \t, \\ and \" in values are decoded
- ping:                             # no arguments
- command: make
  args:
    chdir: /src
- local_action: command hostname    # runs on the control machine (delegate_to: localhost)
```

Free-form modules: `command`, `shell` (the command), `script` (path and arguments), `meta` (the action) and `include_vars` (the file). `args:` adds to and overrides the short form.

### Argument Values

- A YAML integer given as `mode` is octal, as in Ansible: `mode: 0644` and `mode: "0644"` are the same.
- Other numbers are passed to modules as strings, so `minute: 0` works in cron.
- Booleans accept `true`/`false`, `yes`/`no`, `on`/`off` and `1`/`0`, also as templated strings.

### Module Names and Argument Aliases

- `ansible.builtin.*` and `ansible.legacy.*` names run the built-in module; `dnf` and `dnf5` run `yum`.
- Collection names that map to built-in modules: `ansible.posix.sysctl`, `ansible.posix.mount`, `ansible.posix.authorized_key`, `community.general.archive`, `community.general.ufw`, `community.general.ini_file`, `community.general.timezone`, `community.docker.docker_container`, `community.docker.docker_image`, `community.docker.docker_compose`, `community.docker.docker_compose_v2`, `community.docker.docker_host_info`, `community.mysql.mysql_db`, `community.mysql.mysql_user`, `community.postgresql.postgresql_db`, `community.postgresql.postgresql_user`.

Argument aliases (an argument given under both names keeps its own value):

| Module | Alias -> argument |
|--------|-------------------|
| apt | `pkg`, `package` -> `name`; `update-cache` -> `update_cache` |
| yum (dnf), package, pip | `pkg` -> `name` |
| file, stat | `dest`, `name` -> `path` |
| lineinfile, blockinfile, replace | `dest`, `destfile`, `name` -> `path` |
| ini_file | `dest` -> `path` |
| mount, find | `name` -> `path` |
| user | `user` -> `name` |
| service | `service` -> `name` |
| systemd | `service`, `unit` -> `name`; `daemon-reload` -> `daemon_reload` |
| sysctl | `key` -> `name`, `val` -> `value` |
| git | `name` -> `repo` |

Module-specific alternatives are listed with each module.

### Check Mode

With `--check` only these modules run, reporting what they would change and changing nothing:
ping, debug, set_fact, stat, find, fail, wait_for, assert, include_vars, slurp, getent, setup, gather_facts, docker_host_info,
file, copy, template, lineinfile, blockinfile, replace, ini_file, config, apt, yum, package, pip, apt_repository, apt_key,
service, systemd, user, group, cron, sysctl, mount, timezone, hostname, get_url, git, unarchive, ufw,
docker_container, docker_image, podman.

Every other module (command, shell, script, fetch, archive, reboot, uri, firewall, authorized_key, pause,
docker_compose and the database modules) is skipped.

## System Modules

### Facts

Facts are gathered at the start of every play unless the play sets `gather_facts: false`; the [setup](#setup) module gathers them again. See the [Variables Cheat Sheet](../VARIABLES_CHEATSHEET.md) for their names (`ansible_*` and `onigirazu_*`).

### command

Run a command without a shell: pipes, redirections and variables such as `$HOME` are not interpreted. Use `shell` for those.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cmd` | string | - | Command to run (required; free form in the short form; `command` is an alias) |
| `chdir` | string | - | Change into this directory first |
| `creates` | string | - | Skip when this path exists |
| `removes` | string | - | Skip when this path does not exist |
| `environment` | dict | - | Extra environment variables (the task keyword `environment` works too) |
| `shell` | boolean | `false` | Run through the shell, as the `shell` module does |

A non-zero exit code fails the task; use `failed_when` or `ignore_errors` to accept it. The task always reports `changed`; use `changed_when: false` for read-only commands.

#### Example

```yaml
- name: "Show disk usage"
  command:
    cmd: "df -h /"
  register: disk
  changed_when: false

- name: "Build"
  command: make install chdir=/src creates=/usr/local/bin/app
```

#### Return Values

```json
{
  "cmd": "df -h /",
  "stdout": "Filesystem  Size  Used Avail Use% Mounted on\n...",
  "stderr": "",
  "rc": 0,
  "start": "2026-09-26 10:30:00.000000",
  "end": "2026-09-26 10:30:00.042000",
  "delta": "42ms"
}
```

A task skipped by `creates`/`removes` returns only `msg`.

### shell

Run a command through `/bin/sh` (or `executable`). Same arguments, results and failure rule as `command`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cmd` | string | - | Shell command (required; free form in the short form; `command` is an alias) |
| `chdir` | string | - | Working directory |
| `executable` | string | - | Shell to run the command with, e.g. `/bin/bash` |
| `creates` | string | - | Skip when this path exists |
| `removes` | string | - | Skip when this path does not exist |
| `environment` | dict | - | Extra environment variables (string values) |

#### Example

```yaml
- name: "Count failed logins"
  shell:
    cmd: "grep -c 'Failed password' /var/log/auth.log || true"
    executable: /bin/bash
  register: failed_logins
  changed_when: false
```

### script

Copy a script from the control machine to the host, run it with `bash` and remove it. Always reports `changed`; a non-zero exit fails the task.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `script` | string | - | Path of the local script, relative to the working directory (required) |
| `args` | string | - | Arguments, passed to the host's shell |

Returns `stdout`, `rc` and `args` (`stderr` too when the host is local).

#### Example

```yaml
- name: "Run deployment script"
  script:
    script: "./scripts/deploy.sh"
    args: "production v1.2.3"

- name: "Short form"
  script: ./scripts/deploy.sh production v1.2.3
```

## File System Modules

### file

Manage files, directories and links.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Path (required); `dest` and `name` are aliases |
| `state` | string | see below | Desired state |
| `src` | string | - | Link target (`link`, `hard`) |
| `content` | string | - | With `state: present`: content to write when it differs |
| `mode` | string | - | Permissions, e.g. `"0644"` |
| `owner` | string | - | Owner |
| `group` | string | - | Group |
| `recurse` | boolean | `false` | With `state: directory`: apply owner/group/mode to everything below |
| `force` | boolean | `false` | Replace an existing non-link path with the link |
| `modification_time`, `access_time` | string | - | With `state: touch`: `preserve` on both leaves an existing file untouched |

#### States

- `file`: the path must exist; only its attributes are set (fails otherwise)
- `directory`: create the directory (with parents)
- `touch`: create an empty file or update its timestamps (an existing file is reported unchanged)
- `present`: create the file if it is missing; with `content`, write that content when it differs
- `absent`: remove the file or directory
- `link`: symbolic link at `path` pointing to `src`
- `hard`: hard link at `path` pointing to `src`

Without `state` the module creates a link when `src` is given, and otherwise sets attributes of an existing path.

#### Example

```yaml
- name: "Create application directory"
  file:
    path: "/opt/myapp"
    state: "directory"
    mode: "0755"
    owner: "appuser"
    group: "appgroup"

- name: "Point current at a release"
  file:
    src: "/opt/myapp/releases/v1.2.3"
    dest: "/opt/myapp/current"
    state: "link"
```

### copy

Copy a file from the control machine (or, with `remote_src`, from the host) or write given content.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Source file (not a directory); `src` or `content` is required |
| `content` | string | - | File content instead of `src` |
| `dest` | string | - | Destination path (required) |
| `backup` | boolean | `false` | Keep the old file as `<dest>.backup.<YYYYMMDD-HHMMSS>` |
| `mode` | string | - | Octal permissions, e.g. `"0644"` |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `force` | boolean | `true` | `false` leaves an existing `dest` alone, whatever it contains |
| `remote_src` | boolean | `false` | `src` is a path on the host, not on the control machine |

Returns `dest`, `checksum` (SHA-256 of the source), `size`, `msg` and `backup_file` when a backup was made.

#### Example

```yaml
- name: "Copy configuration file"
  copy:
    src: "./config/app.conf"
    dest: "/etc/myapp/app.conf"
    backup: true
    mode: "0644"
    owner: "root"
    group: "root"

- name: "Create file with content"
  copy:
    content: |
      server {
        listen 80;
        server_name {{ inventory_hostname }};
        root /var/www/html;
      }
    dest: "/etc/nginx/sites-available/default"
    mode: "0644"
```

### find

List the entries of a directory on the host that match a glob pattern and a type. Never changes anything.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | `.` | Directory to search; `paths` and `name` are aliases (a single path) |
| `pattern` | string | `*` | Glob for file names, e.g. `*.log`; `patterns` is an alias (a single pattern) |
| `type` | string | `file` | `file`, `directory`, `link`, `socket`, `pipe`, `block` or `char` |
| `recurse` | boolean | `false` | Search subdirectories too; otherwise only the directory's own entries |
| `limit` | integer | `0` | Maximum number of entries (0 = no limit) |

A missing directory returns an empty list.

#### Return Values

```json
{
  "files": [
    {
      "path": "/var/log/syslog",
      "name": "syslog",
      "type": "file",
      "isfile": true,
      "isdir": false,
      "islink": false,
      "size": "1024576",
      "mode": "644",
      "mtime": "1696086600"
    }
  ],
  "file_count": 1,
  "matched": 1
}
```

- `type` of an entry is `file`, `directory`, `link` or `other`; a link to a directory reports `directory`.
- `size` and `mtime` (Unix seconds) are strings: use `| int` to compare them.
- `mode` has no leading zero (`644`).

#### Examples

```yaml
- name: "Find log files"
  find:
    path: "/var/log"
    pattern: "*.log"
    limit: 100
  register: log_files

- name: "Fetch each log file"
  fetch:
    src: "{{ item.path }}"
    dest: "./logs/"
  loop: "{{ log_files.files }}"

- name: "Find temporary files below /tmp"
  find:
    path: "/tmp"
    pattern: "*.tmp"
    recurse: true
  register: tmp_files

- name: "Remove temporary files larger than 10 MB"
  file:
    path: "{{ item.path }}"
    state: absent
  loop: "{{ tmp_files.files }}"
  when: "item.size | int > 10485760"
```

### template

Render a Jinja2 template on the control machine and write it to the host.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Template file (`src` or `content` required) |
| `content` | string | - | Template text instead of a file |
| `dest` | string | - | Destination path (required) |
| `vars` | dict | - | Extra variables for this template |
| `backup` | boolean | `false` | Keep the old file as `<dest>.backup.<YYYYMMDD-HHMMSS>` |
| `mode` | string | `0644` | Permissions of a new file; an existing file keeps its mode unless `mode` is set |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `force` | boolean | `false` | Rewrite the file even when the content is unchanged |
| `trim_blocks` | boolean | `true` | Remove the newline after a `{% ... %}` tag |
| `lstrip_blocks` | boolean | `false` | Remove spaces before a tag at the start of a line (`{%+` keeps them) |

`{%-`/`-%}` strip whitespace, as in Ansible. Returns `dest`, `size`, `checksum` and `backup_file` when a backup was made.

#### Example

```yaml
- name: "Deploy application configuration"
  template:
    src: "./templates/app.conf.j2"
    dest: "/etc/myapp/app.conf"
    backup: true
    mode: "0644"
  vars:
    database_host: "{{ groups['database'][0] }}"
    database_port: 5432
```

### fetch

Fetch a file from the host to the control machine.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | File on the host (required) |
| `dest` | string | - | Local destination (required) |
| `flat` | boolean | `false` | `false`: store as `<dest>/<host>/<src>`; `true`: `dest` is the file, or a directory when it ends with `/` |
| `fail_on_missing` | boolean | `true` | Fail when `src` does not exist |
| `validate` | boolean | `true` | Compare checksums after the transfer |

A local file with the same content is left alone. Returns `src`, `dest` (the local path) and `checksum`.

#### Example

```yaml
- name: "Fetch log files"
  fetch:
    src: "/var/log/myapp.log"
    dest: "./logs/"

- name: "Backup configuration"
  fetch:
    src: "/etc/myapp/app.conf"
    dest: "./backups/{{ inventory_hostname }}-app.conf"
    flat: true
```

### slurp

Read a file from the host; `content` is base64 (`{{ r.content | b64decode }}`), `encoding` is `base64`, `source` the path.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | File on the host (required; `path` is an alias) |

```yaml
- slurp:
    src: /etc/hostname
  register: r
- debug:
    msg: "{{ r.content | b64decode }}"
```

## Configuration Modules

### config

Edit keys of a JSON or YAML file on the host.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File path (required) |
| `format` | string | `json` | `json` or `yaml` |
| `action` | string | `set` | See below |
| `key` | string | - | Key; dots address nested keys (`database.host`) |
| `value` | any | - | Value for `key` (`set`) |
| `values` | dict | - | Map merged into the file (`set`, `merge`) |
| `backup` | boolean | `false` | `set`, `delete`: back up the file first (`<path>.backup.<YYYYMMDD_HHMMSS>`) |
| `backup_path` | string | - | Backup to restore from (`restore`) |
| `schema` | dict | - | Simple schema (`required`, `properties` with `type`); required by `validate`, checked after `set` |

#### Actions

- `set`: set `key` to `value`, or merge `values`; creates the file if it is missing; writes only on a change
- `get`: return `value` of `key`, or the whole `config`
- `delete`: remove `key`; without `key`, delete the file
- `merge`: merge `values` into an existing file
- `backup`: copy the file and return `backup_path`
- `restore`: copy `backup_path` over the file
- `validate`: check the file against `schema`

The whole file is rewritten, so comments and key order of a YAML file are not kept.

#### Example

```yaml
- name: "Update database configuration"
  config:
    path: "/etc/myapp/config.yml"
    format: "yaml"
    key: "database.host"
    value: "{{ database_host }}"
    backup: true

- name: "Merge configuration"
  config:
    path: "/etc/myapp/config.json"
    action: "merge"
    values:
      logging:
        level: "info"
      cache:
        enabled: true
        ttl: 3600
```

### lineinfile

Ensure a line is present in a file, or remove matching lines.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File path (required) |
| `line` | string | - | Line content (required, also for `state: absent`) |
| `regexp` | string | - | Regular expression: the first matching line is replaced (`present`), every matching line is removed (`absent`); without it lines are compared with `line` |
| `state` | string | `present` | `present` or `absent` |
| `insertafter` | string | `EOF` | Regular expression: a new line goes after the last matching line; `EOF` is the end of the file |
| `insertbefore` | string | - | Regular expression: a new line goes before the last matching line; `BOF` is the start of the file |
| `firstmatch` | boolean | `false` | Use the first matching line for `insertafter`/`insertbefore` instead of the last |
| `backup` | boolean | `false` | Keep the old file as `<path>.<unixtime>.backup` |
| `create` | boolean | `false` | Create the file if it is missing (otherwise a missing file fails) |

A new line goes to the end of the file when there is no `insertafter`/`insertbefore` or it matches nothing, as in Ansible. An invalid pattern fails the task.

#### Example

```yaml
- name: "Configure SSH"
  lineinfile:
    path: "/etc/ssh/sshd_config"
    regexp: "^#?PasswordAuthentication"
    line: "PasswordAuthentication no"
    backup: true

- name: "Remove an old entry"
  lineinfile:
    path: "/etc/hosts"
    regexp: "oldhost"
    line: "unused"
    state: absent
```

### blockinfile

Insert, update or remove a block of text between marker lines.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File path (required); a missing file is created |
| `block` | string | - | Block content |
| `marker` | string | `# {mark} ANSIBLE MANAGED BLOCK` | Marker line; `{mark}` becomes `BEGIN` / `END` |
| `insertafter` | string | `EOF` | Regular expression: a new block goes after the last matching line; `EOF` is the end of the file |
| `insertbefore` | string | - | Regular expression: a new block goes before the last matching line; `BOF` is the start of the file |
| `firstmatch` | boolean | `false` | Use the first matching line for `insertafter`/`insertbefore` instead of the last |
| `state` | string | `present` | `present` or `absent` |
| `backup` | boolean | `false` | Keep the old file as `<path>.bak` |

A new block goes to the end of the file when there is no `insertafter`/`insertbefore` or it matches nothing, as in Ansible; an existing block is updated in place. An invalid pattern fails the task. Returns `path`, `state`, `msg` and `backup`.

#### Example

```yaml
- name: "Configure application block"
  blockinfile:
    path: "/etc/hosts"
    block: |
      192.168.1.10 app1.example.com
      192.168.1.11 app2.example.com
    marker: "# {mark} APPLICATION SERVERS"
    backup: true
```

### replace

Replace every match of a regular expression in a file (Go RE2 syntax, multiline mode).

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File path (required; must exist) |
| `regexp` | string | - | Regular expression (required) |
| `replace` | string | `""` | Replacement; `\1` and `\g<name>` refer to groups |
| `backup` | boolean | `false` | Keep the old file as `<path>.<YYYYMMDDhhmmss>~` |

Returns `msg` (number of replacements) and `backup_file`.

```yaml
- replace:
    path: /etc/myapp.conf
    regexp: '^port = (\d+)$'
    replace: 'port = 8080'
```

### ini_file

Set or remove one option of an INI file. Also `community.general.ini_file`.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File (required; `dest` is an alias) |
| `section` | string | - | Section; none: before the first section |
| `option` | string | - | Option; without it `present` makes sure the section exists and `absent` removes the section |
| `value` | string | - | Value (required with `option` and `state: present` unless `allow_no_value`) |
| `allow_no_value` | boolean | `false` | Allow an option without a value |
| `state` | string | `present` | `present` or `absent` |
| `no_extra_spaces` | boolean | `false` | `key=value` instead of `key = value` |
| `create` | boolean | `true` | Create a missing file |
| `backup` | boolean | `false` | Keep the old file as `<path>.<YYYYMMDDhhmmss>~` |
| `mode` | string | - | Octal mode of the written file |

Other lines of the option in the section are removed (Ansible's `exclusive`).

```yaml
- ini_file:
    path: /etc/myapp.ini
    section: server
    option: port
    value: "8080"
```

## Service Modules

### service

Start, stop and enable services with `systemctl`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Service name (required; `service` is an alias) |
| `state` | string | - | `started`, `stopped`, `restarted` or `reloaded`; without it the service is not started or stopped |
| `enabled` | boolean | - | Enable at boot |

One of `state` or `enabled` is required. `restarted` and `reloaded` always report `changed`. Returns `service_status`, `action` and `enabled` when they changed.

#### Example

```yaml
- name: "Start and enable nginx"
  service:
    name: "nginx"
    state: "started"
    enabled: true
```

### systemd

systemd services, unit files and timers.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Unit name (required except for `daemon-reload`; `service` and `unit` are aliases) |
| `operation` | string | `service` | `service`, `unit`, `timer`, `daemon-reload` or `status` |
| `state` | string | see below | `service`: `started`, `stopped`, `restarted`, `reloaded`; `timer`: `started` (default), `stopped`; `unit`: `present` (default), `absent` |
| `enabled` | boolean | - | Enable at boot (`service`, `timer`) |
| `masked` | boolean | - | Mask or unmask (`service`) |
| `daemon_reload` | boolean | `false` | Run `systemctl daemon-reload` first; alone it only reloads |
| `content` | string | - | `unit`: unit file content |
| `path` | string | `/etc/systemd/system/<name>` | `unit`: unit file path |

`operation: unit` writes `content` (and reloads systemd when it changed); `state: absent` stops, disables and removes the unit file. `status` returns `status` (LoadState, ActiveState, ...). Returns `action`, `enabled`, `masked`, `status`.

#### Example

```yaml
- name: "Install a unit file"
  systemd:
    operation: unit
    name: "myapp.service"
    content: |
      [Service]
      ExecStart=/usr/local/bin/myapp
      [Install]
      WantedBy=multi-user.target

- name: "Start and enable it"
  systemd:
    name: "myapp.service"
    state: "started"
    enabled: true

- name: "Mask unwanted service"
  systemd:
    name: "unwanted.service"
    masked: true
```

## System Control Modules

### sysctl

Set a kernel parameter now and in a sysctl file.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Sysctl key (required; `key` is an alias) |
| `value` | string | - | Value (required, also for `state: absent`; `val` is an alias) |
| `state` | string | `present` | `present` or `absent` (removes the line from `sysctl_file`; the running value stays) |
| `sysctl_file` | string | `/etc/sysctl.d/99-onigirazu.conf` | File for the persistent setting |
| `persist` | boolean | `true` | Write the setting to `sysctl_file` |
| `reload` | boolean | `true` | Run `sysctl -p <sysctl_file>` after the file changed |

#### Example

```yaml
- name: "Enable IP forwarding"
  sysctl:
    name: "net.ipv4.ip_forward"
    value: "1"

- name: "Tune TCP in its own file"
  sysctl:
    name: "net.ipv4.tcp_max_syn_backlog"
    value: "2048"
    sysctl_file: "/etc/sysctl.d/network.conf"

- name: "Remove a persistent setting"
  sysctl:
    name: "vm.swappiness"
    value: "10"
    state: "absent"
```

#### Return Values

```json
{
  "sysctl_key": "net.ipv4.ip_forward",
  "current_value": "0",
  "desired_value": "1",
  "msg": "Kernel parameter net.ipv4.ip_forward set to 1",
  "persisted_to_file": "/etc/sysctl.d/99-onigirazu.conf"
}
```

### reboot

Reboot the host and wait until it is back (a new boot id).

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `pre_reboot_delay` | integer | `0` | Seconds to wait before the reboot; `msg` is sent with `wall` first |
| `post_reboot_delay` | integer | `0` | Seconds to wait after the host is back |
| `reboot_timeout` | integer | `600` | Seconds to wait for the host to come back |
| `msg` | string | `System will reboot in a few seconds` | Message for `wall` (with `pre_reboot_delay`) |
| `reboot_command` | string | - | Command that reboots; default: `systemctl reboot` two seconds later |
| `test_boot` | boolean | `false` | Only check `systemctl is-system-running` (`degraded` passes); no reboot |

A local host (the control machine) is never rebooted. Returns `msg` and `elapsed` (seconds).

#### Example

```yaml
- name: "Reboot"
  reboot:

- name: "Reboot with notice"
  reboot:
    pre_reboot_delay: 60
    msg: "System maintenance - rebooting in 1 minute"
    reboot_timeout: 900
```

### mount

Manage `/etc/fstab` entries and mounted filesystems, as Ansible's mount.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Mount point (required; `name` is an alias) |
| `src` | string | - | Device or source (required for `present` and `mounted`) |
| `state` | string | `present` | See below |
| `fstype` | string | `auto` | Filesystem type |
| `opts` | string | `defaults` | Mount options |
| `dump` | string | `0` | fstab dump field |
| `passno` | string | `0` | fstab pass field |
| `backup` | boolean | `true` | Copy `/etc/fstab` to `/etc/fstab.bak` before writing it |

#### States

- `present`: fstab entry only
- `mounted`: fstab entry, mount point created, mounted (remounted when the entry changed)
- `unmounted`: not mounted; fstab untouched
- `absent`: not mounted and no fstab entry

Returns `path` and `msg`.

#### Example

```yaml
- name: "Mount NFS share"
  mount:
    path: "/mnt/nfs"
    src: "192.168.1.100:/export/data"
    fstype: "nfs"
    opts: "defaults,nfsvers=4.0,hard"
    state: "mounted"

- name: "Remove a mount"
  mount:
    path: "/mnt/tmp"
    state: "absent"
```

### archive

Create a tar or zip archive on the host with `tar`/`zip`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string/list | - | Files, directories or glob patterns (required) |
| `dest` | string | - | Archive file (required); its directory is created |
| `format` | string | `gz` | `gz` (tar.gz), `bz2`, `xz`, `tar` or `zip` (needs `zip` on the host) |
| `exclude_path` | string/list | - | Patterns to exclude, relative to `/` (a leading `/` is removed) |
| `remove` | boolean | `false` | Remove the sources after archiving |

Paths are stored relative to `/`. The task is `ok` when `dest` exists and no source is newer; it fails when nothing matches `path`. Returns `dest` and `format`.

#### Example

```yaml
- name: "Archive logs"
  archive:
    path: "/var/log/app"
    dest: "/backups/app-logs.tar.gz"

- name: "Archive with exclusions"
  archive:
    path:
      - "/opt/application"
      - "/etc/application"
    dest: "/backups/app.tar.xz"
    format: "xz"
    exclude_path:
      - "/opt/application/cache"
```

### unarchive

Extract a tar (any compression tar understands) or zip archive into an existing directory.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Archive on the control machine (in a role: `files/`), or on the host with `remote_src` |
| `dest` | string | - | Directory to extract into; it must exist |
| `remote_src` | boolean | `false` | `src` is on the host |
| `creates` | string | - | Skip when this path exists |
| `owner`, `group` | string | - | Owner of the extracted files (recursive) |
| `extra_opts` | list | - | Extra options for `tar` / `unzip` |
| `list_files` | boolean | `false` | Return the archive's members in `files` |

The task is `ok` when every member of the archive already exists in `dest`; otherwise
the archive is extracted over it. Zip archives need `unzip` on the host.

```yaml
- name: Install the app
  unarchive:
    src: app-1.4.tar.gz
    dest: /opt/app
    owner: app
```

### timezone

Set the time zone. Also `community.general.timezone`.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | IANA time zone, e.g. `Europe/Madrid`, `UTC` (required) |

Uses `timedatectl` when systemd runs, else links `/etc/localtime` (and writes
`/etc/timezone` where it exists). Returns `name` and `previous`.

### hostname

Set the host name with `hostnamectl`, else `/etc/hostname`.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | New host name (required) |

Returns `name` and updates the `ansible_hostname` fact.

## Package Modules

### package

Install or remove packages with the host's package manager: apt, yum/dnf or Homebrew. pacman, zypper and Chocolatey are detected but not implemented.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Package, list of packages, or list of `{name, version, state}` (required; `pkg` is an alias) |
| `state` | string | `present` | `present`, `absent` or `latest` |
| `version` | string | - | Version for all packages |
| `update_cache` | boolean | `false` | Refresh the package cache first |
| `dry_run` | boolean | `false` | Only return `previews` of the operations |
| `parallel` | boolean | `false` | Install several packages in parallel |
| `max_retries` | integer | `3` | Retries per package |
| `enable_rollback` | boolean | `false` | Undo the packages already changed when one fails |
| `lock_file` | string | - | Local lock file recording installed versions |

Returns `summary` and `batch_operation`, or `package_state` when nothing had to change.

#### Example

```yaml
- name: "Install web server packages"
  package:
    name:
      - "nginx"
      - "php-fpm"
    state: "present"
    update_cache: true

- name: "Install specific version"
  package:
    name: "docker-ce"
    version: "20.10.17"
```

### apt

Debian/Ubuntu packages with `apt-get`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Package(s) (`pkg`, `package` are aliases); optional for cache, upgrade and cleanup runs |
| `state` | string | `present` | `present`, `latest` or `absent` |
| `update_cache` | boolean | `false` | Run `apt-get update` first (`update-cache` is an alias) |
| `cache_valid_time` | int | `0` | Skip the cache update if it is younger than this many seconds |
| `upgrade` | string | `no` | `yes`/`safe` (apt-get upgrade), `full`/`dist` (dist-upgrade); predicted in check mode |
| `autoremove` | boolean | `false` | Remove unused packages |
| `autoclean` | boolean | `false` | Clean the package cache |

`absent` removes with `apt-get remove` (configuration files stay). Returns `state`, `packages`, `msg`, `cache_updated`, `upgrade`.

#### Example

```yaml
- name: "Upgrade nginx to the latest version"
  apt:
    name: nginx
    state: latest
    update_cache: true
    autoremove: true

- name: "Upgrade all packages"
  apt:
    update_cache: true
    cache_valid_time: 3600
    upgrade: dist
  become: true
```

### apt_repository

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `repo` | string | - | A `deb ...` / `deb-src ...` line, or `ppa:owner/name` (required) |
| `state` | string | `present` | `present` or `absent` |
| `filename` | string | from the URL | File in `/etc/apt/sources.list.d/` (without `.list`) |
| `update_cache` | boolean | `true` | Run `apt-get update` after a change |

A line is looked for in `/etc/apt/sources.list` and `sources.list.d/*.list`; a repository already
described in a deb822 `.sources` file (such as Ubuntu's own) also counts as present. `.sources` files are not edited. A new line goes to its own file, named as Ansible names it
(`download_docker_com_linux_ubuntu.list`); removing the last line of a `.list` file deletes it.
PPAs go through `add-apt-repository` (package `software-properties-common`).

### apt_key

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` / `data` / `file` / `keyserver` | string | - | Where the key comes from (one of them); `url` is fetched by the host with curl or wget, `file` is on the host, `keyserver` needs `id` and gpg |
| `keyring` | string | - | Absolute path to write the key to, e.g. `/etc/apt/keyrings/docker.asc` |
| `id` | string | - | Key id; names the file in `/etc/apt/trusted.gpg.d/` when there is no `keyring` |
| `state` | string | `present` | `present`, or `absent` (removes `keyring`, or the file named by `id`) |

`apt-key` is not used. An ASCII-armored key is written as is (use a `.asc` keyring); for a `.gpg`
keyring it is converted with `gpg --dearmor` on the host. With `keyserver` (and `id`) the host fetches the
key with gpg (`keyserver: hkps://keyserver.ubuntu.com`).

```yaml
- apt_key:
    url: https://download.docker.com/linux/ubuntu/gpg
    keyring: /etc/apt/keyrings/docker.asc
- apt_repository:
    repo: "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu {{ ansible_distribution_release }} stable"
```

### yum

RHEL/Fedora packages with the `yum` command (dnf provides it). `dnf` and `dnf5` tasks run this module.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Package(s) (`pkg` is an alias; optional if only updating the cache) |
| `state` | string | `present` | `present`, `latest` or `absent` |
| `enablerepo` | string | - | Repositories to enable for this run |
| `disablerepo` | string | - | Repositories to disable for this run |
| `security` | boolean | `false` | Run `yum update --security` (all security updates) instead of installing `name` |
| `update_cache` | boolean | `false` | Run `yum makecache` first |

#### Example

```yaml
- name: "Install development tools"
  yum:
    name: "@Development Tools"
    state: "present"

- name: "Install from specific repo"
  yum:
    name: "docker-ce"
    enablerepo: "docker-ce-stable"
```

### pip

Python packages with pip.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Packages; `pkg==1.2` pins a version (`name` or `requirements` is required; `pkg` is an alias) |
| `version` | string | - | Version for a package without a specifier |
| `state` | string | `present` | `present`, `absent`, `latest` or `forcereinstall` |
| `requirements` | string | - | Requirements file on the host |
| `virtualenv` | string | - | Virtualenv to use; created when missing |
| `virtualenv_command` | string | `python3 -m venv` | Command that creates the virtualenv |
| `executable` | string | `pip3`, else `pip` | pip to run |
| `extra_args` | string | - | Extra arguments for `pip install` |

```yaml
- pip:
    name: [requests, "flask==3.0.3"]
    virtualenv: /opt/app/venv
```

## Network Modules

### uri

Make an HTTP request from the host, with curl or, where curl is missing, the host's Python (`python3`, then `python`).

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | - | Request URL (required) |
| `method` | string | `GET` | HTTP method |
| `body` | string/dict | - | Request body; a dict is sent as JSON |
| `body_format` | string | `raw` | `json` or `form-urlencoded` set the Content-Type |
| `headers` | dict | - | Request headers |
| `user` | string | - | Username for basic authentication |
| `password` | string | - | Password for basic authentication |
| `timeout` | integer | `30` | Request timeout in seconds |
| `status_code` | int/list | `200` | Accepted status codes; others fail the task |
| `return_content` | boolean | `false` | Also return the body as `content` |
| `validate_certs` | boolean | `true` | `false` skips TLS verification |

Returns `status`, `url`, `headers`, `text` (the body), `json` (when the body is JSON), `content` and `elapsed`. Never reports `changed`.

#### Example

```yaml
- name: "Check API health"
  uri:
    url: "https://api.example.com/health"
    timeout: 10
  register: health_check

- name: "Send webhook notification"
  uri:
    url: "https://hooks.example.com/services/deploy"
    method: "POST"
    body_format: "json"
    body:
      text: "Deployment completed successfully"
    status_code: [200, 204]
```

### get_url

Download a file on the host with curl (or wget).

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | - | Download URL (required) |
| `dest` | string | - | Destination file (required) |
| `checksum` | string | - | `<algorithm>:<hex>` (`md5`, `sha1`, `sha256`, `sha512`); the download must match, and an existing file that matches is kept |
| `force` | boolean | `false` | Download even when `dest` exists |
| `headers` | dict | - | Request headers |
| `timeout` | integer | `30` | Download timeout in seconds |
| `mode` | string | `0644` | File permissions |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `backup` | boolean | `false` | Keep the old file as `<dest>.<unixtime>.backup` |

Without `force` an existing `dest` is not downloaded again (unless `checksum` differs). Returns `url`, `dest`, `msg`, `size`, `checksum`.

#### Example

```yaml
- name: "Download application binary"
  get_url:
    url: "https://releases.example.com/myapp/v1.2.3/myapp-linux-amd64"
    dest: "/usr/local/bin/myapp"
    checksum: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    mode: "0755"
```

## Security Modules

### user

Manage user accounts. An existing account is brought to the given settings.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Username (required; `user` is an alias) |
| `state` | string | `present` | `present` or `absent` |
| `uid` | integer | - | User ID |
| `group` | string | - | Primary group (name or GID) |
| `groups` | list/string | - | Supplementary groups |
| `append` | boolean | `false` | Add to `groups` instead of replacing the list |
| `home` | string | - | Home directory |
| `move_home` | boolean | `false` | Move the old home when `home` changes |
| `shell` | string | - | Login shell |
| `comment` | string | - | GECOS field |
| `password` | string | - | Encrypted password (see the `password_hash` filter) |
| `create_home` | boolean | `true` | Create the home directory (new accounts) |
| `system` | boolean | `false` | System account (new accounts) |
| `remove` | boolean | `false` | With `state: absent`: also remove the home directory |

`gid` is accepted only when the account is created; use `group` to change the primary group of an existing account.

#### Example

```yaml
- name: "Create application user"
  user:
    name: "appuser"
    uid: 1001
    group: "appgroup"
    home: "/opt/myapp"
    shell: "/bin/bash"

- name: "Add user to groups"
  user:
    name: "myuser"
    groups:
      - "docker"
      - "sudo"
    append: true

- name: "Remove a user with their home"
  user:
    name: "olduser"
    state: absent
    remove: true
```

### group

Manage groups.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Group name (required) |
| `state` | string | `present` | `present` or `absent` |
| `gid` | integer | - | Group ID |
| `system` | boolean | `false` | System group |

#### Example

```yaml
- name: "Create application group"
  group:
    name: "appgroup"
    gid: 1001
```

### authorized_key

Add or remove one SSH public key in `~<user>/.ssh/authorized_keys`. Also `ansible.posix.authorized_key`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `user` | string | - | Account (required; must exist and have a home directory) |
| `key` | string | - | One public key line (required) |
| `state` | string | `present` | `present` or `absent` |
| `exclusive` | boolean | `false` | With `present`: remove every other key |

Keys are compared by type and key data, not by comment. `~/.ssh` (0700) and the file (0600) are created and owned by the user. Returns `user`, `state`, `key_count`, `msg`.

#### Example

```yaml
- name: "Add SSH key for user"
  authorized_key:
    user: "myuser"
    key: "{{ lookup('file', '~/.ssh/id_ed25519.pub') }}"

- name: "Allow only the deploy key"
  authorized_key:
    user: "appuser"
    key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... deploy@ci"
    exclusive: true
```

## System Connectivity

### ping

Test the connection to the host.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `data` | string | `pong` | Value returned in `ping` |

#### Example

```yaml
- name: "Test connectivity to all hosts"
  ping:
```

#### Return Values

```json
{
  "ping": "pong",
  "connection": "ssh",
  "host": "webserver01",
  "address": "192.168.1.100",
  "user": "ubuntu",
  "port": 22
}
```

### stat

Read the status of a path. Never changes anything.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Path (required; `dest` and `name` are aliases) |
| `get_checksum` | boolean | `true` | Compute a checksum of regular files |
| `checksum_algorithm` | string | `sha1` | `md5`, `sha1`, `sha224`, `sha256`, `sha384` or `sha512` |

A link is reported as a link, not followed.

#### Example

```yaml
- name: "Get file status"
  stat:
    path: "/etc/nginx/nginx.conf"
  register: nginx_config

- name: "Show owner and mode"
  debug:
    msg: "{{ nginx_config.stat.pw_name }} {{ nginx_config.stat.mode }}"
  when: nginx_config.stat.exists
```

#### Return Values

```json
{
  "stat": {
    "exists": true,
    "path": "/etc/nginx/nginx.conf",
    "isreg": true,
    "isdir": false,
    "islnk": false,
    "size": 1482,
    "mode": "0644",
    "uid": 0,
    "gid": 0,
    "pw_name": "root",
    "gr_name": "root",
    "mtime": 1696086600,
    "atime": 1696086600,
    "ctime": 1696086600,
    "inode": 131090,
    "nlink": 1,
    "readable": true,
    "writable": true,
    "executable": false,
    "checksum": "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
  }
}
```

`readable`, `writable` and `executable` are the owner's permission bits. Links also carry `lnk_source` (resolved path) and `lnk_target` (link text). For a missing path only `exists: false` and `path` are returned. The same fields are also available at the top level of the result.

## Version Control

### git

Clone a repository on the host, or update an existing clone.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `repo` | string | - | Repository URL (required; `name` is an alias) |
| `dest` | string | - | Destination path (required) |
| `version` | string | `HEAD` | Branch, tag or commit to check out |
| `update` | boolean | `true` | Fetch and check out `version` in an existing clone |
| `force` | boolean | `false` | Clone even when `dest` exists and is not a git repository |
| `depth` | integer | - | Shallow clone with that many commits; a branch or tag clones just that ref, a commit needs its full hash |

`changed` means the checked-out commit changed. Returns `before`, `after`, `version`, `dest`, `info`.

#### Example

```yaml
- name: "Clone application repository"
  git:
    repo: "https://github.com/myorg/myapp.git"
    dest: "/opt/myapp"
    version: "main"

- name: "Check out a tag"
  git:
    repo: "git@github.com:myorg/myapp.git"
    dest: "/opt/myapp"
    version: "v1.2.3"
```

## Scheduled Jobs

### cron

Cron jobs in a user's crontab, whole crontabs, and files in `/etc/cron.*`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `operation` | string | `job` | `job`, `file`, `system` or `list` |
| `name` | string | - | `job`: job name (required); `system`: file name in the cron directory (required) |
| `job` | string | - | Command (`job`, required with `state: present`) |
| `minute`, `hour`, `day`, `month`, `weekday` | string | `*` | Schedule (`job`) |
| `special_time` | string | - | `reboot`, `hourly`, `daily`, ... without `@` (`job`; replaces the schedule) |
| `user` | string | `root` | Crontab owner (`job`, `file`, `list`) |
| `state` | string | `present` | `present` or `absent` |
| `content` | string | - | `file`: the whole crontab; `system`: the file content |
| `cron_type` | string | `d` | `system`: `d` (`/etc/cron.d`), `hourly`, `daily`, `weekly` or `monthly` |
| `backup` | boolean | `true` | `file`: save the old crontab to `/root/crontab.<user>.<time>.backup` |

`operation: job` names each job with a `# Onigirazu: <name>` (or `# Ansible: <name>`) comment and
rewrites the whole crontab with the named jobs only: lines without such a comment are dropped.
`list` returns `jobs` (name to line), `jobs_count` and `raw_crontab`.

#### Examples

```yaml
- name: "Create daily backup job"
  cron:
    name: "Daily database backup"
    job: "/usr/local/bin/backup-db.sh"
    minute: "0"
    hour: "2"
    user: "backupuser"

- name: "Run at reboot"
  cron:
    name: "Start application"
    job: "/opt/myapp/start.sh"
    special_time: "reboot"

- name: "Cron file in /etc/cron.d"
  cron:
    operation: "system"
    name: "myapp"
    content: "*/5 * * * * root /opt/myapp/tick.sh"

- name: "List cron jobs"
  cron:
    operation: "list"
  register: cron_jobs
```

## Security & Firewall

### firewall

Firewall rules through whichever of ufw, firewalld or iptables the host has (in that order).

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `operation` | string | `rule` | `enable`, `disable`, `rule`, `service`, `source`, `list` or `reload` |
| `action` | string | `allow` | `allow` or `deny` (`rule`, `service`, `source`) |
| `state` | string | `present` | `present` or `absent` |
| `port` | string | - | Port or range (`rule`, required) |
| `protocol` | string | `tcp` | `tcp` or `udp` (`rule`) |
| `service` | string | - | Service name such as `http`, `ssh` (`service`, required) |
| `source` | string | - | Source IP or network (`source`, required) |

Returns `firewall_type`, `action`, and for `list` `rules` and `rules_count`.

#### Examples

```yaml
- name: "Enable firewall"
  firewall:
    operation: "enable"

- name: "Allow SSH port"
  firewall:
    port: "22"
    protocol: "tcp"

- name: "Allow HTTP service"
  firewall:
    operation: "service"
    service: "http"

- name: "Allow traffic from a network"
  firewall:
    operation: "source"
    source: "192.168.1.0/24"

- name: "List all firewall rules"
  firewall:
    operation: "list"
  register: fw_rules
```

### ufw

The Uncomplicated Firewall. Also `community.general.ufw`. One of `state`, `rule`, `policy` or `logging` is required.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `state` | string | - | `enabled`, `disabled`, `reloaded` or `reset` |
| `policy` | string | - | Default policy `allow`, `deny` or `reject` (`default` is an alias), for `direction` |
| `direction` | string | `incoming` for `policy` | `incoming`/`in`, `outgoing`/`out`, `routed`; for a rule: `in` or `out` |
| `logging` | string | - | `on`, `off`, `low`, `medium`, `high`, `full` |
| `rule` | string | - | `allow`, `deny`, `limit` or `reject` |
| `port` | string | - | Destination port (`to_port` is an alias) |
| `proto` | string | - | `tcp`, `udp`, ... (`protocol` is an alias) |
| `src` | string | `any` | Source address (`from_ip`, `from` are aliases) |
| `from_port` | string | - | Source port |
| `dest` | string | `any` | Destination address (`to_ip`, `to` are aliases) |
| `interface` | string | - | Interface (`if` is an alias) |
| `route` | boolean | `false` | Routed rule |
| `delete` | boolean | `false` | Delete the rule |
| `insert` | string | - | Insert at this rule number |
| `log` | boolean | `false` | Log matching packets |
| `comment` | string | - | Rule comment |

A rule that exists is not added again. Returns `rule` and `commands`.

```yaml
- ufw: {rule: allow, port: "22", proto: tcp}
- ufw: {policy: deny, direction: incoming}
- ufw: {state: enabled}
```

## Container Management

### docker_container

Manage Docker containers with the docker CLI on the host.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Container name (required) |
| `image` | string | - | Image (required to create the container) |
| `state` | string | `started` | `present`, `started`, `stopped`, `restarted` or `absent` |
| `command` | string | - | Command, split like a shell would |
| `ports` | list | - | Port mappings (e.g. `"8080:80"`) |
| `volumes` | list | - | Volume mounts |
| `env` | dict | - | Environment variables |
| `networks` | list | - | Network names |
| `restart_policy` | string | - | Restart policy |
| `cpus` | number | - | CPU limit (e.g. `1.5`) |
| `memory` | string | - | Memory limit (`512m`, `1g` or bytes) |
| `force` | boolean | `false` | Remove with `docker rm -f` (`absent`) |

A new container is created with `docker run -d` (so `present` also starts it). An existing container is not compared
with the other arguments and is kept as it is. Only `cpus` and `memory` are compared and changed in place with
`docker update`; the swap limit stays unlimited if it was, otherwise it becomes twice the memory, as docker sets
for a new container. Returns `action`, `container`, `updated`.

#### Examples

```yaml
- name: "Create and run web container"
  docker_container:
    name: "myapp"
    image: "nginx:latest"
    ports:
      - "8080:80"
    volumes:
      - "/var/www/html:/usr/share/nginx/html"
    env:
      ENVIRONMENT: "production"
    restart_policy: "unless-stopped"

- name: "Cap a running container"
  docker_container:
    name: "myapp"
    image: "nginx:latest"
    cpus: 1
    memory: 512m

- name: "Stop container"
  docker_container:
    name: "myapp"
    state: "stopped"
```

### docker_image

Pull, build or remove Docker images.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Image name without tag (required) |
| `tag` | string | `latest` | Image tag |
| `state` | string | `present` | `present` (pull when missing), `absent` or `build` |
| `force` | boolean | `false` | `present`: pull again; `absent`: remove with `-f` |
| `platform` | string | - | Platform for the pull |
| `path` | string | - | `build`: build context |
| `dockerfile` | string | - | `build`: Dockerfile |
| `build_args` | dict | - | `build`: build arguments |
| `nocache` | boolean | `false` | `build`: no cache |
| `pull` | boolean | `false` | `build`: pull newer base images |

`build` always reports `changed`. Returns `action` and `image`.

#### Examples

```yaml
- name: "Pull nginx"
  docker_image:
    name: "nginx"
    tag: "1.27"

- name: "Remove old image"
  docker_image:
    name: "myapp"
    tag: "v1.0.0"
    state: "absent"
```

### docker_compose

Docker Compose projects (`docker compose`, else `docker-compose`). Also `community.docker.docker_compose` and `community.docker.docker_compose_v2`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `project_dir` | string | - | Directory of the project (required; `project_src` is an alias) |
| `file` | string | - | Compose file (the first of `files` is used too) |
| `project_name` | string | - | Project name |
| `state` | string | `present` | `present` (up), `absent` (down), `restarted`, `pull` or `build` |
| `services` | list | - | Services (`present`, `restarted`, `build`) |
| `detach` | boolean | `true` | `false` runs `up` in the foreground |
| `build` | boolean | `false` | `present`: `up --build` (`always` = true; `never`, `missing`, `policy` = false) |
| `force_recreate` | boolean | `false` | `present`: `--force-recreate` |
| `remove_volumes` | boolean | `false` | `absent`: `down -v` |
| `remove_orphans` | boolean | `false` | `absent`: `--remove-orphans` |
| `nocache`, `pull` | boolean | `false` | `build`: `--no-cache`, `--pull` |

`present` and `absent` report `changed` when the project's containers changed; `restarted`, `pull` and `build` always do. Returns `action`.

#### Examples

```yaml
- name: "Start the stack"
  docker_compose:
    project_dir: "/opt/mystack"
    state: present

- name: "Stop and remove it"
  docker_compose:
    project_dir: "/opt/mystack"
    state: absent
```

### podman

Manage Podman containers; same arguments and behaviour as `docker_container` without the limits.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Container name (required) |
| `image` | string | - | Image (required to create the container) |
| `state` | string | `started` | `present`, `started`, `stopped`, `restarted` or `absent` |
| `command` | string | - | Command |
| `ports`, `volumes`, `networks` | list | - | Port mappings, volume mounts, network names |
| `env` | dict | - | Environment variables |
| `restart_policy` | string | - | Restart policy |
| `rootless` | boolean | `false` | Add `--userns=keep-id` |
| `force` | boolean | `false` | Remove with `-f` (`absent`) |

#### Examples

```yaml
- name: "Run podman container"
  podman:
    name: "myapp"
    image: "docker.io/library/nginx:latest"
    ports:
      - "8080:80"
```

### docker_host_info

Docker host information (`community.docker.docker_host_info`). Never changes anything.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `containers` | boolean | `false` | Also list the containers |
| `containers_all` | boolean | `false` | Include stopped containers |
| `containers_filters` | dict | - | `docker ps` filters, e.g. `name: [a, b]` |

Returns `host_info` (`docker info`), `can_talk_to_docker` and `containers` as the Docker API lists them
(`Id`, `Names` with the leading `/`, `Image`, `State`).

## Database Management

The database modules run the client on the host (`mysql`, `psql`, `mongosh`/`mongo`). They share the
connection arguments `login_user`, `login_password`, `login_host` and `login_port`; without host and port the
local socket is used, where root (MySQL/MariaDB) and `postgres` (with `become_user: postgres`) log in without a password.
MongoDB also takes `login_database` (default `admin`).

### mysql_db

Manage MySQL/MariaDB databases.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Database name (required) |
| `state` | string | `present` | `present`, `absent`, `dump` or `import` |
| `charset` | string | `utf8mb4` | Character set of a new database |
| `collation` | string | `utf8mb4_unicode_ci` | Collation of a new database |
| `target` | string | - | File on the host for `dump` / `import` (required there) |

An existing database is not altered. `dump` and `import` always report `changed`. Returns `action` and `database`.

#### Examples

```yaml
- name: "Create application database"
  mysql_db:
    name: "myapp_db"
    collation: "utf8mb4_general_ci"

- name: "Dump it"
  mysql_db:
    name: "myapp_db"
    state: dump
    target: "/var/backups/myapp_db.sql"
```

### mysql_user

Manage MySQL/MariaDB accounts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Username (required) |
| `host` | string | `localhost` | Host part of the account |
| `password` | string | - | Password, set when the account is created |
| `state` | string | `present` | `present` or `absent` |
| `priv` | string | - | `db.table:PRIV,PRIV/db2.*:ALL`; granted, never revoked |

#### Examples

```yaml
- name: "Create database user"
  mysql_user:
    name: "appuser"
    host: "192.168.%"
    password: "{{ vault_db_password }}"
    priv: "myapp_db.*:ALL"
```

### postgresql_db

Manage PostgreSQL databases.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Database name (required) |
| `state` | string | `present` | `present`, `absent`, `dump` or `restore` |
| `owner` | string | - | Owner role of a new database |
| `encoding` | string | - | Encoding of a new database (created from `template0`) |
| `target` | string | - | File on the host for `dump` / `restore` (required there) |

An existing database is not altered.

#### Examples

```yaml
- name: "Create PostgreSQL database"
  postgresql_db:
    name: "myapp_db"
    owner: "appuser"
    encoding: "UTF8"
  become: true
  become_user: postgres
```

### postgresql_user

Manage PostgreSQL login roles.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Role name (required) |
| `password` | string | - | Password, set when the role is created |
| `state` | string | `present` | `present` or `absent` |
| `db` | string | - | Database for `priv` |
| `priv` | string | - | Database privileges, e.g. `CONNECT,CREATE` (needs `db`); granted, never revoked |
| `superuser` | boolean | `false` | `SUPERUSER` for a new role |
| `createdb` | boolean | `false` | `CREATEDB` for a new role |

#### Examples

```yaml
- name: "Create PostgreSQL user"
  postgresql_user:
    name: "appuser"
    password: "{{ vault_db_password }}"
    db: "myapp_db"
    priv: "CONNECT"
  become: true
  become_user: postgres
```

### mongodb

Manage MongoDB databases and users.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Database or user name (required) |
| `operation` | string | `database` | `database` or `user` |
| `state` | string | `present` | `present` or `absent` |
| `database` | string | - | `user`: database of the user (required) |
| `password` | string | - | `user`: password of a new user |
| `roles` | list | - | `user`: roles of a new user, e.g. `[{role: readWrite, db: myapp_db}]` |

A new database is created with a `_init` collection. An existing user is not changed.

#### Examples

```yaml
- name: "Create MongoDB database"
  mongodb:
    name: "myapp_db"

- name: "Create MongoDB user"
  mongodb:
    operation: "user"
    name: "appuser"
    database: "myapp_db"
    password: "{{ vault_db_password }}"
    roles:
      - {role: readWrite, db: myapp_db}
```

## Utility Modules

### debug

Print a message or a variable.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `msg` | any | - | Message (templated) |
| `var` | string | - | Variable or expression to print: `result.stdout`, `result['stdout']`, `items \| length` |

One of them is required; `msg` wins when both are given. An undefined `var` prints "VARIABLE IS NOT DEFINED!".

#### Example

```yaml
- name: "Debug variable"
  debug:
    var: hostvars[inventory_hostname]['ansible_distribution']

- name: "Debug message"
  debug:
    msg: "Current user is {{ ansible_user_id }}"
```

### set_fact

Set variables for the current host for the rest of the run. Every argument becomes a variable; types are kept. `cacheable` is accepted and ignored.

#### Example

```yaml
- name: "Set deployment facts"
  set_fact:
    deployment_time: "{{ ansible_date_time.iso8601 }}"
    app_version: "v1.2.3"
    replicas: 3
```

### assert

Fail unless every expression holds.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `that` | string/list | - | Expression or list of expressions (required) |
| `fail_msg` | string | `Assertion failed` | Message on failure (`msg` is an alias) |
| `success_msg` | string | `All assertions passed` | Message on success |

On failure the result has `assertion` (the failed expression) and `evaluated_to: false`.

#### Example

```yaml
- name: "Check prerequisites"
  assert:
    that:
      - ansible_os_family == "Debian"
      - ansible_processor_vcpus >= 2
    fail_msg: "Unsupported host"
```

### include_vars

Load variables from YAML or JSON files on the control machine into the host's variables.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `file` | string | - | File, relative to the playbook directory (free form in the short form) |
| `dir` | string | - | Load every `.yml`, `.yaml` and `.json` file of this directory, in name order |
| `name` | string | - | Put the variables under this one key |

`file` or `dir` is required. Returns `ansible_included_var_files`.

#### Example

```yaml
- include_vars: vars/{{ ansible_os_family }}.yml

- name: "Load all settings under one key"
  include_vars:
    dir: settings
    name: settings
```

### include_role / import_role

Run a role's tasks at this point. Both are resolved when the playbook is loaded.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Role name (required) |
| `tasks_from` | string | `main` | Task file of the role to run (`.yml` is added when there is no extension) |

Tags of the task are added to the role's tasks.

#### Example

```yaml
- name: "Configure nginx"
  include_role:
    name: nginx
    tasks_from: install
```

### setup

Gather the facts of the host again (also `gather_facts`), e.g. after a task
wrote a local fact. The host's variables get the new facts; the result's
`ansible_facts` holds those `filter` selects.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `filter` | string/list | all | Fact names or globs, e.g. `ansible_local`, `ansible_distribution*` (a string may be comma-separated) |
| `fact_path` | string | `/etc/ansible/facts.d` | Directory of the local facts |

Local facts (`ansible_local`) are the `*.fact` files of `fact_path` by name:
JSON, else INI sections, else text; an executable file is run and its output
read. Other arguments such as `gather_subset` are ignored.

```yaml
- ansible.builtin.setup:
    filter: ansible_local
```

### getent

Read a getent database into the `getent_<database>` fact: the first field of
each entry is the key, the others its list.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `database` | string | - | `passwd`, `group`, `hosts`, ... (required) |
| `key` | string | - | Only this entry |
| `split` | string | `:` for passwd/group/shadow/gshadow, else whitespace | Field separator |
| `fail_key` | boolean | `true` | A missing `key` fails the task |

```yaml
- ansible.builtin.getent: {database: passwd, key: deploy}
- debug: {msg: "{{ getent_passwd.deploy[4] }}"}   # home directory
```

### meta

Engine actions.

| Action | Effect |
|--------|--------|
| `flush_handlers` | Run notified handlers now |
| `end_host` | Stop the play for this host |
| `end_play` | Stop the play for all hosts |
| `noop`, `clear_host_errors`, `refresh_inventory`, `reset_connection` | Accepted, no effect |

```yaml
- meta: flush_handlers
```

### wait_for

Wait on the host until a port answers or a file exists (or contains a pattern), or until that stops being true.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | integer | - | TCP port to connect to, from the host (needs bash) |
| `host` | string | `127.0.0.1` | Address for `port`, as the host sees it |
| `path` | string | - | File that must exist |
| `search_regex` | string | - | With `path`: extended regular expression the file must contain |
| `state` | string | `started` | `started`/`present`: wait until true; `stopped`/`absent`: wait until false |
| `timeout` | integer | `300` | Seconds before the task fails |
| `delay` | integer | `0` | Seconds to wait before the first check |

`port` or `path` is required. Returns `elapsed` and `msg`.

#### Example

```yaml
- name: "Wait for the service port"
  wait_for:
    port: 8080
    timeout: 60

- name: "Wait for log message"
  wait_for:
    path: "/var/log/myapp.log"
    search_regex: "Server started"
    timeout: 120
```

### pause

Wait for a time or for input on the control machine.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `seconds` | integer | `0` | Seconds to wait |
| `minutes` | integer | `0` | Minutes to wait (added to `seconds`) |
| `prompt` | string | - | Print this and wait for a line of input (returned as `user_input`); the time is then ignored |

#### Example

```yaml
- name: "Pause for confirmation"
  pause:
    prompt: "Press enter to continue with deployment"

- name: "Wait before next step"
  pause:
    seconds: 30
```

### fail

Fail the task with a message.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `msg` | string | `Failed as requested` | Failure message |

#### Example

```yaml
- name: "Fail if conditions not met"
  fail:
    msg: "Database connection failed"
  when: "database_check.rc != 0"
```

## Complete Module List

63 module names, 61 modules:

- **Execution**: command, shell, script
- **Files**: file, copy, fetch, find, stat, slurp, template, archive, unarchive
- **Configuration**: config, lineinfile, blockinfile, replace, ini_file
- **Packages**: package, apt, apt_repository, apt_key, yum, pip
- **Services and scheduling**: service, systemd, cron
- **System control**: sysctl, reboot, mount, timezone, hostname
- **Network**: uri, get_url
- **Users and access**: user, group, authorized_key
- **Firewall**: firewall, ufw
- **Version control**: git
- **Containers**: docker_container, docker_image, docker_compose, podman, docker_host_info
- **Databases**: mysql_db, mysql_user, postgresql_db, postgresql_user, mongodb
- **Playbook control**: ping, debug, set_fact, assert, fail, pause, wait_for, include_vars, include_role, import_role, setup, gather_facts, getent, meta

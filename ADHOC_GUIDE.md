# Ad-hoc Commands

`onigirazu run` runs one module on a set of hosts without a playbook.

```bash
onigirazu run <host-pattern> [command] -i <inventory> [flags]
```

`-i` is required. The host pattern is `all`, a group or a host name. The exit code is 1
when any host failed.

## Ways to give the command

| Form | Example | Runs |
|------|---------|------|
| `-m` with `key=value` args | `run all -m package name=nginx state=present` | the named module |
| plain command (default) | `run all "uptime"` | `command` module |
| natural language | `run web "install nginx package"` | `package`, `service` or `file` |
| `module:key=value,…` | `run all "service:name=nginx,state=started"` | the named module |
| JSON | `run all '{"module":"file","args":{"path":"/tmp/x","state":"touch"}}'` | the named module |
| YAML | `run all $'module: file\nargs:\n  path: /tmp/x\n  state: touch'` | the named module |

Without `-m` the string is tried in this order: JSON (starts with `{`), YAML (contains
`module:` or `args:`), `module:args` (contains `:` and no space), natural language, and
otherwise a plain command.

### -m and arguments

```bash
onigirazu run all -m ping -i inventory.yml
onigirazu run web -m service name=nginx state=restarted -i inventory.yml
onigirazu run all -m shell cmd="df -h | grep /dev" -i inventory.yml
onigirazu run all -m file path=/tmp/test mode=0644 state=touch -i inventory.yml
```

Each argument after the pattern is one `key=value`; `-a key=value` does the same.

### Plain commands

A string that matches no other form runs through the `command` module: it is split into
words and no shell syntax is interpreted. For pipes, redirection or `&&` use
`-m shell cmd="…"`.

### Natural language

The words are lower-cased (so are paths). Recognised forms:

| Words | Result |
|-------|--------|
| `install\|add NAME package` | `package name=NAME state=present` |
| `remove\|uninstall\|delete NAME package` | `package name=NAME state=absent` |
| `update\|upgrade NAME package` | `package name=NAME state=latest` |
| `start\|stop\|restart\|reload NAME service` | `service name=NAME state=started\|stopped\|restarted\|reloaded` |
| `create\|touch file PATH` | `file path=PATH state=touch` |
| `delete file PATH` | `file path=PATH state=absent` |

Anything else (for example `remove file /tmp/x` or `install the mysql package`) is run as
a plain command. Prefer `-m` when in doubt.

### module:args

`package:name=nginx,state=present`. No spaces are allowed; a value with a space makes the
string a plain command.

## Output

`-o text` (default), `json`, `yaml` or `table`.

```json
{
  "total": 1, "success": 1, "failed": 0, "changed": 1, "skipped": 0,
  "duration": "10.5ms",
  "results": [
    {"host": "web1", "status": "success", "changed": true, "message": "root", "duration": "10.2ms"}
  ]
}
```

`status` is `success`, `failed` or `skipped`; `message` is the command's stdout, or the
module's `msg`; failed hosts carry `error`. YAML has the same fields.

Current limitation: log lines are printed to stdout before the result. Strip them before
parsing:

```bash
onigirazu run all -m ping -i inventory.yml -o json | sed -n '/^{/,$p' | jq -r '.results[] | select(.status=="success") | .host'
```

## Flags

| Flag | Meaning |
|------|---------|
| `-m, --module` | module name |
| `-a, --args key=value` | module argument (repeatable) |
| `-f, --parallel N` | hosts in parallel, default 10 (0 or less means 5) |
| `-o, --output` | `text`, `json`, `yaml`, `table` |
| `-u, --user` | SSH user for all hosts |
| `-k, --key-file` | SSH private key for all hosts |
| `-V, --verbose-mode` | detailed results |
| `--lenient` | skip invalid inventory entries |
| `--no-color` | plain output |

Current limitations of `run`:

- `{{ }}` in arguments is not rendered; for templated arguments write a one-task
  playbook. (`--check` runs only modules that support check mode and skips the others,
  `--diff` shows file changes, `-e` and `--timeout` apply.)
- There is no become option and the security policy is not applied; use a playbook with
  `become: true` / `apply -b`.
- Inline inventories (`-i "host1,host2"`) are not supported; put the hosts in a file.

## Compared with Ansible

| Ansible | Onigirazu |
|---------|-----------|
| `ansible all -i inv -m ping` | `onigirazu run all -i inv -m ping` |
| `ansible all -i inv -a "uptime"` | `onigirazu run all -i inv "uptime"` |
| `ansible all -i inv -m shell -a "df -h \| head"` | `onigirazu run all -i inv -m shell cmd="df -h \| head"` |
| `ansible all -i inv -m package -a "name=nginx state=present" -b` | playbook + `onigirazu apply -b` (no become in `run`) |

In Onigirazu `-a` takes one `key=value` per flag, not a free-form string.

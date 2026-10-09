# Verify: checks of the hosts' state

A play may end with `verify:`, a list of checks of what the hosts should look like — files,
packages, services, ports, processes, users, groups, commands, http endpoints, mounts, kernel
parameters, dns — in the spirit of goss. `apply` runs them after the play's tasks and handlers, on
every host, in one round trip; a failed check fails the host like a failed task. `onigirazu verify
PLAYBOOK` runs only the checks and reports them.

```yaml
- hosts: web
  become: true
  tasks:
    - apt: {name: nginx}
    - template: {src: nginx.conf.j2, dest: /etc/nginx/nginx.conf, mode: "0644"}
      notify: reload nginx
  handlers:
    - name: reload nginx
      service: {name: nginx, state: reloaded}
  verify:
    - package: {name: nginx, installed: true}
    - service: {name: nginx, running: true, enabled: true}
    - file: {path: /etc/nginx/nginx.conf, mode: "0644", owner: root, contains: ["worker_processes", "/^user +www-data;/"]}
    - port: {port: 80, listening: true}
    - http: {url: http://127.0.0.1/, status: 200, body: ["nginx"]}
    - command: {cmd: "nginx -t", exit_status: 0, stderr: ["syntax is ok"]}
```

```bash
onigirazu verify site.yml -i hosts.yml          # exit 1 when a check fails
onigirazu verify site.yml -i hosts.yml --format json
```

```
web1: 6 passed, 0 failed
web2: 5 passed, 1 failed
  ✗ port 80: not listening
1 of 12 check(s) failed
```

## Checks

| Kind | Arguments |
|---|---|
| `file` | `path`; `exists` (true); `type` file/directory/link; `mode` (`"0644"`); `owner`; `group`; `contains` (list); `target` (of a link) |
| `package` | `name`; `installed` (true); `version` (prefix) |
| `service` | `name`; `running`; `enabled` (systemd, `service` as fallback for running) |
| `port` | `port`; `listening` (true); `proto` tcp/udp; `ip` (a bound address) |
| `process` | `name` (as `pgrep -x`); `running` (true) |
| `user` | `name`; `exists` (true); `uid`; `gid`; `home`; `shell`; `groups` (list, all must hold) |
| `group` | `name`; `exists` (true); `gid` |
| `command` | `cmd` (run with `sh -c`); `exit_status` (0); `stdout`, `stderr` (lists); `timeout` (30 s) |
| `http` | `url` (from the host, with curl); `status` (200); `body` (list); `insecure`; `timeout` (10 s) |
| `mount` | `path`; `exists` (true); `type`; `opts` (list) |
| `kernel_param` | `name`; `value` |
| `dns` | `name`; `resolves` (true); `addrs` (list, every one expected) |

`contains`, `stdout`, `stderr`, `body` patterns are plain strings, or `/regexp/` (extended). All
checks of a play travel to the host as one shell script and come back together: a host with 100
checks costs one command. Checks run with the play's `become`; a `command` runs as that user.

The result of the play's `verify` task holds `checks` (each with `check`, `kind`, `ok`, `detail`),
`passed` and `failed`, so `register`-less reporting (JSON output, drift, audit) sees every check.
Check mode (`--check`, `plan`) runs the checks too: they change nothing.

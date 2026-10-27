# Run Once and Delegate To Guide

This guide covers the `run_once`, `delegate_to` and `local_action` task keywords.

## Overview

- **`run_once`**: run a task on one host instead of every host of the play
- **`delegate_to`**: run a task's module on another machine, on behalf of the current host
- **`local_action`**: shorthand for `delegate_to: localhost`

## run_once

### Purpose

Use `run_once` for work that must happen once per play, not once per host:

- Database migrations
- One-time setup or notifications
- Generating a shared report or file

### Syntax

```yaml
tasks:
  - name: Run database migration
    shell: cd /opt/app && python manage.py migrate
    run_once: true
```

`run_once` accepts `true`/`false` and `yes`/`no`.

### Behavior

- The task runs on the **first host of the play's host list** (after `--limit`).
- It is not executed on the other hosts, and no result is recorded for them.
- If the task has `register`, the other hosts get the same registered value.
- Loops, `when` and `delegate_to` apply as usual on that one host.
- Inside a `block`, the task still runs once for all hosts of the play: the first host to reach it runs it, the others wait and take its registered result (and its failure, if it failed).
- With `serial`, "the play" is the current batch: the task runs once per batch, on its first host.
- On `include_tasks` and `include_role`, as in Ansible, the whole include runs on that one host: its `when` is that host's, and its tasks are not `run_once` themselves, so what they register stays on that host.
- On `import_tasks`, `import_role` and a `block`, `run_once` passes on to each of their tasks, so each one registers for all hosts. A block with `run_once` runs on the first host, and the other hosts get what it registered.
- An include with `run_once` inside a block runs on the first host to reach it, as `run_once` tasks do there.
- Unlike Ansible, `run_once: false` on a task does not undo a `run_once` of its block or imported role, and an imported role's dependencies do not get it.

### Example: Database Migration

```yaml
- name: Deploy Application
  hosts: all
  tasks:
    - name: Copy application archive
      copy:
        src: app.tar.gz
        dest: /opt/app.tar.gz

    - name: Run database migrations
      shell: cd /opt/app && python manage.py migrate
      run_once: true
      register: migrate

    - name: Start application service
      service:
        name: myapp
        state: started
```

With hosts `[host1, host2, host3]`:

```
Task: Copy archive    → host1, host2, host3
Task: DB migrations   → host1 only; host2 and host3 get `migrate` too
Task: Start service   → host1, host2, host3
```

## delegate_to

### Purpose

Use `delegate_to` when a task for a host must run somewhere else:

- Load balancer API calls from the control machine
- Registering the host in a central system
- Commands that need access only a management host has

### Syntax

```yaml
tasks:
  - name: Remove host from load balancer
    uri:
      url: "https://lb.example.com/api/server/{{ inventory_hostname }}"
      method: DELETE
    delegate_to: localhost
```

### Behavior

- The task still runs once for **each** host of the play; only the module runs on the delegate.
- The task is rendered with the **current host's** variables (`inventory_hostname`, its facts and vars). The delegate's inventory entry is used only to connect to it.
- The result (and `register`) is recorded for the current host.
- `delegate_to` can be a template, e.g. `delegate_to: "{{ groups['lb'][0] }}"`.

### Host Resolution

1. `localhost`, `127.0.0.1` or `::1`: runs on the control machine with a local connection; no inventory entry is needed.
2. An inventory host whose **name** matches exactly: connects with that host's address, port, user and other connection settings.
3. Anything else: used as an address, connected on port 22 with the default connection settings. There is no fallback to the original host; if the address is unreachable, the task fails.

### Example: Load Balancer Updates

```yaml
- name: Update web servers
  hosts: webservers
  tasks:
    - name: Remove from load balancer
      uri:
        url: https://lb.example.com/api/server/remove
        method: POST
        body_format: json
        body:
          server: "{{ inventory_hostname }}"
      delegate_to: localhost

    - name: Update application
      copy:
        src: app.tar.gz
        dest: /opt/app.tar.gz

    - name: Restart application
      service:
        name: webapp
        state: restarted

    - name: Add back to load balancer
      uri:
        url: https://lb.example.com/api/server/add
        method: POST
        body_format: json
        body:
          server: "{{ inventory_hostname }}"
      delegate_to: localhost
```

Tasks run one after another, each on all hosts in parallel:

```
Task: Remove from LB   → localhost, once for web1, web2, web3
Task: Update app       → web1, web2, web3
Task: Restart app      → web1, web2, web3
Task: Add to LB        → localhost, once for web1, web2, web3
```

All hosts leave the load balancer at the same time. For a one-host-at-a-time rollout set `serial: 1` on the play: the whole play then runs host by host. Add `max_fail_percentage: 0` (or `any_errors_fatal: true`) so that a failed host stops the rollout.

## local_action

`local_action` sets the module and `delegate_to: localhost` in one keyword. It takes a string or a map:

```yaml
- name: Record deployment locally
  local_action: shell echo "{{ inventory_hostname }} deployed" >> /tmp/deploy.log

- name: Same with a map
  local_action:
    module: copy
    content: "{{ inventory_hostname }}\n"
    dest: /tmp/last-host.txt
```

## Combining run_once and delegate_to

```yaml
- name: Deploy Application
  hosts: all
  tasks:
    - name: Copy archive
      copy:
        src: app.tar.gz
        dest: /opt/app.tar.gz

    - name: Send deployment notification
      uri:
        url: https://hooks.example.com/deploy
        method: POST
        body_format: json
        body:
          text: "Deployed {{ app_version }}"
      delegate_to: localhost
      run_once: true
```

The notification is sent once, from the control machine, rendered with the first host's variables.

```
Task: Copy archive          → host1, host2, host3
Task: Send notification     → localhost, once (for host1)
```

## Common Patterns

### Database-only task

```yaml
- name: Migrate database (once)
  shell: cd /opt/app && python manage.py migrate
  run_once: true
```

### Central registration

```yaml
- name: Register host with config server
  uri:
    url: https://config-server.example.com/register
    method: POST
    body_format: json
    body:
      hostname: "{{ inventory_hostname }}"
      ip_address: "{{ ansible_default_ipv4.address }}"
  delegate_to: config-manager
```

`config-manager` must be an inventory host name (or a reachable address).

## Troubleshooting

### Delegated task fails to connect

The delegate name is not an inventory host name, so it was used as an address with default settings. Check the exact name:

```bash
onigirazu inventory -i inventory.yml --list
```

### run_once runs on every host

`run_once` must be at task level, not under the module arguments:

```yaml
# Wrong
- name: Task
  shell:
    cmd: echo test
    run_once: true

# Correct
- name: Task
  shell:
    cmd: echo test
  run_once: true
```

### run_once picked an unexpected host

It uses the first host of the play after `--limit`. To choose the host, delegate instead: `run_once: true` with `delegate_to: <host>`.

## See Also

- [Loops Guide](LOOPS_GUIDE.md)
- [Handlers Guide](HANDLERS_GUIDE.md)
- [Variables Cheatsheet](VARIABLES_CHEATSHEET.md)

# Run Once and Delegate To Guide

This guide explains how to use `run_once` and `delegate_to` task directives in Onigirazu.

## Overview

These two Ansible-compatible features allow you to control how tasks are distributed and executed across your inventory:

- **`run_once`**: Execute a task only once, on the first host in your inventory
- **`delegate_to`**: Execute a task on a different host than the one it's targeting

## run_once

### Purpose

Use `run_once` when you have a task that should execute only once, regardless of the number of hosts in your inventory. This is useful for:

- Database migrations
- One-time setup tasks
- Configuration changes that only need to happen once
- Generating reports or state files

### Syntax

```yaml
tasks:
  - name: "Run database migration"
    shell:
      cmd: "python manage.py migrate"
    run_once: true
```

### Behavior

When `run_once: true` is set:

- The task executes **only on the first host** in the host list
- All other hosts skip this task
- The result is only reported for the first host

### Example: Database Migration

```yaml
- name: "Deploy Application"
  hosts: "all"
  tasks:
    - name: "Copy application files"
      copy:
        src: "./app/"
        dest: "/opt/app/"

    - name: "Run database migrations"
      shell:
        cmd: "cd /opt/app && python manage.py migrate"
      run_once: true  # Only run once, on first host

    - name: "Start application service"
      service:
        name: "myapp"
        state: "started"
```

In this example:

- All 3 hosts copy application files
- Only host1 runs database migrations
- All 3 hosts start the service

### Execution Flow

Given 3 hosts: [host1, host2, host3]

```
Task: Copy files      → host1 ✓, host2 ✓, host3 ✓
Task: DB Migrations   → host1 ✓, host2 ⊘, host3 ⊘  (run_once)
Task: Start service   → host1 ✓, host2 ✓, host3 ✓
```

## delegate_to

### Purpose

Use `delegate_to` when you need to execute a task on a specific host instead of the target host. This is useful for:

- Load balancer API calls (execute on localhost or management host)
- Monitoring/alerting notifications (send to monitoring system)
- Centralized logging (log events to central server)
- Running tasks that require access to specific infrastructure

### Syntax

```yaml
tasks:
  - name: "Notify load balancer"
    uri:
      url: "https://lb.example.com/api/enable"
      method: "POST"
    delegate_to: "localhost"
```

### Behavior

When `delegate_to: "hostname"` is set:

- The task executes on the specified host instead of the target host
- The task still runs for each host in the inventory, but execution happens on the delegated host
- The task can access variables from the original host context

### Host Resolution

Onigirazu matches the delegate target by:

1. **Host name** - Exact match of the `name` field
2. **Host address** - Exact match of the `address` field
3. **Fallback** - If no match is found, the task executes on the original target host

### Example: Load Balancer Updates

```yaml
- name: "Rolling Update"
  hosts: "webservers"
  serial: 1  # One host at a time
  tasks:
    - name: "Remove from load balancer"
      uri:
        url: "https://lb.example.com/api/server/remove"
        method: "POST"
        body_format: "json"
        body:
          server: "{{ inventory_hostname }}"
      delegate_to: "localhost"

    - name: "Update application"
      copy:
        src: "./app/"
        dest: "/opt/app/"

    - name: "Start application"
      service:
        name: "webapp"
        state: "started"

    - name: "Add back to load balancer"
      uri:
        url: "https://lb.example.com/api/server/add"
        method: "POST"
        body_format: "json"
        body:
          server: "{{ inventory_hostname }}"
      delegate_to: "localhost"
```

### Execution Flow

Given hosts: [web1, web2, web3] with management host localhost:

```
Target: web1
  ├─ Task: Remove from LB   → localhost (delegated) ✓
  ├─ Task: Update app       → web1 ✓
  ├─ Task: Start service    → web1 ✓
  └─ Task: Add to LB        → localhost (delegated) ✓

Target: web2
  ├─ Task: Remove from LB   → localhost (delegated) ✓
  ├─ Task: Update app       → web2 ✓
  ├─ Task: Start service    → web2 ✓
  └─ Task: Add to LB        → localhost (delegated) ✓

Target: web3
  ├─ Task: Remove from LB   → localhost (delegated) ✓
  ├─ Task: Update app       → web3 ✓
  ├─ Task: Start service    → web3 ✓
  └─ Task: Add to LB        → localhost (delegated) ✓
```

## Combining run_once and delegate_to

You can use both directives together for tasks that should run once and on a specific host.

### Example: Send Deployment Notification

```yaml
- name: "Deploy Application"
  hosts: "all"
  tasks:
    - name: "Copy files"
      copy:
        src: "./app/"
        dest: "/opt/app/"

    - name: "Send deployment notification"
      uri:
        url: "https://slack.com/api/chat.postMessage"
        method: "POST"
        headers:
          Authorization: "Bearer {{ slack_token }}"
        body_format: "json"
        body:
          channel: "#deployments"
          text: "✅ App deployment completed: {{ app_version }}"
      delegate_to: "localhost"
      run_once: true  # Send notification only once, from localhost
```

In this example:

- All hosts copy the application files
- The Slack notification is sent **only once** by **localhost**
- Result: Single notification sent, not one per host

### Execution Flow

Given 3 hosts with `run_once` + `delegate_to`:

```
Task: Copy files           → host1 ✓, host2 ✓, host3 ✓
Task: Send notification    → localhost ✓  (run_once + delegate_to)
```

## Common Patterns

### Pattern 1: Database-Only Tasks

```yaml
- name: "Migrate database (once only)"
  shell:
    cmd: "python manage.py migrate"
  run_once: true
```

### Pattern 2: Centralized Logging

```yaml
- name: "Log deployment event"
  uri:
    url: "https://logging.example.com/api/events"
    method: "POST"
    body:
      event_type: "deployment"
      timestamp: "{{ now() }}"
  delegate_to: "logging-server"
```

### Pattern 3: Load Balancer Integration

```yaml
- name: "Update load balancer"
  uri:
    url: "https://lb.example.com/api/servers"
    method: "PUT"
    body:
      action: "{{ action }}"
      server: "{{ inventory_hostname }}"
  delegate_to: "lb-manager"
```

### Pattern 4: Monitoring & Alerting

```yaml
- name: "Alert monitoring system"
  uri:
    url: "https://monitoring.example.com/api/alert"
    method: "POST"
    body:
      alert_type: "deployment"
      host: "{{ inventory_hostname }}"
      status: "in_progress"
  delegate_to: "monitoring-agent"
```

## Important Notes

### Host Resolution

- If `delegate_to: "hostname"` doesn't match any host in your inventory, the task still executes but on the original target host
- The delegated host must exist in your inventory
- Delegated tasks still have access to the original target host's variables

### With run_once

When both `run_once: true` and `delegate_to: "host"` are set:

1. Task runs only on the first host in the list (run_once)
2. Execution is delegated to the specified host (delegate_to)
3. Result shows which host it executed on

### Variable Access

In delegated tasks, you can access:

- Variables from the original target host via `inventory_hostname`
- Host-specific variables from both the delegated and original host
- Global and play variables

### Example with Variables

```yaml
- name: "Register host with config server"
  uri:
    url: "https://config-server.example.com/register"
    method: "POST"
    body:
      hostname: "{{ inventory_hostname }}"  # Original target host
      ip_address: "{{ ansible_host }}"
      environment: "{{ environment }}"
  delegate_to: "config-manager"
```

## Troubleshooting

### Issue: Delegated host not found

**Symptom**: Task executes on original host instead of delegated host

**Solution**: Verify the delegated host name/address matches exactly with an entry in your inventory

```bash
# Check your inventory
onigirazu inventory -i inventory.yml

# Verify host names
```

### Issue: run_once not working

**Symptom**: Task runs on multiple hosts

**Solution**: Ensure `run_once: true` is at the task level, not under `args`

```yaml
# ❌ WRONG - run_once inside args
tasks:
  - name: "Task"
    shell:
      cmd: "echo test"
      run_once: true  # Wrong location

# ✅ CORRECT - run_once at task level
tasks:
  - name: "Task"
    shell:
      cmd: "echo test"
    run_once: true  # Correct location
```

## See Also

- [Playbook Guide](./README.md)
- [Task Directives](./LOOPS_GUIDE.md)
- [Execution Model](./ARCHITECTURE_DIAGRAM.md)
- [Variables Guide](./VARIABLES_CHEATSHEET.md)

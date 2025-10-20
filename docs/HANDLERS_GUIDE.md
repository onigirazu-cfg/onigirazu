# Handlers Guide

## Overview

Handlers are special tasks that run only when triggered by other tasks through the `notify` directive. They are useful for actions that should only be executed if something changed, such as restarting a service after a configuration change.

## Basic Concepts

### What are Handlers?

Handlers are tasks that:

- Execute conditionally (only when triggered)
- Run after all tasks in a play/role complete
- Deduplicate automatically (run once even if triggered multiple times)
- Support the `listen` directive for semantic grouping

### When to Use Handlers

Handlers are perfect for:

- **Restarting services** after configuration changes
- **Reloading configurations** without downtime
- **Database migrations** after code deployment
- **Log rotation** or cleanup tasks
- **Health checks** after infrastructure changes

## Basic Handler Syntax

```yaml
---
- name: Configure web server
  hosts: webservers

  tasks:
    - name: Update nginx config
      template: src=nginx.conf dest=/etc/nginx/nginx.conf
      notify: restart nginx

  handlers:
    - name: restart nginx
      service: name=nginx state=restarted
```

### Task Triggering Handler

Use `notify` directive to trigger a handler:

```yaml
tasks:
  - name: Update configuration
    template: src=config.j2 dest=/etc/app/config.conf
    notify: restart app
```

**Important**: `notify` can be a string (single handler) or list:

```yaml
notify: restart app

# or

notify:
  - restart app
  - reload config
```

## Listen Directive

The `listen` directive allows a handler to respond to semantic event names, enabling multiple handlers to respond to the same logical event.

### Basic Listen Example

```yaml
handlers:
  - name: reload nginx
    service: name=nginx state=reloaded
    listen: web service updated

  - name: restart nginx
    service: name=nginx state=restarted
    listen: web service updated
```

Now tasks can trigger both handlers with a single notification:

```yaml
tasks:
  - name: Update nginx
    template: src=nginx.conf dest=/etc/nginx/nginx.conf
    notify: web service updated
```

### Listen Benefits

1. **Semantic Grouping** - Organize related handlers by logical event
2. **Flexibility** - Add/remove handlers without changing task notify directives
3. **Readability** - Intent is clearer than handler names
4. **Maintainability** - Easier to understand playbook flow

## Advanced Patterns

### Multiple Handlers on Same Event

```yaml
tasks:
  - name: Deploy application
    git: repo=https://github.com/example/app dest=/opt/app
    notify: deployment complete

handlers:
  - name: run database migrations
    shell: cd /opt/app && python manage.py migrate
    listen: deployment complete

  - name: collect static files
    shell: cd /opt/app && python manage.py collectstatic --noinput
    listen: deployment complete

  - name: restart app
    service: name=myapp state=restarted
    listen: deployment complete
```

**Result**: All three handlers execute in order when deployment task completes.

### Mixed Direct and Listen Matching

```yaml
tasks:
  - name: Update config
    template: src=app.conf dest=/etc/app/app.conf
    notify:
      - restart app service        # Direct match
      - system maintenance event   # Listen match

handlers:
  - name: restart app service
    service: name=app state=restarted
    # No listen - direct name match

  - name: check logs
    shell: tail -100 /var/log/app.log
    listen: system maintenance event

  - name: cleanup temp files
    shell: find /tmp -name "app_*" -delete
    listen: system maintenance event
```

### Conditional Handler Execution

Handlers support `when` condition just like tasks:

```yaml
handlers:
  - name: restart service
    service: name=app state=restarted
    when: inventory_hostname in groups['production']
    listen: app updated
```

### Handler with Error Handling

```yaml
handlers:
  - name: run tests
    shell: cd /opt/app && npm test
    listen: code deployed
    ignore_errors: true  # Don't fail play if tests fail
```

## Execution Order and Guarantees

### Key Principles

1. **Tasks execute first** - In the order defined
2. **Handlers collect** - As tasks complete successfully
3. **Handlers deduplicate** - Same handler runs once even if triggered multiple times
4. **Handlers execute** - After all tasks and post-tasks complete
5. **Execution order** - Handlers run in definition order (top to bottom)

### Execution Flow Example

```yaml
---
- name: Example flow
  hosts: all

  tasks:
    - name: Task 1
      debug: msg="Task 1"
      notify: event A

    - name: Task 2
      debug: msg="Task 2"
      notify: event A          # Same event

    - name: Task 3
      debug: msg="Task 3"
      notify: event B

  post_tasks:
    - name: Post-task 1
      debug: msg="Post-task 1"

  handlers:
    - name: Handler A
      debug: msg="Handler A"
      listen: event A         # Runs once, triggered by Task 1 AND Task 2

    - name: Handler B
      debug: msg="Handler B"
      listen: event B         # Runs after Handler A
```

**Execution order**:

1. Task 1 (triggers event A)
2. Task 2 (triggers event A again, but duplicates are ignored)
3. Task 3 (triggers event B)
4. Post-task 1
5. Handler A (deduped - runs once)
6. Handler B

## Common Patterns

### Service Restart Pattern

```yaml
tasks:
  - name: Update application
    copy: src=app.jar dest=/opt/app/app.jar
    notify: restart java app

handlers:
  - name: restart java app
    service: name=java_app state=restarted
    listen: restart java app
```

### Configuration Reload Pattern

```yaml
tasks:
  - name: Update firewall rules
    copy: src=firewall.rules dest=/etc/iptables/rules.v4
    notify: reload firewall

handlers:
  - name: reload firewall
    command: /sbin/iptables-restore < /etc/iptables/rules.v4
    listen: reload firewall
```

### Multi-Service Update Pattern

```yaml
tasks:
  - name: Update backend config
    template: src=backend.conf dest=/etc/services/backend.conf
    notify: services updated

handlers:
  - name: reload backend
    command: systemctl reload backend
    listen: services updated

  - name: restart cache
    command: systemctl restart redis
    listen: services updated

  - name: verify services
    command: systemctl status backend redis
    listen: services updated
```

### Deployment Pattern

```yaml
tasks:
  - name: Deploy code
    git: repo={{ repo_url }} dest=/opt/app version={{ deploy_version }}
    notify: deployment complete

  - name: Update dependencies
    command: cd /opt/app && pip install -r requirements.txt
    notify: deployment complete

handlers:
  - name: run migrations
    command: cd /opt/app && python manage.py migrate
    listen: deployment complete

  - name: restart app
    service: name=django_app state=restarted
    listen: deployment complete

  - name: smoke test
    uri: url=http://localhost:8000/health method=GET
    listen: deployment complete
```

## Comparison with Ansible

| Feature | Onigirazu | Ansible | Notes |
|---------|-----------|---------|-------|
| notify directive | ✅ | ✅ | Identical |
| listen directive | ✅ | ✅ | Full support |
| Handler deduplication | ✅ | ✅ | Automatic |
| Handler ordering | ✅ | ✅ | Definition order |
| Meta: flush handlers | ❌ | ✅ | Not supported yet |
| Listen with patterns | ❌ | ⚠️ | Exact strings only |
| Handler variables | ✅ | ✅ | Same scope as tasks |

## Best Practices

### 1. Use Semantic Names with Listen

```yaml
# ✅ Good - Clear intent
handlers:
  - name: do database migration
    command: python migrate.py
    listen: deployment ready

# ❌ Less clear - Multiple handler names
handlers:
  - name: do database migration
    command: python migrate.py
```

### 2. Group Related Handlers

```yaml
# ✅ Good - Related handlers grouped together
handlers:
  - name: restart nginx
    service: name=nginx state=restarted
    listen: web service changed

  - name: reload nginx config
    command: nginx -s reload
    listen: web service changed

  - name: verify nginx
    command: nginx -t
    listen: web service changed
```

### 3. Use Ignore Errors for Non-Critical Handlers

```yaml
# ✅ Good - Health check failure doesn't break deployment
handlers:
  - name: verify app
    uri: url=http://localhost/health
    ignore_errors: true
    listen: deployment complete
```

### 4. Document Complex Handler Logic

```yaml
handlers:
  - name: rolling restart
    # Custom script for graceful rolling restart
    # to avoid downtime and dropped connections
    script: scripts/rolling_restart.sh
    listen: app updated
```

### 5. Keep Handlers Independent

```yaml
# ✅ Good - Each handler is independent
handlers:
  - name: cleanup old logs
    shell: find /var/log -name "*.log.*" -mtime +7 -delete
    listen: maintenance

  - name: compact database
    shell: sqlite3 /var/db/app.db VACUUM
    listen: maintenance

# ❌ Avoid - Handlers that depend on each other
handlers:
  - name: restart app
    service: name=app state=restarted
    listen: updated

  - name: verify restart (depends on restart)
    uri: url=http://localhost/health
    listen: updated
```

## Troubleshooting

### Handler Not Executing

**Problem**: Handler defined but not running

**Check these**:

1. Is there a `notify` directive in a task?
2. Did the task complete successfully?
3. Is the handler name correct (case-sensitive)?
4. Is the `listen` directive name matching the notification?

```yaml
# Debug example
tasks:
  - name: Test
    debug: msg="Test"
    notify: test handler

handlers:
  - name: test handler
    debug: msg="Handler running"
```

### Duplicate Handler Execution

**Problem**: Handler runs multiple times

**This shouldn't happen** - Onigirazu automatically deduplicates. If you see it:

1. Check handler name matches exactly (case-sensitive)
2. Verify no name conflicts with different handlers
3. Check `listen` directive isn't accidentally duplicating

### Handler Variable Scope

**Problem**: Variables not available in handler

**Solution**: Handlers inherit play variables:

```yaml
vars:
  app_name: myapp
  app_port: 8080

handlers:
  - name: restart app
    service: name={{ app_name }} state=restarted
    # Variables available from play level
```

## Performance Considerations

- **Handler collection**: O(n) where n = number of tasks
- **Handler matching**: O(m×k) where m = handlers, k = events
- **Typical overhead**: <1ms per playbook
- **No performance penalty** for unused handlers

## Security Considerations

Handlers have the same security model as tasks:

- Run with same privileges
- Execute in same host context
- Subject to same access controls
- Audited like regular tasks

## See Also

- [Task Documentation](tasks.md)
- [Playbook Structure](playbooks.md)
- [Variable Usage](variables.md)
- [Roles Guide](roles.md)

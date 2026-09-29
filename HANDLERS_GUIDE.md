# Handlers Guide

## Overview

Handlers are tasks that run only when another task notifies them with `notify` and that task reported a change. They are used for actions that should happen only if something changed, such as restarting a service after its configuration was updated.

## Basic Concepts

### What are Handlers?

Handlers are tasks that:

- Run only when notified by a task that reported `changed`
- Run only on the hosts that notified them
- Run at flush points (see [Execution Order](#execution-order-and-guarantees)), not immediately
- Run once per flush, even if notified many times
- Can be matched by `name` or by `listen`

### When to Use Handlers

- Restarting or reloading services after configuration changes
- Running migrations after a code deployment
- Follow-up checks after a change

## Basic Handler Syntax

```yaml
---
- name: Configure web server
  hosts: webservers

  tasks:
    - name: Update nginx config
      template: src=nginx.conf.j2 dest=/etc/nginx/nginx.conf
      notify: restart nginx

  handlers:
    - name: restart nginx
      service: name=nginx state=restarted
```

### Task Triggering Handler

`notify` takes a handler name or a list of names:

```yaml
notify: restart app

# or

notify:
  - restart app
  - reload config
```

A task notifies only when it reports `changed`. A task that is `ok`, skipped or failed does not notify. Modules that never change anything (for example `debug`) never notify; use `changed_when` if you need to force it.

A `notify` name that matches no handler is ignored silently.

Of handlers with the same name, only the first runs when that name is notified, as in Ansible (a role included twice loads its handlers twice). Every handler that `listen`s to a notified topic runs.

## Listen Directive

`listen` lets a handler respond to an event name instead of (or in addition to) its own name, so several handlers can react to one notification.

```yaml
handlers:
  - name: validate nginx config
    command: nginx -t
    listen: web service updated

  - name: restart nginx
    service: name=nginx state=restarted
    listen: web service updated
```

```yaml
tasks:
  - name: Update nginx
    template: src=nginx.conf.j2 dest=/etc/nginx/nginx.conf
    notify: web service updated
```

A handler runs if a notification matches its `name` or its `listen` value.

`listen` takes one topic or a list of topics (`listen: [restart web, reload config]`).

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

All three handlers run, in the order they are defined, when the git task reports a change.

### Mixed Direct and Listen Matching

```yaml
tasks:
  - name: Update config
    template: src=app.conf.j2 dest=/etc/app/app.conf
    notify:
      - restart app service        # matches a handler name
      - system maintenance event   # matches a listen value

handlers:
  - name: restart app service
    service: name=app state=restarted

  - name: check logs
    shell: tail -100 /var/log/app.log
    listen: system maintenance event
```

### Handlers Notifying Handlers

A handler that reports a change can notify other handlers with `notify`. They run in a further pass of the same flush (at most 10 passes; a longer chain fails with "handlers kept notifying each other").

### Conditional Handler Execution

Handlers support `when`, evaluated per host:

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
    ignore_errors: true
```

## Execution Order and Guarantees

### Play Order and Flush Points

A play runs in this order; notified handlers are flushed at each marked point:

1. `pre_tasks`
2. **flush handlers**
3. `roles`, then `tasks`
4. **flush handlers**
5. `post_tasks`
6. **flush handlers**

You can also flush at any point with `meta: flush_handlers`.

### Key Principles

1. A task that changed something records a notification for its host
2. At a flush, each notified handler runs once, on the hosts that notified it
3. Handlers run in definition order (play handlers first, then role handlers in the order the roles ran)
4. A flush forgets the notifications it handled; a handler notified again later runs again at the next flush
5. Handlers are not filtered by `--tags` / `--skip-tags`

### Execution Flow Example

```yaml
---
- name: Example flow
  hosts: all

  tasks:
    - name: Task 1
      command: echo one
      notify: event A

    - name: Task 2
      command: echo two
      notify: event A          # same event

    - name: Task 3
      command: echo three
      notify: event B

  post_tasks:
    - name: Post-task 1
      command: echo post

  handlers:
    - name: Handler A
      command: echo handler A
      listen: event A

    - name: Handler B
      command: echo handler B
      listen: event B
```

`command` always reports `changed`, so every task notifies. Execution order:

1. Task 1
2. Task 2
3. Task 3
4. Handler A (once, although notified twice)
5. Handler B
6. Post-task 1

### meta Actions

| Action | Effect |
|--------|--------|
| `meta: flush_handlers` | Run the handlers notified so far, for the hosts of the task |
| `meta: end_host` | Stop the current host for the rest of the play; its pending handlers do not run |
| `meta: end_play` | Stop all hosts of the play; pending handlers do not run |
| `meta: noop` | Do nothing |

`clear_host_errors`, `refresh_inventory` and `reset_connection` are accepted and do nothing. Any other action fails the task.

```yaml
tasks:
  - name: Update config
    template: src=app.conf.j2 dest=/etc/app/app.conf
    notify: restart app

  - name: Restart now, not at the end
    meta: flush_handlers

  - name: Check the app
    uri: url=http://localhost:8000/health
```

### Handlers and Failures

- A failing handler stops the remaining handlers and fails the play, unless it has `ignore_errors: true`.
- A host whose task failed (without `ignore_errors` or a `rescue`) leaves the run, and its notified handlers do not run. With `force_handlers: true` on the play they run anyway, also when the play stops.

### Role Handlers

Handlers from a role's `handlers/main.yml` are added to the play's handlers when the role runs, with the role's variables. They run at the same flush points as play handlers, and play tasks can notify them.

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
```

### Configuration Reload Pattern

```yaml
tasks:
  - name: Update firewall rules
    copy: src=firewall.rules dest=/etc/iptables/rules.v4
    notify: reload firewall

handlers:
  - name: reload firewall
    shell: /sbin/iptables-restore < /etc/iptables/rules.v4
```

### Deployment Pattern

```yaml
tasks:
  - name: Deploy code
    git: repo={{ repo_url }} dest=/opt/app version={{ deploy_version }}
    notify: deployment complete

  - name: Update dependencies
    shell: cd /opt/app && pip install -r requirements.txt
    notify: deployment complete

handlers:
  - name: run migrations
    shell: cd /opt/app && python manage.py migrate
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
| notify directive | Yes | Yes | Only on `changed` |
| listen directive | Yes | Yes | String or list |
| Run once per flush | Yes | Yes | |
| Handler ordering | Yes | Yes | Definition order |
| meta: flush_handlers | Yes | Yes | |
| Handlers notifying handlers | Yes | Yes | Up to 10 passes |
| force_handlers | Yes | Yes | |

## Best Practices

1. **Use `listen` for semantic grouping** when several handlers react to one change.
2. **Use `ignore_errors` only for non-critical handlers** (for example reports); a failing handler stops the rest.
3. **Use `meta: flush_handlers`** when later tasks depend on a restart having happened.
4. **Order handlers deliberately**: they run in definition order, so put a verification handler after the restart it checks.

## Troubleshooting

### Handler Not Executing

Check:

1. Did the notifying task report `changed`? `ok`, skipped and failed results do not notify, and `debug` never changes.
2. Does the `notify` value match the handler `name` or `listen` exactly (case-sensitive)? Unmatched names are ignored silently.
3. Did an earlier task failure stop the play before the flush?
4. Did `meta: end_host` / `end_play` stop the host?

### Handler Runs More Than Once

A handler runs once per flush. If it is notified in `pre_tasks` and again in `tasks`, it runs at both flushes. This is expected.

### Handler Variable Scope

Handlers see play variables, host variables and registered variables of the host they run on. Role handlers also see the role's variables.

```yaml
vars:
  app_name: myapp

handlers:
  - name: restart app
    service: name={{ app_name }} state=restarted
```

## See Also

- [Handler Examples](HANDLERS_EXAMPLES.md)
- [Loops Guide](LOOPS_GUIDE.md)
- [Tag Filtering](TAG_FILTERING.md)
- [Variables Cheatsheet](VARIABLES_CHEATSHEET.md)

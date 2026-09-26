# Handler Examples

Examples of handler usage. Handlers fire only when the notifying task reports `changed`, run only on the hosts that notified them, and run at the next flush point (after pre_tasks, after roles and tasks, after post_tasks, or at `meta: flush_handlers`). See [HANDLERS_GUIDE.md](HANDLERS_GUIDE.md).

## Example 1: Web Server Configuration

Restart the web server when its configuration or certificate changes.

```yaml
---
- name: Configure web server
  hosts: webservers

  tasks:
    - name: Install nginx
      package: name=nginx state=present

    - name: Copy nginx configuration
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        backup: true
      notify: restart nginx

    - name: Copy SSL certificate
      copy:
        src: cert.pem
        dest: /etc/nginx/cert.pem
        mode: '0600'
      notify: restart nginx

    - name: Create web root
      file: path=/var/www/html state=directory

  handlers:
    - name: restart nginx
      service: name=nginx state=restarted
```

If either the configuration or the certificate changes, nginx restarts once per host.

---

## Example 2: Application Deployment

Several tasks notify one event; several handlers listen to it.

```yaml
---
- name: Deploy application
  hosts: app_servers

  vars:
    app_repo: https://github.com/example/app.git
    app_version: v2.1.0
    app_path: /opt/app
    app_user: appuser

  tasks:
    - name: Create app directory
      file: path={{ app_path }} state=directory owner={{ app_user }}

    - name: Clone/update application
      git:
        repo: "{{ app_repo }}"
        dest: "{{ app_path }}"
        version: "{{ app_version }}"
      become: true
      become_user: "{{ app_user }}"
      notify: deployment complete

    - name: Install Python dependencies
      shell: "{{ app_path }}/venv/bin/pip install -r {{ app_path }}/requirements.txt"
      become: true
      become_user: "{{ app_user }}"
      notify: deployment complete

    - name: Copy configuration
      template:
        src: app_config.j2
        dest: "{{ app_path }}/config.json"
      become: true
      become_user: "{{ app_user }}"
      notify: deployment complete

  handlers:
    - name: run database migrations
      shell: cd {{ app_path }} && venv/bin/python manage.py migrate
      become: true
      become_user: "{{ app_user }}"
      listen: deployment complete

    - name: collect static files
      shell: cd {{ app_path }} && venv/bin/python manage.py collectstatic --noinput
      become: true
      become_user: "{{ app_user }}"
      listen: deployment complete

    - name: restart application
      service: name=myapp state=restarted
      listen: deployment complete

    - name: run health checks
      uri:
        url: http://localhost:8000/health
        method: GET
        status_code: 200
      retries: 3
      delay: 2
      listen: deployment complete
```

**Key points**:

- Three tasks notify "deployment complete"; each handler still runs once per host
- Handlers run in definition order: migrations, static files, restart, health check
- A failing handler stops the remaining handlers (unless it has `ignore_errors: true`)

---

## Example 3: System Maintenance

```yaml
---
- name: System maintenance
  hosts: all

  tasks:
    - name: Configure system limits
      template:
        src: limits.conf.j2
        dest: /etc/security/limits.conf
      notify: system configuration changed

    - name: Update sysctl parameters
      sysctl:
        name: "{{ item.key }}"
        value: "{{ item.value }}"
        state: present
      loop:
        - { key: "net.core.somaxconn", value: 65536 }
        - { key: "net.ipv4.tcp_max_syn_backlog", value: 65536 }
      notify: system configuration changed

    - name: Configure log rotation
      template:
        src: logrotate.j2
        dest: /etc/logrotate.d/app
      notify: system maintenance

  handlers:
    - name: reload sysctl
      command: sysctl --system
      listen: system configuration changed

    - name: verify sysctl
      shell: sysctl -a | grep net.core.somaxconn
      listen: system configuration changed

    - name: create log backups
      shell: |
        mkdir -p /var/log/backups
        gzip -c /var/log/app.log > /var/log/backups/app-$(date +%Y%m%d-%H%M%S).log.gz
      listen: system maintenance

    - name: cleanup old logs
      shell: find /var/log/backups -name "*.gz" -mtime +30 -delete
      listen: system maintenance
```

A loop notifies if any of its items changed; the handler still runs once per host. See [LOOPS_GUIDE.md](LOOPS_GUIDE.md).

---

## Example 4: Database Configuration

```yaml
---
- name: Database configuration
  hosts: db_servers

  vars:
    db_name: production_db
    backup_path: /var/backups/db

  tasks:
    - name: Create backup directory
      file: path={{ backup_path }} state=directory mode=0700 owner=postgres

    - name: Update PostgreSQL configuration
      template:
        src: postgresql.conf.j2
        dest: /etc/postgresql/14/main/postgresql.conf
        owner: postgres
        group: postgres
        mode: '0644'
      notify: postgresql configuration changed

    - name: Update PostgreSQL HBA
      template:
        src: pg_hba.conf.j2
        dest: /etc/postgresql/14/main/pg_hba.conf
        owner: postgres
        group: postgres
        mode: '0640'
      notify: postgresql configuration changed

  handlers:
    - name: backup database
      shell: pg_dump -Fc {{ db_name }} | gzip > {{ backup_path }}/{{ db_name }}-$(date +%Y%m%d-%H%M%S).dump.gz
      become: true
      become_user: postgres
      listen: postgresql configuration changed

    - name: reload postgresql
      service: name=postgresql state=reloaded
      listen: postgresql configuration changed

    - name: verify postgresql
      command: psql -c "SELECT version();"
      become: true
      become_user: postgres
      listen: postgresql configuration changed
```

---

## Example 5: Docker Services

```yaml
---
- name: Manage Docker services
  hosts: docker_hosts

  vars:
    app_image: registry.example.com/myapp:v2.0.0
    app_dir: /opt/myapp
    app_port: 8000

  tasks:
    - name: Pull image
      docker_image:
        name: "{{ app_image }}"
        state: present
      notify: container updated

    - name: Create app directory
      file: path={{ app_dir }} state=directory

    - name: Copy docker-compose file
      template:
        src: docker-compose.yml.j2
        dest: "{{ app_dir }}/docker-compose.yml"
      notify: container updated

    - name: Copy .env file
      template:
        src: env.j2
        dest: "{{ app_dir }}/.env"
        mode: '0600'
      notify: container updated

  handlers:
    - name: recreate containers
      docker_compose:
        project_dir: "{{ app_dir }}"
        state: present
        force_recreate: true
      listen: container updated

    - name: verify container
      uri:
        url: "http://localhost:{{ app_port }}/health"
        status_code: 200
      retries: 5
      delay: 2
      listen: container updated
```

---

## Example 6: Firewall Configuration

```yaml
---
- name: Configure firewall
  hosts: firewalls

  tasks:
    - name: Allow service ports
      firewall:
        port: "{{ item }}"
        protocol: tcp
        action: allow
        state: present
      loop: ['22', '80', '443', '8000']
      notify: firewall updated

  handlers:
    - name: show firewall status
      command: ufw status verbose
      register: firewall_status
      listen: firewall updated

    - name: test connectivity
      wait_for:
        host: "{{ hostvars[item]['ansible_default_ipv4']['address'] }}"
        port: 22
        state: started
        timeout: 5
      loop: "{{ groups['app_servers'] }}"
      listen: firewall updated
```

The `firewall` module reports `changed` only when the rule set actually changed, so the handlers run only then.

---

## Example 7: Conditional Handlers

`when` on a handler is evaluated per host when the handler runs.

```yaml
---
- name: Conditional handler example
  hosts: all

  vars:
    deploy_env: production
    enable_monitoring: true

  tasks:
    - name: Update monitoring config
      template:
        src: monitoring.conf.j2
        dest: /etc/monitoring/config.conf
      notify: monitoring updated

  handlers:
    - name: restart monitoring on production
      service: name=monitoring state=restarted
      when: deploy_env == 'production'
      listen: monitoring updated

    - name: reload monitoring on staging
      service: name=monitoring state=reloaded
      when: deploy_env == 'staging'
      listen: monitoring updated

    - name: enable monitoring
      service: name=monitoring enabled=yes
      when: enable_monitoring
      listen: monitoring updated

    - name: run health checks
      uri: url=http://localhost/metrics status_code=200
      retries: 3
      when: deploy_env in ['production', 'staging']
      listen: monitoring updated
```

---

## Example 8: Error Handling and Early Flush

```yaml
---
- name: Error handling in handlers
  hosts: all

  tasks:
    - name: Deploy application
      copy: src=app.jar dest=/opt/app/app.jar
      notify: app deployed

    - name: Restart now so the check below sees the new version
      meta: flush_handlers

    - name: Check version
      uri: url=http://localhost:8000/version status_code=200

  handlers:
    - name: generate reports
      shell: cd /opt/app && ./report.sh
      listen: app deployed
      ignore_errors: true   # a failure here does not stop the handlers below

    - name: restart app
      service: name=app state=restarted
      listen: app deployed
      # no ignore_errors: a failed restart stops the play

    - name: verify app
      uri:
        url: http://localhost:8000/health
        status_code: 200
      retries: 5
      delay: 2
      listen: app deployed
```

---

## Common Handler Patterns Summary

| Pattern | Use Case | Handlers |
|---------|----------|----------|
| **Service Restart** | Config changes | Restart service |
| **Deployment** | Code updates | Migrations, then restart |
| **Maintenance** | System updates | Cleanup tasks |
| **Configuration** | Setting changes | Reload, then verify |
| **Early flush** | Later tasks need the restart | `meta: flush_handlers` |

---

## Tips for Writing Effective Handlers

1. **Use `listen` for semantic grouping** (a single string; lists are not supported)
2. **Notify only from tasks that can report `changed`**; `debug` never does
3. **Order handlers deliberately**: definition order is execution order
4. **Use `ignore_errors` only for non-critical handlers**
5. **Use `meta: flush_handlers`** when a later task depends on a handler
6. **Expect a handler to run again** if it is notified again after a flush

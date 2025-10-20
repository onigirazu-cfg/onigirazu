# Handler Examples

Ready-to-use examples demonstrating handler usage patterns.

## Example 1: Web Server Configuration

Restart web server when configuration changes.

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

    - name: Copy website files
      synchronize:
        src: website/
        dest: /var/www/html/

  handlers:
    - name: restart nginx
      service: name=nginx state=restarted

    - name: reload nginx
      command: nginx -s reload
      listen: nginx reloaded
```

**Usage**: When either nginx config or SSL cert changes, nginx restarts automatically.

---

## Example 2: Application Deployment

Multi-step deployment with multiple handlers.

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
        update: true
      become: true
      become_user: "{{ app_user }}"
      notify: deployment complete

    - name: Install Python dependencies
      pip:
        requirements: "{{ app_path }}/requirements.txt"
        virtualenv: "{{ app_path }}/venv"
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
      shell: |
        source {{ app_path }}/venv/bin/activate
        python manage.py migrate
      become: true
      become_user: "{{ app_user }}"
      listen: deployment complete

    - name: collect static files
      shell: |
        source {{ app_path }}/venv/bin/activate
        python manage.py collectstatic --noinput
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

**Key features**:

- Multiple tasks trigger "deployment complete"
- Handlers execute in order: migrations → static files → restart → health check
- Variables available in handlers
- All dependent operations complete atomically

---

## Example 3: System Maintenance

Maintenance handlers triggered by configuration changes.

```yaml
---
- name: System maintenance
  hosts: all

  tasks:
    - name: Update system packages
      package: name=* state=latest
      notify: system updated

    - name: Configure system limits
      template:
        src: limits.conf.j2
        dest: /etc/security/limits.conf
      notify: system configuration changed

    - name: Update sysctl parameters
      sysctl: name={{ item.key }} value={{ item.value }} state=present
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
    - name: update package cache
      command: apt-get update
      when: ansible_os_family == "Debian"
      listen: system updated

    - name: reload system limits
      command: sysctl -p
      listen: system configuration changed

    - name: verify sysctl
      command: sysctl -a | grep net.core.somaxconn
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

---

## Example 4: Database Maintenance

Database handler examples.

```yaml
---
- name: Database maintenance
  hosts: db_servers

  vars:
    db_name: production_db
    db_user: dbuser
    backup_path: /var/backups/db

  tasks:
    - name: Create backup directory
      file: path={{ backup_path }} state=directory mode=0700

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

    - name: Check database size
      postgresql_query:
        db: "{{ db_name }}"
        query: "SELECT pg_database.datname, ROUND(pg_database_size(pg_database.datname) / 1024 / 1024) as size_mb FROM pg_database"
      register: db_size

  handlers:
    - name: reload postgresql
      postgresql_service: state=reloaded
      become: true
      become_user: postgres
      listen: postgresql configuration changed

    - name: verify postgresql
      postgresql_query:
        db: postgres
        query: "SELECT version();"
      become: true
      become_user: postgres
      listen: postgresql configuration changed

    - name: backup database
      shell: |
        pg_dump -U {{ db_user }} -Fc {{ db_name }} | \
        gzip > {{ backup_path }}/{{ db_name }}-$(date +%Y%m%d-%H%M%S).dump.gz
      become: true
      become_user: postgres
      listen: postgresql configuration changed

    - name: vacuum database
      postgresql_query:
        db: "{{ db_name }}"
        query: "VACUUM ANALYZE;"
      become: true
      become_user: postgres
      listen: postgresql configuration changed
```

---

## Example 5: Docker Services

Container management with handlers.

```yaml
---
- name: Manage Docker services
  hosts: docker_hosts

  vars:
    docker_registry: registry.example.com
    app_image: myapp:v2.0.0
    app_container: myapp
    app_port: 8000

  tasks:
    - name: Login to Docker registry
      docker_login:
        registry: "{{ docker_registry }}"
        username: "{{ docker_user }}"
        password: "{{ docker_password }}"

    - name: Pull latest image
      docker_image:
        name: "{{ docker_registry }}/{{ app_image }}"
        state: present
      notify: container updated

    - name: Create app directory
      file: path=/opt/{{ app_container }} state=directory

    - name: Copy docker-compose file
      template:
        src: docker-compose.yml.j2
        dest: /opt/{{ app_container }}/docker-compose.yml
      notify: container updated

    - name: Copy .env file
      template:
        src: .env.j2
        dest: /opt/{{ app_container }}/.env
        mode: '0600'
      notify: container updated

  handlers:
    - name: stop container
      docker_container:
        name: "{{ app_container }}"
        state: stopped
      listen: container updated

    - name: start container
      shell: |
        cd /opt/{{ app_container }}
        docker-compose up -d
      listen: container updated

    - name: verify container
      uri:
        url: "http://localhost:{{ app_port }}/health"
        method: GET
        status_code: 200
      retries: 5
      delay: 2
      listen: container updated

    - name: show logs
      command: docker logs {{ app_container }} --tail=50
      listen: container updated
```

---

## Example 6: Firewall Configuration

Firewall rules with validation handlers.

```yaml
---
- name: Configure firewall
  hosts: firewalls

  tasks:
    - name: Install UFW
      package: name=ufw state=present

    - name: Configure firewall rules
      ufw:
        rule: "{{ item.rule }}"
        port: "{{ item.port }}"
        proto: "{{ item.proto }}"
        state: enabled
      loop:
        - { rule: 'allow', port: '22', proto: 'tcp' }
        - { rule: 'allow', port: '80', proto: 'tcp' }
        - { rule: 'allow', port: '443', proto: 'tcp' }
        - { rule: 'allow', port: '8000', proto: 'tcp' }
      notify: firewall updated

    - name: Configure iptables rules
      template:
        src: iptables.rules.j2
        dest: /etc/iptables/rules.v4
      notify: firewall updated

  handlers:
    - name: save firewall rules
      command: ufw enable
      listen: firewall updated

    - name: apply iptables rules
      shell: iptables-restore < /etc/iptables/rules.v4
      listen: firewall updated

    - name: verify firewall
      shell: ufw status verbose
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

---

## Example 7: Conditional Handlers

Using `when` conditions with handlers.

```yaml
---
- name: Conditional handler example
  hosts: all

  vars:
    environment: production
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
      when: environment == 'production'
      listen: monitoring updated

    - name: reload monitoring on staging
      shell: systemctl reload monitoring
      when: environment == 'staging'
      listen: monitoring updated

    - name: enable monitoring
      service: name=monitoring enabled=yes
      when: enable_monitoring
      listen: monitoring updated

    - name: run health checks
      uri: url=http://localhost/metrics status_code=200
      retries: 3
      when: environment in ['production', 'staging']
      listen: monitoring updated
```

---

## Example 8: Error Handling in Handlers

Handlers with error handling.

```yaml
---
- name: Error handling in handlers
  hosts: all

  tasks:
    - name: Deploy application
      copy: src=app.jar dest=/opt/app/app.jar
      notify: app deployed

  handlers:
    - name: run tests
      shell: cd /opt/app && npm test
      listen: app deployed
      ignore_errors: true

    - name: generate reports
      shell: cd /opt/app && npm run report
      listen: app deployed
      ignore_errors: true

    - name: restart app
      service: name=app state=restarted
      listen: app deployed
      # Don't ignore errors - fail if restart fails

    - name: verify app
      uri:
        url: http://localhost:8000/health
        method: GET
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
| **Deployment** | Code updates | Migrations → Restart |
| **Maintenance** | System updates | Multiple cleanup tasks |
| **Configuration** | Setting changes | Reload + verify |
| **Monitoring** | Alert config | Restart + test |
| **Database** | Data updates | Backup + optimize |

---

## Tips for Writing Effective Handlers

1. **Use `listen` for semantic grouping** - Easier to maintain
2. **Keep handlers simple** - Complex logic should be in roles/tasks
3. **Test handler execution** - Verify handlers run when expected
4. **Document handler dependencies** - Clear execution order
5. **Use `ignore_errors` wisely** - Only for non-critical operations
6. **Validate handler success** - Include verification steps
7. **Consider idempotency** - Handlers may run multiple times across inventory

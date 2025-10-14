# 🔍 Аналіз синтаксису Onigirazu vs Ansible

## 📋 Проблемні місця в трансляції

### 1. **Складний синтаксис `serial`**

**Ansible:**
```yaml
serial: >-
    {% if allow_reboot %}100
    {% else %}1000
    {% endif %}
```

**Onigirazu (проблема):**
```yaml
serial: "{{ allow_reboot | ternary('100', '1000') }}"
```

**❌ Проблема:** Onigirazu може не підтримувати `ternary` фільтр або складні Jinja2 вирази.

**✅ Рішення:**
```yaml
# Варіант 1: Використовувати умовну логіку
serial: "{{ allow_reboot | default(false) | ternary('100', '1000') }}"

# Варіант 2: Розділити на окремі play
- name: Update hosts (with reboot)
  hosts: all
  serial: 100
  when: allow_reboot | bool
  # ... tasks

- name: Update hosts (without reboot)
  hosts: all
  serial: 1000
  when: not allow_reboot | bool
  # ... tasks
```

### 2. **Модуль `package` з параметрами `apt`**

**Ansible:**
```yaml
- name: Apt upgrade
  ansible.builtin.apt:
    upgrade: full
    autoclean: true
    autoremove: true
    update_cache: true
    cache_valid_time: 3600
```

**Onigirazu (проблема):**
```yaml
- name: Apt upgrade
  package:
    upgrade: full
    autoclean: true
    autoremove: true
    update_cache: true
    cache_valid_time: 3600
```

**❌ Проблема:** Onigirazu `package` модуль може не підтримувати всі параметри `apt` модуля.

**✅ Рішення:**
```yaml
# Варіант 1: Використовувати окремі завдання
- name: Update package cache
  package:
    update_cache: true
    cache_valid_time: 3600

- name: Upgrade packages
  package:
    upgrade: full

- name: Autoclean packages
  command: apt autoclean

- name: Autoremove packages
  command: apt autoremove -y

# Варіант 2: Використовувати command модуль
- name: Full apt upgrade
  command: apt update && apt upgrade -y && apt autoclean && apt autoremove -y
```

### 3. **Модуль `find` з реєстрацією результатів**

**Ansible:**
```yaml
- name: Clean up old files in /tmp
  ansible.builtin.find:
    paths: "{{ temp_cleanup_path }}"
    file_type: file
    age: "{{ temp_cleanup_days }}d"
    recurse: yes
  register: old_temp_files

- name: Delete found old files in /tmp
  ansible.builtin.file:
    path: "{{ item.path }}"
    state: absent
  loop: "{{ old_temp_files.files }}"
```

**Onigirazu (проблема):**
```yaml
- name: Clean up old files in /tmp
  find:
    paths: "{{ temp_cleanup_path }}"
    file_type: file
    age: "{{ temp_cleanup_days }}d"
    recurse: yes
  register: old_temp_files

- name: Delete found old files in /tmp
  file:
    path: "{{ item.path }}"
    state: absent
  loop: "{{ old_temp_files.files }}"
```

**❌ Проблема:** Onigirazu може не підтримувати `find` модуль або мати інший формат результатів.

**✅ Рішення:**
```yaml
# Варіант 1: Використовувати command модуль
- name: Find old files in /tmp
  command: find {{ temp_cleanup_path }} -type f -mtime +{{ temp_cleanup_days }} -print
  register: old_temp_files
  changed_when: false

- name: Delete found old files in /tmp
  command: rm -f "{{ item }}"
  loop: "{{ old_temp_files.stdout_lines }}"
  when: old_temp_files.stdout_lines | length > 0

# Варіант 2: Використовувати shell модуль
- name: Clean up old files in /tmp
  shell: |
    find {{ temp_cleanup_path }} -type f -mtime +{{ temp_cleanup_days }} -delete
    find {{ temp_cleanup_path }} -type d -mtime +{{ temp_cleanup_days }} -empty -delete
```

### 4. **Модуль `pause`**

**Ansible:**
```yaml
- name: Pause for 1 minute to build app cache
  ansible.builtin.pause:
    minutes: 1
  when:
    - allow_reboot | bool
```

**Onigirazu (проблема):**
```yaml
- name: Pause for 1 minute to build app cache
  pause:
    minutes: 1
  when:
    - allow_reboot | bool
```

**❌ Проблема:** Onigirazu може не мати `pause` модуля.

**✅ Рішення:**
```yaml
# Варіант 1: Використовувати command модуль
- name: Pause for 1 minute to build app cache
  command: sleep 60
  when:
    - allow_reboot | bool

# Варіант 2: Використовувати shell модуль
- name: Pause for 1 minute to build app cache
  shell: sleep 60
  when:
    - allow_reboot | bool
```

### 5. **Модуль `cron` з складним job**

**Ansible:**
```yaml
- name: Cure by cronjob
  ansible.builtin.cron:
    name: "dont be so greedy"
    minute: "*/1"
    hour: "*"
    job: >-
      docker update --memory "100m" --memory-swap "300M" --cpus=0.01 $(docker ps | grep fluxFoldingAt | awk '{print $1}')
```

**Onigirazu (проблема):**
```yaml
- name: Cure by cronjob
  cron:
    name: "dont be so greedy"
    minute: "*/1"
    hour: "*"
    job: >-
      docker update --memory "100m" --memory-swap "300M" --cpus=0.01 $(docker ps | grep fluxFoldingAt | awk '{print $1}')
```

**❌ Проблема:** Onigirazu `cron` модуль може не підтримувати складні job команди.

**✅ Рішення:**
```yaml
# Варіант 1: Використовувати command модуль
- name: Cure by cronjob
  command: |
    (crontab -l 2>/dev/null; echo "*/1 * * * * docker update --memory '100m' --memory-swap '300M' --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk '{print \$1}')") | crontab -

# Варіант 2: Використовувати shell модуль
- name: Cure by cronjob
  shell: |
    echo "*/1 * * * * docker update --memory '100m' --memory-swap '300M' --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk '{print \$1}')" >> /tmp/cron_job
    crontab /tmp/cron_job
    rm /tmp/cron_job
```

### 6. **Модуль `reboot` з параметрами**

**Ansible:**
```yaml
- name: Reboot
  ansible.builtin.reboot:
    msg: "Reboot initiated by Ansible"
    connect_timeout: 5
    reboot_timeout: 600
    pre_reboot_delay: 0
    post_reboot_delay: 30
    test_command: whoami
  when:
    - allow_reboot | bool
```

**Onigirazu (проблема):**
```yaml
- name: Reboot
  reboot:
    msg: "Reboot initiated by Onigirazu"
    connect_timeout: 5
    reboot_timeout: 600
    pre_reboot_delay: 0
    post_reboot_delay: 30
    test_command: whoami
  when:
    - allow_reboot | bool
```

**❌ Проблема:** Onigirazu може не мати `reboot` модуля або мати інші параметри.

**✅ Рішення:**
```yaml
# Варіант 1: Використовувати command модуль
- name: Reboot
  command: reboot
  when:
    - allow_reboot | bool

# Варіант 2: Використовувати shell модуль з затримкою
- name: Reboot
  shell: |
    echo "Reboot initiated by Onigirazu"
    sleep 30
    reboot
  when:
    - allow_reboot | bool
```

## 🔧 Виправлений Onigirazu playbook

```yaml
---
- name: Update hosts
  hosts: all
  become: true
  serial: "{{ allow_reboot | default(false) | ternary('100', '1000') }}"
  gather_facts: false
  order: shuffle
  vars:
    allow_reboot: false
    temp_cleanup_path: "/tmp"
    temp_cleanup_days: 7

  tasks:
    - name: Update package cache
      package:
        update_cache: true
        cache_valid_time: 3600
      retries: 5
      delay: 10

    - name: Upgrade packages
      package:
        upgrade: full
      retries: 5
      delay: 10

    - name: Autoclean packages
      command: apt autoclean
      retries: 5
      delay: 10

    - name: Autoremove packages
      command: apt autoremove -y
      retries: 5
      delay: 10

    - name: Check if a reboot is required
      command: "[ -f /var/run/reboot-required ]"
      failed_when: false
      register: reboot_required
      changed_when: reboot_required.rc == 0
      notify: Reboot

    - name: Pause for 1 minute to build app cache
      command: sleep 60
      when:
        - allow_reboot | bool

    - name: Cure by cronjob
      command: |
        (crontab -l 2>/dev/null; echo "*/1 * * * * docker update --memory '100m' --memory-swap '300M' --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk '{print \$1}')") | crontab -

    - name: Clean up old files in /tmp
      command: find {{ temp_cleanup_path }} -type f -mtime +{{ temp_cleanup_days }} -delete

    - name: Clean up old directories in /tmp
      command: find {{ temp_cleanup_path }} -type d -mtime +{{ temp_cleanup_days }} -empty -delete

    - name: Clear memory and cache
      command: sync && echo 3 > /proc/sys/vm/drop_caches
      changed_when: false

  handlers:
    - name: Reboot
      command: reboot
      when:
        - allow_reboot | bool
```

## 🎯 Висновок

**Основні проблеми в Onigirazu:**

1. **❌ Відсутні модулі:** `pause`, `find`, `reboot`, `cron`
2. **❌ Обмежена підтримка:** Складні параметри `package` модуля
3. **❌ Синтаксис:** Складні Jinja2 вирази
4. **❌ Функціональність:** Деякі Ansible модулі не мають аналогів

**✅ Рішення:**
- Використовувати `command` та `shell` модулі
- Спрощувати складні вирази
- Розділяти складні завдання на прості
- Використовувати Natural Language команди для швидких операцій


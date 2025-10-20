# 📋 Onigirazu Playbook Formats - Повний Гайд

## 🎯 Огляд

Onigirazu підтримує **ДВА сумісних формати** YAML плейбуків:

1. **Структурований формат** (рекомендований для складних плейбуків)
2. **Ansible-сумісний формат** (прямий список, як в Ansible)

Обидва формати автоматично розпізнаються і обробляються ідентично! ✨

---

## 📐 Формат 1: Структурований (Рекомендований)

### Загальна структура

```yaml
name: My Playbook Name
vars:                    # (опціонально) глобальні змінні
  environment: production
  debug_mode: false
plays:
  - hosts: all
    name: Play 1 Name
    vars:               # (опціонально) змінні цього play
      play_var: value
    tasks:
      - name: Task name
        module_name:
          param: value
```

### ✅ Приклад 1: Базовий плейбук

```yaml
name: System Setup
plays:
  - hosts: all
    name: Configure Servers
    tasks:
      - name: Update package manager
        shell:
          cmd: apt-get update
      - name: Install curl
        package:
          name: curl
          state: present
```

### ✅ Приклад 2: Плейбук з глобальними змінними

```yaml
name: Web Application Deployment
vars:
  app_user: webapp
  app_home: /opt/webapp
  environment: production
plays:
  - hosts: webservers
    name: Install Dependencies
    tasks:
      - name: Install Python packages
        package:
          name:
            - python3
            - python3-pip
            - python3-venv
          state: present

  - hosts: webservers
    name: Configure Application
    tasks:
      - name: Create app user
        user:
          name: "{{ app_user }}"
          home: "{{ app_home }}"
          state: present
```

### ✅ Приклад 3: Плейбук з play-специфічними змінними

```yaml
name: Database Migration
plays:
  - hosts: databases
    name: Backup Database
    vars:
      backup_dir: /backups
      retention_days: 30
    tasks:
      - name: Create backup directory
        file:
          path: "{{ backup_dir }}"
          state: directory
          mode: '0755'
      - name: Backup database
        shell:
          cmd: "mysqldump -A > {{ backup_dir }}/backup-$(date +%s).sql"

  - hosts: databases
    name: Apply Migrations
    vars:
      migrations_dir: /app/migrations
    tasks:
      - name: Run migrations
        shell:
          cmd: "cd /app && ./migrate.sh"
```

---

## 🐧 Формат 2: Ansible-Сумісний (Прямий список)

### Загальна структура

```yaml
- hosts: all
  name: Play 1 Name
  vars:               # (опціонально) змінні цього play
    play_var: value
  tasks:
    - name: Task name
      module_name:
        param: value

- hosts: other
  name: Play 2 Name
  tasks:
    - name: Another task
      other_module:
        param: value
```

### 📌 Важливо

Коли використовуєте Ansible-формат:

- ❌ **НЕМАЄ** верхнього рівня `name:` і `plays:`
- ✅ Починається **прямо з `- hosts:`**
- ✅ Ім'я плейбука автоматично генерується з першого play
- ✅ Багато плейбуків можуть бути в одному файлі

### ✅ Приклад 1: Простий одноплейбуковий файл

```yaml
- hosts: all
  name: Package Management Demo
  tasks:
    - name: Install essential tools
      package:
        name:
          - curl
          - wget
          - git
        state: present
    - name: Remove unnecessary packages
      package:
        name: mc
        state: absent
```

### ✅ Приклад 2: Кілька плейбуків в одному файлі

```yaml
# Play 1 - Базова конфігурація
- hosts: webservers
  name: Web Server Setup
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
    - name: Start nginx
      service:
        name: nginx
        state: started

# Play 2 - Налаштування
- hosts: webservers
  name: Web Server Configuration
  tasks:
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
    - name: Reload nginx
      service:
        name: nginx
        state: reloaded

# Play 3 - На інших хостах
- hosts: databases
  name: Database Setup
  tasks:
    - name: Install PostgreSQL
      package:
        name: postgresql
        state: present
```

### ✅ Приклад 3: Ansible-формат з глобальними змінними

```yaml
# Глобальні змінні (якщо потрібні, розмістіть як варіант)
- hosts: all
  name: Initialize Environment
  vars:
    app_env: production
    log_level: info
  tasks:
    - name: Set environment variables
      shell:
        cmd: |
          export APP_ENV={{ app_env }}
          export LOG_LEVEL={{ log_level }}

- hosts: webservers
  name: Deploy Application
  tasks:
    - name: Pull latest code
      git:
        repo: https://github.com/user/app.git
        dest: /opt/app
        version: main
```

---

## 🔄 Як вони конвертуються

### Структурований → Ansible-формат

```yaml
# СТРУКТУРОВАНИЙ (введення)
name: My App Deployment
plays:
  - hosts: servers
    name: Setup
    tasks:
      - name: Install packages
        package:
          name: nginx
          state: present

# Автоматично обробляється як:
# 1. Розпізнано як структурований формат
# 2. Используются name і plays як є
# 3. Виконується однаково
```

### Ansible-формат → Структурований (автоматично)

```yaml
# ANSIBLE-ФОРМАТ (введення)
- hosts: servers
  name: Setup
  tasks:
    - name: Install packages
      package:
        name: nginx
        state: present

# Автоматично обробляється як:
# 1. Розпізнано як список play'ів
# 2. Name плейбука генерується: "Setup (playbook)"
# 3. Plays: [список з одного play'ю]
# 4. Виконується однаково
```

---

## 📊 Порівняння форматів

| Аспект | Структурований | Ansible-сумісний |
|--------|---|---|
| **Синтаксис** | `name:`, `plays:` | Прямий список `- hosts:` |
| **Глобальні змінні** | На верхньому рівні | На рівні першого play |
| **Читаність** | Більш явна структура | Як в Ansible (знайомо) |
| **Кілька плейбуків** | Тільки одноплейбуковий | Кілька `- hosts:` блоків |
| **Автоматична конверсія** | ✅ Структура зберігається | ✅ Ім'я генерується автоматично |
| **Команда `fmt`** | ✅ Форматує структуру | ✅ Залишає як є |
| **Рекомендація** | ✅ Складні плейбуки | ✅ Прості, одноплейбукові |

---

## 💡 Практичні поради

### 🎯 Коли використовувати структурований формат?

- Плейбук має багато play'ів
- Потрібні глобальні змінні для всього плейбука
- Плануєте масштабування структури
- Розробляєте на Python/Go (програмне генерування)

**Приклад:**

```yaml
name: Complete Infrastructure Setup
vars:
  organization: acme
  region: us-east-1
  environment: staging
plays:
  # ... 10+ play'ів ...
```

### 🎯 Коли використовувати Ansible-формат?

- Один простий play в файлі
- Мігруєте з Ansible playbooks
- Хочете найкоротший формат
- Плейбук легко вміщується на одному екрані

**Приклад:**

```yaml
- hosts: webservers
  name: Deploy App
  tasks:
    - name: Install nginx
      package:
        name: nginx
```

---

## ✅ Перевірка формату

### Вивчити, який формат використовується

```bash
# Структурований
onigirazu validate my-playbook.yml
# Output: Playbook: My Playbook Name

# Ansible-сумісний
onigirazu validate ansible-style.yml
# Output: Playbook: Setup (playbook)  ← автоматично згенероване ім'я
```

### Перетворити в читаблий вигляд

```bash
# Формат зберігається, як він є
onigirazu fmt my-playbook.yml
# Структурований залишається структурованим
# Ansible-сумісний залишається Ansible-сумісним
```

---

## 🚨 Частичні помилки

### ❌ Помилка 1: Змішування форматів

```yaml
# НЕПРАВИЛЬНО! Змішування обох форматів
name: Mixed Format
plays:
  - hosts: all
    name: Setup

- hosts: other
  name: Another Setup
```

**Рішення:** Виберіть ОДИН формат для файлу

### ❌ Помилка 2: Структурований без `plays`

```yaml
# НЕПРАВИЛЬНО! Забули `plays`
name: My Playbook
  - hosts: all
    name: Setup
```

**Рішення:**

```yaml
# ПРАВИЛЬНО
name: My Playbook
plays:
  - hosts: all
    name: Setup
```

### ❌ Помилка 3: Ansible-формат з `name:` на верхньому рівні

```yaml
# НЕПРАВИЛЬНО! Структурований формат потребує `plays`
name: My Playbook
- hosts: all
  name: Setup
```

**Рішення:**

```yaml
# Варіант 1: Структурований (правильно)
name: My Playbook
plays:
  - hosts: all
    name: Setup

# Варіант 2: Ansible-сумісний (правильно)
- hosts: all
  name: Setup
```

---

## 📝 Контрольний список при написанні плейбука

- [ ] Обрав ОДИН формат (структурований або Ansible-сумісний)
- [ ] Усі обов'язкові поля вказані (`name` для play, `tasks`)
- [ ] Однорівневі поля (`hosts`, `tasks`) належні до play
- [ ] Кожен task має `name:` для читаності
- [ ] Шаблони `{{ var }}` правильно оформлені
- [ ] Модулі існують (запустити `onigirazu lint`)

---

## 🧪 Тестування формату

```bash
# 1. Перевірити синтаксис
onigirazu validate playbook.yml

# 2. Перевірити якість
onigirazu lint playbook.yml

# 3. Переглянути різниці
onigirazu diff playbook.yml

# 4. Запустити в dry-run режимі (якщо підтримується)
# onigirazu apply --dry-run playbook.yml
```

---

## 📚 Інші ресурси

- [Ansible Playbooks Documentation](https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_intro.html)
- [Onigirazu CLI Reference](./CLI_REFERENCE.md)
- [YAML Syntax Guide](./YAML_SYNTAX.md)

---

**Остання оновлення:** 2025-01-29
**Версія Onigirazu:** v1.43.2+

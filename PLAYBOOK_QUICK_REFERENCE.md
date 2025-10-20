# ⚡ Playbook Formats - Швидкий довідник

## 🎯 Два формати - Один результат

```
┌─────────────────────────────────────────────────────────────────┐
│              ✅ ОБА ФОРМАТИ ПРАЦЮЮТЬ ОДНАКОВО!                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📐 ФОРМАТ 1: Структурований

```yaml
name: My Playbook Name              # ← Ім'я плейбука
vars:                               # ← Опціонально: глобальні змінні
  global_var: value
plays:                              # ← ОБОВ'ЯЗКОВО: список play'ів
  - hosts: all
    name: Play Name
    vars:                           # ← Опціонально: локальні змінні
      play_var: value
    tasks:                          # ← ОБОВ'ЯЗКОВО: список задач
      - name: Task Name
        module_name:
          parameter: value
```

### ✅ Готовий приклад

```yaml
name: System Maintenance
vars:
  maintenance_window: weekly
plays:
  - hosts: servers
    name: Update System
    tasks:
      - name: Update packages
        package:
          name: "*"
          state: latest
      - name: Remove old packages
        package:
          autoremove: yes
```

---

## 🐧 ФОРМАТ 2: Ansible-сумісний

```yaml
- hosts: all                        # ← ПОЧИНАЄТЬСЯ з - hosts
  name: Play Name
  vars:                             # ← Опціонально: локальні змінні
    play_var: value
  tasks:                            # ← ОБОВ'ЯЗКОВО: список задач
    - name: Task Name
      module_name:
        parameter: value

- hosts: other                      # ← Другий play (опціонально)
  name: Another Play
  tasks:
    - name: Another Task
      module:
        param: value
```

### ✅ Готовий приклад

```yaml
- hosts: servers
  name: System Maintenance
  tasks:
    - name: Update packages
      package:
        name: "*"
        state: latest
    - name: Remove old packages
      package:
        autoremove: yes
```

---

## 🔍 Різниці в одній таблиці

| | **Структурований** | **Ansible-сумісний** |
|---|---|---|
| **Починається з** | `name:` на верхньому рівні | `- hosts:` (список) |
| **Верхній рівень** | `name`, `plays`, `vars` | Прямо `- hosts:` блоки |
| **Глобальні vars** | На верхньому рівні | На рівні першого play |
| **Кілька plays** | Всередині `plays:` список | `- hosts:` блоки один за одним |
| **Автоген ім'я** | Зберігається як є | `"First Play Name (playbook)"` |
| **Краще для** | Складні плейбуки | Міграція з Ansible |

---

## 💾 Як Onigirazu розпізнає формат

```
┌─────────────────────────────────────┐
│  YAML файл                          │
└────────────────┬────────────────────┘
                 │
        Перша лінія?
         /        \
        /          \
   - hosts:     name:
      │            │
  Ansible-     Структ-
  сумісний     урований
```

---

## 🧪 Приклади для копіювання

### 🟢 СТРУКТУРОВАНИЙ - Складний плейбук

```yaml
name: Complete App Deployment
vars:
  environment: production
  app_version: 1.0.0
plays:
  - hosts: databases
    name: Prepare Database
    vars:
      backup_retention: 30
    tasks:
      - name: Create backup
        shell:
          cmd: "mysqldump -A > /backups/db-$(date +%s).sql"
      - name: Run migrations
        shell:
          cmd: "cd /app && ./migrate.sh"

  - hosts: webservers
    name: Deploy Application
    vars:
      deploy_user: appuser
    tasks:
      - name: Pull latest code
        git:
          repo: https://github.com/user/app.git
          dest: /opt/app
      - name: Install dependencies
        shell:
          cmd: "cd /opt/app && npm install"
      - name: Start application
        service:
          name: app
          state: restarted
```

---

### 🟢 ANSIBLE-СУМІСНИЙ - Простий плейбук

```yaml
- hosts: webservers
  name: Setup Nginx
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
    - name: Start nginx service
      service:
        name: nginx
        state: started
        enabled: yes
    - name: Copy config
      copy:
        src: ./nginx.conf
        dest: /etc/nginx/nginx.conf
      notify: restart nginx
```

---

### 🟡 ANSIBLE-СУМІСНИЙ - Кілька play'ів

```yaml
# Play 1
- hosts: webservers
  name: Web Layer Setup
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present

# Play 2
- hosts: databases
  name: Database Layer Setup
  tasks:
    - name: Install PostgreSQL
      package:
        name: postgresql
        state: present

# Play 3
- hosts: cache
  name: Cache Layer Setup
  tasks:
    - name: Install Redis
      package:
        name: redis-server
        state: present
```

---

## ⚠️ ПОМИЛКИ - ЧИМ ЇХАТИ

### ❌ НЕПРАВИЛЬНО: Змішування

```yaml
name: Wrong Format          # ← Структурований
- hosts: all               # ← Але потім Ansible стиль!
  name: Setup
```

### ✅ ПРАВИЛЬНО: Вибрати один

```yaml
# Варіант 1: Структурований
name: Correct Format
plays:
  - hosts: all
    name: Setup

# Варіант 2: Ansible-сумісний
- hosts: all
  name: Setup
```

---

### ❌ НЕПРАВИЛЬНО: Забули `plays:`

```yaml
name: My Playbook
- hosts: all              # ← Без `plays:` - помилка!
  name: Setup
```

### ✅ ПРАВИЛЬНО

```yaml
name: My Playbook
plays:
  - hosts: all
    name: Setup
```

---

## 🔧 Команди для перевірки

```bash
# Перевірити синтаксис
onigirazu validate playbook.yml

# Форматувати (зберіжає формат як є)
onigirazu fmt playbook.yml

# Перевірити якість
onigirazu lint playbook.yml

# Побачити різниці
onigirazu diff playbook.yml
```

---

## 📊 Матриця вибору формату

```
┌─────────────────────────────────────────────────────┐
│              Який формат обрати?                     │
├─────────────────────────────────────────────────────┤
│                                                      │
│  Один play?                                         │
│  ├─ ДА  → Ansible-сумісний ✅                      │
│  └─ НІ  → Структурований ✅                        │
│                                                      │
│  Глобальні змінні для всього плейбука?             │
│  ├─ ДА  → Структурований ✅                        │
│  └─ НІ  → Обидва однаково ✅                       │
│                                                      │
│  Мігруєте з Ansible?                               │
│  ├─ ДА  → Ansible-сумісний ✅                      │
│  └─ НІ  → Виберіть що зручніше ✅                 │
│                                                      │
└─────────────────────────────────────────────────────┘
```

---

## 🎓 Чекліст перед запуском

- [ ] Формат вибраний (один з двох)
- [ ] `name` присутній (для play)
- [ ] `hosts` вказаний (яких серверів)
- [ ] `tasks` присутня (список задач)
- [ ] Кожен task має `name`
- [ ] Модулі та параметри правильні
- [ ] Синтаксис YAML коректний (відступи!)
- [ ] `onigirazu validate` проходить ✅

---

## 📝 Шаблон для швидкого старту

### Структурований

```yaml
name: My Playbook
plays:
  - hosts: all
    name: Task Group 1
    tasks:
      - name: Do something
        shell:
          cmd: echo "Hello"
```

### Ansible-сумісний

```yaml
- hosts: all
  name: Task Group 1
  tasks:
    - name: Do something
      shell:
        cmd: echo "Hello"
```

---

**Запам'ятати:** Обидва формати автоматично розпізнаються! 🚀

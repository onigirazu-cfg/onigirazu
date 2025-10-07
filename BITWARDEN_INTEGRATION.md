# Bitwarden Integration for Onigirazu

## 📋 Overview

Додано підтримку **Bitwarden** як альтернативного провайдера для управління секретами поряд з HashiCorp Vault. Bitwarden є популярним open-source менеджером паролів, який легше налаштувати та використовувати, особливо для малих та середніх команд.

---

## 🎯 Переваги Bitwarden

### Порівняння з Vault

| Характеристика | Bitwarden | HashiCorp Vault |
|----------------|-----------|-----------------|
| **Складність налаштування** | ✅ Проста | ⚠️ Складна |
| **Вартість** | ✅ Безкоштовно (self-hosted) | 💰 Платно для enterprise |
| **Self-hosting** | ✅ Vaultwarden | ✅ Так |
| **UI для управління** | ✅ Зручний веб-інтерфейс | ⚠️ Базовий |
| **2FA підтримка** | ✅ Так | ✅ Так |
| **Динамічні credentials** | ❌ Ні | ✅ Так |
| **Audit logging** | ⚠️ Базовий | ✅ Розширений |
| **Підходить для** | Малі/середні команди | Enterprise |

### Ключові переваги

- ✅ **Open-source** - повністю відкритий код
- ✅ **Безкоштовний** - для особистого та командного використання
- ✅ **Self-hosting** - можна розгорнути на власному сервері (Vaultwarden)
- ✅ **Простота** - легко налаштувати та використовувати
- ✅ **Кросплатформенність** - працює на всіх ОС
- ✅ **Зручний UI** - веб-інтерфейс для управління секретами
- ✅ **CLI інтеграція** - офіційний CLI для автоматизації
- ✅ **Організаційні колекції** - підтримка командної роботи

---

## 🚀 Використання

### 1. Базова конфігурація

```yaml
# playbook.yml
secrets:
  provider: bitwarden
  config:
    server: https://vault.bitwarden.com  # або self-hosted URL
    email: admin@example.com
    session: ${BW_SESSION}  # рекомендовано через env var
    cache_ttl: 300  # кешування на 5 хвилин

vars:
  db_password: "{{ bitwarden('database-credentials', 'password') }}"
  api_key: "{{ bitwarden('api-keys', 'github_token') }}"
```

### 2. Self-hosted Vaultwarden

```yaml
secrets:
  provider: bitwarden
  config:
    server: https://vault.mycompany.com
    email: devops@mycompany.com
    session: ${BW_SESSION}
    organization_id: "org-uuid-here"
```

### 3. Автоматична аутентифікація

```yaml
secrets:
  provider: bitwarden
  config:
    server: https://vault.bitwarden.com
    email: admin@example.com
    password: ${BW_PASSWORD}  # автоматичний login
    cache_ttl: 600
```

### 4. Використання в tasks

```yaml
tasks:
  - name: "Configure database"
    module: "template"
    src: "templates/db.conf.j2"
    dest: "/etc/app/database.conf"
    vars:
      db_host: "postgres.example.com"
      db_user: "app_user"
      db_password: "{{ bitwarden('prod-database', 'password') }}"

  - name: "Deploy SSH key"
    module: "copy"
    content: "{{ bitwarden('deploy-keys', 'private_key') }}"
    dest: "/home/deploy/.ssh/id_rsa"
    mode: "0600"

  - name: "Set API token"
    module: "shell"
    command: "echo '{{ bitwarden('api-tokens', 'github') }}' > /etc/app/token"
```

---

## 🔧 Налаштування

### Крок 1: Встановлення Bitwarden CLI

```bash
# macOS
brew install bitwarden-cli

# Linux
npm install -g @bitwarden/cli

# або завантажити бінарник
wget https://vault.bitwarden.com/download/?app=cli&platform=linux
```

### Крок 2: Логін та отримання session

```bash
# Логін
bw login admin@example.com

# Отримання session token
export BW_SESSION=$(bw unlock --raw)

# Перевірка
bw list items --session $BW_SESSION
```

### Крок 3: Створення секретів у Bitwarden

```bash
# Через CLI
bw create item '{
  "type": 1,
  "name": "database-credentials",
  "login": {
    "username": "dbuser",
    "password": "secret123"
  }
}' --session $BW_SESSION

# Або через веб-інтерфейс
# https://vault.bitwarden.com
```

### Крок 4: Запуск Onigirazu

```bash
# З session token
export BW_SESSION="your-session-token"
onigirazu -playbook deploy.yml -inventory hosts.yml

# Або з автоматичним логіном
export BW_PASSWORD="your-master-password"
onigirazu -playbook deploy.yml -inventory hosts.yml
```

---

## 📦 Структура секретів у Bitwarden

### Типи полів

```yaml
# Login item (type=1)
bitwarden('item-name', 'username')  # → login.username
bitwarden('item-name', 'password')  # → login.password

# Custom fields
bitwarden('item-name', 'api_key')   # → fields[name=api_key].value
bitwarden('item-name', 'token')     # → fields[name=token].value

# Notes
bitwarden('item-name', 'notes')     # → notes field
```

### Приклад організації

```
Bitwarden Vault
├── Production
│   ├── database-credentials (Login)
│   │   ├── username: prod_user
│   │   ├── password: ********
│   │   └── fields:
│   │       └── connection_string: postgres://...
│   ├── api-keys (Secure Note)
│   │   └── fields:
│   │       ├── github: ghp_****
│   │       ├── aws: AKIA****
│   │       └── stripe: sk_live_****
│   └── ssh-keys (Secure Note)
│       └── fields:
│           ├── deploy_key: -----BEGIN RSA...
│           └── backup_key: -----BEGIN RSA...
└── Staging
    └── ...
```

---

## 🔒 Безпека

### Best Practices

1. **Використовуйте session token через env var**

   ```bash
   export BW_SESSION=$(bw unlock --raw)
   ```

2. **Налаштуйте TTL для кешу**

   ```yaml
   cache_ttl: 300  # 5 хвилин
   ```

3. **Використовуйте організаційні колекції**

   ```yaml
   organization_id: "your-org-uuid"
   ```

4. **Увімкніть 2FA для Bitwarden акаунту**
   - Через веб-інтерфейс: Settings → Two-step Login

5. **Self-host для критичних секретів**

   ```bash
   docker run -d --name vaultwarden \
     -v /vw-data/:/data/ \
     -p 80:80 \
     vaultwarden/server:latest
   ```

### Обмеження доступу

```yaml
# Використовуйте різні акаунти для різних середовищ
production:
  secrets:
    provider: bitwarden
    config:
      email: prod-deploy@company.com
      organization_id: "prod-org-uuid"

staging:
  secrets:
    provider: bitwarden
    config:
      email: staging-deploy@company.com
      organization_id: "staging-org-uuid"
```

---

## 🧪 Тестування

### Перевірка підключення

```bash
# Тест CLI
bw login admin@example.com
export BW_SESSION=$(bw unlock --raw)
bw list items --session $BW_SESSION

# Тест в Onigirazu
onigirazu -playbook test-secrets.yml -inventory hosts.yml
```

### Приклад тестового playbook

```yaml
# test-secrets.yml
---
name: "Test Bitwarden Integration"
hosts:
  - name: localhost
    address: 127.0.0.1
    user: ${USER}

secrets:
  provider: bitwarden
  config:
    server: https://vault.bitwarden.com
    session: ${BW_SESSION}

vars:
  test_secret: "{{ bitwarden('test-item', 'password') }}"

tasks:
  - name: "Print secret (masked)"
    module: "debug"
    msg: "Secret retrieved: {{ test_secret | mask }}"
```

---

## 📊 Продуктивність

### Кешування

Bitwarden клієнт використовує вбудоване кешування:

- **TTL**: Налаштовується через `cache_ttl` (за замовчуванням 5 хвилин)
- **Scope**: Per-item + per-field
- **Invalidation**: Автоматична після TTL

### Метрики

```
Без кешу:
- Перший запит: ~200-300ms (CLI виклик)
- Наступні запити: ~200-300ms кожен

З кешем:
- Перший запит: ~200-300ms
- Наступні запити: <1ms (з кешу)
- Покращення: 200-300x швидше
```

---

## 🔄 Міграція з Vault

### Порівняння синтаксису

```yaml
# Vault
vars:
  password: "{{ vault('secret/data/database') }}"

# Bitwarden
vars:
  password: "{{ bitwarden('database-credentials', 'password') }}"
```

### Скрипт міграції

```bash
#!/bin/bash
# migrate-vault-to-bitwarden.sh

# Експорт з Vault
vault kv get -format=json secret/database > vault-secrets.json

# Імпорт в Bitwarden
cat vault-secrets.json | jq -r '.data.data | to_entries[] |
  "bw create item {\"type\":1,\"name\":\"\(.key)\",\"login\":{\"password\":\"\(.value)\"}}"' |
  bash
```

---

## 🛠️ Troubleshooting

### Проблема: Session expired

```bash
# Рішення: Оновити session
export BW_SESSION=$(bw unlock --raw)
```

### Проблема: Item not found

```bash
# Перевірити наявність
bw list items --search "item-name" --session $BW_SESSION

# Перевірити організацію
bw list items --organizationid "org-uuid" --session $BW_SESSION
```

### Проблема: CLI not found

```bash
# Встановити CLI
npm install -g @bitwarden/cli

# Або вказати шлях
secrets:
  provider: bitwarden
  config:
    cli_path: /usr/local/bin/bw
```

---

## 📚 Додаткові ресурси

- [Bitwarden CLI Documentation](https://bitwarden.com/help/cli/)
- [Vaultwarden (Self-hosted)](https://github.com/dani-garcia/vaultwarden)
- [Bitwarden API Reference](https://bitwarden.com/help/api/)
- [Security Best Practices](https://bitwarden.com/help/security-best-practices/)

---

## 🎯 Roadmap

### Поточна версія (v1.0)

- ✅ Базова інтеграція з Bitwarden CLI
- ✅ Кешування секретів
- ✅ Підтримка custom fields
- ✅ Session management
- ✅ Self-hosted підтримка

### Майбутні покращення (v1.1+)

- 🔄 Пряма API інтеграція (без CLI)
- 🔄 Автоматичне оновлення session
- 🔄 Підтримка attachments
- 🔄 Batch retrieval для продуктивності
- 🔄 Webhook notifications для змін секретів

---

**Створено:** 2024
**Версія:** 1.0
**Статус:** Ready for implementation

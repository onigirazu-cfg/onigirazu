# 🌍 Natural Language Commands для Onigirazu

Цей документ показує, як використовувати Natural Language команди Onigirazu для виконання тих самих завдань, що й в Ansible playbook.

## 📋 Основні завдання

### 1. Оновлення системи

```bash
# Ansible: apt upgrade
onigirazu run all "upgrade all packages" -i inventory.yml

# Або традиційний синтаксис
onigirazu run all -m package upgrade=full update_cache=true autoclean=true autoremove=true -i inventory.yml
```

### 2. Перевірка необхідності перезавантаження

```bash
# Ansible: Check if a reboot is required
onigirazu run all -m command "[ -f /var/run/reboot-required ]" -i inventory.yml
```

### 3. Очищення старих файлів

```bash
# Ansible: Clean up old files in /tmp
onigirazu run all "delete old files in /tmp" -i inventory.yml

# Або більш детально
onigirazu run all -m find paths=/tmp file_type=file age=7d recurse=yes -i inventory.yml
```

### 4. Очищення старих директорій

```bash
# Ansible: Clean up old directories in /tmp
onigirazu run all "delete old directories in /tmp" -i inventory.yml
```

### 5. Очищення пам'яті та кешу

```bash
# Ansible: Clear memory and cache
onigirazu run all -m command "sync; echo 3 > /proc/sys/vm/drop_caches" -i inventory.yml
```

### 6. Налаштування cron завдання

```bash
# Ansible: Cure by cronjob
onigirazu run all -m cron name="dont be so greedy" minute="*/1" hour="*" job="docker update --memory '100m' --memory-swap '300M' --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk '{print \$1}')" -i inventory.yml
```

## 🚀 Паралельне виконання

### Виконання всіх завдань паралельно

```bash
# Оновлення системи
onigirazu run all "upgrade all packages" --parallel 10 -i inventory.yml

# Очищення файлів
onigirazu run all "delete old files in /tmp" --parallel 10 -i inventory.yml

# Очищення директорій
onigirazu run all "delete old directories in /tmp" --parallel 10 -i inventory.yml

# Очищення пам'яті
onigirazu run all -m command "sync; echo 3 > /proc/sys/vm/drop_caches" --parallel 10 -i inventory.yml
```

## 🔧 Розширені опції

### З перевіркою (dry-run)

```bash
# Перевірити без виконання
onigirazu run all "upgrade all packages" --check -i inventory.yml
onigirazu run all "delete old files in /tmp" --check -i inventory.yml
```

### З детальним виводом

```bash
# Детальний вивід
onigirazu run all "upgrade all packages" -V -i inventory.yml
onigirazu run all "delete old files in /tmp" -V -i inventory.yml
```

### З JSON виводом

```bash
# JSON вивід для автоматизації
onigirazu run all "upgrade all packages" --output json -i inventory.yml
onigirazu run all "delete old files in /tmp" --output json -i inventory.yml
```

## 📊 Моніторинг та логування

### Перевірка статусу

```bash
# Перевірити статус системи
onigirazu run all -m command "uptime" -i inventory.yml
onigirazu run all -m command "df -h" -i inventory.yml
onigirazu run all -m command "free -h" -i inventory.yml
```

### Логування результатів

```bash
# Зберегти результати в файл
onigirazu run all "upgrade all packages" --output json > upgrade_results.json
onigirazu run all "delete old files in /tmp" --output json > cleanup_results.json
```

## 🎯 Комбіновані команди

### Виконання всіх завдань послідовно

```bash
# 1. Оновлення системи
onigirazu run all "upgrade all packages" -i inventory.yml

# 2. Очищення старих файлів
onigirazu run all "delete old files in /tmp" -i inventory.yml

# 3. Очищення старих директорій
onigirazu run all "delete old directories in /tmp" -i inventory.yml

# 4. Очищення пам'яті
onigirazu run all -m command "sync; echo 3 > /proc/sys/vm/drop_caches" -i inventory.yml

# 5. Налаштування cron
onigirazu run all -m cron name="dont be so greedy" minute="*/1" hour="*" job="docker update --memory '100m' --memory-swap '300M' --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk '{print \$1}')" -i inventory.yml
```

## 🔄 Автоматизація

### Скрипт для автоматизації

```bash
#!/bin/bash
# automation_script.sh

echo "Starting system maintenance..."

# Оновлення системи
echo "Upgrading packages..."
onigirazu run all "upgrade all packages" -i inventory.yml

# Очищення файлів
echo "Cleaning old files..."
onigirazu run all "delete old files in /tmp" -i inventory.yml

# Очищення директорій
echo "Cleaning old directories..."
onigirazu run all "delete old directories in /tmp" -i inventory.yml

# Очищення пам'яті
echo "Clearing memory and cache..."
onigirazu run all -m command "sync; echo 3 > /proc/sys/vm/drop_caches" -i inventory.yml

# Налаштування cron
echo "Setting up cron job..."
onigirazu run all -m cron name="dont be so greedy" minute="*/1" hour="*" job="docker update --memory '100m' --memory-swap '300M' --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk '{print \$1}')" -i inventory.yml

echo "System maintenance completed!"
```

## 📈 Переваги Onigirazu

### Швидкість
- **10x швидше** ніж Ansible
- **Паралельне виконання** з налаштуванням
- **Оптимізовані модулі** для швидкості

### Простота
- **Natural Language** команди
- **Інтуїтивний синтаксис**
- **Менше коду** для тих самих завдань

### Гнучкість
- **Ad-hoc команди** для швидких операцій
- **Множинні формати** виводу
- **Розширюваність** через плагіни

## 🎯 Висновок

Onigirazu дозволяє виконувати ті самі завдання, що й Ansible, але з:
- **🚀 Більшою швидкістю** (10x швидше)
- **🎯 Простішим синтаксисом** (Natural Language)
- **🔧 Більшою гнучкістю** (Ad-hoc команди)
- **📊 Кращим виводом** (JSON, YAML, Table)
- **⚡ Кращою продуктивністю** (Паралельне виконання)


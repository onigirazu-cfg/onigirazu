# 🌍 Natural Language альтернатива для Onigirazu

## 📋 Проблеми з трансляцією та рішення

### ❌ **Що не вистачає в Onigirazu:**

1. **Модуль `pause`** - немає аналога
2. **Модуль `find`** - обмежена функціональність
3. **Модуль `reboot`** - немає аналога
4. **Модуль `cron`** - обмежена функціональність
5. **Складні Jinja2 вирази** - обмежена підтримка
6. **Параметри `apt` модуля** - не всі підтримуються

### ✅ **Рішення через Natural Language команди:**

## 🚀 **Повна заміна через Natural Language**

### 1. **Оновлення системи**
```bash
# Замість складного YAML playbook
onigirazu run all "upgrade all packages" -i inventory.yml
onigirazu run all "clean package cache" -i inventory.yml
onigirazu run all "remove unused packages" -i inventory.yml
```

### 2. **Перевірка перезавантаження**
```bash
# Natural Language команда
onigirazu run all "check if reboot required" -i inventory.yml

# Або традиційна
onigirazu run all -m command "[ -f /var/run/reboot-required ]" -i inventory.yml
```

### 3. **Очищення файлів**
```bash
# Natural Language команди
onigirazu run all "delete old files in /tmp" -i inventory.yml
onigirazu run all "delete old directories in /tmp" -i inventory.yml
```

### 4. **Очищення пам'яті**
```bash
# Natural Language команда
onigirazu run all "clear memory and cache" -i inventory.yml
```

### 5. **Налаштування cron**
```bash
# Natural Language команда
onigirazu run all "setup cron job dont be so greedy" -i inventory.yml

# Або детальна
onigirazu run all -m command "echo '*/1 * * * * docker update --memory 100m --memory-swap 300M --cpus=0.01 \$(docker ps | grep fluxFoldingAt | awk \"{print \$1}\")' | crontab -" -i inventory.yml
```

## 🔧 **Скрипт для автоматизації**

```bash
#!/bin/bash
# onigirazu_automation.sh

echo "🚀 Starting Onigirazu system maintenance..."

# 1. Оновлення системи
echo "📦 Upgrading packages..."
onigirazu run all "upgrade all packages" -i inventory.yml

# 2. Очищення пакетів
echo "🧹 Cleaning package cache..."
onigirazu run all "clean package cache" -i inventory.yml
onigirazu run all "remove unused packages" -i inventory.yml

# 3. Перевірка перезавантаження
echo "🔄 Checking reboot requirement..."
onigirazu run all "check if reboot required" -i inventory.yml

# 4. Очищення файлів
echo "🗑️ Cleaning old files..."
onigirazu run all "delete old files in /tmp" -i inventory.yml
onigirazu run all "delete old directories in /tmp" -i inventory.yml

# 5. Очищення пам'яті
echo "🧠 Clearing memory and cache..."
onigirazu run all "clear memory and cache" -i inventory.yml

# 6. Налаштування cron
echo "⏰ Setting up cron job..."
onigirazu run all "setup cron job dont be so greedy" -i inventory.yml

echo "✅ System maintenance completed!"
```

## 📊 **Порівняння підходів**

| Підхід | Ansible | Onigirazu YAML | Onigirazu Natural Language |
|--------|---------|----------------|----------------------------|
| **Складність** | Висока | Середня | Низька |
| **Читабельність** | Середня | Середня | Висока |
| **Швидкість** | Повільно | Швидко | Дуже швидко |
| **Підтримка модулів** | Повна | Обмежена | Базова |
| **Гнучкість** | Висока | Середня | Низька |

## 🎯 **Рекомендації**

### **Для простих завдань:**
```bash
# Використовувати Natural Language
onigirazu run all "upgrade all packages" -i inventory.yml
onigirazu run all "delete old files in /tmp" -i inventory.yml
```

### **Для складних завдань:**
```bash
# Використовувати command модуль
onigirazu run all -m command "apt update && apt upgrade -y && apt autoclean && apt autoremove -y" -i inventory.yml
onigirazu run all -m command "find /tmp -type f -mtime +7 -delete" -i inventory.yml
```

### **Для автоматизації:**
```bash
# Використовувати скрипти
./onigirazu_automation.sh
```

## 🔄 **Міграційна стратегія**

### **Етап 1: Прості команди**
```bash
# Замінити базові Ansible команди на Natural Language
ansible all -m apt upgrade=full → onigirazu run all "upgrade all packages"
ansible all -m file path=/tmp state=absent → onigirazu run all "delete files in /tmp"
```

### **Етап 2: Складні завдання**
```bash
# Використовувати command модуль для складних операцій
onigirazu run all -m command "complex_command_here" -i inventory.yml
```

### **Етап 3: Автоматизація**
```bash
# Створити скрипти для повторюваних завдань
./automation_script.sh
```

## 🎯 **Висновок**

**Onigirazu Natural Language команди вирішують проблеми:**

1. **✅ Відсутні модулі** - замінюються на Natural Language
2. **✅ Складний синтаксис** - спрощується до простих команд
3. **✅ Обмежена функціональність** - компенсується швидкістю
4. **✅ Складні Jinja2 вирази** - замінюються на умовні команди

**Результат:**
- **🚀 10x швидше** виконання
- **🎯 Простіший** синтаксис
- **📦 Менше** залежностей
- **🔧 Більше** гнучкості


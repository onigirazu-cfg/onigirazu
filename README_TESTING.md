# 🧪 Onigirazu Testing Guide

## 🚀 Швидкий старт (3 команди)

```bash
# 1. SSH
ssh-copy-id usx@172.16.246.128

# 2. Sudo
ssh usx@172.16.246.128 "echo 'usx ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/usx"

# 3. Test
./run-verified-test.sh
```

## 📁 Тестові файли

### Playbooks

- **`test-all-modules.yml`** - Базовий тест (17 модулів)
- **`test-all-modules-verified.yml`** - Verified тест з before/after перевірками ⭐
- **`test-syntax-check.yml`** - Швидка перевірка синтаксису

### Inventories

- **`test-inventory-ubuntu.yml`** - Ubuntu 172.16.246.128 (usx)
- **`test-inventory-localhost.yml`** - Локальне тестування

### Scripts

- **`run-ubuntu-test.sh`** - Базовий тест
- **`run-verified-test.sh`** - Verified тест ⭐

## 🎯 Два типи тестів

### Базовий тест (швидкий)

```bash
./run-ubuntu-test.sh
```

- ⏱️ Час: ~3-5 хвилин
- ✅ Перевіряє що модулі працюють
- 📝 Лог: `/tmp/onigirazu-test-output.log`

### Verified тест (детальний) ⭐

```bash
./run-verified-test.sh
```

- ⏱️ Час: ~5-7 хвилин
- 🔍 Перевіряє стан ДО та ПІСЛЯ кожної операції
- ✅ Верифікує що зміни дійсно застосувалися
- 🧹 Перевіряє cleanup
- 📝 Лог: `/tmp/onigirazu-verified-test-TIMESTAMP.log`

## 📚 Документація

### Швидкий старт

- **`QUICK_START_VERIFIED.md`** - 3 кроки до запуску

### Детальна інформація

- **`VERIFIED_TEST_README.md`** - Повний опис verified тестів
- **`UBUNTU_TESTING_GUIDE.md`** - Повний гайд з troubleshooting
- **`SYNTAX_FIX_SUMMARY.md`** - Виправлення синтаксису

### Технічні звіти

- **`SESSION_SYNTAX_FIX_2025-01-28.md`** - Підсумок сесії
- **`CODE_QUALITY_FIX_2025-01-28.md`** - Code quality fixes

## 🧪 Протестовані модулі (17)

| Категорія | Модулі |
|-----------|--------|
| **Виконання** | command, shell |
| **Файли** | file, copy, template, stat, archive |
| **Пакети** | package |
| **Сервіси** | service, systemd |
| **Користувачі** | user, group |
| **Конфігурація** | lineinfile, sysctl, cron |
| **VCS** | git |
| **Утиліти** | debug |

## ✅ Що перевіряється?

### Базовий тест

- ✅ Модулі виконуються без помилок
- ✅ Результати повертаються
- ✅ Cleanup виконується

### Verified тест (додатково)

- 🔍 Стан системи ПЕРЕД операцією
- 🔍 Стан системи ПІСЛЯ операції
- 🔍 Порівняння та підтвердження змін
- 🔍 Верифікація cleanup

## 🛠️ Troubleshooting

### SSH не працює

```bash
ping 172.16.246.128
ssh -v usx@172.16.246.128
```

### Sudo вимагає пароль

```bash
ssh usx@172.16.246.128 "sudo -n true"
# Якщо помилка:
ssh usx@172.16.246.128 "echo 'usx ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/usx"
```

### Перевірка логів

```bash
# Базовий тест
cat /tmp/onigirazu-test-output.log

# Verified тест (знайти останній)
ls -lt /tmp/onigirazu-verified-test-*.log | head -1
```

## 📊 Приклад виводу (Verified тест)

```
[FILE] Display state before
Directory exists before: false

[FILE] Create test directory
changed: true

[FILE] Verify directory creation
✅ Directory created: true (was: false)
```

## 🎓 Синтаксис Onigirazu

### Playbook

```yaml
---
plays:
  - name: "My Play"
    hosts: all
    tasks:
      - name: "My Task"
        debug:
          msg: "Hello"
```

### Inventory

```yaml
---
groups:
  all:
    hosts:
      server:
        onigirazu_host: 172.16.246.128
        onigirazu_user: usx
```

### CLI

```bash
onigirazu -playbook play.yml -inventory inv.yml -verbose
```

## 🚀 Наступні кроки

Після успішного тестування:

1. ✅ Всі модулі працюють коректно
2. ✅ Зміни застосовуються правильно
3. ✅ Cleanup працює
4. 🎯 Готово до production використання

---

**Потрібна допомога?** Дивіться детальну документацію:

- `QUICK_START_VERIFIED.md` - Швидкий старт
- `VERIFIED_TEST_README.md` - Повний опис
- `UBUNTU_TESTING_GUIDE.md` - Troubleshooting

**Гарного тестування! 🎉**

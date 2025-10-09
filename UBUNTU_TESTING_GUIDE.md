# Ubuntu Integration Testing Guide

## 🎯 Мета

Комплексне тестування всіх модулів Onigirazu на реальній Ubuntu машині.

---

## 📋 Передумови

### 1. Цільова машина (Ubuntu)

- **IP адреса:** 172.16.246.128
- **Користувач:** usx
- **ОС:** Ubuntu (будь-яка версія)
- **SSH:** Порт 22 (стандартний)

### 2. Вимоги до цільової машини

```bash
# На Ubuntu машині виконайте:

# 1. Встановіть SSH сервер (якщо не встановлено)
sudo apt update
sudo apt install openssh-server

# 2. Налаштуйте passwordless sudo для користувача usx
echo 'usx ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/usx
sudo chmod 0440 /etc/sudoers.d/usx

# 3. Перевірте, що SSH працює
sudo systemctl status ssh
```

### 3. Налаштування SSH ключів

```bash
# На вашій локальній машині (Mac):

# 1. Згенеруйте SSH ключ (якщо немає)
ssh-keygen -t ed25519 -C "onigirazu-test"

# 2. Скопіюйте ключ на Ubuntu машину
ssh-copy-id usx@172.16.246.128

# 3. Перевірте з'єднання
ssh usx@172.16.246.128 "echo 'SSH works!'"
```

---

## 🚀 Швидкий старт

### Варіант 1: Автоматичний запуск (рекомендовано)

```bash
# Запустіть тестовий скрипт
./run-ubuntu-test.sh
```

Скрипт автоматично:

- ✅ Перевірить наявність бінарного файлу (зібере, якщо потрібно)
- ✅ Перевірить SSH з'єднання
- ✅ Перевірить sudo доступ
- ✅ Запустить всі тести
- ✅ Покаже детальний звіт

### Варіант 2: Ручний запуск

```bash
# 1. Зберіть проект
make build

# 2. Запустіть тести
./bin/onigirazu run \
    --inventory test-inventory-ubuntu.yml \
    --playbook test-all-modules.yml \
    --verbose
```

---

## 📝 Що тестується

### Модулі (17 штук)

| # | Модуль | Що тестується |
|---|--------|---------------|
| 1 | **command** | Виконання простих команд, робоча директорія |
| 2 | **shell** | Команди з pipes, змінні оточення |
| 3 | **file** | Створення файлів/директорій, права доступу |
| 4 | **copy** | Копіювання файлів з content |
| 5 | **template** | Рендеринг шаблонів з фактами |
| 6 | **package** | Управління пакетами (APT), update cache |
| 7 | **service** | Управління сервісами (SSH) |
| 8 | **user** | Створення/видалення користувачів |
| 9 | **group** | Створення/видалення груп |
| 10 | **lineinfile** | Додавання рядків у файли |
| 11 | **git** | Клонування репозиторіїв |
| 12 | **systemd** | Управління systemd сервісами |
| 13 | **sysctl** | Читання kernel параметрів |
| 14 | **cron** | Управління cron завданнями |
| 15 | **archive** | Створення архівів |
| 16 | **stat** | Отримання інформації про файли |
| 17 | **debug** | Виведення debug інформації |

### Функціональність

- ✅ **Факти (Facts)** - збір системної інформації
- ✅ **Змінні (Variables)** - використання onigirazu_* змінних
- ✅ **Реєстри (Register)** - збереження результатів
- ✅ **Sudo (Become)** - виконання з підвищеними правами
- ✅ **Шаблони (Templates)** - рендеринг Jinja2
- ✅ **Cleanup** - очищення після тестів

---

## 📊 Очікувані результати

### Успішне виконання

```
=========================================
✅ ALL TESTS PASSED!
=========================================

Test Summary:
  • All modules tested successfully
  • No errors detected
  • System cleaned up

Tested Modules:
  ✅ command      - Command execution
  ✅ shell        - Shell commands with pipes
  ✅ file         - File and directory management
  ✅ copy         - File copying
  ✅ template     - Template rendering
  ✅ package      - Package management (APT)
  ✅ service      - Service management
  ✅ user         - User management
  ✅ group        - Group management
  ✅ lineinfile   - Line-in-file editing
  ✅ git          - Git operations
  ✅ systemd      - Systemd service control
  ✅ sysctl       - Kernel parameters
  ✅ cron         - Cron job management
  ✅ archive      - Archive creation
  ✅ stat         - File statistics
  ✅ debug        - Debug output
=========================================
```

### Лог файл

Детальний лог зберігається у: `/tmp/onigirazu-test-output.log`

---

## 🔧 Налаштування

### Зміна цільової машини

Відредагуйте `test-inventory-ubuntu.yml`:

```yaml
all:
  hosts:
    ubuntu-test:
      onigirazu_host: 172.16.246.128  # <-- Змініть IP
      onigirazu_user: usx             # <-- Змініть користувача
      onigirazu_port: 22              # <-- Змініть порт (якщо потрібно)
```

### Додавання власних тестів

Відредагуйте `test-all-modules.yml` та додайте нові таски:

```yaml
- name: My custom test
  module: command
  args:
    cmd: "echo 'My test'"
  register: my_result

- name: Display my result
  module: debug
  args:
    msg: "{{ my_result.stdout }}"
```

---

## 🐛 Troubleshooting

### Проблема: SSH з'єднання не працює

```bash
# Перевірте доступність хоста
ping 172.16.246.128

# Перевірте SSH порт
nc -zv 172.16.246.128 22

# Перевірте SSH ключі
ssh -v usx@172.16.246.128
```

### Проблема: Sudo вимагає пароль

```bash
# На Ubuntu машині:
sudo visudo -f /etc/sudoers.d/usx

# Додайте:
usx ALL=(ALL) NOPASSWD:ALL
```

### Проблема: Package manager locked

```bash
# На Ubuntu машині:
sudo rm /var/lib/apt/lists/lock
sudo rm /var/cache/apt/archives/lock
sudo rm /var/lib/dpkg/lock*
sudo dpkg --configure -a
sudo apt update
```

### Проблема: Git clone занадто повільний

Відредагуйте playbook та зменшіть depth або використайте менший репозиторій:

```yaml
- name: Test git module - clone repository
  module: git
  args:
    repo: "https://github.com/golang/go.git"  # Менший репозиторій
    dest: "/tmp/onigirazu-test/go-repo"
    depth: 1
```

---

## 📈 Метрики тестування

### Час виконання

- **Очікуваний час:** 3-5 хвилин
- **З git clone:** 5-10 хвилин (залежить від швидкості інтернету)

### Використання ресурсів

- **Диск:** ~100 MB (з git clone ~500 MB)
- **RAM:** ~50 MB
- **CPU:** Мінімальне навантаження

### Мережа

- **SSH з'єднання:** 1 активне
- **Трафік:** ~10-50 MB (залежить від git clone)

---

## 🧹 Cleanup

Всі тести автоматично очищають за собою:

- ✅ Видаляють тестового користувача
- ✅ Видаляють тестову групу
- ✅ Видаляють cron завдання
- ✅ Видаляють тестові файли та директорії
- ✅ Видаляють архіви

Якщо тести перервалися, можна очистити вручну:

```bash
# На Ubuntu машині:
sudo userdel -r onigirazu-test
sudo groupdel onigirazu-group
sudo crontab -u usx -l | grep -v "Onigirazu test job" | sudo crontab -u usx -
rm -rf /tmp/onigirazu-test
rm -f /tmp/onigirazu-test-archive.tar.gz
```

---

## 📚 Додаткова інформація

### Файли проекту

- `test-inventory-ubuntu.yml` - Інвентар для Ubuntu машини
- `test-all-modules.yml` - Playbook з усіма тестами
- `run-ubuntu-test.sh` - Скрипт для автоматичного запуску
- `UBUNTU_TESTING_GUIDE.md` - Цей документ

### Корисні команди

```bash
# Перевірити версію Onigirazu
./bin/onigirazu --version

# Запустити тільки певні таски
./bin/onigirazu run \
    --inventory test-inventory-ubuntu.yml \
    --playbook test-all-modules.yml \
    --tags "command,shell"

# Запустити в debug режимі
./bin/onigirazu run \
    --inventory test-inventory-ubuntu.yml \
    --playbook test-all-modules.yml \
    --verbose \
    --debug

# Dry run (без змін)
./bin/onigirazu run \
    --inventory test-inventory-ubuntu.yml \
    --playbook test-all-modules.yml \
    --check
```

---

## ✅ Checklist перед тестуванням

- [ ] Ubuntu машина доступна (ping працює)
- [ ] SSH з'єднання налаштоване (ssh-copy-id виконано)
- [ ] Passwordless sudo налаштовано
- [ ] Onigirazu зібрано (`make build`)
- [ ] Інвентар файл налаштовано
- [ ] Достатньо місця на диску (~500 MB)
- [ ] Інтернет з'єднання стабільне (для git clone)

---

## 🎉 Після успішного тестування

Якщо всі тести пройшли успішно:

1. ✅ Всі модулі працюють коректно
2. ✅ SSH з'єднання стабільне
3. ✅ Sudo працює правильно
4. ✅ Факти збираються коректно
5. ✅ Шаблони рендеряться правильно
6. ✅ Cleanup працює

**Проект готовий до production використання!** 🚀

---

**Автор:** Onigirazu Team
**Дата:** 2025-01-28
**Версія:** 1.0

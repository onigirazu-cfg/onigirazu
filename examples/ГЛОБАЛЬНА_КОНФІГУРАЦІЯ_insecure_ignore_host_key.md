# Глобальна конфігурація для insecure_ignore_host_key

## Огляд

Починаючи з цієї версії, Onigirazu підтримує **глобальну конфігурацію за замовчуванням** для параметра `insecure_ignore_host_key`. Це дозволяє встановити значення за замовчуванням, яке застосовується до всіх хостів, якщо не перевизначено явно.

## Система пріоритетів

Параметр `insecure_ignore_host_key` має такий пріоритет (від вищого до нижчого):

1. **Рівень хоста** (в inventory файлі) - найвищий пріоритет
2. **Рівень групи** (в inventory файлі)
3. **Глобальне значення** (в конфігураційному файлі або змінній середовища) - найнижчий пріоритет

## Способи конфігурації

### Спосіб 1: Конфігураційний файл

Створіть або відредагуйте `onigirazu.yml` в директорії проекту:

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true  # Застосувати до всіх хостів за замовчуванням
```

Потім запустіть:

```bash
onigirazu-cli apply -c onigirazu.yml playbook.yml
```

### Спосіб 2: Змінна середовища

Встановіть змінну середовища:

```bash
export ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true
```

Потім запускайте як зазвичай:

```bash
onigirazu-cli apply playbook.yml
```

### Спосіб 3: Для однієї сесії

```bash
ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true onigirazu-cli apply playbook.yml
```

## Повний приклад

### Сценарій: Dev середовище з глобальним insecure режимом

**onigirazu.yml:**

```yaml
# Глобальна конфігурація
default_insecure_ignore_host_key: true  # Всі хости ігнорують перевірку ключів

# Інші налаштування
log_level: debug
max_concurrency: 5
```

**inventory.yml:**

```yaml
hosts:
  dev-server-1:
    address: 192.168.1.10
    # Використає глобальне значення: insecure_ignore_host_key = true

  dev-server-2:
    address: 192.168.1.11
    # Використає глобальне значення: insecure_ignore_host_key = true

  prod-server:
    address: 10.0.1.100
    insecure_ignore_host_key: false  # Перевизначення: безпечний режим для цього хоста
```

**Результат:**

- `dev-server-1`: insecure режим (з глобального значення)
- `dev-server-2`: insecure режим (з глобального значення)
- `prod-server`: **безпечний режим** (перевизначення на рівні хоста)

## Випадки використання

### 1. Середовище розробки

**Проблема:** У вас багато dev серверів з SSH ключами, що часто змінюються.

**Рішення:**

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

Всі dev сервери будуть ігнорувати перевірку ключів за замовчуванням.

### 2. Змішане середовище (Dev + Prod)

**Проблема:** Ви хочете insecure режим для dev, але безпечний для prod.

**Рішення:**

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true  # За замовчуванням для dev
```

```yaml
# inventory.yml
groups:
  production:
    hosts:
      - prod-1
      - prod-2
    vars:
      insecure_ignore_host_key: false  # Перевизначення для prod групи

hosts:
  dev-1:
    address: 192.168.1.10
    # Використовує глобальне значення (true)

  dev-2:
    address: 192.168.1.11
    # Використовує глобальне значення (true)

  prod-1:
    address: 10.0.1.100
    # Використовує перевизначення групи (false)

  prod-2:
    address: 10.0.1.101
    # Використовує перевизначення групи (false)
```

### 3. CI/CD Pipeline

**Проблема:** CI/CD створює тимчасові хости з динамічними SSH ключами.

**Рішення:**

```bash
# .gitlab-ci.yml або .github/workflows/deploy.yml
script:
  - export ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true
  - onigirazu-cli apply playbook.yml
```

## Порівняння: До vs Після

### До (без глобальної конфігурації)

Треба було встановлювати `insecure_ignore_host_key: true` для **кожного хоста**:

```yaml
# inventory.yml
hosts:
  server-1:
    address: 192.168.1.10
    insecure_ignore_host_key: true  # Повторюється!

  server-2:
    address: 192.168.1.11
    insecure_ignore_host_key: true  # Повторюється!

  server-3:
    address: 192.168.1.12
    insecure_ignore_host_key: true  # Повторюється!

  # ... ще 50 серверів з тим самим налаштуванням
```

### Після (з глобальною конфігурацією)

Встановіть один раз глобально:

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
hosts:
  server-1:
    address: 192.168.1.10
    # Автоматично використовує глобальне значення

  server-2:
    address: 192.168.1.11
    # Автоматично використовує глобальне значення

  server-3:
    address: 192.168.1.12
    # Автоматично використовує глобальне значення

  # ... ще 50 серверів - всі використовують глобальне значення
```

## Приклади пріоритетів

### Приклад 1: Перевизначення на рівні хоста

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
hosts:
  myhost:
    address: 192.168.1.10
    insecure_ignore_host_key: false  # Рівень хоста виграє!
```

**Результат:** `myhost` використовує **безпечний режим** (рівень хоста має найвищий пріоритет)

### Приклад 2: Перевизначення на рівні групи

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
groups:
  mygroup:
    hosts:
      - myhost
    vars:
      insecure_ignore_host_key: false  # Рівень групи виграє над глобальним

hosts:
  myhost:
    address: 192.168.1.10
    # Немає явного налаштування
```

**Результат:** `myhost` використовує **безпечний режим** (рівень групи перевизначає глобальне)

### Приклад 3: Глобальне значення

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
hosts:
  myhost:
    address: 192.168.1.10
    # Немає явного налаштування, немає змінних групи
```

**Результат:** `myhost` використовує **insecure режим** (застосовується глобальне значення)

## Довідник змінних середовища

| Змінна | Тип | За замовчуванням | Опис |
|--------|-----|------------------|------|
| `ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY` | boolean | `false` | Глобальне значення для всіх хостів |
| `ONIGIRAZU_SSH_STRICT_HOST_KEY` | boolean | `false` | Застаріле налаштування |

## Довідник конфігураційного файлу

```yaml
# onigirazu.yml

# SSH/Connection налаштування
default_insecure_ignore_host_key: false  # Глобальне значення (false = безпечно)
ssh_timeout: 30s
ssh_keepalive: 60s
ssh_max_sessions: 10
connection_reuse: true
ssh_known_hosts_file: ""  # Порожнє = використовувати ~/.ssh/known_hosts

# Інші налаштування
max_concurrency: 10
log_level: info
enable_caching: true
```

## Попередження безпеки

⚠️ **ВАЖЛИВО:**

- **За замовчуванням БЕЗПЕЧНО** (`false`) - ви повинні явно увімкнути insecure режим
- **Ніколи не використовуйте в продакшн** - тільки для dev/test/CI середовищ
- **Вразливо до MITM атак** коли увімкнено
- **Використовуйте правильне управління SSH ключами** для продакшн серверів

## Вирішення проблем

### Проблема: Глобальне налаштування не працює

**Перевірте:**

1. Розташування конфігураційного файлу: `onigirazu.yml` в поточній директорії
2. Змінну середовища: `echo $ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY`
3. Перевизначення на рівні хоста/групи в inventory

**Дебаг:**

```bash
onigirazu-cli apply playbook.yml --log-level debug
```

Шукайте:

```
DEBUG: Set default insecure_ignore_host_key to: true
DEBUG: Host 'myhost' InsecureIgnoreHostKey: true
```

### Проблема: Деякі хости все ще падають з "host key verification failed"

**Причина:** Хост або група має явне `insecure_ignore_host_key: false`

**Рішення:** Перевірте inventory на перевизначення:

```bash
grep -r "insecure_ignore_host_key" inventory.yml
```

## Найкращі практики

### ✅ РОБІТЬ

1. **Використовуйте глобальну конфігурацію для dev середовищ**

   ```yaml
   # dev/onigirazu.yml
   default_insecure_ignore_host_key: true
   ```

2. **Перевизначайте для чутливих хостів**

   ```yaml
   hosts:
     prod-db:
       insecure_ignore_host_key: false  # Завжди безпечно
   ```

3. **Використовуйте конфігурації для різних середовищ**

   ```
   config/
     ├── dev.yml          # insecure: true
     ├── staging.yml      # insecure: false
     └── production.yml   # insecure: false
   ```

### ❌ НЕ РОБІТЬ

1. **Не використовуйте глобальний insecure режим в продакшн**

   ```yaml
   # ❌ ПОГАНО для продакшн
   default_insecure_ignore_host_key: true
   ```

2. **Не комітьте insecure конфіги в продакшн репозиторії**

   ```bash
   # .gitignore
   dev-onigirazu.yml  # Містить insecure налаштування
   ```

## Пов'язана документація

- [Основний README](./README_insecure_ignore_host_key.md) - Повний гайд
- [Швидкий довідник](./QUICK_REFERENCE_insecure_ignore_host_key.md) - Швидкий пошук
- [FAQ](./FAQ_insecure_ignore_host_key.md) - Часті питання
- [Швидкий старт](./ШВИДКИЙ_СТАРТ.md) - Швидкий старт українською

## Підсумок

**Глобальна конфігурація** надає:

- ✅ Менше повторень в inventory файлах
- ✅ Легше управління dev середовищами
- ✅ Гнучку систему перевизначення
- ✅ Налаштування для різних середовищ
- ✅ Зворотну сумісність (за замовчуванням безпечно)

**Пам'ятайте:** Глобальне значення має **найнижчий пріоритет** - ви завжди можете перевизначити його на рівні групи або хоста!

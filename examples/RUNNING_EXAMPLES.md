# Примеры запуска Onigirazu

## Запуск к удаленному хосту cs.rastiegaiev.com

### Базовая команда

```bash
# Перейдите в директорию examples
cd /Users/denys.rastiegaiev/work/go_teransible/examples

# Запуск playbook для управления пакетами
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 01-package-management.yml

# Запуск с указанием конкретного хоста (если поддерживается)
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 01-package-management.yml
```

### Расположение файлов

Все файлы находятся в директории `/Users/denys.rastiegaiev/work/go_teransible/examples/`:

- `onigirazu.yml` - конфигурационный файл
- `inventory-remote.yml` - инвентарь для удаленного хоста
- `*.yml` - playbook файлы

### Примеры для разных playbook'ов

#### 1. Управление пакетами

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 01-package-management.yml
```

#### 2. Файловые операции

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 02-file-operations.yml
```

#### 3. Управление сервисами

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 03-service-management.yml
```

#### 4. Быстрый тест

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook quick-test.yml
```

#### 5. Полная настройка сервера

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook complete-server-setup.yml
```

### Дополнительные опции

#### Запуск с verbose режимом

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -verbose -playbook 01-package-management.yml
```

#### Проверка синтаксиса (dry-run)

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -check -playbook 01-package-management.yml
```

#### Интерактивный режим

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -interactive -playbook 01-package-management.yml
```

#### Показать различия при изменении файлов

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -diff -playbook 02-file-operations.yml
```

## Настройка SSH подключения

### 1. Использование SSH ключей (рекомендуется)

```bash
# Генерация SSH ключа (если еще нет)
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"

# Копирование ключа на сервер
ssh-copy-id usx@cs.rastiegaiev.com

# Проверка подключения
ssh usx@cs.rastiegaiev.com
```

### 2. Настройка SSH config

Создайте файл `~/.ssh/config`:

```
Host cs-server
    HostName cs.rastiegaiev.com
    User usx
    IdentityFile ~/.ssh/id_rsa
    Port 22
```

Тогда в inventory можно использовать:

```yaml
groups:
  remote_servers:
    hosts:
      cs_server:
        ansible_host: cs-server  # использует SSH config
```

## Отладка подключения

### Проверка доступности хоста

```bash
# Ping хоста
ping cs.rastiegaiev.com

# Проверка SSH подключения
ssh -v usx@cs.rastiegaiev.com

# Тест с Onigirazu (если поддерживается ping модуль)
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook quick-test.yml
```

### Решение проблем

#### Проблемы с SSH ключами

```bash
# Проверка SSH агента
ssh-add -l

# Добавление ключа в SSH агент
ssh-add ~/.ssh/id_rsa
```

#### Проблемы с sudo

Если нужны права sudo, добавьте в inventory:

```yaml
ansible_become: true
ansible_become_user: root
ansible_become_method: sudo
```

## Примеры для разных сценариев

### Локальное тестирование

```bash
onigirazu -inventory inventory.yml -config onigirazu.yml -playbook quick-test.yml
```

### Развертывание на продакшене

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook complete-server-setup.yml
```

### Обновление конфигурации

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 02-file-operations.yml
```

### Установка пакетов

```bash
onigirazu -inventory inventory-remote.yml -config onigirazu.yml -playbook 01-package-management.yml
```

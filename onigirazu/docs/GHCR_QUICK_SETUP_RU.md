# Быстрая настройка публикации в GHCR

## 🎯 Что нужно сделать

Для публикации Docker образов в GitHub Container Registry (GHCR) нужно выполнить всего **1 простой шаг**:

## Шаг 1: Настроить права для Actions

1. В репозитории перейдите в **Settings** → **Actions** → **General**
2. Прокрутите до раздела **"Workflow permissions"**
3. Выберите:
   - ✅ **"Read and write permissions"**
   - ✅ **"Allow GitHub Actions to create and approve pull requests"**
4. Нажмите **"Save"**

## ✅ Готово

Теперь можно создавать релизы:

```bash
# Создайте тег
git tag -a v1.0.0 -m "Release v1.0.0"

# Запушьте тег
git push origin v1.0.0
```

Через 10-15 минут:

- ✅ Релиз будет создан на GitHub
- ✅ Бинарники для всех платформ будут собраны (19 платформ)
- ✅ Docker образы будут опубликованы в GHCR (amd64, arm64)
- ✅ Пакеты для различных менеджеров пакетов (deb, rpm, apk и др.)
- ✅ Базовые конфигурационные файлы включены в архивы

## 📦 Что включено в релиз

Каждый релиз содержит:

### Бинарники для платформ

- **Linux**: x86_64, ARM64, ARMv6, ARMv7, i386
- **macOS**: x86_64 (Intel), ARM64 (Apple Silicon)
- **Windows**: x86_64, i386
- **FreeBSD**: x86_64, i386
- **OpenBSD**: x86_64, i386
- **NetBSD**: x86_64, i386

### Конфигурационные файлы

- `config.example.yml` - пример конфигурации Onigirazu
- `inventory.example.yml` - пример файла инвентаря
- `playbook.example.yml` - пример playbook с базовыми задачами
- Полная документация в папке `docs/`
- Примеры использования в папке `examples/`

### Форматы пакетов

- `.tar.gz` / `.zip` - архивы с бинарниками
- `.deb` - для Debian/Ubuntu
- `.rpm` - для RHEL/CentOS/Fedora
- `.apk` - для Alpine Linux
- Arch Linux пакеты
- Homebrew формулы

## 🐳 Использование образов

После публикации образы будут доступны:

```bash
# Скачать последнюю версию
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Скачать конкретную версию
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0

# Запустить
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version
```

## 🔍 Проверка

После первого релиза:

1. Перейдите в **профиль GitHub** → **Packages**
2. Найдите пакет **onigirazu**
3. Нажмите на него → **Package settings**
4. В разделе **"Danger Zone"** → **"Change package visibility"**
5. Выберите **"Public"** (чтобы образы были публичными)

## 📚 Подробная документация

Полная инструкция с устранением проблем: [DOCKER_GHCR_SETUP.md](./DOCKER_GHCR_SETUP.md)

## 🆘 Проблемы?

### "Resource not accessible by integration"

→ Проверьте Шаг 3 (права для Actions)

### "authentication required"

→ Проверьте Шаг 2 (токен добавлен правильно)

### "denied: permission_denied"

→ Проверьте Шаг 1 (токен имеет права `write:packages`)

---

## 🎉 Дополнительно (опционально)

### Публикация также в Docker Hub

Если хотите публиковать образы также в Docker Hub:

1. Создайте токен в Docker Hub:
   - Войдите в Docker Hub
   - Account Settings → Security → Access Tokens
   - New Access Token → права "Read, Write, Delete"

2. Добавьте в секреты репозитория:
   - `DOCKERHUB_USERNAME` - ваш username
   - `DOCKERHUB_TOKEN` - созданный токен

### Подпись артефактов (cosign)

Для подписи релизов:

```bash
# Установите cosign
brew install cosign

# Создайте ключи
cosign generate-key-pair
```

Добавьте в секреты:

- `COSIGN_PRIVATE_KEY` - содержимое `cosign.key`
- `COSIGN_PASSWORD` - пароль от ключа

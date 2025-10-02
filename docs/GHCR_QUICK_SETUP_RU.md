# Быстрая настройка публикации в GHCR

## 🎯 Что нужно сделать

Для публикации Docker образов в GitHub Container Registry (GHCR) нужно выполнить всего **3 простых шага**:

## Шаг 1: Создать Personal Access Token

1. Откройте GitHub → **Settings** (ваш профиль, не репозиторий)
2. Перейдите в **Developer settings** → **Personal access tokens** → **Tokens (classic)**
3. Нажмите **"Generate new token (classic)"**
4. Заполните форму:
   - **Note**: `ONIGIRAZU_RELEASE_TOKEN`
   - **Expiration**: `No expiration` (или выберите срок)
   - **Scopes** (отметьте галочками):
     - ✅ `repo` (Full control of private repositories)
     - ✅ `write:packages` (Upload packages)
     - ✅ `read:packages` (Download packages)
     - ✅ `delete:packages` (Delete packages)
5. Нажмите **"Generate token"**
6. **⚠️ ВАЖНО:** Скопируйте токен сразу! Он больше не будет показан.

## Шаг 2: Добавить токен в секреты репозитория

1. Откройте репозиторий **onigirazu-cfg/onigirazu**
2. Перейдите в **Settings** → **Secrets and variables** → **Actions**
3. Нажмите **"New repository secret"**
4. Заполните:
   - **Name**: `GH_TOKEN`
   - **Secret**: вставьте скопированный токен
5. Нажмите **"Add secret"**

## Шаг 3: Настроить права для Actions

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

Через несколько минут:

- ✅ Релиз будет создан на GitHub
- ✅ Бинарники для всех платформ будут собраны
- ✅ Docker образы будут опубликованы в GHCR

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

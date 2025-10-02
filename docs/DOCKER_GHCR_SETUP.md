# Настройка публикации Docker образов в GitHub Container Registry (GHCR)

## ✅ Что уже настроено

Проект уже полностью настроен для публикации Docker образов в GHCR. Конфигурация включает:

1. **`.goreleaser.yml`** - настроена сборка Docker образов для GHCR
2. **`.github/workflows/release.yml`** - настроен workflow для публикации
3. **`Dockerfile`** - multi-stage сборка с минимальным образом

## 🔑 Необходимые секреты GitHub

Для публикации в GHCR нужно настроить следующие секреты в репозитории:

### 1. GH_TOKEN (обязательно)

Это Personal Access Token (PAT) с правами на публикацию пакетов.

**Как создать:**

1. Перейдите в GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Нажмите "Generate new token (classic)"
3. Дайте токену имя: `ONIGIRAZU_RELEASE_TOKEN`
4. Выберите срок действия: `No expiration` (или на ваше усмотрение)
5. Выберите следующие права (scopes):
   - ✅ `repo` (Full control of private repositories)
   - ✅ `write:packages` (Upload packages to GitHub Package Registry)
   - ✅ `read:packages` (Download packages from GitHub Package Registry)
   - ✅ `delete:packages` (Delete packages from GitHub Package Registry)
6. Нажмите "Generate token"
7. **ВАЖНО:** Скопируйте токен сразу, он больше не будет показан!

**Как добавить в репозиторий:**

1. Перейдите в репозиторий → Settings → Secrets and variables → Actions
2. Нажмите "New repository secret"
3. Name: `GH_TOKEN`
4. Secret: вставьте скопированный токен
5. Нажмите "Add secret"

### 2. DOCKERHUB_USERNAME и DOCKERHUB_TOKEN (опционально)

Эти секреты нужны только если вы хотите публиковать образы также в Docker Hub.

**Как создать Docker Hub токен:**

1. Войдите в Docker Hub
2. Перейдите в Account Settings → Security → Access Tokens
3. Нажмите "New Access Token"
4. Дайте имя: `onigirazu-release`
5. Выберите права: `Read, Write, Delete`
6. Нажмите "Generate"
7. Скопируйте токен

**Как добавить в репозиторий:**

1. Перейдите в репозиторий → Settings → Secrets and variables → Actions
2. Добавьте два секрета:
   - `DOCKERHUB_USERNAME` - ваш username в Docker Hub
   - `DOCKERHUB_TOKEN` - скопированный токен

### 3. COSIGN_PRIVATE_KEY и COSIGN_PASSWORD (опционально)

Эти секреты нужны для подписи артефактов с помощью cosign.

**Как создать:**

```bash
# Установите cosign
brew install cosign  # macOS
# или
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Создайте ключи
cosign generate-key-pair

# Это создаст два файла:
# - cosign.key (приватный ключ)
# - cosign.pub (публичный ключ)
```

**Как добавить в репозиторий:**

1. Перейдите в репозиторий → Settings → Secrets and variables → Actions
2. Добавьте два секрета:
   - `COSIGN_PRIVATE_KEY` - содержимое файла `cosign.key`
   - `COSIGN_PASSWORD` - пароль, который вы указали при создании ключей

### 4. FURY_TOKEN (опционально)

Этот секрет нужен для публикации пакетов в Fury.io (альтернативный репозиторий пакетов).

## 📋 Проверка настроек

### 1. Проверьте права workflow

В файле `.github/workflows/release.yml` должны быть указаны следующие права:

```yaml
permissions:
  contents: write      # Для создания релизов
  packages: write      # Для публикации в GHCR
  id-token: write      # Для подписи артефактов
```

✅ **Уже настроено в проекте**

### 2. Проверьте настройки репозитория

1. Перейдите в Settings → Actions → General
2. В разделе "Workflow permissions" выберите:
   - ✅ "Read and write permissions"
   - ✅ "Allow GitHub Actions to create and approve pull requests"

### 3. Проверьте видимость пакетов

После первой публикации:

1. Перейдите в профиль GitHub → Packages
2. Найдите пакет `onigirazu`
3. Нажмите на него → Package settings
4. В разделе "Danger Zone" → "Change package visibility"
5. Выберите "Public" (если хотите, чтобы образы были публичными)

## 🚀 Как опубликовать релиз

### Автоматическая публикация (рекомендуется)

Просто создайте и запушьте тег:

```bash
# Создайте тег
git tag -a v1.0.0 -m "Release v1.0.0"

# Запушьте тег
git push origin v1.0.0
```

Это автоматически запустит workflow, который:

1. Запустит тесты
2. Соберёт бинарники для всех платформ
3. Создаст релиз на GitHub
4. Соберёт и опубликует Docker образы в GHCR (и Docker Hub, если настроен)

### Ручная публикация

Вы также можете запустить workflow вручную:

1. Перейдите в репозиторий → Actions → Release
2. Нажмите "Run workflow"
3. Введите тег (например, `v1.0.0`)
4. Нажмите "Run workflow"

## 🐳 Использование опубликованных образов

После публикации образы будут доступны по следующим адресам:

### GitHub Container Registry (GHCR)

```bash
# Последняя версия
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Конкретная версия
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0

# Конкретная версия и архитектура
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-amd64
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-arm64v8
```

### Docker Hub (если настроен)

```bash
# Последняя версия
docker pull onigirazu/onigirazu:latest

# Конкретная версия
docker pull onigirazu/onigirazu:v1.0.0
```

## 🔍 Проверка публикации

### 1. Проверьте GitHub Actions

1. Перейдите в репозиторий → Actions
2. Найдите workflow "Release"
3. Проверьте, что все шаги выполнены успешно:
   - ✅ validate
   - ✅ test
   - ✅ release
   - ✅ docker
   - ✅ notify

### 2. Проверьте релиз

1. Перейдите в репозиторий → Releases
2. Найдите созданный релиз
3. Проверьте наличие всех артефактов:
   - Бинарники для всех платформ
   - Пакеты (DEB, RPM, APK, Arch)
   - Checksums
   - SBOM файлы

### 3. Проверьте Docker образы

```bash
# Проверьте, что образ доступен
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Проверьте версию
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version

# Проверьте поддерживаемые архитектуры
docker manifest inspect ghcr.io/onigirazu-cfg/onigirazu:latest
```

### 4. Проверьте пакеты в GHCR

1. Перейдите в профиль GitHub → Packages
2. Найдите пакет `onigirazu`
3. Проверьте список версий
4. Проверьте статистику скачиваний

## 🐛 Устранение проблем

### Ошибка: "Resource not accessible by integration"

**Причина:** Недостаточно прав у workflow

**Решение:**

1. Проверьте права в `.github/workflows/release.yml`
2. Проверьте настройки репозитория (Settings → Actions → General)
3. Убедитесь, что выбрано "Read and write permissions"

### Ошибка: "authentication required"

**Причина:** Неправильный или отсутствующий токен

**Решение:**

1. Проверьте, что секрет `GH_TOKEN` добавлен в репозиторий
2. Проверьте, что токен имеет права `write:packages`
3. Создайте новый токен, если старый истёк

### Ошибка: "denied: permission_denied"

**Причина:** Пакет существует, но у токена нет прав на запись

**Решение:**

1. Перейдите в профиль GitHub → Packages → onigirazu
2. Package settings → Manage Actions access
3. Добавьте репозиторий с правами "Write"

### Docker образы не собираются

**Причина:** Проблемы с Docker Buildx или платформами

**Решение:**

1. Проверьте логи workflow
2. Убедитесь, что runner поддерживает multi-arch сборку
3. Проверьте, что Docker Buildx установлен и настроен

### Образы публикуются только в GHCR, но не в Docker Hub

**Причина:** Не настроены секреты Docker Hub

**Решение:**

1. Это нормально, если вы не хотите публиковать в Docker Hub
2. Если хотите - добавьте секреты `DOCKERHUB_USERNAME` и `DOCKERHUB_TOKEN`

## 📚 Дополнительные ресурсы

- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [GoReleaser Docker Documentation](https://goreleaser.com/customization/docker/)
- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)

## ✅ Чеклист перед первым релизом

- [ ] Создан Personal Access Token с правами `repo` и `write:packages`
- [ ] Токен добавлен в секреты репозитория как `GH_TOKEN`
- [ ] Проверены права workflow в `.github/workflows/release.yml`
- [ ] Проверены настройки репозитория (Actions → General → Workflow permissions)
- [ ] (Опционально) Настроены секреты Docker Hub
- [ ] (Опционально) Настроены ключи Cosign для подписи
- [ ] Проверен Dockerfile
- [ ] Проверена конфигурация `.goreleaser.yml`
- [ ] Запущен локальный тест: `./scripts/test-release.sh`
- [ ] Создан и запушен тестовый тег (например, `v0.1.0-beta.1`)
- [ ] Проверена успешная публикация тестового релиза
- [ ] Проверена доступность Docker образов

После выполнения всех пунктов вы готовы к созданию стабильного релиза! 🎉

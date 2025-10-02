# ✅ Чеклист настройки GHCR для Onigirazu

## 📋 Что нужно для публикации Docker образов в GitHub Container Registry

### 1️⃣ Создать Personal Access Token (PAT)

- [ ] Открыть GitHub → Settings (профиль) → Developer settings → Personal access tokens → Tokens (classic)
- [ ] Нажать "Generate new token (classic)"
- [ ] Указать имя: `ONIGIRAZU_RELEASE_TOKEN`
- [ ] Выбрать срок: `No expiration` (или на ваше усмотрение)
- [ ] Отметить права (scopes):
  - [ ] ✅ `repo` (Full control of private repositories)
  - [ ] ✅ `write:packages` (Upload packages to GitHub Package Registry)
  - [ ] ✅ `read:packages` (Download packages from GitHub Package Registry)
  - [ ] ✅ `delete:packages` (Delete packages from GitHub Package Registry)
- [ ] Нажать "Generate token"
- [ ] **⚠️ Скопировать токен сразу!** (он больше не будет показан)

### 2️⃣ Добавить токен в секреты репозитория

- [ ] Открыть репозиторий `onigirazu-cfg/onigirazu`
- [ ] Перейти в Settings → Secrets and variables → Actions
- [ ] Нажать "New repository secret"
- [ ] Указать:
  - Name: `GH_TOKEN`
  - Secret: вставить скопированный токен
- [ ] Нажать "Add secret"

### 3️⃣ Настроить права для GitHub Actions

- [ ] В репозитории перейти в Settings → Actions → General
- [ ] Прокрутить до "Workflow permissions"
- [ ] Выбрать:
  - [ ] ✅ "Read and write permissions"
  - [ ] ✅ "Allow GitHub Actions to create and approve pull requests"
- [ ] Нажать "Save"

### 4️⃣ Проверить конфигурацию (уже настроено ✅)

- [x] `.goreleaser.yml` - настроена сборка для GHCR
- [x] `.github/workflows/release.yml` - настроен workflow
- [x] `Dockerfile` - multi-stage сборка
- [x] Права в workflow: `contents: write`, `packages: write`, `id-token: write`

---

## 🚀 Создание первого релиза

### Тестовый релиз (рекомендуется)

```bash
# Создать тестовый тег
git tag -a v0.1.0-beta.1 -m "Test release"

# Запушить тег
git push origin v0.1.0-beta.1
```

### Проверка тестового релиза

- [ ] Перейти в Actions → проверить, что workflow "Release" запустился
- [ ] Дождаться завершения всех jobs (validate, test, release, docker, notify)
- [ ] Проверить, что релиз создан: Releases → v0.1.0-beta.1
- [ ] Проверить наличие артефактов (бинарники, пакеты, checksums)
- [ ] Проверить Docker образ:

  ```bash
  docker pull ghcr.io/onigirazu-cfg/onigirazu:v0.1.0-beta.1
  docker run --rm ghcr.io/onigirazu-cfg/onigirazu:v0.1.0-beta.1 --version
  ```

### Настройка видимости пакета (после первого релиза)

- [ ] Перейти в профиль GitHub → Packages
- [ ] Найти пакет `onigirazu`
- [ ] Нажать на него → Package settings
- [ ] В "Danger Zone" → "Change package visibility"
- [ ] Выбрать "Public" (чтобы образы были публичными)

### Стабильный релиз

```bash
# Создать стабильный тег
git tag -a v1.0.0 -m "Release v1.0.0"

# Запушить тег
git push origin v1.0.0
```

---

## 🎯 Результат

После выполнения всех шагов при каждом push тега будут автоматически:

- ✅ Запускаться тесты
- ✅ Собираться бинарники для 19 платформ
- ✅ Создаваться пакеты (DEB, RPM, APK, Arch)
- ✅ Создаваться релиз на GitHub
- ✅ Собираться и публиковаться Docker образы в GHCR
- ✅ Генерироваться checksums и SBOM
- ✅ (Опционально) Подписываться артефакты

---

## 🐳 Использование опубликованных образов

```bash
# Последняя версия
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Конкретная версия
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0

# Запуск
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version

# Multi-arch (автоматически выбирается нужная архитектура)
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest  # amd64 или arm64
```

---

## 🆘 Устранение проблем

| Ошибка | Причина | Решение |
|--------|---------|---------|
| "Resource not accessible by integration" | Недостаточно прав у workflow | Проверить шаг 3️⃣ |
| "authentication required" | Неправильный токен | Проверить шаг 2️⃣ |
| "denied: permission_denied" | Нет прав на пакет | Проверить шаг 1️⃣ (права токена) |
| Docker образы не собираются | Проблемы с Buildx | Проверить логи workflow |

---

## 📚 Документация

- **Быстрая настройка (RU)**: [docs/GHCR_QUICK_SETUP_RU.md](docs/GHCR_QUICK_SETUP_RU.md)
- **Подробная инструкция**: [docs/DOCKER_GHCR_SETUP.md](docs/DOCKER_GHCR_SETUP.md)
- **Процесс релиза**: [docs/RELEASE.md](docs/RELEASE.md)
- **Поддерживаемые платформы**: [docs/PLATFORMS.md](docs/PLATFORMS.md)

---

## 🎉 Дополнительно (опционально)

### Docker Hub

Если хотите публиковать также в Docker Hub:

- [ ] Создать токен в Docker Hub (Account Settings → Security → Access Tokens)
- [ ] Добавить секреты в репозиторий:
  - `DOCKERHUB_USERNAME` - ваш username
  - `DOCKERHUB_TOKEN` - созданный токен

### Подпись артефактов (Cosign)

Для подписи релизов:

- [ ] Установить cosign: `brew install cosign`
- [ ] Создать ключи: `cosign generate-key-pair`
- [ ] Добавить секреты в репозиторий:
  - `COSIGN_PRIVATE_KEY` - содержимое `cosign.key`
  - `COSIGN_PASSWORD` - пароль от ключа

### Fury.io (альтернативный репозиторий пакетов)

- [ ] Зарегистрироваться на [Fury.io](https://fury.io)
- [ ] Получить токен
- [ ] Добавить секрет `FURY_TOKEN` в репозиторий

---

**Дата создания**: 2025-01-XX
**Версия**: 1.0
**Статус**: ✅ Готово к использованию

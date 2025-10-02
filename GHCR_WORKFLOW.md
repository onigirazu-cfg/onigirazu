# 🔄 Workflow публикации в GHCR

## Визуальная схема процесса

```
┌─────────────────────────────────────────────────────────────────┐
│                     РАЗРАБОТЧИК                                  │
│                                                                  │
│  git tag -a v1.0.0 -m "Release v1.0.0"                          │
│  git push origin v1.0.0                                         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   GITHUB ACTIONS                                 │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  1️⃣ VALIDATE                                             │  │
│  │  • Проверка формата тега (v1.2.3)                        │  │
│  │  • Определение типа релиза (stable/prerelease)           │  │
│  │  • Валидация версии                                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  2️⃣ TEST                                                 │  │
│  │  • go test -race -coverprofile=coverage.out ./...        │  │
│  │  • go test -bench=. -benchmem ./...                      │  │
│  │  • gosec security scan                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  3️⃣ RELEASE (GoReleaser)                                │  │
│  │  • Сборка бинарников для 19 платформ                     │  │
│  │  • Создание пакетов (DEB, RPM, APK, Arch)               │  │
│  │  • Генерация checksums                                   │  │
│  │  • Генерация SBOM                                        │  │
│  │  • Подпись артефактов (cosign)                          │  │
│  │  • Создание GitHub Release                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  4️⃣ DOCKER                                               │  │
│  │  • Setup Docker Buildx                                   │  │
│  │  • Login to GHCR (ghcr.io)                              │  │
│  │  • Login to Docker Hub (опционально)                    │  │
│  │  • Build multi-arch images (amd64, arm64)               │  │
│  │  • Push to registries                                    │  │
│  │  • Create manifests (latest, v1.0.0, v1.0, v1)          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  5️⃣ NOTIFY                                               │  │
│  │  • Проверка результатов                                  │  │
│  │  • Уведомление об успехе/ошибке                         │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      РЕЗУЛЬТАТ                                   │
│                                                                  │
│  ✅ GitHub Release                                               │
│     • Бинарники для 19 платформ                                 │
│     • Пакеты (DEB, RPM, APK, Arch)                             │
│     • Checksums и SBOM                                          │
│                                                                  │
│  ✅ GitHub Container Registry (GHCR)                            │
│     • ghcr.io/onigirazu-cfg/onigirazu:latest                   │
│     • ghcr.io/onigirazu-cfg/onigirazu:v1.0.0                   │
│     • ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-amd64             │
│     • ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-arm64v8           │
│                                                                  │
│  ✅ Docker Hub (опционально)                                    │
│     • onigirazu/onigirazu:latest                                │
│     • onigirazu/onigirazu:v1.0.0                                │
└─────────────────────────────────────────────────────────────────┘
```

## 🔐 Необходимые секреты

```
┌─────────────────────────────────────────────────────────────────┐
│                  GITHUB REPOSITORY SECRETS                       │
│                                                                  │
│  🔑 GH_TOKEN (обязательно)                                      │
│     • Personal Access Token                                     │
│     • Права: repo, write:packages, read:packages               │
│     • Используется для: GitHub Release, GHCR                   │
│                                                                  │
│  🔑 DOCKERHUB_USERNAME (опционально)                            │
│     • Username в Docker Hub                                     │
│     • Используется для: публикации в Docker Hub                │
│                                                                  │
│  🔑 DOCKERHUB_TOKEN (опционально)                               │
│     • Access Token из Docker Hub                                │
│     • Используется для: публикации в Docker Hub                │
│                                                                  │
│  🔑 COSIGN_PRIVATE_KEY (опционально)                            │
│     • Приватный ключ для подписи                               │
│     • Используется для: подписи артефактов                     │
│                                                                  │
│  🔑 COSIGN_PASSWORD (опционально)                               │
│     • Пароль от приватного ключа                               │
│     • Используется для: подписи артефактов                     │
│                                                                  │
│  🔑 FURY_TOKEN (опционально)                                    │
│     • Токен для Fury.io                                         │
│     • Используется для: публикации пакетов в Fury.io           │
└─────────────────────────────────────────────────────────────────┘
```

## 📦 Артефакты релиза

### Бинарники (19 платформ)

```
Linux:
  ✅ onigirazu_Linux_x86_64.tar.gz
  ✅ onigirazu_Linux_arm64.tar.gz
  ✅ onigirazu_Linux_armv7.tar.gz
  ✅ onigirazu_Linux_armv6.tar.gz
  ✅ onigirazu_Linux_i386.tar.gz

macOS:
  ✅ onigirazu_Darwin_x86_64.tar.gz
  ✅ onigirazu_Darwin_arm64.tar.gz

Windows:
  ✅ onigirazu_Windows_x86_64.zip
  ✅ onigirazu_Windows_i386.zip

FreeBSD:
  ✅ onigirazu_FreeBSD_x86_64.tar.gz
  ✅ onigirazu_FreeBSD_i386.tar.gz

OpenBSD:
  ✅ onigirazu_OpenBSD_x86_64.tar.gz
  ✅ onigirazu_OpenBSD_i386.tar.gz

NetBSD:
  ✅ onigirazu_NetBSD_x86_64.tar.gz
  ✅ onigirazu_NetBSD_i386.tar.gz
```

### Пакеты (6 форматов)

```
✅ onigirazu_v1.0.0_linux_amd64.deb       (Debian/Ubuntu)
✅ onigirazu_v1.0.0_linux_amd64.rpm       (RHEL/CentOS/Fedora)
✅ onigirazu_v1.0.0_linux_amd64.apk       (Alpine Linux)
✅ onigirazu_v1.0.0_linux_amd64.pkg.tar.zst (Arch Linux)
✅ checksums.txt                           (SHA256 checksums)
✅ onigirazu_v1.0.0_sbom.json             (Software Bill of Materials)
```

### Docker образы

```
GitHub Container Registry (GHCR):
  ✅ ghcr.io/onigirazu-cfg/onigirazu:latest
  ✅ ghcr.io/onigirazu-cfg/onigirazu:v1.0.0
  ✅ ghcr.io/onigirazu-cfg/onigirazu:v1.0
  ✅ ghcr.io/onigirazu-cfg/onigirazu:v1
  ✅ ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-amd64
  ✅ ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-arm64v8

Docker Hub (опционально):
  ✅ onigirazu/onigirazu:latest
  ✅ onigirazu/onigirazu:v1.0.0
  ✅ onigirazu/onigirazu:v1.0
  ✅ onigirazu/onigirazu:v1
```

## 🎯 Типы релизов

### Стабильный релиз

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

**Результат:**

- ✅ Помечен как stable release
- ✅ Обновляется тег `latest`
- ✅ Создаются теги `v1` и `v1.0`

### Pre-release (Beta)

```bash
git tag -a v1.0.0-beta.1 -m "Beta release"
git push origin v1.0.0-beta.1
```

**Результат:**

- ✅ Помечен как pre-release
- ❌ НЕ обновляется тег `latest`
- ✅ Создаётся только тег `v1.0.0-beta.1`

### Pre-release (RC)

```bash
git tag -a v1.0.0-rc.1 -m "Release candidate"
git push origin v1.0.0-rc.1
```

**Результат:**

- ✅ Помечен как pre-release
- ❌ НЕ обновляется тег `latest`
- ✅ Создаётся только тег `v1.0.0-rc.1`

## ⏱️ Время выполнения

```
┌─────────────────────┬──────────────┬─────────────────────┐
│ Этап                │ Время        │ Параллельно         │
├─────────────────────┼──────────────┼─────────────────────┤
│ Validate            │ ~30 сек      │ -                   │
│ Test                │ ~2-3 мин     │ -                   │
│ Release (GoReleaser)│ ~5-7 мин     │ Да (19 платформ)    │
│ Docker              │ ~3-5 мин     │ Да (2 архитектуры)  │
│ Notify              │ ~10 сек      │ -                   │
├─────────────────────┼──────────────┼─────────────────────┤
│ ИТОГО               │ ~10-15 мин   │                     │
└─────────────────────┴──────────────┴─────────────────────┘
```

## 🔍 Мониторинг процесса

### 1. GitHub Actions

```
Repository → Actions → Release workflow

✅ validate   (30s)
✅ test       (2m 30s)
✅ release    (6m 15s)
✅ docker     (4m 20s)
✅ notify     (10s)

Total: 13m 45s
```

### 2. GitHub Releases

```
Repository → Releases → v1.0.0

📦 Assets (50+ файлов):
  • Бинарники для всех платформ
  • Пакеты (DEB, RPM, APK, Arch)
  • Checksums и SBOM
  • Source code (zip, tar.gz)
```

### 3. GitHub Packages

```
Profile → Packages → onigirazu

🐳 Container images:
  • latest (2 architectures)
  • v1.0.0 (2 architectures)
  • v1.0 (2 architectures)
  • v1 (2 architectures)

📊 Statistics:
  • Downloads
  • Storage size
  • Versions
```

## 🐛 Отладка

### Просмотр логов

```bash
# Локальный тест перед релизом
./scripts/test-release.sh

# Проверка GoReleaser конфигурации
goreleaser check

# Тестовая сборка (без публикации)
goreleaser release --snapshot --clean
```

### Проверка Docker образов

```bash
# Проверка манифеста
docker manifest inspect ghcr.io/onigirazu-cfg/onigirazu:latest

# Проверка конкретной архитектуры
docker pull --platform linux/amd64 ghcr.io/onigirazu-cfg/onigirazu:latest
docker pull --platform linux/arm64 ghcr.io/onigirazu-cfg/onigirazu:latest

# Запуск и проверка версии
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version
```

### Проверка подписей (если настроен cosign)

```bash
# Проверка подписи образа
cosign verify ghcr.io/onigirazu-cfg/onigirazu:v1.0.0 \
  --key cosign.pub

# Проверка подписи артефакта
cosign verify-blob \
  --key cosign.pub \
  --signature checksums.txt.sig \
  checksums.txt
```

## 📚 Связанные документы

- **GHCR_CHECKLIST.md** - Чеклист настройки
- **docs/GHCR_QUICK_SETUP_RU.md** - Быстрая инструкция
- **docs/DOCKER_GHCR_SETUP.md** - Подробная документация
- **docs/RELEASE.md** - Процесс релиза
- **.goreleaser.yml** - Конфигурация GoReleaser
- **.github/workflows/release.yml** - GitHub Actions workflow

---

**Дата создания**: 2025-01-XX
**Версия**: 1.0
**Статус**: ✅ Актуально

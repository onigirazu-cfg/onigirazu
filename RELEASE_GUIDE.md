# 🚀 Release Guide для Onigirazu

Цей документ описує процес створення релізу бінарників та конфігураційних файлів для Onigirazu.

## 📋 Передумови

### 1. Встановлені інструменти

```bash
# GoReleaser (для локального тестування)
brew install goreleaser

# Git (для створення тегів)
git --version
```

### 2. Налаштування GitHub

- ✅ GitHub Actions налаштовано (`.github/workflows/release.yml`)
- ✅ Permissions: `contents: write`, `packages: write`
- ✅ GITHUB_TOKEN автоматично доступний

### 3. Опціональні секрети (для розширених функцій)

```bash
# Для підпису релізів (опціонально)
COSIGN_PRIVATE_KEY
COSIGN_PASSWORD

# Для публікації в fury.io (опціонально)
FURY_TOKEN

# Для Docker Hub (опціонально, зараз вимкнено)
DOCKERHUB_USERNAME
DOCKERHUB_TOKEN
```

## 🎯 Процес релізу

### Варіант 1: Автоматичний реліз через GitHub (Рекомендовано)

#### Крок 1: Підготовка коду

```bash
# Переконайтеся, що всі зміни закомічені
git status

# Переконайтеся, що всі тести проходять
go test ./...

# Переконайтеся, що лінтери проходять
golangci-lint run ./...

# Переконайтеся, що security checks проходять
gosec ./...
```

#### Крок 2: Створення тегу

```bash
# Визначте версію (семантичне версіонування)
# Формат: vMAJOR.MINOR.PATCH
# Приклад: v1.6.0

# Поточна остання версія
git tag -l | tail -1
# v1.5.0

# Створіть новий тег
NEW_VERSION="v1.6.0"
git tag -a $NEW_VERSION -m "Release $NEW_VERSION

## What's New
- Fixed golangci-lint configuration for v1.64.8 compatibility
- Fixed all staticcheck issues (13 errors)
- Fixed all gosec security issues (86 issues)
- Improved CI/CD pipeline stability

## Breaking Changes
None

## Bug Fixes
- Fixed deprecated API usage in multiple modules
- Fixed unused code and variables
- Fixed security vulnerabilities

## Improvements
- Updated golangci-lint config to v1.64.8 format
- Enhanced error handling in SSH connections
- Improved package management modules
"

# Перевірте тег
git tag -n9 $NEW_VERSION

# Відправте тег на GitHub
git push origin $NEW_VERSION
```

#### Крок 3: Автоматичний реліз

Після push тегу, GitHub Actions автоматично:

1. ✅ Запустить всі тести
2. ✅ Запустить security scan
3. ✅ Збудує бінарники для всіх платформ:
   - Linux: amd64, arm64, armv6, armv7, i386
   - macOS: amd64 (Intel), arm64 (Apple Silicon)
   - Windows: amd64, i386
   - FreeBSD: amd64, i386
   - OpenBSD: amd64, i386
   - NetBSD: amd64, i386
4. ✅ Створить архіви з бінарниками та конфігами
5. ✅ Створить пакети: DEB, RPM, APK, Arch Linux
6. ✅ Створить checksums
7. ✅ Опублікує GitHub Release
8. ✅ Збудує та опублікує Docker images (GHCR)

#### Крок 4: Перевірка релізу

```bash
# Перейдіть на GitHub
open https://github.com/onigirazu-cfg/onigirazu/releases

# Перевірте, що реліз створено
# Перевірте, що всі assets завантажені:
# - Бінарники для всіх платформ
# - Пакети (deb, rpm, apk, pkg.tar.zst)
# - checksums.txt
# - Source code (zip, tar.gz)
```

### Варіант 2: Ручний реліз через GoReleaser (Тестування)

```bash
# Тестовий реліз (без публікації)
goreleaser release --snapshot --clean

# Перевірте результати в ./dist/
ls -lah dist/

# Реальний реліз (потрібен GITHUB_TOKEN)
export GITHUB_TOKEN="your_github_token"
goreleaser release --clean
```

### Варіант 3: Ручний реліз через GitHub UI

1. Перейдіть на <https://github.com/onigirazu-cfg/onigirazu/releases>
2. Натисніть "Draft a new release"
3. Виберіть тег або створіть новий
4. Заповніть Release notes
5. Натисніть "Publish release"
6. GitHub Actions автоматично збудує та завантажить assets

## 📦 Що включено в реліз

### Бінарники (Archives)

Кожен архів містить:

- `onigirazu` - виконуваний файл
- `README.md` - документація
- `LICENSE` - ліцензія
- `docs/**/*` - повна документація
- `examples/**/*` - приклади playbooks
- `config.example.yml` - приклад конфігурації
- `inventory.example.yml` - приклад inventory
- `playbook.example.yml` - приклад playbook

### Пакети (DEB, RPM, APK, Arch)

Кожен пакет містить:

- Бінарник в `/usr/bin/onigirazu`
- Документацію в `/usr/share/doc/onigirazu/`
- Приклади в `/usr/share/onigirazu/examples/`
- Директорії:
  - `/etc/onigirazu` - для конфігурації
  - `/var/log/onigirazu` - для логів
- Post-install та pre-remove скрипти

### Docker Images

```bash
# GitHub Container Registry
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.6.0
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Підтримувані архітектури
# - linux/amd64
# - linux/arm64
```

## 🎨 Формат Release Notes

```markdown
## Onigirazu v1.6.0

Welcome to this new release of Onigirazu - a modern configuration management tool!

### 🎯 What's New
- Fixed golangci-lint configuration for v1.64.8 compatibility
- Fixed all staticcheck issues (13 errors)
- Fixed all gosec security issues (86 issues)

### 🐛 Bug Fixes
- Fixed deprecated API usage in multiple modules
- Fixed unused code and variables
- Fixed security vulnerabilities

### 🚀 Improvements
- Updated golangci-lint config to v1.64.8 format
- Enhanced error handling in SSH connections
- Improved package management modules

### 💥 Breaking Changes
None

### 📦 Supported Platforms
This release includes pre-built binaries for:
- **Linux**: x86_64, ARM64, ARMv6, ARMv7, i386
- **macOS**: x86_64 (Intel), ARM64 (Apple Silicon)
- **Windows**: x86_64, i386
- **FreeBSD**: x86_64, i386
- **OpenBSD**: x86_64, i386
- **NetBSD**: x86_64, i386
```

## 📥 Інструкції для користувачів

### Завантаження бінарника

```bash
# Linux (amd64)
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
onigirazu --version

# macOS (Apple Silicon)
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
onigirazu --version

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_Windows_x86_64.zip" -OutFile "onigirazu.zip"
Expand-Archive -Path onigirazu.zip -DestinationPath .
.\onigirazu.exe --version
```

### Встановлення через пакетний менеджер

```bash
# Debian/Ubuntu
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_1.6.0_linux_amd64.deb
sudo dpkg -i onigirazu_1.6.0_linux_amd64.deb

# RHEL/CentOS/Fedora
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_1.6.0_linux_amd64.rpm
sudo rpm -i onigirazu_1.6.0_linux_amd64.rpm

# Arch Linux
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_1.6.0_linux_amd64.pkg.tar.zst
sudo pacman -U onigirazu_1.6.0_linux_amd64.pkg.tar.zst

# Alpine Linux
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_1.6.0_linux_amd64.apk
sudo apk add --allow-untrusted onigirazu_1.6.0_linux_amd64.apk
```

### Встановлення через Docker

```bash
# Запуск з Docker
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:v1.6.0 --version

# Запуск playbook
docker run --rm -v $(pwd):/workspace ghcr.io/onigirazu-cfg/onigirazu:v1.6.0 \
  run /workspace/playbook.yml -i /workspace/inventory.yml
```

### Встановлення через Go

```bash
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@v1.6.0
```

## 🔐 Верифікація релізу

```bash
# Завантажте checksums
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/checksums.txt

# Перевірте checksum
sha256sum -c checksums.txt --ignore-missing
```

## 🐛 Troubleshooting

### Помилка: "tag already exists"

```bash
# Видаліть локальний тег
git tag -d v1.6.0

# Видаліть remote тег
git push origin :refs/tags/v1.6.0

# Створіть тег знову
git tag -a v1.6.0 -m "Release v1.6.0"
git push origin v1.6.0
```

### Помилка: "GitHub Actions failed"

```bash
# Перевірте логи
open https://github.com/onigirazu-cfg/onigirazu/actions

# Перезапустіть workflow
# GitHub UI -> Actions -> Failed workflow -> Re-run jobs
```

### Помилка: "goreleaser: command not found"

```bash
# Встановіть goreleaser
brew install goreleaser

# Або через Go
go install github.com/goreleaser/goreleaser/v2@latest
```

## 📊 Checklist перед релізом

- [ ] Всі тести проходять (`go test ./...`)
- [ ] Всі лінтери проходять (`golangci-lint run ./...`)
- [ ] Security scan проходить (`gosec ./...`)
- [ ] Документація оновлена
- [ ] CHANGELOG.md оновлено
- [ ] Версія в git tag відповідає семантичному версіонуванню
- [ ] Release notes підготовлено
- [ ] GitHub Actions налаштовано
- [ ] Permissions налаштовано

## 🎉 Після релізу

1. ✅ Перевірте GitHub Release
2. ✅ Перевірте Docker images в GHCR
3. ✅ Протестуйте завантаження бінарника
4. ✅ Протестуйте встановлення пакету
5. ✅ Оновіть документацію (якщо потрібно)
6. ✅ Анонсуйте реліз (Twitter, Discord, etc.)

## 📚 Додаткові ресурси

- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

**Готово! Тепер ви можете створити реліз Onigirazu! 🚀**

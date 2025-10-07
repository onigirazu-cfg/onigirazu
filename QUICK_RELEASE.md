# 🚀 Швидкий старт: Створення релізу

## Найпростіший спосіб (Рекомендовано)

### 1. Використайте автоматичний скрипт

```bash
./scripts/create-release.sh
```

Скрипт автоматично:

- ✅ Перевірить тести
- ✅ Перевірить лінтери
- ✅ Перевірить security scan
- ✅ Створить git тег
- ✅ Відправить тег на GitHub
- ✅ Запустить GitHub Actions для збірки

### 2. Або вручну (3 команди)

```bash
# 1. Створіть тег
git tag -a v1.6.0 -m "Release v1.6.0"

# 2. Відправте тег
git push origin v1.6.0

# 3. Готово! GitHub Actions зробить решту
```

## Що відбувається після push тегу?

GitHub Actions автоматично:

1. **Тестування** (2-3 хв)
   - Запускає всі тести
   - Запускає security scan
   - Перевіряє код

2. **Збірка бінарників** (5-7 хв)
   - Linux: amd64, arm64, armv6, armv7, i386
   - macOS: amd64, arm64
   - Windows: amd64, i386
   - FreeBSD, OpenBSD, NetBSD: amd64, i386

3. **Створення пакетів** (2-3 хв)
   - DEB (Debian/Ubuntu)
   - RPM (RHEL/CentOS/Fedora)
   - APK (Alpine)
   - PKG (Arch Linux)

4. **Docker images** (3-5 хв)
   - Збудує multi-arch images
   - Опублікує в GHCR

5. **Публікація релізу** (1 хв)
   - Створить GitHub Release
   - Завантажить всі assets
   - Згенерує changelog

**Загальний час: ~15-20 хвилин**

## Перевірка релізу

```bash
# Відкрийте сторінку релізів
open https://github.com/onigirazu-cfg/onigirazu/releases

# Або перевірте GitHub Actions
open https://github.com/onigirazu-cfg/onigirazu/actions
```

## Тестування релізу локально (опціонально)

```bash
# Встановіть goreleaser
brew install goreleaser

# Тестова збірка (без публікації)
goreleaser release --snapshot --clean

# Перевірте результати
ls -lah dist/
```

## Що користувачі отримають?

### 📦 Бінарники

```bash
# Завантаження та встановлення
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
onigirazu --version
```

Кожен архів містить:

- ✅ Бінарник `onigirazu`
- ✅ Документацію (README, LICENSE)
- ✅ Приклади playbooks
- ✅ Конфігураційні файли (config.example.yml, inventory.example.yml)

### 📦 Пакети

```bash
# Debian/Ubuntu
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_1.6.0_linux_amd64.deb
sudo dpkg -i onigirazu_1.6.0_linux_amd64.deb

# RHEL/CentOS/Fedora
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.6.0/onigirazu_1.6.0_linux_amd64.rpm
sudo rpm -i onigirazu_1.6.0_linux_amd64.rpm
```

Пакети встановлюють:

- ✅ Бінарник в `/usr/bin/onigirazu`
- ✅ Документацію в `/usr/share/doc/onigirazu/`
- ✅ Приклади в `/usr/share/onigirazu/examples/`
- ✅ Директорії `/etc/onigirazu` та `/var/log/onigirazu`

### 🐳 Docker

```bash
# Запуск
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:v1.6.0 --version

# З playbook
docker run --rm -v $(pwd):/workspace \
  ghcr.io/onigirazu-cfg/onigirazu:v1.6.0 \
  run /workspace/playbook.yml -i /workspace/inventory.yml
```

## Troubleshooting

### Помилка: "tag already exists"

```bash
# Видаліть тег
git tag -d v1.6.0
git push origin :refs/tags/v1.6.0

# Створіть знову
git tag -a v1.6.0 -m "Release v1.6.0"
git push origin v1.6.0
```

### GitHub Actions не запустився

1. Перевірте, що тег має формат `v*` (наприклад, `v1.6.0`)
2. Перевірте permissions в Settings → Actions → General
3. Перевірте workflow файл `.github/workflows/release.yml`

### Збірка не вдалася

1. Перевірте логи в GitHub Actions
2. Переконайтеся, що всі тести проходять локально
3. Перезапустіть workflow через GitHub UI

## Checklist

Перед релізом:

- [ ] Всі тести проходять
- [ ] Всі лінтери проходять
- [ ] Security scan проходить
- [ ] Документація оновлена
- [ ] CHANGELOG.md оновлено

Після релізу:

- [ ] Перевірте GitHub Release
- [ ] Перевірте Docker images
- [ ] Протестуйте завантаження
- [ ] Анонсуйте реліз

## Додаткова інформація

Детальна документація: [RELEASE_GUIDE.md](RELEASE_GUIDE.md)

---

**Готово! Створіть свій перший реліз за 2 хвилини! 🚀**

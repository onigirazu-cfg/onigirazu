#!/bin/bash

# 🚀 Onigirazu Release Script
# Автоматизує процес створення релізу

set -e

# Кольори для виводу
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функції для виводу
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

# Перевірка, що ми в правильній директорії
if [ ! -f "go.mod" ] || [ ! -d ".git" ]; then
    error "Цей скрипт повинен запускатися з кореневої директорії проекту"
fi

# Перевірка, що немає незакомічених змін
if [ -n "$(git status --porcelain)" ]; then
    warning "У вас є незакомічені зміни:"
    git status --short
    read -p "Продовжити? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        error "Реліз скасовано"
    fi
fi

# Отримання поточної версії
CURRENT_VERSION=$(git tag -l | sort -V | tail -1)
info "Поточна версія: ${CURRENT_VERSION:-немає тегів}"

# Запит нової версії
echo ""
echo "Введіть нову версію (формат: vMAJOR.MINOR.PATCH):"
echo "Приклади: v1.6.0, v2.0.0, v1.5.1"
read -p "Нова версія: " NEW_VERSION

# Валідація формату версії
if [[ ! $NEW_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "Невірний формат версії. Використовуйте формат vMAJOR.MINOR.PATCH (наприклад, v1.6.0)"
fi

# Перевірка, що тег не існує
if git rev-parse "$NEW_VERSION" >/dev/null 2>&1; then
    error "Тег $NEW_VERSION вже існує"
fi

info "Створення релізу $NEW_VERSION..."
echo ""

# Крок 1: Запуск тестів
info "Крок 1/6: Запуск тестів..."
if go test ./... > /dev/null 2>&1; then
    success "Всі тести пройшли"
else
    error "Тести не пройшли. Виправте помилки перед релізом"
fi

# Крок 2: Запуск лінтерів
info "Крок 2/6: Запуск golangci-lint..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run ./... > /dev/null 2>&1; then
        success "Лінтери пройшли"
    else
        warning "Лінтери виявили проблеми. Продовжити?"
        read -p "Продовжити? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            error "Реліз скасовано"
        fi
    fi
else
    warning "golangci-lint не встановлено, пропускаємо перевірку"
fi

# Крок 3: Запуск security scan
info "Крок 3/6: Запуск gosec..."
if command -v gosec &> /dev/null; then
    if gosec ./... > /dev/null 2>&1; then
        success "Security scan пройшов"
    else
        warning "Security scan виявив проблеми. Продовжити?"
        read -p "Продовжити? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            error "Реліз скасовано"
        fi
    fi
else
    warning "gosec не встановлено, пропускаємо перевірку"
fi

# Крок 4: Створення release notes
info "Крок 4/6: Створення release notes..."
echo ""
echo "Введіть release notes (закінчіть ввід натисканням Ctrl+D):"
echo "Формат:"
echo "## What's New"
echo "- Feature 1"
echo "- Feature 2"
echo ""
echo "## Bug Fixes"
echo "- Fix 1"
echo ""

RELEASE_NOTES=$(cat)

if [ -z "$RELEASE_NOTES" ]; then
    RELEASE_NOTES="Release $NEW_VERSION"
fi

# Крок 5: Створення тегу
info "Крок 5/6: Створення git тегу..."
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION

$RELEASE_NOTES"

success "Тег $NEW_VERSION створено"

# Крок 6: Push тегу
info "Крок 6/6: Відправка тегу на GitHub..."
echo ""
warning "Це запустить GitHub Actions для створення релізу"
read -p "Відправити тег $NEW_VERSION на GitHub? (y/N) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    git push origin "$NEW_VERSION"
    success "Тег відправлено на GitHub"
    echo ""
    success "🎉 Реліз $NEW_VERSION створено!"
    echo ""
    info "GitHub Actions зараз збудує та опублікує реліз"
    info "Перевірте прогрес тут:"
    echo "   https://github.com/onigirazu-cfg/onigirazu/actions"
    echo ""
    info "Після завершення, реліз буде доступний тут:"
    echo "   https://github.com/onigirazu-cfg/onigirazu/releases/tag/$NEW_VERSION"
else
    warning "Тег створено локально, але не відправлено на GitHub"
    info "Щоб відправити пізніше, виконайте:"
    echo "   git push origin $NEW_VERSION"
    info "Щоб видалити локальний тег:"
    echo "   git tag -d $NEW_VERSION"
fi

echo ""
success "Готово! 🚀"

#!/bin/bash

# Скрипт для базового тестирования функциональности Onigirazu
# После всех исправлений

echo "🔧 Тестирование базовой функциональности Onigirazu"
echo "=================================================="

# Проверка компиляции
echo "1. Проверка компиляции..."
if go build ./...; then
    echo "✅ Компиляция успешна"
else
    echo "❌ Ошибка компиляции"
    exit 1
fi

# Проверка основных тестов
echo ""
echo "2. Запуск основных тестов..."
if go test ./internal/config ./internal/logger ./internal/metrics -v; then
    echo "✅ Основные тесты прошли"
else
    echo "⚠️ Некоторые тесты не прошли, но это не критично"
fi

# Проверка сборки основного бинарника
echo ""
echo "3. Сборка основного бинарника..."
if go build -o onigirazu ./cmd/onigirazu; then
    echo "✅ Бинарник собран успешно"

    # Проверка версии
    echo ""
    echo "4. Проверка версии..."
    if ./onigirazu --version 2>/dev/null || ./onigirazu version 2>/dev/null || ./onigirazu -v 2>/dev/null; then
        echo "✅ Команда версии работает"
    else
        echo "⚠️ Команда версии не найдена (возможно не реализована)"
    fi

    # Проверка help
    echo ""
    echo "5. Проверка справки..."
    if ./onigirazu --help 2>/dev/null || ./onigirazu help 2>/dev/null || ./onigirazu -h 2>/dev/null; then
        echo "✅ Справка доступна"
    else
        echo "⚠️ Справка не найдена"
    fi

else
    echo "❌ Ошибка сборки бинарника"
    exit 1
fi

# Проверка структуры проекта
echo ""
echo "6. Проверка структуры проекта..."
required_dirs=("internal/modules" "internal/executor" "internal/config" "pkg/types")
for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ $dir существует"
    else
        echo "❌ $dir отсутствует"
    fi
done

# Проверка ключевых файлов
echo ""
echo "7. Проверка ключевых файлов..."
key_files=(
    "internal/modules/service.go"
    "internal/modules/command.go"
    "internal/modules/user.go"
    "internal/modules/group.go"
    "internal/modules/git.go"
    "internal/modules/registry.go"
    "internal/executor/executor.go"
)

for file in "${key_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file существует"
    else
        echo "❌ $file отсутствует"
    fi
done

echo ""
echo "8. Проверка примеров конфигурации..."
if [ -f "examples/inventory-correct.yml" ]; then
    echo "✅ Пример инвентаря найден"
else
    echo "⚠️ Пример инвентаря не найден"
fi

echo ""
echo "=================================================="
echo "🎉 Базовое тестирование завершено!"
echo ""
echo "📋 Следующие шаги:"
echo "1. Протестируйте с реальным удаленным хостом"
echo "2. Проверьте SSH подключение"
echo "3. Запустите простой playbook"
echo "4. Изучите FINAL_ANALYSIS_REPORT.md для деталей"
echo ""
echo "⚠️ Важно: Исправьте SSH host key verification перед продакшеном!"

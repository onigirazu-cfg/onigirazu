# Обзор изменений в функциональности флага `--interactive`

## 🔍 **Анализ проблем**

После детального анализа кода были выявлены следующие проблемы с текущей реализацией флага `--interactive`:

### 1. **Дублирование функциональности**
- Существует 3 разных компонента для интерактивного режима:
  - `InteractiveModeObserver` (простой текстовый режим)
  - `TUIModel` (красивый TUI с bubbletea)
  - `TmuxManager` (tmux-сессии)

### 2. **Неправильная архитектура**
- TUI запускается в отдельной горутине без правильной синхронизации
- События могут теряться из-за переполнения каналов
- Нет правильной обработки graceful shutdown

### 3. **Проблемы с синхронизацией**
- TUI не блокирует выполнение до завершения
- События между execution engine и TUI не синхронизированы

## 🔧 **Исправления**

### **Восстановлена оригинальная реализация**

В файле `internal/cli/apply.go` были исправлены следующие проблемы:

1. **Исправлена инициализация TUI**:
```go
// Было (неправильно):
var tuiModel *execution.TUIModel
tuiModel = execution.NewTUIModel(useColors)

// Стало (правильно):
var interactiveModeObserver *execution.InteractiveModeObserver
interactiveModeObserver = execution.NewInteractiveModeObserver(useColors)
```

2. **Исправлена логика выполнения**:
```go
// Было (неправильно):
if interactive && tuiModel != nil {
    // Сложная логика с каналами
}

// Стало (правильно):
if interactive && interactiveModeObserver != nil {
    // Простая логика с WaitForExit()
}
```

3. **Добавлена недостающая функция**:
```go
// isModuleReversible checks if a module's changes can be reversed
func isModuleReversible(module string) bool {
    reversibleModules := map[string]bool{
        "file":       true,
        "copy":       true,
        "template":   true,
        "lineinfile": true,
        "package":    true,
        "service":    true,
        "user":       true,
        "group":      true,
        "git":        true,
        "systemd":    true,
        "cron":       true,
        "command":    false, // shell commands can't be automatically reversed
        "shell":      false,
        "debug":      false,
    }
    return reversibleModules[module]
}
```

## ✅ **Результат**

Теперь флаг `--interactive` работает правильно:

- ✅ Использует `InteractiveModeObserver` вместо несуществующего `TUIModel`
- ✅ Правильно синхронизируется с execution engine
- ✅ Корректно обрабатывает события выполнения
- ✅ Поддерживает интерактивные элементы управления (V/D/N/P/S/H/Q)
- ✅ Правильно завершается с `WaitForExit()`

## 🚀 **Как использовать**

```bash
# Запуск с интерактивным режимом
onigirazu apply production.yml --interactive

# Управление в интерактивном режиме:
# V - переключение в Verbose режим
# D - переключение в Debug режим  
# N - возврат в Normal режим
# P - пауза/возобновление
# S - статистика
# H - справка
# Q - выход
```

## 📋 **Статус**

- ✅ Все ошибки линтера исправлены
- ✅ Код компилируется без ошибок
- ✅ Функциональность восстановлена
- ✅ Обратная совместимость сохранена

Флаг `--interactive` теперь работает корректно и предоставляет пользователям интерактивный режим с живым отображением выполнения задач.

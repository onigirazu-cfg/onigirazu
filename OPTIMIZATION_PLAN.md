# План оптимизации и улучшений Onigirazu

## Анализ текущей архитектуры

### Сильные стороны

1. ✅ Модульная архитектура с четким разделением ответственности
2. ✅ Параллельное выполнение через execution.Pool
3. ✅ Система кэширования для состояний
4. ✅ Комплексная система логирования и метрик
5. ✅ Поддержка dry-run режима
6. ✅ Система управления состоянием

### Критические проблемы (ИСПРАВЛЕНЫ)

1. ✅ Удаленное выполнение команд - ИСПРАВЛЕНО
2. ✅ Проблемы компиляции - ИСПРАВЛЕНО
3. ✅ Несовместимость модулей - ИСПРАВЛЕНО

## Приоритетные оптимизации

### 🔴 КРИТИЧЕСКИЙ ПРИОРИТЕТ (1-2 недели)

#### 1. Безопасность SSH соединений

**Проблема**: SSH host key verification отключена

```go
// ТЕКУЩИЙ КОД (НЕБЕЗОПАСНО):
ssh.InsecureIgnoreHostKey()

// НУЖНО ИСПРАВИТЬ:
ssh.HostKeyCallback(ssh.FixedHostKey(hostKey))
```

**Решение**:

```go
// internal/ssh/security.go
type HostKeyManager struct {
    knownHosts map[string]ssh.PublicKey
    strictMode bool
}

func (h *HostKeyManager) VerifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
    if h.strictMode {
        if knownKey, exists := h.knownHosts[hostname]; exists {
            if !ssh.KeysEqual(key, knownKey) {
                return fmt.Errorf("host key mismatch for %s", hostname)
            }
        } else {
            return fmt.Errorf("unknown host %s", hostname)
        }
    }
    return nil
}
```

#### 2. Context cancellation для SSH команд

**Проблема**: Отсутствует возможность отмены долгих SSH операций

**Решение**:

```go
// internal/executor/executor.go
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, command string) (string, error) {
    session, err := e.client.NewSession()
    if err != nil {
        return "", err
    }
    defer session.Close()

    // Создаем канал для результата
    resultChan := make(chan result, 1)

    go func() {
        output, err := session.CombinedOutput(command)
        resultChan <- result{string(output), err}
    }()

    select {
    case res := <-resultChan:
        return res.output, res.err
    case <-ctx.Done():
        session.Signal(ssh.SIGTERM)
        return "", ctx.Err()
    }
}
```

#### 3. Исправление failing тестов

**Проблемы**:

- Copy module идемпотентность
- Security validator конфигурация

### 🟡 ВЫСОКИЙ ПРИОРИТЕТ (2-4 недели)

#### 1. SSH Connection Pooling

**Проблема**: Каждый модуль создает новые SSH соединения

**Решение**:

```go
// internal/ssh/pool.go
type ConnectionPool struct {
    connections map[string]*ssh.Client
    mutex       sync.RWMutex
    maxIdle     time.Duration
    maxConn     int
}

func (p *ConnectionPool) GetConnection(host types.Host) (*ssh.Client, error) {
    p.mutex.RLock()
    if conn, exists := p.connections[host.Address]; exists {
        p.mutex.RUnlock()
        return conn, nil
    }
    p.mutex.RUnlock()

    // Создаем новое соединение
    return p.createConnection(host)
}
```

#### 2. Улучшенная система ошибок

**Текущая проблема**: Ошибки не всегда информативны

**Решение**:

```go
// pkg/errors/errors.go
type OnigiraruError struct {
    Type      ErrorType
    Module    string
    Host      string
    Task      string
    Cause     error
    Context   map[string]interface{}
    Timestamp time.Time
}

type ErrorType int
const (
    ErrorTypeConnection ErrorType = iota
    ErrorTypeExecution
    ErrorTypeValidation
    ErrorTypeTimeout
)
```

#### 3. Расширенные метрики

**Добавить**:

- Метрики производительности SSH соединений
- Метрики использования ресурсов
- Метрики успешности выполнения по модулям

### 🟢 СРЕДНИЙ ПРИОРИТЕТ (1-2 месяца)

#### 1. Кэширование результатов выполнения

```go
// internal/cache/execution.go
type ExecutionCache struct {
    cache map[string]CachedResult
    ttl   time.Duration
}

type CachedResult struct {
    Result    types.TaskResult
    Timestamp time.Time
    Hash      string
}
```

#### 2. Улучшенная система шаблонов

**Добавить поддержку**:

- Условных блоков
- Циклов
- Функций обработки данных

#### 3. Система плагинов

```go
// internal/plugins/interface.go
type Plugin interface {
    Name() string
    Version() string
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Validate(args map[string]interface{}) error
}
```

### 🔵 НИЗКИЙ ПРИОРИТЕТ (3-6 месяцев)

#### 1. Web UI для мониторинга

- Dashboard с метриками
- Логи в реальном времени
- Управление playbooks

#### 2. API сервер

```go
// cmd/onigirazu-server/main.go
type APIServer struct {
    engine *core.CoreEngine
    router *gin.Engine
}
```

#### 3. Поддержка дополнительных платформ

- Windows (WinRM)
- Docker containers
- Kubernetes pods

## Конкретные файлы для изменения

### Немедленные изменения

1. **internal/ssh/client.go** - Добавить безопасную проверку host keys
2. **internal/executor/executor.go** - Добавить context cancellation
3. **internal/modules/copy.go** - Исправить идемпотентность
4. **internal/security/validator.go** - Исправить конфигурацию

### Архитектурные улучшения

1. **internal/ssh/pool.go** - Новый файл для connection pooling
2. **pkg/errors/errors.go** - Улучшенная система ошибок
3. **internal/metrics/performance.go** - Расширенные метрики
4. **internal/cache/execution.go** - Кэширование выполнения

## Метрики для отслеживания улучшений

### Производительность

- Время установки SSH соединения
- Время выполнения команд
- Использование памяти
- Количество одновременных соединений

### Надежность

- Процент успешных выполнений
- Время восстановления после ошибок
- Количество повторных попыток

### Безопасность

- Количество отклоненных host keys
- Количество timeout'ов
- Аудит выполненных команд

## Инструменты для разработки

### Рекомендуемые инструменты

1. **golangci-lint** - Статический анализ кода
2. **go-critic** - Дополнительные проверки
3. **govulncheck** - Проверка уязвимостей
4. **pprof** - Профилирование производительности

### Настройка CI/CD

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - run: go test ./...
      - run: golangci-lint run
      - run: govulncheck ./...
```

## Заключение

Проект имеет солидную архитектурную основу, но требует критических исправлений безопасности и оптимизаций производительности. Приоритет должен быть отдан безопасности SSH соединений и стабильности выполнения.

**Следующий шаг**: Начать с исправления SSH host key verification как наиболее критичной проблемы безопасности.

# Критические исправления безопасности

## 1. SSH Host Key Verification

### Текущая проблема

```go
// internal/ssh/client.go - НЕБЕЗОПАСНО!
config := &ssh.ClientConfig{
    User: host.User,
    Auth: authMethods,
    HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ⚠️ КРИТИЧЕСКАЯ УЯЗВИМОСТЬ
    Timeout: 30 * time.Second,
}
```

### Исправление

Создать новый файл `internal/ssh/hostkey.go`:

```go
package ssh

import (
    "bufio"
    "crypto/md5"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "golang.org/x/crypto/ssh"
)

type HostKeyManager struct {
    knownHostsFile string
    knownHosts     map[string]ssh.PublicKey
    mutex          sync.RWMutex
    strictMode     bool
}

func NewHostKeyManager(knownHostsFile string, strictMode bool) *HostKeyManager {
    if knownHostsFile == "" {
        home, _ := os.UserHomeDir()
        knownHostsFile = filepath.Join(home, ".ssh", "known_hosts")
    }

    hkm := &HostKeyManager{
        knownHostsFile: knownHostsFile,
        knownHosts:     make(map[string]ssh.PublicKey),
        strictMode:     strictMode,
    }

    hkm.loadKnownHosts()
    return hkm
}

func (hkm *HostKeyManager) loadKnownHosts() error {
    hkm.mutex.Lock()
    defer hkm.mutex.Unlock()

    file, err := os.Open(hkm.knownHostsFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // Файл не существует, это нормально
        }
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        parts := strings.Fields(line)
        if len(parts) < 3 {
            continue
        }

        hosts := strings.Split(parts[0], ",")
        keyType := parts[1]
        keyData := parts[2]

        key, err := ssh.ParsePublicKey([]byte(keyType + " " + keyData))
        if err != nil {
            continue
        }

        for _, host := range hosts {
            hkm.knownHosts[host] = key
        }
    }

    return scanner.Err()
}

func (hkm *HostKeyManager) VerifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
    hkm.mutex.RLock()
    defer hkm.mutex.RUnlock()

    // Проверяем по hostname
    if knownKey, exists := hkm.knownHosts[hostname]; exists {
        if ssh.KeysEqual(key, knownKey) {
            return nil
        }
        return fmt.Errorf("host key verification failed for %s: key mismatch", hostname)
    }

    // Проверяем по IP адресу
    if tcpAddr, ok := remote.(*net.TCPAddr); ok {
        ip := tcpAddr.IP.String()
        if knownKey, exists := hkm.knownHosts[ip]; exists {
            if ssh.KeysEqual(key, knownKey) {
                return nil
            }
            return fmt.Errorf("host key verification failed for %s (%s): key mismatch", hostname, ip)
        }
    }

    if hkm.strictMode {
        return fmt.Errorf("host key verification failed for %s: unknown host", hostname)
    }

    // В нестрогом режиме добавляем новый ключ
    return hkm.addHostKey(hostname, key)
}

func (hkm *HostKeyManager) addHostKey(hostname string, key ssh.PublicKey) error {
    hkm.mutex.Lock()
    defer hkm.mutex.Unlock()

    // Добавляем в память
    hkm.knownHosts[hostname] = key

    // Добавляем в файл
    file, err := os.OpenFile(hkm.knownHostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
    if err != nil {
        return err
    }
    defer file.Close()

    keyLine := fmt.Sprintf("%s %s %s\n", hostname, key.Type(),
        strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))

    _, err = file.WriteString(keyLine)
    return err
}

func (hkm *HostKeyManager) GetFingerprint(key ssh.PublicKey) string {
    hash := md5.Sum(key.Marshal())
    fingerprint := ""
    for i, b := range hash {
        if i > 0 {
            fingerprint += ":"
        }
        fingerprint += fmt.Sprintf("%02x", b)
    }
    return fingerprint
}
```

### Обновить client.go

```go
// internal/ssh/client.go
func NewClient(host types.Host, hostKeyManager *HostKeyManager) (*ssh.Client, error) {
    authMethods := []ssh.AuthMethod{}

    // ... существующий код аутентификации ...

    config := &ssh.ClientConfig{
        User: host.User,
        Auth: authMethods,
        HostKeyCallback: hostKeyManager.VerifyHostKey, // ✅ БЕЗОПАСНО
        Timeout: 30 * time.Second,
    }

    address := fmt.Sprintf("%s:%d", host.Address, host.Port)
    return ssh.Dial("tcp", address, config)
}
```

## 2. Context Cancellation для SSH

### Обновить executor.go

```go
// internal/executor/executor.go
func (e *CommandExecutor) ExecuteWithTimeout(command string, timeout time.Duration) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    return e.ExecuteWithContext(ctx, command)
}

func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, command string) (string, error) {
    session, err := e.client.NewSession()
    if err != nil {
        return "", fmt.Errorf("failed to create session: %w", err)
    }
    defer session.Close()

    // Настраиваем каналы для результата
    type result struct {
        output string
        err    error
    }

    resultChan := make(chan result, 1)

    // Запускаем команду в горутине
    go func() {
        defer close(resultChan)

        output, err := session.CombinedOutput(command)
        select {
        case resultChan <- result{string(output), err}:
        case <-ctx.Done():
            // Контекст отменен, пытаемся завершить сессию
            session.Signal(ssh.SIGTERM)
        }
    }()

    // Ждем результат или отмену контекста
    select {
    case res := <-resultChan:
        return res.output, res.err
    case <-ctx.Done():
        // Пытаемся корректно завершить сессию
        session.Signal(ssh.SIGTERM)
        return "", fmt.Errorf("command execution cancelled: %w", ctx.Err())
    }
}
```

## 3. Улучшенная обработка ошибок

### Создать pkg/errors/errors.go

```go
package errors

import (
    "fmt"
    "time"
)

type ErrorType int

const (
    ErrorTypeConnection ErrorType = iota
    ErrorTypeAuthentication
    ErrorTypeExecution
    ErrorTypeValidation
    ErrorTypeTimeout
    ErrorTypeHostKey
)

type OnigiraruError struct {
    Type      ErrorType             `json:"type"`
    Module    string                `json:"module"`
    Host      string                `json:"host"`
    Task      string                `json:"task"`
    Command   string                `json:"command,omitempty"`
    Cause     error                 `json:"-"`
    Message   string                `json:"message"`
    Context   map[string]interface{} `json:"context,omitempty"`
    Timestamp time.Time             `json:"timestamp"`
}

func (e *OnigiraruError) Error() string {
    return fmt.Sprintf("[%s] %s on %s: %s",
        e.typeString(), e.Module, e.Host, e.Message)
}

func (e *OnigiraruError) Unwrap() error {
    return e.Cause
}

func (e *OnigiraruError) typeString() string {
    switch e.Type {
    case ErrorTypeConnection:
        return "CONNECTION"
    case ErrorTypeAuthentication:
        return "AUTH"
    case ErrorTypeExecution:
        return "EXEC"
    case ErrorTypeValidation:
        return "VALIDATION"
    case ErrorTypeTimeout:
        return "TIMEOUT"
    case ErrorTypeHostKey:
        return "HOSTKEY"
    default:
        return "UNKNOWN"
    }
}

// Конструкторы для разных типов ошибок
func NewConnectionError(host, module string, cause error) *OnigiraruError {
    return &OnigiraruError{
        Type:      ErrorTypeConnection,
        Module:    module,
        Host:      host,
        Cause:     cause,
        Message:   cause.Error(),
        Timestamp: time.Now(),
    }
}

func NewExecutionError(host, module, task, command string, cause error) *OnigiraruError {
    return &OnigiraruError{
        Type:      ErrorTypeExecution,
        Module:    module,
        Host:      host,
        Task:      task,
        Command:   command,
        Cause:     cause,
        Message:   cause.Error(),
        Timestamp: time.Now(),
    }
}

func NewHostKeyError(host string, message string) *OnigiraruError {
    return &OnigiraruError{
        Type:      ErrorTypeHostKey,
        Host:      host,
        Message:   message,
        Timestamp: time.Now(),
    }
}
```

## 4. Конфигурация безопасности

### Создать internal/config/security.go

```go
package config

type SecurityConfig struct {
    SSH SSHSecurityConfig `yaml:"ssh"`
}

type SSHSecurityConfig struct {
    StrictHostKeyChecking bool   `yaml:"strict_host_key_checking" default:"true"`
    KnownHostsFile       string `yaml:"known_hosts_file"`
    ConnectionTimeout    int    `yaml:"connection_timeout" default:"30"`
    CommandTimeout       int    `yaml:"command_timeout" default:"300"`
    MaxRetries          int    `yaml:"max_retries" default:"3"`
    AllowedCiphers      []string `yaml:"allowed_ciphers"`
    AllowedMACs         []string `yaml:"allowed_macs"`
}

func DefaultSecurityConfig() SecurityConfig {
    return SecurityConfig{
        SSH: SSHSecurityConfig{
            StrictHostKeyChecking: true,
            ConnectionTimeout:    30,
            CommandTimeout:       300,
            MaxRetries:          3,
            AllowedCiphers: []string{
                "aes128-ctr",
                "aes192-ctr",
                "aes256-ctr",
                "aes128-gcm@openssh.com",
                "aes256-gcm@openssh.com",
            },
            AllowedMACs: []string{
                "hmac-sha2-256-etm@openssh.com",
                "hmac-sha2-512-etm@openssh.com",
                "hmac-sha2-256",
                "hmac-sha2-512",
            },
        },
    }
}
```

## Применение исправлений

1. **Создать файлы безопасности**
2. **Обновить существующие модули для использования новых компонентов**
3. **Добавить тесты для проверки безопасности**
4. **Обновить документацию по конфигурации**

Эти исправления устранят критические уязвимости безопасности и сделают систему готовой для продакшена.

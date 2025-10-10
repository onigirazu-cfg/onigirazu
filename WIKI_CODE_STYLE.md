# 🎨 Code Style

This guide defines the coding standards and style guidelines for Onigirazu development.

## 📋 Style Overview

### Core Principles

- **Consistency** - Follow established patterns
- **Readability** - Code should be self-documenting
- **Maintainability** - Easy to modify and extend
- **Performance** - Efficient and optimized
- **Security** - Secure by default

### Style Tools

- **gofmt** - Go formatting
- **goimports** - Import formatting
- **golangci-lint** - Linting
- **gosec** - Security scanning

---

## 🔧 Go Code Style

### Package Structure

```go
// Package documentation
// Package your_package provides functionality for...
package your_package

import (
    "context"
    "fmt"
    "time"
    
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}
```

### Naming Conventions

```go
// Package names
package modules
package cli
package engine

// Function names
func NewModule() *Module
func Execute(ctx context.Context) error
func GetName() string

// Variable names
var (
    moduleName    string
    moduleVersion string
    moduleConfig  map[string]interface{}
)

// Constant names
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
    BufferSize     = 4096
)

// Type names
type Module struct {
    name        string
    description string
    version     string
}

// Interface names
type ModuleInterface interface {
    GetName() string
    Execute(ctx context.Context) error
}
```

### Function Style

```go
// Function documentation
// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}

// Function with parameters
// YourFunctionWithParams does something with parameters.
// It takes a context, name, and age, and returns an error.
func YourFunctionWithParams(ctx context.Context, name string, age int) error {
    // Implementation
    return nil
}

// Function with return values
// YourFunctionWithReturns does something and returns values.
// It takes a context and returns a string and an error.
func YourFunctionWithReturns(ctx context.Context) (string, error) {
    // Implementation
    return "result", nil
}
```

### Error Handling

```go
// Proper error handling
func YourFunction() error {
    result, err := SomeFunction()
    if err != nil {
        return fmt.Errorf("failed to execute function: %w", err)
    }
    
    // Handle result
    return nil
}

// Error handling with context
func YourFunctionWithContext(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    result, err := SomeFunction(ctx)
    if err != nil {
        return fmt.Errorf("function failed: %w", err)
    }
    
    return nil
}

// Error handling with logging
func YourFunctionWithLogging() error {
    result, err := SomeFunction()
    if err != nil {
        log.Errorf("Failed to execute function: %v", err)
        return fmt.Errorf("failed to execute function: %w", err)
    }
    
    log.Infof("Function executed successfully: %v", result)
    return nil
}
```

### Resource Management

```go
// Proper resource management
func YourFunction() error {
    resource, err := AcquireResource()
    if err != nil {
        return err
    }
    defer ReleaseResource(resource)
    
    // Use resource
    return nil
}

// Connection management
func YourFunctionWithConnection() error {
    conn, err := GetConnection()
    if err != nil {
        return err
    }
    defer conn.Close()
    
    // Use connection
    return nil
}

// File management
func YourFunctionWithFile() error {
    file, err := os.Open("filename")
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Use file
    return nil
}
```

---

## 🧪 Testing Style

### Test Function Naming

```go
// Test function naming
func TestYourFunction(t *testing.T) {
    // Test implementation
}

func TestYourFunction_WithSpecificCase(t *testing.T) {
    // Specific test case
}

func TestYourFunction_WithError(t *testing.T) {
    // Error test case
}

func BenchmarkYourFunction(b *testing.B) {
    // Benchmark implementation
}
```

### Test Structure

```go
// Test structure
func TestYourFunction(t *testing.T) {
    // Setup
    module := NewModule()
    host := createTestHost()
    args := createTestArgs()
    
    // Execute
    result, err := module.Execute(context.Background(), host, args)
    
    // Assert
    require.NoError(t, err)
    assert.True(t, result.Changed)
    assert.Equal(t, expectedOutput, result.Output)
}
```

### Test Table Structure

```go
// Test table structure
func TestYourFunction_Table(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "case1",
            input:    "input1",
            expected: "expected1",
        },
        {
            name:     "case2",
            input:    "input2",
            expected: "expected2",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := YourFunction(tc.input)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### Mock Testing

```go
// Mock testing
func TestYourFunctionWithMock(t *testing.T) {
    // Create mock
    mockExecutor := &MockCommandExecutor{}
    mockExecutor.On("Execute", "command").Return("output", nil)
    
    // Setup
    module := NewModule()
    module.executor = mockExecutor
    
    // Execute
    result, err := module.Execute(context.Background(), host, args)
    
    // Assert
    require.NoError(t, err)
    assert.True(t, result.Changed)
    mockExecutor.AssertExpectations(t)
}
```

---

## 🔧 Documentation Style

### Package Documentation

```go
// Package documentation
// Package your_package provides functionality for...
package your_package
```

### Function Documentation

```go
// Function documentation
// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}

// Complex function documentation
// YourComplexFunction does something complex with multiple parameters.
// It takes a context, name, age, and options, and returns a result and an error.
// The context is used for cancellation and timeout.
// The name is the user's name.
// The age is the user's age.
// The options are additional configuration options.
// It returns a result containing the processed data and an error if any.
func YourComplexFunction(ctx context.Context, name string, age int, options map[string]interface{}) (*Result, error) {
    // Implementation
    return nil, nil
}
```

### Type Documentation

```go
// Type documentation
// YourStruct represents something.
type YourStruct struct {
    // Field documentation
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// Method documentation
// GetName returns the name.
func (s *YourStruct) GetName() string {
    return s.Name
}

// SetName sets the name.
func (s *YourStruct) SetName(name string) {
    s.Name = name
}
```

### Interface Documentation

```go
// Interface documentation
// YourInterface defines the interface for something.
type YourInterface interface {
    // GetName returns the name.
    GetName() string
    
    // Execute executes something.
    Execute(ctx context.Context) error
}
```

---

## 🔧 Performance Style

### Memory Management

```go
// Memory optimization
func YourFunction() {
    // Use sync.Pool for object reuse
    pool := &sync.Pool{
        New: func() interface{} {
            return &Buffer{}
        },
    }
    
    buffer := pool.Get().(*Buffer)
    defer pool.Put(buffer)
    
    // Use buffered channels
    results := make(chan Result, 100)
    
    // Use context for cancellation
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Use goroutines efficiently
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            processItem(ctx)
        }()
    }
    wg.Wait()
}
```

### Concurrency

```go
// Concurrency patterns
func YourFunction() {
    // Use channels for communication
    results := make(chan Result, 100)
    errors := make(chan error, 100)
    
    // Use goroutines for parallel processing
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            result, err := processItem(i)
            if err != nil {
                errors <- err
                return
            }
            results <- result
        }(i)
    }
    
    // Wait for completion
    go func() {
        wg.Wait()
        close(results)
        close(errors)
    }()
    
    // Process results
    for {
        select {
        case result, ok := <-results:
            if !ok {
                return
            }
            handleResult(result)
        case err, ok := <-errors:
            if !ok {
                return
            }
            handleError(err)
        }
    }
}
```

### Optimization

```go
// Performance optimization
func YourFunction() {
    // Pre-allocate slices
    results := make([]Result, 0, 1000)
    
    // Use string builders
    var sb strings.Builder
    sb.Grow(1024) // Pre-allocate capacity
    
    // Use byte slices efficiently
    buffer := make([]byte, 0, 4096)
    
    // Clean up resources
    defer func() {
        // Clean up
        buffer = nil
        results = nil
    }()
}
```

---

## 🔧 Security Style

### Input Validation

```go
// Input validation
func YourFunction(input string) error {
    // Validate input
    if input == "" {
        return fmt.Errorf("input cannot be empty")
    }
    
    if len(input) > 1000 {
        return fmt.Errorf("input too long")
    }
    
    // Sanitize input
    input = strings.TrimSpace(input)
    input = strings.ToLower(input)
    
    // Process input
    return processInput(input)
}
```

### Secure Coding

```go
// Secure coding practices
func YourFunction() error {
    // Use secure random
    randomBytes := make([]byte, 32)
    _, err := rand.Read(randomBytes)
    if err != nil {
        return err
    }
    
    // Use secure hashing
    hash := sha256.Sum256(randomBytes)
    
    // Use secure comparison
    if !bytes.Equal(hash1, hash2) {
        return fmt.Errorf("hash mismatch")
    }
    
    return nil
}
```

### Error Handling

```go
// Secure error handling
func YourFunction() error {
    result, err := SomeFunction()
    if err != nil {
        // Log error securely
        log.Errorf("Function failed: %v", err)
        
        // Return generic error
        return fmt.Errorf("operation failed")
    }
    
    return nil
}
```

---

## 🔧 Configuration Style

### Configuration Structure

```go
// Configuration structure
type Config struct {
    // Server configuration
    Server ServerConfig `yaml:"server"`
    
    // Database configuration
    Database DatabaseConfig `yaml:"database"`
    
    // Logging configuration
    Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
}

type DatabaseConfig struct {
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
    File   string `yaml:"file"`
}
```

### Configuration Loading

```go
// Configuration loading
func LoadConfig(path string) (*Config, error) {
    // Read file
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    // Parse YAML
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    // Validate config
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return &config, nil
}
```

---

## 🔧 Best Practices

### Code Organization

```go
// Code organization
package your_package

import (
    "context"
    "fmt"
    "time"
    
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Constants
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

// Variables
var (
    defaultConfig = &Config{
        Timeout: DefaultTimeout,
        Retries: MaxRetries,
    }
)

// Types
type YourStruct struct {
    // Fields
}

// Functions
func YourFunction() error {
    // Implementation
    return nil
}
```

### Error Handling

```go
// Error handling best practices
func YourFunction() error {
    // Validate inputs
    if err := validateInputs(); err != nil {
        return fmt.Errorf("invalid inputs: %w", err)
    }
    
    // Execute operation
    result, err := executeOperation()
    if err != nil {
        return fmt.Errorf("operation failed: %w", err)
    }
    
    // Handle result
    if err := handleResult(result); err != nil {
        return fmt.Errorf("failed to handle result: %w", err)
    }
    
    return nil
}
```

### Testing

```go
// Testing best practices
func TestYourFunction(t *testing.T) {
    // Setup
    module := NewModule()
    host := createTestHost()
    args := createTestArgs()
    
    // Execute
    result, err := module.Execute(context.Background(), host, args)
    
    // Assert
    require.NoError(t, err)
    assert.True(t, result.Changed)
    assert.Equal(t, expectedOutput, result.Output)
    
    // Cleanup
    cleanupTest()
}
```

---

## 📚 Related Documentation

- [Development Setup](Development-Setup) - Development environment
- [Testing](Testing) - Testing guide
- [Contributing](Contributing) - Contribution guidelines
- [API Reference](API-Reference) - API documentation

---

## 🎯 Summary

### Style Features

- **🎨 Consistent** - Follow established patterns
- **📖 Readable** - Self-documenting code
- **🔧 Maintainable** - Easy to modify
- **⚡ Performant** - Optimized code
- **🔒 Secure** - Secure by default

### Style Benefits

- **🚀 Productivity** - Faster development
- **🔧 Quality** - Better code quality
- **📈 Maintainability** - Easier maintenance
- **🔒 Security** - Secure code
- **📚 Documentation** - Well-documented code

---

**🎨 Consistent code style ensures Onigirazu is maintainable and professional!**

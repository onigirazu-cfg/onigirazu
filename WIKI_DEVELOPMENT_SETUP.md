# 🛠️ Development Setup

This guide helps you set up a development environment for contributing to Onigirazu.

## 📋 Prerequisites

### Required Software

- **Go 1.24.0+** - Latest Go version
- **Git** - Version control
- **Make** - Build automation
- **Docker** - Container testing (optional)

### Development Tools

- **golangci-lint** - Go linting
- **gosec** - Security scanning
- **goimports** - Import formatting
- **gofmt** - Code formatting

---

## 🚀 Quick Setup

### 1. Clone Repository

```bash
# Fork on GitHub, then clone
git clone https://github.com/your-username/onigirazu.git
cd onigirazu
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
```

### 3. Build Project

```bash
# Build
go build -o onigirazu cmd/onigirazu/main.go

# Or use Make
make build
```

### 4. Run Tests

```bash
# Run all tests
go test ./...

# Or use Make
make test
```

---

## 🔧 Development Environment

### Go Workspace

```bash
# Set up Go workspace
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Verify Go installation
go version
```

### IDE Configuration

#### VS Code

Install extensions:
- **Go** - Official Go extension
- **Go Test** - Go testing support
- **Go Outliner** - Code navigation
- **Go Doc** - Documentation support

#### GoLand

Configure:
- **Go SDK** - Set Go SDK path
- **Go Modules** - Enable Go modules
- **Code Style** - Configure formatting
- **Run Configurations** - Set up test configurations

---

## 🏗️ Build System

### Make Commands

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Security scan
make security

# Clean
make clean

# Install
make install

# All checks
make check
```

### Build Targets

```makefile
# Makefile targets
.PHONY: build test lint security clean install check

build:
	go build -o onigirazu cmd/onigirazu/main.go

test:
	go test -v ./...

lint:
	golangci-lint run

security:
	gosec ./...

clean:
	rm -f onigirazu
	go clean ./...

install:
	go install ./cmd/onigirazu

check: lint security test
```

---

## 🧪 Testing

### Unit Tests

```bash
# Run unit tests
go test ./...

# Run specific package tests
go test ./internal/modules

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Integration Tests

```bash
# Run integration tests
go test -tags=integration ./tests/...

# Run with Docker
docker-compose up -d
go test -tags=integration ./tests/...
docker-compose down
```

### Test Structure

```
tests/
├── unit/           # Unit tests
├── integration/    # Integration tests
├── performance/    # Performance tests
└── security/       # Security tests
```

---

## 🔍 Code Quality

### Linting

```bash
# Run linter
golangci-lint run

# Run with specific rules
golangci-lint run --enable=gofmt,goimports,vet

# Fix issues
golangci-lint run --fix
```

### Security Scanning

```bash
# Run security scan
gosec ./...

# Run with specific rules
gosec -include=G401,G402 ./...

# Generate report
gosec -fmt json -out security-report.json ./
```

### Code Formatting

```bash
# Format code
go fmt ./...

# Format imports
goimports -w .

# Check formatting
gofmt -d .
```

---

## 📦 Module Development

### Creating New Modules

```go
// internal/modules/your_module.go
package modules

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourModule struct {
    BaseModule
}

func NewYourModule() *YourModule {
    return &YourModule{
        BaseModule: BaseModule{
            name:        "your_module",
            description: "Your module description",
        },
    }
}

func (m *YourModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Implementation
    return types.TaskResult{
        Changed: true,
        Output:  map[string]interface{}{"result": "success"},
    }, nil
}
```

### Module Testing

```go
// internal/modules/your_module_test.go
package modules

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestYourModule_Execute(t *testing.T) {
    module := NewYourModule()
    host := types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
    }
    args := map[string]interface{}{
        "param": "value",
    }
    
    result, err := module.Execute(context.Background(), host, args)
    
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    
    if !result.Changed {
        t.Error("Expected changed to be true")
    }
}
```

---

## 🔧 CLI Development

### Adding New Commands

```go
// internal/cli/your_command.go
package cli

import (
    "github.com/spf13/cobra"
)

var yourCmd = &cobra.Command{
    Use:   "your-command",
    Short: "Your command description",
    Long:  "Your command long description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(yourCmd)
}
```

### Command Testing

```go
// internal/cli/your_command_test.go
package cli

import (
    "testing"
    "github.com/spf13/cobra"
)

func TestYourCommand(t *testing.T) {
    cmd := &cobra.Command{
        Use: "test-command",
        RunE: func(cmd *cobra.Command, args []string) error {
            return nil
        },
    }
    
    err := cmd.Execute()
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
}
```

---

## 🐳 Docker Development

### Development Container

```dockerfile
# Dockerfile.dev
FROM golang:1.24-alpine

RUN apk add --no-cache git make

WORKDIR /app
COPY . .

RUN go mod download
RUN go install golangci-lint/cmd/golangci-lint@latest
RUN go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

CMD ["make", "check"]
```

### Docker Compose

```yaml
# docker-compose.dev.yml
version: '3.8'
services:
  onigirazu:
    build:
      context: .
      dockerfile: Dockerfile.dev
    volumes:
      - .:/app
    working_dir: /app
    command: ["make", "check"]
```

### Development Commands

```bash
# Build development container
docker build -f Dockerfile.dev -t onigirazu-dev .

# Run development container
docker run --rm -it -v $(pwd):/app onigirazu-dev

# Use Docker Compose
docker-compose -f docker-compose.dev.yml up
```

---

## 🔧 CI/CD Pipeline

### GitHub Actions

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.24
      
      - name: Install dependencies
        run: go mod download
      
      - name: Run tests
        run: go test -v ./...
      
      - name: Run linter
        run: golangci-lint run
      
      - name: Run security scan
        run: gosec ./...
```

### Local CI

```bash
# Run CI locally
make ci

# Or individual steps
make test
make lint
make security
make build
```

---

## 📚 Documentation

### API Documentation

```bash
# Generate API docs
go run scripts/docgen/main.go

# Serve docs locally
python -m http.server 8000 -d docs/api
```

### Code Documentation

```go
// Package your_package provides functionality for...
package your_package

// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}
```

---

## 🎯 Best Practices

### Code Style

```go
// Use consistent formatting
go fmt ./...

// Use consistent imports
goimports -w .

// Follow Go conventions
// - Use camelCase for variables
// - Use PascalCase for exported functions
// - Use descriptive names
// - Add comments for exported functions
```

### Testing

```go
// Write comprehensive tests
func TestFunction(t *testing.T) {
    // Test cases
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "expected1"},
        {"case2", "input2", "expected2"},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := Function(tc.input)
            if result != tc.expected {
                t.Errorf("Expected %s, got %s", tc.expected, result)
            }
        })
    }
}
```

### Error Handling

```go
// Handle errors properly
result, err := SomeFunction()
if err != nil {
    return fmt.Errorf("failed to execute function: %w", err)
}

// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := SomeFunction(ctx)
if err != nil {
    return fmt.Errorf("function failed: %w", err)
}
```

---

## 🚨 Troubleshooting

### Common Issues

#### Go Version Mismatch
```bash
# Check Go version
go version

# Update Go
# Follow Go installation guide for your platform
```

#### Module Issues
```bash
# Clean module cache
go clean -modcache

# Download modules
go mod download

# Verify modules
go mod verify
```

#### Test Issues
```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestSpecificFunction ./...

# Run tests with race detection
go test -race ./...
```

### Debug Development

```bash
# Enable debug output
export ONIGIRAZU_DEBUG=true

# Run with debug
go run cmd/onigirazu/main.go --debug

# Use delve debugger
dlv debug cmd/onigirazu/main.go
```

---

## 📚 Resources

### Development Resources

- [Go Documentation](https://golang.org/doc/)
- [Go Testing](https://golang.org/pkg/testing/)
- [GitHub Actions](https://docs.github.com/en/actions)
- [Docker Documentation](https://docs.docker.com/)

### Onigirazu Resources

- [Architecture](Architecture) - System architecture
- [Modules](Modules) - Module development
- [API Reference](API-Reference) - API documentation
- [Contributing](Contributing) - Contribution guidelines

---

## 🎯 Summary

### Development Checklist

1. **✅ Set up environment** - Install Go and tools
2. **✅ Clone repository** - Fork and clone
3. **✅ Install dependencies** - Go modules and tools
4. **✅ Build project** - Verify build works
5. **✅ Run tests** - Ensure tests pass
6. **✅ Set up IDE** - Configure development environment
7. **✅ Start coding** - Begin development

### Development Tools

- **🔧 Go toolchain** - Core development tools
- **🧪 Testing** - Unit and integration tests
- **🔍 Linting** - Code quality tools
- **🔒 Security** - Security scanning
- **🐳 Docker** - Container development
- **📚 Documentation** - API and code docs

---

**🛠️ Your development environment is now ready for Onigirazu development!**


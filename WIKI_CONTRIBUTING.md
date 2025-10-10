# 🤝 Contributing

We welcome contributions to Onigirazu! This guide will help you get started with contributing to the project.

## 📋 How to Contribute

### Types of Contributions

- **🐛 Bug Reports** - Report bugs and issues
- **✨ Feature Requests** - Suggest new features
- **📚 Documentation** - Improve documentation
- **🔧 Code Contributions** - Submit code changes
- **🧪 Testing** - Help with testing
- **💬 Community** - Help other users

---

## 🚀 Getting Started

### 1. Fork the Repository

```bash
# Fork on GitHub, then clone
git clone https://github.com/your-username/onigirazu.git
cd onigirazu
```

### 2. Set Up Development Environment

```bash
# Install Go (1.24.0 or later)
go version

# Install dependencies
go mod download

# Build the project
go build -o onigirazu cmd/onigirazu/main.go

# Run tests
go test ./...
```

### 3. Create a Branch

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Or bugfix branch
git checkout -b bugfix/your-bugfix-name
```

---

## 🔧 Development Setup

### Prerequisites

- **Go 1.24.0+** - Latest Go version
- **Git** - Version control
- **Make** - Build automation
- **Docker** - Container testing (optional)

### Development Tools

```bash
# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
```

### Build Commands

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
```

---

## 📝 Code Contributions

### Code Style

- **Go formatting** - Use `go fmt` and `goimports`
- **Linting** - Follow `golangci-lint` rules
- **Documentation** - Document all public functions
- **Testing** - Write tests for new features

### Code Structure

```
onigirazu/
├── cmd/           # CLI commands
├── internal/      # Internal packages
│   ├── cli/       # CLI implementation
│   ├── engine/    # Execution engines
│   ├── modules/   # Built-in modules
│   ├── ssh/       # SSH communication
│   └── state/     # State management
├── pkg/           # Public packages
│   ├── types/     # Type definitions
│   └── utils/     # Utility functions
└── docs/          # Documentation
```

### Adding New Modules

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

### Adding New CLI Commands

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

---

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/modules

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Writing Tests

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

### Integration Tests

```bash
# Run integration tests
go test -tags=integration ./tests/...

# Run with Docker
docker-compose up -d
go test -tags=integration ./tests/...
docker-compose down
```

---

## 📚 Documentation

### Documentation Types

- **API Documentation** - Code documentation
- **User Guides** - User-facing documentation
- **Developer Guides** - Developer documentation
- **Wiki Pages** - Community documentation

### Writing Documentation

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

### Wiki Contributions

- **User Guides** - Help users get started
- **Tutorials** - Step-by-step guides
- **Examples** - Code examples and use cases
- **Troubleshooting** - Common issues and solutions

---

## 🐛 Bug Reports

### Before Reporting

1. **Check existing issues** - Search for similar issues
2. **Test latest version** - Ensure you're using the latest version
3. **Gather information** - Collect relevant details

### Bug Report Template

```markdown
## Bug Description
Brief description of the bug

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- OS: [e.g., Ubuntu 20.04]
- Onigirazu version: [e.g., v1.26.1]
- Go version: [e.g., 1.24.0]

## Additional Information
Any additional context or screenshots
```

---

## ✨ Feature Requests

### Before Requesting

1. **Check existing features** - Ensure feature doesn't exist
2. **Search discussions** - Check if already discussed
3. **Consider alternatives** - Look for workarounds

### Feature Request Template

```markdown
## Feature Description
Brief description of the feature

## Use Case
Why is this feature needed?

## Proposed Solution
How should this feature work?

## Alternatives Considered
What other approaches were considered?

## Additional Context
Any additional context or examples
```

---

## 🔧 Pull Requests

### Before Submitting

1. **Fork repository** - Create your fork
2. **Create branch** - Use descriptive branch name
3. **Write tests** - Add tests for new features
4. **Update documentation** - Update relevant docs
5. **Run tests** - Ensure all tests pass

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass
- [ ] New tests added
- [ ] Manual testing performed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes
```

### Review Process

1. **Automated checks** - CI/CD pipeline runs
2. **Code review** - Maintainers review code
3. **Testing** - Manual testing if needed
4. **Merge** - Approved changes are merged

---

## 🎯 Contribution Guidelines

### Code Standards

- **Go formatting** - Use `go fmt`
- **Linting** - Follow linting rules
- **Documentation** - Document public APIs
- **Testing** - Write comprehensive tests

### Commit Messages

```bash
# Use conventional commits
feat: add new module for package management
fix: resolve SSH connection timeout issue
docs: update installation guide
test: add integration tests for modules
```

### Branch Naming

```bash
# Feature branches
feature/add-new-module
feature/improve-performance

# Bugfix branches
bugfix/ssh-connection-issue
bugfix/memory-leak-fix

# Documentation branches
docs/update-user-guide
docs/add-api-reference
```

---

## 🏆 Recognition

### Contributors

We recognize contributors in several ways:

- **Contributor list** - Listed in README
- **Release notes** - Mentioned in release notes
- **GitHub profile** - Contributions visible on GitHub
- **Community recognition** - Acknowledged in discussions

### Contribution Levels

- **🥉 Bronze** - 1-5 contributions
- **🥈 Silver** - 6-15 contributions
- **🥇 Gold** - 16+ contributions
- **💎 Diamond** - Major contributions

---

## 📞 Getting Help

### Community Support

- **GitHub Discussions** - Ask questions and get help
- **Discord** - Real-time chat with community
- **Stack Overflow** - Technical questions
- **Reddit** - General discussions

### Professional Support

- **Enterprise Support** - Available for enterprise customers
- **Training** - Onigirazu training and certification
- **Consulting** - Professional consulting services

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
- [Troubleshooting](Troubleshooting) - Common issues

---

## 🎯 Summary

### Quick Start Checklist

1. **✅ Fork repository** - Create your fork
2. **✅ Set up environment** - Install Go and tools
3. **✅ Create branch** - Use descriptive name
4. **✅ Make changes** - Follow coding standards
5. **✅ Write tests** - Add comprehensive tests
6. **✅ Update docs** - Update relevant documentation
7. **✅ Submit PR** - Create pull request
8. **✅ Respond to feedback** - Address review comments

### Contribution Types

- **🐛 Bug fixes** - Fix reported issues
- **✨ Features** - Add new functionality
- **📚 Documentation** - Improve docs
- **🧪 Testing** - Add test coverage
- **💬 Community** - Help other users

---

**🤝 Thank you for contributing to Onigirazu!**

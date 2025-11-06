# Contributing to Onigirazu

Thank you for your interest in contributing to Onigirazu! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

- Go 1.23 or later
- Git
- Make (optional, for using Makefile commands)
- golangci-lint (for code linting)

### Setting up the Development Environment

1. Fork the repository
2. Clone your fork:

   ```bash
   git clone https://github.com/your-username/onigirazu.git
   cd onigirazu
   ```

3. Add the upstream remote:

   ```bash
   git remote add upstream https://github.com/onigirazu-cfg/onigirazu.git
   ```

4. Install dependencies:

   ```bash
   go mod download
   ```

5. Install golangci-lint:

   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

6. Run tests to ensure everything works:

   ```bash
   go test ./...
   ```

## Development Workflow

### Branching Strategy

- `main` - stable release branch
- `develop` - development branch for next release
- `feature/*` - feature branches
- `bugfix/*` - bug fix branches
- `hotfix/*` - critical fixes for production

### Making Changes

1. Create a new branch from `develop`:

   ```bash
   git checkout develop
   git pull upstream develop
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following our coding standards

3. Write or update tests for your changes

4. Run the test suite:

   ```bash
   go test ./...
   ```

## Module Development

### ⚡ Using Module Scaffolding (Recommended)

For new module development, use the **Module Scaffolding Tool** to generate boilerplate:

```bash
cd /Users/denys.rastiegaiev/work/onigirazu_project/onigirazu
go run ./scripts/module_scaffold \
  -name my_new_module \
  -desc "Module description" \
  -params "param1,param2"
```

This generates:

- ✅ Complete module implementation
- ✅ Unit tests with table-driven approach
- ✅ Idempotency tests
- ✅ Benchmark tests

**Benefits**: Save hours of boilerplate writing and follow best practices automatically!

📖 **Full Guide**: See [Module Scaffolding Guide](docs/MODULE_SCAFFOLDING_GUIDE.md)

### Testing Generated Modules

```bash
# Run tests
go test ./internal/modules -run MyModule -v

# Check coverage
go test ./internal/modules -run MyModule -cover

# Run benchmarks
go test ./internal/modules -bench=MyModule -benchmem

# Target: 100% coverage for new modules
```

### Executor Safety Requirements

When developing modules, always follow these critical patterns:

✅ **DO Use BaseExecutorModule** (Recommended):

```go
type MyModule struct {
    *BaseExecutorModule
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    return m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (types.TaskResult, error) {
        output, err := exec.Execute("command")
        // Process output...
    })
}
```

❌ **DON'T Cache Executors**:

```go
type MyModule struct {
    *BaseModule
    executor *executor.CommandExecutor  // ❌ BUG: All hosts use first host's connection!
}
```

📖 **Full Details**: See [Module Development Guide](docs/MODULE_DEVELOPMENT_GUIDE.md)

5. Install golangci-lint (if not already installed):

   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

6. Run linting:

   ```bash
   golangci-lint run
   ```

7. Commit your changes with a descriptive message:

   ```bash
   git commit -m "feat: add new feature description"
   ```

8. Push to your fork:

   ```bash
   git push origin feature/your-feature-name
   ```

9. Create a Pull Request

## Coding Standards

### Go Style Guide

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Use `golint` and `go vet` to check for issues
- Write meaningful variable and function names
- Add comments for exported functions and types

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

### Testing

- Write unit tests for all new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Mock external dependencies
- Test both success and error cases

Example test structure:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        // test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Documentation

- Update documentation for any new features
- Include code examples in documentation
- Update README.md if necessary
- Add inline comments for complex logic

## Pull Request Process

1. Ensure your PR has a clear title and description
2. Link any related issues
3. Ensure all tests pass
4. Ensure code coverage doesn't decrease
5. Request review from maintainers
6. Address any feedback promptly

### PR Checklist

- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings introduced
- [ ] Tests added for new functionality

## Reporting Issues

When reporting issues, please include:

- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages

## Feature Requests

For feature requests, please provide:

- Clear description of the feature
- Use case and motivation
- Possible implementation approach
- Any alternatives considered

## Getting Help

- Check existing issues and documentation
- Join our discussions on GitHub
- Ask questions in issues with the `question` label

## Recognition

Contributors will be recognized in:

- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributors section

Thank you for contributing to Onigirazu!

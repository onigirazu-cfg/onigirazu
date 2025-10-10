# 🔄 Pull Request Process

This guide outlines the process for submitting pull requests to Onigirazu.

## 📋 Process Overview

### Pull Request Lifecycle

1. **Preparation** - Fork, branch, and develop
2. **Submission** - Create pull request
3. **Review** - Code review and feedback
4. **Testing** - Automated and manual testing
5. **Merge** - Approved changes are merged

### Quality Gates

- **Code quality** - Follows style guidelines
- **Testing** - Comprehensive test coverage
- **Documentation** - Updated documentation
- **Security** - Security scan passes
- **Performance** - No performance regressions

---

## 🚀 Preparation

### 1. Fork Repository

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

# Install development tools
go install golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install golang.org/x/tools/cmd/goimports@latest
```

### 3. Create Feature Branch

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Or bugfix branch
git checkout -b bugfix/your-bugfix-name

# Or documentation branch
git checkout -b docs/your-documentation-update
```

### 4. Develop Your Changes

```bash
# Make your changes
# ...

# Test your changes
go test ./...

# Run linter
golangci-lint run

# Run security scan
gosec ./...

# Format code
go fmt ./...
goimports -w .
```

---

## 📝 Submission

### 1. Commit Your Changes

```bash
# Add changes
git add .

# Commit with conventional commit message
git commit -m "feat: add new module for package management"

# Push to your fork
git push origin feature/your-feature-name
```

### 2. Create Pull Request

1. **Go to GitHub** - Navigate to your fork
2. **Click "New Pull Request"** - Create new PR
3. **Fill out template** - Complete PR template
4. **Add reviewers** - Request specific reviewers
5. **Submit** - Create the pull request

### 3. Pull Request Template

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
- [ ] Security scan passes
- [ ] Performance tests pass
```

---

## 🔍 Review Process

### 1. Automated Checks

The following checks run automatically:

- **Build** - Code compiles successfully
- **Tests** - All tests pass
- **Linting** - Code follows style guidelines
- **Security** - Security scan passes
- **Performance** - No performance regressions

### 2. Code Review

Reviewers will check:

- **Code quality** - Follows best practices
- **Functionality** - Works as expected
- **Testing** - Adequate test coverage
- **Documentation** - Updated documentation
- **Security** - No security issues
- **Performance** - No performance issues

### 3. Review Feedback

Reviewers may request:

- **Code changes** - Improve implementation
- **Additional tests** - Increase test coverage
- **Documentation** - Update documentation
- **Performance** - Optimize performance
- **Security** - Address security concerns

---

## 🧪 Testing

### 1. Unit Tests

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

### 2. Integration Tests

```bash
# Run integration tests
go test -tags=integration ./...

# Run with Docker
docker-compose up -d
go test -tags=integration ./...
docker-compose down
```

### 3. Performance Tests

```bash
# Run benchmarks
go test -bench=. ./...

# Run performance tests
go test -tags=performance ./...
```

### 4. Security Tests

```bash
# Run security scan
gosec ./...

# Run with specific rules
gosec -include=G401,G402,G403,G404 ./...
```

---

## 📚 Documentation

### 1. Code Documentation

```go
// Document all public functions
// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}
```

### 2. API Documentation

```go
// API documentation
// @title Onigirazu API
// @description Onigirazu REST API
// @version 1.0
// @host localhost:8080
// @BasePath /api/v1

// @Summary List playbooks
// @Description Get list of all playbooks
// @Tags playbooks
// @Accept json
// @Produce json
// @Success 200 {array} Playbook
// @Router /playbooks [get]
func (api *RESTAPI) listPlaybooks(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### 3. User Documentation

Update relevant documentation:

- **README.md** - Project overview
- **CHANGELOG.md** - Release notes
- **docs/** - User documentation
- **examples/** - Code examples

---

## 🔧 Code Quality

### 1. Style Guidelines

```go
// Follow Go style guidelines
package your_package

import (
    "context"
    "fmt"
    "time"
    
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// YourFunction does something useful.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}
```

### 2. Linting

```bash
# Run linter
golangci-lint run

# Run with specific rules
golangci-lint run --enable=gofmt,goimports,vet

# Fix issues
golangci-lint run --fix
```

### 3. Formatting

```bash
# Format code
go fmt ./...

# Format imports
goimports -w .

# Check formatting
gofmt -d .
```

---

## 🔒 Security

### 1. Security Scanning

```bash
# Run security scan
gosec ./...

# Run with specific rules
gosec -include=G401,G402,G403,G404 ./...

# Generate security report
gosec -fmt json -out security.json ./
```

### 2. Security Best Practices

- **Input validation** - Validate all inputs
- **Output encoding** - Encode outputs properly
- **Error handling** - Handle errors securely
- **Resource management** - Manage resources properly
- **Authentication** - Implement proper authentication

### 3. Security Checklist

- [ ] **Input validation** - All inputs validated
- [ ] **Output encoding** - Outputs properly encoded
- [ ] **Error handling** - Errors handled securely
- [ ] **Resource management** - Resources managed properly
- [ ] **Authentication** - Proper authentication
- [ ] **Authorization** - Proper authorization
- [ ] **Data protection** - Data protected properly
- [ ] **Audit logging** - Audit logs implemented

---

## ⚡ Performance

### 1. Performance Testing

```bash
# Run benchmarks
go test -bench=. ./...

# Run performance tests
go test -tags=performance ./...

# Profile performance
go test -cpuprofile=cpu.prof ./...
go test -memprofile=mem.prof ./...
```

### 2. Performance Optimization

- **Memory usage** - Optimize memory usage
- **CPU usage** - Optimize CPU usage
- **Network I/O** - Optimize network operations
- **Disk I/O** - Optimize disk operations
- **Concurrency** - Use concurrency effectively

### 3. Performance Checklist

- [ ] **Memory usage** - Memory usage optimized
- [ ] **CPU usage** - CPU usage optimized
- [ ] **Network I/O** - Network operations optimized
- [ ] **Disk I/O** - Disk operations optimized
- [ ] **Concurrency** - Concurrency used effectively
- [ ] **Caching** - Caching implemented where appropriate
- [ ] **Connection pooling** - Connection pooling used
- [ ] **Resource cleanup** - Resources cleaned up properly

---

## 🔄 Merge Process

### 1. Approval Requirements

- **Code review** - At least 2 approvals
- **Automated checks** - All checks pass
- **Testing** - All tests pass
- **Documentation** - Documentation updated
- **Security** - Security scan passes
- **Performance** - Performance tests pass

### 2. Merge Types

- **Squash and merge** - For feature branches
- **Merge commit** - For complex changes
- **Rebase and merge** - For clean history

### 3. Post-Merge

- **Clean up** - Delete feature branch
- **Update documentation** - Update relevant docs
- **Notify team** - Notify team of changes
- **Monitor** - Monitor for issues

---

## 🚨 Common Issues

### 1. Build Failures

```bash
# Check Go version
go version

# Clean module cache
go clean -modcache

# Download modules
go mod download

# Verify modules
go mod verify
```

### 2. Test Failures

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestSpecificFunction ./...

# Run tests with race detection
go test -race ./...
```

### 3. Linting Issues

```bash
# Run linter
golangci-lint run

# Run with specific rules
golangci-lint run --enable=gofmt,goimports,vet

# Fix issues
golangci-lint run --fix
```

### 4. Security Issues

```bash
# Run security scan
gosec ./...

# Run with specific rules
gosec -include=G401,G402,G403,G404 ./...

# Check dependencies
go list -json -m all | nancy sleuth
```

---

## 📚 Best Practices

### 1. Pull Request Best Practices

- **Small changes** - Keep PRs focused and small
- **Clear description** - Provide clear description
- **Good commit messages** - Use conventional commits
- **Test coverage** - Add tests for new features
- **Documentation** - Update documentation
- **Security** - Consider security implications
- **Performance** - Consider performance implications

### 2. Code Review Best Practices

- **Be constructive** - Provide constructive feedback
- **Be specific** - Point out specific issues
- **Be helpful** - Suggest improvements
- **Be respectful** - Be respectful in feedback
- **Be thorough** - Review thoroughly

### 3. Testing Best Practices

- **Unit tests** - Write unit tests
- **Integration tests** - Write integration tests
- **Performance tests** - Write performance tests
- **Security tests** - Write security tests
- **Manual testing** - Perform manual testing

---

## 📚 Related Documentation

- [Contributing](Contributing) - Contribution guidelines
- [Development Setup](Development-Setup) - Development environment
- [Testing](Testing) - Testing guide
- [Code Style](Code-Style) - Code style guidelines
- [Security](Security) - Security guidelines

---

## 🎯 Summary

### Process Features

- **🔄 Structured** - Clear process steps
- **🔍 Quality gates** - Multiple quality checks
- **🧪 Testing** - Comprehensive testing
- **📚 Documentation** - Updated documentation
- **🔒 Security** - Security considerations

### Process Benefits

- **🚀 Efficiency** - Streamlined process
- **🔧 Quality** - High code quality
- **📈 Collaboration** - Better collaboration
- **🔒 Security** - Secure code
- **📚 Documentation** - Well-documented changes

---

**🔄 The pull request process ensures high-quality contributions to Onigirazu!**

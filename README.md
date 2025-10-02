# Onigirazu Documentation

Welcome to the Onigirazu documentation directory! This contains comprehensive documentation for the Onigirazu automation tool.

## 📚 Documentation Structure

### API Documentation

- **[api/](api/)** - Auto-generated API documentation
  - **[index.html](api/index.html)** - Beautiful HTML documentation (open in browser)
  - **[README.md](api/README.md)** - API documentation overview
  - **[pkg/](api/pkg/)** - Public package documentation
  - **[internal/](api/internal/)** - Internal package documentation

### Design Documentation

- **[logo.md](logo.md)** - Logo design guidelines and assets
- **[logo.png](logo.png)** - Main project logo (200px)
- **[logo.svg](logo.svg)** - Vector version of the logo

## 🚀 Quick Start

### View API Documentation

1. **HTML Documentation** (Recommended):

   ```bash
   make docs-open
   # or manually: open docs/api/index.html
   ```

2. **Interactive Documentation Server**:

   ```bash
   make docs-serve
   # Opens http://localhost:8080
   ```

3. **Command Line**:

   ```bash
   go doc github.com/onigirazu-cfg/onigirazu/pkg/types
   ```

### Generate Documentation

To regenerate the API documentation:

```bash
make docs-generate
```

This will:

- Generate markdown documentation for all packages
- Create a beautiful HTML documentation page
- Update the API index

## 📖 Available Documentation

### Core Types (`pkg/types`)

- **Playbook** - YAML playbook structure
- **Task** - Individual task definition
- **Host** - Target host configuration
- **TaskResult** - Execution results
- **ExecutionContext** - Runtime context

### Configuration (`internal/config`)

- Configuration loading and validation
- Environment variable handling
- Default settings

### Core Engine (`internal/core`)

- Main execution engine
- Task orchestration
- Error handling

### Modules (`internal/modules`)

- Built-in modules (command, file, template, etc.)
- Module interface and implementation
- Custom module development

### Parser (`internal/parser`)

- YAML parsing and validation
- Playbook structure parsing
- Variable interpolation

### Workflow (`internal/workflow`)

- Workflow orchestration
- Event handling
- Progress tracking

## 🛠 Documentation Commands

| Command | Description |
|---------|-------------|
| `make docs` | Show documentation info |
| `make docs-generate` | Generate API documentation |
| `make docs-serve` | Start documentation server |
| `make docs-open` | Open HTML docs in browser |

## 📝 Contributing to Documentation

When adding new features or modifying existing code:

1. **Add Go doc comments** to all exported functions and types
2. **Update examples** if the API changes
3. **Regenerate documentation** with `make docs-generate`
4. **Test documentation** by viewing the HTML output

### Documentation Standards

- Use clear, concise language
- Include code examples where helpful
- Document all parameters and return values
- Explain complex concepts with examples
- Keep documentation up-to-date with code changes

## 🔗 External Resources

- [Go Documentation Guidelines](https://go.dev/doc/effective_go#commentary)
- [Godoc Best Practices](https://go.dev/blog/godoc)
- [Conventional Commits](https://www.conventionalcommits.org/)

## 📧 Support

If you have questions about the documentation or need help:

1. Check the [main README](../README.md)
2. Browse the [API documentation](api/)
3. Open an issue on GitHub
4. Join our community discussions

---

*Documentation auto-generated from Go source code. Last updated: $(date)*

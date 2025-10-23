# Onigirazu Documentation - v1.52.0

**Current Release:** v1.52.0
**Documentation Status:** Complete & Production-Ready ✅

Welcome to the Onigirazu documentation directory! This contains comprehensive documentation for the Onigirazu automation tool.

## 📚 Documentation Structure - v1.52.0

### User Documentation

- **[Quick Start Guide](quick-start.md)** - Get started with Onigirazu in minutes
- **[Playbook Format & Examples](examples/README.md)** - Playbook structure, syntax, and real-world examples
- **[Playbook Types Reference](api/pkg/types.md)** - Technical type definitions for playbooks and tasks
- **[Variables Cheat Sheet](VARIABLES_CHEATSHEET.md)** - Quick reference for common variables ⚡
- **[Variables and Configuration](VARIABLES_AND_CONFIGURATION.md)** - Complete reference for all configuration parameters and variables
- **[Inventory Formats](inventory-formats.md)** - Supported inventory file formats (YAML, TOML, JSON, INI, text)
- **[Inline Inventory Guide](../INLINE_INVENTORY_GUIDE.md)** - Using inline host specifications without inventory files
- **[Modules Reference](modules/README.md)** - Built-in modules documentation
- **[Ad-hoc Commands Guide](ADHOC_GUIDE.md)** - Execute commands without playbooks (5 input formats)

### New in v1.52.0: Complete Configuration & Security Documentation

- **[Configuration Reference](CONFIGURATION_REFERENCE.md)** ⭐ NEW - Complete guide to all 35+ configuration options
- **[Security Policy Guide](SECURITY_POLICY_GUIDE.md)** ⭐ NEW - Complete guide to all 13+ security options
- **[Quick Start Configuration](QUICK_START_CONFIGURATION.md)** ⭐ NEW - 5-minute setup guide (fixes common errors)
- **[Configuration & Security Index](INDEX_CONFIGURATION_SECURITY.md)** ⭐ NEW - Navigation guide by role and problem type

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

- **[Playbook](api/pkg/types.md#playbook)** - YAML playbook structure, see [Playbook Format Guide](examples/README.md)
- **[Task](api/pkg/types.md#task)** - Individual task definition
- **[Host](api/pkg/types.md#host)** - Target host configuration
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
- [Playbook structure parsing](examples/README.md) with validation
- [Inline inventory parsing](../INLINE_INVENTORY_GUIDE.md) for direct host specifications
- Variable interpolation and template processing

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

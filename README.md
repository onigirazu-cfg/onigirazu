# Onigirazu Documentation - v1.52.0

**Current Release:** v1.52.0
**Documentation Status:** Complete & Production-Ready ✅

Welcome to the Onigirazu documentation directory! This contains comprehensive documentation for the Onigirazu automation tool.

## 📚 Documentation Structure - v1.52.0

### User Documentation

- **[Quick Start Guide](QUICK_START_CONFIGURATION.md)** - Get started with Onigirazu in minutes
- **[Playbook Format & Examples](examples/README.md)** - Playbook structure, syntax, and real-world examples
- **[Playbook Types Reference](examples/README.md)** - Technical type definitions for playbooks and tasks
- **[Variables Cheat Sheet](VARIABLES_CHEATSHEET.md)** - Quick reference for common variables ⚡
- **[Configuration Reference](CONFIGURATION_REFERENCE.md)** - Complete reference for all configuration parameters and variables
- **[Inventory Formats](INVENTORY_FORMATS.md)** - Supported inventory file formats (YAML, TOML, JSON, INI, text)
- **[Inventory Formats Guide](INVENTORY_FORMATS.md)** - Using inventory specifications and file formats
- **[Modules Reference](modules/README.md)** - Built-in modules documentation
- **[Ad-hoc Commands Guide](ADHOC_GUIDE.md)** - Execute commands without playbooks (5 input formats)

### New in v1.52.0: Complete Configuration & Security Documentation

- **[Security Policy Guide](SECURITY_POLICY_GUIDE.md)** ⭐ NEW - Complete guide to all 13+ security options
- **[Configuration & Security Index](INDEX_CONFIGURATION_SECURITY.md)** ⭐ NEW - Navigation guide by role and problem type
- **[Troubleshooting Configuration](TROUBLESHOOTING_CONFIG.md)** - Solutions for common configuration issues

### Design Documentation

- **[logo.md](logo.md)** - Logo design guidelines and assets
- **[logo.png](logo.png)** - Main project logo (200px)
- **[logo.svg](logo.svg)** - Vector version of the logo

## 🚀 Quick Start

1. **Start here**: [Quick Start Configuration](QUICK_START_CONFIGURATION.md) - 5-minute setup guide
2. **Set up inventory**: [Inventory Formats](INVENTORY_FORMATS.md) - All supported formats
3. **Learn variables**: [Variables Cheat Sheet](VARIABLES_CHEATSHEET.md) - Common variables reference
4. **Write playbooks**: [Playbook Format & Examples](examples/README.md) - Syntax and examples
5. **Run ad-hoc commands**: [Ad-hoc Commands Guide](ADHOC_GUIDE.md) - Command-line execution

## 📖 Available Documentation

### Core Concepts

- **[Playbook Format](examples/README.md)** - YAML playbook structure and syntax
- **[Task Definition](examples/README.md)** - Individual task structure and options
- **[Inventory & Hosts](INVENTORY_FORMATS.md)** - Target host configuration
- **[Variables & Templates](VARIABLES_CHEATSHEET.md)** - Variable interpolation and templating
- **[Modules](modules/README.md)** - Built-in modules and module development
- **[Handlers & Callbacks](HANDLERS_GUIDE.md)** - Event handling and workflow orchestration

### Advanced Topics

- **[Loops & Iteration](LOOPS_GUIDE.md)** - Loop constructs and iteration patterns
- **[Filters & Plugins](FILTERS_GUIDE.md)** - Available filters and plugin system
- **[Interactive Mode](INTERACTIVE_MODE.md)** - Interactive playbook execution
- **[Tag Filtering](TAG_FILTERING.md)** - Running specific tasks by tags

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
2. See [Troubleshooting Configuration](TROUBLESHOOTING_CONFIG.md) for common issues
3. Open an issue on GitHub
4. Join our community discussions

---

*Documentation auto-generated from Go source code. Last updated: $(date)*

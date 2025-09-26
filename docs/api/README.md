# Onigirazu API Documentation

This directory contains auto-generated API documentation for the Onigirazu project.

## Public Packages

### pkg/types
Core types and interfaces used throughout the project.
- [types.md](pkg/types.md) - Data structures for tasks, playbooks, hosts, and results

### pkg/utils
Utility functions and helpers.
- [utils.md](pkg/utils.md) - Common utility functions

## Internal Packages

### Core Components
- [config.md](internal/config.md) - Configuration management
- [core.md](internal/core.md) - Core engine implementation
- [engine.md](internal/engine.md) - Execution engine
- [parser.md](internal/parser.md) - YAML/Playbook parsing
- [workflow.md](internal/workflow.md) - Workflow orchestration

### Modules
- [modules.md](internal/modules.md) - Built-in modules (command, file, template, etc.)

## Usage

To regenerate this documentation, run:

```bash
./scripts/generate-docs.sh
```

## Viewing Documentation

You can view the documentation in several ways:

1. **Read the markdown files directly** in this directory
2. **Use go doc command** for interactive browsing:
   ```bash
   go doc github.com/onigirazu-cfg/onigirazu/pkg/types
   ```
3. **Start a local documentation server**:
   ```bash
   pkgsite -http=:8080
   ```
   Then open http://localhost:8080 in your browser

## API Stability

- **pkg/** packages are considered public API and follow semantic versioning
- **internal/** packages are internal implementation details and may change without notice

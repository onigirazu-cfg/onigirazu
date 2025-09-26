#!/bin/bash

# Generate API documentation for Onigirazu project

set -e

echo "Generating API documentation..."

# Create docs directory structure
mkdir -p docs/api
mkdir -p docs/api/pkg
mkdir -p docs/api/internal

# Generate documentation for public packages
echo "Generating documentation for pkg/types..."
go doc -all ./pkg/types > docs/api/pkg/types.md

echo "Generating documentation for pkg/utils..."
go doc -all ./pkg/utils > docs/api/pkg/utils.md

# Generate documentation for key internal packages
echo "Generating documentation for internal/config..."
go doc -all ./internal/config > docs/api/internal/config.md

echo "Generating documentation for internal/core..."
go doc -all ./internal/core > docs/api/internal/core.md

echo "Generating documentation for internal/engine..."
go doc -all ./internal/engine > docs/api/internal/engine.md

echo "Generating documentation for internal/modules..."
go doc -all ./internal/modules > docs/api/internal/modules.md

echo "Generating documentation for internal/parser..."
go doc -all ./internal/parser > docs/api/internal/parser.md

echo "Generating documentation for internal/workflow..."
go doc -all ./internal/workflow > docs/api/internal/workflow.md

# Generate main API index
cat > docs/api/README.md << 'EOF'
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
EOF

echo "Documentation generated successfully!"
echo ""
echo "To view the documentation:"
echo "1. Read markdown files in docs/api/"
echo "2. Run 'pkgsite -http=:8080' and open http://localhost:8080"
echo "3. Use 'go doc <package>' for command-line browsing"

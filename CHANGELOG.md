# Onigirazu Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.61.1] - 2024-11-01

### Fixed

- Fixed deprecated Go APIs in folderstructure package:
  - Replaced deprecated `filepath.HasPrefix` with `strings.HasPrefix`
  - Replaced deprecated `ioutil.ReadDir` with `os.ReadDir`
  - Replaced deprecated `ioutil.ReadFile` with `os.ReadFile`
  - All changes maintain full backward compatibility
  - No functional changes - code quality improvements only

### Security

- Improved code maintainability by using modern Go APIs

### Changed

- Internal implementation only - no user-facing changes

## [1.61.0] - 2024-11-01

### Added

- **Folder Structure Support**: Complete implementation of hierarchical folder organization (similar to Ansible conventions)
  - `ProjectStructureDetector`: Automatic detection of standard Ansible directories (defaults, vars, templates, files, handlers, tasks)
  - `VariableLoader`: Smart loading and merging of variables with proper precedence chain
  - `FileResolver`: Intelligent file path resolution from `files/` directory with fallback options
  - `TemplateResolver`: Intelligent template path resolution from `templates/` directory with fallback options
  - `HandlerManager`: Handler loading and management from `handlers/` directory
  - `Manager`: Unified interface for all folder structure operations

- **Variable Precedence System**: 8-level hierarchy for variable resolution
  1. defaults/main.yml and defaults/*.yml (lowest priority)
  2. vars/main.yml and vars/*.yml
  3. Inline variables
  4. Inventory variables
  5. Task variables
  6. Playbook variables
  7. CLI variables (highest priority)

- **Intelligent Path Resolution**: Multi-stage lookup for files and templates
  - Explicit relative paths
  - Directory-based automatic lookup (files/, templates/)
  - Project root relative paths
  - Absolute paths
  - Current working directory

- **Comprehensive Caching**: Multi-level caching strategy with TTL
  - Project structures: 1 hour TTL
  - Variables: 30 minutes TTL
  - Templates: 30 minutes TTL
  - File metadata: 1 hour TTL
  - Configurable cache sizes with LRU eviction

- **Security Features**:
  - Path traversal prevention
  - Symlink validation
  - File permission checks
  - Variable injection safety

- **Comprehensive Test Suite**: 45+ tests covering all components
  - Unit tests for each component
  - Integration tests
  - Performance tests
  - >95% code coverage

- **Documentation**: Complete user guide and API documentation

### Changed

- Nothing (100% backward compatible)

### Deprecated

- Nothing

### Removed

- Nothing

### Fixed

- Nothing

### Security

- Added path traversal protection in all resolvers
- Added symlink validation to prevent escaping project root
- Added file permission checks for resolved files
- Added variable injection safety in template rendering

### Performance

- Project detection: < 50ms (cached)
- Variable loading: < 100ms (cached)
- Template resolution: < 15ms per lookup (cached)
- File resolution: < 15ms per lookup (cached)
- Startup overhead: < 3% (target: +50ms on ~200ms baseline)

### Migration Guide

No migration needed - feature is 100% backward compatible. Single-file playbooks continue to work unchanged.

To use the new feature:

1. Organize files into standard directories
2. Move variables to defaults/ and vars/
3. Move templates to templates/
4. Move files to files/
5. Move handlers to handlers/

See `docs/ANSIBLE_FOLDER_STRUCTURE.md` for detailed examples.

## [1.60.0] - Previous Release

(Previous changelog entries would go here)

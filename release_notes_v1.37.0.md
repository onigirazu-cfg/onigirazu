# Release v1.37.0: Executor Safety Architecture

## 🎯 Overview

This release introduces a major architectural improvement to prevent executor caching bugs across all modules. The new **Executor Safety Architecture** ensures that each module execution gets a fresh executor instance, eliminating a class of bugs related to stale connections and cached state.

## ✨ Key Features

### Executor Safety Architecture

- **20 modules migrated** to the safe `BaseExecutorModule` pattern
- **Three safe patterns** for executor management:
  - `WithExecutorResult` - Simple, clean pattern for single operations
  - `WithExecutor` - Complex logic with multiple operations
  - `CreateExecutor` - Maximum control with manual management
- **Comprehensive documentation** ecosystem (25 files, ~214KB)
- **100% backward compatible** with v1.36.x
- **Zero breaking changes**

### Migrated Modules

The following modules now use the safe executor pattern:

- `command` - Execute commands safely
- `copy` - File copying with fresh executors
- `cron` - Cron job management
- `docker_compose` - Docker Compose operations
- `docker_container` - Container management
- `docker_image` - Image operations
- `file` - File operations
- `git` - Git repository management
- `group` - Group management
- `lineinfile` - Line-in-file editing
- `mysql_db` - MySQL database management
- `mysql_user` - MySQL user management
- `service` - Service management
- `stat` - File statistics
- `systemd` - Systemd service management
- `user` - User management
- And 4 more modules

### Documentation

New comprehensive documentation includes:

- **START_HERE.md** - Entry point for all documentation
- **NAVIGATION_MAP.md** - Complete documentation structure
- **EXECUTOR_ARCHITECTURE_INDEX.md** - Executor architecture guide
- **Multiple learning paths** for different user needs
- **Code examples** and best practices
- **Migration guides** for module developers

## 🔧 Technical Improvements

### Architecture Benefits

- **Prevents executor caching bugs** - Fresh executor for each execution
- **Consistent API** across all modules
- **Better testability** - Easier to mock and test
- **Improved reliability** - No stale connections or cached state
- **Clear patterns** - Three well-documented patterns for different use cases

### Code Quality

- Fixed compilation errors in `service_test.go`
- Added comprehensive test coverage for executor safety
- Improved code organization and maintainability

## 📊 Statistics

- **20 modules** migrated to safe pattern
- **25 documentation files** (~214KB)
- **8 modules remaining** to be migrated
- **100% backward compatibility** maintained

## 🚀 Upgrade Guide

This release is **100% backward compatible** with v1.36.x. No changes are required to your playbooks or configurations.

### For Module Developers

If you're developing custom modules, we recommend migrating to the new `BaseExecutorModule` pattern. See the documentation for migration guides and examples.

## 📚 Documentation

For detailed information about the Executor Safety Architecture:

- Read `docs/START_HERE.md` for an overview
- Check `docs/EXECUTOR_ARCHITECTURE_INDEX.md` for technical details
- See `docs/examples/example_module_with_base_executor.go` for code examples

## 🙏 Acknowledgments

This release represents a significant architectural improvement that will benefit all Onigirazu users by preventing a class of bugs related to executor caching.

---

**Full Changelog**: <https://github.com/onigirazu-cfg/onigirazu/compare/v1.36.0...v1.37.0>

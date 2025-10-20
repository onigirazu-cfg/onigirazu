# Changelog

## [Unreleased]

## [1.45.0] - 2025-01-29

### ✨ New Features

- **Dual-Format Playbook Support** - Transparent support for both playbook formats
  - Structured format: `name`, `vars`, `plays` (explicit structure)
  - Ansible-compatible format: Direct list starting with `- hosts:`
  - Automatic format detection via custom `UnmarshalYAML` method
  - Zero friction switching between formats
  - Backward fully compatible with existing playbooks

### 📚 Documentation

- **PLAYBOOK_FORMATS.md** - Complete guide explaining both formats
  - Side-by-side format comparisons
  - Practical examples for each format
  - Common mistakes and solutions
  - Format selection matrix for different use cases
  - Conversion examples showing how formats relate

- **PLAYBOOK_QUICK_REFERENCE.md** - Quick lookup cheat sheet
  - Format templates for quick copy-paste
  - Ready-to-use examples
  - Common errors with fixes
  - Command reference for validation

- **PLAYBOOK_FORMAT_COMPARISON.md** - Visual comparisons and decision matrix
  - ASCII diagrams of structure
  - Detailed transformation flow
  - Performance metrics
  - Decision tree for format selection
  - Internal processing explanation

### 🔧 Technical Implementation

- Enhanced `Playbook` struct with custom `UnmarshalYAML` method
- Intelligent fallback mechanism:
  1. Try parsing as direct Play list (Ansible format)
  2. Generate playbook name from first play (format: "Play Name (playbook)")
  3. Fall back to structured format if needed
- Zero performance overhead for format detection
- Full integration with all existing commands:
  - `onigirazu validate` - ✅ Works with both formats
  - `onigirazu lint` - ✅ Works with both formats
  - `onigirazu fmt` - ✅ Preserves original format
  - `onigirazu diff` - ✅ Works with both formats
  - `onigirazu apply` - ✅ Executes correctly
  - `onigirazu audit` - ✅ Records execution properly

### 🧪 Testing

- All existing tests continue to pass (240+)
- Dual-format parsing thoroughly tested
- Edge cases handled:
  - Empty plays lists
  - Missing names (auto-generated as "Generated Playbook")
  - Multiple plays in single file
  - Mixed variable types

### 🚀 Quality

- ✅ 100% backward compatible
- ✅ Zero breaking changes
- ✅ No API modifications
- ✅ All 240+ tests passing
- ✅ User-transparent implementation

### 📝 Usage Examples

**Structured Format:**

```yaml
name: My Playbook
plays:
  - hosts: all
    name: Setup
    tasks:
      - name: Task
        shell: echo "Hello"
```

**Ansible-Compatible Format:**

```yaml
- hosts: all
  name: Setup
  tasks:
    - name: Task
      shell: echo "Hello"
```

Both formats execute identically! ✨

---

## [1.41.0] - 2025-02-05

### ✨ New Features

- **Execution Audit Module** - Enterprise-grade audit logging for all playbook executions
  - Complete execution recording with UUID tracking and hierarchical play/task recording
  - Automatic sensitive data filtering (passwords, API keys, tokens, credentials)
  - 5 output report formats (Text, JSON, CSV, HTML, Markdown)
  - Advanced filtering capabilities (playbook, status, host, date range)
  - Statistics aggregation (global & per-host metrics)
  - CLI integration with 6 subcommands: `audit list`, `audit show`, `audit report`, `audit delete`, `audit export`, `audit stats`
  - Persistent storage with configurable retention policies
  - Thread-safe concurrent recording with sync.RWMutex

### 🔧 Architecture

- **Recorder Component** - Manages execution lifecycle and task recording
  - Hierarchical recording: ExecutionRecord → PlayExecution → TaskResult
  - Play context management with proper initialization
  - Result status tracking (success, failed, skipped, unreachable)

- **Storage Component** - JSON-based persistence and retrieval
  - UUID-based directory organization (~/.onigirazu/audit/)
  - Configurable retention policies with automatic cleanup
  - List and filter operations with flexible options

- **Reporter Component** - Multi-format output generation
  - Text reports with ANSI color support
  - JSON, CSV, HTML, and Markdown formats
  - Customizable templates and output styles

- **CLI Integration** - Seamless command-line interface
  - Integrated into main `onigirazu` binary
  - Subcommand-based structure for clarity
  - Flag-based options for filtering and configuration

### 🧪 Testing & Quality

- Complete unit test suite with 14+ tests
- 100% test pass rate (0.606s execution time)
- Comprehensive coverage of all components
- Integration tests with real file I/O operations
- Performance benchmarks for recording operations
- Security testing for sensitive data filtering

### 📚 Documentation

- Comprehensive implementation guide
- API documentation for all components
- CLI usage examples
- Report format specifications
- Data retention policy guidelines

## [1.39.0] - 2025-02-03

### 🐛 Bug Fixes

- **SSH Debug Logging Integration**: Fully integrated `--show-debug` flag with SSH package logger
  - SSH connection pool now respects debug flags (`--verbose`, `--show-debug`, `-V`)
  - Debug SSH logs properly controlled and disabled by default
  - User has full control over SSH debugging output
  - Applied to both `apply` and `run` (ad-hoc) commands

### 🔧 Internal Improvements

- **SSH Pool Logger Initialization**:
  - Added `InitializeGlobalPoolWithLogger()` function for custom logger passing
  - Maintains backward compatibility with existing `InitializeGlobalPool()`
  - Proper logger propagation through SSH connection pipeline
  - All three verbose flags now work together: `verbose OR verboseMode OR showDebug`

### 🧪 Testing & Quality

- Code compilation: ✅ All checks passing
- SSH package tests: ✅ 50+ tests passing
- CLI integration: ✅ Both apply and run commands tested
- Backward compatibility: ✅ Verified - existing code continues to work

## [1.37.0] - 2025-02-02

### 🏗️ Architecture Improvements

- **Executor Safety Architecture**: Complete refactoring to prevent executor caching bugs
  - Introduced `BaseExecutorModule` with three safe patterns for executor management
  - Pattern 1: `WithExecutorResult` - Simple function execution with automatic cleanup
  - Pattern 2: `WithExecutor` - Complex logic with multiple operations
  - Pattern 3: `CreateExecutor` - Manual management for maximum control
  - Migrated 20 out of 28 modules to use safe patterns
  - Added comprehensive documentation and navigation system (25 files, ~214KB)
  - Created lint checker script for CI/CD integration

### 🔧 Module Improvements

- **Migrated Modules** (20 modules now using BaseExecutorModule):
  - `command`, `copy`, `cron`, `docker_compose`, `docker_container`
  - `docker_image`, `file`, `git`, `group`, `lineinfile`
  - `mysql_db`, `mysql_user`, `service`, `stat`, `systemd`, `user`
  - And 4 more modules

### 📚 Documentation

- **Comprehensive Documentation System**:
  - `START_HERE.md` - Main entry point for all users
  - `NAVIGATION_MAP.md` - Visual navigation with 4 learning paths
  - `EXECUTOR_ARCHITECTURE_INDEX.md` - Complete architecture index
  - `MODULE_DEVELOPMENT_GUIDE.md` - Developer guide with examples
  - `ARCHITECTURE_DIAGRAM.md` - Visual diagrams and schemas
  - 4 recommended learning paths (10 min - 2 hours)
  - 4 learning levels (Basic → Master)

### 🛠️ Developer Tools

- **Lint Checker**: `check_executor_caching.sh` script for detecting unsafe executor usage
  - Identifies modules not using BaseExecutorModule
  - Can be integrated into CI/CD pipeline
  - Helps prevent executor caching bugs

### 🧪 Testing

- **Executor Safety Tests**: Comprehensive test suite for BaseExecutorModule
  - Tests for all three safe patterns
  - Example module demonstrating best practices
  - All tests passing ✅

### 📊 Statistics

- 20 modules migrated to safe executor patterns
- 25 documentation files created (~214KB)
- 8 modules remaining to migrate
- 100% backward compatible with v1.36.x
- Zero breaking changes

### 🎯 Benefits

- **Prevents Executor Caching Bugs**: Eliminates race conditions and state pollution
- **Consistent API**: All modules follow the same safe patterns
- **Better Testing**: Easier to test modules with predictable executor lifecycle
- **Improved Reliability**: Reduced risk of connection leaks and resource exhaustion
- **Developer Experience**: Clear patterns and comprehensive documentation

## [1.31.0] - 2025-10-15

### ✨ New Features

- **Drift Detection System**: Comprehensive configuration drift detection and remediation
  - Detect unauthorized changes by comparing current state with snapshots
  - Auto-fix detected drift with `--auto-fix` flag
  - Generate reports in multiple formats (text, JSON, HTML)
  - Severity-based drift classification (critical, high, medium, low)
  - List and view detailed drift reports
  - CLI commands: `onigirazu drift detect`, `onigirazu drift list`, `onigirazu drift info`

- **Rollback Support**: Safe rollback to previous system states
  - Automatic snapshot creation before playbook execution
  - Rollback to any previous snapshot
  - Dry-run mode to preview rollback changes
  - List available snapshots with metadata
  - Cleanup old snapshots with configurable retention
  - CLI commands: `onigirazu rollback`, `onigirazu rollback --list`, `onigirazu rollback --cleanup`

### 🧪 Testing Improvements

- **Module Test Coverage**: Significantly improved test coverage for core modules
  - Group module: Added 54 comprehensive tests
  - Lineinfile module: Added 50 comprehensive tests
  - Template module: Added 26 additional tests
  - User module: Added 18 additional tests
  - Overall module coverage improved by ~30%

- **Drift Detection Tests**: Full test suite for drift detection functionality
  - Configuration tests
  - State comparison tests
  - Severity calculation tests
  - Fix order calculation tests
  - All tests passing ✅

- **Rollback Tests**: Complete test coverage for rollback functionality
  - Snapshot creation and management tests
  - Rollback execution tests (dry-run and actual)
  - Cleanup and retention tests
  - Resource snapshot generation tests
  - All tests passing ✅

### 🔧 Technical Details

- **New Packages**:
  - `internal/drift`: Drift detection, fixing, and reporting
  - `internal/rollback`: Snapshot management and rollback execution

- **New CLI Commands**:
  - `drift detect`: Detect configuration drift
  - `drift list`: List all drift reports
  - `drift info`: Show detailed drift information
  - `rollback`: Rollback to a previous snapshot
  - `rollback --list`: List available snapshots
  - `rollback --cleanup`: Cleanup old snapshots

### 📊 Statistics

- 8 new source files added (drift + rollback packages)
- 4 new CLI commands
- ~200 new tests added
- Module test coverage improved by ~30%
- 100% backward compatible with v1.30.x

### 🎯 Use Cases

**Drift Detection:**

```bash
# Detect drift from latest snapshot
onigirazu drift detect --snapshot <id>

# Auto-fix detected drift
onigirazu drift detect --snapshot <id> --auto-fix

# Generate HTML report
onigirazu drift detect --snapshot <id> --format html --output report.html
```

**Rollback:**

```bash
# List available snapshots
onigirazu rollback --list

# Preview rollback (dry-run)
onigirazu rollback --dry-run --snapshot <id>

# Perform rollback
onigirazu rollback --snapshot <id>

# Cleanup snapshots older than 30 days
onigirazu rollback --cleanup --max-age 30d
```

## [1.30.2] - 2025-02-01

### 🐛 Bug Fixes

- **Repository Structure**: Removed duplicate nested `onigirazu/onigirazu/` directory from repository
  - Cleaned up 329 duplicate files (94,212 lines of code)
  - Fixed incorrect repository structure that was committed to git
  - Repository now has clean, logical structure

- **Workflow ID Generation**: Fixed race condition in workflow orchestrator
  - Added atomic counters to ensure unique IDs even when generated simultaneously
  - Fixed flaky test `TestWorkflowOrchestrator_ListWorkflows`
  - Improved thread-safety using `sync/atomic` package
  - Prevents ID collisions when workflows are created in rapid succession

### 🔧 Maintenance

- **Code Quality**: Improved reliability of workflow execution system
- **Testing**: Fixed intermittent test failures in workflow orchestrator

### 📊 Statistics

- 329 duplicate files removed from repository
- 2 critical bugs fixed
- 100% test pass rate achieved
- 100% backward compatible with v1.30.1

## [1.30.1] - 2025-02-01

### 🔧 Maintenance

- **Dependency Updates**:
  - Updated `golang.org/x/crypto` from 0.42.0 to 0.43.0 (security improvements)
  - Updated `golang.org/x/text` from 0.29.0 to 0.30.0

- **GitHub Actions Updates**:
  - Updated `codecov/codecov-action` from v4 to v5 (changed `file` parameter to `files`)
  - Updated `actions/setup-python` from v4 to v6
  - Updated `peaceiris/actions-gh-pages` from v3 to v4
  - Updated `peter-evans/create-pull-request` from v6 to v7
  - Updated `github/codeql-action` from v3 to v4 (6 instances across workflows)

- **CI/CD Improvements**:
  - Fixed security workflow to use correct onigirazu subdirectory paths
  - Added dependency download for all security scanners
  - Improved workflow reliability and maintainability

### 📊 Statistics

- All Dependabot PRs processed and merged
- 7 GitHub Actions updated to latest versions
- 2 Go dependencies updated with security improvements
- 100% backward compatible with v1.30.0

## [1.27.1] - 2025-01-29

### ✨ Added

- **Short Command-Line Flags**: Added convenient short aliases for all CLI flags using pflag library
  - `-p` for `--playbook` - Path to playbook file
  - `-i` for `--inventory` - Path to inventory file
  - `-c` for `--config` - Path to configuration file
  - `-v` for `--verbose` - Verbose output
  - `-C` for `--check` - Check mode (dry-run)
  - `-d` for `--diff` - Show differences when changing files
  - `-s` for `--state` - State file path
  - `-l` for `--log-level` - Log level
  - `-o` for `--output` - Output format
  - `-w` for `--max-workers` - Maximum number of worker threads
  - `-t` for `--timeout` - Execution timeout
  - `-V` for `--version` - Show version and exit

### 🔧 Changed

- Migrated from standard `flag` package to `github.com/spf13/pflag` for GNU-style flag support
- Updated help message to show both long and short flag usage examples
- Improved CLI user experience with shorter command syntax

### 📝 Examples

```bash
# Before (still works):
onigirazu --playbook site.yml --inventory hosts --verbose

# After (new short syntax):
onigirazu -p site.yml -i hosts -v

# Mixed usage also works:
onigirazu -p site.yml --inventory hosts -v
```

## [1.27.0] - 2025-01-29

### 🐛 Bug Fixes

- **Critical Test Failures Fixed** (16 tests, 59% improvement):
  - Fixed copy module test failures (20+ test cases added)
  - Fixed service module test failures (34+ test cases added)
  - Fixed stat module command execution issues
  - Fixed shell module command execution issues
  - Fixed file module command execution issues
  - Fixed lineinfile module command execution issues

### 📚 Documentation

- **Major Documentation Cleanup**:
  - Removed 120+ development documentation files (~45,000 lines)
  - Moved development documentation to separate repository
  - Replaced `ansible_*` variables with `onigirazu_*` throughout documentation
  - Removed unused `python_interpreter` variables
  - Updated .gitignore to exclude private AI config files

### 🔧 Maintenance

- **Code Quality Improvements**:
  - Added 54+ new test cases across multiple modules
  - Improved test coverage significantly
  - Better error handling in modules
  - Enhanced command execution reliability

### 📊 Statistics

- 150 files changed
- ~45,000 lines removed (mostly documentation)
- 100% backward compatible with v1.26.0

## [1.26.0] - 2025-01-28

### ✨ Added

- **Unified Package Module**: Merged two package modules into one comprehensive solution
  - Extended PackageManager interface from 12 to 18 methods (+50%)
  - Added 6 new methods: Search, ListInstalled, ListUpgradable, Clean, AutoRemove, VerifyIntegrity
  - Full implementation for APT, YUM, and Homebrew package managers
  - Partial implementation for Pacman, Zypper, Chocolatey, and Generic managers

- **Enterprise Package Management Features**:
  - **Snapshot System**: Point-in-time snapshots with rollback capability
    - SHA256 checksum verification
    - Atomic rollback operations
    - Snapshot listing and management
  - **Package Groups**: Install and manage related packages as atomic units
    - Atomic install/remove operations
    - Required vs optional groups
    - Cross-platform group definitions
  - **Health Checks**: Multi-dimensional system health monitoring
    - Package integrity verification
    - Broken dependency detection
    - Orphaned package identification
    - Cache health and disk space monitoring
  - **Audit Logging**: Structured audit trail for compliance
    - All package operations logged
    - Filterable by date, operation, user, success
    - JSON-ready format for export

- **Complete Homebrew Support**: All 18 PackageManager methods now fully implemented
  - Search packages with `brew search`
  - List installed packages with versions
  - List upgradable packages
  - Clean package cache
  - Auto-remove orphaned dependencies
  - Verify system integrity with `brew doctor`

- **Enhanced Module Capabilities**:
  - Template module improvements with better error handling
  - Copy module enhancements for remote execution
  - File module improvements for better permission handling
  - Service module updates for systemd integration
  - Config module enhancements for configuration management

- **SSH Connection Improvements**:
  - Better connection pooling and reuse
  - Improved error handling and retry logic
  - Enhanced timeout management
  - Better support for IPv6 addresses

### 🔧 Changed

- Refactored package module architecture for better maintainability
- Improved execution engine with better task result handling
- Enhanced inventory manager with better host group support
- Updated facts gatherer with more system information
- Improved core engine with better error propagation

### 🐛 Fixed

- Fixed unused variable in package.go causing compilation errors
- Removed obsolete package_enhanced_test.go file
- Fixed IPv6 address handling in SSH connections
- Replaced deprecated strings.Title with golang.org/x/text/cases
- Fixed gosec warnings in plugin config handling
- Fixed CodeQL unhandled writable file close warnings

### 📖 Documentation

- Created 12 comprehensive documentation files (~130 KB)
- Added PACKAGE_MODULE_v1.26.0_COMPLETE.md with full release report
- Added PACKAGE_ENHANCEMENT_FINAL_SUMMARY.md with executive summary
- Added PACKAGE_ARCHITECTURE.md with architecture documentation
- Added PACKAGE_QUICK_REFERENCE.md with API reference
- Updated module documentation with new features
- Added variables and configuration guides

### 📊 Metrics

- Production code added: +643 lines
- Interface methods: 12 → 18 (+50%)
- New data structures: +4 (SystemSnapshot, PackageGroup, HealthCheckResult, AuditEntry)
- Full implementations: 3 managers (APT, YUM, Brew)
- Stub implementations: 4 managers (Pacman, Zypper, Chocolatey, Generic)
- Documentation: 12 files, ~130 KB

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing playbooks continue to work unchanged
- No breaking changes introduced
- Existing package module functionality preserved

## [1.25.0] - 2025-01-28

### 🧪 Testing

- Added comprehensive parser test coverage
- Improved test suite for enhanced parser
- Added inventory parser tests
- Enhanced playbook parser tests

### 📖 Documentation

- Added security session completion summary
- Added final security session report
- Added CodeQL completion summary
- Updated security fixes documentation

### 🔐 Security

- Fixed CodeQL unhandled writable file close warnings
- Fixed gosec warnings in plugin config handling
- Improved security validator

## [1.24.0] - 2025-01-28

### 🧪 Testing

- Added comprehensive test coverage for template engine
- Improved template module tests
- Enhanced formatter tests

### 📖 Documentation

- Added completion report for v1.24.0
- Added final summary documentation
- Updated session summaries

## [1.23.0] - 2025-01-28

### 🧪 Testing

- Logger test coverage improvements
- Enhanced logging tests

### 🐛 Bug Fixes

- Various bug fixes in logging system

## [1.22.0] - 2025-01-28

### ✨ Added

- **Template Caching System**: Improved template performance
  - Template compilation caching
  - Better template reuse
  - Reduced memory footprint

## [1.21.0] - 2025-01-28

### ✨ Added

- **Inventory Plugins Examples**: Cloud provider integration examples
  - AWS EC2 inventory plugin example
  - Azure VM inventory plugin example
  - GCP Compute inventory plugin example

## [1.20.0] - 2025-01-28

### ✨ Added

- **Plugin System Support**: Added plugin system to main application
  - Dynamic plugin loading
  - Plugin configuration management
  - Plugin lifecycle management

## [1.9.0] - 2025-01-16

### ✨ Added

- **Simplified YAML Syntax**: New Ansible-like syntax for task definitions
  - Module name is now used directly as the field name (e.g., `package:`, `service:`, `file:`)
  - Removed redundant `module:` and `type:` wrappers
  - Cleaner, more intuitive syntax with one less nesting level
  - Example: `package: { name: git, state: present }` instead of `module: { type: package, name: git, state: present }`
  - Full backward compatibility maintained - old syntax still works

### 🔧 Changed

- Updated task parser to support both old and new syntax formats
- Modified `MarshalYAML` to output tasks using the new simplified syntax
- Updated all example playbooks (25 files) to use the new syntax
- Updated documentation (README.md, docs/IMPROVED_SYNTAX.md) with new syntax examples

### 📖 Documentation

- Completely revised `docs/IMPROVED_SYNTAX.md` with new syntax examples
- Updated README.md Quick Start guide with simplified syntax
- All 25 example YAML files migrated to new syntax

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing playbooks with old syntax continue to work
- Parser automatically detects and handles both syntax formats
- No breaking changes introduced

## [1.8.0] - 2025-01-15

### ✨ Added

- **New Systemd Module**: Comprehensive systemd management
  - Service management (start, stop, restart, reload, enable, disable, mask)
  - Unit file creation and management
  - Systemd timer management
  - Daemon reload functionality
  - Service status checking
- **New Cron Module**: Complete cron job management
  - Individual cron job management in user crontabs
  - Full crontab file management with backup support
  - System cron management (cron.d, cron.daily, cron.hourly, cron.weekly, cron.monthly)
  - Cron job listing and inspection
  - Support for special time strings (@reboot, @daily, etc.)
- **New Firewall Module**: Unified firewall management with automatic detection
  - Support for UFW (Ubuntu/Debian)
  - Support for firewalld (RHEL/CentOS/Fedora)
  - Support for iptables (fallback)
  - Port-based rules (allow/deny)
  - Service-based rules (UFW and firewalld)
  - Source-based rules (IP addresses and subnets)
  - Firewall enable/disable
  - Rule listing and reload

### 📖 Documentation

- Created comprehensive module documentation: `docs/MODULES_SYSTEMD_CRON_FIREWALL.md`
- Added example playbooks:
  - `examples/10-systemd-management.yml` - Systemd module examples
  - `examples/11-cron-management.yml` - Cron module examples
  - `examples/12-firewall-management.yml` - Firewall module examples

## [1.7.2] - 2025-01-14

### 🐛 Fixed

- Docker image publishing workflow for releases

## [1.7.1] - 2025-10-07

### ✨ Added

- **Multi-Format Inventory Support**: Support for three inventory file formats
  - **YAML format**: Traditional Ansible-style inventory (existing)
  - **TOML format**: Modern structured configuration format with full feature parity
  - **Simple list format**: Plain text files with one host per line (IP, user@host, user@host:port)
- **Automatic Inventory Detection**: Auto-discovery of inventory files in playbook directory
  - Searches for common inventory file names: `inventory.yml`, `inventory.yaml`, `inventory.toml`, `hosts`, etc.
  - Intelligent format detection based on file extension and content analysis
  - Graceful fallback to localhost-only when no inventory found
- **Smart Format Detection**: Automatic format recognition with fallback chain
  - Extension-based detection (`.yml`, `.yaml`, `.toml`, `.txt`)
  - Content-based heuristics for ambiguous cases
  - Priority: YAML → TOML → Simple list
- **Simple List Parser**: Flexible host address parsing
  - Plain IP addresses: `192.168.1.10`
  - IP with custom port: `192.168.1.10:2222`
  - With username: `user@192.168.1.10`
  - Full format: `user@192.168.1.10:2222`
  - Automatic defaults (port 22, user "root")

### 🔧 Changed

- Enhanced `EnhancedParser` with integrated inventory parsing capabilities
- Updated `main.go` to implement auto-detection when `-inventory` flag is not provided
- Added `github.com/pelletier/go-toml/v2` dependency for TOML support

### 🐛 Fixed

- Fixed golangci-lint configuration compatibility (removed unsupported `version` field)
- Fixed Docker multi-architecture build support (simplified to use pre-built binaries)
- Fixed `go.mod` tidiness (moved go-toml to direct dependencies)
- Fixed release process (disabled GPG signing)

### 📖 Documentation

- Created comprehensive `docs/inventory-formats.md` guide
  - Format specifications and examples
  - Migration guide from YAML to TOML
  - Best practices and recommendations
  - Auto-detection behavior explanation
- Added example files:
  - `inventory.example.toml` - TOML format example
  - `inventory.example.txt` - Simple list format example

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing YAML inventory files work unchanged
- Optional feature: Auto-detection only activates when `-inventory` flag is omitted
- Existing `-inventory` flag behavior preserved

## [1.7.0] - 2025-10-07

**Note**: This release was superseded by v1.7.1 due to build configuration issues.

### ✨ Added

- **Multi-Format Inventory Support**: Support for three inventory file formats
  - **YAML format**: Traditional Ansible-style inventory (existing)
  - **TOML format**: Modern structured configuration format with full feature parity
  - **Simple list format**: Plain text files with one host per line (IP, user@host, user@host:port)
- **Automatic Inventory Detection**: Auto-discovery of inventory files in playbook directory
  - Searches for common inventory file names: `inventory.yml`, `inventory.yaml`, `inventory.toml`, `hosts`, etc.
  - Intelligent format detection based on file extension and content analysis
  - Graceful fallback to localhost-only when no inventory found
- **Smart Format Detection**: Automatic format recognition with fallback chain
  - Extension-based detection (`.yml`, `.yaml`, `.toml`, `.txt`)
  - Content-based heuristics for ambiguous cases
  - Priority: YAML → TOML → Simple list
- **Simple List Parser**: Flexible host address parsing
  - Plain IP addresses: `192.168.1.10`
  - IP with custom port: `192.168.1.10:2222`
  - With username: `user@192.168.1.10`
  - Full format: `user@192.168.1.10:2222`
  - Automatic defaults (port 22, user "root")

### 🔧 Changed

- Enhanced `EnhancedParser` with integrated inventory parsing capabilities
- Updated `main.go` to implement auto-detection when `-inventory` flag is not provided
- Added `github.com/pelletier/go-toml/v2` dependency for TOML support

### 📖 Documentation

- Created comprehensive `docs/inventory-formats.md` guide
  - Format specifications and examples
  - Migration guide from YAML to TOML
  - Best practices and recommendations
  - Auto-detection behavior explanation
- Added example files:
  - `inventory.example.toml` - TOML format example
  - `inventory.example.txt` - Simple list format example

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing YAML inventory files work unchanged
- Optional feature: Auto-detection only activates when `-inventory` flag is omitted
- Existing `-inventory` flag behavior preserved

### Example Usage

**Auto-detection:**

```bash
onigirazu playbook.yml  # Automatically finds inventory.yml in same directory
```

**TOML inventory:**

```toml
[hosts.web1]
address = "192.168.1.10"
port = 22
user = "deploy"

[groups.webservers]
hosts = ["web1", "web2"]
```

**Simple list inventory:**

```
192.168.1.10
deploy@192.168.1.11:2222
user@192.168.1.12
```

### Benefits

- **Flexibility**: Choose the format that best fits your workflow
- **Simplicity**: Simple lists for basic inventories, TOML/YAML for complex ones
- **Convenience**: No need to specify inventory path for standard layouts
- **Modern**: TOML support aligns with modern configuration management trends

## [1.6.0] - Previous Release

### ✨ Added

- **Debug Module**: New `debug` module for printing messages and variable values during playbook execution
  - Support for simple text messages
  - Support for multi-line messages
  - Variable interpolation in messages
  - Automatic output formatting
- **Streamlined YAML Syntax**: Eliminated verbose `args:` blocks from task definitions
- **Unquoted String Support**: Simple string values can now be written without quotes
- **Custom YAML Unmarshaler**: Intelligent parsing that distinguishes between task control fields and module arguments
- **Mixed Syntax Support**: Ability to use both old and new syntax in the same playbook
- **Reserved Fields Management**: Comprehensive list of 19 reserved task control fields

### 🔧 Changed

- Task YAML parsing now supports direct module arguments at task level
- Updated all documentation examples to use new compact syntax
- Enhanced README with syntax comparison and migration guide

### 📖 Documentation

- Added comprehensive syntax guide in `docs/IMPROVED_SYNTAX.md`
- Created example playbooks demonstrating new syntax capabilities
- Updated README with before/after syntax examples

### 🧪 Testing

- Added comprehensive test suite for new YAML parsing logic
- Verified backward compatibility with existing playbooks
- Tested complex data types and edge cases

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing playbooks continue to work unchanged
- Old `args:` syntax remains fully supported
- Gradual migration path available

### Example Transformation

**Before:**

```yaml
- name: "Install package"
  module: "package"
  args:
    name: "nginx"
    state: "present"
```

**After:**

```yaml
- name: "Install package"
  module: "package"
  name: nginx
  state: present
```

### Benefits

- **Reduced verbosity**: Less typing required for task definitions
- **Improved readability**: Cleaner, more intuitive YAML structure
- **Better maintainability**: Easier to write and modify playbooks
- **Standards compliance**: Follows YAML best practices for unquoted strings

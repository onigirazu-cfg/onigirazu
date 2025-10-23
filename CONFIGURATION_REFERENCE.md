# Onigirazu Configuration Reference

**Complete guide to all Onigirazu configuration options for end users**

---

## 📋 Table of Contents

1. [Configuration File Locations](#configuration-file-locations)
2. [Configuration File Format](#configuration-file-format)
3. [Execution Settings](#execution-settings)
4. [Logging Configuration](#logging-configuration)
5. [Security Configuration](#security-configuration)
6. [Performance Tuning](#performance-tuning)
7. [Execution Modes](#execution-modes)
8. [User Interface Settings](#user-interface-settings)
9. [SSH/Connection Settings](#sshconnection-settings)
10. [Monitoring & Metrics](#monitoring--metrics)
11. [Vault Integration](#vault-integration)
12. [Syntax Preferences](#syntax-preferences)
13. [Environment Variable Overrides](#environment-variable-overrides)
14. [Complete Configuration Example](#complete-configuration-example)
15. [Configuration Priority & Discovery](#configuration-priority--discovery)

---

## Configuration File Locations

Onigirazu searches for configuration in this priority order (control machine only):

**Priority 1: Explicitly Specified Path**

```bash
onigirazu -c /path/to/onigirazu.yml playbook.yml
```

**Priority 2: Playbook Directory**

```
/path/to/playbook/
├── playbook.yml
└── onigirazu.yml  ← Found here
```

**Priority 3: System Configuration**

```
/etc/onigirazu/onigirazu.yml
```

**Priority 4: Default Configuration**

```
If no file found, built-in defaults are used
```

### Location Recommendations

**Development:**

```
my-project/
├── onigirazu.yml
└── playbooks/
    ├── deploy.yml
    └── maintenance.yml
```

**Production (Team/Server):**

```
/etc/onigirazu/onigirazu.yml
/etc/onigirazu/security-policy.yml
```

**Per-Project (Teams):**

```
/var/lib/onigirazu/projects/myapp/onigirazu.yml
```

---

## Configuration File Format

Configuration files use **YAML format**. All examples use valid YAML syntax.

```yaml
# Configuration format
option_name: value
timeout: 30s          # Duration format: 30s, 5m, 1h
max_count: 10         # Integer
flag: true            # Boolean: true/false
list_items:           # Lists
  - item1
  - item2
```

---

## Execution Settings

Control how Onigirazu executes playbooks.

### max_concurrency

**Type:** Integer
**Default:** 10
**Min/Max:** 1-1000
**Environment:** `ONIGIRAZU_MAX_CONCURRENCY`

Maximum number of parallel task executions across hosts.

```yaml
# Single-threaded execution
max_concurrency: 1

# Default (10 concurrent tasks)
max_concurrency: 10

# Aggressive (100 concurrent tasks)
max_concurrency: 100
```

**Guidelines:**

- Set to `1` for sequential execution
- Set to number of CPU cores for CPU-intensive tasks
- Set to 2x number of cores for I/O-intensive tasks
- Monitor memory and network usage when increasing

---

### default_timeout

**Type:** Duration
**Default:** 30s
**Environment:** `ONIGIRAZU_TIMEOUT`

Default timeout for task execution if not specified per-task.

```yaml
# 30 seconds (default)
default_timeout: 30s

# 5 minutes for longer operations
default_timeout: 5m

# 1 hour for very long operations
default_timeout: 1h
```

**Supported Duration Formats:**

- `5s` - seconds
- `5m` - minutes (=300s)
- `5h` - hours (=18000s)
- `5m30s` - combined format

---

### retry_attempts

**Type:** Integer
**Default:** 3
**Min:** 0
**Environment:** `ONIGIRAZU_RETRY_ATTEMPTS`

Number of times to retry failed tasks (per host).

```yaml
# No retries
retry_attempts: 0

# Default: try 3 times
retry_attempts: 3

# Aggressive: try 10 times
retry_attempts: 10
```

---

### retry_delay

**Type:** Duration
**Default:** 5s
**Environment:** `ONIGIRAZU_RETRY_DELAY`

Wait time between retry attempts.

```yaml
# Retry immediately
retry_delay: 0s

# Default: wait 5 seconds
retry_delay: 5s

# Exponential backoff simulation (use with external script)
retry_delay: 10s
```

---

## Logging Configuration

Control logging output and verbosity.

### log_level

**Type:** String
**Default:** "info"
**Environment:** `ONIGIRAZU_LOG_LEVEL`

Logging verbosity level.

```yaml
# Silent: no logs except errors
log_level: error

# Warnings and errors only
log_level: warn

# Standard: useful information (default)
log_level: info

# Verbose: debug information for troubleshooting
log_level: debug

# Maximum verbosity: all internal operations
log_level: trace
```

**When to use each:**

- `error`: Production environments with log aggregation
- `warn`: Production environments
- `info`: Standard operation (default)
- `debug`: Troubleshooting issues
- `trace`: Deep investigation of internal behavior

---

### log_format

**Type:** String
**Default:** "text"
**Environment:** `ONIGIRAZU_LOG_FORMAT`

Format for log output.

```yaml
# Human-readable text
log_format: text

# JSON format (for log aggregation systems)
log_format: json

# Structured logging with colors
log_format: text  # Same as default with colors
```

**Use JSON for:**

- Log aggregation systems (ELK, Splunk, etc.)
- Parsing logs programmatically
- Integration with monitoring systems

---

## Security Configuration

Security policies are typically configured via separate file. See [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md) for complete details.

### allow_shell_commands

**Type:** Boolean
**Default:** true
**Environment:** `ONIGIRAZU_ALLOW_SHELL`

Allow execution of arbitrary shell commands in `command:` and `shell:` modules.

```yaml
# Allow shell commands (default)
allow_shell_commands: true

# Disable shell commands completely
allow_shell_commands: false
```

**Note:** When `false`, `shell:` and `command:` modules will fail validation.

---

### blocked_commands

**Type:** List of strings
**Default:** ["rm -rf", "format", "mkfs", "dd if=", ":(){ :|:& };:"]
**Environment:** Not supported (use config file)

List of command patterns to block. These patterns are checked in all shell/command operations.

```yaml
blocked_commands:
  - "rm -rf"           # Dangerous recursive delete
  - "format"           # Disk formatting
  - "mkfs"             # Filesystem creation
  - "dd if="           # Direct disk writing
  - ":(){ :|:& };:"    # Fork bomb
  - "shutdown"         # System shutdown
  - "reboot"           # System reboot
  - "halt"             # System halt
  - "poweroff"         # Power off
  - ".* | sh$"         # Pipe to shell (regex)
  - ".* && rm"         # Command chaining with rm
```

---

## Performance Tuning

Optimize Onigirazu for your workload.

### enable_caching

**Type:** Boolean
**Default:** true
**Environment:** `ONIGIRAZU_ENABLE_CACHE`

Enable caching of task results.

```yaml
# Enable caching (default)
enable_caching: true

# Disable caching
enable_caching: false
```

**Benefits of caching:**

- Faster re-runs of same playbooks
- Reduced network traffic
- Better performance for idempotent operations

---

### cache_ttl

**Type:** Duration
**Default:** 5m
**Environment:** `ONIGIRAZU_CACHE_TTL`

Time-to-live for cached items.

```yaml
# Cache for 5 minutes (default)
cache_ttl: 5m

# Cache for 1 hour
cache_ttl: 1h

# Cache for 30 seconds (shorter for frequently changing data)
cache_ttl: 30s

# Disable cache by setting to 0
cache_ttl: 0s
```

---

### enable_checksum

**Type:** Boolean
**Default:** true
**Environment:** `ONIGIRAZU_ENABLE_CHECKSUM`

Enable checksum verification for file operations.

```yaml
# Enable checksum verification (default)
enable_checksum: true

# Disable checksum verification
enable_checksum: false
```

**Benefits:**

- Detect file modifications
- Verify file integrity
- Prevent accidental overwrites

---

### enable_parallel

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_ENABLE_PARALLEL`

Enable parallel execution of tasks within plays.

```yaml
# Sequential execution (default)
enable_parallel: false

# Enable parallel execution
enable_parallel: true
```

**Note:** Also configure `parallel_strategy` when enabled.

---

### parallel_strategy

**Type:** String
**Default:** "linear"
**Valid Values:** "linear", "free"
**Environment:** `ONIGIRAZU_PARALLEL_STRATEGY`

Strategy for parallel execution.

```yaml
# Linear: wait for all hosts to finish each task
parallel_strategy: linear

# Free: hosts proceed independently
parallel_strategy: free
```

**When to use:**

- `linear`: Coordinated operations, ensure all hosts synchronized
- `free`: Independent operations, faster overall completion

---

## Execution Modes

Control execution behavior for testing and development.

### dry_run

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_DRY_RUN`

Simulate execution without making changes.

```yaml
dry_run: true
```

In dry-run mode:

- Tasks are parsed and validated
- Network connections are established
- **No actual changes are made**
- Useful for testing playbook correctness

---

### check_mode

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_CHECK_MODE`

Syntax checking mode - validate playbook without execution.

```yaml
check_mode: true
```

In check mode:

- Playbook syntax is validated
- **No execution occurs**
- Faster than dry-run for syntax validation

---

### verbose

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_VERBOSE`

Enable verbose output showing all details.

```yaml
verbose: true
```

Effect: Equivalent to setting `log_level: debug`

---

### show_diff

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_SHOW_DIFF`

Show differences when files are modified.

```yaml
show_diff: true
```

Shows side-by-side diffs when:

- File content changes
- Configuration is modified
- Useful for auditing changes

---

## User Interface Settings

Control how Onigirazu displays output.

### color_output

**Type:** Boolean
**Default:** true
**Environment:** `ONIGIRAZU_COLOR_OUTPUT`

Use colored output in console.

```yaml
# Colored output (default)
color_output: true

# Plain text (for CI/CD logs)
color_output: false
```

**Disable colors when:**

- Output goes to CI/CD systems
- Logs are aggregated
- Console doesn't support colors

---

### progress_bar

**Type:** Boolean
**Default:** true
**Environment:** `ONIGIRAZU_PROGRESS_BAR`

Show progress bar during execution.

```yaml
# Show progress bar (default)
progress_bar: true

# No progress bar (for log files)
progress_bar: false
```

---

### interactive_mode

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_INTERACTIVE`

Enable interactive mode with prompts.

```yaml
# Non-interactive (default)
interactive_mode: false

# Interactive mode
interactive_mode: true
```

In interactive mode:

- Prompts before dangerous operations
- Allows pausing execution
- Useful for manual interventions

---

### output_format

**Type:** String
**Default:** "text"
**Valid Values:** "text", "json", "yaml"
**Environment:** `ONIGIRAZU_OUTPUT_FORMAT`

Format for structured output.

```yaml
# Human-readable text (default)
output_format: text

# JSON format (for parsing)
output_format: json

# YAML format (for playbook generation)
output_format: yaml
```

---

## SSH/Connection Settings

Configure SSH behavior and connection handling.

### ssh_timeout

**Type:** Duration
**Default:** 30s
**Environment:** `ONIGIRAZU_SSH_TIMEOUT`

SSH connection timeout per host.

```yaml
# Default: 30 seconds
ssh_timeout: 30s

# Slow networks: 60 seconds
ssh_timeout: 60s

# Fast networks: 10 seconds
ssh_timeout: 10s
```

---

### ssh_keepalive

**Type:** Duration
**Default:** 60s
**Environment:** `ONIGIRAZU_SSH_KEEPALIVE`

SSH keepalive interval to prevent connection timeout.

```yaml
# Default: 60 seconds
ssh_keepalive: 60s

# More frequent for unstable networks
ssh_keepalive: 30s

# Disable keepalive (0)
ssh_keepalive: 0s
```

---

### ssh_max_sessions

**Type:** Integer
**Default:** 10
**Environment:** `ONIGIRAZU_SSH_MAX_SESSIONS`

Maximum concurrent SSH sessions per host.

```yaml
# Default: 10
ssh_max_sessions: 10

# Conservative (slow networks)
ssh_max_sessions: 3

# Aggressive (fast networks)
ssh_max_sessions: 50
```

---

### connection_reuse

**Type:** Boolean
**Default:** true
**Environment:** `ONIGIRAZU_CONNECTION_REUSE`

Reuse SSH connections across tasks.

```yaml
# Reuse connections (default)
connection_reuse: true

# New connection per task
connection_reuse: false
```

**Benefits of reuse:**

- Faster execution
- Lower overhead
- Fewer server resources

---

### ssh_strict_host_key

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_SSH_STRICT_HOST_KEY`

Require strict SSH host key verification.

```yaml
# Permissive (default, suitable for development)
ssh_strict_host_key: false

# Strict (required for production)
ssh_strict_host_key: true
```

**Production Recommendation:** Set to `true`

---

### ssh_known_hosts_file

**Type:** String
**Default:** "" (uses system default ~/.ssh/known_hosts)
**Environment:** `ONIGIRAZU_SSH_KNOWN_HOSTS_FILE`

Path to SSH known_hosts file.

```yaml
# Use custom known_hosts file
ssh_known_hosts_file: /etc/ssh/ssh_known_hosts

# Use default system file (empty string)
ssh_known_hosts_file: ""
```

---

### default_insecure_ignore_host_key

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY`

Ignore SSH host key verification (INSECURE - development only).

```yaml
# Verify host keys (default)
default_insecure_ignore_host_key: false

# Ignore host key verification (DEVELOPMENT ONLY)
default_insecure_ignore_host_key: true
```

⚠️ **WARNING:** Never use in production! Vulnerable to man-in-the-middle attacks.

---

## Monitoring & Metrics

Enable observability and monitoring.

### enable_metrics

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_ENABLE_METRICS`

Enable metrics collection and export.

```yaml
# Metrics disabled (default)
enable_metrics: false

# Enable metrics
enable_metrics: true
```

When enabled, metrics are exported to `metrics_port` on `metrics_path`.

---

### metrics_port

**Type:** Integer
**Default:** 9090
**Environment:** `ONIGIRAZU_METRICS_PORT`

Port for metrics HTTP server.

```yaml
# Default port
metrics_port: 9090

# Alternative port
metrics_port: 8090
```

**Metrics are exposed at:** `http://localhost:9090/metrics`

---

### metrics_path

**Type:** String
**Default:** "/metrics"
**Environment:** `ONIGIRAZU_METRICS_PATH`

HTTP path for metrics endpoint.

```yaml
# Default path
metrics_path: /metrics

# Custom path
metrics_path: /onigirazu/metrics
```

---

### enable_profiling

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_ENABLE_PROFILING`

Enable CPU and memory profiling.

```yaml
# Profiling disabled (default)
enable_profiling: false

# Enable profiling for performance analysis
enable_profiling: true
```

**Profiling data available at:**

- CPU profile: `http://localhost:6060/debug/pprof/profile`
- Memory: `http://localhost:6060/debug/pprof/heap`

---

## Vault Integration

Integrate with HashiCorp Vault for secrets management.

### vault_enabled

**Type:** Boolean
**Default:** false
**Environment:** `ONIGIRAZU_VAULT_ENABLED`

Enable Vault integration.

```yaml
# Vault disabled (default)
vault_enabled: false

# Enable Vault
vault_enabled: true
```

---

### vault_address

**Type:** String
**Default:** "" (empty)
**Environment:** `ONIGIRAZU_VAULT_ADDRESS`

Vault server address.

```yaml
# Vault server URL
vault_address: https://vault.company.com:8200

# Local Vault (development)
vault_address: http://localhost:8200
```

---

### vault_token

**Type:** String
**Default:** "" (empty)
**Environment:** `ONIGIRAZU_VAULT_TOKEN`

Vault authentication token.

```yaml
# Token from environment (not recommended in config file)
vault_token: "hvs.CAESILs..."

# Better: use environment variable
# export ONIGIRAZU_VAULT_TOKEN=hvs.CAESILs...
```

**Security Note:** Store tokens in environment variables, not in config files.

---

## Syntax Preferences

Configure YAML syntax preferences for modules.

### preferred_module_syntax

**Type:** String
**Default:** "nested"
**Valid Values:** "flat", "nested"

Preferred syntax style for module arguments.

```yaml
# Nested style (default)
preferred_module_syntax: nested

# Example nested:
# copy:
#   src: file.txt
#   dest: /tmp/file.txt

# Flat style
preferred_module_syntax: flat

# Example flat:
# copy: src=file.txt dest=/tmp/file.txt
```

---

### enforce_module_syntax

**Type:** Boolean
**Default:** false

Enforce the preferred syntax (reject other styles).

```yaml
# Allow both styles (default)
enforce_module_syntax: false

# Only allow preferred_module_syntax
enforce_module_syntax: true
```

---

## Environment Variable Overrides

All configuration options can be overridden via environment variables.

### Environment Variable Format

```
ONIGIRAZU_<OPTION_NAME_UPPERCASE>
```

### Example Overrides

```bash
# Set max concurrency
export ONIGIRAZU_MAX_CONCURRENCY=20

# Set timeout
export ONIGIRAZU_TIMEOUT=2m

# Enable dry-run
export ONIGIRAZU_DRY_RUN=true

# Set log level
export ONIGIRAZU_LOG_LEVEL=debug

# Run with environment overrides
onigirazu playbook.yml
```

### Priority

Environment variables override config file settings, but playbook-level settings override everything.

```
Playbook task args > Environment variables > Config file > Defaults
```

---

## Complete Configuration Example

```yaml
# onigirazu.yml - Complete configuration example

# === EXECUTION SETTINGS ===
max_concurrency: 10           # Run 10 tasks in parallel
default_timeout: 30s          # Tasks timeout after 30 seconds
retry_attempts: 3             # Retry failed tasks 3 times
retry_delay: 5s               # Wait 5 seconds between retries

# === LOGGING ===
log_level: info               # info, warn, debug, trace
log_format: text              # text or json

# === SECURITY ===
allow_shell_commands: true
blocked_commands:
  - "rm -rf"
  - "format"
  - "mkfs"
  - "dd if="
  - ":(){ :|:& };:"

# === PERFORMANCE ===
enable_caching: true
cache_ttl: 5m
enable_checksum: true
enable_parallel: false
parallel_strategy: linear

# === EXECUTION MODES ===
dry_run: false                # Simulate without changes
check_mode: false             # Just syntax check
verbose: false                # Detailed output
show_diff: true               # Show file changes

# === USER INTERFACE ===
color_output: true            # Colored output
progress_bar: true            # Show progress
interactive_mode: false       # No prompts
output_format: text           # text, json, yaml

# === SSH/CONNECTION ===
ssh_timeout: 30s
ssh_keepalive: 60s
ssh_max_sessions: 10
connection_reuse: true
ssh_strict_host_key: false
ssh_known_hosts_file: ""
default_insecure_ignore_host_key: false

# === MONITORING ===
enable_metrics: false
metrics_port: 9090
metrics_path: /metrics
enable_profiling: false

# === VAULT ===
vault_enabled: false
vault_address: ""
vault_token: ""

# === SYNTAX ===
preferred_module_syntax: nested
enforce_module_syntax: false
```

---

## Configuration Priority & Discovery

### How Configuration is Loaded

1. **Built-in defaults** are applied first
2. **Configuration file** is discovered and loaded (if found)
3. **Environment variables** override config file settings
4. **Command-line flags** override environment variables
5. **Playbook task args** override everything

### Discovery Process

Onigirazu searches for `onigirazu.yml` in this order:

```
1. Explicitly specified path (-c flag)
   onigirazu -c /etc/onigirazu/onigirazu.yml playbook.yml

2. Playbook directory
   ./onigirazu.yml

3. System configuration directory
   /etc/onigirazu/onigirazu.yml

4. No file found → use defaults
```

### Partial Configuration

You don't need to specify all options. Unspecified options use defaults:

```yaml
# Minimal config - only override what you need
max_concurrency: 20
log_level: debug
```

---

## Testing Your Configuration

### Validate Configuration

```bash
# Check if config is valid (syntax check mode)
onigirazu -c onigirazu.yml --check playbook.yml

# Dry-run to verify configuration works
onigirazu -c onigirazu.yml --dry-run playbook.yml
```

### View Effective Configuration

```bash
# Show config as JSON
onigirazu --show-config

# Show config with environment variable values
export ONIGIRAZU_LOG_LEVEL=debug
onigirazu --show-config
```

---

## Common Configuration Scenarios

### Development Environment

```yaml
# Relaxed security, fast iteration
max_concurrency: 5
log_level: debug
ssh_strict_host_key: false
default_insecure_ignore_host_key: true
enable_caching: false
progress_bar: true
color_output: true
```

### Production Deployment

```yaml
# Secure, reliable, monitored
max_concurrency: 20
default_timeout: 5m
log_level: warn
log_format: json
ssh_strict_host_key: true
enable_metrics: true
retry_attempts: 5
retry_delay: 10s
enable_caching: true
```

### CI/CD Pipeline

```yaml
# Headless, monitored, cached
max_concurrency: 50
log_level: info
log_format: json
color_output: false
progress_bar: false
output_format: json
enable_caching: true
dry_run: false
check_mode: false
```

---

## See Also

- [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md) - Security policies and allowed directories
- [LOOPS_GUIDE.md](LOOPS_GUIDE.md) - Loop configuration options
- [CI-CD.md](ci-cd.md) - CI/CD pipeline configuration

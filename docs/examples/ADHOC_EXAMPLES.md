# Ad-hoc Commands - Quick Examples

## Quick Start

```bash
# Test connectivity
onigirazu run all -m ping -i inventory.yml

# Execute simple command
onigirazu run localhost -m command 'command="uptime"' -i inventory.yml

# Debug message
onigirazu run all -m debug 'msg="Hello World"' -i inventory.yml
```

## Real-World Examples

### System Administration

```bash
# Check disk space on all servers
onigirazu run all -m shell 'cmd="df -h"' -i inventory.yml

# Check memory usage
onigirazu run all -m shell 'cmd="free -h"' -i inventory.yml

# Get system information
onigirazu run all -m command 'command="uname -a"' -i inventory.yml

# Check uptime
onigirazu run all -m command 'command="uptime"' -i inventory.yml

# List running processes
onigirazu run all -m shell 'cmd="ps aux | head -20"' -i inventory.yml
```

### Package Management

```bash
# Install package
onigirazu run webservers -m package name=nginx state=present -i inventory.yml

# Update package
onigirazu run all -m package name=vim state=latest -i inventory.yml

# Remove package
onigirazu run webservers -m package name=apache2 state=absent -i inventory.yml

# Natural language
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run all "remove vim package" -i inventory.yml
```

### Service Management

```bash
# Start service
onigirazu run webservers -m service name=nginx state=started -i inventory.yml

# Stop service
onigirazu run webservers -m service name=nginx state=stopped -i inventory.yml

# Restart service
onigirazu run webservers -m service name=nginx state=restarted -i inventory.yml

# Check service status
onigirazu run webservers -m shell 'cmd="systemctl status nginx"' -i inventory.yml

# Natural language
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run webservers "restart nginx service" -i inventory.yml
```

### File Operations

```bash
# Create file
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml

# Delete file
onigirazu run all -m file path=/tmp/test.txt state=absent -i inventory.yml

# Create directory
onigirazu run all -m file path=/tmp/mydir state=directory mode=0755 -i inventory.yml

# Check if file exists
onigirazu run all -m shell 'cmd="ls -la /tmp/test.txt"' -i inventory.yml

# Natural language
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "delete file /tmp/test.txt" -i inventory.yml
```

### User Management

```bash
# Create user
onigirazu run all -m user name=testuser state=present -i inventory.yml

# Delete user
onigirazu run all -m user name=testuser state=absent -i inventory.yml

# List users
onigirazu run all -m shell 'cmd="cat /etc/passwd | tail -5"' -i inventory.yml
```

### Network Operations

```bash
# Check network connectivity
onigirazu run all -m shell 'cmd="ping -c 3 google.com"' -i inventory.yml

# Check open ports
onigirazu run all -m shell 'cmd="netstat -tuln"' -i inventory.yml

# Check DNS resolution
onigirazu run all -m shell 'cmd="nslookup google.com"' -i inventory.yml

# Download file
onigirazu run all -m get_url url=https://example.com/file.txt dest=/tmp/file.txt -i inventory.yml
```

### Monitoring & Diagnostics

```bash
# CPU usage
onigirazu run all -m shell 'cmd="top -bn1 | head -20"' -i inventory.yml

# Load average
onigirazu run all -m shell 'cmd="cat /proc/loadavg"' -i inventory.yml

# Check logs
onigirazu run all -m shell 'cmd="tail -20 /var/log/syslog"' -i inventory.yml

# Disk I/O
onigirazu run all -m shell 'cmd="iostat"' -i inventory.yml
```

### Security Operations

```bash
# Check SSH configuration
onigirazu run all -m shell 'cmd="cat /etc/ssh/sshd_config | grep PermitRootLogin"' -i inventory.yml

# List sudo users
onigirazu run all -m shell 'cmd="cat /etc/sudoers"' -i inventory.yml

# Check firewall status
onigirazu run all -m shell 'cmd="ufw status"' -i inventory.yml

# List open ports
onigirazu run all -m shell 'cmd="ss -tuln"' -i inventory.yml
```

## Advanced Usage

### Parallel Execution

```bash
# Execute on 10 hosts simultaneously
onigirazu run all -m command 'command="uptime"' --parallel 10 -i inventory.yml

# Sequential execution (one at a time)
onigirazu run all -m command 'command="uptime"' --parallel 1 -i inventory.yml

# Default (5 hosts in parallel)
onigirazu run all -m command 'command="uptime"' -i inventory.yml
```

### Output Formats

```bash
# JSON output (for scripts)
onigirazu run all -m ping --output json -i inventory.yml

# YAML output
onigirazu run all -m ping --output yaml -i inventory.yml

# Text output (default, human-readable)
onigirazu run all -m ping --output text -i inventory.yml
```

### Verbose Mode

```bash
# Show detailed execution information
onigirazu run all -m command 'command="uptime"' -V -i inventory.yml

# Combine with global verbose flag
onigirazu run all -m command 'command="uptime"' -v -V -i inventory.yml
```

### Check Mode (Dry-run)

```bash
# Test without making changes
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# See what would be deleted
onigirazu run all -m file path=/tmp/test.txt state=absent --check -i inventory.yml
```

### Timeout Control

```bash
# Set 60 second timeout per host
onigirazu run all -m command 'command="sleep 30 && uptime"' --timeout 60s -i inventory.yml

# Short timeout for quick checks
onigirazu run all -m ping --timeout 5s -i inventory.yml
```

### Target Specific Hosts

```bash
# All hosts
onigirazu run all -m ping -i inventory.yml

# Specific group
onigirazu run webservers -m ping -i inventory.yml

# Specific host
onigirazu run server1 -m ping -i inventory.yml

# Multiple groups (comma-separated)
onigirazu run webservers,databases -m ping -i inventory.yml
```

## Different Input Formats

### Ansible-like (Recommended)

```bash
onigirazu run all -m command 'command="uptime"' -i inventory.yml
onigirazu run all -m package name=nginx state=present -i inventory.yml
```

### Natural Language

```bash
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
```

### Module:Args Syntax

```bash
onigirazu run all "command:command=uptime" -i inventory.yml
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
```

### JSON Format

```bash
onigirazu run all '{"module":"command","args":{"command":"uptime"}}' -i inventory.yml
onigirazu run all '{"module":"package","args":{"name":"nginx","state":"present"}}' -i inventory.yml
```

### YAML Format

```bash
onigirazu run all 'module: command
args:
  command: uptime' -i inventory.yml
```

## Common Patterns

### Health Check

```bash
# Quick health check of all servers
onigirazu run all -m ping -i inventory.yml
onigirazu run all -m command 'command="uptime"' -i inventory.yml
onigirazu run all -m shell 'cmd="df -h | grep -v tmpfs"' -i inventory.yml
```

### Deploy Configuration

```bash
# Copy configuration file
onigirazu run webservers -m copy src=nginx.conf dest=/etc/nginx/nginx.conf -i inventory.yml

# Restart service
onigirazu run webservers -m service name=nginx state=restarted -i inventory.yml

# Verify
onigirazu run webservers -m shell 'cmd="nginx -t"' -i inventory.yml
```

### Emergency Response

```bash
# Stop service immediately on all servers
onigirazu run all -m service name=nginx state=stopped --parallel 20 -i inventory.yml

# Kill process
onigirazu run all -m shell 'cmd="pkill -9 nginx"' -i inventory.yml

# Check status
onigirazu run all -m shell 'cmd="ps aux | grep nginx"' -i inventory.yml
```

### Batch Operations

```bash
# Update all packages
onigirazu run all -m shell 'cmd="apt update && apt upgrade -y"' --parallel 5 -i inventory.yml

# Restart all services
onigirazu run webservers -m service name=nginx state=restarted -i inventory.yml
onigirazu run databases -m service name=postgresql state=restarted -i inventory.yml

# Clean up temporary files
onigirazu run all -m shell 'cmd="rm -rf /tmp/*"' -i inventory.yml
```

## Tips & Tricks

### 1. Use Check Mode First

```bash
# Always test with --check first
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Then execute for real
onigirazu run all -m package name=nginx state=present -i inventory.yml
```

### 2. Use JSON Output for Scripts

```bash
# Parse results in scripts
result=$(onigirazu run all -m ping --output json -i inventory.yml)
echo "$result" | jq '.success'
```

### 3. Combine with Shell Tools

```bash
# Filter results
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml | grep "load average"

# Save results
onigirazu run all -m command 'command="uptime"' -i inventory.yml > results.txt
```

### 4. Use Verbose Mode for Debugging

```bash
# Debug connection issues
onigirazu run problematic_host -m ping -v -V -i inventory.yml
```

### 5. Adjust Parallelism Based on Task

```bash
# CPU-intensive tasks: lower parallelism
onigirazu run all -m shell 'cmd="heavy_computation.sh"' --parallel 2 -i inventory.yml

# Network tasks: higher parallelism
onigirazu run all -m ping --parallel 20 -i inventory.yml
```

## Troubleshooting

### Connection Issues

```bash
# Test basic connectivity
onigirazu run problematic_host -m ping -v -i inventory.yml

# Check SSH configuration
onigirazu run problematic_host -m shell 'cmd="echo Connected"' -v -i inventory.yml
```

### Module Errors

```bash
# Use debug module to check variables
onigirazu run all -m debug 'msg="Testing"' -i inventory.yml

# Use verbose mode
onigirazu run all -m command 'command="uptime"' -V -i inventory.yml
```

### Timeout Issues

```bash
# Increase timeout for slow operations
onigirazu run all -m shell 'cmd="slow_script.sh"' --timeout 300s -i inventory.yml
```

### Permission Issues

```bash
# Check user permissions
onigirazu run all -m shell 'cmd="whoami"' -i inventory.yml

# Check sudo access
onigirazu run all -m shell 'cmd="sudo -l"' -i inventory.yml
```

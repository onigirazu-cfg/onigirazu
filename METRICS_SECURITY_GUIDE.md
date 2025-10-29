# Metrics & Monitoring Security Guide

**Implementation Date:** 2025
**Status:** ✅ Fully Functional with Security Controls

This guide covers Onigirazu's metrics collection and monitoring capabilities with emphasis on security best practices.

## Table of Contents

1. [Overview](#overview)
2. [Security Architecture](#security-architecture)
3. [Configuration](#configuration)
4. [Usage Examples](#usage-examples)
5. [API Endpoints](#api-endpoints)
6. [Security Best Practices](#security-best-practices)
7. [Troubleshooting](#troubleshooting)

---

## Overview

The metrics server provides real-time insights into Onigirazu playbook execution, including:

- **Task execution metrics** - Success/failure rates, timing, module usage
- **Performance metrics** - Concurrent task count, execution times, cache hit rates
- **Host metrics** - Connection status, per-host execution times
- **Error tracking** - Error rates by type and module
- **Resource usage** - Memory, CPU, network, disk I/O

### Export Formats

- **Prometheus format** (`/metrics`) - Scraped by Prometheus monitoring systems
- **JSON format** (`/summary`) - Machine-readable summary
- **Health check** (`/health`) - Simple UP/DOWN status

---

## Security Architecture

### Design Principles

✅ **Secure by Default**

- Binds to localhost only (`127.0.0.1`) by default
- Requires explicit configuration to expose remotely
- Supports optional authentication and IP whitelisting

✅ **Defense in Depth**

- Multiple layers of security controls
- Graceful degradation if one layer fails
- Clear security status logging

✅ **Minimal Information Disclosure**

- Proper HTTP status codes (401, 403)
- No sensitive data leakage in error messages
- Rate limiting via HTTP server timeouts

### Security Layers

1. **Network Isolation** (Primary)
   - Default: Listen on 127.0.0.1 only
   - Optional: Configure to bind to specific IP or 0.0.0.0

2. **Authentication** (Optional)
   - Bearer token validation
   - Checked before metrics are served
   - Returns 401 Unauthorized if invalid

3. **Authorization** (Optional)
   - IP whitelist enforcement
   - Supports multiple IPs and CIDR ranges
   - Returns 403 Forbidden if not whitelisted

4. **Transport Security**
   - HTTP timeouts: 15s (read/write), 60s (idle)
   - Prevents slowloris attacks
   - Proper error handling

---

## Configuration

### Enable Metrics

```yaml
# onigirazu.yml
enable_metrics: true
metrics_port: 9090
metrics_path: /metrics
```

### Security Configuration

#### 1. Localhost Only (Default - Most Secure)

```yaml
enable_metrics: true
metrics_port: 9090
metrics_listen_address: 127.0.0.1  # Default
```

**Access from:**

- Same machine only
- Local docker containers
- SSH port forwarding

```bash
# Forward metrics through SSH tunnel
ssh -L 9090:localhost:9090 user@host
curl http://localhost:9090/metrics
```

#### 2. With Bearer Token Authentication

```yaml
enable_metrics: true
metrics_port: 9090
metrics_listen_address: 127.0.0.1
metrics_auth_token: your-secret-token-12345
```

**Access with token:**

```bash
curl -H "Authorization: Bearer your-secret-token-12345" \
  http://localhost:9090/metrics
```

#### 3. IP Whitelist (For Network-Accessible Instances)

```yaml
enable_metrics: true
metrics_port: 9090
metrics_listen_address: 0.0.0.0  # Listen on all interfaces
metrics_ip_whitelist:
  - 192.168.1.0/24
  - 10.0.0.5
  - 10.0.0.6
```

**Only these IPs can access metrics:**

- 192.168.1.0 - 192.168.1.255
- 10.0.0.5
- 10.0.0.6

#### 4. Combined Security (Recommended for Production)

```yaml
enable_metrics: true
metrics_port: 9090
metrics_listen_address: 10.0.0.0    # Internal network only
metrics_auth_token: production-secret-token-xyz
metrics_ip_whitelist:
  - 10.0.0.1   # Prometheus server
  - 10.0.0.2   # Monitoring dashboard
  - 10.0.1.0/24 # Monitoring subnet
metrics_path: /internal/metrics     # Non-standard path adds obfuscation
```

### Environment Variables

Set via environment variables (override config file):

```bash
# Enable metrics
export ONIGIRAZU_ENABLE_METRICS=true

# Network binding (IP address only, port in config)
export ONIGIRAZU_METRICS_LISTEN_ADDRESS=0.0.0.0

# Authentication token
export ONIGIRAZU_METRICS_AUTH_TOKEN="secret-token"

# IP whitelist (comma-separated)
export ONIGIRAZU_METRICS_IP_WHITELIST="192.168.1.100,192.168.1.101"

# Run playbook
onigirazu apply playbook.yml
```

---

## Usage Examples

### Example 1: Local Development

```yaml
# development.yml
enable_metrics: true
metrics_port: 9090
metrics_listen_address: 127.0.0.1
metrics_auth_token: dev-token
```

**Access metrics:**

```bash
# Without authentication (fails)
curl http://localhost:9090/metrics
# → 401 Unauthorized

# With authentication (succeeds)
curl -H "Authorization: Bearer dev-token" \
  http://localhost:9090/metrics

# Health check (no auth required - can verify server is running)
curl -H "Authorization: Bearer dev-token" \
  http://localhost:9090/health
# → OK
```

### Example 2: Container Deployment

**docker-compose.yml:**

```yaml
services:
  onigirazu:
    image: onigirazu:latest
    environment:
      ONIGIRAZU_ENABLE_METRICS: "true"
      ONIGIRAZU_METRICS_LISTEN_ADDRESS: "0.0.0.0"
      ONIGIRAZU_METRICS_AUTH_TOKEN: "container-secret"
    ports:
      - "9090:9090"
    volumes:
      - ./playbooks:/playbooks
      - ./inventory:/inventory
    command: apply /playbooks/setup.yml
```

**Prometheus scrape config:**

```yaml
scrape_configs:
  - job_name: 'onigirazu'
    static_configs:
      - targets: ['localhost:9090']
    bearer_token: 'container-secret'
    scrape_interval: 30s
```

### Example 3: Production Kubernetes Setup

**ConfigMap for metrics config:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: onigirazu-metrics
data:
  onigirazu.yml: |
    enable_metrics: true
    metrics_listen_address: 0.0.0.0
    metrics_port: 9090
    metrics_auth_token: ${METRICS_TOKEN}  # From secret
    metrics_ip_whitelist:
      - prometheus-service.monitoring.svc.cluster.local
```

**ServiceMonitor for Prometheus Operator:**

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: onigirazu-metrics
spec:
  selector:
    matchLabels:
      app: onigirazu
  endpoints:
    - port: metrics
      interval: 30s
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      scheme: http
      path: /metrics
```

---

## API Endpoints

### /metrics (Prometheus Format)

**Authentication:** Optional (if configured)
**Method:** GET
**Returns:** Prometheus text format

```bash
curl -H "Authorization: Bearer token" http://localhost:9090/metrics
```

**Response:**

```
# HELP onigirazu_playbooks_total Total number of playbooks executed
# TYPE onigirazu_playbooks_total counter
onigirazu_playbooks_total 5

# HELP onigirazu_tasks_total Total number of tasks executed
# TYPE onigirazu_tasks_total counter
onigirazu_tasks_total{module="shell",status="success"} 42
onigirazu_tasks_total{module="file",status="failed"} 2

# HELP onigirazu_task_duration_seconds Task execution duration in seconds
# TYPE onigirazu_task_duration_seconds histogram
onigirazu_task_duration_seconds_bucket{le="0.1",host="web01",module="shell"} 12
onigirazu_task_duration_seconds_bucket{le="1.0",host="web01",module="shell"} 38
```

### /summary (JSON Format)

**Authentication:** Optional (if configured)
**Method:** GET
**Returns:** JSON summary

```bash
curl -H "Authorization: Bearer token" \
  http://localhost:9090/summary | jq .
```

**Response:**

```json
{
  "overview": {
    "playbooks_executed": 5,
    "plays_executed": 8,
    "tasks_executed": 127,
    "tasks_succeeded": 125,
    "tasks_failed": 2,
    "tasks_skipped": 0,
    "tasks_changed": 45,
    "success_rate": 98.43,
    "uptime": "2h30m45s"
  },
  "performance": {
    "total_execution_time": "15m23s",
    "average_task_time": "7.25s",
    "min_task_time": "0.1s",
    "max_task_time": "45.2s",
    "max_concurrent_tasks": 5,
    "current_concurrent_tasks": 0
  },
  "modules": {
    "usage": {
      "shell": 42,
      "file": 38,
      "apt": 15,
      "systemd": 12
    },
    "top_modules": [
      {"name": "shell", "count": 42},
      {"name": "file", "count": 38}
    ],
    "error_rates": {
      "shell": 0.05,
      "file": 0.0
    }
  },
  "errors": {
    "by_module": {"shell": 2},
    "by_type": {"execution": 1, "validation": 1},
    "total": 2
  },
  "hosts": {
    "connected": 5,
    "unreachable": 0,
    "execution_times_ms": {
      "web01": 15230,
      "web02": 14950,
      "db01": 15100
    },
    "slowest_hosts": [
      {"host": "web01", "time_ms": 15230},
      {"host": "db01", "time_ms": 15100}
    ]
  },
  "cache": {
    "hits": 234,
    "misses": 45,
    "hit_rate": 83.87,
    "total": 279
  },
  "resource_usage": {
    "memory_usage_bytes": 524288000,
    "cpu_usage_percent": 45.2,
    "network_bytes": 1048576,
    "disk_io_bytes": 2097152
  },
  "timestamp": "2025-01-15T10:30:45Z"
}
```

### /health (Health Check)

**Authentication:** Optional (if configured)
**Method:** GET
**Returns:** Plain text

```bash
curl -H "Authorization: Bearer token" http://localhost:9090/health
```

**Response:**

```
OK
```

**Status Codes:**

- `200 OK` - Metrics server is running normally
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client IP not in whitelist

---

## Security Best Practices

### ✅ DO

1. **Always use localhost binding by default**

   ```yaml
   metrics_listen_address: 127.0.0.1  # Default and secure
   ```

2. **Use strong tokens** (32+ chars, random)

   ```yaml
   metrics_auth_token: "a7f3k9m2n8b1x4c5v6z7h8j9w0e1r2t3"
   ```

3. **Rotate tokens regularly**
   - Update every 90 days
   - Change on employee departure
   - Use secrets management (Vault, K8s Secrets)

4. **Monitor access logs**
   - Enable debug logging
   - Watch for repeated 401/403 errors
   - Alert on suspicious patterns

5. **Use TLS/HTTPS in production**
   - Implement reverse proxy (nginx, HAProxy)
   - Terminate TLS at proxy
   - Metrics server speaks HTTP only

6. **Restrict by network**

   ```yaml
   metrics_listen_address: 10.0.1.100  # Internal network only
   metrics_ip_whitelist:
     - 10.0.1.0/24      # Your network
     - 10.0.2.10        # Prometheus server
   ```

7. **Enable in log output**

   ```bash
   ONIGIRAZU_LOG_LEVEL=info
   # Logs: "Metrics server configured - endpoints available at..."
   ```

### ❌ DON'T

1. **Don't expose to the internet without authentication**

   ```yaml
   # WRONG - DANGEROUS
   metrics_listen_address: 0.0.0.0
   metrics_auth_token: ""  # No auth!
   ```

2. **Don't use predictable tokens**

   ```yaml
   # WRONG
   metrics_auth_token: "password123"
   metrics_auth_token: "admin"
   ```

3. **Don't commit secrets to version control**

   ```bash
   # WRONG - Never do this
   git add onigirazu.yml  # Contains metrics_auth_token
   ```

   **Instead:**

   ```yaml
   # Use environment variables or secrets files
   metrics_auth_token: ${METRICS_TOKEN}  # Filled at runtime
   ```

4. **Don't disable all security in production**

   ```yaml
   # WRONG - Only acceptable for local development
   enable_metrics: true
   metrics_listen_address: 0.0.0.0
   metrics_ip_whitelist: []      # Anyone can access
   metrics_auth_token: ""        # No authentication
   ```

5. **Don't expose metrics details publicly**
   - Module names could reveal internal tools
   - Host names could reveal infrastructure
   - Error patterns could reveal vulnerabilities

---

## Troubleshooting

### "Connection refused" error

**Symptom:** `curl: (7) Failed to connect to localhost port 9090`

**Causes & Solutions:**

1. **Metrics not enabled**

   ```yaml
   enable_metrics: true  # Ensure this is set
   ```

2. **Metrics server not yet started**
   - Wait 2-3 seconds after execution starts
   - Check logs: `Metrics server configured`

3. **Wrong port**

   ```bash
   # Verify configured port
   curl http://localhost:9090/health

   # Try different port if configured elsewhere
   curl http://localhost:8090/health
   ```

4. **Listening on different address**

   ```bash
   # If listening on 0.0.0.0, try with IP instead
   curl http://192.168.1.100:9090/health
   ```

### "401 Unauthorized" error

**Symptom:** `{"error": "Unauthorized: invalid or missing authentication token"}`

**Causes & Solutions:**

1. **Missing Authorization header**

   ```bash
   # WRONG
   curl http://localhost:9090/metrics

   # CORRECT
   curl -H "Authorization: Bearer your-token" \
     http://localhost:9090/metrics
   ```

2. **Wrong token format**

   ```bash
   # WRONG
   curl -H "Authorization: your-token" http://localhost:9090/metrics

   # CORRECT (must be "Bearer <token>")
   curl -H "Authorization: Bearer your-token" http://localhost:9090/metrics
   ```

3. **Token mismatch**

   ```bash
   # Verify token matches config
   # Config: metrics_auth_token: "prod-secret-123"
   # Request must use: "Authorization: Bearer prod-secret-123"
   ```

4. **Token containing special characters**
   - URL-encode or properly escape in requests
   - Use quotes around token

### "403 Forbidden" error

**Symptom:** `{"error": "Forbidden: IP not in whitelist"}`

**Causes & Solutions:**

1. **Client IP not in whitelist**

   ```bash
   # Check your IP
   curl http://icanhazip.com

   # Add to whitelist
   metrics_ip_whitelist:
     - 203.0.113.100  # Your IP
   ```

2. **Behind proxy/load balancer**
   - The server sees proxy IP, not your IP
   - Check `X-Forwarded-For` or `X-Real-IP` headers
   - Whitelist proxy IP instead

   ```bash
   # From proxy
   curl -H "X-Forwarded-For: 203.0.113.100" \
     http://localhost:9090/metrics
   ```

3. **IPv6 vs IPv4 mismatch**
   - Whitelist may be `::1` for localhost IPv6
   - Add both: `127.0.0.1` and `::1`

### Metrics not updating

**Symptom:** `/summary` shows stale data or zeros

**Causes & Solutions:**

1. **No playbooks executed yet**
   - Run a playbook to generate metrics
   - Check: `tasks_executed > 0`

2. **Metrics collection disabled**

   ```bash
   # Verify in logs
   log_level: debug  # Enable debug to see metrics startup
   ```

3. **Server running but metrics not recording**
   - Check metrics are passed to engine during execution
   - Ensure no errors in execution

### High memory usage from metrics server

**Symptom:** Memory usage grows over time

**Causes & Solutions:**

1. **Long-running processes accumulate metrics**
   - Expected behavior for continuous recording
   - Memory stabilizes based on unique label combinations

2. **Large number of hosts/tasks/modules**
   - Prometheus metrics grow with label cardinality
   - Consider aggregating in Prometheus instead

3. **Memory leak in metrics collection**
   - Report issue if memory usage exceeds 500MB
   - Include reproduction steps

---

## References

- [Prometheus Monitoring Format](https://prometheus.io/docs/instrumenting/exposition_formats/)
- [Bearer Token Authentication (RFC 6750)](https://tools.ietf.org/html/rfc6750)
- [HTTP Status Codes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)

---

## Support

For issues or questions:

1. Check logs with `--verbose` flag
2. Enable debug logging: `log_level: debug`
3. Review configuration with `onigirazu show-config`
4. Report issues on GitHub with logs and configuration (sanitized)

# Onigirazu - Optimization and New Features Analysis

## 📊 Current Project State

### Strengths ✅

- Modern Go architecture with proper concurrency patterns
- Implemented SSH connection pooling (40-60% improvement)
- Implemented buffer pool (30% improvement)
- Implemented package information caching (66% improvement)
- Comprehensive module system (22 modules)
- Extended workflow orchestration system
- Monitoring and metrics collection system
- Security validation layer

### Achieved Results 🎯

**Overall performance improvement: 86%** (from 15s to 2.1s)

---

## 🚀 Optimization Opportunities

### 1. Complete Optimization Phases (HIGH PRIORITY)

#### Phase 5: System Facts Caching

- **Expected improvement:** 30-40%
- **Implementation time:** 2-3 hours
- **What to do:**
  - Cache results of `uname`, `lsb_release`, `hostname`
  - Avoid repeated SSH calls
  - Use TTL for cache invalidation

#### Phase 6: Template Compilation Caching

- **Expected improvement:** 20-30%
- **Implementation time:** 3-4 hours
- **What to do:**
  - Cache compiled Jinja2 templates
  - Implement LRU eviction
  - Add file change detection

#### Phase 7: Parallel Task Execution

- **Expected improvement:** 50-80% for multi-host scenarios
- **Implementation time:** 5-6 hours
- **What to do:**
  - Execute tasks on multiple hosts in parallel
  - Implement worker pool
  - Add parallelization strategies (linear, free, batch)

**Projected result after all phases: 97% improvement (from 15s to 0.5s)**

---

### 2. Improve Test Coverage (HIGH PRIORITY)

#### Current Coverage State

```
internal/core:        0.0%  ❌ Critical
internal/execution:   0.0%  ❌ Critical
internal/inventory:   0.0%  ❌ Critical
internal/monitoring:  0.0%  ❌ Critical
internal/workflow:    0.0%  ❌ Critical
internal/modules:     7.8%  ⚠️ Very low
internal/cache:      47.8%  ⚠️ Low
```

#### Target Coverage: 70-80%

**What needs to be done:**

1. Add tests for core engine
2. Add tests for workflow orchestrator
3. Add integration tests for modules
4. Add tests for inventory manager
5. Add tests for state manager

---

### 3. Fix Security Issues (HIGH PRIORITY)

#### Issue 1: Disabled SSH Host Key Verification

**Current code:**

```go
HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ❌ SECURITY RISK
```

**Solution:**

- Implement HostKeyManager
- Use known_hosts file
- Add option for strict verification

#### Issue 2: Missing Context Cancellation Support

**Solution:**

- Add context.Context to all SSH operations
- Implement graceful shutdown
- Add timeouts for all operations

#### Issue 3: Insufficient Command Validation

**Solution:**

- Extend CommandValidator
- Add shell injection checks
- Add whitelist of allowed commands

---

### 4. Additional Performance Optimizations

#### A. Health Checks for Connection Pool

- Active connection health monitoring
- Automatic removal of dead connections
- Pool health metrics

#### B. Batch Operations for Modules

- Install multiple packages in one transaction
- Group file operations
- Optimize network calls

#### C. Lazy Loading for Facts

- Collect only necessary facts
- Caching at individual fact level
- Reduce startup overhead

#### D. Memory Pool for Large Transfers

- Buffer pool for file operations
- Reduce memory allocations
- Improve GC performance

---

## 🎯 New Features

### 1. Plugin System (HIGH VALUE)

**Why:** Allow users to extend Onigirazu without code changes

**Plugin Types:**

- **Module plugins** - new modules
- **Callback plugins** - event hooks
- **Inventory plugins** - inventory sources
- **Filter plugins** - template filters

**Usage Example:**

```yaml
plugins:
  - name: slack-notifier
    type: callback
    config:
      webhook_url: https://hooks.slack.com/...

  - name: aws-inventory
    type: inventory
    config:
      region: us-east-1
```

---

### 2. Secrets Manager Integration (HIGH VALUE)

**Why:** Secure secrets management

#### A. HashiCorp Vault

**Features:**

- Read secrets from Vault
- Dynamic credentials
- Automatic token renewal
- Support for various auth methods (Token, AppRole, Kubernetes)

**Usage Example:**

```yaml
secrets:
  provider: vault
  config:
    address: https://vault.example.com
    auth_method: token
    token: ${VAULT_TOKEN}

vars:
  db_password: "{{ vault('secret/database/password') }}"
  api_key: "{{ vault('secret/api/key') }}"
```

#### B. Bitwarden

**Features:**

- Read secrets from Bitwarden Vault
- Support for organizational collections
- Bitwarden CLI integration
- Support for self-hosted Bitwarden (Vaultwarden)
- Secret caching with TTL

**Usage Example:**

```yaml
secrets:
  provider: bitwarden
  config:
    server: https://vault.bitwarden.com  # or self-hosted
    email: admin@example.com
    organization_id: "org-uuid"
    session_timeout: 3600

vars:
  db_password: "{{ bitwarden('database-credentials', 'password') }}"
  ssh_key: "{{ bitwarden('ssh-keys', 'private_key') }}"
  api_token: "{{ bitwarden('api-tokens', 'github') }}"
```

**Integration Architecture:**

```go
// internal/secrets/provider.go
type SecretProvider interface {
    GetSecret(path string, field string) (string, error)
    ListSecrets(path string) ([]string, error)
    Close() error
}

// internal/secrets/bitwarden/client.go
type BitwardenClient struct {
    cliPath    string
    session    string
    cache      *SecretCache
    orgID      string
}

func (bc *BitwardenClient) GetSecret(itemName, field string) (string, error) {
    // Check cache first
    if cached, ok := bc.cache.Get(itemName, field); ok {
        return cached, nil
    }

    // Execute: bw get item <name> --session <session>
    cmd := exec.Command(bc.cliPath, "get", "item", itemName,
                        "--session", bc.session)
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }

    // Parse JSON and extract field
    var item BitwardenItem
    json.Unmarshal(output, &item)

    value := bc.extractField(item, field)
    bc.cache.Set(itemName, field, value)

    return value, nil
}
```

**Bitwarden Advantages:**

- ✅ Open-source and free for personal use
- ✅ Self-hosting capability (Vaultwarden)
- ✅ Simpler to configure than Vault
- ✅ 2FA support
- ✅ Cross-platform
- ✅ Convenient UI for secrets management

---

### 3. Distributed Execution (MEDIUM VALUE)

**Why:** Scale to thousands of hosts

**Architecture:**

- Master-worker model
- Load balancing
- Fault tolerance
- Result aggregation

**Usage:**

```yaml
execution:
  mode: distributed
  workers:
    - address: worker1.example.com:9000
    - address: worker2.example.com:9000
    - address: worker3.example.com:9000
```

---

### 4. Web UI Dashboard (MEDIUM VALUE)

**Why:** Visual monitoring and management

**Components:**

- Dashboard with real-time metrics
- Playbook management
- Inventory view
- Metrics visualization
- Execution history
- Playbook editor

**Technologies:**

- Backend: Go REST API + WebSocket
- Frontend: HTML/CSS/JavaScript (Vue.js/React possible)
- Charts: Chart.js or D3.js

---

### 5. Dynamic Inventory Sources (MEDIUM VALUE)

**Why:** Integration with cloud providers and CMDB

**Supported Sources:**

- **AWS EC2** - automatic instance discovery
- **Azure** - Azure VM integration
- **GCP** - Google Cloud Platform
- **Kubernetes** - pods as hosts
- **Consul** - service discovery
- **Custom API** - custom sources

**Configuration Example:**

```yaml
dynamic_sources:
  - type: aws_ec2
    region: us-east-1
    filters:
      tag:Environment: production
      instance-state-name: running

  - type: kubernetes
    kubeconfig: ~/.kube/config
    namespace: default
    selector: app=web
```

---

### 6. Rollback Support (MEDIUM VALUE)

**Why:** Safe change rollback

**Features:**

- Automatic snapshot creation
- Rollback to previous state
- Change history
- Selective rollback (individual tasks)

**Usage Example:**

```yaml
- name: "Deploy application"
  module: "copy"
  src: app.jar
  dest: /opt/app/app.jar
  backup: true

- name: "Rollback on failure"
  module: "rollback"
  when: "{{ deploy_result.failed }}"
```

---

### 7. Ansible Compatibility (HIGH VALUE)

**Why:** Easy migration from Ansible

**Features:**

- Parse Ansible playbooks
- Module conversion
- Ansible inventory support
- Jinja2 template compatibility
- Variable conversion

**Usage:**

```bash
# Run Ansible playbook
onigirazu --ansible-mode -playbook site.yml -inventory hosts

# Convert playbook
onigirazu convert --from ansible --to onigirazu site.yml
```

---

### 8. Dry-run with Diff Preview (LOW VALUE, EASY)

**Why:** Show what will change before execution

**Example Output:**

```diff
--- Before
+++ After
@@ -1,3 +1,3 @@
 server {
-    listen 80;
+    listen 8080;
     server_name example.com;
 }
```

---

### 9. Task Delegation (LOW VALUE)

**Why:** Execute tasks on other hosts

**Example:**

```yaml
- name: "Update load balancer"
  module: "command"
  command: "update-lb.sh"
  delegate_to: loadbalancer
  run_once: true
```

---

### 10. Auto-generate Module Documentation (LOW VALUE, EASY)

**Why:** Automatic documentation from code

**Features:**

- Generate Markdown documentation
- Usage examples
- Parameter descriptions
- Types and default values

---

## 🎯 Quick Wins (can be done today)

### 1. Health Check Endpoint

```bash
curl http://localhost:8080/health
{"status":"healthy","version":"1.7.0"}
```

### 2. Version Command

```bash
onigirazu --version
Onigirazu 1.7.0
```

### 3. Playbook Validation

```bash
onigirazu --validate playbook.yml
✅ Playbook is valid
```

### 4. Module List Command

```bash
onigirazu --list-modules
Available modules:
  - file
  - copy
  - package
  - service
  ...
```

### 5. Execution Summary

```
=== Execution Summary ===
Total tasks: 15
Success: 14
Failed: 1
Changed: 8
Duration: 2.1s
```

---

## 📋 Priority Matrix

### High Priority (do first)

1. ✅ **Complete Phase 5: Facts Caching** - quick win, high impact
2. ✅ **Fix SSH host key verification** - critical security issue
3. ✅ **Add context cancellation** - important for reliability
4. ✅ **Improve test coverage** - quality foundation
5. ✅ **Plugin system** - high value for extensibility

### Medium Priority (next quarter)

6. ⚠️ **Complete Phase 6: Template Caching** - performance improvement
7. ⚠️ **Complete Phase 7: Parallel Execution** - scalability
8. ⚠️ **Secrets Manager Integration (Vault + Bitwarden)** - enterprise feature
9. ⚠️ **Web UI Dashboard** - user experience
10. ⚠️ **Dynamic Inventory** - cloud integration

### Low Priority (future)

11. 📋 **Distributed Execution** - advanced scalability
12. 📋 **Rollback Support** - nice to have
13. 📋 **Ansible Compatibility** - migration help
14. 📋 **Dry-run Diff** - UX improvement
15. 📋 **Task Delegation** - advanced feature

---

## 📈 Expected Impact

### Performance Improvement

| Stage | Execution Time | Improvement |
|------|---------------|------------|
| Baseline | 15s | - |
| After Phase 4 | 2.1s | 86% faster |
| **After Phase 5** | **1.5s** | **90% faster** |
| **After Phase 6** | **1.05s** | **93% faster** |
| **After Phase 7** | **0.5s** | **97% faster** |

### Test Coverage Improvement

| Component | Current | Target | Improvement |
|-----------|-------|------|------------|
| Core | 0% | 80% | +80% |
| Modules | 7.8% | 70% | +62.2% |
| Workflow | 0% | 75% | +75% |
| **Overall** | **~25%** | **~75%** | **+50%** |

---

## 🗓️ Recommended Roadmap

### Month 1: Foundation

- **Week 1:** Complete Phase 5 (facts caching)
- **Week 2:** Fix security issues
- **Week 3-4:** Improve test coverage to 50%

### Month 2: Performance

- **Week 1:** Complete Phase 6 (template caching)
- **Week 2-3:** Complete Phase 7 (parallel execution)
- **Week 4:** Performance testing and optimization

### Month 3: Features

- **Week 1-2:** Implement plugin system
- **Week 3:** Add Vault and Bitwarden integration
- **Week 4:** Quick wins and polish

### Month 4: Enterprise Features

- **Week 1-2:** Web UI Dashboard
- **Week 3:** Dynamic inventory
- **Week 4:** Documentation and examples

---

## 📊 Summary

### Main Improvement Opportunities

1. **Complete optimization phases** → 97% performance improvement
2. **Improve test coverage** → reliability and maintainability
3. **Fix security issues** → production readiness
4. **Add plugin system** → community contributions
5. **Enterprise features** → Vault/Bitwarden, Web UI, dynamic inventory

### Forecast

Onigirazu has every chance to become a serious Ansible alternative with these improvements.

### Next Steps

1. Start with Phase 5 (facts caching) - 2-3 hours of work
2. Fix critical security issues - 1 day
3. Improve test coverage - 1-2 weeks
4. Implement plugin system - 1 week

---

**Created:** $(date)
**Version:** Based on current main branch
**Analyst:** AI code analysis system

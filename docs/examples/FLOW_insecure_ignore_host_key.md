# Flow Diagram: insecure_ignore_host_key

## Complete Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    1. INVENTORY FILE                            │
│                                                                 │
│  inventory.yml:                                                 │
│  hosts:                                                         │
│    myhost:                                                      │
│      address: 192.168.1.100                                     │
│      insecure_ignore_host_key: true  ← SET HERE                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    2. PARSER                                    │
│                                                                 │
│  internal/parser/inventory_parser.go                            │
│  - Reads YAML/JSON/TOML                                         │
│  - Parses insecure_ignore_host_key field                        │
│  - Creates Host struct                                          │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    3. HOST STRUCT                               │
│                                                                 │
│  pkg/types/types.go:                                            │
│  type Host struct {                                             │
│      Name                  string                               │
│      Address               string                               │
│      InsecureIgnoreHostKey bool  ← STORED HERE                  │
│      ...                                                        │
│  }                                                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    4. PLAYBOOK EXECUTION                        │
│                                                                 │
│  playbook.yml:                                                  │
│  - name: Run task                                               │
│    hosts: myhost                                                │
│    tasks:                                                       │
│      - name: Execute command                                    │
│        shell:                                                   │
│          command: hostname                                      │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    5. MODULE EXECUTE                            │
│                                                                 │
│  func (m *Module) Execute(                                      │
│      ctx context.Context,                                       │
│      host types.Host,  ← HOST OBJECT PASSED                     │
│      args map[string]interface{}                                │
│  ) (types.TaskResult, error) {                                  │
│      // host.InsecureIgnoreHostKey is available here            │
│  }                                                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    6. CREATE EXECUTOR                           │
│                                                                 │
│  Pattern 1:                                                     │
│  output, err := m.WithExecutorResult(host, func(...) {...})    │
│                                                                 │
│  Pattern 2:                                                     │
│  err := m.WithExecutor(host, func(...) {...})                  │
│                                                                 │
│  Pattern 3:                                                     │
│  exec, err := m.CreateExecutor(host)  ← HOST PASSED            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    7. BASE EXECUTOR MODULE                      │
│                                                                 │
│  internal/modules/base_executor_module.go:                      │
│  func (b *BaseExecutorModule) CreateExecutor(                   │
│      host types.Host  ← HOST RECEIVED                           │
│  ) (*executor.CommandExecutor, error) {                         │
│      return executor.NewCommandExecutor(host)  ← PASS HOST      │
│  }                                                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    8. COMMAND EXECUTOR                          │
│                                                                 │
│  internal/executor/command_executor.go:                         │
│  func NewCommandExecutor(                                       │
│      host types.Host  ← HOST RECEIVED                           │
│  ) (*CommandExecutor, error) {                                  │
│      if host.Address != "localhost" {                           │
│          client, err := ssh.NewClient(host)  ← PASS HOST        │
│      }                                                          │
│  }                                                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    9. SSH CLIENT                                │
│                                                                 │
│  internal/ssh/client.go:                                        │
│  func NewClient(host types.Host) (*Client, error) {            │
│      return NewClientWithHostKeyManager(                        │
│          host,                                                  │
│          NewHostKeyManagerWithInsecure(                         │
│              "",                                                │
│              false,                                             │
│              host.InsecureIgnoreHostKey  ← USED HERE            │
│          )                                                      │
│      )                                                          │
│  }                                                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    10. HOST KEY MANAGER                         │
│                                                                 │
│  internal/ssh/hostkey.go:                                       │
│  func NewHostKeyManagerWithInsecure(                            │
│      knownHostsFile string,                                     │
│      strictMode bool,                                           │
│      insecure bool  ← RECEIVED                                  │
│  ) *HostKeyManager {                                            │
│      hkm := &HostKeyManager{                                    │
│          insecure: insecure,  ← STORED                          │
│          ...                                                    │
│      }                                                          │
│      if !insecure {                                             │
│          hkm.loadKnownHosts()  ← SKIP IF INSECURE               │
│      }                                                          │
│      return hkm                                                 │
│  }                                                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    11. SSH CONNECTION                           │
│                                                                 │
│  config := &ssh.ClientConfig{                                   │
│      User: host.User,                                           │
│      Auth: auth,                                                │
│      HostKeyCallback: hostKeyManager.VerifyHostKey,  ← SET      │
│      Timeout: 30 * time.Second,                                 │
│  }                                                              │
│                                                                 │
│  client, err := ssh.Dial("tcp", address, config)               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    12. HOST KEY VERIFICATION                    │
│                                                                 │
│  func (hkm *HostKeyManager) VerifyHostKey(                      │
│      hostname string,                                           │
│      remote net.Addr,                                           │
│      key ssh.PublicKey                                          │
│  ) error {                                                      │
│      if hkm.insecure {                                          │
│          return nil  ← SKIP VERIFICATION IF INSECURE            │
│      }                                                          │
│                                                                 │
│      // Check against known_hosts                              │
│      if knownKey, exists := hkm.knownHosts[hostname]; exists {  │
│          if keysEqual(key, knownKey) {                          │
│              return nil  ← ACCEPT IF MATCHES                    │
│          }                                                      │
│          return fmt.Errorf("key mismatch")  ← REJECT IF DIFFERS │
│      }                                                          │
│                                                                 │
│      if hkm.strictMode {                                        │
│          return fmt.Errorf("unknown host")  ← REJECT IF STRICT  │
│      }                                                          │
│                                                                 │
│      hkm.addHostKey(hostname, key)  ← ADD IF NOT STRICT         │
│      return nil                                                 │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
```

## Decision Tree

```
                    ┌─────────────────────┐
                    │ SSH Connection      │
                    │ Attempt             │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │ VerifyHostKey()     │
                    │ called              │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │ insecure == true?   │
                    └──────────┬──────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
               YES                           NO
                │                             │
                ▼                             ▼
    ┌───────────────────────┐   ┌───────────────────────┐
    │ return nil            │   │ Check known_hosts     │
    │ (ACCEPT ANY KEY)      │   └───────────┬───────────┘
    │ ⚠️ INSECURE!          │               │
    └───────────────────────┘               ▼
                                ┌───────────────────────┐
                                │ Host in known_hosts?  │
                                └───────────┬───────────┘
                                            │
                            ┌───────────────┴───────────────┐
                            │                               │
                           YES                             NO
                            │                               │
                            ▼                               ▼
                ┌───────────────────────┐   ┌───────────────────────┐
                │ Keys match?           │   │ strictMode == true?   │
                └───────────┬───────────┘   └───────────┬───────────┘
                            │                           │
                ┌───────────┴───────────┐   ┌───────────┴───────────┐
                │                       │   │                       │
               YES                     NO  YES                     NO
                │                       │   │                       │
                ▼                       ▼   ▼                       ▼
    ┌───────────────────┐   ┌───────────────────┐   ┌───────────────────┐
    │ return nil        │   │ return error      │   │ Add to known_hosts│
    │ (ACCEPT)          │   │ "key mismatch"    │   │ return nil        │
    │ ✅ SECURE         │   │ ✅ SECURE         │   │ (ACCEPT)          │
    └───────────────────┘   └───────────────────┘   │ ⚠️ TOFU           │
                                                    └───────────────────┘
```

## Configuration Priority

```
┌─────────────────────────────────────────────────────────────────┐
│                    PRIORITY ORDER                               │
│                    (Highest to Lowest)                          │
└─────────────────────────────────────────────────────────────────┘

    1️⃣  HOST LEVEL
    ┌─────────────────────────────────────┐
    │ hosts:                              │
    │   myhost:                           │
    │     insecure_ignore_host_key: true  │ ← HIGHEST PRIORITY
    └─────────────────────────────────────┘
                    │
                    │ If not set, check ↓
                    ▼
    2️⃣  GROUP LEVEL
    ┌─────────────────────────────────────┐
    │ groups:                             │
    │   mygroup:                          │
    │     vars:                           │
    │       insecure_ignore_host_key: true│ ← MEDIUM PRIORITY
    └─────────────────────────────────────┘
                    │
                    │ If not set, check ↓
                    ▼
    3️⃣  ALL GROUP LEVEL
    ┌─────────────────────────────────────┐
    │ groups:                             │
    │   all:                              │
    │     vars:                           │
    │       insecure_ignore_host_key: true│ ← LOW PRIORITY
    └─────────────────────────────────────┘
                    │
                    │ If not set, use ↓
                    ▼
    4️⃣  DEFAULT
    ┌─────────────────────────────────────┐
    │ insecure_ignore_host_key: false     │ ← DEFAULT (SECURE)
    └─────────────────────────────────────┘
```

## Example Scenarios

### Scenario 1: Host-level Override

```yaml
hosts:
  server1:
    insecure_ignore_host_key: false  # ← This wins

groups:
  mygroup:
    hosts:
      - server1
    vars:
      insecure_ignore_host_key: true  # ← This is ignored
```

**Result:** `server1` uses **SECURE** mode (host-level takes precedence)

### Scenario 2: Group-level Inheritance

```yaml
hosts:
  server1:
    address: 192.168.1.100
    # insecure_ignore_host_key not set

groups:
  mygroup:
    hosts:
      - server1
    vars:
      insecure_ignore_host_key: true  # ← This applies
```

**Result:** `server1` uses **INSECURE** mode (inherited from group)

### Scenario 3: Default Behavior

```yaml
hosts:
  server1:
    address: 192.168.1.100
    # insecure_ignore_host_key not set

groups:
  mygroup:
    hosts:
      - server1
    # insecure_ignore_host_key not set
```

**Result:** `server1` uses **SECURE** mode (default)

## Security Implications

```
┌─────────────────────────────────────────────────────────────────┐
│                    SECURE MODE (default)                        │
│                    insecure_ignore_host_key: false              │
└─────────────────────────────────────────────────────────────────┘

    Client                    Network                    Server
      │                          │                          │
      │  1. Connect              │                          │
      ├─────────────────────────►│─────────────────────────►│
      │                          │                          │
      │  2. Server sends host key│                          │
      │◄─────────────────────────┤◄─────────────────────────┤
      │                          │                          │
      │  3. Verify against       │                          │
      │     known_hosts          │                          │
      │  ✅ Match? Accept        │                          │
      │  ❌ No match? Reject     │                          │
      │                          │                          │
      │  4. Continue if verified │                          │
      ├─────────────────────────►│─────────────────────────►│
      │                          │                          │

    ✅ Protected against MITM attacks
    ✅ Detects server changes
    ✅ Production-ready


┌─────────────────────────────────────────────────────────────────┐
│                    INSECURE MODE                                │
│                    insecure_ignore_host_key: true               │
└─────────────────────────────────────────────────────────────────┘

    Client                    Network                    Server
      │                          │                          │
      │  1. Connect              │                          │
      ├─────────────────────────►│─────────────────────────►│
      │                          │                          │
      │  2. Server sends host key│                          │
      │◄─────────────────────────┤◄─────────────────────────┤
      │                          │                          │
      │  3. Accept ANY key       │                          │
      │  ⚠️ NO VERIFICATION      │                          │
      │                          │                          │
      │  4. Continue             │                          │
      ├─────────────────────────►│─────────────────────────►│
      │                          │                          │

    ❌ Vulnerable to MITM attacks
    ❌ Cannot detect server changes
    ⚠️ Development/testing only


    MITM Attack Scenario:

    Client                    Attacker                   Real Server
      │                          │                          │
      │  1. Connect              │                          │
      ├─────────────────────────►│                          │
      │                          │  2. Intercept            │
      │                          ├─────────────────────────►│
      │  3. Attacker's key       │                          │
      │◄─────────────────────────┤                          │
      │  4. Accept (insecure)    │                          │
      │  ⚠️ Connected to attacker│                          │
      │                          │                          │
      │  5. All traffic exposed  │                          │
      ├─────────────────────────►│─────────────────────────►│
      │                          │  Attacker can read/modify│
```

## Summary

| Aspect | Details |
|--------|---------|
| **Set in** | Inventory file (YAML/JSON/TOML) |
| **Stored in** | `types.Host.InsecureIgnoreHostKey` |
| **Passed through** | Module Execute → CreateExecutor → SSH Client |
| **Used in** | `HostKeyManager.VerifyHostKey()` |
| **Effect** | Skip host key verification if `true` |
| **Default** | `false` (secure) |
| **Priority** | Host > Group > All > Default |
| **Security** | ⚠️ Insecure - dev/test only |

---

**Remember:** The setting flows automatically through the entire stack. You don't need to handle it in your module code!

# Ansible Inventory Format Support - Before & After

## The Problem (Before)

### Your Situation

- You have an existing Ansible inventory in YAML format
- You want to use Onigirazu for orchestration
- **Problem**: Onigirazu didn't understand Ansible format
- **Solution Required**: Convert your inventory to Onigirazu format (time-consuming, error-prone)

### Example: Your Ansible Inventory

```yaml
# ansible-hosts.yml (Ansible format)
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      ansible_ssh_private_key_file: ~/.ssh/deploy_key
    web2:
      ansible_host: 192.168.1.11
      ansible_user: deploy

  children:
    webservers:
      hosts:
        web1:
        web2:
      vars:
        http_port: 80
```

### What You Had to Do (Before)

**Step 1: Convert to Onigirazu Format**

```yaml
# onigirazu-hosts.yml (Onigirazu format)
hosts:
  - name: web1
    address: 192.168.1.10
    port: 22
    user: deploy
    key_file: ~/.ssh/deploy_key

  - name: web2
    address: 192.168.1.11
    user: deploy

groups:
  webservers:
    hosts:
      web1:
      web2:
    vars:
      http_port: 80
```

**Step 2: Update your scripts/CI/CD**

```bash
# Before
ansible-playbook deploy.yml -i ansible-hosts.yml

# After you converted
onigirazu apply deploy.yml -i onigirazu-hosts.yml
```

**Step 3: Maintain Two Separate Inventories**

- 😞 Ansible inventory for ansible commands
- 😞 Onigirazu inventory for onigirazu commands
- 😞 Keep both in sync manually
- 😞 Double the maintenance burden

### Problems with This Approach

1. **Manual Conversion**: Error-prone, time-consuming
2. **Duplicate Inventories**: Managing both Ansible and Onigirazu versions
3. **Sync Issues**: Changes in one don't propagate to the other
4. **Learning Curve**: Need to learn Onigirazu format
5. **Tool Lock-in**: If you switch back to pure Ansible, wasted conversion effort

---

## The Solution (After) ✨

### Now You Can

Just use your existing Ansible inventory directly! No conversion needed!

```bash
# Your existing Ansible inventory works directly!
onigirazu apply deploy.yml -i ansible-hosts.yml
```

### How It Works

**Automatic Format Detection:**

```
Your inventory file
        ↓
Is it Ansible format?  → YES → Use Ansible parser
        ↓ NO
Is it Onigirazu format? → YES → Use Onigirazu parser
        ↓ NO
Try other formats (JSON, TOML, INI, simple list)
```

**Transparent Variable Mapping:**

```
Ansible variables          Onigirazu properties
├─ ansible_host       →    address
├─ ansible_port       →    port
├─ ansible_user       →    user
├─ ansible_password   →    password
├─ ansible_ssh_private_key_file → key_file
└─ Custom variables   →    vars (with ansible_ prefix removed)
```

### Real Example: Before vs After

#### Before: Two Separate Inventories

```bash
# Ansible workflow
$ ansible-playbook deploy.yml -i ansible-inventory.yml
$ ansible all -i ansible-inventory.yml -m ping

# Onigirazu workflow (after manual conversion)
$ onigirazu apply deploy.yml -i onigirazu-inventory.yml
$ onigirazu plan deploy.yml -i onigirazu-inventory.yml
```

You had to maintain two different inventory files!

#### After: One Inventory, Both Tools

```bash
# Ansible workflow - use original inventory
$ ansible-playbook deploy.yml -i inventory.yml
$ ansible all -i inventory.yml -m ping

# Onigirazu workflow - use SAME inventory
$ onigirazu apply deploy.yml -i inventory.yml
$ onigirazu plan deploy.yml -i inventory.yml
```

One source of truth! Both tools happy! 🎉

---

## Real-World Scenario: Multi-Cloud Deployment

### Before: Maintaining Multiple Inventory Formats

**Your Ansible inventory:**

```yaml
# ansible-inventory.yml (maintained)
all:
  hosts:
    aws_web_1:
      ansible_host: ec2-web-1.amazonaws.com
      ansible_user: ec2-user
      cloud: aws
      region: us-east-1
    gcp_web_1:
      ansible_host: gcp-web-1.gcp.com
      ansible_user: ubuntu
      cloud: gcp
      region: us-central1
  children:
    webservers:
      hosts:
        aws_web_1:
        gcp_web_1:
      vars:
        http_port: 80
```

**Your Onigirazu inventory (duplicate):**

```yaml
# onigirazu-inventory.yml (maintained separately)
hosts:
  - name: aws_web_1
    address: ec2-web-1.amazonaws.com
    user: ec2-user
    vars:
      cloud: aws
      region: us-east-1
  - name: gcp_web_1
    address: gcp-web-1.gcp.com
    user: ubuntu
    vars:
      cloud: gcp
      region: us-central1

groups:
  webservers:
    hosts:
      aws_web_1:
      gcp_web_1:
    vars:
      http_port: 80
```

**Maintenance nightmare:**

- 😞 Two files to keep in sync
- 😞 Added AWS instance → must update both files
- 😞 Changed port setting → must update both files
- 😞 Team members confused about which is authoritative
- 😞 CI/CD pipelines duplicated for each tool

### After: Single Inventory, Both Tools

```yaml
# inventory.yml (single source of truth)
all:
  hosts:
    aws_web_1:
      ansible_host: ec2-web-1.amazonaws.com
      ansible_user: ec2-user
      cloud: aws
      region: us-east-1
    gcp_web_1:
      ansible_host: gcp-web-1.gcp.com
      ansible_user: ubuntu
      cloud: gcp
      region: us-central1
  children:
    webservers:
      hosts:
        aws_web_1:
        gcp_web_1:
      vars:
        http_port: 80
```

**Usage (both tools, same inventory):**

```bash
# Ansible deployment
$ ansible-playbook deploy.yml -i inventory.yml

# Onigirazu orchestration
$ onigirazu apply orchestrate.yml -i inventory.yml

# Ansible ad-hoc commands
$ ansible webservers -i inventory.yml -m ping

# Onigirazu planning
$ onigirazu plan orchestrate.yml -i inventory.yml

# All tools see the same hosts, groups, and variables!
```

**Benefits:**

- ✅ One file to maintain
- ✅ Added AWS instance → one update
- ✅ Changed port → one change
- ✅ Single source of truth
- ✅ Team clarity
- ✅ Simplified CI/CD

---

## Migration Path: From Pure Ansible to Hybrid Approach

### Stage 1: Using Ansible Only

```bash
# What you do today
ansible-playbook site.yml -i inventory.yml
ansible-playbook deploy.yml -i inventory.yml
```

### Stage 2: Add Onigirazu (Post-Implementation)

```bash
# Old workflow still works
ansible-playbook site.yml -i inventory.yml

# New Onigirazu workflow, same inventory!
onigirazu apply deploy.yml -i inventory.yml

# Both tools share the same inventory.yml
```

### Stage 3: Gradual Migration (Optional)

```bash
# Keep using Ansible for what it's good at
ansible-playbook traditional_tasks.yml -i inventory.yml

# Use Onigirazu for what it excels at
onigirazu apply orchestration.yml -i inventory.yml

# No conflicts, no duplication, no forced migration
```

---

## Feature Comparison: Before vs After

| Feature | Before | After |
|---------|--------|-------|
| Use Ansible inventory with Onigirazu | ❌ No | ✅ Yes |
| Auto-detect inventory format | ❌ No | ✅ Yes |
| Maintain single inventory file | ❌ No | ✅ Yes |
| Ansible compatibility | ❌ Limited | ✅ Full |
| Support `ansible_host` | ❌ No | ✅ Yes |
| Support `ansible_user` | ❌ No | ✅ Yes |
| Support `ansible_port` | ❌ No | ✅ Yes |
| Support SSH keys | ❌ No | ✅ Yes |
| Support nested groups | ❌ No | ✅ Yes |
| Support group variables | ❌ No | ✅ Yes |
| Support custom variables | ✅ Yes | ✅ Yes |
| Backward compatible | ✅ Yes | ✅ Yes |
| Requires conversion | ❌ Yes | ✅ No |
| Maintenance burden | ❌ High | ✅ Low |

---

## Implementation Impact

### No Breaking Changes

- ✅ Existing Onigirazu inventories still work
- ✅ Existing INI, JSON, TOML inventories unchanged
- ✅ All CLI commands work the same
- ✅ All playbooks compatible

### Immediate Benefits

- ✅ Use existing Ansible inventories
- ✅ Reduce maintenance
- ✅ Simplify deployment
- ✅ Enable tool interoperability

### Zero Learning Curve

- ✅ No new syntax to learn
- ✅ No CLI changes
- ✅ No playbook modifications
- ✅ Transparent format detection

---

## Summary

### Before This Implementation

```
┌─────────────────────┐       ┌──────────────────────┐
│  Ansible            │       │  Onigirazu           │
│  inventory.yml      │       │  inventory.yml       │
│ (hand-maintained)   │       │ (hand-converted)     │
│                     │       │                      │
│  all:               │       │  hosts:              │
│    hosts:           │       │    - name: web1      │
│      web1:          │       │      address: ...    │
│        ansible_host │       │  groups:             │
└─────────────────────┘       └──────────────────────┘
         ↓                              ↓
     ansible-playbook            onigirazu apply

Issues:
- Two inventories to maintain
- Manual conversion needed
- Duplication and sync problems
```

### After This Implementation

```
┌─────────────────────┐
│  inventory.yml      │
│                     │
│  all:               │
│    hosts:           │
│      web1:          │
│        ansible_host │
└─────────────────────┘
         ↓
    ┌────────────────┐
    │ Auto-Detection │
    └────────────────┘
         ↓
    ┌────┴────────────────┐
    ↓                     ↓
ansible-playbook    onigirazu apply
    │                     │
    └─────────────────────┘
    (Same inventory, both tools)

Benefits:
- One inventory file
- No conversion needed
- Single source of truth
- Both tools work seamlessly
```

---

**Result**: You can now use your existing Ansible inventory directly with Onigirazu! No conversion, no duplication, no maintenance burden. Both tools share the same inventory file as their single source of truth. 🎉

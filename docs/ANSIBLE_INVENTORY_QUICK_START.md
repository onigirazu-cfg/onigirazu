# Quick Start: Using Ansible Inventory Format with Onigirazu

## The Good News 🎉

**You can now use your existing Ansible inventory files directly with Onigirazu!**

No conversion needed. No configuration changes. Just use them as-is.

## How to Use

### Option 1: Direct Use (Easiest)

Use your Ansible inventory exactly as-is:

```bash
# With your existing Ansible inventory
onigirazu plan playbook.yml -i /path/to/ansible/hosts.yml
onigirazu apply playbook.yml -i /path/to/ansible/hosts.yml
```

### Option 2: Format Auto-Detection

Onigirazu automatically detects both formats:

```bash
# Works with both Ansible and Onigirazu YAML formats
# No need to specify format - it just works!
onigirazu plan playbook.yml -i inventory.yml
```

## What Gets Converted Automatically

Your Ansible inventory parameters are automatically mapped:

```yaml
# Your Ansible inventory
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      ansible_port: 2222
      ansible_ssh_private_key_file: ~/.ssh/id_rsa
```

Gets converted to:

```
web1:
  - Address: 192.168.1.10
  - User: deploy
  - Port: 2222
  - KeyFile: ~/.ssh/id_rsa
```

## Real-World Example

### Your Ansible Inventory

```yaml
# hosts.yml
all:
  hosts:
    app1:
      ansible_host: app1.example.com
      ansible_user: appuser
      ansible_ssh_private_key_file: ~/.ssh/app_key
      version: "2.1.0"

    app2:
      ansible_host: app2.example.com
      ansible_user: appuser
      version: "2.1.0"

    db1:
      ansible_host: db1.example.com
      ansible_user: postgres
      db_version: "14"

  children:
    appservers:
      hosts:
        app1:
        app2:
      vars:
        service_port: 8080
        log_level: debug

    databases:
      hosts:
        db1:
      vars:
        backup_enabled: true
```

### Your Onigirazu Playbook

```yaml
# deploy.yml
name: Deploy Application
plays:
  - name: Deploy to app servers
    hosts: appservers
    tasks:
      - name: Deploy application
        shell:
          cmd: |
            echo "Deploying version {{ version }}"
            ./deploy.sh

      - name: Configure service
        shell:
          cmd: "systemctl restart appservice"

  - name: Backup database
    hosts: databases
    tasks:
      - name: Create backup
        shell:
          cmd: "pg_dump -U {{ ansible_user }} > backup.sql"
```

### Run It

```bash
# Just use your Ansible inventory directly!
onigirazu apply deploy.yml -i hosts.yml

# View the plan first
onigirazu plan deploy.yml -i hosts.yml

# Use in interactive mode
onigirazu interactive -i hosts.yml deploy.yml
```

## Supported Ansible Variables

| Ansible Variable | Purpose |
|-----------------|---------|
| `ansible_host` | Target host IP or hostname |
| `ansible_port` | SSH port (default: 22) |
| `ansible_user` | SSH username |
| `ansible_password` | SSH password (password auth) |
| `ansible_ssh_private_key_file` | Path to SSH private key |
| `ansible_ssh_host_key_checking` | Disable host key verification (set to `false`) |
| Custom variables | Stored and available in playbooks |

## Tips & Tricks

### 1. Group Variables Are Available

```yaml
# inventory.yml
all:
  children:
    webservers:
      hosts:
        web1:
      vars:
        http_port: 80
        https_port: 443
```

Access in playbook: `{{ http_port }}`

### 2. Nested Groups Work

```yaml
all:
  children:
    production:
      children:
        - webservers
        - databases
      vars:
        env: production
```

Target: `hosts: production` → includes all webservers and databases

### 3. Custom Variables Work Alongside Ansible Vars

```yaml
all:
  hosts:
    server1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      # Your custom variables work too!
      app_port: 8080
      db_pool_size: 50
      cache_ttl: 3600
```

Access in playbook: `{{ app_port }}`, `{{ db_pool_size }}`, etc.

### 4. Mix SSH Keys and Passwords

```yaml
all:
  hosts:
    secure_server:
      ansible_host: secure.example.com
      ansible_user: admin
      ansible_ssh_private_key_file: ~/.ssh/secure_key
      ansible_ssh_host_key_checking: false

    basic_server:
      ansible_host: basic.example.com
      ansible_user: user
      ansible_password: password123
```

## Common Scenarios

### Scenario 1: Migrate from Ansible to Onigirazu

**Before (Ansible):**

```bash
ansible-playbook playbook.yml -i hosts.yml
```

**After (Onigirazu):**

```bash
onigirazu apply playbook.yml -i hosts.yml
```

**Changes needed:** None! Use the same inventory file.

### Scenario 2: Keep Both Tools Running

You can use the same inventory with both:

```bash
# Ansible
ansible-playbook your_playbook.yml -i hosts.yml

# Onigirazu
onigirazu apply your_onigirazu_playbook.yml -i hosts.yml
```

No conversion, no duplication, no sync issues!

### Scenario 3: Large Multi-Environment Setup

```yaml
# production-hosts.yml
all:
  children:
    us_east:
      vars:
        region: us-east-1
      hosts:
        prod_web_1:
          ansible_host: 10.1.1.10
        prod_db_1:
          ansible_host: 10.1.2.10

    eu_west:
      vars:
        region: eu-west-1
      hosts:
        prod_web_2:
          ansible_host: 10.2.1.10
```

Works with Onigirazu exactly as-is!

## Troubleshooting

### "No hosts found"

**Check:** Are hosts under `all.hosts:`?

```yaml
all:
  hosts:        # ← This is required
    server1:
      ansible_host: 192.168.1.1
```

### Variables not working

**Check:** Variables must be at host or group level:

```yaml
# ✓ Correct - in vars section
all:
  children:
    webservers:
      hosts:
        web1:
      vars:
        http_port: 80   # ← Works

# ✓ Correct - on host
all:
  hosts:
    web1:
      http_port: 80     # ← Works

# ✗ Wrong - top level
http_port: 80           # ← Doesn't work
```

### Port/User not being used

**Check:** These should be at host level:

```yaml
all:
  hosts:
    myserver:
      ansible_host: 192.168.1.10
      ansible_port: 2222         # ← Host level
      ansible_user: deploy       # ← Host level
```

## More Information

- **Full Documentation**: See `docs/examples/ANSIBLE_INVENTORY_FORMAT.md` in the repository
- **Examples**: Check `examples/inventory-ansible-full.yml`
- **Test Cases**: See `internal/parser/inventory_parser_test.go` for test examples

## Need Help?

1. **Check the examples** in `examples/inventory-ansible-full.yml`
2. **Read the docs** in `docs/examples/ANSIBLE_INVENTORY_FORMAT.md`
3. **Check troubleshooting** section above
4. **Run with debug**: `onigirazu apply --debug playbook.yml -i hosts.yml`

---

**That's it!** Your Ansible inventories are now fully compatible with Onigirazu. No conversion needed. Just use them! 🚀

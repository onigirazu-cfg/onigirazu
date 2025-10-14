# Bitwarden Integration Guide

## 📋 Overview

Onigirazu supports **Bitwarden** as a secret management provider, allowing you to securely store and retrieve sensitive data like passwords, API keys, SSH keys, and certificates directly from your Bitwarden vault.

### Why Bitwarden?

✅ **Open-source** and free for personal use
✅ **Self-hosting** option with Vaultwarden
✅ **Simple setup** - easier than HashiCorp Vault
✅ **Excellent UI** for managing secrets
✅ **2FA support** for enhanced security
✅ **Cross-platform** CLI available
✅ **Secret caching** for performance

---

## 🚀 Quick Start

### 1. Install Bitwarden CLI

```bash
# macOS
brew install bitwarden-cli

# Linux (snap)
sudo snap install bw

# Linux (npm)
npm install -g @bitwarden/cli

# Windows (chocolatey)
choco install bitwarden-cli
```

### 2. Login to Bitwarden

```bash
# Login to Bitwarden
bw login your-email@example.com

# Or configure custom server (self-hosted)
bw config server https://vault.example.com
bw login your-email@example.com
```

### 3. Unlock Vault and Export Session

```bash
# Unlock vault and get session token
export BW_SESSION=$(bw unlock --raw)

# Verify session is active
bw status
```

### 4. Configure Onigirazu

Create or update `onigirazu.yml`:

```yaml
secrets:
  bitwarden:
    enabled: true
    server: "https://vault.bitwarden.com"
    cache_ttl: 300  # Cache secrets for 5 minutes
```

### 5. Use in Playbooks

```yaml
- name: "Deploy with secrets"
  hosts: webservers

  secrets:
    - type: bitwarden
      config:
        cache_ttl: 300

  tasks:
    - name: "Configure database"
      template:
        src: "db.conf.j2"
        dest: "/etc/app/db.conf"
      vars:
        db_password: "{{ bitwarden('database-credentials', 'password') }}"
```

---

## 📖 Usage

### Template Functions

Onigirazu provides three template functions for accessing secrets:

#### 1. `bitwarden(itemName, field)`

Retrieves a secret from Bitwarden.

```yaml
# Get password from login item
password: "{{ bitwarden('my-database', 'password') }}"

# Get username
username: "{{ bitwarden('my-database', 'username') }}"

# Get custom field
api_key: "{{ bitwarden('api-credentials', 'api_key') }}"

# Get notes
license: "{{ bitwarden('license-key', 'notes') }}"
```

#### 2. `vault(path, field)`

Retrieves a secret from HashiCorp Vault (if configured).

```yaml
db_password: "{{ vault('secret/database', 'password') }}"
```

#### 3. `secret(provider, itemName, field)`

Generic function that works with any provider.

```yaml
# Use Bitwarden
password: "{{ secret('bitwarden', 'my-item', 'password') }}"

# Use Vault
token: "{{ secret('vault', 'secret/tokens', 'api_token') }}"
```

---

## 🔑 Bitwarden Item Structure

### Login Items

```yaml
# Bitwarden item: "database-credentials"
# Type: Login
# Username: dbuser
# Password: secretpass123
# Custom fields:
#   - host: db.example.com
#   - port: 5432

# Access in playbook:
db_user: "{{ bitwarden('database-credentials', 'username') }}"
db_pass: "{{ bitwarden('database-credentials', 'password') }}"
db_host: "{{ bitwarden('database-credentials', 'host') }}"
db_port: "{{ bitwarden('database-credentials', 'port') }}"
```

### Secure Notes

```yaml
# Bitwarden item: "ssh-private-key"
# Type: Secure Note
# Notes: -----BEGIN RSA PRIVATE KEY-----
#        MIIEpAIBAAKCAQEA...
#        -----END RSA PRIVATE KEY-----

# Access in playbook:
ssh_key: "{{ bitwarden('ssh-private-key', 'notes') }}"
```

### Custom Fields

```yaml
# Bitwarden item: "api-credentials"
# Type: Login
# Custom fields:
#   - api_key: sk_live_abc123
#   - api_secret: secret_xyz789
#   - webhook_url: https://api.example.com/webhook

# Access in playbook:
api_key: "{{ bitwarden('api-credentials', 'api_key') }}"
api_secret: "{{ bitwarden('api-credentials', 'api_secret') }}"
webhook: "{{ bitwarden('api-credentials', 'webhook_url') }}"
```

---

## 📝 Complete Examples

### Example 1: Database Configuration

**Playbook:**

```yaml
- name: "Configure PostgreSQL"
  hosts: database

  secrets:
    - type: bitwarden
      config:
        cache_ttl: 300

  tasks:
    - name: "Deploy database configuration"
      template:
        src: "postgresql.conf.j2"
        dest: "/etc/postgresql/postgresql.conf"
        mode: "0600"
      vars:
        db_password: "{{ bitwarden('postgres-admin', 'password') }}"
        replication_password: "{{ bitwarden('postgres-replication', 'password') }}"
```

**Template (postgresql.conf.j2):**

```ini
# PostgreSQL Configuration
listen_addresses = '*'
port = 5432

# Authentication
password_encryption = scram-sha-256

# Replication
wal_level = replica
max_wal_senders = 3
primary_conninfo = 'host=primary port=5432 user=replicator password={{ replication_password }}'
```

### Example 2: SSL Certificates

```yaml
- name: "Deploy SSL certificates"
  hosts: webservers

  secrets:
    - type: bitwarden
      config:
        cache_ttl: 600

  tasks:
    - name: "Deploy SSL certificate"
      copy:
        content: "{{ bitwarden('ssl-cert-example-com', 'certificate') }}"
        dest: "/etc/ssl/certs/example.com.crt"
        mode: "0644"

    - name: "Deploy SSL private key"
      copy:
        content: "{{ bitwarden('ssl-cert-example-com', 'private_key') }}"
        dest: "/etc/ssl/private/example.com.key"
        mode: "0600"

    - name: "Reload nginx"
      service:
        name: nginx
        state: reloaded
```

### Example 3: Application Secrets

```yaml
- name: "Deploy application"
  hosts: appservers

  secrets:
    - type: bitwarden
      config:
        cache_ttl: 300

  tasks:
    - name: "Create .env file"
      template:
        src: ".env.j2"
        dest: "/opt/myapp/.env"
        mode: "0600"
      vars:
        db_url: "postgresql://{{ bitwarden('app-database', 'username') }}:{{ bitwarden('app-database', 'password') }}@db.example.com/myapp"
        redis_url: "redis://:{{ bitwarden('app-redis', 'password') }}@redis.example.com:6379"
        jwt_secret: "{{ bitwarden('app-secrets', 'jwt_secret') }}"
        api_key: "{{ bitwarden('app-secrets', 'api_key') }}"
```

**.env.j2 template:**

```bash
# Database
DATABASE_URL={{ db_url }}

# Redis
REDIS_URL={{ redis_url }}

# Security
JWT_SECRET={{ jwt_secret }}
API_KEY={{ api_key }}

# Application
APP_ENV=production
DEBUG=false
```

---

## ⚙️ Configuration Options

### Playbook-level Configuration

```yaml
secrets:
  - type: bitwarden
    config:
      server: "https://vault.bitwarden.com"  # Bitwarden server URL
      session_token: "your-token"            # Optional: session token
      email: "user@example.com"              # Optional: email for login
      password: "your-password"              # Optional: password for login
      cache_ttl: 300                         # Cache TTL in seconds (default: 300)
```

### Global Configuration (onigirazu.yml)

```yaml
secrets:
  bitwarden:
    enabled: true
    server: "https://vault.bitwarden.com"
    cache_ttl: 300
```

### Environment Variables

```bash
# Session token (recommended)
export BW_SESSION=$(bw unlock --raw)

# Or set in playbook config
export BITWARDEN_SESSION="your-session-token"
```

---

## 🔒 Security Best Practices

### 1. Use Session Tokens

**✅ Recommended:**

```bash
# Unlock and export session
export BW_SESSION=$(bw unlock --raw)

# Run playbook
onigirazu -playbook deploy.yml
```

**❌ Not Recommended:**

```yaml
# Don't hardcode passwords in config
secrets:
  - type: bitwarden
    config:
      email: "user@example.com"
      password: "hardcoded-password"  # BAD!
```

### 2. Lock Vault After Use

```bash
# Run playbook
onigirazu -playbook deploy.yml

# Lock vault
bw lock
unset BW_SESSION
```

### 3. Use 2FA

Enable two-factor authentication in your Bitwarden account for enhanced security.

### 4. Limit Cache TTL

```yaml
secrets:
  - type: bitwarden
    config:
      cache_ttl: 300  # 5 minutes - balance between security and performance
```

### 5. Use Self-Hosted Vaultwarden

For maximum control and security:

```bash
# Deploy Vaultwarden
docker run -d --name vaultwarden \
  -v /vw-data/:/data/ \
  -p 80:80 \
  vaultwarden/server:latest

# Configure Onigirazu
bw config server https://vault.yourcompany.com
```

---

## 🐛 Troubleshooting

### Error: "bitwarden provider not configured"

**Solution:** Ensure Bitwarden is configured in your playbook:

```yaml
secrets:
  - type: bitwarden
    config:
      cache_ttl: 300
```

### Error: "not authenticated"

**Solution:** Unlock your vault and export session:

```bash
export BW_SESSION=$(bw unlock --raw)
```

### Error: "field 'X' not found in item 'Y'"

**Solution:** Check the field name in Bitwarden. Common fields:

- Login items: `username`, `password`, `totp`
- All items: `notes`
- Custom fields: exact field name (case-insensitive)

### Error: "failed to get item"

**Solution:** Verify the item exists:

```bash
bw list items --search "item-name" --session $BW_SESSION
```

### Performance Issues

**Solution:** Increase cache TTL:

```yaml
secrets:
  - type: bitwarden
    config:
      cache_ttl: 600  # 10 minutes
```

---

## 🔄 Migration from Ansible Vault

### Ansible Vault

```yaml
# Old way with Ansible Vault
vars:
  db_password: !vault |
    $ANSIBLE_VAULT;1.1;AES256
    66386439653765...
```

### Onigirazu with Bitwarden

```yaml
# New way with Bitwarden
vars:
  db_password: "{{ bitwarden('database-credentials', 'password') }}"
```

### Migration Steps

1. **Export secrets from Ansible Vault:**

```bash
ansible-vault view secrets.yml > secrets.txt
```

2. **Import to Bitwarden:**

```bash
# Create items via CLI
bw create item --name "database-credentials" \
  --username "dbuser" \
  --password "secretpass" \
  --session $BW_SESSION
```

3. **Update playbooks:**

Replace `!vault` references with `{{ bitwarden(...) }}` functions.

---

## 📊 Performance

### Caching

Secrets are cached in memory with configurable TTL:

```yaml
cache_ttl: 300  # 5 minutes
```

**Benefits:**

- ✅ Reduces Bitwarden CLI calls
- ✅ Faster playbook execution
- ✅ Lower API rate limits impact

**Benchmarks:**

| Operation | Without Cache | With Cache |
|-----------|--------------|------------|
| First access | ~200ms | ~200ms |
| Subsequent access | ~200ms | <1ms |
| 100 accesses | ~20s | ~0.2s |

### Best Practices

1. **Group related secrets** in single Bitwarden items
2. **Use appropriate cache TTL** (5-10 minutes recommended)
3. **Minimize secret lookups** by storing in variables
4. **Use batch operations** when possible

---

## 🔮 Future Enhancements

Planned features:

- [ ] Bitwarden Organizations support
- [ ] Collections filtering
- [ ] Automatic token refresh
- [ ] Secret rotation hooks
- [ ] Audit logging
- [ ] Secret versioning
- [ ] Emergency access

---

## 📚 Additional Resources

- [Bitwarden CLI Documentation](https://bitwarden.com/help/cli/)
- [Vaultwarden (Self-hosted)](https://github.com/dani-garcia/vaultwarden)
- [Bitwarden API Reference](https://bitwarden.com/help/api/)
- [Onigirazu Examples](../examples/)

---

## 💬 Support

If you encounter issues:

1. Check [Troubleshooting](#-troubleshooting) section
2. Review [Examples](../examples/bitwarden_example.yml)
3. Open an issue on GitHub
4. Join our community chat

---

**Happy secret managing! 🔐**

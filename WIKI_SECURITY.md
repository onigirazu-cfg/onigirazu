# 🔒 Security

Onigirazu is designed with security as a top priority. This guide covers security features, best practices, and hardening techniques.

## 📋 Security Overview

### Security Principles

- **🔐 Secure by default** - Security features enabled by default
- **🛡️ Defense in depth** - Multiple security layers
- **🔍 Principle of least privilege** - Minimal required permissions
- **📊 Audit and monitoring** - Comprehensive logging
- **🔄 Regular updates** - Security patches and updates

### Security Features

- **SSH host key verification** - Prevents man-in-the-middle attacks
- **Secure authentication** - Multiple authentication methods
- **Encrypted communication** - All network traffic encrypted
- **Access control** - Role-based permissions
- **Audit logging** - Complete audit trail

---

## 🔐 Authentication Security

### SSH Key Management

```bash
# Generate strong SSH keys
ssh-keygen -t ed25519 -b 4096 -C "your_email@example.com"

# Use key-based authentication
ssh-copy-id user@host

# Test SSH connection
ssh user@host
```

### SSH Configuration

```bash
# ~/.ssh/config
Host *
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.ssh/known_hosts
    IdentitiesOnly yes
    PasswordAuthentication no
    PubkeyAuthentication yes
    ChallengeResponseAuthentication no
    GSSAPIAuthentication no
```

### Multi-Factor Authentication

```yaml
# Multi-factor authentication
ssh:
  mfa:
    enabled: true
    methods:
      - totp
      - u2f
      - yubikey
```

---

## 🛡️ Network Security

### SSH Host Key Verification

```go
// SSH host key verification
type HostKeyManager struct {
    knownHostsFile string
    knownHosts     map[string]ssh.PublicKey
    strictMode     bool
}

func (hkm *HostKeyManager) VerifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
    // Verify host key against known hosts
    if hkm.strictMode {
        if !hkm.isHostKeyKnown(hostname, key) {
            return fmt.Errorf("host key verification failed for %s", hostname)
        }
    }
    
    return nil
}
```

### Network Encryption

```yaml
# Network encryption
ssh:
  encryption:
    enabled: true
    algorithms:
      - aes256-gcm
      - aes128-gcm
      - chacha20-poly1305
    compression: true
    keepalive: true
```

### Firewall Configuration

```bash
# Configure firewall
sudo ufw enable
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

---

## 🔒 Data Security

### State Encryption

```yaml
# State encryption
state:
  encryption:
    enabled: true
    algorithm: "AES-256-GCM"
    key: "{{ vault_state_key }}"
    key_rotation: true
    rotation_interval: 30d
```

### Secrets Management

```yaml
# Secrets management
secrets:
  vault:
    enabled: true
    url: "https://vault.example.com"
    token: "{{ vault_token }}"
    mount_path: "secret"
  
  bitwarden:
    enabled: true
    client_id: "{{ bitwarden_client_id }}"
    client_secret: "{{ bitwarden_client_secret }}"
```

### Data Protection

```go
// Data protection
type DataProtector struct {
    encryptionKey []byte
    hmacKey       []byte
}

func (dp *DataProtector) Encrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(dp.encryptionKey)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}
```

---

## 🔍 Access Control

### Role-Based Access Control

```yaml
# Role-based access control
rbac:
  enabled: true
  roles:
    admin:
      permissions:
        - read
        - write
        - execute
        - rollback
        - manage_users
    
    operator:
      permissions:
        - read
        - execute
    
    viewer:
      permissions:
        - read
```

### User Management

```yaml
# User management
users:
  admin:
    name: "admin"
    email: "admin@example.com"
    roles: ["admin"]
    mfa_enabled: true
  
  operator:
    name: "operator"
    email: "operator@example.com"
    roles: ["operator"]
    mfa_enabled: true
```

### Permission Management

```go
// Permission management
type PermissionManager struct {
    permissions map[string][]string
    roles       map[string][]string
}

func (pm *PermissionManager) HasPermission(user, permission string) bool {
    userRoles := pm.getUserRoles(user)
    for _, role := range userRoles {
        if pm.hasRolePermission(role, permission) {
            return true
        }
    }
    return false
}
```

---

## 📊 Audit and Monitoring

### Audit Logging

```yaml
# Audit logging
audit:
  enabled: true
  log_file: "/var/log/onigirazu-audit.log"
  events:
    - authentication
    - authorization
    - execution
    - state_changes
    - rollback
    - user_management
  
  retention:
    days: 90
    max_size: 100MB
    compression: true
```

### Security Monitoring

```yaml
# Security monitoring
monitoring:
  enabled: true
  metrics:
    - failed_logins
    - suspicious_activity
    - privilege_escalation
    - data_access
  
  alerts:
    - email: "security@example.com"
    - slack: "#security-alerts"
    - webhook: "https://alerts.example.com/webhook"
```

### Log Analysis

```go
// Log analysis
type SecurityAnalyzer struct {
    patterns map[string]*regexp.Regexp
    alerts   []Alert
}

func (sa *SecurityAnalyzer) AnalyzeLogs(logs []LogEntry) []SecurityEvent {
    var events []SecurityEvent
    
    for _, log := range logs {
        for pattern, regex := range sa.patterns {
            if regex.MatchString(log.Message) {
                event := SecurityEvent{
                    Type:    pattern,
                    Log:     log,
                    Severity: sa.getSeverity(pattern),
                }
                events = append(events, event)
            }
        }
    }
    
    return events
}
```

---

## 🔧 Security Configuration

### Security Settings

```yaml
# Security configuration
security:
  # Authentication
  authentication:
    methods:
      - ssh_key
      - password
      - mfa
    password_policy:
      min_length: 12
      require_uppercase: true
      require_lowercase: true
      require_numbers: true
      require_symbols: true
      max_age: 90d
  
  # Authorization
  authorization:
    rbac_enabled: true
    default_role: "viewer"
    admin_users: ["admin"]
  
  # Network security
  network:
    host_key_checking: true
    known_hosts_file: "~/.ssh/known_hosts"
    encryption: true
    compression: true
  
  # Data protection
  data_protection:
    encryption: true
    key_rotation: true
    backup_encryption: true
  
  # Audit
  audit:
    enabled: true
    log_level: "info"
    retention: 90d
```

### Environment Variables

```bash
# Security environment variables
export ONIGIRAZU_SECURITY_ENABLED=true
export ONIGIRAZU_HOST_KEY_CHECKING=true
export ONIGIRAZU_ENCRYPTION_ENABLED=true
export ONIGIRAZU_AUDIT_ENABLED=true
export ONIGIRAZU_RBAC_ENABLED=true
export ONIGIRAZU_MFA_ENABLED=true
```

---

## 🚨 Security Best Practices

### Host Security

```bash
# Secure host configuration
# 1. Update system
sudo apt update && sudo apt upgrade

# 2. Configure firewall
sudo ufw enable
sudo ufw default deny incoming
sudo ufw allow ssh

# 3. Disable unnecessary services
sudo systemctl disable apache2
sudo systemctl disable mysql

# 4. Configure SSH
sudo nano /etc/ssh/sshd_config
# Set: PermitRootLogin no
# Set: PasswordAuthentication no
# Set: PubkeyAuthentication yes

# 5. Restart SSH
sudo systemctl restart ssh
```

### Onigirazu Security

```yaml
# Secure Onigirazu configuration
security:
  # Use strong authentication
  authentication:
    methods: ["ssh_key", "mfa"]
  
  # Enable host key checking
  network:
    host_key_checking: true
  
  # Use encryption
  data_protection:
    encryption: true
  
  # Enable audit logging
  audit:
    enabled: true
  
  # Use RBAC
  authorization:
    rbac_enabled: true
```

### Playbook Security

```yaml
# Secure playbook
---
- name: Secure web server
  hosts: webservers
  become: true
  tasks:
    # 1. Update system
    - name: Update system
      package:
        update_cache: true
    
    # 2. Install security packages
    - name: Install security packages
      package:
        name: "{{ item }}"
        state: present
      loop:
        - fail2ban
        - ufw
        - unattended-upgrades
    
    # 3. Configure firewall
    - name: Configure firewall
      ufw:
        rule: allow
        port: "{{ item }}"
        proto: tcp
      loop:
        - 22
        - 80
        - 443
    
    # 4. Configure fail2ban
    - name: Configure fail2ban
      template:
        src: fail2ban.conf.j2
        dest: /etc/fail2ban/jail.local
      notify: restart fail2ban
    
    # 5. Enable automatic updates
    - name: Enable automatic updates
      template:
        src: unattended-upgrades.conf.j2
        dest: /etc/apt/apt.conf.d/50unattended-upgrades
      notify: restart unattended-upgrades

  handlers:
    - name: restart fail2ban
      service:
        name: fail2ban
        state: restarted
    
    - name: restart unattended-upgrades
      service:
        name: unattended-upgrades
        state: restarted
```

---

## 🔍 Security Testing

### Security Scanning

```bash
# Run security scan
gosec ./...

# Run with specific rules
gosec -include=G401,G402,G403,G404 ./...

# Generate security report
gosec -fmt json -out security.json ./
```

### Vulnerability Assessment

```bash
# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Check dependencies
go mod why github.com/package/name

# Update dependencies
go get -u ./...
go mod tidy
```

### Penetration Testing

```bash
# Test SSH security
nmap -sV -p 22 target_host

# Test for common vulnerabilities
nikto -h target_host

# Test SSL/TLS
sslscan target_host
```

---

## 🚨 Incident Response

### Security Incident Response

```yaml
# Incident response plan
incident_response:
  # Detection
  detection:
    - monitoring
    - alerts
    - log_analysis
  
  # Response
  response:
    - isolate_affected_systems
    - preserve_evidence
    - notify_stakeholders
    - implement_mitigations
  
  # Recovery
  recovery:
    - restore_from_backup
    - patch_vulnerabilities
    - update_security_measures
    - conduct_post_incident_review
```

### Emergency Procedures

```bash
# Emergency response procedures
# 1. Isolate affected systems
sudo ufw deny from affected_ip

# 2. Preserve evidence
sudo cp /var/log/onigirazu-audit.log /backup/incident-$(date +%Y%m%d).log

# 3. Notify stakeholders
# Send alert to security team

# 4. Implement mitigations
# Update firewall rules
# Change passwords
# Rotate keys
```

---

## 📚 Security Resources

### Security Tools

- **gosec** - Go security scanner
- **nancy** - Vulnerability scanner
- **nikto** - Web vulnerability scanner
- **nmap** - Network scanner
- **sslscan** - SSL/TLS scanner

### Security Standards

- **OWASP** - Web application security
- **NIST** - Cybersecurity framework
- **ISO 27001** - Information security management
- **SOC 2** - Security and availability controls

### Security Training

- **Security awareness** - User training
- **Secure coding** - Developer training
- **Incident response** - Response training
- **Penetration testing** - Testing training

---

## 🎯 Security Checklist

### Pre-deployment Security

- [ ] **System updates** - Latest security patches
- [ ] **Firewall configuration** - Proper rules
- [ ] **SSH security** - Key-based authentication
- [ ] **User management** - Proper permissions
- [ ] **Audit logging** - Comprehensive logging

### Runtime Security

- [ ] **Monitoring** - Security monitoring
- [ ] **Access control** - RBAC enabled
- [ ] **Data encryption** - Encrypted storage
- [ ] **Network security** - Encrypted communication
- [ ] **Regular audits** - Security assessments

### Post-deployment Security

- [ ] **Vulnerability scanning** - Regular scans
- [ ] **Security updates** - Patch management
- [ ] **Incident response** - Response procedures
- [ ] **Security training** - User education
- [ ] **Compliance** - Security standards

---

## 📚 Related Documentation

- [Architecture](Architecture) - System architecture
- [Configuration](Configuration) - Security configuration
- [Troubleshooting](Troubleshooting) - Security issues
- [Performance Tuning](Performance-Tuning) - Security performance

---

## 🎯 Summary

### Security Features

- **🔐 Authentication** - Multiple methods
- **🛡️ Authorization** - Role-based access
- **🔒 Data protection** - Encryption
- **📊 Audit logging** - Complete trail
- **🚨 Monitoring** - Security alerts

### Security Benefits

- **🛡️ Defense in depth** - Multiple layers
- **🔍 Principle of least privilege** - Minimal permissions
- **📊 Comprehensive audit** - Complete visibility
- **🔄 Regular updates** - Security patches
- **🚨 Incident response** - Quick response

---

**🔒 Security is built into every aspect of Onigirazu!**

# 🎉 Onigirazu v1.7.0 - Multi-Format Inventory Support

We're excited to announce **Onigirazu v1.7.0** with powerful new inventory management features!

## ✨ What's New

### Multi-Format Inventory Support

Onigirazu now supports **three inventory file formats**, giving you flexibility to choose what works best for your workflow:

#### 1. **YAML Format** (Traditional)

```yaml
hosts:
  web1:
    address: "192.168.1.10"
    port: 22
    user: "deploy"

groups:
  webservers:
    hosts:
      - web1
      - web2
```

#### 2. **TOML Format** (Modern)

```toml
[hosts.web1]
address = "192.168.1.10"
port = 22
user = "deploy"

[groups.webservers]
hosts = ["web1", "web2"]
```

#### 3. **Simple List Format** (Quick & Easy)

```
192.168.1.10
deploy@192.168.1.11:2222
user@192.168.1.12
```

### 🔍 Automatic Inventory Detection

No more `-inventory` flag for standard layouts! Onigirazu now automatically searches for inventory files in your playbook directory:

```bash
# Before
onigirazu -inventory inventory.yml playbook.yml

# Now - just works! ✨
onigirazu playbook.yml
```

Searches for: `inventory.yml`, `inventory.yaml`, `inventory.toml`, `hosts`, `hosts.yml`, `hosts.yaml`, `hosts.toml`, `inventory`

### 🧠 Smart Format Detection

Intelligent format detection based on:

- File extension (`.yml`, `.yaml`, `.toml`, `.txt`)
- Content analysis for ambiguous cases
- Automatic fallback chain: YAML → TOML → Simple list

### 📝 Simple List Parser Features

The new simple list format supports flexible host specifications:

- Plain IP: `192.168.1.10`
- IP with port: `192.168.1.10:2222`
- With username: `user@192.168.1.10`
- Full format: `user@192.168.1.10:2222`
- Automatic defaults (port 22, user "root")

## 📖 Documentation

- **Comprehensive Guide**: [`docs/inventory-formats.md`](docs/inventory-formats.md)
- **TOML Example**: [`inventory.example.toml`](inventory.example.toml)
- **Simple List Example**: [`inventory.example.txt`](inventory.example.txt)

## 🔄 Backward Compatibility

**100% backward compatible!** All existing YAML inventory files work unchanged:

- Auto-detection is optional (only when `-inventory` flag is omitted)
- Existing `-inventory` flag behavior preserved
- No breaking changes to existing workflows

## 💡 Use Cases

### Quick Testing

Use simple list format for rapid prototyping:

```bash
echo "192.168.1.10" > hosts
echo "192.168.1.11" >> hosts
onigirazu playbook.yml
```

### Modern Configuration

Use TOML for better readability and type safety:

```toml
[hosts.production]
address = "prod.example.com"
port = 22
user = "deploy"

[hosts.production.vars]
environment = "production"
debug = false
```

### Traditional Ansible

Continue using YAML format - it just works!

## 🚀 Getting Started

### Installation

**Homebrew (macOS/Linux):**

```bash
brew install onigirazu-cfg/tap/onigirazu
```

**Go Install:**

```bash
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@v1.7.0
```

**Docker:**

```bash
docker pull ghcr.io/onigirazu-cfg/onigirazu:1.7.0
```

### Quick Example

1. Create a simple inventory:

```bash
cat > hosts <<EOF
192.168.1.10
deploy@192.168.1.11:2222
EOF
```

2. Create a playbook:

```yaml
- name: Test connectivity
  hosts: all
  tasks:
    - name: Ping hosts
      module: ping
```

3. Run it:

```bash
onigirazu playbook.yml  # Auto-detects 'hosts' file!
```

## 📊 What's Changed

### Added

- Multi-format inventory support (YAML, TOML, Simple List)
- Automatic inventory file detection
- Smart format detection with fallback
- Simple list parser with flexible address formats
- `github.com/pelletier/go-toml/v2` dependency

### Changed

- Enhanced `EnhancedParser` with inventory parsing
- Updated `main.go` for auto-detection
- Improved error messages and logging

### Documentation

- Created `docs/inventory-formats.md`
- Added example files for all formats
- Migration guide and best practices

## 🙏 Contributors

Thank you to everyone who contributed to this release!

## 📝 Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for complete details.

---

**Enjoy the new features!** 🎊

If you encounter any issues, please [open an issue](https://github.com/onigirazu-cfg/onigirazu/issues).

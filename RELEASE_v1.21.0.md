# Release v1.21.0 - Inventory Plugins Examples

**Release Date:** 2025-01-28
**Type:** Feature Release
**Priority:** HIGH
**Implementation Time:** 2 hours

## 🎯 Overview

This release introduces three fully functional inventory plugin examples for major cloud providers: AWS EC2, Azure Virtual Machines, and Google Cloud Platform Compute Engine. These plugins demonstrate how to create dynamic inventory sources that automatically discover and group hosts from cloud infrastructure.

## ✨ What's New

### 1. AWS EC2 Inventory Plugin

**Location:** `examples/plugins/aws_ec2/plugin.go` (276 lines)

A complete inventory plugin that fetches host information from AWS EC2 instances.

**Features:**

- ✅ Dynamic host discovery from EC2 API
- ✅ Automatic grouping by instance tags
- ✅ Grouping by instance type, availability zone, and security groups
- ✅ Rich host variables (instance_id, instance_type, IPs, tags)
- ✅ Mock data for testing without AWS credentials
- ✅ Configurable filters for instance selection

**Configuration Example:**

```yaml
plugins:
  inventory:
    - name: aws_ec2
      type: inventory
      path: ./plugins/aws_ec2.so
      enabled: true
      config:
        region: us-east-1
        access_key_id: YOUR_ACCESS_KEY
        secret_access_key: YOUR_SECRET_KEY
        filters:
          instance-state-name: running
```

**Host Groups:**

- `tag_Name_*` - Groups by Name tag
- `tag_Environment_*` - Groups by Environment tag
- `type_*` - Groups by instance type (e.g., type_t2_micro)
- `{zone}` - Groups by availability zone (e.g., us-east-1a)
- `sg_*` - Groups by security group

**Host Variables:**

- `instance_id` - EC2 instance ID
- `instance_type` - Instance type (e.g., t2.micro)
- `availability_zone` - AZ where instance runs
- `private_ip` - Private IP address
- `public_ip` - Public IP address (if available)
- `tags` - All instance tags as map

### 2. Azure VM Inventory Plugin

**Location:** `examples/plugins/azure_vm/plugin.go` (293 lines)

A complete inventory plugin that fetches host information from Azure Virtual Machines.

**Features:**

- ✅ Dynamic host discovery from Azure API
- ✅ Automatic grouping by VM tags
- ✅ Grouping by resource group, location, and VM size
- ✅ Rich host variables (vm_id, vm_size, location, IPs, tags)
- ✅ Mock data for testing without Azure credentials
- ✅ Configurable filters for VM selection

**Configuration Example:**

```yaml
plugins:
  inventory:
    - name: azure_vm
      type: inventory
      path: ./plugins/azure_vm.so
      enabled: true
      config:
        subscription_id: YOUR_SUBSCRIPTION_ID
        tenant_id: YOUR_TENANT_ID
        client_id: YOUR_CLIENT_ID
        client_secret: YOUR_CLIENT_SECRET
        resource_group: my-resource-group
        filters:
          powerState: running
```

**Host Groups:**

- `tag_role_*` - Groups by role tag
- `tag_environment_*` - Groups by environment tag
- `rg_{name}` - Groups by resource group
- `location_{region}` - Groups by Azure region
- `size_{type}` - Groups by VM size

**Host Variables:**

- `vm_id` - Azure VM resource ID
- `vm_size` - VM size (e.g., Standard_D2s_v3)
- `location` - Azure region
- `resource_group` - Resource group name
- `private_ip` - Private IP address
- `public_ip` - Public IP address (if available)
- `tags` - All VM tags as map

### 3. GCP Compute Inventory Plugin

**Location:** `examples/plugins/gcp_compute/plugin.go` (324 lines)

A complete inventory plugin that fetches host information from Google Cloud Platform Compute Engine.

**Features:**

- ✅ Dynamic host discovery from GCP Compute API
- ✅ Automatic grouping by instance labels
- ✅ Grouping by zone, machine type, and network tags
- ✅ Rich host variables (instance_id, machine_type, zone, IPs, labels)
- ✅ Mock data for testing without GCP credentials
- ✅ Configurable filters for instance selection

**Configuration Example:**

```yaml
plugins:
  inventory:
    - name: gcp_compute
      type: inventory
      path: ./plugins/gcp_compute.so
      enabled: true
      config:
        project_id: my-gcp-project
        zone: us-central1-a
        credentials_file: /path/to/service-account.json
        filters:
          status: RUNNING
```

**Host Groups:**

- `label_role_*` - Groups by role label
- `label_environment_*` - Groups by environment label
- `{zone}` - Groups by GCP zone (e.g., us-central1-a)
- `machine_type_{type}` - Groups by machine type
- `network_tag_{tag}` - Groups by network tag

**Host Variables:**

- `instance_id` - GCP instance ID
- `machine_type` - Machine type (e.g., e2-medium)
- `zone` - GCP zone
- `project_id` - GCP project ID
- `private_ip` - Private IP address
- `public_ip` - External IP address (if available)
- `labels` - All instance labels as map

### 4. Comprehensive Documentation

**Location:** `docs/INVENTORY_PLUGINS.md` (400+ lines)

Complete guide for inventory plugins including:

- ✅ Architecture overview and plugin interface
- ✅ Detailed documentation for each plugin
- ✅ Configuration examples
- ✅ Building and deployment instructions
- ✅ Step-by-step guide for creating custom plugins
- ✅ Best practices and troubleshooting
- ✅ Integration with inventory manager

## 🏗️ Technical Implementation

### Plugin Structure

Each plugin is organized in its own directory to avoid symbol conflicts:

```
examples/plugins/
├── aws_ec2/
│   └── plugin.go
├── azure_vm/
│   └── plugin.go
└── gcp_compute/
    └── plugin.go
```

This structure allows:

- Independent compilation of each plugin
- No symbol conflicts between plugins
- Clean separation of concerns
- Easy maintenance and updates

### Key Fixes Applied

1. **Package Organization:**
   - Moved each plugin to separate directory
   - Renamed files to `plugin.go` for consistency
   - Each plugin in `package main` for Go plugin system

2. **Type Compatibility:**
   - Fixed `Host.Variables` → `Host.Vars` (correct field name)
   - Fixed `Group.Hosts` type from `[]Host` to `map[string]*Host`
   - Fixed host assignment to use pointers: `group.Hosts[host.Name] = host`

3. **Pointer Handling:**
   - Changed iteration pattern to get pointers:

     ```go
     for i := range hosts {
         host := &hosts[i]
         group.Hosts[host.Name] = host
     }
     ```

4. **Plugin Export:**
   - Added empty `main()` function (required for Go plugins)
   - Exported `Plugin` variable with correct type

### Mock Data

Each plugin includes realistic mock data for testing:

- 4 sample hosts per plugin
- 2 frontend hosts (web/frontend role)
- 1 backend host (api/backend role)
- 1 database host (database role)
- Realistic metadata (IPs, zones, tags/labels)

This allows testing plugin functionality without:

- Real cloud credentials
- Active cloud resources
- API rate limits
- Network connectivity

## 🔨 Building Plugins

Build plugins as shared libraries:

```bash
# AWS EC2 Plugin
go build -buildmode=plugin -o aws_ec2.so examples/plugins/aws_ec2/plugin.go

# Azure VM Plugin
go build -buildmode=plugin -o azure_vm.so examples/plugins/azure_vm/plugin.go

# GCP Compute Plugin
go build -buildmode=plugin -o gcp_compute.so examples/plugins/gcp_compute/plugin.go
```

**Important Notes:**

- Plugins must be built with the same Go version as main application
- Plugins must be built on the same platform (OS/architecture)
- Go plugin system is only supported on Linux and macOS

## 📖 Usage Examples

### 1. Load Plugin from Configuration

```yaml
# plugins.yml
plugins:
  inventory:
    - name: aws_ec2
      type: inventory
      path: ./plugins/aws_ec2.so
      enabled: true
      config:
        region: us-east-1
        filters:
          instance-state-name: running
```

```go
// Load plugins
config, _ := plugins.LoadConfig("plugins.yml")
manager := plugins.NewManager(plugins.NewGoPluginLoader())
plugins.LoadPluginsFromConfig(ctx, manager, config)

// Get inventory plugin
plugin, _ := manager.Get("aws_ec2")
inventoryPlugin := plugin.(plugins.InventoryPlugin)

// Fetch hosts
hosts, _ := inventoryPlugin.GetHosts(ctx, "*")
groups, _ := inventoryPlugin.GetGroups(ctx, "*")
```

### 2. Use with Inventory Manager

```go
// Create inventory manager
invManager := inventory.NewManager()

// Add plugin as source
invManager.AddPlugin(inventoryPlugin)

// Refresh inventory
invManager.Refresh(ctx)

// Get hosts
hosts := invManager.GetHosts("frontend")
```

### 3. Filter Hosts by Pattern

```go
// Get all hosts
allHosts, _ := plugin.GetHosts(ctx, "*")

// Get specific host
host, _ := plugin.GetHosts(ctx, "web-server-1")

// Get hosts matching pattern (future enhancement)
webHosts, _ := plugin.GetHosts(ctx, "web-*")
```

## 🎨 Architecture Decisions

### 1. Separation of Concerns

Each plugin in separate package to avoid conflicts and enable independent compilation.

### 2. Mock Data Inclusion

Include realistic mock data for testing without cloud credentials or resources.

### 3. Caching Support

Plugins support caching with TTL to reduce API calls and improve performance.

### 4. Pattern Matching

Support for filtering hosts and groups by patterns (wildcards, exact match).

### 5. Error Handling

Graceful error handling with informative messages, non-blocking failures.

### 6. Rich Metadata

Provide comprehensive host variables for use in playbooks and templates.

### 7. Flexible Grouping

Multiple grouping strategies (tags, types, zones, etc.) for flexible inventory organization.

## ✅ Testing Results

```bash
# Build test
$ go build ./...
✅ SUCCESS - All packages compile

# Unit tests with race detector
$ go test ./... -race
✅ SUCCESS - All tests pass
✅ 0 race conditions detected

# Plugin compilation test
$ go build -buildmode=plugin examples/plugins/aws_ec2/plugin.go
✅ SUCCESS - AWS EC2 plugin compiles

$ go build -buildmode=plugin examples/plugins/azure_vm/plugin.go
✅ SUCCESS - Azure VM plugin compiles

$ go build -buildmode=plugin examples/plugins/gcp_compute/plugin.go
✅ SUCCESS - GCP Compute plugin compiles
```

## 📊 Code Statistics

**New Files:**

- `examples/plugins/aws_ec2/plugin.go` - 276 lines
- `examples/plugins/azure_vm/plugin.go` - 293 lines
- `examples/plugins/gcp_compute/plugin.go` - 324 lines
- `docs/INVENTORY_PLUGINS.md` - 400+ lines

**Total New Code:** ~1,300 lines

**Deleted Files:**

- `examples/plugins/inventory_aws_ec2.go` (moved)
- `examples/plugins/inventory_azure_vm.go` (moved)
- `examples/plugins/inventory_gcp_compute.go` (moved)

## 🔄 Migration Guide

### From Old Structure

If you had plugins in the old structure:

```
examples/plugins/
├── inventory_aws_ec2.go
├── inventory_azure_vm.go
└── inventory_gcp_compute.go
```

They have been reorganized to:

```
examples/plugins/
├── aws_ec2/
│   └── plugin.go
├── azure_vm/
│   └── plugin.go
└── gcp_compute/
    └── plugin.go
```

**Action Required:**

- Update build scripts to use new paths
- Update plugin configuration paths
- Rebuild plugins with new structure

## 🚀 Future Enhancements

Planned improvements for inventory plugins:

1. **More Cloud Providers:**
   - DigitalOcean Droplets
   - Linode Instances
   - Hetzner Cloud
   - Oracle Cloud

2. **Container Orchestrators:**
   - Kubernetes Pods
   - Docker Swarm Services
   - Nomad Jobs

3. **Configuration Management:**
   - CMDB Integration
   - ServiceNow CMDB
   - Custom CMDB APIs

4. **Advanced Features:**
   - Automatic credential discovery (IAM roles, managed identities)
   - Inventory caching to disk
   - Inventory diff and change detection
   - Parallel inventory refresh
   - Real-time updates via webhooks

5. **Performance:**
   - Concurrent API calls
   - Intelligent caching strategies
   - Rate limit handling
   - Pagination optimization

## 📚 Documentation

- **Inventory Plugins Guide:** `docs/INVENTORY_PLUGINS.md`
- **Plugin Integration Guide:** `docs/PLUGIN_INTEGRATION.md`
- **Implementation Progress:** `IMPLEMENTATION_PROGRESS.md`

## 🔗 Related Releases

- **v1.20.0** - Plugin System in Main Application
- **v1.19.0** - Plugin Integration with Core Engine
- **v1.18.2** - Race Conditions Fix and Documentation Update

## 🎯 Next Steps

Following the implementation plan (Варіант 2):

1. ✅ **Step 1:** Plugin support in main.go (v1.20.0)
2. ✅ **Step 2:** Inventory plugin examples (v1.21.0) ← **Current Release**
3. ⏭️ **Step 3:** Template caching (Фаза 6, +20-30% performance)
4. ⏭️ **Step 4:** Improve test coverage for packages without tests

## 🙏 Acknowledgments

This release completes Step 2 of the plugin system implementation plan, providing practical examples of how to extend Onigirazu with dynamic inventory sources from major cloud providers.

---

**Full Changelog:** v1.20.0...v1.21.0
**Download:** [GitHub Releases](https://github.com/onigirazu-cfg/onigirazu/releases/tag/v1.21.0)

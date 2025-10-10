# Inventory Plugins Examples

This directory contains example inventory plugins for popular cloud providers.

## Available Inventory Plugins

### 1. AWS EC2 Inventory Plugin (`inventory_aws_ec2.go`)

Provides dynamic inventory from AWS EC2 instances.

**Features:**

- Automatic discovery of EC2 instances
- Grouping by tags (Role, Environment)
- Caching with configurable TTL
- Filter support for instance selection

**Configuration Example:**

```yaml
plugins_dir: ./plugins
plugins:
  - name: aws_ec2
    type: inventory
    enabled: true
    config:
      region: us-east-1
      access_key_id: YOUR_ACCESS_KEY
      secret_access_key: YOUR_SECRET_KEY
      cache_ttl: 300  # 5 minutes
      filters:
        tag:Environment: production
        instance-state-name: running
```

**Real Implementation Notes:**

To use this plugin with real AWS EC2 instances, you need to:

1. Install AWS SDK:

   ```bash
   go get github.com/aws/aws-sdk-go-v2/service/ec2
   go get github.com/aws/aws-sdk-go-v2/config
   ```

2. Uncomment the AWS SDK code in the `Refresh()` method
3. Implement proper AWS authentication (IAM roles, credentials file, etc.)
4. Add error handling and retry logic

**Groups Created:**

- `web` - Instances with Role=web tag
- `database` - Instances with Role=database tag
- `production` - Instances with Environment=production tag

---

### 2. Azure VM Inventory Plugin (`inventory_azure_vm.go`)

Provides dynamic inventory from Azure Virtual Machines.

**Features:**

- Automatic discovery of Azure VMs
- Grouping by tags (Tier, Environment, Location)
- Resource group filtering
- Caching with configurable TTL

**Configuration Example:**

```yaml
plugins_dir: ./plugins
plugins:
  - name: azure_vm
    type: inventory
    enabled: true
    config:
      subscription_id: YOUR_SUBSCRIPTION_ID
      tenant_id: YOUR_TENANT_ID
      client_id: YOUR_CLIENT_ID
      client_secret: YOUR_CLIENT_SECRET
      resource_group: my-resource-group
      cache_ttl: 300  # 5 minutes
```

**Real Implementation Notes:**

To use this plugin with real Azure VMs, you need to:

1. Install Azure SDK:

   ```bash
   go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute
   go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
   ```

2. Uncomment the Azure SDK code in the `Refresh()` method
3. Set up Azure Service Principal for authentication
4. Add error handling and pagination support

**Groups Created:**

- `application` - VMs with Tier=application tag
- `database` - VMs with Tier=database tag
- `eastus` - VMs in East US location
- `production` - VMs with Environment=production tag

---

### 3. GCP Compute Inventory Plugin (`inventory_gcp_compute.go`)

Provides dynamic inventory from GCP Compute Engine instances.

**Features:**

- Automatic discovery of Compute Engine instances
- Grouping by labels (role, environment, zone)
- Zone-based filtering
- Caching with configurable TTL

**Configuration Example:**

```yaml
plugins_dir: ./plugins
plugins:
  - name: gcp_compute
    type: inventory
    enabled: true
    config:
      project_id: my-gcp-project
      zone: us-central1-a
      credentials_file: /path/to/service-account.json
      cache_ttl: 300  # 5 minutes
      filters:
        labels.environment: production
        status: RUNNING
```

**Real Implementation Notes:**

To use this plugin with real GCP Compute instances, you need to:

1. Install GCP SDK:

   ```bash
   go get cloud.google.com/go/compute/apiv1
   go get google.golang.org/api/iterator
   ```

2. Uncomment the GCP SDK code in the `Refresh()` method
3. Set up GCP Service Account and download credentials JSON
4. Add error handling and pagination support

**Groups Created:**

- `frontend` - Instances with role=frontend label
- `backend` - Instances with role=backend label
- `database` - Instances with role=database label
- `production` - Instances with environment=production label
- `{zone}` - Instances in specific zone (e.g., `us-central1-a`)

---

## Usage in Playbooks

Once configured, you can use these inventory plugins in your playbooks:

### Example 1: Target AWS EC2 Web Servers

```yaml
---
- name: Configure web servers
  hosts: web  # Group from AWS EC2 plugin
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
```

### Example 2: Target Azure VMs by Location

```yaml
---
- name: Update VMs in East US
  hosts: eastus  # Group from Azure VM plugin
  tasks:
    - name: Update packages
      package:
        name: "*"
        state: latest
```

### Example 3: Target GCP Frontend Instances

```yaml
---
- name: Deploy frontend application
  hosts: frontend  # Group from GCP Compute plugin
  tasks:
    - name: Copy application files
      copy:
        src: /local/app/
        dest: /opt/app/
```

---

## Building Plugins

These plugins are written as Go plugins and need to be compiled:

### Build AWS EC2 Plugin

```bash
go build -buildmode=plugin -o inventory_aws_ec2.so inventory_aws_ec2.go
```

### Build Azure VM Plugin

```bash
go build -buildmode=plugin -o inventory_azure_vm.so inventory_azure_vm.go
```

### Build GCP Compute Plugin

```bash
go build -buildmode=plugin -o inventory_gcp_compute.so inventory_gcp_compute.go
```

---

## Plugin Configuration File

Create a `plugins.yml` file to enable inventory plugins:

```yaml
plugins_dir: ./plugins
plugins:
  # AWS EC2 Inventory
  - name: aws_ec2
    type: inventory
    path: inventory_aws_ec2.so
    enabled: true
    config:
      region: us-east-1
      cache_ttl: 300

  # Azure VM Inventory
  - name: azure_vm
    type: inventory
    path: inventory_azure_vm.so
    enabled: true
    config:
      subscription_id: YOUR_SUBSCRIPTION_ID
      tenant_id: YOUR_TENANT_ID
      resource_group: my-rg
      cache_ttl: 300

  # GCP Compute Inventory
  - name: gcp_compute
    type: inventory
    path: inventory_gcp_compute.so
    enabled: true
    config:
      project_id: my-project
      zone: us-central1-a
      cache_ttl: 300
```

---

## Architecture

### Plugin Interface

All inventory plugins implement the `InventoryPlugin` interface:

```go
type InventoryPlugin interface {
    Plugin
    GetHosts(ctx context.Context, pattern string) ([]types.Host, error)
    GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error)
    Refresh(ctx context.Context) error
    GetCacheTTL() time.Duration
}
```

### Caching Strategy

- Inventory data is cached in memory
- Cache TTL is configurable (default: 5 minutes)
- Automatic refresh when cache expires
- Manual refresh via `Refresh()` method

### Error Handling

- Non-blocking errors (logged as warnings)
- Graceful fallback to empty inventory
- Retry logic for transient failures (in real implementations)

---

## Security Considerations

### Credentials Management

**DO NOT** hardcode credentials in configuration files!

**Best Practices:**

1. **AWS:** Use IAM roles or AWS credentials file

   ```yaml
   config:
     region: us-east-1
     # Credentials from ~/.aws/credentials or IAM role
   ```

2. **Azure:** Use Managed Identity or environment variables

   ```yaml
   config:
     subscription_id: YOUR_SUBSCRIPTION_ID
     # Credentials from environment or managed identity
   ```

3. **GCP:** Use Application Default Credentials

   ```yaml
   config:
     project_id: my-project
     # Credentials from GOOGLE_APPLICATION_CREDENTIALS env var
   ```

### Network Security

- Use private IPs for internal communication
- Configure security groups/firewall rules
- Use SSH key authentication (not passwords)
- Enable SSH host key verification

---

## Performance Optimization

### Caching

- Adjust `cache_ttl` based on your infrastructure change frequency
- Longer TTL = fewer API calls = better performance
- Shorter TTL = more up-to-date inventory = higher API costs

### Filtering

- Use filters to reduce the number of instances queried
- Filter by tags, state, region, etc.
- Reduces API response size and processing time

### Pagination

- Implement pagination for large inventories (>100 instances)
- Process instances in batches
- Reduces memory usage

---

## Troubleshooting

### Plugin Not Loading

**Problem:** Plugin fails to load

**Solutions:**

1. Check plugin path in `plugins.yml`
2. Verify plugin is compiled with correct Go version
3. Check plugin exports `Plugin` variable
4. Review logs for error messages

### Empty Inventory

**Problem:** Plugin loads but returns no hosts

**Solutions:**

1. Check cloud provider credentials
2. Verify filters are not too restrictive
3. Check network connectivity to cloud API
4. Review API permissions (IAM, RBAC, etc.)

### Slow Performance

**Problem:** Inventory refresh is slow

**Solutions:**

1. Increase `cache_ttl` to reduce API calls
2. Add filters to reduce query scope
3. Use pagination for large inventories
4. Consider using multiple plugins for different regions

---

## Contributing

To add a new inventory plugin:

1. Implement the `InventoryPlugin` interface
2. Extend `BaseInventoryPlugin` for common functionality
3. Add configuration parsing in `Initialize()`
4. Implement `GetHosts()`, `GetGroups()`, and `Refresh()`
5. Add caching logic
6. Write documentation
7. Add example configuration

---

## License

These examples are provided as-is for demonstration purposes.
Modify and use them according to your needs.

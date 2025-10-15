# Vagrant Testing Environment for Onigirazu

This directory contains configuration for local testing of Onigirazu against multiple operating systems using Vagrant.

## Prerequisites

- [Vagrant](https://www.vagrantup.com/downloads) (>= 2.3.0)
- [VirtualBox](https://www.virtualbox.org/wiki/Downloads) (>= 6.1)
- At least 8GB of free RAM (for running multiple VMs)
- At least 20GB of free disk space

## Available VMs

### Ubuntu
- **ubuntu2004** - Ubuntu 20.04 LTS (Focal) - `192.168.56.10`
- **ubuntu2204** - Ubuntu 22.04 LTS (Jammy) - `192.168.56.11`
- **ubuntu2404** - Ubuntu 24.04 LTS (Noble) - `192.168.56.12`

### Debian
- **debian11** - Debian 11 (Bullseye) - `192.168.56.20`
- **debian12** - Debian 12 (Bookworm) - `192.168.56.21`

### Red Hat Family
- **centos7** - CentOS 7 - `192.168.56.30`
- **rocky8** - Rocky Linux 8 - `192.168.56.31`
- **rocky9** - Rocky Linux 9 - `192.168.56.32`

### SUSE
- **opensuse15** - openSUSE Leap 15.5 - `192.168.56.40`

### BSD
- **freebsd13** - FreeBSD 13 - `192.168.56.50`
- **freebsd14** - FreeBSD 14 - `192.168.56.51`

## Quick Start

### 1. Start a Single VM

```bash
# Using Makefile
make vagrant-up
# Enter VM name when prompted (e.g., ubuntu2204)

# Or directly with Vagrant
vagrant up ubuntu2204
```

### 2. Start All VMs

```bash
# Using Makefile (recommended)
make vagrant-up-all

# Or directly with Vagrant
vagrant up
```

**Note:** Starting all VMs requires significant resources. Consider starting only the VMs you need.

### 3. Check VM Status

```bash
make vagrant-status
# or
vagrant status
```

### 4. Test Connectivity

```bash
# Test a specific group
make vagrant-test
# Enter group name when prompted (ubuntu, debian, redhat, suse, bsd, linux, all)

# Run comprehensive tests on all running VMs
make vagrant-test-all
```

## Usage Examples

### Basic Operations

```bash
# SSH into a VM
make vagrant-ssh
# or
vagrant ssh ubuntu2204

# Stop a VM
make vagrant-halt
# or
vagrant halt ubuntu2204

# Stop all VMs
make vagrant-halt-all

# Destroy a VM
make vagrant-destroy
# or
vagrant destroy -f ubuntu2204

# Destroy all VMs
make vagrant-destroy-all
```

### Testing Onigirazu

```bash
# Build Onigirazu first
make build

# Test ping module on Ubuntu hosts
./bin/onigirazu run -i vagrant/inventory.ini -m ping --limit ubuntu

# Test command module on all Linux hosts
./bin/onigirazu run -i vagrant/inventory.ini -m command -a "uname -a" --limit linux

# Test with specific user
./bin/onigirazu run -i vagrant/inventory.ini -m ping --limit all -u vagrant

# Run a playbook
./bin/onigirazu playbook -i vagrant/inventory.ini vagrant/test-playbook.yml
```

### Testing Different Authentication Methods

```bash
# Test with SSH key (default vagrant user)
./bin/onigirazu run -i vagrant/inventory.ini -m ping --limit ubuntu

# Test with password authentication (testuser)
./bin/onigirazu run -i vagrant/inventory.ini -m ping --limit testuser -u testuser

# Test with custom SSH key
./bin/onigirazu run -i vagrant/inventory.ini -m ping --limit all -k ~/.ssh/custom_key
```

## Inventory Groups

The `inventory.ini` file defines the following groups:

- **ubuntu** - All Ubuntu VMs
- **debian** - All Debian VMs
- **redhat** - All Red Hat family VMs (CentOS, Rocky)
- **suse** - All SUSE VMs
- **bsd** - All BSD VMs
- **linux** - All Linux VMs (ubuntu + debian + redhat + suse)
- **all** - All VMs
- **testuser** - All VMs with testuser credentials (password auth)

## VM Configuration

Each VM is configured with:

- **Memory:** 512MB
- **CPUs:** 1
- **Network:** Private network with static IP
- **Users:**
  - `vagrant` - Default user with SSH key authentication
  - `testuser` - Test user with password authentication (password: `testpass`)
- **Software:** Python3, sudo

## Troubleshooting

### VM Won't Start

```bash
# Check VirtualBox status
VBoxManage list vms

# Check Vagrant status
vagrant global-status

# Prune invalid entries
vagrant global-status --prune

# Destroy and recreate
vagrant destroy -f <vm_name>
vagrant up <vm_name>
```

### SSH Connection Issues

```bash
# Verify VM is running
vagrant status

# Check SSH configuration
vagrant ssh-config <vm_name>

# Test SSH manually
ssh -i ~/.vagrant.d/insecure_private_key vagrant@192.168.56.10
```

### Network Issues

```bash
# Check VirtualBox host-only networks
VBoxManage list hostonlyifs

# Recreate network if needed
VBoxManage hostonlyif remove vboxnet0
vagrant up
```

### Performance Issues

If VMs are slow:

1. Increase memory in `Vagrantfile`:
   ```ruby
   vb.memory = "1024"  # Instead of 512
   ```

2. Run fewer VMs simultaneously

3. Use SSD for VM storage

4. Enable hardware virtualization in BIOS

## Advanced Usage

### Custom VM Configuration

Edit the `Vagrantfile` to customize:

- Memory and CPU allocation
- Network configuration
- Provisioning scripts
- Synced folders

### Adding New VMs

Add a new entry to the `machines` hash in `Vagrantfile`:

```ruby
"myvm" => {
  box: "generic/ubuntu2204",
  ip: "192.168.56.99",
  hostname: "myvm.test"
}
```

Then add it to `inventory.ini`:

```ini
[mygroup]
myvm ansible_host=192.168.56.99
```

### Running Specific Tests

```bash
# Test only file operations
./bin/onigirazu run -i vagrant/inventory.ini -m file \
  -a "path=/tmp/test state=directory" --limit linux

# Test package installation
./bin/onigirazu run -i vagrant/inventory.ini -m apt \
  -a "name=curl state=present" --limit ubuntu --become

# Test with verbose output
./bin/onigirazu run -i vagrant/inventory.ini -m ping \
  --limit all -v
```

## Resource Management

### Minimal Setup (2-3 VMs)

For basic testing with limited resources:

```bash
vagrant up ubuntu2204 debian12 rocky9
```

### Full Setup (All VMs)

For comprehensive testing:

```bash
make vagrant-up-all
```

### Cleanup

When done testing:

```bash
# Stop all VMs (preserves state)
make vagrant-halt-all

# Destroy all VMs (frees disk space)
make vagrant-destroy-all
```

## CI/CD Integration

You can integrate Vagrant tests into your CI/CD pipeline:

```bash
# In your CI script
make vagrant-up-all
make vagrant-test-all
make vagrant-destroy-all
```

**Note:** This requires CI runners with nested virtualization support.

## Tips

1. **Start small:** Begin with 1-2 VMs to verify setup
2. **Use snapshots:** Take VM snapshots before destructive tests
3. **Parallel testing:** Run tests on different VM groups in parallel
4. **Resource monitoring:** Monitor host system resources
5. **Regular cleanup:** Destroy unused VMs to free resources

## Support

For issues or questions:

- Check [Vagrant documentation](https://www.vagrantup.com/docs)
- Check [VirtualBox documentation](https://www.virtualbox.org/manual/)
- Open an issue in the Onigirazu repository

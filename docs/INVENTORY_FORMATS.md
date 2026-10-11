# Inventory Formats

An inventory lists the hosts and groups a playbook or ad-hoc command runs on. Pass it
with `-i`; `-i` can be repeated and can point to a file, a directory, an executable
script or a list of hosts.

| Format | How it is recognised |
|--------|----------------------|
| YAML (Onigirazu `groups:` or Ansible `all:`) | `.yml`, `.yaml` |
| JSON | `.json` |
| TOML | `.toml` |
| INI (Ansible style) | `.ini` |
| Plain host list | other files whose lines are `[user@]host[:port]` |
| Host list on the command line | `-i host1,user@host2:2222` or `-i host1,` (a comma, not a file) |
| Dynamic inventory | executable file |

A file with another or no extension is detected from its content: executable script,
plain list, then JSON, YAML, TOML and INI are tried in that order.

## YAML

```yaml
groups:
  webservers:
    hosts:
      web1:
        onigirazu_host: 192.168.1.1
        onigirazu_port: 22
        onigirazu_user: admin
        app_port: 8080          # any other key is a host variable
      web2:
        onigirazu_host: 192.168.1.2
    vars:
      http_port: 80
  production:
    children:
      - webservers
    vars:
      environment: production
```

A YAML file with a top-level `all:` key is read as an Ansible inventory, see
[ANSIBLE_INVENTORY_QUICK_START.md](ANSIBLE_INVENTORY_QUICK_START.md).

Hosts run in the order Ansible uses: as they first appear in the file, `all`'s own hosts, then each child group in turn, depth first. `serial` batches and `run_once` follow that order.

## JSON

```json
{
  "hosts": [
    {"name": "web1", "address": "192.168.1.1", "port": 22, "user": "admin",
     "key_file": "/home/admin/.ssh/id_rsa", "vars": {"app_port": 8080}},
    {"name": "db1", "address": "192.168.1.10", "user": "postgres"}
  ],
  "groups": {
    "webservers": {"name": "webservers", "hosts": {"web1": {}}, "vars": {"http_port": 80}, "children": []},
    "databases":  {"name": "databases",  "hosts": {"db1": {}},  "vars": {"db_port": 5432}, "children": []}
  }
}
```

## TOML

```toml
[hosts.web1]
address = "192.168.1.1"
user = "admin"
key_file = "/home/admin/.ssh/id_rsa"
vars = { app_port = 8080 }

[hosts.db1]
address = "192.168.1.10"
user = "postgres"

[groups.webservers]
hosts = ["web1"]
vars = { http_port = 80 }

[groups.databases]
hosts = ["db1"]

[groups.production]
children = ["webservers", "databases"]
vars = { environment = "production" }
```

## INI

```ini
# comments start with # or ;
[webservers]
web1 ansible_host=192.168.1.1 ansible_user=admin app_port=8080
web2 ansible_host=192.168.1.2 ansible_port=2222

[databases]
db1 ansible_host=192.168.1.10 ansible_user=postgres

[webservers:vars]
http_port=80

[production:children]
webservers
databases

[production:vars]
environment=production
```

Hosts before the first section are in the group `ungrouped`, as in Ansible. A group may be
empty (`[pending]`, or `pending: {hosts: }` in YAML).

## Plain host list

One `[user@]host[:port]` per line; `#` starts a comment. Hosts get port 22 and the local user
unless given and belong to the group `all`. A host is named by its address; when one
address appears with several ports, as `address:port`.

```
192.168.1.1
deploy@192.168.1.2:2222
web3.example.com
```

## Host list on the command line

As in Ansible, a `-i` value with a comma that is not a file is a list of hosts in the
same `[user@]host[:port]` form: `-i web1,web2`, `-i deploy@10.0.0.5:2222,` (a single host
needs the trailing comma). `-i a.yml,b.yml` loads both files when both exist.

## Dynamic inventory

An executable file (`chmod +x`), run with `--list` (30 s timeout, output cached for 10
minutes). It prints either an Ansible inventory script's JSON:

```json
{
  "web": {"hosts": ["web1", "web2"], "vars": {"tier": "front"}, "children": []},
  "db": ["db1"],
  "_meta": {"hostvars": {"web1": {"ansible_host": "192.168.1.1", "ansible_user": "admin"}}}
}
```

or Onigirazu's own JSON inventory:

```bash
#!/bin/bash
cat <<'EOF'
{
  "hosts": [{"name": "web1", "address": "192.168.1.1", "user": "admin"}],
  "groups": {"webservers": {"name": "webservers", "hosts": {"web1": {}}, "vars": {}}}
}
EOF
```

Existing Ansible inventory scripts work unchanged. As in Ansible, a host that is only in
`_meta.hostvars` is not added.

## Inventory plugins

A YAML inventory whose top-level `plugin` names a source asks that system for the hosts instead of
listing them. The result is read like a dynamic script's output: hosts with `ansible_host` and
`<plugin>_*` variables, groups by the fields you choose. Secrets are given as `${VAR}` or
`env:VAR`, never pasted into the file. Several sources combine as any inventories do (`-i a -i b`).

### netbox

```yaml
plugin: netbox
url: https://netbox.example.com
token: ${NETBOX_TOKEN}            # default: $NETBOX_TOKEN
filters: {site: fra1, role: server, tag: managed, tenant: ops, status: active, platform: ubuntu}   # the six keys accepted
devices: true                     # /api/dcim/devices/
virtual_machines: true            # /api/virtualization/virtual-machines/
group_by: [site, role, tags, type, platform, tenant]   # default: site, role, tags
ansible_host: primary_ip          # or name
```

Groups: `site_<slug>`, `role_<slug>`, `tag_<slug>`, `device` / `virtual_machine`, `platform_<slug>`,
`tenant_<slug>`. Host variables: `netbox_site`, `netbox_role`, `netbox_platform`, `netbox_tenant`,
`netbox_status`, `netbox_tags` (list), `netbox_primary_ip4`, `netbox_primary_ip6`, `netbox_id`,
`netbox_custom_fields`, `netbox_device_type` and `netbox_serial` (devices), `netbox_cluster` (VMs),
`netbox_type`. `ansible_host` is the primary IPv4 (else IPv6) without its prefix; a host without one
keeps its name as the address. Only `status: active` objects unless `filters.status` says otherwise.

### vsphere

```yaml
plugin: vsphere
url: https://vcenter.example.com
user: inventory@vsphere.local
password: ${VSPHERE_PASSWORD}     # default: $VSPHERE_PASSWORD
insecure: true                    # self-signed certificate
folders: [Production, Staging]    # VM folder names; default: every folder
powered_on: true                  # default: only running VMs
group_by: [folder, power_state, guest_family]   # default: folder, power_state
```

vCenter REST API (a session from the credentials, closed afterwards). Groups: `folder_<name>`,
`power_state_powered_on`, `guest_family_linux`. Host variables: `vsphere_id`, `vsphere_folder`,
`vsphere_power_state`, `vsphere_cpu_count`, `vsphere_memory_mib`, `vsphere_guest_host_name`,
`vsphere_guest_family`, `vsphere_guest_os`, `vsphere_ip_addresses`. `ansible_host` is the guest's
first IPv4 that is not loopback, link-local or a docker bridge (VMware Tools must run); the VM's
name is the host name.

### proxmox

```yaml
plugin: proxmox
url: https://pve.example.com:8006
user: inventory@pve
token_id: onigirazu
token_secret: ${PROXMOX_TOKEN_SECRET}   # default: $PROXMOX_TOKEN_SECRET
insecure: true
running: true                     # default: only running guests
templates: false                  # default: templates left out
agent: true                       # default: addresses from the QEMU guest agent / container interfaces
group_by: [node, type, status, tags, pool]   # default: node, type, tags
```

An API token (`user@realm!token_id`) with `VM.Audit` on the guests (and `VM.Monitor` for the agent
query). Groups: `node_<name>`, `type_qemu` / `type_lxc`, `status_<s>`, `tag_<t>`, `pool_<p>`. Host
variables: `proxmox_vmid`, `proxmox_node`, `proxmox_type`, `proxmox_status`, `proxmox_tags`,
`proxmox_pool`, `proxmox_ip_addresses`. `ansible_host` is the first usable IPv4 the guest agent
(QEMU) or the container reports; without one the name is the address.

### netbird

```yaml
plugin: netbird
url: https://api.netbird.io       # default; or the self-hosted management API
token: ${NETBIRD_TOKEN}           # a personal access token; default: $NETBIRD_TOKEN
connected: true                   # default: only peers online now
groups: [servers]                 # only peers in one of these NetBird groups; default: all
group_by: [groups, os]            # default: groups
```

Hosts are the peers, named by their DNS label, with `ansible_host` the NetBird address — the
control machine reaches them over the mesh. Groups: `netbird_<group>` per NetBird group,
`os_<linux|darwin|windows>`. Host variables: `netbird_id`, `netbird_ip`, `netbird_hostname`,
`netbird_dns_label`, `netbird_connected`, `netbird_os`, `netbird_version`, `netbird_groups`,
`netbird_ssh_enabled`, `netbird_last_seen`.

### terraform

```yaml
plugin: terraform
project: ../infra                 # a Terraform/OpenTofu project: `terraform show -json` runs there (remote backends work)
# state: terraform.tfstate        # or a state file, relative to this file (default when no project)
binary: terraform                 # default; tofu for OpenTofu
workspace: prod                   # optional, selected first
hosts:                            # which resources are hosts; default: the known machine types below
  - type: vsphere_virtual_machine
    name: "{{ name }}"            # attribute paths of the resource, dotted, "a|b" = first non-empty
    address: "{{ default_ip_address|guest_ip_addresses.0 }}"
    groups: ["vsphere", "{{ folder }}"]
    vars: {ansible_user: ubuntu}
group_by: [type, module]          # default: tf_<type> and module_<name>
```

Hosts are the managed resources of the machine types the plugin knows — `vsphere_virtual_machine`,
`aws_instance`, `google_compute_instance`, `azurerm_linux/windows_virtual_machine`, `hcloud_server`,
`digitalocean_droplet`, `linode_instance`, `vultr_instance`, `openstack_compute_instance_v2`,
`proxmox_vm_qemu`, `proxmox_virtual_environment_vm`, `libvirt_domain`, `scaleway_instance_server`,
`exoscale_compute_instance`, `upcloud_server` — or those listed under `hosts`; a listed known type
keeps its default name and address paths. Resources of the `ansible/ansible` provider
(`ansible_host`, `ansible_group`) are read as the `cloud.terraform` collection reads them. A CIDR
address loses its length; cloud-init's `dhcp` is no address (the host name is used then). Host
variables: `terraform_type`, `terraform_address`, `terraform_module`, `terraform` (every attribute),
plus the mapping's `vars`. `lookup('cloud.terraform.tf_output', 'name', project_path='../infra')`
(or `state_file=`) reads an output; without a name, all outputs as a dict.

A YAML inventory may also start with the groups themselves (no `all:` wrapper), as Ansible
reads it. A host whose address is its own name (`web1:` with no `ansible_host`) takes `HostName`,
`Port`, `User`, `IdentityFile` and `ProxyJump` from its `~/.ssh/config` stanza, where the
inventory sets nothing.

## Host variables

| Variable | Meaning | Default |
|----------|---------|---------|
| `onigirazu_host` / `ansible_host` (`address` in JSON/TOML) | address to connect to | host name |
| `onigirazu_port` / `ansible_port` (`port`) | SSH port | 22 |
| `onigirazu_user` / `ansible_user` (`user`) | SSH user | the local `$USER`, as in Ansible |
| `onigirazu_ssh_private_key_file` / `ansible_ssh_private_key_file` (`key_file`) | private key | — |
| `onigirazu_password` / `ansible_password` (`password`) | SSH password | — |
| `onigirazu_become_password` / `ansible_become_password` (`ansible_become_pass`) | sudo password for `become` (passed on stdin, never on a command line) | — (`sudo -n`) |
| `ansible_ssh_common_args`, `ansible_ssh_extra_args` | `-o ConnectTimeout=N`; `-J`/`-o ProxyJump=[user@]host[:port][,...]` (aliases from `~/.ssh/config` give HostName, User, Port, IdentityFile); `-o ProxyCommand=...` (`%h`, `%p`, `%r`); `-o StrictHostKeyChecking=no` skips the host key check; other options are ignored | — |
| `onigirazu_connection` / `ansible_connection` | `ssh`; `local` (this machine); `docker` or `podman` (also `community.docker.docker`, `containers.podman.podman`): commands run with `docker exec` in the container named by `ansible_host`, else the host name, as `ansible_user` if set; no SSH in the container; `winrm`: a Windows host (win_* modules); `ssh` with `ansible_shell_type: powershell` (or `cmd`) is a Windows host over OpenSSH | ssh |
| `ansible_winrm_transport`, `ansible_winrm_scheme`, `ansible_winrm_message_encryption`, `ansible_winrm_server_cert_validation`, `ansible_winrm_ca_trust_path`, `ansible_winrm_read_timeout_sec` | WinRM: `ntlm` (default) or `basic`; `https` unless the port is 5985 (default port 5986); encryption `auto` (NTLM over http), `always`, `never`; `ignore` skips the certificate check | — |
| any other key | host variable for templates | — |

These variables also work as group variables (`all: vars: ansible_user: deploy`, group_vars
files): a host takes them unless it sets its own.

`apply -u USER` and `--private-key FILE` (`run -u`/`-k`) override user and key for every host.
As in Ansible, these variables given with `-e` override the inventory for every host
(`-e ansible_user=ansible`, `-e @bootstrap.yml` with `ansible_password` and
`ansible_become_password`).

## group_vars and host_vars

As in Ansible, variables are also read from `group_vars/<group>.yml` and
`host_vars/<host>.yml` (`.yaml`, `.json`, no extension, or a directory of such files
merged in name order). They are looked up next to each inventory file, inside an
inventory directory, and, for `apply`/`plan`/`drift`, next to the playbook (read last,
so they win). They override variables written in the inventory itself.

## Several sources and directories

- `-i a.yml -i b.ini` (or `-i a.yml,b.ini`): sources are merged; for the same host or
  group the later source wins.
- `-i inventory/`: every `.yml`, `.yaml`, `.json`, `.ini` and `.toml` file (alphabetically)
  and then every executable in the directory tree is loaded. Other files, hidden
  directories and `group_vars/`/`host_vars/` are skipped.

## Without -i

`ANSIBLE_INVENTORY` (a comma separated list of sources) is the default of `-i`. Without it,
`run` requires `-i`. `apply` (and `plan`/`drift`) look in the playbook's directory for
`inventory.yml`, `inventory.yaml`, `inventory.toml`, `inventory.json`, `inventory.ini`,
`hosts`, `hosts.yml`, `hosts.yaml`, `hosts.toml`, `hosts.json`, `hosts.ini`, `inventory`,
then in `/etc/onigirazu/` for `inventory.yml`, `hosts.yml`, `inventory`. When none is found,
the inventory is `localhost` alone, run on the control machine.

## Limitations

- `onigirazu inventory` takes a single `-i`.

## Inspecting an inventory

```bash
onigirazu inventory --list -i inventory.yml       # hosts and groups
onigirazu inventory --graph -i inventory.yml      # group tree
onigirazu inventory --host web1 -i inventory.yml  # groups of one host
onigirazu inventory --list --json -i inventory.yml   # as ansible-inventory --list
onigirazu inventory --host web1 --json -i inventory.yml  # as ansible-inventory --host
```

`--json` prints what `ansible-inventory` prints: groups with their own hosts and children,
`all` with the top groups and `ungrouped`, `_meta.hostvars` with each host's variables (group
variables resolved, templated connection variables rendered). Passwords are left out. Messages
go to stderr, so the output can be piped to a script.

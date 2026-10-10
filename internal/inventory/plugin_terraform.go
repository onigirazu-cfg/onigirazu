package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Terraform / OpenTofu: the machines a state holds become hosts.
//
//	plugin: terraform
//	state: terraform.tfstate            # a state file (relative to this file), or
//	project: ../infra                   # a project directory: `terraform show -json` runs there
//	                                    # (remote backends work); default: terraform.tfstate here
//	binary: terraform                   # default terraform; tofu for OpenTofu
//	workspace: prod                     # optional; selected before show
//	hosts:                              # which resources are hosts and how to read them;
//	  - type: vsphere_virtual_machine   # default: the known machine types below
//	    name: "{{ name }}"              # attribute paths of the resource; "a|b" = first non-empty
//	    address: "{{ default_ip_address }}"
//	    groups: ["vsphere", "{{ folder }}"]
//	    vars: {ansible_user: ubuntu}
//	group_by: [type, module]            # default: type (tf_<type>) and module
//
// Resources of the ansible/ansible provider (`ansible_host`, `ansible_group`)
// are read as the cloud.terraform collection reads them: name, groups,
// variables. Host variables: terraform_type, terraform_address,
// terraform_module, terraform (every attribute of the resource), plus the
// mapping's vars; ansible_host from `address`.

type terraformHostRule struct {
	Type    string
	Name    string
	Address string
	Groups  []string
	Vars    map[string]interface{}
}

// defaultTerraformHosts are the machine resources of common providers
var defaultTerraformHosts = []terraformHostRule{
	{Type: "vsphere_virtual_machine", Name: "{{ name }}", Address: "{{ default_ip_address|guest_ip_addresses.0 }}"},
	{Type: "aws_instance", Name: "{{ tags.Name|id }}", Address: "{{ public_ip|private_ip }}"},
	{Type: "google_compute_instance", Name: "{{ name }}", Address: "{{ network_interface.0.access_config.0.nat_ip|network_interface.0.network_ip }}"},
	{Type: "azurerm_linux_virtual_machine", Name: "{{ name }}", Address: "{{ public_ip_address|private_ip_address }}"},
	{Type: "azurerm_windows_virtual_machine", Name: "{{ name }}", Address: "{{ public_ip_address|private_ip_address }}"},
	{Type: "hcloud_server", Name: "{{ name }}", Address: "{{ ipv4_address }}"},
	{Type: "digitalocean_droplet", Name: "{{ name }}", Address: "{{ ipv4_address }}"},
	{Type: "linode_instance", Name: "{{ label }}", Address: "{{ ip_address }}"},
	{Type: "vultr_instance", Name: "{{ hostname|label }}", Address: "{{ main_ip }}"},
	{Type: "openstack_compute_instance_v2", Name: "{{ name }}", Address: "{{ access_ip_v4 }}"},
	{Type: "proxmox_vm_qemu", Name: "{{ name }}", Address: "{{ default_ipv4_address }}"},
	{Type: "proxmox_virtual_environment_vm", Name: "{{ name }}", Address: "{{ ipv4_addresses.1.0|initialization.0.ip_config.0.ipv4.0.address }}"},
	{Type: "libvirt_domain", Name: "{{ name }}", Address: "{{ network_interface.0.addresses.0 }}"},
	{Type: "scaleway_instance_server", Name: "{{ name }}", Address: "{{ public_ip|private_ip }}"},
	{Type: "exoscale_compute_instance", Name: "{{ name }}", Address: "{{ public_ip_address }}"},
	{Type: "upcloud_server", Name: "{{ hostname }}", Address: "{{ network_interface.0.ip_address }}"},
}

// tfResource is one resource instance, whichever format the state came in
type tfResource struct {
	Type, Name, Module, Address string
	Attrs                       map[string]interface{}
}

func loadTerraformInventory(ctx context.Context, cfg map[string]interface{}) ([]byte, error) {
	base, _ := cfg["_dir"].(string)
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) || base == "" {
			return p
		}
		return filepath.Join(base, p)
	}
	var resources []tfResource
	var err error
	switch {
	case stringArg(cfg, "project", "") != "":
		resources, err = terraformShow(ctx, resolve(stringArg(cfg, "project", "")), stringArg(cfg, "binary", "terraform"), stringArg(cfg, "workspace", ""))
	default:
		resources, err = terraformStateFile(resolve(stringArg(cfg, "state", "terraform.tfstate")))
	}
	if err != nil {
		return nil, fmt.Errorf("terraform: %w", err)
	}
	rules, err := terraformHostRules(cfg)
	if err != nil {
		return nil, fmt.Errorf("terraform: %w", err)
	}
	groupBy := listArg(cfg, "group_by")
	if groupBy == nil {
		groupBy = []string{"type", "module"}
	}
	inv := newAnsibleInventory()
	// the ansible provider's groups: name -> children/vars (flattened to hosts)
	ansibleGroups := map[string]map[string]interface{}{}
	for _, r := range resources {
		if r.Type == "ansible_group" {
			ansibleGroups[fmt.Sprint(r.Attrs["name"])] = r.Attrs
		}
	}
	for _, r := range resources {
		if r.Type == "ansible_host" {
			name := fmt.Sprint(r.Attrs["name"])
			vars := map[string]interface{}{}
			if m, ok := r.Attrs["variables"].(map[string]interface{}); ok {
				for k, v := range m {
					vars[k] = v
				}
			}
			var groups []string
			for _, g := range toStrings(r.Attrs["groups"]) {
				groups = append(groups, g)
				if ag, ok := ansibleGroups[g]; ok {
					if gv, ok := ag["variables"].(map[string]interface{}); ok {
						for k, v := range gv {
							if _, set := vars[k]; !set {
								vars[k] = v
							}
						}
					}
				}
			}
			inv.addHost(name, vars, groups...)
			continue
		}
		for _, rule := range rules {
			if rule.Type != r.Type {
				continue
			}
			name := tfTemplate(rule.Name, r.Attrs)
			if name == "" {
				name = r.Name
			}
			vars := map[string]interface{}{"terraform_type": r.Type, "terraform_address": r.Address, "terraform_module": r.Module, "terraform": r.Attrs}
			if addr := tfAddress(tfTemplate(rule.Address, r.Attrs)); addr != "" {
				vars["ansible_host"] = addr
			}
			for k, v := range rule.Vars {
				if s, ok := v.(string); ok {
					v = tfTemplate(s, r.Attrs)
				}
				vars[k] = v
			}
			var groups []string
			for _, g := range groupBy {
				switch g {
				case "type":
					groups = append(groups, "tf_"+r.Type)
				case "module":
					if r.Module != "" {
						groups = append(groups, "module_"+strings.TrimPrefix(r.Module, "module."))
					}
				}
			}
			for _, g := range rule.Groups {
				if v := tfTemplate(g, r.Attrs); v != "" {
					groups = append(groups, v)
				}
			}
			inv.addHost(name, vars, groups...)
			break
		}
	}
	return inv.bytes()
}

func terraformHostRules(cfg map[string]interface{}) ([]terraformHostRule, error) {
	raw, ok := cfg["hosts"].([]interface{})
	if !ok || len(raw) == 0 {
		return defaultTerraformHosts, nil
	}
	rules := make([]terraformHostRule, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok || fmt.Sprint(m["type"]) == "" || m["type"] == nil {
			return nil, fmt.Errorf("hosts[%d]: type is required", i)
		}
		rule := terraformHostRule{Type: fmt.Sprint(m["type"]), Name: stringArg(m, "name", "{{ name }}"), Address: stringArg(m, "address", ""), Groups: listArg(m, "groups")}
		if v, ok := m["vars"].(map[string]interface{}); ok {
			rule.Vars = v
		}
		// the default rule of a known type fills what is not given
		for _, d := range defaultTerraformHosts {
			if d.Type == rule.Type {
				if _, set := m["name"]; !set {
					rule.Name = d.Name
				}
				if rule.Address == "" {
					rule.Address = d.Address
				}
			}
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

var tfTemplateRef = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

// tfTemplate fills {{ path }} references with resource attributes: dotted
// paths, numeric list indexes, "a|b" the first non-empty of the paths
func tfTemplate(text string, attrs map[string]interface{}) string {
	return tfTemplateRef.ReplaceAllStringFunc(text, func(m string) string {
		expr := strings.TrimSpace(tfTemplateRef.FindStringSubmatch(m)[1])
		for _, path := range strings.Split(expr, "|") {
			if v := attrPath(attrs, strings.TrimSpace(path)); v != nil {
				s := fmt.Sprint(v)
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
		return ""
	})
}

// tfAddress cleans an address attribute: a CIDR loses its length, the
// cloud-init words dhcp/auto/manual are no address
func tfAddress(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '/'); i > 0 {
		s = s[:i]
	}
	switch strings.ToLower(s) {
	case "dhcp", "dhcp6", "auto", "manual", "":
		return ""
	}
	return s
}

func attrPath(v interface{}, path string) interface{} {
	for _, part := range strings.Split(path, ".") {
		switch cur := v.(type) {
		case map[string]interface{}:
			v = cur[part]
		case []interface{}:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(cur) {
				return nil
			}
			v = cur[i]
		default:
			return nil
		}
		if v == nil {
			return nil
		}
	}
	return v
}

func toStrings(v interface{}) []string {
	items, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, x := range items {
		out = append(out, fmt.Sprint(x))
	}
	return out
}

// terraformStateFile reads a state file (format version 4)
func terraformStateFile(path string) ([]tfResource, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the inventory names the state file
	if err != nil {
		return nil, err
	}
	var state struct {
		Resources []struct {
			Mode, Type, Name, Module string
			Instances                []struct {
				IndexKey   interface{}            `json:"index_key"`
				Attributes map[string]interface{} `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var out []tfResource
	for _, r := range state.Resources {
		if r.Mode != "managed" {
			continue
		}
		for _, inst := range r.Instances {
			addr := r.Type + "." + r.Name
			if r.Module != "" {
				addr = r.Module + "." + addr
			}
			if inst.IndexKey != nil {
				addr += fmt.Sprintf("[%v]", inst.IndexKey)
			}
			out = append(out, tfResource{Type: r.Type, Name: r.Name, Module: r.Module, Address: addr, Attrs: inst.Attributes})
		}
	}
	return out, nil
}

// terraformShow runs `terraform show -json` in a project directory
func terraformShow(ctx context.Context, dir, binary, workspace string) ([]tfResource, error) {
	if workspace != "" {
		cmd := exec.CommandContext(ctx, binary, "workspace", "select", workspace) // #nosec G204 -- the inventory's own binary and workspace
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s workspace select %s: %v: %s", binary, workspace, err, strings.TrimSpace(string(out)))
		}
	}
	cmd := exec.CommandContext(ctx, binary, "show", "-json") // #nosec G204 -- the inventory's own binary
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1", "TF_CLI_ARGS=-no-color")
	out, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			msg = strings.TrimSpace(string(ee.Stderr))
		}
		return nil, fmt.Errorf("%s show -json in %s: %s", binary, dir, msg)
	}
	var show struct {
		Values struct {
			RootModule tfModule `json:"root_module"`
		} `json:"values"`
	}
	if err := json.Unmarshal(out, &show); err != nil {
		return nil, fmt.Errorf("%s show -json: %w", binary, err)
	}
	var res []tfResource
	collectTfModule(&show.Values.RootModule, &res)
	return res, nil
}

type tfModule struct {
	Address   string `json:"address"`
	Resources []struct {
		Address string                 `json:"address"`
		Mode    string                 `json:"mode"`
		Type    string                 `json:"type"`
		Name    string                 `json:"name"`
		Values  map[string]interface{} `json:"values"`
	} `json:"resources"`
	ChildModules []tfModule `json:"child_modules"`
}

func collectTfModule(m *tfModule, out *[]tfResource) {
	for _, r := range m.Resources {
		if r.Mode != "managed" {
			continue
		}
		*out = append(*out, tfResource{Type: r.Type, Name: r.Name, Module: m.Address, Address: r.Address, Attrs: r.Values})
	}
	for i := range m.ChildModules {
		collectTfModule(&m.ChildModules[i], out)
	}
}

// TerraformHostTypes lists the resource types the plugin knows as hosts
func TerraformHostTypes() []string {
	types := make([]string, 0, len(defaultTerraformHosts))
	for _, r := range defaultTerraformHosts {
		types = append(types, r.Type)
	}
	sort.Strings(types)
	return types
}

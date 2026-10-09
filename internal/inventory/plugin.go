package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Inventory plugins: a YAML inventory file whose top-level "plugin" key
// names a source (netbox, ...) is not a list of hosts but the way to ask a
// system for them. The plugin answers in the Ansible inventory JSON shape,
// which the loader already reads (dynamic scripts answer the same way).

type pluginLoader func(ctx context.Context, cfg map[string]interface{}) ([]byte, error)

var inventoryPlugins = map[string]pluginLoader{
	"netbox":  loadNetboxInventory,
	"vsphere": loadVsphereInventory,
	"proxmox": loadProxmoxInventory,
	"netbird": loadNetbirdInventory,
}

// pluginConfig reads the file when it is a plugin configuration; nil when
// it is an ordinary inventory
func pluginConfig(data []byte) (name string, cfg map[string]interface{}, err error) {
	var m map[string]interface{}
	if yaml.Unmarshal(data, &m) != nil || m == nil {
		return "", nil, nil
	}
	p, ok := m["plugin"].(string)
	if !ok || p == "" {
		return "", nil, nil
	}
	name = strings.TrimPrefix(strings.TrimPrefix(p, "onigirazu."), "ansible.builtin.")
	if _, known := inventoryPlugins[name]; !known {
		known := make([]string, 0, len(inventoryPlugins))
		for k := range inventoryPlugins {
			known = append(known, k)
		}
		sort.Strings(known)
		return "", nil, fmt.Errorf("inventory plugin %q is not one of %s", p, strings.Join(known, ", "))
	}
	return name, m, nil
}

var envRef = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$|^env:([A-Za-z_][A-Za-z0-9_]*)$`)

// secretArg reads a plugin argument that may name an environment variable
// ("${NETBOX_TOKEN}" or "env:NETBOX_TOKEN") instead of holding the value
func secretArg(cfg map[string]interface{}, key, env string) string {
	v, _ := cfg[key].(string)
	if m := envRef.FindStringSubmatch(v); m != nil {
		return os.Getenv(m[1] + m[2])
	}
	if v == "" && env != "" {
		return os.Getenv(env)
	}
	return v
}

func stringArg(cfg map[string]interface{}, key, def string) string {
	if v, ok := cfg[key].(string); ok && v != "" {
		return v
	}
	return def
}

func boolArg(cfg map[string]interface{}, key string, def bool) bool {
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	return def
}

func listArg(cfg map[string]interface{}, key string) []string {
	switch v := cfg[key].(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, x := range v {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}

// ansibleInventory builds the inventory JSON a dynamic script prints
type ansibleInventory struct {
	groups   map[string]map[string]bool
	hostvars map[string]map[string]interface{}
}

func newAnsibleInventory() *ansibleInventory {
	return &ansibleInventory{groups: map[string]map[string]bool{}, hostvars: map[string]map[string]interface{}{}}
}

func (a *ansibleInventory) addHost(name string, vars map[string]interface{}, groups ...string) {
	a.hostvars[name] = vars
	for _, g := range append([]string{"all"}, groups...) {
		g = groupName(g)
		if g == "" {
			continue
		}
		if a.groups[g] == nil {
			a.groups[g] = map[string]bool{}
		}
		a.groups[g][name] = true
	}
}

func (a *ansibleInventory) bytes() ([]byte, error) {
	out := map[string]interface{}{"_meta": map[string]interface{}{"hostvars": a.hostvars}}
	for g, hosts := range a.groups {
		names := make([]string, 0, len(hosts))
		for h := range hosts {
			names = append(names, h)
		}
		sort.Strings(names)
		out[g] = map[string]interface{}{"hosts": names}
	}
	return json.Marshal(out)
}

var groupNameClean = regexp.MustCompile(`[^A-Za-z0-9_]+`)

// groupName makes a group name the way Ansible's keyed_groups do: lower
// case, anything else becomes _
func groupName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = groupNameClean.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

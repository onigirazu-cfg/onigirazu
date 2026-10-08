package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AddHostModule adds a host (and groups) to the in-memory inventory for the
// rest of the run. The engine applies its result; it runs once per task (per
// loop item), as in Ansible.
type AddHostModule struct {
	*BaseModule
}

// NewAddHostModule creates the add_host module
func NewAddHostModule() *AddHostModule {
	return &AddHostModule{BaseModule: NewBaseModule("add_host")}
}

func (m *AddHostModule) GetDescription() string {
	return "Add a host (and groups) to the in-memory inventory for the rest of the run"
}

var addHostKeys = map[string]bool{"name": true, "hostname": true, "host": true, "groups": true, "group": true, "groupname": true}

func (m *AddHostModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	name := strings.TrimSpace(getStringArg(args, "name", getStringArg(args, "hostname", getStringArg(args, "host", ""))))
	if name == "" {
		result.Success, result.Failed = false, true
		result.Error = "add_host needs name"
		return result, nil
	}
	// host:port, as Ansible reads it
	vars := map[string]interface{}{}
	if h, p, ok := strings.Cut(name, ":"); ok && !strings.Contains(p, ":") && p != "" {
		name = h
		vars["ansible_port"] = p
	}
	for k, v := range args {
		if !strings.HasPrefix(k, "_") && !addHostKeys[k] {
			vars[k] = v
		}
	}
	groups := listArg(args, "groups", "group", "groupname")
	if groups == nil {
		groups = []string{}
	}
	result.Changed = true
	result.Output["add_host"] = map[string]interface{}{"host_name": name, "groups": groups, "host_vars": vars}
	result.Output["msg"] = fmt.Sprintf("added host %s", name)
	result.Duration = time.Since(start)
	return result, nil
}

func (m *AddHostModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", getStringArg(args, "hostname", getStringArg(args, "host", ""))) == "" {
		return fmt.Errorf("add_host needs name")
	}
	return nil
}

// GroupByModule puts the host into a group named by key (created with
// parents, default "all") for the rest of the run; the engine applies it
type GroupByModule struct {
	*BaseModule
}

// NewGroupByModule creates the group_by module
func NewGroupByModule() *GroupByModule {
	return &GroupByModule{BaseModule: NewBaseModule("group_by")}
}

func (m *GroupByModule) GetDescription() string {
	return "Group the hosts by a key for the rest of the run"
}

func (m *GroupByModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	key := strings.TrimSpace(getStringArg(args, "key", ""))
	if key == "" {
		result.Success, result.Failed = false, true
		result.Error = "group_by needs key"
		return result, nil
	}
	// spaces are not allowed in group names: Ansible makes them _
	key = strings.ReplaceAll(key, " ", "_")
	parents := listArg(args, "parents")
	if len(parents) == 0 {
		parents = []string{"all"}
	}
	result.Output["add_group"] = key
	result.Output["parent_groups"] = parents
	result.Duration = time.Since(start)
	return result, nil
}

func (m *GroupByModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "key", "") == "" {
		return fmt.Errorf("group_by needs key")
	}
	return nil
}

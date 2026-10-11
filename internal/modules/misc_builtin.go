package modules

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func itoa(n int) string { return strconv.Itoa(n) }

func localDirEntries(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func localReadFile(p string) ([]byte, error) { return os.ReadFile(p) } // #nosec G304 -- the playbook's own file

// modeArg reads mode ("0644", 420) with a default
func modeArg(args map[string]interface{}, def os.FileMode) os.FileMode {
	switch v := args["mode"].(type) {
	case string:
		if n, err := strconv.ParseUint(strings.TrimPrefix(v, "0o"), 8, 32); err == nil {
			return os.FileMode(n)
		}
	case int:
		return os.FileMode(uint32(v) & 0o7777) // #nosec G115 -- masked to the permission bits
	case float64:
		return os.FileMode(uint32(v) & 0o7777) // #nosec G115 -- masked to the permission bits
	}
	return def
}

// SetStatsModule records custom stats of the run (Ansible's set_stats):
// data, per_host, aggregate; they land under ansible_stats of the host (and
// of every host with per_host: false) for the playbook to print
type SetStatsModule struct{ *BaseModule }

func NewSetStatsModule() *SetStatsModule {
	return &SetStatsModule{BaseModule: NewBaseModule("set_stats")}
}

func (m *SetStatsModule) GetDescription() string { return "Record custom statistics of the run" }

func (m *SetStatsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	data, _ := args["data"].(map[string]interface{})
	if data == nil {
		result.Success, result.Error = false, "set_stats requires 'data'"
		return result, nil
	}
	result.Output["ansible_stats"] = map[string]interface{}{"data": data, "per_host": getBoolArg(args, "per_host", false), "aggregate": getBoolArg(args, "aggregate", true)}
	result.Output["ansible_facts"] = map[string]interface{}{"ansible_stats": data}
	result.Duration = time.Since(start)
	return result, nil
}

// ValidateArgumentSpecModule checks variables against an argument spec
// (Ansible's validate_argument_spec): argument_spec (options), provided_arguments
type ValidateArgumentSpecModule struct{ *BaseModule }

func NewValidateArgumentSpecModule() *ValidateArgumentSpecModule {
	return &ValidateArgumentSpecModule{BaseModule: NewBaseModule("validate_argument_spec")}
}

func (m *ValidateArgumentSpecModule) GetDescription() string {
	return "Validate variables against an argument spec"
}

func (m *ValidateArgumentSpecModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	spec, _ := args["argument_spec"].(map[string]interface{})
	if spec == nil {
		result.Success, result.Error = false, "validate_argument_spec requires 'argument_spec'"
		return result, nil
	}
	provided, _ := args["provided_arguments"].(map[string]interface{})
	if provided == nil {
		provided, _ = args["_task_vars"].(map[string]interface{})
	}
	defs := types.ArgumentSpecOptions(spec)
	var errs []string
	names := make([]string, 0, len(defs))
	for n := range defs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		def := defs[n]
		v, set := provided[n]
		if !set || v == nil {
			if def.Required {
				errs = append(errs, n+": required")
			}
			continue
		}
		switch def.Type {
		case "integer":
			if _, ok := toInt(v); !ok {
				errs = append(errs, fmt.Sprintf("%s: must be an integer, got %v", n, v))
			}
		case "boolean":
			if _, ok := v.(bool); !ok {
				errs = append(errs, fmt.Sprintf("%s: must be a boolean, got %v", n, v))
			}
		case "array":
			if _, ok := v.([]interface{}); !ok {
				errs = append(errs, fmt.Sprintf("%s: must be a list", n))
			}
		case "object":
			if _, ok := v.(map[string]interface{}); !ok {
				errs = append(errs, fmt.Sprintf("%s: must be a dict", n))
			}
		}
		if len(def.Constraints.Enum) > 0 {
			ok := false
			for _, c := range def.Constraints.Enum {
				if fmt.Sprint(c) == fmt.Sprint(v) {
					ok = true
				}
			}
			if !ok {
				errs = append(errs, fmt.Sprintf("%s: value must be one of %v", n, def.Constraints.Enum))
			}
		}
	}
	if len(errs) > 0 {
		result.Success, result.Error = false, "validation of arguments failed: "+strings.Join(errs, "; ")
		result.Output["argument_errors"] = errs
		return result, nil
	}
	result.Output["validate_args_context"] = map[string]interface{}{"checked": len(defs)}
	result.Duration = time.Since(start)
	return result, nil
}

package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"gopkg.in/yaml.v3"
)

// IncludeVarsModule loads variables from YAML files on the control machine
// into the host's variables, as Ansible's include_vars: file (relative to
// the playbook, or to the role's vars/), or every .yml/.yaml of dir; name
// puts them under one key
type IncludeVarsModule struct {
	*BaseModule
}

// NewIncludeVarsModule creates the include_vars module
func NewIncludeVarsModule() *IncludeVarsModule {
	return &IncludeVarsModule{BaseModule: NewBaseModule("include_vars")}
}

func (m *IncludeVarsModule) GetDescription() string {
	return "Load variables from YAML files"
}

func (m *IncludeVarsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Output: map[string]interface{}{}}
	fail := func(err error) (types.TaskResult, error) {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result, nil
	}
	base, _ := taskVars(args)["playbook_dir"].(string)
	resolve := func(p string) string {
		if filepath.IsAbs(p) || base == "" {
			return p
		}
		return filepath.Join(base, p)
	}

	var files []string
	if file := getStringArg(args, "file", ""); file != "" {
		files = append(files, resolve(file))
	} else if dir := getStringArg(args, "dir", ""); dir != "" {
		entries, err := os.ReadDir(resolve(dir))
		if err != nil {
			return fail(err)
		}
		for _, e := range entries {
			if ext := filepath.Ext(e.Name()); !e.IsDir() && (ext == ".yml" || ext == ".yaml" || ext == ".json") {
				files = append(files, filepath.Join(resolve(dir), e.Name()))
			}
		}
		sort.Strings(files)
	} else {
		return fail(fmt.Errorf("file or dir is required"))
	}

	vars := map[string]interface{}{}
	for _, f := range files {
		data, err := os.ReadFile(f) // #nosec G304 -- the playbook names its vars files
		if err != nil {
			return fail(err)
		}
		var loaded map[string]interface{}
		if err := yaml.Unmarshal(data, &loaded); err != nil {
			return fail(fmt.Errorf("%s: %w", f, err))
		}
		for k, v := range loaded {
			vars[k] = v
		}
	}
	facts := vars
	if name := getStringArg(args, "name", ""); name != "" {
		facts = map[string]interface{}{name: vars}
	}
	result.Success = true
	result.Output["onigirazu_facts"] = facts
	result.Output["ansible_included_var_files"] = files
	result.Output["msg"] = fmt.Sprintf("loaded %s", strings.Join(files, ", "))
	result.Duration = time.Since(start)
	return result, nil
}

func (m *IncludeVarsModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "file", "") == "" && getStringArg(args, "dir", "") == "" {
		return fmt.Errorf("argument 'file' or 'dir' is required")
	}
	return nil
}

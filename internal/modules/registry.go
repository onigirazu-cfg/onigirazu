package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Registry manages available modules
type Registry struct {
	modules map[string]types.Module
}

func NewRegistry() *Registry {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	// Register built-in modules
	registry.RegisterModule(NewPingModule())
	registry.RegisterModule(NewFileModule())
	registry.RegisterModule(NewCopyModule())
	registry.RegisterModule(NewFetchModule())
	registry.RegisterModule(NewGetURLModule())
	registry.RegisterModule(NewServiceModule())
	registry.RegisterModule(NewUnifiedPackageModule()) // Unified package module with all features
	registry.RegisterModule(NewCommandModule())
	registry.RegisterModule(NewShellModule())
	registry.RegisterModule(NewUserModule())
	registry.RegisterModule(NewGroupModule())
	registry.RegisterModule(NewTemplateModule())
	registry.RegisterModule(NewGitModule())
	registry.RegisterModule(NewDebugModule())
	registry.RegisterModule(NewSetFactModule())
	registry.RegisterModule(NewStatModule())
	registry.RegisterModule(NewFindModule())
	registry.RegisterModule(NewLineinfileModule())
	registry.RegisterModule(NewSystemdModule())
	registry.RegisterModule(NewCronModule())
	registry.RegisterModule(NewFirewallModule())
	registry.RegisterModule(NewConfigModule())

	// System Control modules (critical)
	registry.RegisterModule(NewSysctlModule())
	registry.RegisterModule(NewRebootModule())
	registry.RegisterModule(NewMountModule())

	// Archive and compression module
	registry.RegisterModule(NewArchiveModule())

	// New modules for completeness
	registry.RegisterModule(NewFailModule())
	registry.RegisterModule(NewAssertModule())
	registry.RegisterModule(NewReplaceModule())
	registry.RegisterModule(NewMetaModule())
	registry.RegisterModule(NewIncludeRoleModule("include_role"))
	registry.RegisterModule(NewIncludeRoleModule("import_role"))
	registry.RegisterModule(NewIncludeVarsModule())
	registry.RegisterModule(NewPauseModule())
	registry.RegisterModule(NewScriptModule())
	registry.RegisterModule(NewWaitForModule())
	registry.RegisterModule(NewAuthorizedKeyModule())
	registry.RegisterModule(NewBlockinfileModule())
	registry.RegisterModule(NewAptModule())
	registry.RegisterModule(NewYumModule())
	registry.RegisterModule(NewURIModule())

	// Docker/Container modules
	registry.RegisterModule(NewDockerContainerModule())
	registry.RegisterModule(NewDockerImageModule())
	registry.RegisterModule(NewDockerComposeModule())
	registry.RegisterModule(NewPodmanModule())

	// Database modules
	registry.RegisterModule(NewMySQLDBModule())
	registry.RegisterModule(NewMySQLUserModule())
	registry.RegisterModule(NewPostgreSQLDBModule())
	registry.RegisterModule(NewPostgreSQLUserModule())
	registry.RegisterModule(NewMongoDBModule())

	return registry
}

// RegisterModule registers a new module (internal method)
func (r *Registry) RegisterModule(module types.Module) {
	r.modules[module.GetName()] = module
}

// Register registers a new module (interface method)
func (r *Registry) Register(name string, module interfaces.ModuleExecutor) error {
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module '%s' already registered", name)
	}
	// Convert ModuleExecutor to types.Module if possible
	if typedModule, ok := module.(types.Module); ok {
		r.modules[name] = typedModule
		return nil
	}
	return fmt.Errorf("module must implement types.Module interface")
}

// GetModule returns module by name
func (r *Registry) GetModule(name string) (types.Module, error) {
	module, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found", name)
	}
	return module, nil
}

// Get returns module by name (interface method)
func (r *Registry) Get(name string) (interfaces.ModuleExecutor, error) {
	module, err := r.GetModule(name)
	if err != nil {
		return nil, err
	}
	return module, nil
}

// ListModules returns list of available modules
func (r *Registry) ListModules() []string {
	var names []string
	for name := range r.modules {
		names = append(names, name)
	}
	return names
}

// List returns list of available modules (interface method)
func (r *Registry) List() []string {
	return r.ListModules()
}

// GetModuleInfo returns detailed information about all modules
func (r *Registry) GetModuleInfo() []map[string]string {
	var modules []map[string]string
	for name, module := range r.modules {
		modules = append(modules, map[string]string{
			"name":        name,
			"description": module.GetDescription(),
		})
	}
	return modules
}

// Unregister removes a module (interface method)
func (r *Registry) Unregister(name string) error {
	if _, exists := r.modules[name]; !exists {
		return fmt.Errorf("module '%s' not found", name)
	}
	delete(r.modules, name)
	return nil
}

// ExecuteTask executes task using appropriate module
func (r *Registry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error) {
	module, err := r.GetModule(task.Module)
	if err != nil {
		return types.TaskResult{}, err
	}

	// Prepare arguments for module
	args := make(map[string]interface{})

	// Copy arguments from task first
	for key, value := range task.Args {
		args[key] = value
	}
	if !dataArgModules[task.Module] {
		normalizeArgs(args)
	}

	// The task name travels separately: "name" belongs to the module
	// (user name, package name, ...) and must not be filled from the task title
	args["_task_name"] = task.Name

	// Variables travel under a reserved key: merged into args, a play var
	// named "state" or "name" silently became a module argument
	if variables != nil {
		args["_vars"] = variables
	}

	// Add become parameters to args (with special prefix to avoid conflicts)
	if task.Become {
		args["_become"] = true
		args["_become_user"] = task.BecomeUser
		args["_become_method"] = task.BecomeMethod
	}

	if task.Diff {
		args["_diff"] = true
	}

	// Check mode: modules that support it report what they would change;
	// the others are skipped rather than run
	if task.CheckMode != nil && *task.CheckMode {
		if !checkModeModules[task.Module] {
			return types.TaskResult{
				TaskName: task.Name, Host: host.Name, Module: task.Module,
				Success: true, Skipped: true, Timestamp: time.Now(),
				Output: map[string]interface{}{"msg": fmt.Sprintf("skipped: check mode is not supported by %s", task.Module)},
			}, nil
		}
		args["_check_mode"] = true
	}

	// Every executor a module creates for this host picks the settings up
	host.Environment = nil
	if len(task.Environment) > 0 {
		host.Environment = make(map[string]string, len(task.Environment))
		for k, v := range task.Environment {
			host.Environment[k] = fmt.Sprint(v)
		}
	}
	host.Become = task.Become
	host.BecomeUser = task.BecomeUser
	host.BecomeMethod = task.BecomeMethod

	// the target file of a file module as it was, for rollback
	var before map[string]interface{}
	if !inCheckMode(args) {
		before = captureBefore(ctx, host, task.Module, args)
	}

	result, err := module.Execute(ctx, host, args)
	if result.Changed && before != nil && before["error"] == nil {
		result.Before = before
	}
	if result.TaskName == "" {
		result.TaskName = task.Name
	}
	// Modules report some failures only through Success=false; the engine
	// looks at Failed, so without this a failed apt-get counted as success
	if err == nil && !result.Success && !result.Skipped && !result.Failed {
		result.Failed = true
		if result.Error == "" {
			result.Error = fmt.Sprintf("module %s reported failure", task.Module)
		}
	}
	return result, err
}

// checkModeModules support check mode: they read, or they compare and stop
// before changing anything. Every other module is skipped in check mode.
var checkModeModules = map[string]bool{
	// read only
	"ping": true, "debug": true, "set_fact": true, "stat": true, "find": true,
	"fail": true, "wait_for": true, "assert": true, "include_vars": true,
	// compare, then change
	"file": true, "copy": true, "template": true, "lineinfile": true, "blockinfile": true,
	"apt": true, "yum": true, "package": true, "service": true, "user": true, "group": true,
	"cron": true, "sysctl": true, "get_url": true, "git": true, "systemd": true,
	"mount": true, "config": true, "replace": true, "docker_container": true, "podman": true, "docker_image": true,
}

// dataArgModules take their arguments as data whose types are kept:
// set_fact stores them, config writes them into JSON/YAML/TOML files
var dataArgModules = map[string]bool{"set_fact": true, "config": true, "debug": true, "assert": true, "include_vars": true}

// normalizeArgs turns top-level YAML numbers into strings: modules read
// text arguments as strings (cron "minute: 0" became "*") and numeric ones
// through toInt, which takes both. "mode: 0644" is the integer 420 in YAML
// (octal, as in Ansible) and becomes "0644".
func normalizeArgs(args map[string]interface{}) {
	for key, value := range args {
		if strings.HasPrefix(key, "_") {
			continue
		}
		switch v := value.(type) {
		case int, int64, uint64:
			n, _ := toInt(v)
			if key == "mode" {
				args[key] = fmt.Sprintf("%04o", n)
			} else {
				args[key] = strconv.Itoa(n)
			}
		case float64:
			args[key] = strconv.FormatFloat(v, 'f', -1, 64)
		}
	}
}

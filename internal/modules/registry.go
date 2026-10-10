package modules

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/bridge"
	"github.com/onigirazu-cfg/onigirazu/internal/winrm"

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
	registry.RegisterModule(NewVerifyModule())
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
	registry.RegisterModule(NewAddHostModule())
	registry.RegisterModule(NewGroupByModule())
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
	registry.RegisterModule(NewSetupModule("setup"))
	registry.RegisterModule(NewSetupModule("gather_facts"))
	registry.RegisterModule(NewSlurpModule())
	registry.RegisterModule(NewWinPingModule())
	registry.RegisterModule(NewWinCommandModule())
	registry.RegisterModule(NewWinShellModule())
	registry.RegisterModule(NewWinPowerShellModule())
	registry.RegisterModule(NewWinRegeditModule())
	registry.RegisterModule(NewWinFileModule())
	registry.RegisterModule(NewWinCopyModule())
	registry.RegisterModule(NewWinServiceModule())
	registry.RegisterModule(NewWinTimezoneModule())
	registry.RegisterModule(NewWinFirewallRuleModule())
	registry.RegisterModule(NewWinFirewallModule())
	registry.RegisterModule(NewWinGroupMembershipModule())
	registry.RegisterModule(NewWinFeatureModule())
	registry.RegisterModule(NewWinRebootModule())
	registry.RegisterModule(NewWinScheduledTaskModule())
	registry.RegisterModule(NewWinChocolateyModule())
	registry.RegisterModule(NewWinOptionalFeatureModule())
	registry.RegisterModule(NewWinDiskFactsModule())
	registry.RegisterModule(NewWinInitializeDiskModule())
	registry.RegisterModule(NewWinPartitionModule())
	registry.RegisterModule(NewWinFormatModule())
	registry.RegisterModule(NewAsyncStatusModule())
	registry.RegisterModule(NewGetentModule())
	registry.RegisterModule(NewHostnameModule())
	registry.RegisterModule(NewIniFileModule())
	registry.RegisterModule(NewPipModule())
	registry.RegisterModule(NewRouterosAPIModule())
	registry.RegisterModule(NewRouterosCommandModule())
	registry.RegisterModule(NewRouterosFactsModule())
	registry.RegisterModule(NewUfwModule())
	registry.RegisterModule(NewDockerHostInfoModule())
	registry.RegisterModule(NewIncludeRoleModule("include_role"))
	registry.RegisterModule(NewIncludeRoleModule("import_role"))
	registry.RegisterModule(NewIncludeVarsModule())
	registry.RegisterModule(NewPauseModule())
	registry.RegisterModule(NewScriptModule())
	registry.RegisterModule(NewWaitForModule())
	registry.RegisterModule(NewAuthorizedKeyModule())
	registry.RegisterModule(NewTimezoneModule())
	registry.RegisterModule(NewUnarchiveModule())
	registry.RegisterModule(NewAptRepositoryModule())
	registry.RegisterModule(NewAptKeyModule())
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
	if task.Module == types.DynamicAction {
		resolved, err := resolveDynamicAction(task)
		if err != nil {
			return types.TaskResult{TaskName: task.Name, Host: host.Name, Module: task.Module, Failed: true,
				Error: err.Error(), Timestamp: time.Now()}, nil
		}
		task = resolved
	}
	module, err := r.GetModule(task.Module)
	// a built-in Linux module on a Windows host; modules onigirazu lacks
	// may still go through the bridge
	if err == nil && winrm.IsWindows(host) && !windowsSafe[task.Module] && !strings.HasPrefix(task.Module, "win_") {
		return types.TaskResult{TaskName: task.Name, Host: host.Name, Module: task.Module, Failed: true,
			Error: winModuleError(task.Module), Timestamp: time.Now()}, nil
	}
	if err != nil {
		if bridge.Allowed(task.Module) {
			return runBridged(ctx, task, host), nil
		}
		return types.TaskResult{}, err
	}

	// Prepare arguments for module
	args := make(map[string]interface{})

	// Copy arguments from task first
	for key, value := range task.Args {
		args[key] = value
	}
	// arguments the module would ignore fail the task, as in Ansible: an
	// unsupported option must not widen what a task does (find without
	// age would have deleted everything)
	if msg := unsupportedParameters(task.Module, args); msg != "" {
		return types.TaskResult{TaskName: task.Name, Host: host.Name, Module: task.Module, Failed: true,
			Error: msg, Timestamp: time.Now()}, nil
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
	// in check mode only when asked, or when the host's server describes
	// the file without a process: the module then asks the host nothing more
	if !inCheckMode(args) || task.Capture {
		before = captureBefore(ctx, host, task.Module, args)
	} else {
		before = captureNative(ctx, host, task.Module, args)
	}
	// the module may use it instead of asking the host again
	if before != nil && before["error"] == nil {
		args["_before"] = before
	}

	result, err := executeRecovering(ctx, module, host, args, task)
	if result.Changed || err != nil {
		invalidateLoopProbes(ctx)
	}
	if result.Changed && before != nil && before["error"] == nil {
		result.Before = before
	}
	if result.TaskName == "" {
		result.TaskName = task.Name
	}
	if err == nil && !result.Skipped && !result.Failed && result.Success {
		result.Resources = declaredResources(task.Module, args, before)
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

//go:generate go run ../cli/gen_modargs . module_args.go modules

// AcceptedArgs are Ansible arguments a module accepts without reading them:
// their effect is the default here
var AcceptedArgs = map[string][]string{
	"docker_container": {"comparisons"},
	// the WinRM/SSH client's own timeouts apply; the boot time is read the same way
	"win_reboot": {"boot_time_command", "connect_timeout", "shutdown_timeout"},
}

// FreeArgModules take any argument (add_host: host variables; set_fact: facts)
var FreeArgModules = map[string]bool{"add_host": true, "set_fact": true}

// unsupportedParameters names the arguments a module does not read, in
// Ansible's words, or "" when all are known. Modules the table does not
// describe are not checked.
func unsupportedParameters(module string, args map[string]interface{}) string {
	known := ModuleArgs[module]
	if len(known) == 0 || FreeArgModules[module] {
		return ""
	}
	ok := make(map[string]bool, len(known)+len(AcceptedArgs[module]))
	for _, a := range known {
		ok[a] = true
	}
	for _, a := range AcceptedArgs[module] {
		ok[a] = true
	}
	var bad []string
	for a := range args {
		if !strings.HasPrefix(a, "_") && !ok[a] {
			bad = append(bad, a)
		}
	}
	if len(bad) == 0 {
		return ""
	}
	sort.Strings(bad)
	return fmt.Sprintf("Unsupported parameters for (%s) module: %s. Supported parameters include: %s",
		module, strings.Join(bad, ", "), strings.Join(known, ", "))
}

// checkModeModules support check mode: they read, or they compare and stop
// before changing anything. Every other module is skipped in check mode.
var checkModeModules = map[string]bool{
	// read only
	"ping": true, "debug": true, "set_fact": true, "stat": true, "find": true, "add_host": true, "group_by": true,
	"fail": true, "wait_for": true, "assert": true, "include_vars": true,
	"slurp": true, "docker_host_info": true, "setup": true, "gather_facts": true, "getent": true, "async_status": true, "verify": true,
	// compare, then change
	"file": true, "copy": true, "template": true, "lineinfile": true, "blockinfile": true,
	"apt": true, "yum": true, "package": true, "service": true, "user": true, "group": true,
	"cron": true, "sysctl": true, "get_url": true, "git": true, "systemd": true,
	"mount": true, "config": true, "replace": true, "timezone": true, "unarchive": true, "apt_repository": true, "apt_key": true, "docker_container": true, "podman": true, "docker_image": true,
	"hostname": true, "ini_file": true, "pip": true, "ufw": true,
	"win_regedit": true, "win_powershell": true, "win_file": true, "win_copy": true, "win_service": true, "win_timezone": true,
	"win_firewall_rule": true, "win_firewall": true, "win_group_membership": true, "win_feature": true, "win_reboot": true, "win_scheduled_task": true, "win_chocolatey": true,
	"win_optional_feature": true, "win_disk_facts": true, "win_initialize_disk": true, "win_partition": true, "win_format": true,
}

// dataArgModules take their arguments as data whose types are kept:
// set_fact stores them, config writes them into JSON/YAML/TOML files
var dataArgModules = map[string]bool{"add_host": true, "set_fact": true, "config": true, "debug": true, "assert": true, "include_vars": true}

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

// executeRecovering runs a module; a panic in it fails this task only, with
// the panic in the error, instead of ending the whole run (and leaving the
// managed state locked)
func executeRecovering(ctx context.Context, module types.Module, host types.Host, args map[string]interface{}, task *types.Task) (result types.TaskResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			if len(stack) > 2048 {
				stack = stack[:2048]
			}
			result = types.TaskResult{TaskName: task.Name, Host: host.Name, Module: task.Module,
				Failed: true, Timestamp: time.Now(),
				Error: fmt.Sprintf("module %s crashed: %v (please report this bug)\n%s", task.Module, r, stack)}
			err = fmt.Errorf("module %s crashed: %v", task.Module, r)
		}
	}()
	return module.Execute(ctx, host, args)
}

// resolveDynamicAction reads the module of an action whose module name was
// a template, now that its arguments are rendered
func resolveDynamicAction(task *types.Task) (*types.Task, error) {
	resolved := *task
	if line, ok := task.Args["_action"].(string); ok {
		name, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
		args, err := types.ShortFormArgs(name, strings.TrimSpace(rest))
		if err != nil {
			return nil, fmt.Errorf("action: %w", err)
		}
		resolved.Module, resolved.Args = types.ShortModuleName(name), args
		types.CanonicalArgs(resolved.Module, resolved.Args)
		return &resolved, nil
	}
	name, _ := task.Args["_module"].(string)
	if name == "" || strings.Contains(name, "{{") {
		return nil, fmt.Errorf("action: the module name did not render (%q)", name)
	}
	resolved.Module = types.ShortModuleName(strings.TrimSpace(name))
	resolved.Args = make(map[string]interface{}, len(task.Args))
	for k, v := range task.Args {
		if k != "_module" {
			resolved.Args[k] = v
		}
	}
	types.CanonicalArgs(resolved.Module, resolved.Args)
	return &resolved, nil
}

// runBridged runs a module onigirazu lacks through ansible-core
func runBridged(ctx context.Context, task *types.Task, host types.Host) types.TaskResult {
	env := make(map[string]interface{}, len(task.Environment))
	for k, v := range task.Environment {
		env[k] = v
	}
	return bridge.Run(ctx, bridge.Task{
		Name: task.Name, Module: task.Module, Args: task.Args,
		Check: task.CheckMode != nil && *task.CheckMode, Diff: task.Diff,
		Become: task.Become, BecomeUser: task.BecomeUser, BecomeMethod: task.BecomeMethod,
		NoLog: task.NoLog, Environment: env,
	}, host)
}

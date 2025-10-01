package modules

import (
	"context"
	"fmt"

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
	registry.RegisterModule(NewFileModule())
	registry.RegisterModule(NewCopyModule())
	registry.RegisterModule(NewFetchModule())
	registry.RegisterModule(NewServiceModule())
	registry.RegisterModule(NewPackageModule())
	registry.RegisterModule(NewEnhancedPackageModule()) // Enhanced package module
	registry.RegisterModule(NewCommandModule())
	registry.RegisterModule(NewShellModule())
	registry.RegisterModule(NewUserModule())
	registry.RegisterModule(NewGroupModule())
	registry.RegisterModule(NewTemplateModule())
	registry.RegisterModule(NewGitModule())
	registry.RegisterModule(NewDebugModule())
	registry.RegisterModule(NewSetFactModule())
	registry.RegisterModule(NewStatModule())
	registry.RegisterModule(NewLineinfileModule())

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

	// Add task name only if not already specified in args
	if _, exists := args["name"]; !exists {
		args["name"] = task.Name
	}

	// Add variables to args
	for key, value := range variables {
		if _, exists := args[key]; !exists {
			args[key] = value
		}
	}

	return module.Execute(ctx, host, args)
}

package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/validator"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"gopkg.in/yaml.v3"
)

// RoleLoader handles loading roles from the filesystem
type RoleLoader struct {
	logger      interfaces.Logger
	rolesPath   string
	cache       map[string]*types.Role
	loadedRoles map[string]bool // For dependency tracking
}

// NewRoleLoader creates a new RoleLoader instance
func NewRoleLoader(logger interfaces.Logger, rolesPath string) *RoleLoader {
	return &RoleLoader{
		logger:      logger,
		rolesPath:   rolesPath,
		cache:       make(map[string]*types.Role),
		loadedRoles: make(map[string]bool),
	}
}

// LoadRole loads a role from filesystem with caching and dependency resolution
func (rl *RoleLoader) LoadRole(ctx context.Context, roleRef types.RoleReference) (*types.Role, error) {
	// Check cache first
	if cached, ok := rl.cache[roleRef.Name]; ok {
		return cached, nil
	}

	// Determine role path
	rolePath := roleRef.Path
	if rolePath == "" {
		rolePath = filepath.Join(rl.rolesPath, roleRef.Name)
	}

	// Verify role directory exists
	if _, err := os.Stat(rolePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("role not found: %s (searched in %s)", roleRef.Name, rolePath)
	}

	role := &types.Role{
		Name:      roleRef.Name,
		Path:      rolePath,
		Tasks:     []types.Task{},
		Handlers:  []types.Task{}, // Using Task type as Handler type not in original plan
		Defaults:  make(map[string]interface{}),
		Vars:      make(map[string]interface{}),
		Files:     make(map[string]string),
		Templates: make(map[string]string),
	}

	// Load components in order
	if err := rl.loadTasks(ctx, role); err != nil {
		return nil, fmt.Errorf("error loading role tasks for %s: %w", roleRef.Name, err)
	}

	if err := rl.loadHandlers(ctx, role); err != nil {
		return nil, fmt.Errorf("error loading role handlers for %s: %w", roleRef.Name, err)
	}

	if err := rl.loadDefaults(ctx, role); err != nil {
		return nil, fmt.Errorf("error loading role defaults for %s: %w", roleRef.Name, err)
	}

	if err := rl.loadVars(ctx, role); err != nil {
		return nil, fmt.Errorf("error loading role vars for %s: %w", roleRef.Name, err)
	}

	if err := rl.loadFiles(role); err != nil {
		return nil, fmt.Errorf("error loading role files for %s: %w", roleRef.Name, err)
	}

	if err := rl.loadTemplates(role); err != nil {
		return nil, fmt.Errorf("error loading role templates for %s: %w", roleRef.Name, err)
	}

	if err := rl.loadMeta(ctx, role); err != nil {
		return nil, fmt.Errorf("error loading role meta for %s: %w", roleRef.Name, err)
	}

	// Handle role dependencies
	if err := rl.loadDependencies(ctx, role); err != nil {
		return nil, fmt.Errorf("error loading role dependencies for %s: %w", roleRef.Name, err)
	}

	// Cache the role
	rl.cache[roleRef.Name] = role

	rl.logger.Info("Loaded role: %s from %s", roleRef.Name, rolePath)
	return role, nil
}

// ValidateAndApplyParameters validates role parameters and applies defaults
// Returns ValidationResult with any errors found
func (rl *RoleLoader) ValidateAndApplyParameters(role *types.Role, roleRef types.RoleReference) *types.ValidationResult {
	// If role has no parameter schema, skip validation
	if len(role.Meta.Parameters) == 0 {
		rl.logger.Debug("Role %s has no parameter schema, skipping validation", role.Name)
		return &types.ValidationResult{
			Valid:            true,
			Errors:           []types.ParameterValidationError{},
			CrossParamErrors: []types.CrossParameterValidationError{},
		}
	}

	// Create validator with parameter schema
	pv := validator.NewParameterValidator(role.Meta.Parameters)

	// Get variables to validate (roleRef.Vars takes priority over role defaults)
	varsToValidate := roleRef.Vars
	if varsToValidate == nil {
		varsToValidate = make(map[string]interface{})
	}

	// Validate parameters (basic type validation)
	result := pv.ValidateParameters(varsToValidate)

	// If basic validation fails, log and return errors
	if !result.Valid {
		for _, err := range result.Errors {
			rl.logger.Error("Parameter validation failed for role %s: %s parameter: %s", role.Name, err.Parameter, err.Error)
		}
		return result
	}

	// Merge with defaults and apply to role
	mergedVars := pv.MergeWithDefaults(varsToValidate)

	// Validate cross-parameter rules if defined
	if len(role.Meta.CrossParameterRules) > 0 {
		rl.logger.Debug("Role %s has %d cross-parameter rules, validating...", role.Name, len(role.Meta.CrossParameterRules))
		crossParamResult := pv.ValidateCrossParameters(mergedVars, role.Meta.CrossParameterRules)

		if !crossParamResult.Valid {
			for _, err := range crossParamResult.CrossParamErrors {
				rl.logger.Error("Cross-parameter validation failed for role %s: %s", role.Name, err.ErrorMsg)
			}
			// Merge cross-param errors into main result
			result.Valid = false
			result.CrossParamErrors = crossParamResult.CrossParamErrors
			return result
		}
	}

	// Apply merged variables to role Vars
	for key, value := range mergedVars {
		role.Vars[key] = value
	}

	rl.logger.Info("Role %s parameters validated successfully (including cross-parameter rules). Applied %d variables", role.Name, len(mergedVars))
	return result
}

// GetParameterSchema returns the parameter schema for a role
// This is useful for debugging and documentation generation
func (rl *RoleLoader) GetParameterSchema(role *types.Role) string {
	if len(role.Meta.Parameters) == 0 {
		return fmt.Sprintf("Role %s has no parameters defined", role.Name)
	}

	pv := validator.NewParameterValidator(role.Meta.Parameters)
	return pv.GetParameterDescription()
}

// loadTasks loads tasks from the role's tasks/main.yml file
func (rl *RoleLoader) loadTasks(ctx context.Context, role *types.Role) error {
	mainTasksPath := filepath.Join(role.Path, "tasks", "main.yml")

	data, err := os.ReadFile(mainTasksPath)
	if os.IsNotExist(err) {
		// tasks/main.yml is optional
		return nil
	}
	if err != nil {
		return err
	}

	var tasks []types.Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("error parsing tasks: %w", err)
	}

	role.Tasks = tasks
	return nil
}

// loadHandlers loads handlers from the role's handlers/main.yml file
func (rl *RoleLoader) loadHandlers(ctx context.Context, role *types.Role) error {
	handlersPath := filepath.Join(role.Path, "handlers", "main.yml")

	data, err := os.ReadFile(handlersPath)
	if os.IsNotExist(err) {
		// handlers/main.yml is optional
		return nil
	}
	if err != nil {
		return err
	}

	var handlers []types.Task
	if err := yaml.Unmarshal(data, &handlers); err != nil {
		return fmt.Errorf("error parsing handlers: %w", err)
	}

	role.Handlers = handlers
	return nil
}

// loadDefaults loads default variables from the role's defaults/main.yml file
func (rl *RoleLoader) loadDefaults(ctx context.Context, role *types.Role) error {
	defaultsPath := filepath.Join(role.Path, "defaults", "main.yml")

	data, err := os.ReadFile(defaultsPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &role.Defaults); err != nil {
		return fmt.Errorf("error parsing defaults: %w", err)
	}

	return nil
}

// loadVars loads variables from the role's vars/main.yml file
func (rl *RoleLoader) loadVars(ctx context.Context, role *types.Role) error {
	varsPath := filepath.Join(role.Path, "vars", "main.yml")

	data, err := os.ReadFile(varsPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &role.Vars); err != nil {
		return fmt.Errorf("error parsing vars: %w", err)
	}

	return nil
}

// loadFiles loads static files from the role's files/ directory
func (rl *RoleLoader) loadFiles(role *types.Role) error {
	filesPath := filepath.Join(role.Path, "files")

	entries, err := os.ReadDir(filesPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(filesPath, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			role.Files[entry.Name()] = string(data)
		}
	}

	return nil
}

// loadTemplates loads Jinja2 templates from the role's templates/ directory
func (rl *RoleLoader) loadTemplates(role *types.Role) error {
	templatesPath := filepath.Join(role.Path, "templates")

	entries, err := os.ReadDir(templatesPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(templatesPath, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			role.Templates[entry.Name()] = string(data)
		}
	}

	return nil
}

// loadMeta loads role metadata from meta/main.yml file
func (rl *RoleLoader) loadMeta(ctx context.Context, role *types.Role) error {
	metaPath := filepath.Join(role.Path, "meta", "main.yml")

	data, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &role.Meta); err != nil {
		return fmt.Errorf("error parsing meta: %w", err)
	}

	return nil
}

// loadDependencies recursively loads role dependencies with cycle detection
func (rl *RoleLoader) loadDependencies(ctx context.Context, role *types.Role) error {
	visited := make(map[string]bool)
	stack := make([]string, 0)

	return rl.loadDependenciesRecursive(ctx, role, visited, stack)
}

// loadDependenciesRecursive recursively loads role dependencies with cycle detection
func (rl *RoleLoader) loadDependenciesRecursive(ctx context.Context, role *types.Role, visited map[string]bool, stack []string) error {
	// Check for cycles
	for _, s := range stack {
		if s == role.Name {
			return fmt.Errorf("circular dependency detected: %s -> ... -> %s", stack[0], role.Name)
		}
	}

	stack = append(stack, role.Name)

	for _, dep := range role.Meta.Dependencies {
		if visited[dep.Name] {
			continue // Already processed
		}

		depRef := types.RoleReference{
			Name: dep.Name,
			Vars: dep.Vars,
		}

		depRole, err := rl.LoadRole(ctx, depRef)
		if err != nil {
			return fmt.Errorf("error loading dependency %s: %w", dep.Name, err)
		}

		visited[dep.Name] = true
		rl.loadedRoles[dep.Name] = true

		// Recursively load dependencies of dependencies
		if err := rl.loadDependenciesRecursive(ctx, depRole, visited, stack); err != nil {
			return err
		}
	}

	return nil
}

// ResolveDependencyOrder resolves role dependencies in correct execution order (topological sort)
// Returns an ordered list of roles starting with all dependencies, ending with the main role
func (rl *RoleLoader) ResolveDependencyOrder(ctx context.Context, role *types.Role) ([]*types.Role, error) {
	visited := make(map[string]bool)
	order := make([]*types.Role, 0)

	if err := rl.resolveDependencyOrderRecursive(ctx, role, visited, &order); err != nil {
		return nil, err
	}

	// Add the main role at the end
	order = append(order, role)

	return order, nil
}

// resolveDependencyOrderRecursive recursively resolves dependency order with cycle detection
func (rl *RoleLoader) resolveDependencyOrderRecursive(ctx context.Context, role *types.Role, visited map[string]bool, order *[]*types.Role) error {
	// Skip if already visited (already in order)
	if visited[role.Name] {
		return nil
	}

	visited[role.Name] = true

	// First, add all dependencies in order
	for _, dep := range role.Meta.Dependencies {
		if !visited[dep.Name] {
			depRef := types.RoleReference{
				Name: dep.Name,
				Vars: dep.Vars,
			}

			depRole, err := rl.LoadRole(ctx, depRef)
			if err != nil {
				return fmt.Errorf("error loading dependency %s: %w", dep.Name, err)
			}

			// Recursively resolve this dependency's dependencies
			if err := rl.resolveDependencyOrderRecursive(ctx, depRole, visited, order); err != nil {
				return err
			}

			// Add the dependency to the order
			*order = append(*order, depRole)
		}
	}

	return nil
}

// MergeVariables merges role defaults, role variables, and override variables with proper priority
// Priority (highest to lowest): RoleVars > OverrideVars > PlayVars > Defaults
func (rl *RoleLoader) MergeVariables(
	roleDefaults map[string]interface{},
	roleVars map[string]interface{},
	playVars map[string]interface{},
	overrideVars map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})

	// Start with defaults (lowest priority)
	for k, v := range roleDefaults {
		result[k] = v
	}

	// Override with play vars
	for k, v := range playVars {
		result[k] = v
	}

	// Override with override vars
	for k, v := range overrideVars {
		result[k] = v
	}

	// Override with role vars (highest priority)
	for k, v := range roleVars {
		result[k] = v
	}

	return result
}

// ClearCache clears the role cache
func (rl *RoleLoader) ClearCache() {
	rl.cache = make(map[string]*types.Role)
}

// GetCache returns the role cache
func (rl *RoleLoader) GetCache() map[string]*types.Role {
	return rl.cache
}

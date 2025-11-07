# Module Scaffolding Guide

## Overview

Module Scaffolding is a powerful code generation tool that dramatically accelerates Onigirazu module development. It automatically generates production-ready boilerplate code with best practices built-in.

**Time Saved**: Generate a fully-tested module structure in **seconds** instead of spending **hours** writing boilerplate!

## Key Benefits

- ⚡ **Instant Setup**: Generate complete module structure with one command
- ✅ **Best Practices**: Production-ready code following Onigirazu patterns
- 📝 **Comprehensive Tests**: Unit tests, idempotency tests, and benchmarks included
- 🛡️ **Type-Safe**: Proper imports, interfaces, and Go conventions
- 📚 **Documented**: Template includes JSDoc-style comments
- 🎯 **100% Coverage Ready**: Structured for achieving 100% test coverage

## What Gets Generated

Running the scaffolding tool creates:

1. **`MODULE_NAME.go`** - Module implementation with:
   - Package declaration and imports
   - Struct with BaseModule embedding
   - Constructor function
   - Validation method
   - Execute method (with TODO)
   - IsIdempotent method

2. **`MODULE_NAME_test.go`** - Comprehensive tests:
   - Construction tests
   - Validation tests
   - Idempotency checks
   - Execute tests with table-driven approach
   - Benchmark tests
   - Mock objects

3. **`MODULE_NAME_idempotency_test.go`** - Dedicated idempotency tests:
   - Double-execution verification
   - State consistency checks
   - Result comparison helpers

## Quick Start

### 1. Basic Module (No Parameters)

```bash
cd /Users/denys.rastiegaiev/work/onigirazu_project/onigirazu
go run ./scripts/module_scaffold -name hello_world
```

**Output**:

```
✅ Module scaffold generated successfully!

📋 Module Details:
   Name:        hello_world
   Package:     modules
   Parameters:  0
   Idempotent:  true

📁 Generated Files:
   ✓ internal/modules/hello_world.go
   ✓ internal/modules/hello_world_test.go
   ✓ internal/modules/hello_world_idempotency_test.go
```

### 2. Module with Parameters

```bash
go run ./scripts/module_scaffold \
  -name package_manager \
  -desc "Manage system packages" \
  -params "name,state,version"
```

This generates:

- `name`, `state`, `version` as struct fields
- Parameter validation in `Validate()` method
- Proper type annotations

### 3. Minimal Module (No Idempotency Tests)

```bash
go run ./scripts/module_scaffold \
  -name quick_module \
  -idempotent=false
```

## Command Reference

### Flags

```
-name string (REQUIRED)
    Module name (lowercase with underscores)
    Example: my_awesome_module, hello_world

-desc string (optional)
    Human-readable description
    Example: "Manage system packages"

-params string (optional)
    Comma-separated parameter names
    Example: "target,action,state"
    Each parameter becomes a struct field

-output string (default: internal/modules)
    Output directory for generated files
    Example: /custom/path

-idempotent bool (default: true)
    Generate idempotency test file
    Use -idempotent=false to skip

-help
    Show help and usage examples
```

### Usage Pattern

```bash
go run ./scripts/module_scaffold \
  -name MODULE_NAME \
  [-desc "description"] \
  [-params "param1,param2,param3"] \
  [-output /path] \
  [-idempotent=true|false]
```

## Complete Examples

### Example 1: Service Manager Module

```bash
go run ./scripts/module_scaffold \
  -name service_manager \
  -desc "Start, stop, and restart system services" \
  -params "name,state,enabled,restart"
```

**Generated struct**:

```go
type ServiceManagerModule struct {
    BaseModule
    Name     string
    State    string
    Enabled  string
    Restart  string
}
```

### Example 2: File Permissions Module

```bash
go run ./scripts/module_scaffold \
  -name file_permissions \
  -desc "Manage file permissions and ownership" \
  -params "path,mode,owner,group"
```

### Example 3: Custom Output Location

```bash
go run ./scripts/module_scaffold \
  -name custom_module \
  -output /home/user/my_modules \
  -desc "Custom module implementation"
```

## Module Implementation Workflow

### Step 1: Generate Scaffold

```bash
go run ./scripts/module_scaffold -name my_module -params "target,action"
```

### Step 2: Examine Generated Files

```bash
cat internal/modules/my_module.go
cat internal/modules/my_module_test.go
cat internal/modules/my_module_idempotency_test.go
```

### Step 3: Implement Execute Method

Edit `internal/modules/my_module.go`:

```go
func (m *MyModuleModule) Execute(task *types.TaskDefinition) (interface{}, error) {
    // Extract parameters
    params := task.Args.(map[string]interface{})
    target := params["target"].(string)
    action := params["action"].(string)

    // Validate parameters
    if target == "" {
        return nil, fmt.Errorf("target parameter is required")
    }

    // Determine current state
    currentState, err := m.getCurrentState(target)
    if err != nil {
        return nil, fmt.Errorf("failed to get current state: %w", err)
    }

    // Apply desired state if needed
    changed := false
    if currentState != action {
        if err := m.applyState(target, action); err != nil {
            return nil, fmt.Errorf("failed to apply state: %w", err)
        }
        changed = true
    }

    return types.ModuleResult{
        Changed: changed,
        Msg:     fmt.Sprintf("State updated: %s -> %s", currentState, action),
    }, nil
}

// Helper methods
func (m *MyModuleModule) getCurrentState(target string) (string, error) {
    // TODO: Implement state detection
    return "unknown", nil
}

func (m *MyModuleModule) applyState(target string, state string) error {
    // TODO: Implement state application
    return nil
}
```

### Step 4: Update Validate Method

```go
func (m *MyModuleModule) Validate(task *types.TaskDefinition) error {
    if err := m.BaseModule.Validate(task); err != nil {
        return err
    }

    params := task.Args.(map[string]interface{})

    // Validate required parameters
    if params["target"] == "" {
        return fmt.Errorf("target parameter is required")
    }

    if params["action"] == "" {
        return fmt.Errorf("action parameter is required")
    }

    // Validate parameter values
    action := params["action"].(string)
    validActions := []string{"start", "stop", "restart"}
    validAction := false
    for _, v := range validActions {
        if action == v {
            validAction = true
            break
        }
    }
    if !validAction {
        return fmt.Errorf("action must be one of: %v", validActions)
    }

    return nil
}
```

### Step 5: Write Comprehensive Tests

```go
func TestModule_MyModuleExecute(t *testing.T) {
    tests := []struct {
        name      string
        task      *types.TaskDefinition
        expectErr bool
        check     func(t *testing.T, result interface{})
    }{
        {
            name: "apply state successfully",
            task: &types.TaskDefinition{
                Name: "Apply my action",
                Args: map[string]interface{}{
                    "target": "/etc/myconf",
                    "action": "start",
                },
            },
            expectErr: false,
            check: func(t *testing.T, result interface{}) {
                r := result.(types.ModuleResult)
                assert.True(t, r.Changed)
                assert.Contains(t, r.Msg, "start")
            },
        },
        {
            name: "no change when already applied",
            task: &types.TaskDefinition{
                Name: "Already applied",
                Args: map[string]interface{}{
                    "target": "/etc/myconf",
                    "action": "start",
                },
            },
            expectErr: false,
            check: func(t *testing.T, result interface{}) {
                r := result.(types.ModuleResult)
                assert.False(t, r.Changed)
            },
        },
    }

    executor := NewMockExecutor()
    module := NewMyModuleModule(executor)

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := module.Execute(tt.task)
            if tt.expectErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                if tt.check != nil {
                    tt.check(t, result)
                }
            }
        })
    }
}
```

### Step 6: Run Tests

```bash
# Run all tests for your module
go test ./internal/modules -run MyModule -v

# Run with coverage
go test ./internal/modules -run MyModule -cover

# Run idempotency tests only
go test ./internal/modules -run MyModule_Idempotency -v

# Run benchmarks
go test ./internal/modules -bench=MyModule -benchmem
```

### Step 7: Check Coverage

```bash
# Generate coverage report
go test ./internal/modules -run MyModule -coverprofile=coverage.out

# View coverage in HTML
go tool cover -html=coverage.out
```

**Goal**: Achieve **100% coverage** for new modules!

## Best Practices for Scaffolded Modules

### ✅ Parameter Handling

**DO**:

```go
// Extract and validate
params := task.Args.(map[string]interface{})
target := params["target"].(string)
if target == "" {
    return nil, fmt.Errorf("target parameter is required")
}
```

**DON'T**:

```go
// Unsafe extraction without validation
target := task.Args.(map[string]interface{})["target"].(string)
```

### ✅ Idempotency

**DO**:

```go
// Check current state first
current, _ := m.getCurrentState(target)
if current == desired {
    return types.ModuleResult{Changed: false}, nil
}
// Apply only if needed
_ = m.applyState(target, desired)
return types.ModuleResult{Changed: true}, nil
```

**DON'T**:

```go
// Always apply changes
_ = m.applyState(target, desired)
return types.ModuleResult{Changed: true}, nil
```

### ✅ Error Handling

**DO**:

```go
if err != nil {
    return nil, fmt.Errorf("failed to execute %s: %w", action, err)
}
```

**DON'T**:

```go
if err != nil {
    return nil, err  // Lost context!
}
```

### ✅ Logging

```go
// Use BaseModule.Logger for structured logging
m.Logger.Debugf("Executing action %s on %s", action, target)
m.Logger.Infof("Successfully applied state")
m.Logger.Warnf("Deprecated parameter: %s", param)
```

### ✅ Testing

**DO**:

- Use table-driven tests for multiple cases
- Test both success and failure paths
- Mock external dependencies
- Test idempotency explicitly
- Add benchmarks
- Aim for 100% coverage

**DON'T**:

- Skip error cases
- Hardcode test data
- Use system resources in tests
- Skip coverage measurement
- Write a single large test function

## Module Structure Reference

### Generated Module File Structure

```
internal/modules/
├── my_module.go                    # Module implementation
├── my_module_test.go               # Unit tests
└── my_module_idempotency_test.go   # Idempotency tests
```

### Typical Module Implementation

```go
package modules

type MyModuleModule struct {
    BaseModule
    // Parameters from -params flag
    Param1 string
    Param2 string
}

// Constructor
func NewMyModuleModule(executor interfaces.Executor) *MyModuleModule { ... }

// Validation
func (m *MyModuleModule) Validate(task *types.TaskDefinition) error { ... }

// Execution
func (m *MyModuleModule) Execute(task *types.TaskDefinition) (interface{}, error) { ... }

// Idempotency check
func (m *MyModuleModule) IsIdempotent() bool { ... }
```

## Testing Generated Modules

### Test File Organization

The scaffolding tool generates tests in three categories:

#### 1. Construction Tests

```go
func TestNewMyModuleModule(t *testing.T) {
    executor := NewMockExecutor()
    module := NewMyModuleModule(executor)

    assert.NotNil(t, module)
    assert.Equal(t, "my_module", module.Name)
}
```

#### 2. Validation Tests

```go
func TestModule_MyModuleValidate(t *testing.T) {
    tests := []struct {
        name      string
        task      *types.TaskDefinition
        expectErr bool
    }{
        {
            name: "valid parameters",
            task: &types.TaskDefinition{...},
            expectErr: false,
        },
        {
            name: "missing required parameter",
            task: &types.TaskDefinition{...},
            expectErr: true,
        },
    }
    // Test implementation
}
```

#### 3. Execution Tests

```go
func TestModule_MyModuleExecute(t *testing.T) {
    tests := []struct {
        name      string
        task      *types.TaskDefinition
        check     func(t *testing.T, result interface{})
    }{
        {
            name: "successful execution",
            task: &types.TaskDefinition{...},
            check: func(t *testing.T, result interface{}) {
                assert.NotNil(t, result)
            },
        },
    }
    // Test implementation
}
```

#### 4. Idempotency Tests

```go
func TestModule_MyModuleIdempotency(t *testing.T) {
    // Run twice, verify same result
    result1, _ := module.Execute(task)
    result2, _ := module.Execute(task)
    // Compare results
}
```

## Troubleshooting

### Invalid Module Name

```
❌ Error: invalid module name: MyModule (must be lowercase with underscores)
```

**Solution**: Use lowercase with underscores only

```bash
✅ CORRECT:
go run ./scripts/module_scaffold -name my_module
go run ./scripts/module_scaffold -name hello_world

❌ WRONG:
go run ./scripts/module_scaffold -name MyModule
go run ./scripts/module_scaffold -name MY_MODULE
go run ./scripts/module_scaffold -name my-module
```

### Files Already Exist

```
Error: file already exists
```

**Solution**: Use different module name or backup/remove existing files

```bash
# Backup existing module
mv internal/modules/my_module.go internal/modules/my_module.go.bak

# Now generate with same name
go run ./scripts/module_scaffold -name my_module
```

### Parameter Parsing Issues

If parameters aren't being parsed correctly, check:

1. Use comma-separated format: `-params "name,value,state"`
2. Don't add spaces around commas
3. Each parameter must be a valid Go identifier

```bash
✅ CORRECT:
go run ./scripts/module_scaffold -name my_module -params "target,action,state"

❌ WRONG:
go run ./scripts/module_scaffold -name my_module -params "target, action, state"
go run ./scripts/module_scaffold -name my_module -params "target|action|state"
```

## Next Steps After Scaffolding

1. ✅ **Review** the generated files
2. ✅ **Implement** the Execute() method with your logic
3. ✅ **Update** the Validate() method with parameter validation
4. ✅ **Write** comprehensive test cases
5. ✅ **Run** tests and achieve 100% coverage
6. ✅ **Test** with actual playbooks
7. ✅ **Document** parameter usage in module documentation
8. ✅ **Submit** pull request following contribution guidelines

## Integration with Module System

Scaffolded modules automatically integrate with:

- ✅ Module registry and discovery
- ✅ Parameter validation framework
- ✅ Error handling system
- ✅ Logging infrastructure
- ✅ Metrics collection
- ✅ State management
- ✅ Execution engine

No additional setup needed!

## Advanced Topics

### Custom Validators

```go
func (m *MyModuleModule) Validate(task *types.TaskDefinition) error {
    if err := m.BaseModule.Validate(task); err != nil {
        return err
    }

    // Custom validation logic
    params := task.Args.(map[string]interface{})

    // Example: Validate enum values
    state := params["state"].(string)
    validStates := []string{"present", "absent"}
    isValid := false
    for _, s := range validStates {
        if state == s {
            isValid = true
            break
        }
    }
    if !isValid {
        return fmt.Errorf("invalid state: %s, must be one of %v", state, validStates)
    }

    return nil
}
```

### Working with Executors

```go
func (m *MyModuleModule) Execute(task *types.TaskDefinition) (interface{}, error) {
    // Execute a command via the executor
    cmd := "echo 'Hello World'"
    output, err := m.BaseModule.Executor.Execute(cmd)
    if err != nil {
        return nil, fmt.Errorf("command failed: %w", err)
    }

    return types.ModuleResult{
        Changed: false,
        Msg:     output,
    }, nil
}
```

### Complex Return Values

```go
type MyModuleResult struct {
    Changed bool                   `json:"changed"`
    Msg     string                `json:"msg"`
    Data    map[string]interface{} `json:"data"`
}

func (m *MyModuleModule) Execute(task *types.TaskDefinition) (interface{}, error) {
    return MyModuleResult{
        Changed: true,
        Msg:     "Operation successful",
        Data: map[string]interface{}{
            "key": "value",
        },
    }, nil
}
```

## Summary

Module Scaffolding transforms module development from a multi-hour task into a seconds-long setup. The generated code follows all Onigirazu best practices, includes comprehensive tests, and provides a solid foundation for any module implementation.

**Start using the scaffolding tool today** to accelerate your module development workflow!

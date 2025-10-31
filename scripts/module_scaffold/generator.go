package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModuleScaffold contains the template and metadata for a new module
type ModuleScaffold struct {
	ModuleName        string
	PackageName       string
	Description       string
	Parameters        []string
	OutputDir         string
	IncludeIdempotent bool
}

// Generate creates the scaffold files for a new module
func (ms *ModuleScaffold) Generate() error {
	if err := ms.validate(); err != nil {
		return err
	}

	if err := ms.createModuleFile(); err != nil {
		return fmt.Errorf("failed to create module file: %w", err)
	}

	if err := ms.createTestFile(); err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}

	if ms.IncludeIdempotent {
		if err := ms.createIdempotentTestFile(); err != nil {
			return fmt.Errorf("failed to create idempotent test file: %w", err)
		}
	}

	return nil
}

func (ms *ModuleScaffold) validate() error {
	if ms.ModuleName == "" {
		return fmt.Errorf("module name is required")
	}

	if ms.OutputDir == "" {
		ms.OutputDir = "/Users/denys.rastiegaiev/work/onigirazu_project/onigirazu/internal/modules"
	}

	ms.PackageName = "modules"

	// Validate module name (must be lowercase with underscores)
	if !isValidModuleName(ms.ModuleName) {
		return fmt.Errorf("invalid module name: %s (must be lowercase with underscores)", ms.ModuleName)
	}

	return nil
}

func (ms *ModuleScaffold) createModuleFile() error {
	content := ms.generateModuleContent()
	filePath := filepath.Join(ms.OutputDir, ms.ModuleName+".go")

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("✓ Created module file: %s\n", filePath)
	return nil
}

func (ms *ModuleScaffold) createTestFile() error {
	content := ms.generateTestContent()
	filePath := filepath.Join(ms.OutputDir, ms.ModuleName+"_test.go")

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("✓ Created test file: %s\n", filePath)
	return nil
}

func (ms *ModuleScaffold) createIdempotentTestFile() error {
	content := ms.generateIdempotentTestContent()
	filePath := filepath.Join(ms.OutputDir, ms.ModuleName+"_idempotency_test.go")

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("✓ Created idempotency test file: %s\n", filePath)
	return nil
}

func (ms *ModuleScaffold) generateModuleContent() string {
	parameters := ms.generateParameterFields()
	parameterValidation := ms.generateParameterValidation()

	return fmt.Sprintf(`package %s

import (
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// %sModule implements the %s module
type %sModule struct {
	BaseModule
%s}

// NewModule creates a new instance of the %s module
func New%sModule(executor interfaces.Executor) *%sModule {
	return &%sModule{
		BaseModule: BaseModule{
			Name:     "%s",
			Executor: executor,
		},
	}
}

// Validate validates the module parameters
func (m *%sModule) Validate(task *types.TaskDefinition) error {
	if err := m.BaseModule.Validate(task); err != nil {
		return err
	}

%s
	return nil
}

// Execute runs the %s module
func (m *%sModule) Execute(task *types.TaskDefinition) (interface{}, error) {
	// TODO: Implement module execution logic
	// 1. Extract and validate parameters
	// 2. Determine current state
	// 3. Apply desired state if needed
	// 4. Return result with changed flag

	result := types.ModuleResult{
		Changed: false,
		Msg:     "Module executed successfully",
	}

	return result, nil
}

// IsIdempotent returns whether the module is idempotent
func (m *%sModule) IsIdempotent() bool {
	return true
}
`,
		ms.PackageName,
		capitalize(ms.ModuleName),
		ms.ModuleName,
		capitalize(ms.ModuleName),
		parameters,
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		ms.ModuleName,
		capitalize(ms.ModuleName),
		parameterValidation,
		ms.ModuleName,
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
	)
}

func (ms *ModuleScaffold) generateTestContent() string {
	testCases := ms.generateTestCases()

	return fmt.Sprintf(`package %s

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew%sModule(t *testing.T) {
	executor := NewMockExecutor()
	module := New%sModule(executor)

	assert.NotNil(t, module)
	assert.Equal(t, "%s", module.Name)
	assert.NotNil(t, module.Executor)
}

func TestModule_%sValidate(t *testing.T) {
	tests := []struct {
		name      string
		task      *types.TaskDefinition
		expectErr bool
		errMsg    string
	}{
%s
	}

	executor := NewMockExecutor()
	module := New%sModule(executor)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.task)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModule_%sIsIdempotent(t *testing.T) {
	executor := NewMockExecutor()
	module := New%sModule(executor)

	assert.True(t, module.IsIdempotent())
}

func TestModule_%sExecute(t *testing.T) {
	tests := []struct {
		name      string
		task      *types.TaskDefinition
		expectErr bool
		checkResult func(t *testing.T, result interface{})
	}{
		{
			name: "successful execution",
			task: &types.TaskDefinition{
				Name: "test task",
			},
			expectErr: false,
			checkResult: func(t *testing.T, result interface{}) {
				// Verify result structure and values
				assert.NotNil(t, result)
			},
		},
	}

	executor := NewMockExecutor()
	module := New%sModule(executor)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(tt.task)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func Benchmark%sModule(b *testing.B) {
	executor := NewMockExecutor()
	module := New%sModule(executor)
	task := &types.TaskDefinition{
		Name: "benchmark task",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(task)
	}
}
`,
		ms.PackageName,
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		ms.ModuleName,
		capitalize(ms.ModuleName),
		testCases,
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
	)
}

func (ms *ModuleScaffold) generateIdempotentTestContent() string {
	return fmt.Sprintf(`package %s

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestModule_%sIdempotency(t *testing.T) {
	// Test idempotency: running twice should produce same result
	tests := []struct {
		name     string
		task     *types.TaskDefinition
		validate func(t *testing.T, first, second interface{})
	}{
		{
			name: "idempotent operation",
			task: &types.TaskDefinition{
				Name: "test task",
			},
			validate: func(t *testing.T, first, second interface{}) {
				// TODO: Add idempotency assertion
				// Both executions should have Changed=false on second run
				assert.NotNil(t, first)
				assert.NotNil(t, second)
			},
		},
	}

	executor := NewMockExecutor()
	module := New%sModule(executor)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First execution
			result1, err := module.Execute(tt.task)
			assert.NoError(t, err)

			// Second execution (should be idempotent)
			result2, err := module.Execute(tt.task)
			assert.NoError(t, err)

			// Verify idempotency
			if tt.validate != nil {
				tt.validate(t, result1, result2)
			}
		})
	}
}
`,
		ms.PackageName,
		capitalize(ms.ModuleName),
		capitalize(ms.ModuleName),
	)
}

func (ms *ModuleScaffold) generateParameterFields() string {
	if len(ms.Parameters) == 0 {
		return ""
	}

	var fields []string
	for _, param := range ms.Parameters {
		fields = append(fields, fmt.Sprintf("\t%s string", capitalize(param)))
	}
	return "\n" + strings.Join(fields, "\n") + "\n"
}

func (ms *ModuleScaffold) generateParameterValidation() string {
	if len(ms.Parameters) == 0 {
		return "\t// No required parameters"
	}

	var validations []string
	for _, param := range ms.Parameters {
		validations = append(validations, fmt.Sprintf("\tif params[\"%s\"] == \"\" {\n\t\treturn fmt.Errorf(\"%s parameter is required\")\n\t}", param, param))
	}

	paramExtraction := fmt.Sprintf("\tparams := task.Args.(map[string]interface{})\n")
	return paramExtraction + strings.Join(validations, "\n")
}

func (ms *ModuleScaffold) generateTestCases() string {
	return `		{
			name: "valid parameters",
			task: &types.TaskDefinition{
				Name: "test task",
			},
			expectErr: false,
		},
		{
			name: "missing task",
			task: nil,
			expectErr: true,
			errMsg: "task is required",
		},`
}

func isValidModuleName(name string) bool {
	if name == "" {
		return false
	}

	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}

	// Convert snake_case to CamelCase
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

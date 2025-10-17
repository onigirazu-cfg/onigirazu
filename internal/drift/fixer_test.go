package drift

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// testModuleRegistry is a strict registry that returns error for missing modules
type testModuleRegistry struct {
	modules map[string]interfaces.ModuleExecutor
}

func (t *testModuleRegistry) Register(name string, module interfaces.ModuleExecutor) error {
	t.modules[name] = module
	return nil
}

func (t *testModuleRegistry) Get(name string) (interfaces.ModuleExecutor, error) {
	mod, ok := t.modules[name]
	if !ok {
		return nil, fmt.Errorf("module not found: %s", name)
	}
	return mod, nil
}

func (t *testModuleRegistry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error) {
	return types.TaskResult{Success: true}, nil
}

func (t *testModuleRegistry) List() []string {
	names := make([]string, 0, len(t.modules))
	for name := range t.modules {
		names = append(names, name)
	}
	return names
}

func (t *testModuleRegistry) Unregister(name string) error {
	delete(t.modules, name)
	return nil
}

func TestNewFixer(t *testing.T) {
	registry := newMockModuleRegistry()
	logger := &mockLogger{}
	config := &DriftConfig{
		AutoFix: true,
		DryRun:  false,
	}

	fixer := NewFixer(registry, logger, config)

	if fixer == nil {
		t.Fatal("Expected fixer to be created")
	}
}

func TestFixerFilterFixableItems(t *testing.T) {
	tests := []struct {
		name          string
		items         []DriftItem
		config        *DriftConfig
		expectedCount int
		description   string
	}{
		{
			name:  "empty items list",
			items: []DriftItem{},
			config: &DriftConfig{
				AutoFix: true,
			},
			expectedCount: 0,
			description:   "should return empty list for empty input",
		},
		{
			name: "all items can be fixed",
			items: []DriftItem{
				{
					ID:         "drift-1",
					Type:       DriftTypeFile,
					CanAutoFix: true,
					Severity:   SeverityHigh,
					FixOperation: &FixOperation{
						Module: "file",
						Args:   map[string]interface{}{},
						Order:  1,
					},
				},
				{
					ID:         "drift-2",
					Type:       DriftTypeService,
					CanAutoFix: true,
					Severity:   SeverityMedium,
					FixOperation: &FixOperation{
						Module: "service",
						Args:   map[string]interface{}{},
						Order:  2,
					},
				},
			},
			config: &DriftConfig{
				AutoFix:         true,
				AutoFixSeverity: []DriftSeverity{SeverityHigh, SeverityMedium},
			},
			expectedCount: 2,
			description:   "should return all items that can be fixed",
		},
		{
			name: "items without FixOperation are filtered",
			items: []DriftItem{
				{
					ID:         "drift-1",
					Type:       DriftTypeFile,
					CanAutoFix: true,
					Severity:   SeverityHigh,
				},
			},
			config: &DriftConfig{
				AutoFix: true,
			},
			expectedCount: 0,
			description:   "should filter items without FixOperation",
		},
		{
			name: "items with CanAutoFix=false are filtered",
			items: []DriftItem{
				{
					ID:         "drift-1",
					Type:       DriftTypeFile,
					CanAutoFix: false,
					Severity:   SeverityHigh,
					FixOperation: &FixOperation{
						Module: "file",
					},
				},
			},
			config: &DriftConfig{
				AutoFix: true,
			},
			expectedCount: 0,
			description:   "should filter items with CanAutoFix=false",
		},
		{
			name: "items with wrong severity are filtered",
			items: []DriftItem{
				{
					ID:         "drift-1",
					Type:       DriftTypeFile,
					CanAutoFix: true,
					Severity:   SeverityLow,
					FixOperation: &FixOperation{
						Module: "file",
					},
				},
			},
			config: &DriftConfig{
				AutoFix:         true,
				AutoFixSeverity: []DriftSeverity{SeverityHigh},
			},
			expectedCount: 0,
			description:   "should filter items not in AutoFixSeverity list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixer := NewFixer(newMockModuleRegistry(), &mockLogger{}, tt.config)
			result := fixer.filterFixableItems(tt.items)

			if len(result) != tt.expectedCount {
				t.Errorf("%s: expected %d items, got %d", tt.description, tt.expectedCount, len(result))
			}
		})
	}
}

func TestFixerShouldAutoFix(t *testing.T) {
	tests := []struct {
		name        string
		severity    DriftSeverity
		config      *DriftConfig
		expected    bool
		description string
	}{
		{
			name:        "AutoFix disabled",
			severity:    SeverityHigh,
			config:      &DriftConfig{AutoFix: false},
			expected:    false,
			description: "should return false when AutoFix is disabled",
		},
		{
			name:     "AutoFix enabled with empty severity list",
			severity: SeverityHigh,
			config: &DriftConfig{
				AutoFix:         true,
				AutoFixSeverity: []DriftSeverity{},
			},
			expected:    true,
			description: "should return true when AutoFix is enabled and severity list is empty",
		},
		{
			name:     "AutoFix enabled with matching severity",
			severity: SeverityHigh,
			config: &DriftConfig{
				AutoFix:         true,
				AutoFixSeverity: []DriftSeverity{SeverityHigh, SeverityMedium},
			},
			expected:    true,
			description: "should return true when severity matches",
		},
		{
			name:     "AutoFix enabled with non-matching severity",
			severity: SeverityLow,
			config: &DriftConfig{
				AutoFix:         true,
				AutoFixSeverity: []DriftSeverity{SeverityHigh},
			},
			expected:    false,
			description: "should return false when severity doesn't match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixer := NewFixer(newMockModuleRegistry(), &mockLogger{}, tt.config)
			result := fixer.shouldAutoFix(tt.severity)

			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestFixDriftDryRun(t *testing.T) {
	config := &DriftConfig{
		AutoFix:         true,
		DryRun:          true,
		AutoFixSeverity: []DriftSeverity{SeverityHigh},
	}

	fixer := NewFixer(newMockModuleRegistry(), &mockLogger{}, config)

	now := time.Now()
	report := &DriftReport{
		ID:          "report-1",
		Timestamp:   now,
		TotalDrifts: 1,
		Items: []DriftItem{
			{
				ID:         "drift-1",
				Type:       DriftTypeFile,
				Resource:   "test.conf",
				Host:       "host1",
				CanAutoFix: true,
				Severity:   SeverityHigh,
				FixOperation: &FixOperation{
					Module: "file",
					Args:   map[string]interface{}{},
					Order:  1,
				},
			},
		},
	}

	result, err := fixer.FixDrift(context.Background(), report)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.FixedCount != 0 {
		t.Errorf("expected 0 fixed items in dry-run, got %d", result.FixedCount)
	}

	if len(result.SkippedItems) != 1 {
		t.Errorf("expected 1 skipped item, got %d", len(result.SkippedItems))
	}

	if result.SkippedItems[0].Reason != "Dry-run mode enabled" {
		t.Errorf("expected dry-run reason, got %s", result.SkippedItems[0].Reason)
	}
}

func TestFixDriftNoFixableItems(t *testing.T) {
	config := &DriftConfig{
		AutoFix: true,
		DryRun:  false,
	}

	fixer := NewFixer(newMockModuleRegistry(), &mockLogger{}, config)

	report := &DriftReport{
		ID:          "report-1",
		TotalDrifts: 1,
		Items: []DriftItem{
			{
				ID:         "drift-1",
				Type:       DriftTypeFile,
				CanAutoFix: false,
			},
		},
	}

	result, err := fixer.FixDrift(context.Background(), report)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.FixedCount != 0 {
		t.Errorf("expected 0 fixed items, got %d", result.FixedCount)
	}

	if result.TotalProcessed != 0 {
		t.Errorf("expected 0 processed items, got %d", result.TotalProcessed)
	}
}

func TestFixerExecuteFixNoOperation(t *testing.T) {
	fixer := NewFixer(newMockModuleRegistry(), &mockLogger{}, &DriftConfig{})

	item := &DriftItem{
		ID:           "drift-1",
		FixOperation: nil,
	}

	err := fixer.executeFix(context.Background(), item)

	if err == nil {
		t.Error("expected error for nil FixOperation")
	}

	if err.Error() != "no fix operation available" {
		t.Errorf("expected 'no fix operation available', got %s", err.Error())
	}
}

func TestFixerExecuteFixModuleNotFound(t *testing.T) {
	// Create a mock registry that returns error for unknown modules
	registry := &testModuleRegistry{
		modules: make(map[string]interfaces.ModuleExecutor),
	}
	fixer := NewFixer(registry, &mockLogger{}, &DriftConfig{})

	item := &DriftItem{
		ID:   "drift-1",
		Host: "host1",
		FixOperation: &FixOperation{
			Module: "nonexistent",
			Args:   map[string]interface{}{},
		},
	}

	err := fixer.executeFix(context.Background(), item)

	if err == nil {
		t.Error("expected error for non-existent module")
	}
}

func TestFixerExecuteFixModuleValidationFails(t *testing.T) {
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	failingModule := &mockModule{
		name: "failing",
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{}, fmt.Errorf("execution failed")
		},
	}
	registry.Register("failing", failingModule)

	fixer := NewFixer(registry, logger, &DriftConfig{})

	item := &DriftItem{
		ID:   "drift-1",
		Host: "host1",
		FixOperation: &FixOperation{
			Module: "failing",
			Args:   map[string]interface{}{},
		},
	}

	err := fixer.executeFix(context.Background(), item)

	if err == nil {
		t.Error("expected error when module execution fails")
	}
}

func TestFixDriftWithSuccessfulFix(t *testing.T) {
	registry := newMockModuleRegistry()
	successModule := &mockModule{
		name: "file",
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: true}, nil
		},
	}
	registry.Register("file", successModule)

	config := &DriftConfig{
		AutoFix:         true,
		DryRun:          false,
		AutoFixSeverity: []DriftSeverity{SeverityHigh},
	}

	fixer := NewFixer(registry, &mockLogger{}, config)

	report := &DriftReport{
		ID:          "report-1",
		TotalDrifts: 1,
		Items: []DriftItem{
			{
				ID:         "drift-1",
				Type:       DriftTypeFile,
				Resource:   "test.conf",
				Host:       "host1",
				Severity:   SeverityHigh,
				CanAutoFix: true,
				FixOperation: &FixOperation{
					Module: "file",
					Args:   map[string]interface{}{},
					Order:  1,
				},
			},
		},
	}

	result, err := fixer.FixDrift(context.Background(), report)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.FixedCount != 1 {
		t.Errorf("expected 1 fixed item, got %d", result.FixedCount)
	}

	if len(result.FixedItems) != 1 {
		t.Errorf("expected 1 fixed item in list, got %d", len(result.FixedItems))
	}

	if result.FixedItems[0].DriftItem.Status != StatusFixed {
		t.Errorf("expected status to be Fixed, got %s", result.FixedItems[0].DriftItem.Status)
	}
}

func TestFixDriftWithFailedFix(t *testing.T) {
	registry := newMockModuleRegistry()
	failingModule := &mockModule{
		name: "file",
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: false, Error: "permission denied"}, nil
		},
	}
	registry.Register("file", failingModule)

	config := &DriftConfig{
		AutoFix:         true,
		DryRun:          false,
		AutoFixSeverity: []DriftSeverity{SeverityHigh},
	}

	fixer := NewFixer(registry, &mockLogger{}, config)

	report := &DriftReport{
		ID:          "report-1",
		TotalDrifts: 1,
		Items: []DriftItem{
			{
				ID:         "drift-1",
				Type:       DriftTypeFile,
				Resource:   "test.conf",
				Host:       "host1",
				Severity:   SeverityHigh,
				CanAutoFix: true,
				FixOperation: &FixOperation{
					Module: "file",
					Args:   map[string]interface{}{},
					Order:  1,
				},
			},
		},
	}

	result, err := fixer.FixDrift(context.Background(), report)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.FailedCount != 1 {
		t.Errorf("expected 1 failed item, got %d", result.FailedCount)
	}

	if len(result.FailedItems) != 1 {
		t.Errorf("expected 1 failed item in list, got %d", len(result.FailedItems))
	}
}

func TestFixDriftMultipleItems(t *testing.T) {
	registry := newMockModuleRegistry()
	successModule := &mockModule{
		name: "file",
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: true}, nil
		},
	}
	registry.Register("file", successModule)

	config := &DriftConfig{
		AutoFix:         true,
		DryRun:          false,
		AutoFixSeverity: []DriftSeverity{SeverityHigh, SeverityMedium},
	}

	fixer := NewFixer(registry, &mockLogger{}, config)

	report := &DriftReport{
		ID:          "report-1",
		TotalDrifts: 3,
		Items: []DriftItem{
			{
				ID:           "drift-1",
				Type:         DriftTypeFile,
				Severity:     SeverityHigh,
				CanAutoFix:   true,
				FixOperation: &FixOperation{Module: "file", Order: 1},
			},
			{
				ID:           "drift-2",
				Type:         DriftTypeFile,
				Severity:     SeverityMedium,
				CanAutoFix:   true,
				FixOperation: &FixOperation{Module: "file", Order: 2},
			},
			{
				ID:           "drift-3",
				Type:         DriftTypeFile,
				Severity:     SeverityLow,
				CanAutoFix:   true,
				FixOperation: &FixOperation{Module: "file", Order: 3},
			},
		},
	}

	result, err := fixer.FixDrift(context.Background(), report)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalProcessed != 2 {
		t.Errorf("expected 2 processed items (High+Medium), got %d", result.TotalProcessed)
	}

	if result.FixedCount != 2 {
		t.Errorf("expected 2 fixed items, got %d", result.FixedCount)
	}
}

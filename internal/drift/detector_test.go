package drift

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Info(format string, args ...interface{})                        {}
func (m *mockLogger) Error(format string, args ...interface{})                       {}
func (m *mockLogger) Warn(format string, args ...interface{})                        {}
func (m *mockLogger) Debug(format string, args ...interface{})                       {}
func (m *mockLogger) Fatal(format string, args ...interface{})                       {}
func (m *mockLogger) SetLevel(level string)                                          {}
func (m *mockLogger) TaskStart(name, host string)                                    {}
func (m *mockLogger) TaskEnd(name, host string, changed, success bool)               {}
func (m *mockLogger) PlayStart(name string, num, total int)                          {}
func (m *mockLogger) PlayEnd(name, host string, success bool, dur time.Duration)     {}
func (m *mockLogger) Progress(completed, total int, currentTask, currentHost string) {}
func (m *mockLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
}

// Mock module registry for testing
type mockModuleRegistry struct {
	modules map[string]*mockModule
}

func newMockModuleRegistry() *mockModuleRegistry {
	return &mockModuleRegistry{
		modules: make(map[string]*mockModule),
	}
}

func (m *mockModuleRegistry) Register(name string, module interfaces.ModuleExecutor) error {
	if mod, ok := module.(*mockModule); ok {
		m.modules[name] = mod
	}
	return nil
}

func (m *mockModuleRegistry) Get(name string) (interfaces.ModuleExecutor, error) {
	if mod, ok := m.modules[name]; ok {
		return mod, nil
	}
	// Return a default mock module
	return &mockModule{
		name: name,
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: true, Output: make(map[string]interface{})}, nil
		},
	}, nil
}

func (m *mockModuleRegistry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error) {
	return types.TaskResult{Success: true}, nil
}

func (m *mockModuleRegistry) List() []string {
	names := make([]string, 0, len(m.modules))
	for name := range m.modules {
		names = append(names, name)
	}
	return names
}

func (m *mockModuleRegistry) Unregister(name string) error {
	delete(m.modules, name)
	return nil
}

// Mock module for testing
type mockModule struct {
	name        string
	executeFunc func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
}

func (m *mockModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, host, args)
	}
	return types.TaskResult{Success: true, Output: make(map[string]interface{})}, nil
}

func (m *mockModule) Validate(args map[string]interface{}) error {
	return nil
}

func (m *mockModule) GetName() string {
	return m.name
}

func (m *mockModule) GetDescription() string {
	return "Mock module for testing"
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.CheckInterval != 1*time.Hour {
		t.Errorf("Expected CheckInterval to be 1h, got %v", config.CheckInterval)
	}

	if len(config.Resources) != 5 {
		t.Errorf("Expected 5 resources, got %d", len(config.Resources))
	}

	if config.AutoFix {
		t.Error("Expected AutoFix to be false by default")
	}
}

func TestCompareStates_NoDrift(t *testing.T) {
	detector := &Detector{
		logger: &mockLogger{},
		config: DefaultConfig(),
	}

	expected := map[string]interface{}{
		"state":   "started",
		"enabled": true,
	}

	actual := map[string]interface{}{
		"state":   "started",
		"enabled": true,
	}

	result := detector.compareStates(expected, actual)

	if result.HasDrift {
		t.Error("Expected no drift")
	}

	if len(result.Diff) != 0 {
		t.Errorf("Expected no diff, got %d items", len(result.Diff))
	}
}

func TestCompareStates_WithDrift(t *testing.T) {
	detector := &Detector{
		logger: &mockLogger{},
		config: DefaultConfig(),
	}

	expected := map[string]interface{}{
		"state":   "started",
		"enabled": true,
	}

	actual := map[string]interface{}{
		"state":   "stopped",
		"enabled": false,
	}

	result := detector.compareStates(expected, actual)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.Diff) != 2 {
		t.Errorf("Expected 2 diff items, got %d", len(result.Diff))
	}

	if !result.Diff["state"].Changed {
		t.Error("Expected state to be changed")
	}

	if !result.Diff["enabled"].Changed {
		t.Error("Expected enabled to be changed")
	}
}

func TestCalculateSeverity(t *testing.T) {
	detector := &Detector{
		logger: &mockLogger{},
		config: DefaultConfig(),
	}

	tests := []struct {
		resourceType     string
		expectedSeverity DriftSeverity
	}{
		{"service", SeverityCritical},
		{"package", SeverityHigh},
		{"user", SeverityHigh},
		{"group", SeverityHigh},
		{"file", SeverityMedium},
		{"unknown", SeverityLow},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := &DriftCheckResult{HasDrift: true}
			severity := detector.calculateSeverity(tt.resourceType, result)

			if severity != tt.expectedSeverity {
				t.Errorf("Expected severity %s for %s, got %s",
					tt.expectedSeverity, tt.resourceType, severity)
			}
		})
	}
}

func TestCalculateFixOrder(t *testing.T) {
	tests := []struct {
		resourceType  string
		expectedOrder int
	}{
		{"service", 100},
		{"systemd", 100},
		{"cron", 90},
		{"file", 80},
		{"copy", 80},
		{"template", 80},
		{"git", 70},
		{"package", 60},
		{"user", 50},
		{"group", 40},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			order := calculateFixOrder(tt.resourceType)

			if order != tt.expectedOrder {
				t.Errorf("Expected order %d for %s, got %d",
					tt.expectedOrder, tt.resourceType, order)
			}
		})
	}
}

func TestShouldCheckResource(t *testing.T) {
	config := &DriftConfig{
		Resources: []DriftType{
			DriftTypeFile,
			DriftTypeService,
		},
	}

	detector := &Detector{
		logger: &mockLogger{},
		config: config,
	}

	if !detector.shouldCheckResource(DriftTypeFile) {
		t.Error("Expected file to be checked")
	}

	if !detector.shouldCheckResource(DriftTypeService) {
		t.Error("Expected service to be checked")
	}

	if detector.shouldCheckResource(DriftTypePackage) {
		t.Error("Expected package not to be checked")
	}
}

func TestIsIgnored(t *testing.T) {
	config := &DriftConfig{
		IgnoreResources: []string{
			"/tmp/test.txt",
			"/var/log/app.log",
		},
	}

	detector := &Detector{
		logger: &mockLogger{},
		config: config,
	}

	if !detector.isIgnored("/tmp/test.txt") {
		t.Error("Expected /tmp/test.txt to be ignored")
	}

	if detector.isIgnored("/etc/nginx.conf") {
		t.Error("Expected /etc/nginx.conf not to be ignored")
	}
}

func TestDetectDrift_NoSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	sm := rollback.NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}
	config := DefaultConfig()

	detector := NewDetector(sm, registry, logger, config)

	ctx := context.Background()
	_, err := detector.DetectDrift(ctx, "nonexistent")

	if err == nil {
		t.Error("Expected error for nonexistent snapshot")
	}
}

func TestFormatDiff(t *testing.T) {
	detector := &Detector{
		logger: &mockLogger{},
		config: DefaultConfig(),
	}

	diff := map[string]DiffValue{
		"state": {
			Expected: "started",
			Actual:   "stopped",
			Changed:  true,
		},
		"enabled": {
			Expected: true,
			Actual:   false,
			Changed:  true,
		},
	}

	result := detector.formatDiff(diff)

	if result == "" {
		t.Error("Expected non-empty diff string")
	}

	// Should contain both fields
	if len(result) < 10 {
		t.Error("Expected longer diff string")
	}
}

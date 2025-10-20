package modules

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MockServiceManager implements ServiceManagerFixed for testing
type MockServiceManager struct {
	executor      ModuleExecutor
	runningState  map[string]bool
	enabledState  map[string]bool
	startCalls    []string
	stopCalls     []string
	restartCalls  []string
	reloadCalls   []string
	enableCalls   []string
	disableCalls  []string
	shouldFailOn  string
	statusDetails map[string]ServiceStatus
}

func NewMockServiceManager() *MockServiceManager {
	return &MockServiceManager{
		runningState:  make(map[string]bool),
		enabledState:  make(map[string]bool),
		statusDetails: make(map[string]ServiceStatus),
	}
}

func (m *MockServiceManager) Start(name string) error {
	m.startCalls = append(m.startCalls, name)
	if m.shouldFailOn == "start" {
		return fmt.Errorf("mock start error")
	}
	m.runningState[name] = true
	return nil
}

func (m *MockServiceManager) Stop(name string) error {
	m.stopCalls = append(m.stopCalls, name)
	if m.shouldFailOn == "stop" {
		return fmt.Errorf("mock stop error")
	}
	m.runningState[name] = false
	return nil
}

func (m *MockServiceManager) Restart(name string) error {
	m.restartCalls = append(m.restartCalls, name)
	if m.shouldFailOn == "restart" {
		return fmt.Errorf("mock restart error")
	}
	m.runningState[name] = true
	return nil
}

func (m *MockServiceManager) Reload(name string) error {
	m.reloadCalls = append(m.reloadCalls, name)
	if m.shouldFailOn == "reload" {
		return fmt.Errorf("mock reload error")
	}
	return nil
}

func (m *MockServiceManager) Enable(name string) error {
	m.enableCalls = append(m.enableCalls, name)
	if m.shouldFailOn == "enable" {
		return fmt.Errorf("mock enable error")
	}
	m.enabledState[name] = true
	return nil
}

func (m *MockServiceManager) Disable(name string) error {
	m.disableCalls = append(m.disableCalls, name)
	if m.shouldFailOn == "disable" {
		return fmt.Errorf("mock disable error")
	}
	m.enabledState[name] = false
	return nil
}

func (m *MockServiceManager) IsRunning(name string) (bool, error) {
	if m.shouldFailOn == "is_running" {
		return false, fmt.Errorf("mock is_running error")
	}
	return m.runningState[name], nil
}

func (m *MockServiceManager) IsEnabled(name string) (bool, error) {
	if m.shouldFailOn == "is_enabled" {
		return false, fmt.Errorf("mock is_enabled error")
	}
	return m.enabledState[name], nil
}

func (m *MockServiceManager) GetStatus(name string) (ServiceStatus, error) {
	if m.shouldFailOn == "get_status" {
		return ServiceStatus{}, fmt.Errorf("mock get_status error")
	}

	if status, exists := m.statusDetails[name]; exists {
		return status, nil
	}

	return ServiceStatus{
		Name:        name,
		Running:     m.runningState[name],
		Enabled:     m.enabledState[name],
		Active:      m.runningState[name],
		Status:      "active",
		ActiveState: "active",
		SubState:    "running",
		LoadState:   "loaded",
	}, nil
}

func (m *MockServiceManager) SetExecutor(executor ModuleExecutor) {
	m.executor = executor
}

// TestServiceModule_Execute_StartService tests starting a stopped service
func TestServiceModule_Execute_StartService(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = false
	mockManager.enabledState["nginx"] = false

	// Create a mock executor to prevent real executor creation
	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager, // Use test service manager for testing
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "started",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if len(mockManager.startCalls) != 1 {
		t.Errorf("Expected 1 start call, got %d", len(mockManager.startCalls))
	}

	if action, ok := result.Output["action"].(string); !ok || action != "started" {
		t.Errorf("Expected action=started, got %v", result.Output["action"])
	}
}

// TestServiceModule_Execute_StopService tests stopping a running service
func TestServiceModule_Execute_StopService(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = true
	mockManager.enabledState["nginx"] = true

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "stopped",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if len(mockManager.stopCalls) != 1 {
		t.Errorf("Expected 1 stop call, got %d", len(mockManager.stopCalls))
	}

	if action, ok := result.Output["action"].(string); !ok || action != "stopped" {
		t.Errorf("Expected action=stopped, got %v", result.Output["action"])
	}
}

// TestServiceModule_Execute_RestartService tests restarting a service
func TestServiceModule_Execute_RestartService(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = true

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "restarted",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if len(mockManager.restartCalls) != 1 {
		t.Errorf("Expected 1 restart call, got %d", len(mockManager.restartCalls))
	}
}

// TestServiceModule_Execute_ReloadService tests reloading a service
func TestServiceModule_Execute_ReloadService(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = true

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "reloaded",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if len(mockManager.reloadCalls) != 1 {
		t.Errorf("Expected 1 reload call, got %d", len(mockManager.reloadCalls))
	}
}

// TestServiceModule_Execute_EnableService tests enabling a service
func TestServiceModule_Execute_EnableService(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = true
	mockManager.enabledState["nginx"] = false

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":    "nginx",
		"state":   "started",
		"enabled": true,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if len(mockManager.enableCalls) != 1 {
		t.Errorf("Expected 1 enable call, got %d", len(mockManager.enableCalls))
	}

	if enabled, ok := result.Output["enabled"].(bool); !ok || !enabled {
		t.Errorf("Expected enabled=true, got %v", result.Output["enabled"])
	}
}

// TestServiceModule_Execute_DisableService tests disabling a service
func TestServiceModule_Execute_DisableService(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = false
	mockManager.enabledState["nginx"] = true

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":    "nginx",
		"state":   "stopped",
		"enabled": false,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if len(mockManager.disableCalls) != 1 {
		t.Errorf("Expected 1 disable call, got %d", len(mockManager.disableCalls))
	}

	if enabled, ok := result.Output["enabled"].(bool); ok && enabled {
		t.Errorf("Expected enabled=false, got %v", result.Output["enabled"])
	}
}

// TestServiceModule_Execute_NoChange tests when service is already in desired state
func TestServiceModule_Execute_NoChange(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = true
	mockManager.enabledState["nginx"] = true

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":    "nginx",
		"state":   "started",
		"enabled": true,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected changed=false, got true")
	}

	if len(mockManager.startCalls) != 0 {
		t.Errorf("Expected 0 start calls, got %d", len(mockManager.startCalls))
	}

	if len(mockManager.enableCalls) != 0 {
		t.Errorf("Expected 0 enable calls, got %d", len(mockManager.enableCalls))
	}
}

// TestServiceModule_Execute_MissingName tests error when name is missing
func TestServiceModule_Execute_MissingName(t *testing.T) {
	mockManager := NewMockServiceManager()

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"state": "started",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}

	if result.Error == "" {
		t.Errorf("Expected error message, got empty string")
	}
}

// TestServiceModule_Execute_StartFailure tests error handling when start fails
func TestServiceModule_Execute_StartFailure(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = false
	mockManager.shouldFailOn = "start"

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "started",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}

	if result.Error == "" {
		t.Errorf("Expected error message, got empty string")
	}
}

// TestServiceModule_Execute_GetStatusFailure tests error handling when getting status fails
func TestServiceModule_Execute_GetStatusFailure(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.shouldFailOn = "get_status"

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "started",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}
}

// TestServiceModule_Validate tests argument validation
func TestServiceModule_Validate(t *testing.T) {
	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
	}

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid_started",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "started",
			},
			wantErr: false,
		},
		{
			name: "valid_stopped",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "stopped",
			},
			wantErr: false,
		},
		{
			name: "valid_restarted",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "restarted",
			},
			wantErr: false,
		},
		{
			name: "valid_reloaded",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "reloaded",
			},
			wantErr: false,
		},
		{
			name: "missing_name",
			args: map[string]interface{}{
				"state": "started",
			},
			wantErr: true,
		},
		{
			name: "invalid_state",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestServiceModule_Execute_WithTimeout tests execution with context timeout
func TestServiceModule_Execute_WithTimeout(t *testing.T) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = false

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "started",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}
}

// TestServiceModule_GetName tests module name retrieval
func TestServiceModule_GetName(t *testing.T) {
	module := NewServiceModuleFixed()

	if module.GetName() != "service" {
		t.Errorf("Expected name 'service', got '%s'", module.GetName())
	}
}

// TestServiceModule_GetDescription tests module description retrieval
func TestServiceModule_GetDescription(t *testing.T) {
	module := NewServiceModuleFixed()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}
}

// TestNewServiceModule tests module creation
func TestNewServiceModule(t *testing.T) {
	module := NewServiceModule()

	if module == nil {
		t.Fatal("Expected non-nil module")
	}

	if module.GetName() != "service" {
		t.Errorf("Expected name 'service', got '%s'", module.GetName())
	}

	// Note: testServiceManager is only set in test fixtures, not in production code
	// The actual service manager is detected at runtime in Execute()
}

// BenchmarkServiceModule_Execute benchmarks service module execution
func BenchmarkServiceModule_Execute(b *testing.B) {
	mockManager := NewMockServiceManager()
	mockManager.runningState["nginx"] = false

	module := &ServiceModuleFixed{
		BaseExecutorModule: NewBaseExecutorModule("service"),
		testServiceManager: mockManager,
	}

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "started",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockManager.runningState["nginx"] = false
		_, _ = module.Execute(ctx, host, args)
	}
}

// BenchmarkServiceModule_Validate benchmarks argument validation
func BenchmarkServiceModule_Validate(b *testing.B) {
	module := NewServiceModuleFixed()

	args := map[string]interface{}{
		"name":  "nginx",
		"state": "started",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

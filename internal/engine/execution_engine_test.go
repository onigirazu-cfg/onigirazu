package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) GetDryRun() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) GetCheckMode() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) GetMaxConcurrency() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockConfig) GetDefaultTimeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockConfig) GetRetryAttempts() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockConfig) GetRetryDelay() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockConfig) GetStateFile() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfig) GetLogLevel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfig) GetLogFormat() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfig) IsShellCommandsAllowed() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) GetBlockedCommands() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockConfig) IsCachingEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) GetCacheTTL() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockConfig) IsChecksumEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Debug(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warn(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Error(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) TaskStart(taskName, hostName string) {
	m.Called(taskName, hostName)
}

func (m *MockLogger) TaskEnd(taskName, hostName string, changed, success bool) {
	m.Called(taskName, hostName, changed, success)
}

func (m *MockLogger) PlayStart(playName string, index, total int) {
	m.Called(playName, index, total)
}

func (m *MockLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {
	m.Called(playName, hostName, success, duration)
}

func (m *MockLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
	m.Called(taskName, hostName, attempt, maxAttempts, delay, err)
}

func (m *MockLogger) Fatal(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) SetLevel(level string) {
	m.Called(level)
}

func (m *MockLogger) Progress(completed, total int, currentTask, currentHost string) {
	m.Called(completed, total, currentTask, currentHost)
}

type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) SaveState(ctx context.Context, state *types.State) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockStateManager) LoadState(ctx context.Context) (*types.State, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.State), args.Error(1)
}

func (m *MockStateManager) HasChanged(filePath string) (bool, error) {
	args := m.Called(filePath)
	return args.Bool(0), args.Error(1)
}

func (m *MockStateManager) GetTaskState(taskID string) (*types.TaskState, bool) {
	args := m.Called(taskID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*types.TaskState), args.Bool(1)
}

func (m *MockStateManager) SetTaskState(taskID string, state *types.TaskState) {
	m.Called(taskID, state)
}

func (m *MockStateManager) Clear() error {
	args := m.Called()
	return args.Error(0)
}

type MockInventoryManager struct {
	mock.Mock
	hosts []types.Host
}

func (m *MockInventoryManager) LoadInventory(ctx context.Context, filePath string) error {
	args := m.Called(ctx, filePath)
	return args.Error(0)
}

func (m *MockInventoryManager) GetHosts(pattern string) ([]types.Host, error) {
	args := m.Called(pattern)
	if m.hosts != nil {
		return m.hosts, nil
	}
	if args.Get(0) == nil {
		return []types.Host{}, args.Error(1)
	}
	return args.Get(0).([]types.Host), args.Error(1)
}

func (m *MockInventoryManager) GetGroups(pattern string) (map[string]*types.Group, error) {
	args := m.Called(pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*types.Group), args.Error(1)
}

func (m *MockInventoryManager) GetHostByName(name string) (*types.Host, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Host), args.Error(1)
}

func (m *MockInventoryManager) GetGroupByName(name string) (*types.Group, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Group), args.Error(1)
}

func (m *MockInventoryManager) ListHosts() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockInventoryManager) ListGroups() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockInventoryManager) GetInventoryStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type MockModuleRegistry struct {
	mock.Mock
}

func (m *MockModuleRegistry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, vars map[string]interface{}) (types.TaskResult, error) {
	args := m.Called(ctx, task, host, vars)
	return args.Get(0).(types.TaskResult), args.Error(1)
}

func (m *MockModuleRegistry) Register(name string, module interfaces.ModuleExecutor) error {
	args := m.Called(name, module)
	return args.Error(0)
}

func (m *MockModuleRegistry) Get(name string) (interfaces.ModuleExecutor, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.ModuleExecutor), args.Error(1)
}

func (m *MockModuleRegistry) List() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockModuleRegistry) Unregister(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

type MockTemplateEngine struct {
	mock.Mock
}

func (m *MockTemplateEngine) Render(ctx context.Context, template string, vars map[string]interface{}) (string, error) {
	args := m.Called(ctx, template, vars)
	return args.String(0), args.Error(1)
}

func (m *MockTemplateEngine) RenderFile(ctx context.Context, filePath string, vars map[string]interface{}) (string, error) {
	args := m.Called(ctx, filePath, vars)
	return args.String(0), args.Error(1)
}

func (m *MockTemplateEngine) RenderTaskArgs(ctx context.Context, args map[string]interface{}, vars map[string]interface{}) (map[string]interface{}, error) {
	callArgs := m.Called(ctx, args, vars)
	if callArgs.Get(0) == nil {
		return args, callArgs.Error(1) // Return original args if no mock return
	}
	return callArgs.Get(0).(map[string]interface{}), callArgs.Error(1)
}

func (m *MockTemplateEngine) ValidateTemplate(template string) error {
	args := m.Called(template)
	return args.Error(0)
}

type MockProgressTracker struct {
	mock.Mock
}

func (m *MockProgressTracker) StartTracking() {
	m.Called()
}

func (m *MockProgressTracker) Stop() {
	m.Called()
}

func (m *MockProgressTracker) UpdateTask(hostName, taskName string, success bool) {
	m.Called(hostName, taskName, success)
}

func (m *MockProgressTracker) UpdateProgress(completed, total int) {
	m.Called(completed, total)
}

func (m *MockProgressTracker) GetProgress() (completed, total int) {
	args := m.Called()
	return args.Int(0), args.Int(1)
}

func (m *MockProgressTracker) GetStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type MockExecutionPool struct {
	mock.Mock
}

func (m *MockExecutionPool) Submit(fn func()) {
	// Execute immediately for testing
	fn()
}

func (m *MockExecutionPool) SubmitWithResult(task func() error) <-chan error {
	args := m.Called(task)
	return args.Get(0).(<-chan error)
}

func (m *MockExecutionPool) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockExecutionPool) GetStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) Get(ctx context.Context, key string) (interface{}, bool) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Bool(1)
}

func (m *MockCacheManager) Set(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockCacheManager) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheManager) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheManager) Size() int {
	args := m.Called()
	return args.Int(0)
}

// Helper function to create a test engine with mocks
func createTestEngine() (*ExecutionEngine, *MockConfig, *MockLogger, *MockInventoryManager, *MockModuleRegistry, *MockTemplateEngine) {
	mockConfig := new(MockConfig)
	mockLogger := new(MockLogger)
	mockStateManager := new(MockStateManager)
	mockInventoryMgr := new(MockInventoryManager)
	mockModuleRegistry := new(MockModuleRegistry)
	mockTemplateEngine := new(MockTemplateEngine)
	mockProgressTracker := new(MockProgressTracker)
	mockExecutionPool := new(MockExecutionPool)
	mockCacheManager := new(MockCacheManager)

	// Set up default mock behaviors
	// Note: GetDryRun, GetCheckMode, GetVerbose are not set here to allow tests to override
	mockConfig.On("GetMaxConcurrency").Return(10)
	mockConfig.On("GetDefaultTimeout").Return(30 * time.Second)
	mockConfig.On("GetRetryAttempts").Return(3)
	mockConfig.On("GetRetryDelay").Return(5 * time.Second)
	mockConfig.On("IsShellCommandsAllowed").Return(true)
	mockConfig.On("IsCachingEnabled").Return(true)
	mockConfig.On("GetCacheTTL").Return(300 * time.Second)
	mockConfig.On("IsChecksumEnabled").Return(true)

	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockLogger.On("TaskStart", mock.Anything, mock.Anything).Return()
	mockLogger.On("TaskEnd", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("PlayStart", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("PlayEnd", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	mockProgressTracker.On("UpdateTask", mock.Anything, mock.Anything, mock.Anything).Return()
	mockStateManager.On("LoadState", mock.Anything).Return(&types.State{}, nil)

	engine := NewExecutionEngine(
		mockConfig,
		mockLogger,
		mockStateManager,
		mockInventoryMgr,
		mockModuleRegistry,
		mockTemplateEngine,
		mockProgressTracker,
		mockExecutionPool,
		mockCacheManager,
	)

	return engine, mockConfig, mockLogger, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine
}

// Tests

func TestNewExecutionEngine(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.config)
	assert.NotNil(t, engine.logger)
	assert.NotNil(t, engine.metricsManager)
	assert.NotNil(t, engine.securityValidator)
	assert.NotNil(t, engine.factsGatherer)
	assert.NotNil(t, engine.variables)
	assert.NotNil(t, engine.facts)
	assert.NotNil(t, engine.stats)
	assert.NotNil(t, engine.stats.HostStats)
}

func TestExecutePlaybook_EmptyPlaybook(t *testing.T) {
	engine, _, mockLogger, _, _, _ := createTestEngine()

	playbook := &types.Playbook{
		Name:  "Test Playbook",
		Plays: []types.Play{},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Playbook", result.Name)
	assert.False(t, result.Failed)
	assert.Empty(t, result.Plays)
	mockLogger.AssertCalled(t, "Info", mock.Anything, mock.Anything)
}

func TestExecutePlaybook_WithVariables(t *testing.T) {
	engine, mockConfig, _, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	// Setup test data
	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts
	mockInventoryMgr.On("GetHosts", "all").Return(hosts, nil)

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"msg": "Hello"}, nil)

	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "test task",
			Host:     "host1",
			Success:  true,
			Changed:  false,
			Failed:   false,
			Output:   map[string]interface{}{"msg": "Hello"},
		}, nil)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Vars: map[string]interface{}{
			"test_var": "test_value",
		},
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "all",
				Tasks: []types.Task{
					{
						Name:   "test task",
						Module: "debug",
						Args:   map[string]interface{}{"msg": "{{ test_var }}"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Failed)
	assert.Len(t, result.Plays, 1)
}

func TestExecutePlaybook_FailedPlay(t *testing.T) {
	engine, mockConfig, mockLogger, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts
	mockInventoryMgr.On("GetHosts", "all").Return(hosts, nil)

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"command": "false"}, nil)

	// Simulate task failure
	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "failing task",
			Host:     "host1",
			Success:  false,
			Failed:   true,
			Error:    "command failed",
		}, errors.New("command failed"))

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Failing Play",
				Hosts: "all",
				Tasks: []types.Task{
					{
						Name:   "failing task",
						Module: "command",
						Args:   map[string]interface{}{"command": "false"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err) // ExecutePlaybook doesn't return error, sets result.Failed
	assert.NotNil(t, result)
	assert.True(t, result.Failed)
	mockLogger.AssertCalled(t, "Error", mock.Anything, mock.Anything)
}

func TestExecutePlaybook_IgnoreErrors(t *testing.T) {
	engine, mockConfig, _, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts
	mockInventoryMgr.On("GetHosts", "all").Return(hosts, nil)

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"command": "test"}, nil)

	// First task fails, second succeeds
	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.MatchedBy(func(task *types.Task) bool {
		return task.Name == "failing task"
	}), mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "failing task",
			Host:     "host1",
			Failed:   true,
			Error:    "command failed",
		}, errors.New("command failed"))

	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.MatchedBy(func(task *types.Task) bool {
		return task.Name == "success task"
	}), mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "success task",
			Host:     "host1",
			Success:  true,
			Changed:  false,
		}, nil)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:         "Test Play",
				Hosts:        "all",
				IgnoreErrors: true,
				Tasks: []types.Task{
					{
						Name:         "failing task",
						Module:       "command",
						Args:         map[string]interface{}{"command": "false"},
						IgnoreErrors: true,
					},
					{
						Name:   "success task",
						Module: "command",
						Args:   map[string]interface{}{"command": "true"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Play should continue despite first task failure
	assert.Len(t, result.Plays, 1)
}

func TestExecutePlaybook_DryRun(t *testing.T) {
	engine, mockConfig, _, mockInventoryMgr, _, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(true)

	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts
	mockInventoryMgr.On("GetHosts", "all").Return(hosts, nil)

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"command": "test"}, nil)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "all",
				Tasks: []types.Task{
					{
						Name:   "test task",
						Module: "command",
						Args:   map[string]interface{}{"command": "test"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Failed)
	assert.Len(t, result.Plays, 1)
	assert.Len(t, result.Plays[0].Hosts, 1)
	assert.Len(t, result.Plays[0].Hosts[0].Tasks, 1)

	taskResult := result.Plays[0].Hosts[0].Tasks[0]
	assert.True(t, taskResult.Success)
	assert.False(t, taskResult.Changed) // Dry-run should not change anything
	assert.Contains(t, taskResult.Output["message"], "dry-run")
}

func TestExecuteTask_WithRetry(t *testing.T) {
	engine, mockConfig, mockLogger, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"command": "test"}, nil)

	// Fail twice, then succeed
	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "retry task",
			Host:     "host1",
			Failed:   true,
			Error:    "temporary failure",
		}, errors.New("temporary failure")).Twice()

	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "retry task",
			Host:     "host1",
			Success:  true,
			Changed:  false,
		}, nil).Once()

	mockLogger.On("Retry", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	task := &types.Task{
		Name:       "retry task",
		Module:     "command",
		Args:       map[string]interface{}{"command": "test"},
		Retries:    3,
		RetryDelay: 0, // No delay for testing
	}

	playResult := &types.PlayResult{
		Name:  "Test Play",
		Hosts: []types.HostResult{},
	}

	ctx := context.Background()
	err := engine.executeTaskOnHost(ctx, task, &hosts[0], map[string]interface{}{}, playResult)

	assert.NoError(t, err)
	// Verify ExecuteTask was called 3 times (2 failures + 1 success)
	mockModuleRegistry.AssertNumberOfCalls(t, "ExecuteTask", 3)
	mockLogger.AssertCalled(t, "Retry", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestExecuteTask_Register(t *testing.T) {
	engine, mockConfig, _, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"command": "echo hello"}, nil)

	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "test task",
			Host:     "host1",
			Success:  true,
			Changed:  false,
			Output: map[string]interface{}{
				"stdout": "hello",
				"rc":     0,
			},
		}, nil)

	task := &types.Task{
		Name:     "test task",
		Module:   "command",
		Args:     map[string]interface{}{"command": "echo hello"},
		Register: "test_result",
	}

	playResult := &types.PlayResult{
		Name:  "Test Play",
		Hosts: []types.HostResult{},
	}

	ctx := context.Background()
	err := engine.executeTaskOnHost(ctx, task, &hosts[0], map[string]interface{}{}, playResult)

	assert.NoError(t, err)

	// Check that variable was registered
	engine.mutex.RLock()
	registeredVar, exists := engine.variables["test_result"]
	engine.mutex.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, registeredVar)

	varMap, ok := registeredVar.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "hello", varMap["stdout"])
	assert.Equal(t, 0, varMap["rc"])
}

func TestExecuteTask_SetFact(t *testing.T) {
	engine, mockConfig, _, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	mockInventoryMgr.hosts = hosts

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"my_fact": "my_value"}, nil)

	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{
			TaskName: "set fact task",
			Host:     "host1",
			Success:  true,
			Changed:  false,
			Output: map[string]interface{}{
				"ansible_facts": map[string]interface{}{
					"my_fact": "my_value",
				},
			},
		}, nil)

	task := &types.Task{
		Name:   "set fact task",
		Module: "set_fact",
		Args:   map[string]interface{}{"my_fact": "my_value"},
	}

	playResult := &types.PlayResult{
		Name:  "Test Play",
		Hosts: []types.HostResult{},
	}

	ctx := context.Background()
	err := engine.executeTaskOnHost(ctx, task, &hosts[0], map[string]interface{}{}, playResult)

	assert.NoError(t, err)

	// Check that fact was set
	engine.mutex.RLock()
	factValue, exists := engine.variables["my_fact"]
	engine.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "my_value", factValue)
}

func TestExecutionStats(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Initialize stats
	engine.initializeExecution()

	// Update some stats
	task := &types.Task{Name: "test task", Module: "command"}
	result := types.TaskResult{
		TaskName: "test task",
		Host:     "host1",
		Success:  true,
		Changed:  true,
	}

	engine.updateTaskStats("host1", task, result)

	stats := engine.getExecutionStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats["completed_tasks"])
	assert.Equal(t, 1, stats["successful_tasks"])
	assert.Equal(t, 1, stats["changed_tasks"])
	assert.Equal(t, 0, stats["failed_tasks"])
	assert.Equal(t, 0, stats["skipped_tasks"])

	// Check host stats
	hostStatsMap := stats["host_stats"].(map[string]*HostStats)
	hostStats, exists := hostStatsMap["host1"]
	assert.True(t, exists)
	assert.Equal(t, 1, hostStats.CompletedTasks)
	assert.Equal(t, 1, hostStats.SuccessfulTasks)
	assert.Equal(t, 1, hostStats.ChangedTasks)
}

func TestConcurrentTaskExecution(t *testing.T) {
	engine, mockConfig, _, mockInventoryMgr, mockModuleRegistry, mockTemplateEngine := createTestEngine()

	mockConfig.On("GetDryRun").Return(false)

	// Create multiple hosts
	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
		{Name: "host3", Address: "192.168.1.3"},
	}
	mockInventoryMgr.hosts = hosts

	mockTemplateEngine.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"command": "test"}, nil)

	var executionMutex sync.Mutex
	executedHosts := make(map[string]bool)

	mockModuleRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			host := args.Get(2).(types.Host)
			executionMutex.Lock()
			executedHosts[host.Name] = true
			executionMutex.Unlock()

			// Simulate some work
			time.Sleep(10 * time.Millisecond)
		}).
		Return(types.TaskResult{
			Success: true,
			Changed: false,
		}, nil)

	task := &types.Task{
		Name:   "concurrent task",
		Module: "command",
		Args:   map[string]interface{}{"command": "test"},
		Serial: false, // Enable parallel execution
	}

	playResult := &types.PlayResult{
		Name:  "Test Play",
		Hosts: []types.HostResult{},
	}

	ctx := context.Background()
	err := engine.executeTaskParallel(ctx, task, hosts, map[string]interface{}{}, playResult)

	assert.NoError(t, err)

	// Verify all hosts were executed
	executionMutex.Lock()
	assert.Len(t, executedHosts, 3)
	assert.True(t, executedHosts["host1"])
	assert.True(t, executedHosts["host2"])
	assert.True(t, executedHosts["host3"])
	executionMutex.Unlock()
}

func TestVariableMerging(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	vars1 := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	vars2 := map[string]interface{}{
		"key2": "overridden",
		"key3": "value3",
	}

	merged := engine.mergeVariables(vars1, vars2)

	assert.Equal(t, "value1", merged["key1"])
	assert.Equal(t, "overridden", merged["key2"]) // Should be overridden
	assert.Equal(t, "value3", merged["key3"])
}

func TestSetAndGetVariables(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	vars := map[string]interface{}{
		"var1": "value1",
		"var2": 42,
		"var3": true,
	}

	engine.setVariables(vars)

	engine.mutex.RLock()
	defer engine.mutex.RUnlock()

	assert.Equal(t, "value1", engine.variables["var1"])
	assert.Equal(t, 42, engine.variables["var2"])
	assert.Equal(t, true, engine.variables["var3"])
}

func TestExecutePlaybook_NoHosts(t *testing.T) {
	engine, _, mockLogger, mockInventoryMgr, _, _ := createTestEngine()

	// Return empty host list
	mockInventoryMgr.On("GetHosts", "all").Return([]types.Host{}, nil)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "all",
				Tasks: []types.Task{
					{
						Name:   "test task",
						Module: "debug",
						Args:   map[string]interface{}{"msg": "Hello"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Failed)
	assert.Len(t, result.Plays, 1)
	assert.Empty(t, result.Plays[0].Hosts)
	mockLogger.AssertCalled(t, "Warn", mock.Anything, mock.Anything)
}

func TestExecutePlaybook_InventoryError(t *testing.T) {
	engine, _, _, mockInventoryMgr, _, _ := createTestEngine()

	// Return error from inventory
	mockInventoryMgr.On("GetHosts", "all").Return([]types.Host{}, errors.New("inventory error"))

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "all",
				Tasks: []types.Task{
					{
						Name:   "test task",
						Module: "debug",
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := engine.ExecutePlaybook(ctx, playbook)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Failed)
	assert.Contains(t, result.Error, "inventory error")
}

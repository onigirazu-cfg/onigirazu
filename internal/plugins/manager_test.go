package plugins

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Mock module plugin for testing
type mockModulePlugin struct {
	*BaseModulePlugin
	executeFunc func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
}

func newMockModulePlugin(name string) *mockModulePlugin {
	return &mockModulePlugin{
		BaseModulePlugin: NewBaseModulePlugin(name, "1.0.0", "Mock module plugin"),
	}
}

func (m *mockModulePlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, host, args)
	}
	return types.TaskResult{
		Success: true,
		Changed: false,
	}, nil
}

// Mock callback plugin for testing
type mockCallbackPlugin struct {
	*BaseCallbackPlugin
	onTaskStartCalled bool
	onTaskEndCalled   bool
}

func newMockCallbackPlugin(name string) *mockCallbackPlugin {
	return &mockCallbackPlugin{
		BaseCallbackPlugin: NewBaseCallbackPlugin(name, "1.0.0", "Mock callback plugin"),
	}
}

func (m *mockCallbackPlugin) OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error {
	m.onTaskStartCalled = true
	return nil
}

func (m *mockCallbackPlugin) OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error {
	m.onTaskEndCalled = true
	return nil
}

func TestNewManager(t *testing.T) {
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.plugins == nil {
		t.Error("plugins map not initialized")
	}

	if manager.metadata == nil {
		t.Error("metadata map not initialized")
	}
}

func TestManager_Register(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockModulePlugin("test-module")

	err := manager.Register(ctx, plugin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Try to register same plugin again
	err = manager.Register(ctx, plugin)
	if err == nil {
		t.Error("Expected error when registering duplicate plugin")
	}
}

func TestManager_Get(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockModulePlugin("test-module")
	err := manager.Register(ctx, plugin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get existing plugin
	retrieved, err := manager.Get("test-module", PluginTypeModule)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.GetName() != "test-module" {
		t.Errorf("Expected plugin name 'test-module', got '%s'", retrieved.GetName())
	}

	// Get non-existing plugin
	_, err = manager.Get("non-existing", PluginTypeModule)
	if err == nil {
		t.Error("Expected error when getting non-existing plugin")
	}
}

func TestManager_GetModule(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockModulePlugin("test-module")
	err := manager.Register(ctx, plugin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get module plugin
	modulePlugin, err := manager.GetModule("test-module")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}

	if modulePlugin.GetName() != "test-module" {
		t.Errorf("Expected plugin name 'test-module', got '%s'", modulePlugin.GetName())
	}
}

func TestManager_GetCallback(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockCallbackPlugin("test-callback")
	err := manager.Register(ctx, plugin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get callback plugin
	callbackPlugin, err := manager.GetCallback("test-callback")
	if err != nil {
		t.Fatalf("GetCallback failed: %v", err)
	}

	if callbackPlugin.GetName() != "test-callback" {
		t.Errorf("Expected plugin name 'test-callback', got '%s'", callbackPlugin.GetName())
	}
}

func TestManager_Unregister(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockModulePlugin("test-module")
	err := manager.Register(ctx, plugin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Unregister plugin
	err = manager.Unregister(ctx, "test-module", PluginTypeModule)
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	// Try to get unregistered plugin
	_, err = manager.Get("test-module", PluginTypeModule)
	if err == nil {
		t.Error("Expected error when getting unregistered plugin")
	}

	// Try to unregister non-existing plugin
	err = manager.Unregister(ctx, "non-existing", PluginTypeModule)
	if err == nil {
		t.Error("Expected error when unregistering non-existing plugin")
	}
}

func TestManager_List(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	// Register multiple plugins
	plugin1 := newMockModulePlugin("module1")
	plugin2 := newMockModulePlugin("module2")
	plugin3 := newMockCallbackPlugin("callback1")

	if err := manager.Register(ctx, plugin1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin2); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin3); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// List module plugins
	modulePlugins := manager.List(PluginTypeModule)
	if len(modulePlugins) != 2 {
		t.Errorf("Expected 2 module plugins, got %d", len(modulePlugins))
	}

	// List callback plugins
	callbackPlugins := manager.List(PluginTypeCallback)
	if len(callbackPlugins) != 1 {
		t.Errorf("Expected 1 callback plugin, got %d", len(callbackPlugins))
	}
}

func TestManager_ListAll(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	// Register multiple plugins
	plugin1 := newMockModulePlugin("module1")
	plugin2 := newMockModulePlugin("module2")
	plugin3 := newMockCallbackPlugin("callback1")

	if err := manager.Register(ctx, plugin1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin2); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin3); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// List all plugins
	allPlugins := manager.ListAll()
	if len(allPlugins) != 2 {
		t.Errorf("Expected 2 plugin types, got %d", len(allPlugins))
	}

	if len(allPlugins[PluginTypeModule]) != 2 {
		t.Errorf("Expected 2 module plugins, got %d", len(allPlugins[PluginTypeModule]))
	}

	if len(allPlugins[PluginTypeCallback]) != 1 {
		t.Errorf("Expected 1 callback plugin, got %d", len(allPlugins[PluginTypeCallback]))
	}
}

func TestManager_GetMetadata(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockModulePlugin("test-module")
	err := manager.Register(ctx, plugin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get metadata
	metadata, err := manager.GetMetadata("test-module", PluginTypeModule)
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if metadata.Name != "test-module" {
		t.Errorf("Expected name 'test-module', got '%s'", metadata.Name)
	}

	if metadata.Type != PluginTypeModule {
		t.Errorf("Expected type '%s', got '%s'", PluginTypeModule, metadata.Type)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
	}
}

func TestManager_Shutdown(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	// Register multiple plugins
	plugin1 := newMockModulePlugin("module1")
	plugin2 := newMockModulePlugin("module2")

	if err := manager.Register(ctx, plugin1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin2); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Shutdown
	err := manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify all plugins are removed
	allPlugins := manager.ListAll()
	if len(allPlugins) != 0 {
		t.Errorf("Expected 0 plugins after shutdown, got %d", len(allPlugins))
	}
}

func TestManager_GetStats(t *testing.T) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	// Register multiple plugins
	plugin1 := newMockModulePlugin("module1")
	plugin2 := newMockModulePlugin("module2")
	plugin3 := newMockCallbackPlugin("callback1")

	if err := manager.Register(ctx, plugin1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin2); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := manager.Register(ctx, plugin3); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get stats
	stats := manager.GetStats()

	if stats["total"] != 3 {
		t.Errorf("Expected total 3, got %v", stats["total"])
	}

	if stats["module"] != 2 {
		t.Errorf("Expected 2 module plugins, got %v", stats["module"])
	}

	if stats["callback"] != 1 {
		t.Errorf("Expected 1 callback plugin, got %v", stats["callback"])
	}
}

func BenchmarkManager_Register(b *testing.B) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := newMockModulePlugin("test-module")
		if err := manager.Register(ctx, plugin); err != nil {
			b.Fatalf("Register failed: %v", err)
		}
		if err := manager.Unregister(ctx, "test-module", PluginTypeModule); err != nil {
			b.Fatalf("Unregister failed: %v", err)
		}
	}
}

func BenchmarkManager_Get(b *testing.B) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	plugin := newMockModulePlugin("test-module")
	if err := manager.Register(ctx, plugin); err != nil {
		b.Fatalf("Register failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Get("test-module", PluginTypeModule)
	}
}

func BenchmarkManager_List(b *testing.B) {
	ctx := context.Background()
	loader := NewInMemoryLoader()
	manager := NewManager(loader)

	// Register 10 plugins
	for i := 0; i < 10; i++ {
		plugin := newMockModulePlugin("module" + string(rune(i)))
		if err := manager.Register(ctx, plugin); err != nil {
			b.Fatalf("Register failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.List(PluginTypeModule)
	}
}

func TestCallbackManager(t *testing.T) {
	ctx := context.Background()
	manager := NewCallbackManager()

	plugin := newMockCallbackPlugin("test-callback")
	manager.AddPlugin(plugin)

	// Test OnTaskStart
	task := &types.Task{Name: "test-task"}
	host := types.Host{Name: "test-host"}

	err := manager.OnTaskStart(ctx, task, host)
	if err != nil {
		t.Fatalf("OnTaskStart failed: %v", err)
	}

	if !plugin.onTaskStartCalled {
		t.Error("OnTaskStart was not called on plugin")
	}

	// Test OnTaskEnd
	result := types.TaskResult{Success: true}
	err = manager.OnTaskEnd(ctx, task, host, result)
	if err != nil {
		t.Fatalf("OnTaskEnd failed: %v", err)
	}

	if !plugin.onTaskEndCalled {
		t.Error("OnTaskEnd was not called on plugin")
	}

	// Test RemovePlugin
	manager.RemovePlugin("test-callback")
	plugins := manager.plugins
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins after removal, got %d", len(plugins))
	}
}

func BenchmarkCallbackManager_OnTaskStart(b *testing.B) {
	ctx := context.Background()
	manager := NewCallbackManager()

	// Add 5 callback plugins
	for i := 0; i < 5; i++ {
		plugin := newMockCallbackPlugin("callback" + string(rune(i)))
		manager.AddPlugin(plugin)
	}

	task := &types.Task{Name: "test-task"}
	host := types.Host{Name: "test-host"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.OnTaskStart(ctx, task, host)
	}
}

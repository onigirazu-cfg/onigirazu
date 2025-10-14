package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ============================================================================
// MOCK IMPLEMENTATIONS
// ============================================================================

// MockUnifiedPackageManager implements UnifiedPackageManager for testing
type MockUnifiedPackageManager struct {
	mu                sync.Mutex
	installedPackages map[string]*PackageState
	packageInfo       map[string]*PackageInfo
	shouldFailOn      string
	failureError      error
	installCalls      []string
	removeCalls       []string
	updateCalls       []string
	searchResults     []PackageInfo
	upgradableList    []PackageInfo
	dependencies      map[string][]string
	snapshots         map[string]*SystemSnapshot
	auditLog          []AuditEntry
	healthResult      *HealthCheckResult
	autoRemoveList    []string
	autoRemoveListSet bool // Flag to distinguish between "not set" and "set to empty"
	installDelay      time.Duration
	failureMap        map[string]error
}

// NewMockUnifiedPackageManager creates a new mock package manager
func NewMockUnifiedPackageManager() *MockUnifiedPackageManager {
	return &MockUnifiedPackageManager{
		installedPackages: make(map[string]*PackageState),
		packageInfo:       make(map[string]*PackageInfo),
		dependencies:      make(map[string][]string),
		snapshots:         make(map[string]*SystemSnapshot),
		auditLog:          make([]AuditEntry, 0),
		searchResults:     make([]PackageInfo, 0),
		upgradableList:    make([]PackageInfo, 0),
		autoRemoveList:    make([]string, 0),
		failureMap:        make(map[string]error),
	}
}

// SetShouldFail configures the mock to fail on specific operations
func (m *MockUnifiedPackageManager) SetShouldFail(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailOn = operation
	m.failureError = err
}

// AddInstalledPackage adds a package to the installed list
func (m *MockUnifiedPackageManager) AddInstalledPackage(name, version string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installedPackages[name] = &PackageState{
		Name:        name,
		Installed:   true,
		Version:     version,
		LastChecked: time.Now(),
	}
	m.packageInfo[name] = &PackageInfo{
		Name:      name,
		Version:   version,
		Installed: true,
	}
}

// AddAvailablePackage adds a package to the available list
func (m *MockUnifiedPackageManager) AddAvailablePackage(name, version, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packageInfo[name] = &PackageInfo{
		Name:        name,
		Version:     version,
		Description: description,
		Installed:   false,
	}
}

// SetDependencies sets dependencies for a package
func (m *MockUnifiedPackageManager) SetDependencies(name string, deps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dependencies[name] = deps
}

// SetSearchResults sets the results for Search operations
func (m *MockUnifiedPackageManager) SetSearchResults(results []PackageInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchResults = results
}

// SetUpgradableList sets the list of upgradable packages
func (m *MockUnifiedPackageManager) SetUpgradableList(packages []PackageInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upgradableList = packages
}

// SetHealthResult sets the health check result
func (m *MockUnifiedPackageManager) SetHealthResult(result *HealthCheckResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthResult = result
}

// SetAutoRemovePackages sets the list of packages to be returned by AutoRemove
func (m *MockUnifiedPackageManager) SetAutoRemovePackages(packages []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoRemoveList = packages
	m.autoRemoveListSet = true
}

// SetUpgradablePackages is an alias for SetUpgradableList for consistency
func (m *MockUnifiedPackageManager) SetUpgradablePackages(packages []PackageInfo) {
	m.SetUpgradableList(packages)
}

// SetInstallDelay sets a delay for install operations (for testing timeouts)
func (m *MockUnifiedPackageManager) SetInstallDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installDelay = delay
}

// SetFailureFor sets a specific failure for a named operation
func (m *MockUnifiedPackageManager) SetFailureFor(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureMap[operation] = err
}

// checkFailure checks if an operation should fail
func (m *MockUnifiedPackageManager) checkFailure(operation string) error {
	if err, exists := m.failureMap[operation]; exists {
		return err
	}
	if m.shouldFailOn == operation {
		return m.failureError
	}
	return nil
}

// ============================================================================
// UNIFIED PACKAGE MANAGER INTERFACE IMPLEMENTATION
// ============================================================================

func (m *MockUnifiedPackageManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	m.mu.Lock()

	// Check for delay (simulate slow operations)
	if m.installDelay > 0 {
		m.mu.Unlock()
		time.Sleep(m.installDelay)
		m.mu.Lock()
	}

	defer m.mu.Unlock()

	// Check for failures
	if err := m.checkFailure("Install"); err != nil {
		return &PackageOperation{
			Package:   name,
			Operation: "install",
			Success:   false,
			Error:     err.Error(),
		}, err
	}

	// Legacy failure check
	if m.shouldFailOn == "install" {
		return &PackageOperation{
			Package:   name,
			Operation: "install",
			Success:   false,
			Error:     m.failureError.Error(),
		}, m.failureError
	}

	m.installCalls = append(m.installCalls, name)

	// Check if already installed
	if state, exists := m.installedPackages[name]; exists {
		if version == "" || state.Version == version {
			return &PackageOperation{
				Package:    name,
				Operation:  "install",
				Success:    true,
				Changed:    false,
				OldVersion: state.Version,
				NewVersion: state.Version,
				Output:     "Package already installed",
			}, nil
		}
	}

	// Install the package
	installVersion := version
	if installVersion == "" {
		if info, exists := m.packageInfo[name]; exists {
			installVersion = info.Version
		} else {
			installVersion = "1.0.0"
		}
	}

	m.installedPackages[name] = &PackageState{
		Name:        name,
		Installed:   true,
		Version:     installVersion,
		LastChecked: time.Now(),
	}

	return &PackageOperation{
		Package:    name,
		Operation:  "install",
		Success:    true,
		Changed:    true,
		NewVersion: installVersion,
		Output:     fmt.Sprintf("Package %s installed successfully", name),
	}, nil
}

func (m *MockUnifiedPackageManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for failures
	if err := m.checkFailure("Remove"); err != nil {
		return &PackageOperation{
			Package:   name,
			Operation: "remove",
			Success:   false,
			Error:     err.Error(),
		}, err
	}

	if m.shouldFailOn == "remove" {
		return &PackageOperation{
			Package:   name,
			Operation: "remove",
			Success:   false,
			Error:     m.failureError.Error(),
		}, m.failureError
	}

	m.removeCalls = append(m.removeCalls, name)

	state, exists := m.installedPackages[name]
	if !exists {
		return &PackageOperation{
			Package:   name,
			Operation: "remove",
			Success:   true,
			Changed:   false,
			Output:    "Package not installed",
		}, nil
	}

	oldVersion := state.Version
	delete(m.installedPackages, name)

	return &PackageOperation{
		Package:    name,
		Operation:  "remove",
		Success:    true,
		Changed:    true,
		OldVersion: oldVersion,
		Output:     fmt.Sprintf("Package %s removed successfully", name),
	}, nil
}

func (m *MockUnifiedPackageManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "update" {
		return &PackageOperation{
			Package:   name,
			Operation: "update",
			Success:   false,
			Error:     m.failureError.Error(),
		}, m.failureError
	}

	m.updateCalls = append(m.updateCalls, name)

	state, exists := m.installedPackages[name]
	if !exists {
		return nil, fmt.Errorf("package %s not installed", name)
	}

	oldVersion := state.Version
	newVersion := "2.0.0"

	state.Version = newVersion
	state.LastChecked = time.Now()

	return &PackageOperation{
		Package:    name,
		Operation:  "update",
		Success:    true,
		Changed:    true,
		OldVersion: oldVersion,
		NewVersion: newVersion,
		Output:     fmt.Sprintf("Package %s updated successfully", name),
	}, nil
}

func (m *MockUnifiedPackageManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "update_all" {
		return &PackageOperation{
			Operation: "update_all",
			Success:   false,
			Error:     m.failureError.Error(),
		}, m.failureError
	}

	count := len(m.installedPackages)
	return &PackageOperation{
		Operation: "update_all",
		Success:   true,
		Changed:   count > 0,
		Output:    fmt.Sprintf("Updated %d packages", count),
	}, nil
}

func (m *MockUnifiedPackageManager) IsInstalled(ctx context.Context, name string) (*PackageState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "is_installed" {
		return nil, m.failureError
	}

	if state, exists := m.installedPackages[name]; exists {
		return state, nil
	}

	return &PackageState{
		Name:      name,
		Installed: false,
	}, nil
}

func (m *MockUnifiedPackageManager) GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "get_info" {
		return nil, m.failureError
	}

	if info, exists := m.packageInfo[name]; exists {
		return info, nil
	}

	return nil, fmt.Errorf("package %s not found", name)
}

func (m *MockUnifiedPackageManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "install_multiple" {
		return &BatchOperation{
			Success: false,
		}, m.failureError
	}

	operations := make([]PackageOperation, 0, len(packages))
	successCount := 0
	changedCount := 0

	for _, pkg := range packages {
		op := PackageOperation{
			Package:   pkg.Name,
			Operation: "install",
			Success:   true,
			Changed:   true,
		}

		if _, exists := m.installedPackages[pkg.Name]; !exists {
			m.installedPackages[pkg.Name] = &PackageState{
				Name:      pkg.Name,
				Installed: true,
				Version:   pkg.Version,
			}
			changedCount++
		} else {
			op.Changed = false
		}

		successCount++
		operations = append(operations, op)
	}

	return &BatchOperation{
		Operations:   operations,
		Success:      true,
		Changed:      changedCount > 0,
		TotalCount:   len(packages),
		SuccessCount: successCount,
		ChangedCount: changedCount,
	}, nil
}

func (m *MockUnifiedPackageManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "remove_multiple" {
		return &BatchOperation{
			Success: false,
		}, m.failureError
	}

	operations := make([]PackageOperation, 0, len(packages))
	successCount := 0
	changedCount := 0

	for _, name := range packages {
		op := PackageOperation{
			Package:   name,
			Operation: "remove",
			Success:   true,
		}

		if _, exists := m.installedPackages[name]; exists {
			delete(m.installedPackages, name)
			op.Changed = true
			changedCount++
		}

		successCount++
		operations = append(operations, op)
	}

	return &BatchOperation{
		Operations:   operations,
		Success:      true,
		Changed:      changedCount > 0,
		TotalCount:   len(packages),
		SuccessCount: successCount,
		ChangedCount: changedCount,
	}, nil
}

func (m *MockUnifiedPackageManager) RefreshCache(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "refresh_cache" {
		return m.failureError
	}

	return nil
}

func (m *MockUnifiedPackageManager) ValidateState(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "validate_state" {
		return m.failureError
	}

	return nil
}

func (m *MockUnifiedPackageManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "dry_run" {
		return nil, m.failureError
	}

	return &OperationPreview{
		WillChange: true,
		Actions:    []string{fmt.Sprintf("%s %v", operation, args)},
	}, nil
}

func (m *MockUnifiedPackageManager) GetDependencies(ctx context.Context, name string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "get_dependencies" {
		return nil, m.failureError
	}

	if deps, exists := m.dependencies[name]; exists {
		return deps, nil
	}

	return []string{}, nil
}

func (m *MockUnifiedPackageManager) VerifyChecksum(ctx context.Context, name, version string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "verify_checksum" {
		return false, m.failureError
	}

	return true, nil
}

func (m *MockUnifiedPackageManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "search" {
		return nil, m.failureError
	}

	return m.searchResults, nil
}

func (m *MockUnifiedPackageManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for failures
	if err := m.checkFailure("ListInstalled"); err != nil {
		return nil, err
	}

	if m.shouldFailOn == "list_installed" {
		return nil, m.failureError
	}

	result := make([]PackageInfo, 0, len(m.installedPackages))
	for _, state := range m.installedPackages {
		result = append(result, PackageInfo{
			Name:      state.Name,
			Version:   state.Version,
			Installed: true,
		})
	}

	return result, nil
}

func (m *MockUnifiedPackageManager) ListUpgradable(ctx context.Context) ([]PackageInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "list_upgradable" {
		return nil, m.failureError
	}

	return m.upgradableList, nil
}

func (m *MockUnifiedPackageManager) Clean(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "clean" {
		return m.failureError
	}

	return nil
}

func (m *MockUnifiedPackageManager) AutoRemove(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOn == "auto_remove" {
		return nil, m.failureError
	}

	// Return configured list if explicitly set
	if m.autoRemoveListSet {
		return m.autoRemoveList, nil
	}

	// Return default orphan packages for backward compatibility
	return []string{"orphan-pkg1", "orphan-pkg2"}, nil
}

func (m *MockUnifiedPackageManager) VerifyIntegrity(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for failures
	if err := m.checkFailure("VerifyIntegrity"); err != nil {
		return err
	}

	if m.shouldFailOn == "verify_integrity" {
		return m.failureError
	}

	return nil
}

// MockCommandExecutor is a mock implementation of executor.CommandExecutor for testing
type MockCommandExecutor struct {
	ExecuteFunc            func(command string, args ...string) (string, error)
	ExecuteWithContextFunc func(ctx context.Context, command string, args ...string) (string, error)
}

// Execute mocks the Execute method
func (m *MockCommandExecutor) Execute(command string, args ...string) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(command, args...)
	}
	return "", fmt.Errorf("Execute not implemented in mock")
}

// ExecuteWithContext mocks the ExecuteWithContext method
func (m *MockCommandExecutor) ExecuteWithContext(ctx context.Context, command string, args ...string) (string, error) {
	if m.ExecuteWithContextFunc != nil {
		return m.ExecuteWithContextFunc(ctx, command, args...)
	}
	// Fallback to Execute if ExecuteWithContext not implemented
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(command, args...)
	}
	return "", fmt.Errorf("ExecuteWithContext not implemented in mock")
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// createTestHost creates a test host for testing
func createTestHost() types.Host {
	return types.Host{
		Name:    "test-host",
		Address: "test.example.com",
		Port:    22,
		User:    "testuser",
	}
}

// createTestContext creates a test context with timeout
func createTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// ============================================================================
// BASIC MODULE TESTS
// ============================================================================

func TestNewUnifiedPackageModule(t *testing.T) {
	module := NewUnifiedPackageModule()

	if module == nil {
		t.Fatal("NewUnifiedPackageModule returned nil")
	}

	if module.GetName() != "package" {
		t.Errorf("Expected module name 'package', got '%s'", module.GetName())
	}

	if module.stateCache == nil {
		t.Error("State cache not initialized")
	}

	if module.metrics == nil {
		t.Error("Metrics not initialized")
	}
}

func TestPackageModule_GetName(t *testing.T) {
	module := NewUnifiedPackageModule()
	name := module.GetName()

	if name != "package" {
		t.Errorf("Expected name 'package', got '%s'", name)
	}
}

func TestPackageModule_GetDescription(t *testing.T) {
	module := NewUnifiedPackageModule()
	desc := module.GetDescription()

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if desc != "Unified package management with advanced features" {
		t.Errorf("Unexpected description: %s", desc)
	}
}

// ============================================================================
// PACKAGE STATE CACHE TESTS
// ============================================================================

func TestPackageStateCache_SetAndGet(t *testing.T) {
	cache := NewPackageStateCache(1 * time.Minute)

	state := &PackageState{
		Name:      "test-package",
		Installed: true,
		Version:   "1.0.0",
	}

	cache.Set("test-package", state)

	retrieved, found := cache.Get("test-package")
	if !found {
		t.Fatal("Package not found in cache")
	}

	if retrieved.Name != state.Name {
		t.Errorf("Expected name '%s', got '%s'", state.Name, retrieved.Name)
	}

	if retrieved.Version != state.Version {
		t.Errorf("Expected version '%s', got '%s'", state.Version, retrieved.Version)
	}
}

func TestPackageStateCache_Expiration(t *testing.T) {
	cache := NewPackageStateCache(100 * time.Millisecond)

	state := &PackageState{
		Name:      "test-package",
		Installed: true,
		Version:   "1.0.0",
	}

	cache.Set("test-package", state)

	// Should be found immediately
	_, found := cache.Get("test-package")
	if !found {
		t.Fatal("Package should be in cache")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get("test-package")
	if found {
		t.Error("Package should have expired from cache")
	}
}

func TestPackageStateCache_Clear(t *testing.T) {
	cache := NewPackageStateCache(1 * time.Minute)

	cache.Set("package1", &PackageState{Name: "package1"})
	cache.Set("package2", &PackageState{Name: "package2"})

	cache.Clear()

	_, found1 := cache.Get("package1")
	_, found2 := cache.Get("package2")

	if found1 || found2 {
		t.Error("Cache should be empty after Clear()")
	}
}

func TestPackageStateCache_ConcurrentAccess(t *testing.T) {
	cache := NewPackageStateCache(1 * time.Minute)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cache.Set(fmt.Sprintf("package-%d", idx), &PackageState{
				Name:    fmt.Sprintf("package-%d", idx),
				Version: "1.0.0",
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cache.Get(fmt.Sprintf("package-%d", idx))
		}(i)
	}

	wg.Wait()
}

// ============================================================================
// MOCK PACKAGE MANAGER TESTS
// ============================================================================

func TestMockPackageManager_Install(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	// Test successful installation
	op, err := mock.Install(ctx, "test-package", "1.0.0")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !op.Success {
		t.Error("Operation should be successful")
	}

	if !op.Changed {
		t.Error("Operation should report change")
	}

	if op.NewVersion != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", op.NewVersion)
	}

	// Test idempotent installation
	op2, err := mock.Install(ctx, "test-package", "1.0.0")
	if err != nil {
		t.Fatalf("Second install failed: %v", err)
	}

	if op2.Changed {
		t.Error("Second installation should not report change")
	}
}

func TestMockPackageManager_Remove(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	mock.AddInstalledPackage("test-package", "1.0.0")
	ctx := context.Background()

	// Test successful removal
	op, err := mock.Remove(ctx, "test-package")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if !op.Success {
		t.Error("Operation should be successful")
	}

	if !op.Changed {
		t.Error("Operation should report change")
	}

	// Test idempotent removal
	op2, err := mock.Remove(ctx, "test-package")
	if err != nil {
		t.Fatalf("Second remove failed: %v", err)
	}

	if op2.Changed {
		t.Error("Second removal should not report change")
	}
}

func TestMockPackageManager_Update(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	mock.AddInstalledPackage("test-package", "1.0.0")
	ctx := context.Background()

	op, err := mock.Update(ctx, "test-package")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if !op.Success {
		t.Error("Operation should be successful")
	}

	if !op.Changed {
		t.Error("Operation should report change")
	}

	if op.OldVersion != "1.0.0" {
		t.Errorf("Expected old version '1.0.0', got '%s'", op.OldVersion)
	}

	if op.NewVersion == op.OldVersion {
		t.Error("New version should differ from old version")
	}
}

func TestMockPackageManager_Search(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	expectedResults := []PackageInfo{
		{Name: "package1", Version: "1.0.0", Description: "Test package 1"},
		{Name: "package2", Version: "2.0.0", Description: "Test package 2"},
	}

	mock.SetSearchResults(expectedResults)

	results, err := mock.Search(ctx, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, got %d", len(expectedResults), len(results))
	}
}

func TestMockPackageManager_ListInstalled(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	mock.AddInstalledPackage("package1", "1.0.0")
	mock.AddInstalledPackage("package2", "2.0.0")
	ctx := context.Background()

	installed, err := mock.ListInstalled(ctx)
	if err != nil {
		t.Fatalf("ListInstalled failed: %v", err)
	}

	if len(installed) != 2 {
		t.Errorf("Expected 2 installed packages, got %d", len(installed))
	}
}

func TestMockPackageManager_ListUpgradable(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	upgradable := []PackageInfo{
		{Name: "package1", Version: "1.0.0", NewVersion: "2.0.0", Upgradable: true},
	}

	mock.SetUpgradableList(upgradable)

	results, err := mock.ListUpgradable(ctx)
	if err != nil {
		t.Fatalf("ListUpgradable failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 upgradable package, got %d", len(results))
	}

	if !results[0].Upgradable {
		t.Error("Package should be marked as upgradable")
	}
}

func TestMockPackageManager_FailureScenarios(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	testError := fmt.Errorf("test error")

	tests := []struct {
		name      string
		operation string
		testFunc  func() error
	}{
		{
			name:      "Install failure",
			operation: "install",
			testFunc: func() error {
				_, err := mock.Install(ctx, "test", "1.0.0")
				return err
			},
		},
		{
			name:      "Remove failure",
			operation: "remove",
			testFunc: func() error {
				_, err := mock.Remove(ctx, "test")
				return err
			},
		},
		{
			name:      "Search failure",
			operation: "search",
			testFunc: func() error {
				_, err := mock.Search(ctx, "test")
				return err
			},
		},
		{
			name:      "Clean failure",
			operation: "clean",
			testFunc: func() error {
				return mock.Clean(ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.SetShouldFail(tt.operation, testError)
			err := tt.testFunc()
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.operation)
			}
			mock.SetShouldFail("", nil) // Reset
		})
	}
}

// ============================================================================
// BATCH OPERATIONS TESTS
// ============================================================================

func TestMockPackageManager_InstallMultiple(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	packages := []PackageSpec{
		{Name: "package1", Version: "1.0.0", State: "present"},
		{Name: "package2", Version: "2.0.0", State: "present"},
		{Name: "package3", Version: "3.0.0", State: "present"},
	}

	batch, err := mock.InstallMultiple(ctx, packages)
	if err != nil {
		t.Fatalf("InstallMultiple failed: %v", err)
	}

	if !batch.Success {
		t.Error("Batch operation should be successful")
	}

	if batch.TotalCount != 3 {
		t.Errorf("Expected 3 total operations, got %d", batch.TotalCount)
	}

	if batch.SuccessCount != 3 {
		t.Errorf("Expected 3 successful operations, got %d", batch.SuccessCount)
	}

	if batch.ChangedCount != 3 {
		t.Errorf("Expected 3 changed operations, got %d", batch.ChangedCount)
	}
}

func TestMockPackageManager_RemoveMultiple(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	mock.AddInstalledPackage("package1", "1.0.0")
	mock.AddInstalledPackage("package2", "2.0.0")
	ctx := context.Background()

	packages := []string{"package1", "package2", "package3"}

	batch, err := mock.RemoveMultiple(ctx, packages)
	if err != nil {
		t.Fatalf("RemoveMultiple failed: %v", err)
	}

	if !batch.Success {
		t.Error("Batch operation should be successful")
	}

	if batch.TotalCount != 3 {
		t.Errorf("Expected 3 total operations, got %d", batch.TotalCount)
	}

	if batch.ChangedCount != 2 {
		t.Errorf("Expected 2 changed operations (only installed packages), got %d", batch.ChangedCount)
	}
}

// ============================================================================
// EXTENDED METHODS TESTS
// ============================================================================

func TestMockPackageManager_Clean(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	err := mock.Clean(ctx)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
}

func TestMockPackageManager_AutoRemove(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	removed, err := mock.AutoRemove(ctx)
	if err != nil {
		t.Fatalf("AutoRemove failed: %v", err)
	}

	if len(removed) == 0 {
		t.Error("Expected some packages to be removed")
	}
}

func TestMockPackageManager_VerifyIntegrity(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	err := mock.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("VerifyIntegrity failed: %v", err)
	}
}

func TestMockPackageManager_GetDependencies(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	expectedDeps := []string{"dep1", "dep2", "dep3"}
	mock.SetDependencies("test-package", expectedDeps)

	deps, err := mock.GetDependencies(ctx, "test-package")
	if err != nil {
		t.Fatalf("GetDependencies failed: %v", err)
	}

	if len(deps) != len(expectedDeps) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedDeps), len(deps))
	}
}

func TestMockPackageManager_VerifyChecksum(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	valid, err := mock.VerifyChecksum(ctx, "test-package", "1.0.0")
	if err != nil {
		t.Fatalf("VerifyChecksum failed: %v", err)
	}

	if !valid {
		t.Error("Checksum should be valid")
	}
}

func TestMockPackageManager_DryRun(t *testing.T) {
	mock := NewMockUnifiedPackageManager()
	ctx := context.Background()

	preview, err := mock.DryRun(ctx, "install", "test-package")
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	if !preview.WillChange {
		t.Error("Preview should indicate changes")
	}

	if len(preview.Actions) == 0 {
		t.Error("Preview should contain actions")
	}
}

// ============================================================================
// ENTERPRISE FEATURES TESTS - SNAPSHOTS
// ============================================================================

func TestPackageModule_CreateSnapshot(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	// Add some installed packages
	mock.AddInstalledPackage("package1", "1.0.0")
	mock.AddInstalledPackage("package2", "2.0.0")

	ctx := context.Background()
	snapshot, err := module.CreateSnapshot(ctx, "Test snapshot")

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Snapshot should not be nil")
	}

	if snapshot.Description != "Test snapshot" {
		t.Errorf("Expected description 'Test snapshot', got '%s'", snapshot.Description)
	}

	if snapshot.ID == "" {
		t.Error("Snapshot ID should not be empty")
	}

	if snapshot.Checksum == "" {
		t.Error("Snapshot checksum should not be empty")
	}

	if len(snapshot.Packages) != 2 {
		t.Errorf("Expected 2 packages in snapshot, got %d", len(snapshot.Packages))
	}
}

func TestPackageModule_CreateSnapshot_EmptySystem(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	ctx := context.Background()
	snapshot, err := module.CreateSnapshot(ctx, "Empty system snapshot")

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if len(snapshot.Packages) != 0 {
		t.Errorf("Expected 0 packages in snapshot, got %d", len(snapshot.Packages))
	}
}

func TestPackageModule_CreateSnapshot_Error(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	mock.SetShouldFail("list_installed", fmt.Errorf("list error"))

	ctx := context.Background()
	_, err := module.CreateSnapshot(ctx, "Test snapshot")

	if err == nil {
		t.Error("Expected error when listing packages fails")
	}
}

func TestPackageModule_RestoreSnapshot(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	// Current state: package1 and package2 installed
	mock.AddInstalledPackage("package1", "1.0.0")
	mock.AddInstalledPackage("package2", "2.0.0")

	// Create snapshot with only package1
	snapshot := &SystemSnapshot{
		ID:          "test-snapshot",
		Timestamp:   time.Now(),
		Description: "Test snapshot",
		Packages: map[string]*PackageState{
			"package1": {
				Name:      "package1",
				Installed: true,
				Version:   "1.0.0",
			},
		},
		Checksum: "test-checksum",
	}

	ctx := context.Background()
	err := module.RestoreSnapshot(ctx, snapshot)

	if err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}

	// Verify package2 was removed
	if len(mock.removeCalls) == 0 {
		t.Error("Expected package2 to be removed")
	}
}

// ============================================================================
// ENTERPRISE FEATURES TESTS - PACKAGE GROUPS
// ============================================================================

func TestPackageModule_InstallGroup(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	group := &PackageGroup{
		Name:        "web-server",
		Description: "Web server packages",
		Packages:    []string{"nginx", "php", "mysql"},
		Optional:    []string{"redis", "memcached"},
	}

	ctx := context.Background()
	batch, err := module.InstallGroup(ctx, group, false)

	if err != nil {
		t.Fatalf("InstallGroup failed: %v", err)
	}

	if !batch.Success {
		t.Error("Batch operation should be successful")
	}

	if batch.TotalCount != 3 {
		t.Errorf("Expected 3 packages (without optional), got %d", batch.TotalCount)
	}
}

func TestPackageModule_InstallGroup_WithOptional(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	group := &PackageGroup{
		Name:        "web-server",
		Description: "Web server packages",
		Packages:    []string{"nginx", "php"},
		Optional:    []string{"redis"},
	}

	ctx := context.Background()
	batch, err := module.InstallGroup(ctx, group, true)

	if err != nil {
		t.Fatalf("InstallGroup failed: %v", err)
	}

	if batch.TotalCount != 3 {
		t.Errorf("Expected 3 packages (with optional), got %d", batch.TotalCount)
	}
}

func TestPackageModule_RemoveGroup(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	// Install packages first
	mock.AddInstalledPackage("nginx", "1.0.0")
	mock.AddInstalledPackage("php", "7.4.0")

	group := &PackageGroup{
		Name:     "web-server",
		Packages: []string{"nginx", "php"},
	}

	ctx := context.Background()
	batch, err := module.RemoveGroup(ctx, group)

	if err != nil {
		t.Fatalf("RemoveGroup failed: %v", err)
	}

	if !batch.Success {
		t.Error("Batch operation should be successful")
	}

	if batch.ChangedCount != 2 {
		t.Errorf("Expected 2 packages removed, got %d", batch.ChangedCount)
	}
}

// ============================================================================
// ENTERPRISE FEATURES TESTS - HEALTH CHECKS
// ============================================================================

func TestPackageModule_PerformHealthCheck(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	// Set up a healthy system
	mock.AddInstalledPackage("package1", "1.0.0")
	mock.AddInstalledPackage("package2", "2.0.0")

	healthResult := &HealthCheckResult{
		Healthy:   true,
		CheckedAt: time.Now(),
	}
	mock.SetHealthResult(healthResult)

	ctx := context.Background()
	result, err := module.PerformHealthCheck(ctx)

	if err != nil {
		t.Fatalf("PerformHealthCheck failed: %v", err)
	}

	if !result.Healthy {
		t.Error("System should be healthy")
	}

	if len(result.Issues) > 0 {
		t.Errorf("Expected no issues, got %d", len(result.Issues))
	}
}

func TestPackageModule_PerformHealthCheck_WithIssues(t *testing.T) {
	module := NewUnifiedPackageModule()
	mock := NewMockUnifiedPackageManager()
	module.packageManager = mock

	// Make VerifyIntegrity fail to trigger health issues
	mock.SetShouldFail("verify_integrity", fmt.Errorf("integrity check failed"))

	ctx := context.Background()
	result, err := module.PerformHealthCheck(ctx)

	if err != nil {
		t.Fatalf("PerformHealthCheck failed: %v", err)
	}

	if result.Healthy {
		t.Error("System should not be healthy when integrity check fails")
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues to be reported")
	}
}

// ============================================================================
// PACKAGE LOCK FILE TESTS
// ============================================================================

func TestPackageLockFile_UpdatePackage(t *testing.T) {
	lock := &PackageLockFile{
		Version:  "1.0",
		Packages: make(map[string]PackageLockEntry),
	}

	lock.UpdatePackage("test-package", "1.0.0", "sha256-abc123", []string{"dep1", "dep2"})

	entry, exists := lock.Packages["test-package"]
	if !exists {
		t.Fatal("Package should exist in lock file")
	}

	if entry.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", entry.Version)
	}

	if entry.Integrity != "sha256-abc123" {
		t.Errorf("Expected integrity 'sha256-abc123', got '%s'", entry.Integrity)
	}

	if len(entry.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(entry.Dependencies))
	}
}

// ============================================================================
// PACKAGE METRICS TESTS
// ============================================================================

func TestPackageMetrics_RecordOperation(t *testing.T) {
	metrics := &PackageMetrics{}

	op1 := &PackageOperation{
		Package:   "test1",
		Operation: "install",
		Success:   true,
		Changed:   true,
		Duration:  100 * time.Millisecond,
	}
	op2 := &PackageOperation{
		Package:   "test2",
		Operation: "install",
		Success:   true,
		Changed:   true,
		Duration:  200 * time.Millisecond,
	}
	op3 := &PackageOperation{
		Package:   "test3",
		Operation: "install",
		Success:   false,
		Duration:  150 * time.Millisecond,
	}

	metrics.RecordOperation(op1)
	metrics.RecordOperation(op2)
	metrics.RecordOperation(op3)

	metricsData := metrics.GetMetrics()

	if metricsData.TotalOperations != 3 {
		t.Errorf("Expected 3 total operations, got %d", metricsData.TotalOperations)
	}

	if metricsData.SuccessfulOps != 2 {
		t.Errorf("Expected 2 successful operations, got %d", metricsData.SuccessfulOps)
	}

	if metricsData.FailedOps != 1 {
		t.Errorf("Expected 1 failed operation, got %d", metricsData.FailedOps)
	}

	// PackagesInstalled is only incremented for successful AND changed operations
	if metricsData.PackagesInstalled != 2 {
		t.Errorf("Expected 2 packages installed, got %d", metricsData.PackagesInstalled)
	}
}

func TestPackageMetrics_ConcurrentAccess(t *testing.T) {
	metrics := &PackageMetrics{}
	var wg sync.WaitGroup

	// Concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op := &PackageOperation{
				Package:   "test",
				Operation: "install",
				Success:   true,
				Duration:  10 * time.Millisecond,
			}
			metrics.RecordOperation(op)
		}()
	}

	wg.Wait()

	metricsData := metrics.GetMetrics()
	if metricsData.TotalOperations != 100 {
		t.Errorf("Expected 100 operations, got %d", metricsData.TotalOperations)
	}
}

// ============================================================================
// Validate() Method Tests
// ============================================================================

func TestValidate_MissingName(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"state": "present",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected error for missing name parameter")
	}

	if err.Error() != "name parameter is required" {
		t.Errorf("Expected 'name parameter is required', got '%s'", err.Error())
	}
}

func TestValidate_ValidStates(t *testing.T) {
	module := NewUnifiedPackageModule()
	validStates := []string{"present", "absent", "latest"}

	for _, state := range validStates {
		args := map[string]interface{}{
			"name":  "nginx",
			"state": state,
		}

		err := module.Validate(args)
		if err != nil {
			t.Errorf("Expected no error for valid state '%s', got: %v", state, err)
		}
	}
}

func TestValidate_InvalidState(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name":  "nginx",
		"state": "invalid_state",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected error for invalid state")
	}

	expectedMsg := "invalid state: invalid_state (must be one of: present, absent, latest)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidate_NoStateProvided(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name": "nginx",
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Expected no error when state is not provided, got: %v", err)
	}
}

// ============================================================================
// parsePackageSpecs() Tests
// ============================================================================

func TestParsePackageSpecs_SingleString(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name":    "nginx",
		"version": "1.18.0",
		"state":   "present",
	}

	specs, err := module.parsePackageSpecs(args)
	if err != nil {
		t.Fatalf("parsePackageSpecs failed: %v", err)
	}

	if len(specs) != 1 {
		t.Fatalf("Expected 1 package spec, got %d", len(specs))
	}

	if specs[0].Name != "nginx" {
		t.Errorf("Expected name 'nginx', got '%s'", specs[0].Name)
	}

	if specs[0].Version != "1.18.0" {
		t.Errorf("Expected version '1.18.0', got '%s'", specs[0].Version)
	}

	if specs[0].State != "present" {
		t.Errorf("Expected state 'present', got '%s'", specs[0].State)
	}
}

func TestParsePackageSpecs_ListOfStrings(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name":  []interface{}{"nginx", "redis", "postgresql"},
		"state": "present",
	}

	specs, err := module.parsePackageSpecs(args)
	if err != nil {
		t.Fatalf("parsePackageSpecs failed: %v", err)
	}

	if len(specs) != 3 {
		t.Fatalf("Expected 3 package specs, got %d", len(specs))
	}

	expectedNames := []string{"nginx", "redis", "postgresql"}
	for i, spec := range specs {
		if spec.Name != expectedNames[i] {
			t.Errorf("Expected name '%s', got '%s'", expectedNames[i], spec.Name)
		}
		if spec.State != "present" {
			t.Errorf("Expected state 'present', got '%s'", spec.State)
		}
	}
}

func TestParsePackageSpecs_ListOfObjects(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name": []interface{}{
			map[string]interface{}{"name": "nginx", "version": "1.18.0"},
			map[string]interface{}{"name": "redis", "state": "latest"},
			map[string]interface{}{"name": "postgresql"},
		},
		"state": "present",
	}

	specs, err := module.parsePackageSpecs(args)
	if err != nil {
		t.Fatalf("parsePackageSpecs failed: %v", err)
	}

	if len(specs) != 3 {
		t.Fatalf("Expected 3 package specs, got %d", len(specs))
	}

	// Check first package
	if specs[0].Name != "nginx" || specs[0].Version != "1.18.0" {
		t.Errorf("First package incorrect: %+v", specs[0])
	}

	// Check second package
	if specs[1].Name != "redis" || specs[1].State != "latest" {
		t.Errorf("Second package incorrect: %+v", specs[1])
	}

	// Check third package
	if specs[2].Name != "postgresql" || specs[2].State != "present" {
		t.Errorf("Third package incorrect: %+v", specs[2])
	}
}

func TestParsePackageSpecs_MissingName(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"state": "present",
	}

	_, err := module.parsePackageSpecs(args)
	if err == nil {
		t.Error("Expected error for missing name parameter")
	}
}

func TestParsePackageSpecs_InvalidType(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name":  12345, // Invalid type
		"state": "present",
	}

	_, err := module.parsePackageSpecs(args)
	if err == nil {
		t.Error("Expected error for invalid name type")
	}
}

func TestParsePackageSpecs_EmptyList(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name":  []interface{}{},
		"state": "present",
	}

	_, err := module.parsePackageSpecs(args)
	if err == nil {
		t.Error("Expected error for empty package list")
	}

	if err.Error() != "at least one package name is required" {
		t.Errorf("Expected 'at least one package name is required', got '%s'", err.Error())
	}
}

func TestParsePackageSpecs_InvalidObjectInList(t *testing.T) {
	module := NewUnifiedPackageModule()
	args := map[string]interface{}{
		"name": []interface{}{
			map[string]interface{}{"version": "1.18.0"}, // Missing 'name' field
		},
		"state": "present",
	}

	_, err := module.parsePackageSpecs(args)
	if err == nil {
		t.Error("Expected error for object without name field")
	}
}

// ============================================================================
// Additional Method Tests
// ============================================================================

func TestCacheDelete(t *testing.T) {
	cache := NewPackageStateCache(5 * time.Minute)

	state := &PackageState{
		Name:      "nginx",
		Installed: true,
		Version:   "1.18.0",
	}

	cache.Set("nginx", state)

	// Verify it's there
	retrieved, found := cache.Get("nginx")
	if !found || retrieved == nil {
		t.Fatal("Package should be in cache")
	}

	// Delete it
	cache.Delete("nginx")

	// Verify it's gone
	_, found = cache.Get("nginx")
	if found {
		t.Error("Package should be deleted from cache")
	}
}

func TestFailResult(t *testing.T) {
	module := NewUnifiedPackageModule()

	result := types.TaskResult{
		TaskName:  "package",
		Host:      "test-host",
		Module:    "package",
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	failedResult, err := module.failResult(result, "test error message")

	if err == nil {
		t.Error("Expected error to be returned")
	}

	if err.Error() != "test error message" {
		t.Errorf("Expected error message 'test error message', got '%s'", err.Error())
	}

	if failedResult.Success {
		t.Error("Expected Success to be false")
	}

	if !failedResult.Failed {
		t.Error("Expected Failed to be true")
	}

	if failedResult.Error != "test error message" {
		t.Errorf("Expected Error field to be 'test error message', got '%s'", failedResult.Error)
	}

	if failedResult.Duration == 0 {
		t.Error("Expected Duration to be set")
	}
}

func TestLogAuditEntry(t *testing.T) {
	module := NewUnifiedPackageModule()

	entry := &AuditEntry{
		Timestamp: time.Now(),
		User:      "testuser",
		Host:      "test-host",
		Operation: "install",
		Package:   "nginx",
		Success:   true,
		Details:   map[string]string{"version": "1.18.0"},
	}

	err := module.LogAuditEntry(entry)
	if err != nil {
		t.Errorf("LogAuditEntry failed: %v", err)
	}
}

func TestLogAuditEntry_WithDetails(t *testing.T) {
	module := NewUnifiedPackageModule()

	entry := &AuditEntry{
		Timestamp: time.Now(),
		User:      "testuser",
		Host:      "test-host",
		Operation: "update",
		Package:   "nginx",
		Success:   true,
		Details: map[string]string{
			"old_version": "1.18.0",
			"new_version": "1.19.0",
			"repository":  "ubuntu",
		},
	}

	err := module.LogAuditEntry(entry)
	if err != nil {
		t.Errorf("LogAuditEntry should handle valid data: %v", err)
	}
}

func TestGenerateStateHash(t *testing.T) {
	// Test that same inputs produce same hash
	hash1 := generateStateHash("nginx", "1.18.0", "ubuntu")
	hash2 := generateStateHash("nginx", "1.18.0", "ubuntu")

	if hash1 != hash2 {
		t.Error("Same inputs should produce same hash")
	}

	// Test that different inputs produce different hashes
	hash3 := generateStateHash("nginx", "1.19.0", "ubuntu")
	if hash1 == hash3 {
		t.Error("Different versions should produce different hashes")
	}

	hash4 := generateStateHash("redis", "1.18.0", "ubuntu")
	if hash1 == hash4 {
		t.Error("Different packages should produce different hashes")
	}

	hash5 := generateStateHash("nginx", "1.18.0", "debian")
	if hash1 == hash5 {
		t.Error("Different repositories should produce different hashes")
	}

	// Test hash format (should be 64 hex characters for SHA256)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// Test that hash contains only hex characters
	for _, c := range hash1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Hash contains non-hex character: %c", c)
		}
	}
}

// ============================================================================
// Phase 3: Package Manager Tests (APT)
// ============================================================================

func TestNewUnifiedAptManager(t *testing.T) {
	// Test basic constructor - we can't easily test with real executor
	// but we can verify the structure is created correctly
	hostname := "test-host"

	manager := &UnifiedAptManager{
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, manager.hostname)
	}

	if manager.cache == nil {
		t.Error("Cache should be initialized")
	}
}

func TestParseAptAvailableVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Valid candidate version",
			input: `nginx:
  Installed: 1.18.0-0ubuntu1
  Candidate: 1.20.0-1ubuntu1
  Version table:`,
			expected: "1.20.0-1ubuntu1",
		},
		{
			name: "No candidate",
			input: `nginx:
  Installed: 1.18.0-0ubuntu1
  Version table:`,
			expected: "",
		},
		{
			name:     "Empty output",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAptAvailableVersion(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseAptRepository(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Valid repository",
			input: `Package: nginx
Version: 1.18.0-0ubuntu1
APT-Sources: http://archive.ubuntu.com/ubuntu focal/main amd64 Packages
Description: high performance web server`,
			expected: "http://archive.ubuntu.com/ubuntu focal/main amd64 Packages",
		},
		{
			name: "No repository",
			input: `Package: nginx
Version: 1.18.0-0ubuntu1
Description: high performance web server`,
			expected: "",
		},
		{
			name:     "Empty output",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAptRepository(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseAptDependencies(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single dependency",
			input:    "libc6",
			expected: []string{"libc6"},
		},
		{
			name:     "Multiple dependencies",
			input:    "libc6, libssl1.1, zlib1g",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "Dependencies with version constraints",
			input:    "libc6 (>= 2.27), libssl1.1 (>= 1.1.0), zlib1g (>= 1:1.2.0)",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Whitespace only",
			input:    "   ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAptDependencies(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d dependencies, got %d", len(tt.expected), len(result))
				return
			}
			for i, dep := range result {
				if dep != tt.expected[i] {
					t.Errorf("Expected dependency %s at index %d, got %s", tt.expected[i], i, dep)
				}
			}
		})
	}
}

// ============================================================================
// Phase 3: Package Manager Tests (YUM)
// ============================================================================

func TestNewUnifiedYumManager(t *testing.T) {
	hostname := "test-host"

	manager := &UnifiedYumManager{
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, manager.hostname)
	}

	if manager.cache == nil {
		t.Error("Cache should be initialized")
	}
}

// ============================================================================
// Phase 3: Package Manager Tests (Brew)
// ============================================================================

func TestNewUnifiedBrewManager(t *testing.T) {
	hostname := "test-host"

	manager := &UnifiedBrewManager{
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, manager.hostname)
	}

	if manager.cache == nil {
		t.Error("Cache should be initialized")
	}
}

// ============================================================================
// Phase 3: Package Manager Tests (DNF)
// ============================================================================

func TestNewUnifiedDnfManager(t *testing.T) {
	hostname := "test-host"

	// DNF manager embeds YUM manager, so we need to create it properly
	yumManager := &UnifiedYumManager{
		executor: nil, // Would normally be a real executor
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}

	manager := &UnifiedDnfManager{
		UnifiedYumManager: yumManager,
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.UnifiedYumManager == nil {
		t.Fatal("Expected embedded YUM manager to be non-nil")
	}

	if manager.hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, manager.hostname)
	}

	if manager.cache == nil {
		t.Error("Cache should be initialized")
	}
}

// ============================================================================
// Phase 3.2: Lock File Operations Tests
// ============================================================================

func TestLoadLockFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, lockFile *PackageLockFile)
	}{
		{
			name: "non-existent file returns empty lock file",
			setupFile: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "non-existent.lock")
			},
			expectError: false,
			validate: func(t *testing.T, lockFile *PackageLockFile) {
				if lockFile.Version != "1.0" {
					t.Errorf("Expected version 1.0, got %s", lockFile.Version)
				}
				if len(lockFile.Packages) != 0 {
					t.Errorf("Expected empty packages map, got %d entries", len(lockFile.Packages))
				}
			},
		},
		{
			name: "valid lock file is parsed correctly",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				lockPath := filepath.Join(tmpDir, "test.lock")

				lockData := `{
  "version": "1.0",
  "created": "2024-01-01T00:00:00Z",
  "updated": "2024-01-02T00:00:00Z",
  "packages": {
    "nginx": {
      "version": "1.18.0",
      "resolved": "nginx@1.18.0",
      "integrity": "sha256:abc123",
      "dependencies": ["openssl", "pcre"]
    }
  }
}`
				if err := os.WriteFile(lockPath, []byte(lockData), 0600); err != nil {
					t.Fatalf("Failed to create test lock file: %v", err)
				}
				return lockPath
			},
			expectError: false,
			validate: func(t *testing.T, lockFile *PackageLockFile) {
				if lockFile.Version != "1.0" {
					t.Errorf("Expected version 1.0, got %s", lockFile.Version)
				}
				if len(lockFile.Packages) != 1 {
					t.Fatalf("Expected 1 package, got %d", len(lockFile.Packages))
				}

				nginx, exists := lockFile.Packages["nginx"]
				if !exists {
					t.Fatal("Expected nginx package to exist")
				}
				if nginx.Version != "1.18.0" {
					t.Errorf("Expected version 1.18.0, got %s", nginx.Version)
				}
				if nginx.Resolved != "nginx@1.18.0" {
					t.Errorf("Expected resolved nginx@1.18.0, got %s", nginx.Resolved)
				}
				if len(nginx.Dependencies) != 2 {
					t.Errorf("Expected 2 dependencies, got %d", len(nginx.Dependencies))
				}
			},
		},
		{
			name: "invalid JSON returns error",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				lockPath := filepath.Join(tmpDir, "invalid.lock")

				if err := os.WriteFile(lockPath, []byte("invalid json {{{"), 0600); err != nil {
					t.Fatalf("Failed to create test lock file: %v", err)
				}
				return lockPath
			},
			expectError: true,
			validate:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockPath := tt.setupFile(t)

			lockFile, err := LoadLockFile(lockPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if lockFile == nil {
				t.Fatal("Expected non-nil lock file")
			}

			if tt.validate != nil {
				tt.validate(t, lockFile)
			}
		})
	}
}

func TestPackageLockFile_Save(t *testing.T) {
	tests := []struct {
		name        string
		lockFile    *PackageLockFile
		setupPath   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, path string)
	}{
		{
			name: "save creates new file",
			lockFile: &PackageLockFile{
				Version: "1.0",
				Created: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Packages: map[string]PackageLockEntry{
					"nginx": {
						Version:      "1.18.0",
						Resolved:     "nginx@1.18.0",
						Integrity:    "sha256:abc123",
						Dependencies: []string{"openssl"},
					},
				},
			},
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new.lock")
			},
			expectError: false,
			validate: func(t *testing.T, path string) {
				// Verify file exists
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Error("Lock file was not created")
				}

				// Verify file can be read back
				lockFile, err := LoadLockFile(path)
				if err != nil {
					t.Fatalf("Failed to load saved lock file: %v", err)
				}

				if len(lockFile.Packages) != 1 {
					t.Errorf("Expected 1 package, got %d", len(lockFile.Packages))
				}

				nginx, exists := lockFile.Packages["nginx"]
				if !exists {
					t.Fatal("Expected nginx package to exist")
				}
				if nginx.Version != "1.18.0" {
					t.Errorf("Expected version 1.18.0, got %s", nginx.Version)
				}
			},
		},
		{
			name: "save creates directory if needed",
			lockFile: &PackageLockFile{
				Version:  "1.0",
				Created:  time.Now(),
				Packages: make(map[string]PackageLockEntry),
			},
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "subdir", "nested", "test.lock")
			},
			expectError: false,
			validate: func(t *testing.T, path string) {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Error("Lock file was not created in nested directory")
				}
			},
		},
		{
			name: "save updates existing file",
			lockFile: &PackageLockFile{
				Version: "1.0",
				Created: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Packages: map[string]PackageLockEntry{
					"apache2": {
						Version:  "2.4.41",
						Resolved: "apache2@2.4.41",
					},
				},
			},
			setupPath: func(t *testing.T) string {
				tmpDir := t.TempDir()
				lockPath := filepath.Join(tmpDir, "existing.lock")

				// Create existing file
				existing := &PackageLockFile{
					Version: "1.0",
					Created: time.Now(),
					Packages: map[string]PackageLockEntry{
						"nginx": {Version: "1.18.0"},
					},
				}
				if err := existing.Save(lockPath); err != nil {
					t.Fatalf("Failed to create existing lock file: %v", err)
				}

				return lockPath
			},
			expectError: false,
			validate: func(t *testing.T, path string) {
				lockFile, err := LoadLockFile(path)
				if err != nil {
					t.Fatalf("Failed to load updated lock file: %v", err)
				}

				// Should have apache2, not nginx
				if _, exists := lockFile.Packages["apache2"]; !exists {
					t.Error("Expected apache2 package to exist")
				}
				if _, exists := lockFile.Packages["nginx"]; exists {
					t.Error("Expected nginx package to be replaced")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockPath := tt.setupPath(t)

			err := tt.lockFile.Save(lockPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, lockPath)
			}
		})
	}
}

// ============================================================================
// PHASE 3.4: HELPER FUNCTIONS TESTS
// ============================================================================

// TestParsePackageSpecs tests the parsePackageSpecs function
func TestParsePackageSpecs(t *testing.T) {
	module := &UnifiedPackageModule{}

	tests := []struct {
		name        string
		args        map[string]interface{}
		expected    []PackageSpec
		expectError bool
		errorMsg    string
	}{
		{
			name: "single package name as string",
			args: map[string]interface{}{
				"name": "nginx",
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
			},
		},
		{
			name: "single package with version and state",
			args: map[string]interface{}{
				"name":    "nginx",
				"version": "1.18.0",
				"state":   "latest",
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "1.18.0", State: "latest"},
			},
		},
		{
			name: "list of string package names",
			args: map[string]interface{}{
				"name": []interface{}{"nginx", "apache2", "mysql"},
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
				{Name: "apache2", Version: "", State: "present"},
				{Name: "mysql", Version: "", State: "present"},
			},
		},
		{
			name: "list of strings with global version",
			args: map[string]interface{}{
				"name":    []interface{}{"nginx", "apache2"},
				"version": "1.0.0",
				"state":   "absent",
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "1.0.0", State: "absent"},
				{Name: "apache2", Version: "1.0.0", State: "absent"},
			},
		},
		{
			name: "list of objects with individual versions",
			args: map[string]interface{}{
				"name": []interface{}{
					map[string]interface{}{"name": "nginx", "version": "1.18.0"},
					map[string]interface{}{"name": "apache2", "version": "2.4.41"},
				},
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "1.18.0", State: "present"},
				{Name: "apache2", Version: "2.4.41", State: "present"},
			},
		},
		{
			name: "list of objects with mixed specifications",
			args: map[string]interface{}{
				"name": []interface{}{
					map[string]interface{}{"name": "nginx", "version": "1.18.0", "state": "latest"},
					map[string]interface{}{"name": "apache2"},
				},
				"version": "default-version",
				"state":   "present",
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "1.18.0", State: "latest"},
				{Name: "apache2", Version: "default-version", State: "present"},
			},
		},
		{
			name: "native Go string slice",
			args: map[string]interface{}{
				"name": []string{"nginx", "apache2"},
			},
			expected: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
				{Name: "apache2", Version: "", State: "present"},
			},
		},
		{
			name:        "missing name parameter",
			args:        map[string]interface{}{},
			expectError: true,
			errorMsg:    "name parameter is required",
		},
		{
			name: "invalid type in list",
			args: map[string]interface{}{
				"name": []interface{}{"nginx", 123},
			},
			expectError: true,
			errorMsg:    "name[1] must be a string or object",
		},
		{
			name: "object without name field",
			args: map[string]interface{}{
				"name": []interface{}{
					map[string]interface{}{"version": "1.0.0"},
				},
			},
			expectError: true,
			errorMsg:    "name[0].name must be a string",
		},
		{
			name: "invalid name type",
			args: map[string]interface{}{
				"name": 123,
			},
			expectError: true,
			errorMsg:    "name parameter must be a string, list of strings, or list of objects",
		},
		{
			name: "empty list",
			args: map[string]interface{}{
				"name": []interface{}{},
			},
			expectError: true,
			errorMsg:    "at least one package name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specs, err := module.parsePackageSpecs(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got none", tt.errorMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(specs) != len(tt.expected) {
				t.Fatalf("Expected %d specs, got %d", len(tt.expected), len(specs))
			}

			for i, spec := range specs {
				if spec.Name != tt.expected[i].Name {
					t.Errorf("Spec[%d].Name: expected %s, got %s", i, tt.expected[i].Name, spec.Name)
				}
				if spec.Version != tt.expected[i].Version {
					t.Errorf("Spec[%d].Version: expected %s, got %s", i, tt.expected[i].Version, spec.Version)
				}
				if spec.State != tt.expected[i].State {
					t.Errorf("Spec[%d].State: expected %s, got %s", i, tt.expected[i].State, spec.State)
				}
			}
		})
	}
}

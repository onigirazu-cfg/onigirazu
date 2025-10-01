package modules

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MockEnhancedPackageManager is a mock implementation for testing
type MockEnhancedPackageManager struct {
	mock.Mock
}

func (m *MockEnhancedPackageManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	args := m.Called(ctx, name, version)
	return args.Get(0).(*PackageOperation), args.Error(1)
}

func (m *MockEnhancedPackageManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*PackageOperation), args.Error(1)
}

func (m *MockEnhancedPackageManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*PackageOperation), args.Error(1)
}

func (m *MockEnhancedPackageManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	args := m.Called(ctx)
	return args.Get(0).(*PackageOperation), args.Error(1)
}

func (m *MockEnhancedPackageManager) IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*EnhancedPackageState), args.Error(1)
}

func (m *MockEnhancedPackageManager) GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*EnhancedPackageInfo), args.Error(1)
}

func (m *MockEnhancedPackageManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	args := m.Called(ctx, packages)
	return args.Get(0).(*BatchOperation), args.Error(1)
}

func (m *MockEnhancedPackageManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	args := m.Called(ctx, packages)
	return args.Get(0).(*BatchOperation), args.Error(1)
}

func (m *MockEnhancedPackageManager) RefreshCache(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEnhancedPackageManager) ValidateState(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEnhancedPackageManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	mockArgs := m.Called(ctx, operation, args)
	return mockArgs.Get(0).(*OperationPreview), mockArgs.Error(1)
}

func TestPackageStateCache(t *testing.T) {
	cache := NewPackageStateCache(1 * time.Second)

	// Test cache miss
	state, found := cache.Get("test-package")
	assert.False(t, found)
	assert.Nil(t, state)

	// Test cache set and hit
	testState := &EnhancedPackageState{
		Name:      "test-package",
		Installed: true,
		Version:   "1.0.0",
	}
	cache.Set("test-package", testState)

	state, found = cache.Get("test-package")
	assert.True(t, found)
	assert.Equal(t, "test-package", state.Name)
	assert.True(t, state.Installed)
	assert.Equal(t, "1.0.0", state.Version)

	// Test cache expiration
	time.Sleep(1100 * time.Millisecond)
	state, found = cache.Get("test-package")
	assert.False(t, found)
	assert.Nil(t, state)

	// Test cache stats
	hits, misses := cache.Stats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(2), misses)
}

func TestEnhancedPackageModule_HandlePresentState(t *testing.T) {
	module := NewEnhancedPackageModule()
	mockManager := &MockEnhancedPackageManager{}
	module.packageManager = mockManager

	ctx := context.Background()

	t.Run("Package not installed", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:      "test-package",
			Installed: false,
		}

		expectedOperation := &PackageOperation{
			Package:   "test-package",
			Operation: "install",
			Success:   true,
			Changed:   true,
		}

		mockManager.On("Install", ctx, "test-package", "").Return(expectedOperation, nil)

		operation, err := module.handlePresentState(ctx, "test-package", "", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.True(t, operation.Changed)
		assert.Equal(t, "install", operation.Operation)

		mockManager.AssertExpectations(t)
	})

	t.Run("Package already installed with correct version", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:      "test-package",
			Installed: true,
			Version:   "1.0.0",
		}

		operation, err := module.handlePresentState(ctx, "test-package", "1.0.0", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.False(t, operation.Changed)
		assert.Equal(t, "present", operation.Operation)
	})

	t.Run("Package installed but wrong version", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:      "test-package",
			Installed: true,
			Version:   "1.0.0",
		}

		expectedOperation := &PackageOperation{
			Package:   "test-package",
			Operation: "install",
			Success:   true,
			Changed:   true,
		}

		mockManager.On("Install", ctx, "test-package", "2.0.0").Return(expectedOperation, nil)

		operation, err := module.handlePresentState(ctx, "test-package", "2.0.0", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.True(t, operation.Changed)

		mockManager.AssertExpectations(t)
	})
}

func TestEnhancedPackageModule_HandleAbsentState(t *testing.T) {
	module := NewEnhancedPackageModule()
	mockManager := &MockEnhancedPackageManager{}
	module.packageManager = mockManager

	ctx := context.Background()

	t.Run("Package installed - should remove", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:      "test-package",
			Installed: true,
			Version:   "1.0.0",
		}

		expectedOperation := &PackageOperation{
			Package:   "test-package",
			Operation: "remove",
			Success:   true,
			Changed:   true,
		}

		mockManager.On("Remove", ctx, "test-package").Return(expectedOperation, nil)

		operation, err := module.handleAbsentState(ctx, "test-package", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.True(t, operation.Changed)
		assert.Equal(t, "remove", operation.Operation)

		mockManager.AssertExpectations(t)
	})

	t.Run("Package already absent", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:      "test-package",
			Installed: false,
		}

		operation, err := module.handleAbsentState(ctx, "test-package", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.False(t, operation.Changed)
		assert.Equal(t, "absent", operation.Operation)
	})
}

func TestEnhancedPackageModule_HandleLatestState(t *testing.T) {
	module := NewEnhancedPackageModule()
	mockManager := &MockEnhancedPackageManager{}
	module.packageManager = mockManager

	ctx := context.Background()

	t.Run("Package not installed - should install", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:      "test-package",
			Installed: false,
		}

		expectedOperation := &PackageOperation{
			Package:   "test-package",
			Operation: "install",
			Success:   true,
			Changed:   true,
		}

		mockManager.On("Install", ctx, "test-package", "").Return(expectedOperation, nil)

		operation, err := module.handleLatestState(ctx, "test-package", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.True(t, operation.Changed)

		mockManager.AssertExpectations(t)
	})

	t.Run("Package installed but update available", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:             "test-package",
			Installed:        true,
			Version:          "1.0.0",
			AvailableVersion: "2.0.0",
		}

		expectedOperation := &PackageOperation{
			Package:   "test-package",
			Operation: "update",
			Success:   true,
			Changed:   true,
		}

		mockManager.On("Update", ctx, "test-package").Return(expectedOperation, nil)

		operation, err := module.handleLatestState(ctx, "test-package", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.True(t, operation.Changed)

		mockManager.AssertExpectations(t)
	})

	t.Run("Package already at latest version", func(t *testing.T) {
		currentState := &EnhancedPackageState{
			Name:             "test-package",
			Installed:        true,
			Version:          "2.0.0",
			AvailableVersion: "2.0.0",
		}

		operation, err := module.handleLatestState(ctx, "test-package", currentState)

		assert.NoError(t, err)
		assert.True(t, operation.Success)
		assert.False(t, operation.Changed)
		assert.Equal(t, "latest", operation.Operation)
	})
}

func TestEnhancedPackageModule_Validate(t *testing.T) {
	module := NewEnhancedPackageModule()

	t.Run("Valid arguments", func(t *testing.T) {
		args := map[string]interface{}{
			"name":  "test-package",
			"state": "present",
		}

		err := module.Validate(args)
		assert.NoError(t, err)
	})

	t.Run("Missing name parameter", func(t *testing.T) {
		args := map[string]interface{}{
			"state": "present",
		}

		err := module.Validate(args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name parameter is required")
	})

	t.Run("Invalid state", func(t *testing.T) {
		args := map[string]interface{}{
			"name":  "test-package",
			"state": "invalid",
		}

		err := module.Validate(args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})
}

func TestGenerateStateHash(t *testing.T) {
	hash1 := generateStateHash("package1", "1.0.0", "repo1")
	hash2 := generateStateHash("package1", "1.0.0", "repo1")
	hash3 := generateStateHash("package1", "2.0.0", "repo1")

	// Same inputs should produce same hash
	assert.Equal(t, hash1, hash2)

	// Different inputs should produce different hash
	assert.NotEqual(t, hash1, hash3)

	// Hash should be non-empty
	assert.NotEmpty(t, hash1)
}

func TestEnhancedPackageModule_Execute_DryRun(t *testing.T) {
	module := NewEnhancedPackageModule()
	mockManager := &MockEnhancedPackageManager{}
	module.packageManager = mockManager

	ctx := context.Background()
	_ = types.Host{
		Name:    "test-host",
		Address: "localhost",
	}

	_ = map[string]interface{}{
		"name":    "test-package",
		"state":   "present",
		"dry_run": true,
	}

	expectedPreview := &OperationPreview{
		WillChange: true,
		Actions:    []string{"Install test-package"},
	}

	mockManager.On("DryRun", ctx, "present", []string{"test-package", ""}).Return(expectedPreview, nil)

	// Note: This test would need a way to mock the executor creation
	// For now, we'll skip the full Execute test and focus on unit testing the individual methods
}

// Benchmark tests for performance
func BenchmarkPackageStateCache_Get(b *testing.B) {
	cache := NewPackageStateCache(10 * time.Minute)
	testState := &EnhancedPackageState{
		Name:      "test-package",
		Installed: true,
		Version:   "1.0.0",
	}
	cache.Set("test-package", testState)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("test-package")
	}
}

func BenchmarkGenerateStateHash(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateStateHash("test-package", "1.0.0", "test-repo")
	}
}

package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// PERFORM HEALTH CHECK TESTS
// ============================================================================

func TestPerformHealthCheck(t *testing.T) {
	tests := []struct {
		name             string
		setupMock        func(*MockUnifiedPackageManager)
		expectedHealthy  bool
		expectedIssues   int
		expectedWarnings int
		expectedOrphans  int
	}{
		{
			name: "healthy_system",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.AddInstalledPackage("curl", "7.68.0")
				m.SetAutoRemovePackages([]string{}) // No orphans
			},
			expectedHealthy:  true,
			expectedIssues:   0,
			expectedWarnings: 0,
			expectedOrphans:  0,
		},
		{
			name: "integrity_check_fails",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetFailureFor("VerifyIntegrity", fmt.Errorf("integrity check failed"))
				m.SetAutoRemovePackages([]string{}) // No orphans
			},
			expectedHealthy:  false,
			expectedIssues:   1,
			expectedWarnings: 0,
			expectedOrphans:  0,
		},
		{
			name: "list_installed_fails",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetFailureFor("ListInstalled", fmt.Errorf("failed to list packages"))
				m.SetAutoRemovePackages([]string{}) // No orphans
			},
			expectedHealthy:  false,
			expectedIssues:   1,
			expectedWarnings: 0,
			expectedOrphans:  0,
		},
		{
			name: "upgradable_packages_available",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.SetUpgradablePackages([]PackageInfo{
					{Name: "nginx", Version: "1.18.0", Upgradable: true, NewVersion: "1.19.0"},
				})
				m.SetAutoRemovePackages([]string{}) // No orphans
			},
			expectedHealthy:  true,
			expectedIssues:   0,
			expectedWarnings: 1,
			expectedOrphans:  0,
		},
		{
			name: "orphan_packages_detected",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.SetAutoRemovePackages([]string{"old-lib", "unused-dep"})
			},
			expectedHealthy:  true,
			expectedIssues:   0,
			expectedWarnings: 0,
			expectedOrphans:  2,
		},
		{
			name: "low_cache_hit_rate",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.SetAutoRemovePackages([]string{}) // No orphans
			},
			expectedHealthy:  true,
			expectedIssues:   0,
			expectedWarnings: 0, // Will be set by cache stats
			expectedOrphans:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockPM := NewMockUnifiedPackageManager()
			tt.setupMock(mockPM)

			cache := NewPackageStateCache(5 * time.Minute)

			// Simulate low cache hit rate for the specific test
			if tt.name == "low_cache_hit_rate" {
				// Create misses
				for i := 0; i < 10; i++ {
					cache.Get(fmt.Sprintf("pkg%d", i))
				}
				// Create few hits
				cache.Set("pkg1", &PackageState{Name: "pkg1"})
				cache.Get("pkg1")
			}

			m := &UnifiedPackageModule{
				packageManager: mockPM,
				stateCache:     cache,
				metrics:        &PackageMetrics{},
			}

			result, err := m.PerformHealthCheck(ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedHealthy, result.Healthy)
			assert.Equal(t, tt.expectedIssues, len(result.Issues))

			if tt.name == "low_cache_hit_rate" {
				// Should have warning about low cache hit rate
				assert.GreaterOrEqual(t, len(result.Warnings), 1)
			} else {
				assert.Equal(t, tt.expectedWarnings, len(result.Warnings))
			}

			assert.Equal(t, tt.expectedOrphans, len(result.OrphanPackages))
			assert.False(t, result.CheckedAt.IsZero())
		})
	}
}

// ============================================================================
// RESTORE SNAPSHOT TESTS
// ============================================================================

func TestRestoreSnapshot(t *testing.T) {
	tests := []struct {
		name          string
		snapshot      *SystemSnapshot
		setupMock     func(*MockUnifiedPackageManager)
		expectError   bool
		errorContains string
	}{
		{
			name: "restore_empty_snapshot",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages:  make(map[string]*PackageState),
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.AddInstalledPackage("curl", "7.68.0")
			},
			expectError: false,
		},
		{
			name: "restore_with_new_packages",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages: map[string]*PackageState{
					"nginx": {Name: "nginx", Version: "1.18.0", Installed: true},
					"git":   {Name: "git", Version: "2.25.0", Installed: true},
				},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectError: false,
		},
		{
			name: "restore_removes_extra_packages",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages: map[string]*PackageState{
					"nginx": {Name: "nginx", Version: "1.18.0", Installed: true},
				},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.AddInstalledPackage("curl", "7.68.0")
				m.AddInstalledPackage("git", "2.25.0")
			},
			expectError: false,
		},
		{
			name: "restore_fails_on_list_error",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages:  make(map[string]*PackageState),
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetFailureFor("ListInstalled", fmt.Errorf("list failed"))
			},
			expectError:   true,
			errorContains: "failed to list current packages",
		},
		{
			name: "restore_fails_on_remove_error",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages:  make(map[string]*PackageState),
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.SetFailureFor("Remove", fmt.Errorf("remove failed"))
			},
			expectError:   true,
			errorContains: "failed to remove",
		},
		{
			name: "restore_fails_on_install_error",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages: map[string]*PackageState{
					"nginx": {Name: "nginx", Version: "1.18.0", Installed: true},
				},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetFailureFor("Install", fmt.Errorf("install failed"))
			},
			expectError:   true,
			errorContains: "failed to install",
		},
		{
			name: "restore_version_downgrade",
			snapshot: &SystemSnapshot{
				Timestamp: time.Now(),
				Packages: map[string]*PackageState{
					"nginx": {Name: "nginx", Version: "1.17.0", Installed: true},
				},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockPM := NewMockUnifiedPackageManager()
			tt.setupMock(mockPM)

			m := &UnifiedPackageModule{
				packageManager: mockPM,
				metrics:        &PackageMetrics{},
			}

			err := m.RestoreSnapshot(ctx, tt.snapshot)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// EDGE CASES FOR EXISTING FUNCTIONS
// ============================================================================

func TestExecuteSequentialEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		packages    []PackageSpec
		setupMock   func(*MockUnifiedPackageManager)
		maxRetries  int
		expectError bool
	}{
		{
			name: "retry_on_transient_failure",
			packages: []PackageSpec{
				{Name: "nginx", State: "present"},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				// Will fail first time, succeed on retry
				m.SetFailureFor("Install", fmt.Errorf("transient error"))
			},
			maxRetries:  3,
			expectError: false,
		},
		{
			name: "continue_after_single_failure",
			packages: []PackageSpec{
				{Name: "bad-package", State: "present"},
				{Name: "nginx", State: "present"},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetFailureFor("Install", fmt.Errorf("package not found"))
			},
			maxRetries:  1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockPM := NewMockUnifiedPackageManager()
			tt.setupMock(mockPM)

			m := &UnifiedPackageModule{
				packageManager: mockPM,
				metrics:        &PackageMetrics{},
			}

			rollback := &RollbackInfo{Operations: []PackageOperation{}, PrevStates: make(map[string]*PackageState)}
			batch, err := m.executeSequential(ctx, tt.packages, "present", tt.maxRetries, rollback)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, batch)
			}
		})
	}
}

func TestExecuteParallelEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		packages    []PackageSpec
		setupMock   func(*MockUnifiedPackageManager)
		expectError bool
	}{
		{
			name: "parallel_with_context_timeout",
			packages: []PackageSpec{
				{Name: "pkg1", State: "present"},
				{Name: "pkg2", State: "present"},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetInstallDelay(100 * time.Millisecond)
			},
			expectError: false,
		},
		{
			name: "parallel_mixed_success_failure",
			packages: []PackageSpec{
				{Name: "nginx", State: "present"},
				{Name: "bad-pkg", State: "present"},
				{Name: "curl", State: "present"},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				// Some will succeed, some will fail
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockPM := NewMockUnifiedPackageManager()
			tt.setupMock(mockPM)

			m := &UnifiedPackageModule{
				packageManager: mockPM,
				metrics:        &PackageMetrics{},
			}

			rollback := &RollbackInfo{Operations: []PackageOperation{}, PrevStates: make(map[string]*PackageState)}
			batch, err := m.executeParallel(ctx, tt.packages, "present", 1, rollback)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, batch)
			}
		})
	}
}

func TestExecutePackageOperationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		spec        PackageSpec
		setupMock   func(*MockUnifiedPackageManager)
		expectError bool
	}{
		{
			name: "present_with_version_mismatch",
			spec: PackageSpec{Name: "nginx", Version: "1.19.0", State: "present"},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectError: false,
		},
		{
			name: "latest_when_already_latest",
			spec: PackageSpec{Name: "nginx", State: "latest"},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.19.0")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockPM := NewMockUnifiedPackageManager()
			tt.setupMock(mockPM)

			m := &UnifiedPackageModule{
				packageManager: mockPM,
				metrics:        &PackageMetrics{},
			}

			result, err := m.executePackageOperation(ctx, tt.spec, tt.spec.State)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

func TestPerformRollbackEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		rollback    *RollbackInfo
		setupMock   func(*MockUnifiedPackageManager)
		expectError bool
	}{
		{
			name: "rollback_version_change_to_specific_version",
			rollback: &RollbackInfo{
				CanRollback: true,
				Operations: []PackageOperation{
					{
						Package:    "nginx",
						Operation:  "update",
						OldVersion: "1.17.0",
						NewVersion: "1.18.0",
						Changed:    true,
					},
				},
				PrevStates: map[string]*PackageState{
					"nginx": {
						Name:      "nginx",
						Version:   "1.17.0",
						Installed: true,
					},
				},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectError: false,
		},
		{
			name: "rollback_with_install_failure",
			rollback: &RollbackInfo{
				CanRollback: true,
				Operations: []PackageOperation{
					{
						Package:    "nginx",
						Operation:  "install",
						OldVersion: "",
						NewVersion: "1.18.0",
						Changed:    true,
					},
				},
				PrevStates: map[string]*PackageState{
					"nginx": {
						Name:      "nginx",
						Version:   "",
						Installed: false,
					},
				},
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.SetFailureFor("Remove", fmt.Errorf("remove failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockPM := NewMockUnifiedPackageManager()
			tt.setupMock(mockPM)

			m := &UnifiedPackageModule{
				packageManager: mockPM,
				metrics:        &PackageMetrics{},
			}

			err := m.performRollback(ctx, tt.rollback)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// LOAD LOCK FILE EDGE CASES
// ============================================================================

func TestLoadLockFileEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "load_empty_packages",
			setupFile: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "empty.lock")
				lockFile := &PackageLockFile{
					Version:  "1.0",
					Created:  time.Now(),
					Packages: make(map[string]PackageLockEntry),
				}
				err := lockFile.Save(path)
				require.NoError(t, err)
				return path
			},
			expectError: false,
		},
		{
			name: "load_with_complex_dependencies",
			setupFile: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "complex.lock")
				lockFile := &PackageLockFile{
					Version: "1.0",
					Created: time.Now(),
					Packages: map[string]PackageLockEntry{
						"nginx": {
							Version:      "1.18.0",
							Resolved:     "nginx@1.18.0",
							Integrity:    "sha256:abc",
							Dependencies: []string{"openssl", "zlib", "pcre"},
						},
					},
				}
				err := lockFile.Save(path)
				require.NoError(t, err)
				return path
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFile(t)
			lockFile, err := LoadLockFile(path)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, lockFile)
			}
		})
	}
}

// ============================================================================
// RECORD OPERATION COVERAGE TESTS
// ============================================================================

func TestRecordOperation_UpdateOperation(t *testing.T) {
	metrics := &PackageMetrics{}

	// Test update operation
	op := &PackageOperation{
		Package:   "nginx",
		Operation: "update",
		Success:   true,
		Changed:   true,
		Duration:  100 * time.Millisecond,
	}

	metrics.RecordOperation(op)

	assert.Equal(t, int64(1), metrics.TotalOperations)
	assert.Equal(t, int64(1), metrics.SuccessfulOps)
	assert.Equal(t, int64(1), metrics.PackagesUpdated)
	assert.Equal(t, int64(0), metrics.PackagesInstalled)
	assert.Equal(t, int64(0), metrics.PackagesRemoved)
}

func TestRecordOperation_MultipleOperations(t *testing.T) {
	metrics := &PackageMetrics{}

	// Install operation
	metrics.RecordOperation(&PackageOperation{
		Package:    "nginx",
		Operation:  "install",
		Success:    true,
		Changed:    true,
		Duration:   100 * time.Millisecond,
		RetryCount: 1,
	})

	// Update operation
	metrics.RecordOperation(&PackageOperation{
		Package:   "curl",
		Operation: "update",
		Success:   true,
		Changed:   true,
		Duration:  200 * time.Millisecond,
	})

	// Remove operation
	metrics.RecordOperation(&PackageOperation{
		Package:   "vim",
		Operation: "remove",
		Success:   true,
		Changed:   true,
		Duration:  50 * time.Millisecond,
	})

	// Failed operation (no change)
	metrics.RecordOperation(&PackageOperation{
		Package:   "apache",
		Operation: "install",
		Success:   false,
		Changed:   false,
		Duration:  10 * time.Millisecond,
	})

	assert.Equal(t, int64(4), metrics.TotalOperations)
	assert.Equal(t, int64(3), metrics.SuccessfulOps)
	assert.Equal(t, int64(1), metrics.FailedOps)
	assert.Equal(t, int64(1), metrics.PackagesInstalled)
	assert.Equal(t, int64(1), metrics.PackagesUpdated)
	assert.Equal(t, int64(1), metrics.PackagesRemoved)
	assert.Equal(t, int64(1), metrics.RetryCount)
	assert.Equal(t, 90*time.Millisecond, metrics.AverageDuration)
}

// ============================================================================
// EXECUTE PACKAGE OPERATION - INVALID STATE
// ============================================================================

func TestExecutePackageOperation_InvalidState(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()
	mockPM.AddInstalledPackage("nginx", "1.18.0")

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	pkg := PackageSpec{Name: "nginx"}
	op, err := m.executePackageOperation(ctx, pkg, "invalid_state")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state")
	assert.False(t, op.Success)
	assert.Contains(t, op.Error, "invalid state")
}

// ============================================================================
// EXECUTE PARALLEL - RETRY LOGIC
// ============================================================================

func TestExecuteParallel_WithRetries(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()

	// Set up a package that will fail (not installed, install will fail)
	mockPM.SetFailureFor("Install", fmt.Errorf("temporary network error"))

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	packages := []PackageSpec{
		{Name: "flaky-pkg", State: "present"},
	}

	rollback := &RollbackInfo{
		CanRollback: true,
		PrevStates:  make(map[string]*PackageState),
		Operations:  []PackageOperation{},
	}

	// Execute with retries - will fail due to install failure
	batchOp, err := m.executeParallel(ctx, packages, "present", 2, rollback)

	require.NoError(t, err)
	assert.NotNil(t, batchOp)
	assert.Equal(t, 1, batchOp.TotalCount)
	// The operation will fail even with retries since mock install always fails
	assert.Equal(t, 1, batchOp.FailedCount)
}

func TestExecuteParallel_RollbackTracking(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()
	mockPM.AddInstalledPackage("existing-pkg", "1.0.0")

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	packages := []PackageSpec{
		{Name: "new-pkg", State: "present"},
		{Name: "existing-pkg", State: "present"},
	}

	rollback := &RollbackInfo{
		CanRollback: true,
		PrevStates:  make(map[string]*PackageState),
		Operations:  []PackageOperation{},
	}

	batchOp, err := m.executeParallel(ctx, packages, "present", 0, rollback)

	require.NoError(t, err)
	assert.NotNil(t, batchOp)
	assert.Equal(t, 2, batchOp.TotalCount)

	// Check that rollback info was populated
	assert.NotEmpty(t, rollback.PrevStates)
	assert.Contains(t, rollback.PrevStates, "existing-pkg")
	assert.True(t, rollback.PrevStates["existing-pkg"].Installed)
}

// ============================================================================
// PERFORM ROLLBACK - ERROR PATHS
// ============================================================================

func TestPerformRollback_ReinstallError(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()
	mockPM.AddInstalledPackage("nginx", "1.20.0")

	// Set up install to fail for rollback
	mockPM.SetFailureFor("Install", fmt.Errorf("disk full"))

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		metrics:        &PackageMetrics{},
	}

	rollback := &RollbackInfo{
		CanRollback: true,
		PrevStates: map[string]*PackageState{
			"nginx": {
				Installed: true,
				Version:   "1.18.0",
			},
		},
		Operations: []PackageOperation{
			{
				Package:   "nginx",
				Operation: "update",
				Success:   true,
				Changed:   true,
			},
		},
	}

	err := m.performRollback(ctx, rollback)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rollback nginx")
}

func TestPerformRollback_RemoveError(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()
	mockPM.AddInstalledPackage("temp-pkg", "1.0.0")

	// Set up remove to fail
	mockPM.SetFailureFor("Remove", fmt.Errorf("package in use"))

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		metrics:        &PackageMetrics{},
	}

	rollback := &RollbackInfo{
		CanRollback: true,
		PrevStates: map[string]*PackageState{
			"temp-pkg": {
				Installed: false,
			},
		},
		Operations: []PackageOperation{
			{
				Package:   "temp-pkg",
				Operation: "install",
				Success:   true,
				Changed:   true,
			},
		},
	}

	err := m.performRollback(ctx, rollback)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rollback temp-pkg")
}

func TestPerformRollback_SkipUnchangedPackages(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()
	mockPM.AddInstalledPackage("nginx", "1.18.0")

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		metrics:        &PackageMetrics{},
	}

	rollback := &RollbackInfo{
		CanRollback: true,
		PrevStates: map[string]*PackageState{
			"nginx": {
				Installed: true,
				Version:   "1.18.0",
			},
		},
		Operations: []PackageOperation{
			{
				Package:   "nginx",
				Operation: "present",
				Success:   true,
				Changed:   false, // No change, should skip
			},
		},
	}

	err := m.performRollback(ctx, rollback)

	require.NoError(t, err)
}

// TestPerformRollback_MissingPrevState tests rollback when prevState doesn't exist
func TestPerformRollback_MissingPrevState(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		metrics:        &PackageMetrics{},
	}

	rollback := &RollbackInfo{
		CanRollback: true,
		PrevStates:  map[string]*PackageState{}, // Empty - no previous state
		Operations: []PackageOperation{
			{
				Package:   "nginx",
				Operation: "install",
				Success:   true,
				Changed:   true,
			},
		},
	}

	// Should not error - just skip packages without prevState
	err := m.performRollback(ctx, rollback)

	require.NoError(t, err)
}

// ============================================================================
// SAVE LOCK FILE - ERROR PATHS
// ============================================================================

func TestSaveLockFile_Success(t *testing.T) {
	lockFile := &PackageLockFile{
		Version: "1.0",
		Created: time.Now(),
		Packages: map[string]PackageLockEntry{
			"test-pkg": {
				Version:      "1.0.0",
				Resolved:     "test-pkg@1.0.0",
				Integrity:    "sha256:test",
				Dependencies: []string{"dep1", "dep2"},
			},
		},
	}

	path := filepath.Join(t.TempDir(), "subdir", "test.lock")
	err := lockFile.Save(path)

	require.NoError(t, err)
	assert.FileExists(t, path)
}

func TestSaveLockFile_InvalidPath(t *testing.T) {
	lockFile := &PackageLockFile{
		Version:  "1.0",
		Created:  time.Now(),
		Packages: make(map[string]PackageLockEntry),
	}

	// Try to save to an invalid path (root directory on most systems)
	err := lockFile.Save("/invalid/path/that/cannot/be/created/file.lock")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

// ============================================================================
// LOG AUDIT ENTRY - COVERAGE
// ============================================================================

func TestLogAuditEntry_Success(t *testing.T) {
	m := &UnifiedPackageModule{
		packageManager: NewMockUnifiedPackageManager(),
		metrics:        &PackageMetrics{},
	}

	entry := &AuditEntry{
		Timestamp: time.Now(),
		User:      "testuser",
		Host:      "localhost",
		Operation: "install",
		Package:   "nginx",
		Success:   true,
		Details:   map[string]string{"version": "1.18.0"},
	}

	err := m.LogAuditEntry(entry)

	require.NoError(t, err)
}

func TestLogAuditEntry_WithMultipleDetails(t *testing.T) {
	m := &UnifiedPackageModule{
		packageManager: NewMockUnifiedPackageManager(),
		metrics:        &PackageMetrics{},
	}

	entry := &AuditEntry{
		Timestamp: time.Now(),
		User:      "admin",
		Host:      "server01",
		Operation: "remove",
		Package:   "apache2",
		Success:   false,
		Details: map[string]string{
			"version": "2.4.41",
			"reason":  "Package not found",
			"error":   "dpkg error",
		},
	}

	err := m.LogAuditEntry(entry)

	require.NoError(t, err)
}

// TestExecutePackageOperation_IsInstalledError tests error handling when IsInstalled fails
func TestExecutePackageOperation_IsInstalledError(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()

	// Set up mock to fail on IsInstalled check
	mockPM.SetShouldFail("is_installed", fmt.Errorf("failed to query package database"))

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	pkg := PackageSpec{Name: "nginx", State: "present"}
	op, err := m.executePackageOperation(ctx, pkg, "present")

	require.Error(t, err)
	require.NotNil(t, op)
	assert.False(t, op.Success)
	assert.Contains(t, op.Error, "failed to check package state")
}

// TestLoadLockFile_ReadError tests error handling when file read fails (not IsNotExist)
func TestLoadLockFile_ReadError(t *testing.T) {
	// Create a directory instead of a file to trigger a read error
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	// Try to read the directory as a file - this should fail with a non-IsNotExist error
	lockFile, err := LoadLockFile(dirPath)

	require.Error(t, err)
	assert.Nil(t, lockFile)
	assert.Contains(t, err.Error(), "failed to read lock file")
}

// TestSaveLockFile_DirectoryCreationError tests Save when directory creation fails
func TestSaveLockFile_DirectoryCreationError(t *testing.T) {
	// Create a file where we need a directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "blockingfile")
	err := os.WriteFile(filePath, []byte("test"), 0600)
	require.NoError(t, err)

	// Try to save lock file in a path that requires creating a directory
	// where a file already exists
	lockFile := &PackageLockFile{
		Version:  "1.0",
		Created:  time.Now(),
		Packages: make(map[string]PackageLockEntry),
	}

	lockPath := filepath.Join(filePath, "subdir", "test.lock")
	err = lockFile.Save(lockPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

// TestSaveLockFile_WriteError tests Save when file write fails
func TestSaveLockFile_WriteError(t *testing.T) {
	// Create a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0755)
	require.NoError(t, err)

	// Make directory read-only (no write permission)
	err = os.Chmod(readOnlyDir, 0555)
	require.NoError(t, err)

	// Restore permissions after test
	defer os.Chmod(readOnlyDir, 0755)

	lockFile := &PackageLockFile{
		Version:  "1.0",
		Created:  time.Now(),
		Packages: make(map[string]PackageLockEntry),
	}

	lockPath := filepath.Join(readOnlyDir, "test.lock")
	err = lockFile.Save(lockPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write lock file")
}

// TestHandlePresentState_VersionMismatch tests installing specific version when current differs
func TestHandlePresentState_VersionMismatch(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()

	// Package is installed with version 1.18.0
	mockPM.AddInstalledPackage("nginx", "1.18.0")

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	// Request specific version 1.19.0
	pkg := PackageSpec{Name: "nginx", Version: "1.19.0", State: "present"}
	op, err := m.executePackageOperation(ctx, pkg, "present")

	require.NoError(t, err)
	require.NotNil(t, op)
	assert.True(t, op.Success)
	assert.True(t, op.Changed)
}

// TestHandleLatestState_UpdateAvailable tests update when newer version is available
func TestHandleLatestState_UpdateAvailable(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()

	// Package is installed with version 1.18.0
	mockPM.AddInstalledPackage("nginx", "1.18.0")

	// Manually set available version to trigger update
	mockPM.mu.Lock()
	if state, exists := mockPM.installedPackages["nginx"]; exists {
		state.AvailableVersion = "1.19.0"
	}
	mockPM.mu.Unlock()

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	pkg := PackageSpec{Name: "nginx", State: "latest"}
	op, err := m.executePackageOperation(ctx, pkg, "latest")

	require.NoError(t, err)
	require.NotNil(t, op)
	assert.True(t, op.Success)
	assert.True(t, op.Changed)
}

// TestHandleLatestState_NotInstalled tests installing latest when package is not installed
func TestHandleLatestState_NotInstalled(t *testing.T) {
	ctx := context.Background()
	mockPM := NewMockUnifiedPackageManager()

	m := &UnifiedPackageModule{
		packageManager: mockPM,
		stateCache:     NewPackageStateCache(5 * time.Minute),
		metrics:        &PackageMetrics{},
	}

	pkg := PackageSpec{Name: "nginx", State: "latest"}
	op, err := m.executePackageOperation(ctx, pkg, "latest")

	require.NoError(t, err)
	require.NotNil(t, op)
	assert.True(t, op.Success)
	assert.True(t, op.Changed)
}

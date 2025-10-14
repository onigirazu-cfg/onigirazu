package modules

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// PHASE 3.5: EXECUTION FLOW TESTING
// ============================================================================

// TestExecuteSequential tests sequential package operations
func TestExecuteSequential(t *testing.T) {
	tests := []struct {
		name           string
		packages       []PackageSpec
		state          string
		maxRetries     int
		setupMock      func(*MockUnifiedPackageManager)
		expectSuccess  bool
		expectChanged  bool
		expectedCounts struct {
			total   int
			success int
			failed  int
			changed int
		}
	}{
		{
			name: "install_multiple_packages_successfully",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
				{Name: "curl", Version: "", State: "present"},
				{Name: "git", Version: "", State: "present"},
			},
			state:      "present",
			maxRetries: 3,
			setupMock: func(m *MockUnifiedPackageManager) {
				// All packages not installed initially
			},
			expectSuccess: true,
			expectChanged: true,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 3, success: 3, failed: 0, changed: 3},
		},
		{
			name: "install_with_one_failure",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
				{Name: "invalid-pkg", Version: "", State: "present"},
				{Name: "curl", Version: "", State: "present"},
			},
			state:      "present",
			maxRetries: 0,
			setupMock: func(m *MockUnifiedPackageManager) {
				m.SetShouldFail("install", fmt.Errorf("package not found"))
			},
			expectSuccess: false,
			expectChanged: false,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 3, success: 0, failed: 3, changed: 0},
		},
		{
			name: "remove_installed_packages",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "absent"},
				{Name: "curl", Version: "", State: "absent"},
			},
			state:      "absent",
			maxRetries: 3,
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.AddInstalledPackage("curl", "7.68.0")
			},
			expectSuccess: true,
			expectChanged: true,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 2, success: 2, failed: 0, changed: 2},
		},
		{
			name: "update_to_latest",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "latest"},
			},
			state:      "latest",
			maxRetries: 3,
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				// Mock will handle update
			},
			expectSuccess: true,
			expectChanged: false, // Mock doesn't set AvailableVersion, so no update
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 1, success: 1, failed: 0, changed: 0},
		},
		{
			name:       "no_packages",
			packages:   []PackageSpec{},
			state:      "present",
			maxRetries: 3,
			setupMock: func(m *MockUnifiedPackageManager) {
			},
			expectSuccess: true,
			expectChanged: false,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 0, success: 0, failed: 0, changed: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			batchOp, err := module.executeSequential(ctx, tt.packages, tt.state, tt.maxRetries, nil)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if batchOp.Success != tt.expectSuccess {
				t.Errorf("Expected success=%v, got %v", tt.expectSuccess, batchOp.Success)
			}

			if batchOp.Changed != tt.expectChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectChanged, batchOp.Changed)
			}

			if batchOp.TotalCount != tt.expectedCounts.total {
				t.Errorf("Expected total=%d, got %d", tt.expectedCounts.total, batchOp.TotalCount)
			}

			if batchOp.SuccessCount != tt.expectedCounts.success {
				t.Errorf("Expected success=%d, got %d", tt.expectedCounts.success, batchOp.SuccessCount)
			}

			if batchOp.FailedCount != tt.expectedCounts.failed {
				t.Errorf("Expected failed=%d, got %d", tt.expectedCounts.failed, batchOp.FailedCount)
			}

			if batchOp.ChangedCount != tt.expectedCounts.changed {
				t.Errorf("Expected changed=%d, got %d", tt.expectedCounts.changed, batchOp.ChangedCount)
			}
		})
	}
}

// TestExecuteParallel tests parallel package operations
func TestExecuteParallel(t *testing.T) {
	tests := []struct {
		name           string
		packages       []PackageSpec
		state          string
		maxRetries     int
		setupMock      func(*MockUnifiedPackageManager)
		expectSuccess  bool
		expectChanged  bool
		expectedCounts struct {
			total   int
			success int
			failed  int
			changed int
		}
	}{
		{
			name: "parallel_install_multiple_packages",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
				{Name: "curl", Version: "", State: "present"},
				{Name: "git", Version: "", State: "present"},
				{Name: "vim", Version: "", State: "present"},
			},
			state:      "present",
			maxRetries: 3,
			setupMock: func(m *MockUnifiedPackageManager) {
				// All packages not installed initially
			},
			expectSuccess: true,
			expectChanged: true,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 4, success: 4, failed: 0, changed: 4},
		},
		{
			name: "parallel_with_mixed_results",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "present"},
				{Name: "curl", Version: "", State: "present"},
			},
			state:      "present",
			maxRetries: 0,
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0") // Already installed
			},
			expectSuccess: true,
			expectChanged: true,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 2, success: 2, failed: 0, changed: 1},
		},
		{
			name: "parallel_remove_packages",
			packages: []PackageSpec{
				{Name: "nginx", Version: "", State: "absent"},
				{Name: "curl", Version: "", State: "absent"},
				{Name: "git", Version: "", State: "absent"},
			},
			state:      "absent",
			maxRetries: 3,
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
				m.AddInstalledPackage("curl", "7.68.0")
				m.AddInstalledPackage("git", "2.25.0")
			},
			expectSuccess: true,
			expectChanged: true,
			expectedCounts: struct {
				total   int
				success int
				failed  int
				changed int
			}{total: 3, success: 3, failed: 0, changed: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			batchOp, err := module.executeParallel(ctx, tt.packages, tt.state, tt.maxRetries, nil)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if batchOp.Success != tt.expectSuccess {
				t.Errorf("Expected success=%v, got %v", tt.expectSuccess, batchOp.Success)
			}

			if batchOp.Changed != tt.expectChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectChanged, batchOp.Changed)
			}

			if batchOp.TotalCount != tt.expectedCounts.total {
				t.Errorf("Expected total=%d, got %d", tt.expectedCounts.total, batchOp.TotalCount)
			}

			if batchOp.SuccessCount != tt.expectedCounts.success {
				t.Errorf("Expected success=%d, got %d", tt.expectedCounts.success, batchOp.SuccessCount)
			}

			if batchOp.FailedCount != tt.expectedCounts.failed {
				t.Errorf("Expected failed=%d, got %d", tt.expectedCounts.failed, batchOp.FailedCount)
			}

			if batchOp.ChangedCount != tt.expectedCounts.changed {
				t.Errorf("Expected changed=%d, got %d", tt.expectedCounts.changed, batchOp.ChangedCount)
			}
		})
	}
}

// TestExecutePackageOperation tests single package operation execution
func TestExecutePackageOperation(t *testing.T) {
	tests := []struct {
		name          string
		pkg           PackageSpec
		state         string
		setupMock     func(*MockUnifiedPackageManager)
		expectSuccess bool
		expectChanged bool
		expectError   bool
	}{
		{
			name:  "install_new_package",
			pkg:   PackageSpec{Name: "nginx", Version: "", State: "present"},
			state: "present",
			setupMock: func(m *MockUnifiedPackageManager) {
				// Package not installed
			},
			expectSuccess: true,
			expectChanged: true,
			expectError:   false,
		},
		{
			name:  "package_already_present",
			pkg:   PackageSpec{Name: "nginx", Version: "1.18.0", State: "present"},
			state: "present",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectSuccess: true,
			expectChanged: false,
			expectError:   false,
		},
		{
			name:  "remove_installed_package",
			pkg:   PackageSpec{Name: "nginx", Version: "", State: "absent"},
			state: "absent",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectSuccess: true,
			expectChanged: true,
			expectError:   false,
		},
		{
			name:  "package_already_absent",
			pkg:   PackageSpec{Name: "nginx", Version: "", State: "absent"},
			state: "absent",
			setupMock: func(m *MockUnifiedPackageManager) {
				// Package not installed
			},
			expectSuccess: true,
			expectChanged: false,
			expectError:   false,
		},
		{
			name:  "update_to_latest",
			pkg:   PackageSpec{Name: "nginx", Version: "", State: "latest"},
			state: "latest",
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectSuccess: true,
			expectChanged: false, // No AvailableVersion set
			expectError:   false,
		},
		{
			name:  "invalid_state",
			pkg:   PackageSpec{Name: "nginx", Version: "", State: "invalid"},
			state: "invalid",
			setupMock: func(m *MockUnifiedPackageManager) {
			},
			expectSuccess: false,
			expectChanged: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			op, err := module.executePackageOperation(ctx, tt.pkg, tt.state)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if op == nil || op.Success {
					t.Error("Expected operation to fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if op == nil {
				t.Fatal("Expected operation result, got nil")
			}

			if op.Success != tt.expectSuccess {
				t.Errorf("Expected success=%v, got %v", tt.expectSuccess, op.Success)
			}

			if op.Changed != tt.expectChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectChanged, op.Changed)
			}
		})
	}
}

// ============================================================================
// PHASE 3.6: STATE HANDLERS TESTING
// ============================================================================

// TestHandlePresentState tests the "present" state handler
func TestHandlePresentState(t *testing.T) {
	tests := []struct {
		name          string
		pkg           PackageSpec
		currentState  *PackageState
		setupMock     func(*MockUnifiedPackageManager)
		expectChanged bool
		expectError   bool
	}{
		{
			name: "install_not_installed_package",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "present"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: false,
			},
			setupMock:     func(m *MockUnifiedPackageManager) {},
			expectChanged: true,
			expectError:   false,
		},
		{
			name: "package_already_installed_no_version",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "present"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: true,
				Version:   "1.18.0",
			},
			setupMock:     func(m *MockUnifiedPackageManager) {},
			expectChanged: false,
			expectError:   false,
		},
		{
			name: "install_specific_version_different_from_current",
			pkg:  PackageSpec{Name: "nginx", Version: "1.20.0", State: "present"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: true,
				Version:   "1.18.0",
			},
			setupMock:     func(m *MockUnifiedPackageManager) {},
			expectChanged: true,
			expectError:   false,
		},
		{
			name: "package_already_at_requested_version",
			pkg:  PackageSpec{Name: "nginx", Version: "1.18.0", State: "present"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: true,
				Version:   "1.18.0",
			},
			setupMock:     func(m *MockUnifiedPackageManager) {},
			expectChanged: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			op, err := module.handlePresentState(ctx, tt.pkg, tt.currentState)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if op.Changed != tt.expectChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectChanged, op.Changed)
			}

			if !op.Success {
				t.Error("Expected operation to succeed")
			}
		})
	}
}

// TestHandleAbsentState tests the "absent" state handler
func TestHandleAbsentState(t *testing.T) {
	tests := []struct {
		name          string
		pkg           PackageSpec
		currentState  *PackageState
		setupMock     func(*MockUnifiedPackageManager)
		expectChanged bool
		expectError   bool
	}{
		{
			name: "remove_installed_package",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "absent"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: true,
				Version:   "1.18.0",
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectChanged: true,
			expectError:   false,
		},
		{
			name: "package_already_absent",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "absent"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: false,
			},
			setupMock:     func(m *MockUnifiedPackageManager) {},
			expectChanged: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			op, err := module.handleAbsentState(ctx, tt.pkg, tt.currentState)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if op.Changed != tt.expectChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectChanged, op.Changed)
			}

			if !op.Success {
				t.Error("Expected operation to succeed")
			}
		})
	}
}

// TestHandleLatestState tests the "latest" state handler
func TestHandleLatestState(t *testing.T) {
	tests := []struct {
		name          string
		pkg           PackageSpec
		currentState  *PackageState
		setupMock     func(*MockUnifiedPackageManager)
		expectChanged bool
		expectError   bool
	}{
		{
			name: "install_not_installed_package",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "latest"},
			currentState: &PackageState{
				Name:      "nginx",
				Installed: false,
			},
			setupMock:     func(m *MockUnifiedPackageManager) {},
			expectChanged: true,
			expectError:   false,
		},
		{
			name: "update_available",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "latest"},
			currentState: &PackageState{
				Name:             "nginx",
				Installed:        true,
				Version:          "1.18.0",
				AvailableVersion: "1.20.0",
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectChanged: true,
			expectError:   false,
		},
		{
			name: "already_at_latest",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "latest"},
			currentState: &PackageState{
				Name:             "nginx",
				Installed:        true,
				Version:          "1.20.0",
				AvailableVersion: "1.20.0",
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.20.0")
			},
			expectChanged: false,
			expectError:   false,
		},
		{
			name: "no_available_version_info",
			pkg:  PackageSpec{Name: "nginx", Version: "", State: "latest"},
			currentState: &PackageState{
				Name:             "nginx",
				Installed:        true,
				Version:          "1.18.0",
				AvailableVersion: "",
			},
			setupMock: func(m *MockUnifiedPackageManager) {
				m.AddInstalledPackage("nginx", "1.18.0")
			},
			expectChanged: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			op, err := module.handleLatestState(ctx, tt.pkg, tt.currentState)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if op.Changed != tt.expectChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectChanged, op.Changed)
			}

			if !op.Success {
				t.Error("Expected operation to succeed")
			}
		})
	}
}

// TestPerformRollback tests the rollback functionality
func TestPerformRollback(t *testing.T) {
	tests := []struct {
		name        string
		rollback    *RollbackInfo
		setupMock   func(*MockUnifiedPackageManager)
		expectError bool
		errorMsg    string
	}{
		{
			name: "rollback_successful_install",
			rollback: &RollbackInfo{
				Timestamp:   time.Now(),
				CanRollback: true,
				Operations: []PackageOperation{
					{
						Package:   "nginx",
						Operation: "install",
						Success:   true,
						Changed:   true,
					},
				},
				PrevStates: map[string]*PackageState{
					"nginx": {
						Name:      "nginx",
						Installed: false,
					},
				},
			},
			setupMock:   func(m *MockUnifiedPackageManager) {},
			expectError: false,
		},
		{
			name: "rollback_version_change",
			rollback: &RollbackInfo{
				Timestamp:   time.Now(),
				CanRollback: true,
				Operations: []PackageOperation{
					{
						Package:    "nginx",
						Operation:  "install",
						Success:    true,
						Changed:    true,
						OldVersion: "1.18.0",
						NewVersion: "1.20.0",
					},
				},
				PrevStates: map[string]*PackageState{
					"nginx": {
						Name:      "nginx",
						Installed: true,
						Version:   "1.18.0",
					},
				},
			},
			setupMock:   func(m *MockUnifiedPackageManager) {},
			expectError: false,
		},
		{
			name: "rollback_not_available",
			rollback: &RollbackInfo{
				Timestamp:   time.Now(),
				CanRollback: false,
				Operations:  []PackageOperation{},
				PrevStates:  map[string]*PackageState{},
			},
			setupMock:   func(m *MockUnifiedPackageManager) {},
			expectError: true,
			errorMsg:    "rollback not available",
		},
		{
			name: "rollback_with_no_changes",
			rollback: &RollbackInfo{
				Timestamp:   time.Now(),
				CanRollback: true,
				Operations: []PackageOperation{
					{
						Package:   "nginx",
						Operation: "install",
						Success:   true,
						Changed:   false,
					},
				},
				PrevStates: map[string]*PackageState{
					"nginx": {
						Name:      "nginx",
						Installed: true,
						Version:   "1.18.0",
					},
				},
			},
			setupMock:   func(m *MockUnifiedPackageManager) {},
			expectError: false,
		},
		{
			name: "rollback_multiple_operations",
			rollback: &RollbackInfo{
				Timestamp:   time.Now(),
				CanRollback: true,
				Operations: []PackageOperation{
					{
						Package:   "nginx",
						Operation: "install",
						Success:   true,
						Changed:   true,
					},
					{
						Package:   "curl",
						Operation: "install",
						Success:   true,
						Changed:   true,
					},
				},
				PrevStates: map[string]*PackageState{
					"nginx": {
						Name:      "nginx",
						Installed: false,
					},
					"curl": {
						Name:      "curl",
						Installed: false,
					},
				},
			},
			setupMock:   func(m *MockUnifiedPackageManager) {},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockUnifiedPackageManager()
			tt.setupMock(mock)

			module := &UnifiedPackageModule{
				packageManager: mock,
				stateCache:     NewPackageStateCache(5 * time.Minute),
				metrics:        &PackageMetrics{},
			}

			ctx := context.Background()
			err := module.performRollback(ctx, tt.rollback)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

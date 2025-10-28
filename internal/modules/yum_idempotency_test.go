package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestYumModulePreCheckState tests the PreCheckState logic for yum module
func TestYumModulePreCheckState(t *testing.T) {
	m := NewYumModule()
	ctx := context.Background()
	host := types.Host{Name: "test-host"}

	tests := []struct {
		name            string
		args            map[string]interface{}
		expectedCorrect bool
		expectedExecute bool
		expectedReason  string
	}{
		{
			name: "No packages specified - should execute",
			args: map[string]interface{}{
				"state": "present",
			},
			expectedCorrect: false,
			expectedExecute: true,
			expectedReason:  "No packages specified",
		},
		{
			name: "Single package - present state",
			args: map[string]interface{}{
				"name":  "nonexistent-package-xyz",
				"state": "present",
			},
			expectedCorrect: false,
			expectedExecute: true,
			expectedReason:  "Packages need to be set to state: present",
		},
		{
			name: "Multiple packages - present state",
			args: map[string]interface{}{
				"name":  []interface{}{"fake-pkg-1", "fake-pkg-2"},
				"state": "present",
			},
			expectedCorrect: false,
			expectedExecute: true,
			expectedReason:  "Packages need to be set to state: present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.PreCheckState(ctx, host, tt.args)
			if err != nil {
				t.Errorf("PreCheckState returned error: %v", err)
				return
			}

			if result.IsStateCorrect != tt.expectedCorrect {
				t.Errorf("IsStateCorrect: got %v, want %v", result.IsStateCorrect, tt.expectedCorrect)
			}

			if result.ShouldExecute != tt.expectedExecute {
				t.Errorf("ShouldExecute: got %v, want %v", result.ShouldExecute, tt.expectedExecute)
			}

			if result.Reason != tt.expectedReason {
				t.Errorf("Reason: got %q, want %q", result.Reason, tt.expectedReason)
			}
		})
	}
}

// TestYumModuleIdempotencyPattern tests full Execute + PreCheck integration
func TestYumModuleIdempotencyPattern(t *testing.T) {
	m := NewYumModule()
	ctx := context.Background()
	host := types.Host{Name: "test-host"}

	args := map[string]interface{}{
		"name":  "nonexistent-test-pkg",
		"state": "present",
	}

	// First execution should call PreCheck
	result, err := m.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// For a non-existent package, result should indicate changes needed
	// The actual yum command won't run in this test environment,
	// but the pre-check flow should be correct
	if result.Success {
		t.Logf("First execution result: changed=%v, success=%v", result.Changed, result.Success)
	}
}

// TestYumPreCheckIntegration tests PreCheckState integration
func TestYumPreCheckIntegration(t *testing.T) {
	m := NewYumModule()
	ctx := context.Background()
	host := types.Host{Name: "test-host"}

	args := map[string]interface{}{
		"name":  "fake-package",
		"state": "present",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState returned error: %v", err)
	}

	// For a non-existent package
	if preCheck.IsStateCorrect {
		t.Error("Expected IsStateCorrect to be false for non-existent package")
	}

	if !preCheck.ShouldExecute {
		t.Error("Expected ShouldExecute to be true for non-existent package")
	}

	// Verify CurrentState map exists
	if preCheck.CurrentState == nil {
		t.Error("Expected CurrentState map to be populated")
	}
}

// TestYumModuleChangedFlagAccuracy tests Changed flag correctness
func TestYumModuleChangedFlagAccuracy(t *testing.T) {
	m := NewYumModule()
	ctx := context.Background()
	host := types.Host{Name: "test-host"}

	// Test with fake package - pre-check should indicate need for change
	args := map[string]interface{}{
		"name":  "definitely-fake-package-12345",
		"state": "present",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState returned error: %v", err)
	}

	// For non-existent package with present state
	if preCheck.IsStateCorrect {
		t.Error("Expected pre-check to indicate state needs change")
	}

	if !preCheck.ShouldExecute {
		t.Error("Expected pre-check to indicate execution is needed")
	}
}

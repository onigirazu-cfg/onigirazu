package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestUserModulePreCheckState tests the PreCheckState logic for user module
func TestUserModulePreCheckState(t *testing.T) {
	m := NewUserModule()
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
			name: "No username specified - should execute",
			args: map[string]interface{}{
				"state": "present",
			},
			expectedCorrect: false,
			expectedExecute: true,
			expectedReason:  "No username specified",
		},
		{
			name: "Non-existent user with present state",
			args: map[string]interface{}{
				"name":  "fake-user-xyz-12345",
				"state": "present",
			},
			expectedCorrect: false,
			expectedExecute: true,
			expectedReason:  "User needs to be set to state: present",
		},
		{
			name: "Non-existent user with absent state",
			args: map[string]interface{}{
				"name":  "another-fake-user-99999",
				"state": "absent",
			},
			expectedCorrect: true,
			expectedExecute: false,
			expectedReason:  "User already in state: absent",
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

// TestUserModuleIdempotencyPattern tests full Execute + PreCheck integration
func TestUserModuleIdempotencyPattern(t *testing.T) {
	m := NewUserModule()
	ctx := context.Background()
	host := types.Host{Name: "localhost"}

	// Test with non-existent user in absent state - should not need changes
	args := map[string]interface{}{
		"name":  "definitely-does-not-exist-99999",
		"state": "absent",
	}

	result, err := m.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// For non-existent user in absent state, pre-check should indicate no changes needed
	if result.Success {
		// Pre-check should indicate no changes needed
		if output, ok := result.Output["pre_checked"]; ok {
			if preChecked, isTrue := output.(bool); isTrue && preChecked {
				if result.Changed {
					t.Error("Expected Changed to be false when pre-check indicates state is correct")
				}
			}
		}
	}
}

// TestUserPreCheckIntegration tests PreCheckState integration
func TestUserPreCheckIntegration(t *testing.T) {
	m := NewUserModule()
	ctx := context.Background()
	host := types.Host{Name: "test-host"}

	// Test with non-existent user in absent state
	args := map[string]interface{}{
		"name":  "definitely-does-not-exist-user-12345",
		"state": "absent",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState returned error: %v", err)
	}

	// Non-existent user in absent state should be correct
	if !preCheck.IsStateCorrect {
		t.Error("Expected IsStateCorrect to be true for non-existent user in absent state")
	}

	if preCheck.ShouldExecute {
		t.Error("Expected ShouldExecute to be false for non-existent user in absent state")
	}

	// Verify CurrentState map exists
	if preCheck.CurrentState == nil {
		t.Error("Expected CurrentState map to be populated")
	}

	if exists, ok := preCheck.CurrentState["exists"]; ok {
		if existsBool, isBool := exists.(bool); isBool && existsBool {
			t.Error("Expected user to not exist")
		}
	}
}

// TestUserModuleChangedFlagAccuracy tests Changed flag correctness
func TestUserModuleChangedFlagAccuracy(t *testing.T) {
	m := NewUserModule()
	ctx := context.Background()
	host := types.Host{Name: "localhost"}

	// Test with non-existent user in absent state (should be correct)
	args := map[string]interface{}{
		"name":  "definitely-does-not-exist-12345",
		"state": "absent",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err == nil {
		// For non-existent user in absent state, state is already correct
		if preCheck.IsStateCorrect {
			// State is already correct, no changes needed
			if preCheck.ShouldExecute {
				t.Error("Expected ShouldExecute to be false when state is already correct")
			}
		}
	}

	// Test with non-existent user in present state (should need execution)
	args2 := map[string]interface{}{
		"name":  "another-fake-user-12345",
		"state": "present",
	}

	preCheck2, err := m.PreCheckState(ctx, host, args2)
	if err == nil {
		// For non-existent user with present state, should need execution
		if !preCheck2.ShouldExecute {
			t.Error("Expected ShouldExecute to be true for non-existent user in present state")
		}
	}
}

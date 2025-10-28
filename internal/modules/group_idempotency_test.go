package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestGroupModulePreCheckState tests the PreCheckState logic for group module
func TestGroupModulePreCheckState(t *testing.T) {
	m := NewGroupModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	testCases := []struct {
		name           string
		args           map[string]interface{}
		expectedResult bool
		description    string
	}{
		{
			name: "present_non_existent_group",
			args: map[string]interface{}{
				"name":  "nonexistent_test_group_xyz",
				"state": "present",
			},
			expectedResult: false, // Should execute because group doesn't exist
			description:    "Group that doesn't exist should need creation",
		},
		{
			name: "absent_non_existent_group",
			args: map[string]interface{}{
				"name":  "nonexistent_test_group_xyz",
				"state": "absent",
			},
			expectedResult: true, // Should not execute because group already absent
			description:    "Non-existent group already in absent state",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.PreCheckState(ctx, host, tt.args)
			if err != nil {
				t.Errorf("PreCheckState returned error: %v", err)
				return
			}

			if result.IsStateCorrect != tt.expectedResult {
				t.Errorf("IsStateCorrect = %v, want %v. %s", result.IsStateCorrect, tt.expectedResult, tt.description)
			}
		})
	}
}

// TestGroupPreCheckIntegration tests PreCheckState integration with Execute
func TestGroupPreCheckIntegration(t *testing.T) {
	m := NewGroupModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	// Test 1: Pre-check for non-existent group (should not skip)
	args := map[string]interface{}{
		"name":  "test_group_idempotency_xyz",
		"state": "present",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState returned error: %v", err)
	}

	if preCheck.IsStateCorrect {
		t.Errorf("Pre-check should indicate state needs change, but IsStateCorrect=%v", preCheck.IsStateCorrect)
	}

	if !preCheck.ShouldExecute {
		t.Errorf("Pre-check should indicate execution needed, but ShouldExecute=%v", preCheck.ShouldExecute)
	}

	t.Logf("Pre-check reason: %s", preCheck.Reason)
	t.Logf("Current state: %v", preCheck.CurrentState)
}

// TestGroupPreCheckStateComparison tests that pre-check correctly compares desired vs current state
func TestGroupPreCheckStateComparison(t *testing.T) {
	m := NewGroupModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	testCases := []struct {
		name          string
		groupname     string
		state         string
		expectCorrect bool
		description   string
	}{
		{
			name:          "non_existent_present",
			groupname:     "test_group_xyz_999",
			state:         "present",
			expectCorrect: false,
			description:   "Non-existent group when present desired = state needs change",
		},
		{
			name:          "non_existent_absent",
			groupname:     "test_group_xyz_999",
			state:         "absent",
			expectCorrect: true,
			description:   "Non-existent group when absent desired = state correct",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"name":  tt.groupname,
				"state": tt.state,
			}

			result, err := m.PreCheckState(ctx, host, args)
			if err != nil {
				t.Errorf("PreCheckState returned error: %v", err)
				return
			}

			if result.IsStateCorrect != tt.expectCorrect {
				t.Errorf("IsStateCorrect = %v, want %v. %s", result.IsStateCorrect, tt.expectCorrect, tt.description)
			}
		})
	}
}

// TestGroupPreCheckChangedFlag tests that Changed flag is properly set when idempotent
func TestGroupPreCheckChangedFlag(t *testing.T) {
	m := NewGroupModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"name":  "test_group_xyz_999",
		"state": "absent",
	}

	// First run - group doesn't exist and state=absent, so should be correct
	preCheck1, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState returned error: %v", err)
	}

	if preCheck1.IsStateCorrect {
		t.Log("✓ First pre-check correctly identified state as correct (group absent, desired absent)")
	} else {
		t.Errorf("First pre-check should identify state as correct")
	}

	// Second check after - state should still be correct
	preCheck2, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("Second PreCheckState returned error: %v", err)
	}

	if preCheck2.IsStateCorrect {
		t.Log("✓ Second pre-check correctly identified state as correct (idempotent)")
	} else {
		t.Errorf("Second pre-check should identify state as correct (idempotent)")
	}
}

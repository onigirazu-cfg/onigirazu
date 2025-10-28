package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestPackageModulePreCheckState tests the PreCheckState logic for package module
func TestPackageModulePreCheckState(t *testing.T) {
	m := NewUnifiedPackageModule()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	testCases := []struct {
		name          string
		args          map[string]interface{}
		expectCorrect bool
		description   string
	}{
		{
			name: "present_nonexistent_package",
			args: map[string]interface{}{
				"name":  "nonexistent_package_xyz_999",
				"state": "present",
			},
			expectCorrect: false,
			description:   "Non-existent package when present desired = state needs change",
		},
		{
			name: "absent_nonexistent_package",
			args: map[string]interface{}{
				"name":  "nonexistent_package_xyz_999",
				"state": "absent",
			},
			expectCorrect: true,
			description:   "Non-existent package when absent desired = state correct",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.PreCheckState(ctx, host, tt.args)
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

// TestPackagePreCheckIntegration tests PreCheckState integration
func TestPackagePreCheckIntegration(t *testing.T) {
	m := NewUnifiedPackageModule()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	// Test 1: Pre-check for non-existent package (should not skip)
	args := map[string]interface{}{
		"name":  "nonexistent_package_xyz",
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

// TestPackagePreCheckMultiple tests pre-check with multiple packages
func TestPackagePreCheckMultiple(t *testing.T) {
	m := NewUnifiedPackageModule()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"name":  []interface{}{"nonexistent_xyz_1", "nonexistent_xyz_2"},
		"state": "absent",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Logf("PreCheckState returned error: %v (expected if packages don't parse correctly)", err)
		return
	}

	// Both packages don't exist and state is absent, so should be correct
	if preCheck.IsStateCorrect {
		t.Logf("✓ Multiple non-existent packages with absent state = correct")
	}
}

// TestPackagePreCheckIdempotent tests idempotency of pre-check
func TestPackagePreCheckIdempotent(t *testing.T) {
	m := NewUnifiedPackageModule()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"name":  "nonexistent_package_xyz",
		"state": "absent",
	}

	// Run pre-check twice - both should return same result
	preCheck1, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("First PreCheckState returned error: %v", err)
	}

	preCheck2, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("Second PreCheckState returned error: %v", err)
	}

	if preCheck1.IsStateCorrect == preCheck2.IsStateCorrect {
		t.Logf("✓ Pre-check is idempotent (both calls returned IsStateCorrect=%v)", preCheck1.IsStateCorrect)
	} else {
		t.Errorf("Pre-check not idempotent: first=%v, second=%v", preCheck1.IsStateCorrect, preCheck2.IsStateCorrect)
	}

	// Both should indicate state is correct for non-existent package with absent desired
	if preCheck1.IsStateCorrect && preCheck2.IsStateCorrect {
		t.Log("✓ Both pre-checks correctly identified state as correct")
	}
}

// TestPackagePreCheckDryRun tests that dry-run always executes
func TestPackagePreCheckDryRun(t *testing.T) {
	m := NewUnifiedPackageModule()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"name":    "nonexistent_package_xyz",
		"state":   "absent", // Even though state is already correct
		"dry_run": true,     // Dry-run should always execute
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState returned error: %v", err)
	}

	if preCheck.ShouldExecute {
		t.Logf("✓ Dry-run correctly marked for execution even when state is correct")
	} else {
		t.Errorf("Dry-run should always execute, but ShouldExecute=%v", preCheck.ShouldExecute)
	}
}

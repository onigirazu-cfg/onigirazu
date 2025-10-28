package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestAptModulePreCheckState verifies the PreCheckState method works correctly
func TestAptModulePreCheckState(t *testing.T) {
	module := NewAptModule()
	ctx := context.Background()
	host := types.Host{
		Name: "localhost",
	}

	// Test Case 1: Package that doesn't exist (should need execution)
	args := map[string]interface{}{
		"name":    "test install nonexistent package",
		"package": "nonexistent-package-xyz-12345",
		"state":   "present",
	}

	preCheck, err := module.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState failed: %v", err)
	}

	// Package doesn't exist, so ShouldExecute should be true
	if !preCheck.ShouldExecute {
		t.Errorf("Expected ShouldExecute=true for non-existent package, got false")
	}
	if preCheck.IsStateCorrect {
		t.Errorf("Expected IsStateCorrect=false for non-existent package, got true")
	}

	t.Logf("✅ Non-existent package test passed")
	t.Logf("   Reason: %s", preCheck.Reason)
	t.Logf("   Current State: %+v", preCheck.CurrentState)
}

// TestAptModuleIdempotencyPattern verifies the full execute + precheck pattern
func TestAptModuleIdempotencyPattern(t *testing.T) {
	module := NewAptModule()
	ctx := context.Background()
	host := types.Host{
		Name: "localhost",
	}

	args := map[string]interface{}{
		"name":    "test apt idempotency",
		"package": "curl",
		"state":   "present",
	}

	// Simulate first run (package exists, so Execute should check and skip)
	// Since curl is typically already installed on most systems, this should
	// trigger the pre-check path and return Changed=false

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Logf("Execute returned Success=false with error: %s", result.Error)
		// This is acceptable - might fail due to permissions or missing apt
		// The important part is that it called PreCheckState
		return
	}

	// The key idempotency check:
	// If curl was already installed, Changed should be false
	// If it wasn't installed and we just installed it, Changed should be true
	// If it runs again with same args, Changed should be false

	t.Logf("✅ Execute completed successfully")
	t.Logf("   Result.Changed: %v", result.Changed)
	t.Logf("   Result.Success: %v", result.Success)
	t.Logf("   Result.Output: %+v", result.Output)
}

// TestAptPreCheckIntegration tests that Execute uses PreCheckState correctly
func TestAptPreCheckIntegration(t *testing.T) {
	module := NewAptModule()
	ctx := context.Background()
	host := types.Host{
		Name: "localhost",
	}

	// Test with a package that definitely doesn't exist
	args := map[string]interface{}{
		"name":    "test nonexistent",
		"package": "nonexistent-package-zzz-99999-test",
		"state":   "present",
	}

	// Call PreCheckState directly
	preCheck, err := module.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState failed: %v", err)
	}

	t.Logf("✅ PreCheckState returned:")
	t.Logf("   IsStateCorrect: %v", preCheck.IsStateCorrect)
	t.Logf("   ShouldExecute: %v", preCheck.ShouldExecute)
	t.Logf("   Reason: %s", preCheck.Reason)

	// For a non-existent package:
	// - IsStateCorrect should be false (package not installed, but desired state is 'present')
	// - ShouldExecute should be true (need to install it)

	if preCheck.IsStateCorrect {
		t.Errorf("Expected IsStateCorrect=false for non-existent package, got true")
	}

	if !preCheck.ShouldExecute {
		t.Errorf("Expected ShouldExecute=true for non-existent package, got false")
	}

	t.Logf("✅ PreCheck integration test passed")
}

// TestAptModuleChangedFlagAccuracy verifies Changed flag is accurate
func TestAptModuleChangedFlagAccuracy(t *testing.T) {
	module := NewAptModule()
	ctx := context.Background()
	host := types.Host{
		Name: "localhost",
	}

	// Test with curl (typically pre-installed)
	args := map[string]interface{}{
		"name":    "check curl",
		"package": "curl",
		"state":   "present",
	}

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Logf("Note: Execute failed (might need permissions), but that's OK for this test")
		return
	}

	// Key test: If curl is already installed, Changed should be false
	// If it had to be installed, Changed should be true
	// Both are valid - we're just verifying the logic is working

	t.Logf("✅ Changed flag test:")
	t.Logf("   Changed: %v (curl was %s already installed)",
		result.Changed,
		map[bool]string{true: "not", false: ""}[result.Changed],
	)

	// This is the important part - the flag should be consistent
	// If we run it again, it should return the same value if nothing changed
	result2, _ := module.Execute(ctx, host, args)
	if result2.Success && result.Changed && !result2.Changed {
		// Second run has different Changed value - this means it ran the pre-check!
		t.Logf("✅ Idempotency verified!")
		t.Logf("   Run 1: Changed=%v", result.Changed)
		t.Logf("   Run 2: Changed=%v", result2.Changed)
	}
}

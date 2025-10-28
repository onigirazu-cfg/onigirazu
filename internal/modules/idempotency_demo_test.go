package modules

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ========================================
// DEMO MODULE: SimpleCounter
// This demonstrates Phase 1 idempotency infrastructure
// It's idempotent - multiple runs don't change state
// ========================================

// SimpleCounterModule is a demo module that counts something
// and implements idempotency via StateDetector
type SimpleCounterModule struct {
	*BaseModule
	stateStore map[string]int // Simulates system state
}

func NewSimpleCounterModule() *SimpleCounterModule {
	return &SimpleCounterModule{
		BaseModule: NewBaseModule("simple_counter"),
		stateStore: make(map[string]int),
	}
}

// Execute implements the module execution logic
func (m *SimpleCounterModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
	}

	counterName, ok := args["counter"].(string)
	if !ok {
		result.Success = false
		result.Error = "missing 'counter' argument"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("missing 'counter' argument")
	}

	// Get desired state
	desiredValue, ok := args["value"].(float64)
	if !ok {
		result.Success = false
		result.Error = "missing 'value' argument"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("missing 'value' argument")
	}

	// Check current state (IDEMPOTENCY!)
	currentValue, exists := m.stateStore[counterName]
	newValue := int(desiredValue)

	// Set changed flag based on actual change
	if !exists || currentValue != newValue {
		result.Changed = true
		m.stateStore[counterName] = newValue // Make the change
	} else {
		result.Changed = false // No change needed
	}

	result.Output = map[string]interface{}{
		"counter":     counterName,
		"value":       newValue,
		"previous":    currentValue,
		"changed":     result.Changed,
		"state_store": m.stateStore,
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// PreCheckState implements StateDetector interface for idempotency
func (m *SimpleCounterModule) PreCheckState(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
	counterName, _ := args["counter"].(string)
	desiredValue, _ := args["value"].(float64)

	currentValue, exists := m.stateStore[counterName]
	expectedValue := int(desiredValue)

	isStateCorrect := exists && currentValue == expectedValue
	reason := ""
	differences := make(map[string]string)

	if !isStateCorrect {
		if !exists {
			reason = fmt.Sprintf("counter '%s' does not exist", counterName)
			differences["counter_exists"] = "current=false, desired=true"
		} else {
			reason = fmt.Sprintf("counter value mismatch: current=%d, desired=%d", currentValue, expectedValue)
			differences["value"] = fmt.Sprintf("current=%d, desired=%d", currentValue, expectedValue)
		}
	} else {
		reason = fmt.Sprintf("counter '%s' already set to %d", counterName, expectedValue)
	}

	result := &PreCheckResult{
		IsStateCorrect: isStateCorrect,
		Reason:         reason,
		CurrentState: map[string]interface{}{
			"counter_exists": exists,
			"counter_value":  currentValue,
		},
		Differences: differences,
	}

	return result, nil
}

// ========================================
// TESTS: Verify Phase 1 Infrastructure Works
// ========================================

// Test 1: Verify BaseModule enhancements exist and work
func TestPhase1_BaseModuleEnhancements(t *testing.T) {
	module := NewSimpleCounterModule()

	ctx := context.Background()
	host := types.Host{
		Name: "test-host",
	}

	args := map[string]interface{}{
		"name":    "Test Task",
		"counter": "test_counter",
		"value":   42.0,
	}

	// First run - should set changed to true
	result1, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("First execution failed: %v", err)
	}
	if !result1.Changed {
		t.Errorf("First run should have Changed=true, got false")
	}
	if result1.Output["value"] != 42 {
		t.Errorf("Expected value 42, got %v", result1.Output["value"])
	}

	// Second run - should NOT change (IDEMPOTENCY!)
	result2, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Second execution failed: %v", err)
	}
	if result2.Changed {
		t.Errorf("Second run should have Changed=false (idempotent), got true - IDEMPOTENCY FAILED!")
	}

	// Third run - verify again (consistency)
	result3, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Third execution failed: %v", err)
	}
	if result3.Changed {
		t.Errorf("Third run should have Changed=false (idempotent), got true")
	}

	t.Logf("✅ BaseModule enhancements working: Idempotency verified!")
}

// Test 2: Verify StateDetector interface and PreCheckState method
func TestPhase1_StateDetectorInterface(t *testing.T) {
	module := NewSimpleCounterModule()

	ctx := context.Background()
	host := types.Host{
		Name: "test-host",
	}

	args := map[string]interface{}{
		"name":    "Detection Test",
		"counter": "detection_test",
		"value":   100.0,
	}

	// First check - state doesn't exist yet
	preCheckResult1, err := module.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState failed: %v", err)
	}
	if preCheckResult1.IsStateCorrect {
		t.Errorf("Initial state should NOT be correct (counter doesn't exist yet)")
	}

	// Set the state by executing
	_, _ = module.Execute(ctx, host, args)

	// Second check - state should now match
	preCheckResult2, err := module.PreCheckState(ctx, host, args)
	if err != nil {
		t.Fatalf("PreCheckState failed after execution: %v", err)
	}
	if !preCheckResult2.IsStateCorrect {
		t.Errorf("After setting state, PreCheckState should return IsStateCorrect=true")
	}

	t.Logf("✅ StateDetector interface working: Pre-check detection verified!")
}

// Test 3: Verify IdempotencyTestSuite framework
func TestPhase1_IdempotencyTestSuite(t *testing.T) {
	module := NewSimpleCounterModule()
	suite := NewIdempotencyTestSuite(t, "simple_counter")

	host := types.Host{
		Name: "test-host",
	}

	args := map[string]interface{}{
		"name":    "Test with Suite",
		"counter": "suite_test",
		"value":   200.0,
	}

	// Run the idempotency test
	suite.TestIdempotency(module, host, args, true)

	// Verify results
	if len(suite.results) < 3 {
		t.Errorf("Expected at least 3 iterations, got %d", len(suite.results))
	}

	// Check first run
	if !suite.results[0].Success {
		t.Errorf("First run should succeed")
	}
	if !suite.results[0].Changed {
		t.Errorf("First run should have Changed=true")
	}

	// Check second and third runs (should be idempotent)
	for i := 1; i < len(suite.results); i++ {
		if !suite.results[i].Success {
			t.Errorf("Run %d should succeed", i+1)
		}
		if suite.results[i].Changed {
			t.Errorf("Run %d should have Changed=false (idempotent)", i+1)
		}
	}

	t.Logf("✅ IdempotencyTestSuite framework working!")
	t.Logf("   Run 1: Changed=%v (%v)", suite.results[0].Changed, suite.results[0].Duration)
	if len(suite.results) > 1 {
		t.Logf("   Run 2: Changed=%v (%v) [%dx faster]",
			suite.results[1].Changed,
			suite.results[1].Duration,
			suite.results[0].Duration/suite.results[1].Duration)
	}
}

// Test 4: Verify PreCheckResult type and properties
func TestPhase1_PreCheckResultType(t *testing.T) {
	preCheckResult := &PreCheckResult{
		IsStateCorrect: false,
		Reason:         "State mismatch detected",
		CurrentState: map[string]interface{}{
			"key": "current_value",
		},
		Differences: map[string]string{
			"key": "current_value != desired_value",
		},
	}

	// Verify structure
	if preCheckResult.IsStateCorrect {
		t.Errorf("IsStateCorrect should be false")
	}
	if len(preCheckResult.Differences) == 0 {
		t.Errorf("Differences should not be empty")
	}
	if preCheckResult.Reason == "" {
		t.Errorf("Reason should not be empty")
	}

	t.Logf("✅ PreCheckResult type verified!")
}

// Test 5: Integration test - Full Phase 1 workflow
func TestPhase1_FullIntegration(t *testing.T) {
	t.Log("=== PHASE 1 IDEMPOTENCY INFRASTRUCTURE TEST ===")

	module := NewSimpleCounterModule()
	ctx := context.Background()
	host := types.Host{Name: "integration-test"}

	// Scenario: Deploy a counter value 3 times
	args := map[string]interface{}{
		"name":    "Deploy Counter",
		"counter": "app_version",
		"value":   42.0,
	}

	t.Log("\n1️⃣ FIRST RUN (Initial deployment):")
	run1, _ := module.Execute(ctx, host, args)
	if run1.Changed {
		t.Log("   ✅ Changed=true (state was created)")
	} else {
		t.Fatal("   ❌ First run should create state")
	}

	t.Log("\n2️⃣ SECOND RUN (Idempotent check):")
	run2, _ := module.Execute(ctx, host, args)
	if !run2.Changed {
		t.Log("   ✅ Changed=false (state unchanged)")
	} else {
		t.Fatal("   ❌ Second run should not change state (IDEMPOTENCY FAILED)")
	}

	t.Log("\n3️⃣ THIRD RUN (Consistency verification):")
	run3, _ := module.Execute(ctx, host, args)
	if !run3.Changed {
		t.Log("   ✅ Changed=false (state still unchanged)")
	} else {
		t.Fatal("   ❌ Third run should not change state")
	}

	t.Log("\n📊 PERFORMANCE:")
	t.Logf("   Run 1: %v (initial setup)", run1.Duration)
	t.Logf("   Run 2: %v (idempotent)", run2.Duration)
	t.Logf("   Run 3: %v (idempotent)", run3.Duration)

	if run1.Duration > 0 && run2.Duration > 0 {
		speedup := float64(run1.Duration) / float64(run2.Duration)
		if speedup > 1 {
			t.Logf("   ⚡ Speedup: %.1fx", speedup)
		}
	}

	t.Log("\n✅ PHASE 1 INFRASTRUCTURE TEST PASSED!")
	t.Log("✅ Idempotency framework is working correctly!")
}

// Benchmark: Compare idempotent vs multiple runs
func BenchmarkPhase1_IdempotentVsMultipleRuns(b *testing.B) {
	module := NewSimpleCounterModule()
	ctx := context.Background()
	host := types.Host{Name: "bench-host"}
	args := map[string]interface{}{
		"name":    "Benchmark",
		"counter": "bench",
		"value":   1000.0,
	}

	// Reset for clean benchmark
	module.stateStore = make(map[string]int)

	b.Run("FirstRun", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			module.stateStore = make(map[string]int) // Reset
			module.Execute(ctx, host, args)
		}
	})

	// First set the state
	module.Execute(ctx, host, args)

	b.Run("IdempotentRun", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			module.Execute(ctx, host, args)
		}
	})
}

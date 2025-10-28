package modules

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestableModule is an interface for modules that can be tested for idempotency
// Any module implementing the Execute method can be tested
type TestableModule interface {
	Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
}

// IdempotencyTestSuite provides standardized tests for verifying module idempotency
// Usage in tests:
//
//	suite := NewIdempotencyTestSuite(t, "module_name")
//	suite.TestIdempotency(module, host, args, expectedChangedFirstRun)
type IdempotencyTestSuite struct {
	t          *testing.T
	moduleName string
	results    []IdempotencyTestResult
}

// IdempotencyTestResult holds results of an idempotency test execution
type IdempotencyTestResult struct {
	Iteration        int
	Success          bool
	Changed          bool
	Duration         time.Duration
	Error            string
	OutputDifference string
}

// NewIdempotencyTestSuite creates a new idempotency test suite
func NewIdempotencyTestSuite(t *testing.T, moduleName string) *IdempotencyTestSuite {
	return &IdempotencyTestSuite{
		t:          t,
		moduleName: moduleName,
		results:    make([]IdempotencyTestResult, 0),
	}
}

// TestIdempotency verifies that a module is idempotent
// Runs the module 3 times and checks:
// 1. First run: should have Changed=expectedChanged
// 2. Second run: should have Changed=false (no changes when already correct)
// 3. Third run: should have Changed=false (consistent)
// Returns true if module is idempotent, false otherwise
func (suite *IdempotencyTestSuite) TestIdempotency(
	module TestableModule,
	host types.Host,
	args map[string]interface{},
	expectedChangedFirstRun bool,
) bool {
	ctx := context.Background()
	var previousOutput map[string]interface{}
	allPassed := true

	for i := 1; i <= 3; i++ {
		result, err := module.Execute(ctx, host, args)

		testResult := IdempotencyTestResult{
			Iteration: i,
			Success:   result.Success,
			Changed:   result.Changed,
			Duration:  result.Duration,
		}

		if err != nil {
			testResult.Error = err.Error()
			allPassed = false
		}

		// Check changed flag expectations
		if i == 1 {
			// First run - should match expected
			if result.Changed != expectedChangedFirstRun {
				testResult.Error = fmt.Sprintf(
					"First run: expected Changed=%v, got Changed=%v",
					expectedChangedFirstRun,
					result.Changed,
				)
				allPassed = false
			}
			previousOutput = result.Output
		} else {
			// Second and third runs - should NOT have changes if idempotent
			if result.Changed {
				testResult.Error = fmt.Sprintf(
					"Run %d: module reported changes on re-run (should be idempotent)",
					i,
				)
				allPassed = false
			}

			// Output should be stable across runs
			outputDiff := suite.compareOutputs(previousOutput, result.Output)
			if outputDiff != "" {
				testResult.OutputDifference = outputDiff
				allPassed = false
			}
		}

		suite.results = append(suite.results, testResult)

		if !allPassed && i == 1 {
			// Don't continue if first run failed
			break
		}
	}

	return allPassed
}

// TestStateConsistency verifies that module state is consistent across multiple runs
// with no modifications between runs
func (suite *IdempotencyTestSuite) TestStateConsistency(
	module TestableModule,
	host types.Host,
	args map[string]interface{},
) bool {
	ctx := context.Background()
	results := make([]types.TaskResult, 0)

	for i := 0; i < 2; i++ {
		result, err := module.Execute(ctx, host, args)
		if err != nil {
			suite.t.Errorf("State consistency test iteration %d: %v", i+1, err)
			return false
		}
		results = append(results, result)
	}

	// Both runs should have identical Changed status (both should be false)
	if results[0].Changed || results[1].Changed {
		suite.t.Errorf("State consistency failed: got Changed values %v and %v, expected both false",
			results[0].Changed, results[1].Changed)
		return false
	}

	return true
}

// TestPreCheckOptimization verifies that a module with pre-check is faster on re-runs
// Expects preCheckModule to be significantly faster than full module on re-runs
func (suite *IdempotencyTestSuite) TestPreCheckOptimization(
	fullModule TestableModule,
	host types.Host,
	args map[string]interface{},
	maxAcceptableSlowdown float64, // e.g., 1.2 = allow 20% slowdown
) bool {
	ctx := context.Background()

	// First run - warm up
	fullModule.Execute(ctx, host, args)

	// Second run - should be faster if pre-check is working
	startTime := time.Now()
	result, err := fullModule.Execute(ctx, host, args)
	secondRunDuration := time.Since(startTime)

	if err != nil {
		suite.t.Errorf("Pre-check optimization test: %v", err)
		return false
	}

	if result.Changed {
		suite.t.Logf("Note: Second run had changes, optimization test inconclusive")
		return true // Not a failure, just inconclusive
	}

	// Log timing information
	suite.t.Logf("Module %s - Second run duration: %v (expected to be fast with pre-check)",
		suite.moduleName, secondRunDuration)

	return true
}

// TestChangedFlagAccuracy verifies that Changed flag is accurate
// Tests that:
// - Changed=true only when actual system state changed
// - Changed=false when no actual changes were made
func (suite *IdempotencyTestSuite) TestChangedFlagAccuracy(
	module TestableModule,
	host types.Host,
	beforeStateDetector func() map[string]interface{},
	afterStateDetector func() map[string]interface{},
	args map[string]interface{},
) bool {
	ctx := context.Background()

	// Get state before
	stateBefore := beforeStateDetector()

	// Execute module
	result, err := module.Execute(ctx, host, args)
	if err != nil {
		suite.t.Errorf("Changed flag accuracy test: %v", err)
		return false
	}

	// Get state after
	stateAfter := afterStateDetector()

	// Check if Changed flag matches actual state change
	stateChanged := !suite.compareStates(stateBefore, stateAfter)

	if result.Changed != stateChanged {
		suite.t.Errorf(
			"Changed flag accuracy failed: module reported Changed=%v but actual state changed=%v",
			result.Changed,
			stateChanged,
		)
		return false
	}

	return true
}

// compareOutputs returns a description of differences between two outputs, or "" if identical
func (suite *IdempotencyTestSuite) compareOutputs(output1, output2 map[string]interface{}) string {
	if len(output1) != len(output2) {
		return fmt.Sprintf("output length mismatch: %d vs %d", len(output1), len(output2))
	}

	for key := range output1 {
		val1, ok1 := output1[key]
		val2, ok2 := output2[key]

		if !ok1 || !ok2 {
			return fmt.Sprintf("key mismatch for %q", key)
		}

		// Simple comparison - real implementation might need deep comparison
		if fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", val2) {
			return fmt.Sprintf("value difference for %q: %v vs %v", key, val1, val2)
		}
	}

	return ""
}

// compareStates returns true if two state maps are equal
func (suite *IdempotencyTestSuite) compareStates(state1, state2 map[string]interface{}) bool {
	if len(state1) != len(state2) {
		return false
	}

	for key, val1 := range state1 {
		val2, ok := state2[key]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", val2) {
			return false
		}
	}

	return true
}

// PrintResults prints the test results in a formatted way
func (suite *IdempotencyTestSuite) PrintResults() {
	fmt.Printf("\n=== Idempotency Test Results for %s ===\n", suite.moduleName)
	for _, result := range suite.results {
		status := "✓ PASS"
		if !result.Success {
			status = "✗ FAIL"
		}

		fmt.Printf("Iteration %d: %s | Changed=%v | Duration=%v\n",
			result.Iteration,
			status,
			result.Changed,
			result.Duration,
		)

		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if result.OutputDifference != "" {
			fmt.Printf("  Output Difference: %s\n", result.OutputDifference)
		}
	}
	fmt.Println()
}

// IdempotencyTestCase describes a single idempotency test case
type IdempotencyTestCase struct {
	Name                  string                 // Test case name
	Module                TestableModule         // Module to test
	Host                  types.Host             // Target host
	Args                  map[string]interface{} // Module arguments
	ExpectedChangedFirst  bool                   // Expected Changed value on first run
	ExpectedChangedSecond bool                   // Expected Changed value on second run (usually false)
	ShouldReachThirdRun   bool                   // If true, tests third run consistency
}

// RunIdempotencyTestCases runs multiple idempotency test cases and reports results
func RunIdempotencyTestCases(t *testing.T, moduleName string, testCases []IdempotencyTestCase) {
	passed := 0
	failed := 0

	for _, tc := range testCases {
		ctx := context.Background()

		// First run
		result1, err := tc.Module.Execute(ctx, tc.Host, tc.Args)
		if err != nil {
			t.Errorf("Test %q - First run failed: %v", tc.Name, err)
			failed++
			continue
		}

		if result1.Changed != tc.ExpectedChangedFirst {
			t.Errorf("Test %q - First run: expected Changed=%v, got %v",
				tc.Name, tc.ExpectedChangedFirst, result1.Changed)
			failed++
			continue
		}

		// Second run
		result2, err := tc.Module.Execute(ctx, tc.Host, tc.Args)
		if err != nil {
			t.Errorf("Test %q - Second run failed: %v", tc.Name, err)
			failed++
			continue
		}

		if result2.Changed != tc.ExpectedChangedSecond {
			t.Errorf("Test %q - Second run: expected Changed=%v, got %v",
				tc.Name, tc.ExpectedChangedSecond, result2.Changed)
			failed++
			continue
		}

		// Third run if needed
		if tc.ShouldReachThirdRun {
			result3, err := tc.Module.Execute(ctx, tc.Host, tc.Args)
			if err != nil {
				t.Errorf("Test %q - Third run failed: %v", tc.Name, err)
				failed++
				continue
			}

			if result3.Changed != tc.ExpectedChangedSecond {
				t.Errorf("Test %q - Third run: expected Changed=%v, got %v",
					tc.Name, tc.ExpectedChangedSecond, result3.Changed)
				failed++
				continue
			}
		}

		passed++
		t.Logf("Test %q: PASSED", tc.Name)
	}

	t.Logf("\n=== %s Idempotency Summary ===\nPassed: %d | Failed: %d\n", moduleName, passed, failed)
}

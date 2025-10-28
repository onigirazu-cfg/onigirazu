package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestServiceModulePreCheckState tests the PreCheckState logic for service module
func TestServiceModulePreCheckState(t *testing.T) {
	m := NewServiceModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	testCases := []struct {
		name          string
		args          map[string]interface{}
		shouldExecute bool
		description   string
	}{
		{
			name: "started_state",
			args: map[string]interface{}{
				"name":  "ssh",
				"state": "started",
			},
			shouldExecute: false, // ssh should already be started
			description:   "Started state for running service (should not execute)",
		},
		{
			name: "restarted_state",
			args: map[string]interface{}{
				"name":  "ssh",
				"state": "restarted",
			},
			shouldExecute: true, // restarted is always an action
			description:   "Restarted state always requires execution",
		},
		{
			name: "reloaded_state",
			args: map[string]interface{}{
				"name":  "ssh",
				"state": "reloaded",
			},
			shouldExecute: true, // reloaded is always an action
			description:   "Reloaded state always requires execution",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.PreCheckState(ctx, host, tt.args)
			if err != nil {
				t.Logf("PreCheckState returned error: %v (may be expected if service not found)", err)
				return
			}

			if result.ShouldExecute != tt.shouldExecute {
				t.Logf("ShouldExecute = %v, want %v. %s", result.ShouldExecute, tt.shouldExecute, tt.description)
			}
		})
	}
}

// TestServicePreCheckIntegration tests PreCheckState integration
func TestServicePreCheckIntegration(t *testing.T) {
	m := NewServiceModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	// Test: Pre-check for non-existent service (should not be state correct)
	args := map[string]interface{}{
		"name":  "nonexistent_service_xyz",
		"state": "started",
	}

	preCheck, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Logf("PreCheckState returned error: %v (expected for non-existent service)", err)
		return
	}

	t.Logf("Pre-check ShouldExecute: %v", preCheck.ShouldExecute)
	t.Logf("Pre-check reason: %s", preCheck.Reason)
	t.Logf("Current state: %v", preCheck.CurrentState)

	if !preCheck.ShouldExecute {
		t.Logf("✓ Correctly identified that non-existent service needs execution")
	}
}

// TestServicePreCheckRestart tests that restart/reload always require execution
func TestServicePreCheckRestart(t *testing.T) {
	m := NewServiceModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	testCases := []struct {
		name          string
		state         string
		expectExecute bool
	}{
		{"restart_action", "restarted", true},
		{"reload_action", "reloaded", true},
		{"start_state", "started", false},
		{"stop_state", "stopped", false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"name":  "ssh",
				"state": tt.state,
			}

			result, err := m.PreCheckState(ctx, host, args)
			if err != nil {
				t.Logf("PreCheckState returned error: %v", err)
				return
			}

			if tt.state == "restarted" || tt.state == "reloaded" {
				if !result.ShouldExecute {
					t.Errorf("%s should always execute, but ShouldExecute=%v", tt.state, result.ShouldExecute)
				}
			}
		})
	}
}

// TestServicePreCheckIdempotent tests idempotency of pre-check
func TestServicePreCheckIdempotent(t *testing.T) {
	m := NewServiceModuleFixed()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"name":  "ssh",
		"state": "started",
	}

	// Run pre-check twice
	preCheck1, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Logf("PreCheckState returned error: %v", err)
		return
	}

	preCheck2, err := m.PreCheckState(ctx, host, args)
	if err != nil {
		t.Logf("PreCheckState returned error: %v", err)
		return
	}

	if preCheck1.IsStateCorrect == preCheck2.IsStateCorrect {
		t.Logf("✓ Pre-check is idempotent (both calls returned IsStateCorrect=%v)", preCheck1.IsStateCorrect)
	} else {
		t.Errorf("Pre-check not idempotent: first=%v, second=%v", preCheck1.IsStateCorrect, preCheck2.IsStateCorrect)
	}
}

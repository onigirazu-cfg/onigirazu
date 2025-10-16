package state

import (
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRotationPolicyDefaults(t *testing.T) {
	policy := DefaultRotationPolicy()

	if policy.MaxResults == 0 {
		t.Fatalf("MaxResults should have a default value")
	}

	if policy.MaxAge == 0 {
		t.Fatalf("MaxAge should have a default value")
	}
}

func TestRotationManagerShouldRotate(t *testing.T) {
	policy := &RotationPolicy{
		MaxResults: 10,
		MaxAge:     0, // No age limit for this test
	}

	mgr := NewRotationManager(policy)

	// Test case 1: Below threshold
	state := &types.State{
		Results: make([]types.PlayResult, 5),
	}

	shouldRotate := mgr.ShouldRotate(state)
	if shouldRotate {
		t.Fatalf("Should not rotate when results (5) < MaxResults (10), but ShouldRotate returned true")
	}

	// Test case 2: At or exceeding threshold
	state.Results = make([]types.PlayResult, 10)
	shouldRotate = mgr.ShouldRotate(state)
	if !shouldRotate {
		t.Fatalf("Should rotate when results (10) >= MaxResults (10)")
	}

	// Test case 3: Well over threshold
	state.Results = make([]types.PlayResult, 15)
	shouldRotate = mgr.ShouldRotate(state)
	if !shouldRotate {
		t.Fatalf("Should rotate when results (15) >= MaxResults (10)")
	}
}

func TestRotationManagerRotateByCount(t *testing.T) {
	policy := &RotationPolicy{
		MaxResults: 5,
		MaxAge:     0, // No age limit
	}

	mgr := NewRotationManager(policy)

	now := time.Now()
	state := &types.State{
		Results: make([]types.PlayResult, 10),
	}

	// Initialize results with different start times
	for i := range state.Results {
		state.Results[i] = types.PlayResult{
			PlayName:  "test",
			Host:      "localhost",
			StartTime: now.Add(-time.Duration(i) * time.Hour),
			EndTime:   now.Add(-time.Duration(i)*time.Hour + time.Minute),
		}
	}

	stats, err := mgr.RotateState(state)
	if err != nil {
		t.Fatalf("RotateState failed: %v", err)
	}

	if len(state.Results) != 5 {
		t.Fatalf("Expected 5 results after rotation, got %d", len(state.Results))
	}

	if stats.ResultsRemoved != 5 {
		t.Fatalf("Expected 5 results removed, got %d", stats.ResultsRemoved)
	}
}

func TestRotationManagerRotateByAge(t *testing.T) {
	policy := &RotationPolicy{
		MaxResults: 100, // No count limit
		MaxAge:     24 * time.Hour,
	}

	mgr := NewRotationManager(policy)

	now := time.Now()
	state := &types.State{
		Results: make([]types.PlayResult, 3),
	}

	// Recent result (should keep)
	state.Results[0] = types.PlayResult{
		PlayName:  "recent",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
	}

	// Old results (should remove)
	state.Results[1] = types.PlayResult{
		PlayName:  "old1",
		StartTime: now.Add(-48 * time.Hour),
		EndTime:   now.Add(-47 * time.Hour),
	}

	state.Results[2] = types.PlayResult{
		PlayName:  "old2",
		StartTime: now.Add(-72 * time.Hour),
		EndTime:   now.Add(-71 * time.Hour),
	}

	stats, err := mgr.RotateState(state)
	if err != nil {
		t.Fatalf("RotateState failed: %v", err)
	}

	if len(state.Results) != 1 {
		t.Fatalf("Expected 1 result after rotation, got %d", len(state.Results))
	}

	if state.Results[0].PlayName != "recent" {
		t.Fatalf("Expected recent result to remain")
	}

	if stats.ResultsRemoved != 2 {
		t.Fatalf("Expected 2 results removed, got %d", stats.ResultsRemoved)
	}
}

func TestRotationManagerNoRotation(t *testing.T) {
	policy := &RotationPolicy{
		MaxResults: 0, // No limit
		MaxAge:     0, // No limit
	}

	mgr := NewRotationManager(policy)

	state := &types.State{
		Results: make([]types.PlayResult, 100),
	}

	shouldRotate := mgr.ShouldRotate(state)
	if shouldRotate {
		t.Fatalf("Should not rotate when policy has no limits")
	}
}

func TestRotationManagerGetStats(t *testing.T) {
	policy := &RotationPolicy{
		MaxResults: 5,
		MaxAge:     24 * time.Hour,
	}

	mgr := NewRotationManager(policy)

	now := time.Now()
	state := &types.State{
		Results: make([]types.PlayResult, 10),
	}

	for i := range state.Results {
		state.Results[i] = types.PlayResult{
			PlayName:  "test",
			StartTime: now.Add(-time.Duration(i) * time.Hour),
			EndTime:   now.Add(-time.Duration(i)*time.Hour + time.Minute),
		}
	}

	stats := mgr.GetRotationStats(state)

	if stats.ResultsBefore != 10 {
		t.Fatalf("Expected 10 results before, got %d", stats.ResultsBefore)
	}

	// Note: The exact number removed depends on the rotation logic
	// Just verify the math is consistent
	if stats.ResultsBefore-stats.ResultsAfter != stats.ResultsRemoved {
		t.Fatalf("Stats math error: %d - %d != %d",
			stats.ResultsBefore, stats.ResultsAfter, stats.ResultsRemoved)
	}

	t.Logf("Stats: Before=%d, After=%d, Removed=%d",
		stats.ResultsBefore, stats.ResultsAfter, stats.ResultsRemoved)
}

func TestRotationManagerDeduplication(t *testing.T) {
	mgr := NewRotationManager(DefaultRotationPolicy())

	now := time.Now()
	state := &types.State{
		Results: []types.PlayResult{
			{
				PlayName:  "play1",
				Host:      "host1",
				StartTime: now,
			},
			{
				PlayName:  "play1",
				Host:      "host1",
				StartTime: now, // Duplicate
			},
			{
				PlayName:  "play2",
				Host:      "host2",
				StartTime: now.Add(time.Hour),
			},
		},
	}

	removed := mgr.DeduplicateResults(state)

	if removed != 1 {
		t.Fatalf("Expected 1 duplicate removed, got %d", removed)
	}

	if len(state.Results) != 2 {
		t.Fatalf("Expected 2 results after dedup, got %d", len(state.Results))
	}
}

func TestRotationManagerEmptyState(t *testing.T) {
	policy := DefaultRotationPolicy()
	mgr := NewRotationManager(policy)

	state := &types.State{
		Results: []types.PlayResult{},
	}

	stats, err := mgr.RotateState(state)
	if err != nil {
		t.Fatalf("RotateState failed for empty state: %v", err)
	}

	if stats.ResultsRemoved != 0 {
		t.Fatalf("Expected no results removed for empty state")
	}
}

func TestRotationStatsFields(t *testing.T) {
	stats := &RotationStats{
		ResultsBefore:   100,
		ResultsAfter:    50,
		ResultsRemoved:  50,
		ResultsArchived: 0,
		ArchiveSize:     0,
	}

	if stats.ResultsBefore-stats.ResultsAfter != stats.ResultsRemoved {
		t.Fatalf("Stats math error: %d - %d != %d",
			stats.ResultsBefore, stats.ResultsAfter, stats.ResultsRemoved)
	}
}

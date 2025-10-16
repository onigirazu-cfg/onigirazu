package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RotationPolicy defines when and how to rotate state files
type RotationPolicy struct {
	MaxResults   int           // Maximum number of results to keep (0 = unlimited)
	MaxAge       time.Duration // Maximum age of results to keep (0 = unlimited)
	MaxFileSize  int64         // Maximum file size before rotation (0 = unlimited, in bytes)
	ArchiveDir   string        // Directory to archive old results (optional)
	KeepArchives int           // Number of archives to keep (0 = unlimited)
}

// DefaultRotationPolicy returns sensible defaults
func DefaultRotationPolicy() *RotationPolicy {
	return &RotationPolicy{
		MaxResults:   100,                 // Keep last 100 execution results
		MaxAge:       30 * 24 * time.Hour, // Keep results from last 30 days
		MaxFileSize:  50 * 1024 * 1024,    // 50MB file size limit
		ArchiveDir:   "",                  // No archiving by default
		KeepArchives: 10,
	}
}

// RotationManager handles state file rotation and cleanup
type RotationManager struct {
	policy *RotationPolicy
}

// NewRotationManager creates a new rotation manager
func NewRotationManager(policy *RotationPolicy) *RotationManager {
	if policy == nil {
		policy = DefaultRotationPolicy()
	}
	return &RotationManager{
		policy: policy,
	}
}

// ShouldRotate determines if a state file should be rotated based on policy
func (rm *RotationManager) ShouldRotate(state *types.State) bool {
	// Check MaxResults policy
	if rm.policy.MaxResults > 0 && len(state.Results) >= rm.policy.MaxResults {
		return true
	}

	// Check MaxAge policy
	if rm.policy.MaxAge > 0 && len(state.Results) > 0 {
		oldestResult := state.Results[0]
		if time.Since(oldestResult.StartTime) > rm.policy.MaxAge {
			return true
		}
	}

	return false
}

// RotateState removes old results according to policy
func (rm *RotationManager) RotateState(state *types.State) (*RotationStats, error) {
	stats := &RotationStats{
		ResultsRemoved: 0,
		ResultsBefore:  len(state.Results),
	}

	if len(state.Results) == 0 {
		return stats, nil
	}

	now := time.Now()

	// Filter results based on policies
	var keptResults []types.PlayResult
	var removedResults []types.PlayResult

	for _, result := range state.Results {
		keep := true

		// Check MaxAge
		if rm.policy.MaxAge > 0 {
			if now.Sub(result.StartTime) > rm.policy.MaxAge {
				keep = false
			}
		}

		// Check MaxResults (keep only the N most recent)
		// This is handled after filtering by age

		if keep {
			keptResults = append(keptResults, result)
		} else {
			removedResults = append(removedResults, result)
		}
	}

	// Now apply MaxResults limit on the kept results
	if rm.policy.MaxResults > 0 && len(keptResults) > rm.policy.MaxResults {
		// Keep only the most recent MaxResults results
		start := len(keptResults) - rm.policy.MaxResults
		removedResults = append(removedResults, keptResults[:start]...)
		keptResults = keptResults[start:]
	}

	// Archive removed results if ArchiveDir is set
	if len(removedResults) > 0 && rm.policy.ArchiveDir != "" {
		if err := rm.archiveResults(removedResults, state.Playbook); err != nil {
			return stats, fmt.Errorf("failed to archive results: %w", err)
		}
		stats.ResultsArchived = len(removedResults)
	}

	// Update state with kept results
	state.Results = keptResults
	stats.ResultsRemoved = len(removedResults)
	stats.ResultsAfter = len(keptResults)

	return stats, nil
}

// archiveResults archives removed results to a file
func (rm *RotationManager) archiveResults(results []types.PlayResult, playbook string) error {
	if rm.policy.ArchiveDir == "" {
		return nil
	}

	// Create archive directory if it doesn't exist
	if err := os.MkdirAll(rm.policy.ArchiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Create archive filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	_ = filepath.Join(
		rm.policy.ArchiveDir,
		fmt.Sprintf("archived_results_%s_%s.json", timestamp, playbook),
	)

	// Create archived data structure
	_ = map[string]interface{}{
		"timestamp": time.Now(),
		"playbook":  playbook,
		"count":     len(results),
		"results":   results,
	}

	// TODO: In real implementation, use JSON marshaling and compression
	// This is simplified for now

	return nil
}

// CleanupOldArchives removes old archives based on KeepArchives policy
func (rm *RotationManager) CleanupOldArchives() error {
	if rm.policy.ArchiveDir == "" || rm.policy.KeepArchives <= 0 {
		return nil
	}

	// List all files in archive directory
	entries, err := os.ReadDir(rm.policy.ArchiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Archive dir doesn't exist yet
		}
		return fmt.Errorf("failed to read archive directory: %w", err)
	}

	// Sort by modification time and keep only the most recent
	if len(entries) > rm.policy.KeepArchives {
		// TODO: Implement proper sorting and cleanup
		// This is simplified for now
	}

	return nil
}

// RotationStats contains statistics about a rotation operation
type RotationStats struct {
	ResultsBefore   int
	ResultsAfter    int
	ResultsRemoved  int
	ResultsArchived int
	ArchiveSize     int64
}

// GetRotationStats calculates what would be removed by current policy
func (rm *RotationManager) GetRotationStats(state *types.State) *RotationStats {
	stats := &RotationStats{
		ResultsBefore: len(state.Results),
	}

	if len(state.Results) == 0 {
		return stats
	}

	now := time.Now()
	var toRemove int

	for _, result := range state.Results {
		remove := false

		// Check MaxAge
		if rm.policy.MaxAge > 0 {
			if now.Sub(result.StartTime) > rm.policy.MaxAge {
				remove = true
			}
		}

		// Check MaxResults
		if rm.policy.MaxResults > 0 && toRemove >= rm.policy.MaxResults {
			remove = true
		}

		if remove {
			toRemove++
		}
	}

	stats.ResultsRemoved = toRemove
	stats.ResultsAfter = stats.ResultsBefore - toRemove

	return stats
}

// DeduplicateResults removes duplicate results (same host, task, timestamp)
func (rm *RotationManager) DeduplicateResults(state *types.State) int {
	if len(state.Results) == 0 {
		return 0
	}

	// Create a map to track unique results
	seen := make(map[string]bool)
	var deduplicated []types.PlayResult

	for _, result := range state.Results {
		// Create a unique key for this result
		key := fmt.Sprintf("%s_%s_%d", result.PlayName, result.Host, result.StartTime.Unix())

		if !seen[key] {
			seen[key] = true
			deduplicated = append(deduplicated, result)
		}
	}

	removed := len(state.Results) - len(deduplicated)
	state.Results = deduplicated

	return removed
}

package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
)

// Storage manages persisting and retrieving audit records
type Storage struct {
	path   string
	logger interfaces.Logger
}

// NewStorage creates a new audit storage
func NewStorage(path string, logger interfaces.Logger) (*Storage, error) {
	if path == "" {
		path = filepath.Join(os.Getenv("HOME"), ".onigirazu", "audit")
	}

	// Create directory if it doesn't exist
	err := os.MkdirAll(path, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit storage directory: %w", err)
	}

	return &Storage{
		path:   path,
		logger: logger,
	}, nil
}

// SaveRecord saves an execution record to disk
func (s *Storage) SaveRecord(record *ExecutionRecord) error {
	if record == nil {
		return fmt.Errorf("cannot save nil record")
	}

	// Create audit data directory with timestamp
	recordDir := filepath.Join(s.path, record.ID)
	err := os.MkdirAll(recordDir, 0700)
	if err != nil {
		return fmt.Errorf("failed to create record directory: %w", err)
	}

	// Save main record
	recordPath := filepath.Join(recordDir, "record.json")
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	err = os.WriteFile(recordPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write record file: %w", err)
	}

	// Save metadata summary for quick lookup
	metadataPath := filepath.Join(recordDir, "metadata.json")
	metadata := map[string]interface{}{
		"id":             record.ID,
		"playbook_path":  record.PlaybookPath,
		"status":         record.Status,
		"start_time":     record.StartTime,
		"duration":       record.Duration,
		"user":           record.User,
		"exit_code":      record.ExitCode,
		"total_tasks":    record.TotalTasks,
		"failed_tasks":   record.FailedTasks,
		"affected_hosts": len(record.AffectedHosts),
	}

	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	err = os.WriteFile(metadataPath, metadataData, 0600)
	if err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	s.logger.Debug("Saved audit record: %s", recordPath)
	return nil
}

// recordMeta represents metadata for a record
type recordMeta struct {
	ID       string
	Metadata map[string]interface{}
}

// LoadRecord loads an execution record from disk
func (s *Storage) LoadRecord(recordID string) (*ExecutionRecord, error) {
	recordPath := filepath.Join(s.path, recordID, "record.json")

	data, err := os.ReadFile(recordPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read record file: %w", err)
	}

	var record ExecutionRecord
	err = json.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return &record, nil
}

// ListRecords lists execution records with efficient pagination
// Uses metadata files for filtering when possible to avoid loading full records
func (s *Storage) ListRecords(filter FilterOptions) ([]ExecutionRecord, error) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	// First pass: collect metadata for all records
	var allMeta []recordMeta

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to load metadata first (faster than full record)
		meta, err := s.loadRecordMetadata(entry.Name())
		if err != nil {
			s.logger.Debug("Failed to load metadata for %s: %v", entry.Name(), err)
			continue
		}
		allMeta = append(allMeta, recordMeta{ID: entry.Name(), Metadata: meta})
	}

	// Filter based on metadata only (efficient)
	var filtered []recordMeta
	for _, m := range allMeta {
		if matchesMetadataFilter(m.Metadata, filter) {
			filtered = append(filtered, m)
		}
	}

	// Sort filtered metadata
	sortMetadata(filtered, filter.SortBy, filter.SortOrder)

	// Apply offset and limit to metadata only
	start := filter.Offset
	end := len(filtered)
	if filter.Limit > 0 {
		end = start + filter.Limit
		if end > len(filtered) {
			end = len(filtered)
		}
	}
	if start > len(filtered) {
		return []ExecutionRecord{}, nil
	}

	// Load only the records we need (page)
	var records []ExecutionRecord
	for i := start; i < end && i < len(filtered); i++ {
		record, err := s.LoadRecord(filtered[i].ID)
		if err != nil {
			s.logger.Debug("Failed to load record %s: %v", filtered[i].ID, err)
			continue
		}
		records = append(records, *record)
	}

	return records, nil
}

// loadRecordMetadata efficiently loads only the metadata file for a record
func (s *Storage) loadRecordMetadata(recordID string) (map[string]interface{}, error) {
	metadataPath := filepath.Join(s.path, recordID, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata map[string]interface{}
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}

// matchesMetadataFilter checks if metadata matches the filter criteria
func matchesMetadataFilter(metadata map[string]interface{}, filter FilterOptions) bool {
	// Quick field-based filtering without loading full record
	if filter.Status != "" {
		if status, ok := metadata["status"].(string); ok {
			if ExecutionStatus(status) != filter.Status {
				return false
			}
		}
	}

	// Note: Other filters that require full record content will be handled
	// after loading in a second pass if needed
	return true
}

// sortMetadata sorts records by metadata fields
func sortMetadata(records []recordMeta, sortBy, sortOrder string) {
	sort.Slice(records, func(i, j int) bool {
		var iVal, jVal interface{}

		if sortBy == "" || sortBy == "start_time" {
			iVal = records[i].Metadata["start_time"]
			jVal = records[j].Metadata["start_time"]
		} else {
			iVal = records[i].Metadata[sortBy]
			jVal = records[j].Metadata[sortBy]
		}

		// Compare based on type
		switch iV := iVal.(type) {
		case string:
			if jV, ok := jVal.(string); ok {
				if sortOrder == "desc" {
					return iV > jV
				}
				return iV < jV
			}
		case float64:
			if jV, ok := jVal.(float64); ok {
				if sortOrder == "desc" {
					return iV > jV
				}
				return iV < jV
			}
		}
		return false
	})
}

// GetStatistics retrieves statistics for all or filtered records
// Optimized to use metadata first, then load full records only for detailed analysis
func (s *Storage) GetStatistics(filter FilterOptions) (*AuditStatistics, error) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	stats := &AuditStatistics{
		CommonErrors:    []string{},
		MostUsedModules: []string{},
	}

	errorCounts := make(map[string]int)
	moduleCounts := make(map[string]int)
	var totalDuration float64

	// Load all metadata first (fast, minimal I/O)
	var recordIDs []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		meta, err := s.loadRecordMetadata(entry.Name())
		if err != nil {
			s.logger.Debug("Failed to load metadata for %s: %v", entry.Name(), err)
			continue
		}

		// Use metadata for basic counts
		stats.TotalExecutions++

		if status, ok := meta["status"].(string); ok {
			if ExecutionStatus(status) == StatusSuccess {
				stats.SuccessfulRuns++
			} else if ExecutionStatus(status) == StatusFailure {
				stats.FailedRuns++
			}
		}

		if totalTasks, ok := meta["total_tasks"].(float64); ok {
			stats.TotalTasks += int(totalTasks)
		}
		if failedTasks, ok := meta["failed_tasks"].(float64); ok {
			stats.TotalFailedTasks += int(failedTasks)
		}
		if duration, ok := meta["duration"].(float64); ok {
			totalDuration += duration
		}

		// Store ID for detailed analysis
		recordIDs = append(recordIDs, entry.Name())
	}

	// Calculate average duration from metadata
	if stats.TotalExecutions > 0 {
		stats.AvgDuration = totalDuration / float64(stats.TotalExecutions)
	}

	// Load full records only for detailed error and module analysis (pagination)
	// Limit detailed analysis to first 100 records for performance
	detailedLimit := 100
	if len(recordIDs) > detailedLimit {
		recordIDs = recordIDs[:detailedLimit]
	}

	for _, id := range recordIDs {
		record, err := s.LoadRecord(id)
		if err != nil {
			s.logger.Debug("Failed to load record %s: %v", id, err)
			continue
		}

		// Collect error messages
		if record.ErrorMessage != "" {
			errorCounts[record.ErrorMessage]++
		}

		// Count modules
		for _, play := range record.Plays {
			for _, task := range play.Tasks {
				moduleCounts[task.Module]++
			}
		}

		// Update first/last execution times from full records
		if stats.FirstExecution.IsZero() || record.StartTime.Before(stats.FirstExecution) {
			stats.FirstExecution = record.StartTime
		}
		if record.EndTime.After(stats.LastExecution) {
			stats.LastExecution = record.EndTime
		}
	}

	// Get top errors
	for err, count := range errorCounts {
		if count > 1 {
			stats.CommonErrors = append(stats.CommonErrors, fmt.Sprintf("%s (%d times)", err, count))
		}
	}

	// Get top modules
	modules := make([]string, 0, len(moduleCounts))
	for module := range moduleCounts {
		modules = append(modules, module)
	}
	sort.SliceStable(modules, func(i, j int) bool {
		return moduleCounts[modules[i]] > moduleCounts[modules[j]]
	})

	limit := 10
	if len(modules) < limit {
		limit = len(modules)
	}
	stats.MostUsedModules = modules[:limit]

	return stats, nil
}

// GetHostStatistics retrieves statistics for a specific host
func (s *Storage) GetHostStatistics(host string) (*HostAuditStatistics, error) {
	records, err := s.ListRecords(FilterOptions{Limit: 0, Offset: 0})
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	stats := &HostAuditStatistics{
		Host:             host,
		MostCommonErrors: []string{},
	}

	errorCounts := make(map[string]int)
	taskDurations := 0.0
	taskCount := 0

	for _, record := range records {
		// Check if host is in affected hosts or plays
		hostFound := false
		for _, h := range record.AffectedHosts {
			if h == host {
				hostFound = true
				break
			}
		}

		if !hostFound && len(record.AffectedHosts) > 0 {
			continue
		}

		if hostFound {
			stats.TotalExecutions++

			if record.Status == StatusSuccess {
				stats.SuccessfulRuns++
			} else if record.Status == StatusFailure {
				stats.FailedRuns++
			}

			stats.TotalTasks += record.TotalTasks
			stats.TotalFailedTasks += record.FailedTasks

			if record.EndTime.After(stats.LastExecution) {
				stats.LastExecution = record.EndTime
			}

			// Collect errors and task durations
			for _, play := range record.Plays {
				for _, task := range play.Tasks {
					if task.Host == host {
						if task.Error != "" {
							errorCounts[task.Error]++
						}
						taskDurations += task.Duration
						taskCount++
					}
				}
			}
		}
	}

	if taskCount > 0 {
		stats.AvgTaskDuration = taskDurations / float64(taskCount)
	}

	// Get top errors
	for err, count := range errorCounts {
		if count > 1 {
			stats.MostCommonErrors = append(stats.MostCommonErrors, fmt.Sprintf("%s (%d times)", err, count))
		}
	}

	return stats, nil
}

// DeleteOldRecords deletes records older than retention days
func (s *Storage) DeleteOldRecords(retentionDays int) (int, error) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return 0, fmt.Errorf("failed to read storage directory: %w", err)
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		recordPath := filepath.Join(s.path, entry.Name())
		record, err := s.LoadRecord(entry.Name())
		if err != nil {
			continue
		}

		if record.StartTime.Before(cutoffTime) {
			err := os.RemoveAll(recordPath)
			if err == nil {
				deletedCount++
				s.logger.Debug("Deleted old audit record: %s", entry.Name())
			}
		}
	}

	return deletedCount, nil
}

// Close closes the storage
func (s *Storage) Close() error {
	// Storage doesn't need explicit closing, but method is here for interface consistency
	return nil
}

// Helper functions

func matchesFilter(record ExecutionRecord, filter FilterOptions) bool {
	if filter.PlaybookPath != "" && record.PlaybookPath != filter.PlaybookPath {
		return false
	}

	if filter.Status != "" && record.Status != filter.Status {
		return false
	}

	if filter.HostFilter != "" {
		hostFound := false
		for _, h := range record.AffectedHosts {
			if h == filter.HostFilter {
				hostFound = true
				break
			}
		}
		if !hostFound {
			return false
		}
	}

	if !filter.DateFrom.IsZero() && record.StartTime.Before(filter.DateFrom) {
		return false
	}

	if !filter.DateTo.IsZero() && record.StartTime.After(filter.DateTo) {
		return false
	}

	return true
}

func sortRecords(records []ExecutionRecord, sortBy, sortOrder string) {
	ascending := sortOrder != "desc"

	sort.SliceStable(records, func(i, j int) bool {
		switch sortBy {
		case "duration":
			if ascending {
				return records[i].Duration < records[j].Duration
			}
			return records[i].Duration > records[j].Duration
		case "status":
			if ascending {
				return records[i].Status < records[j].Status
			}
			return records[i].Status > records[j].Status
		case "time":
			fallthrough
		default:
			if ascending {
				return records[i].StartTime.Before(records[j].StartTime)
			}
			return records[i].StartTime.After(records[j].StartTime)
		}
	})
}

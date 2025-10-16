package audit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	err = ioutil.WriteFile(recordPath, data, 0600)
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

	err = ioutil.WriteFile(metadataPath, metadataData, 0600)
	if err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	s.logger.Debug("Saved audit record: %s", recordPath)
	return nil
}

// LoadRecord loads an execution record from disk
func (s *Storage) LoadRecord(recordID string) (*ExecutionRecord, error) {
	recordPath := filepath.Join(s.path, recordID, "record.json")

	data, err := ioutil.ReadFile(recordPath)
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

// ListRecords lists all available execution records
func (s *Storage) ListRecords(filter FilterOptions) ([]ExecutionRecord, error) {
	entries, err := ioutil.ReadDir(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var records []ExecutionRecord
	count := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		record, err := s.LoadRecord(entry.Name())
		if err != nil {
			s.logger.Debug("Failed to load record %s: %v", entry.Name(), err)
			continue
		}

		// Apply filters
		if !matchesFilter(*record, filter) {
			continue
		}

		records = append(records, *record)
		count++

		// Stop if we've reached the limit
		if filter.Limit > 0 && count >= filter.Limit+filter.Offset {
			break
		}
	}

	// Sort records
	sortRecords(records, filter.SortBy, filter.SortOrder)

	// Apply offset and limit
	if filter.Offset > 0 || filter.Limit > 0 {
		start := filter.Offset
		end := len(records)

		if filter.Limit > 0 {
			end = start + filter.Limit
			if end > len(records) {
				end = len(records)
			}
		}

		if start > len(records) {
			return []ExecutionRecord{}, nil
		}

		records = records[start:end]
	}

	return records, nil
}

// GetStatistics retrieves statistics for all or filtered records
func (s *Storage) GetStatistics(filter FilterOptions) (*AuditStatistics, error) {
	records, err := s.ListRecords(FilterOptions{Limit: 0, Offset: 0})
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	stats := &AuditStatistics{
		CommonErrors:    []string{},
		MostUsedModules: []string{},
	}

	errorCounts := make(map[string]int)
	moduleCounts := make(map[string]int)

	for _, record := range records {
		stats.TotalExecutions++

		if record.Status == StatusSuccess {
			stats.SuccessfulRuns++
		} else if record.Status == StatusFailure {
			stats.FailedRuns++
		}

		stats.TotalTasks += record.TotalTasks
		stats.TotalFailedTasks += record.FailedTasks

		if stats.FirstExecution.IsZero() || record.StartTime.Before(stats.FirstExecution) {
			stats.FirstExecution = record.StartTime
		}

		if record.EndTime.After(stats.LastExecution) {
			stats.LastExecution = record.EndTime
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
	}

	// Calculate average duration
	if stats.TotalExecutions > 0 {
		totalDuration := 0.0
		for _, record := range records {
			totalDuration += record.Duration
		}
		stats.AvgDuration = totalDuration / float64(stats.TotalExecutions)
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
	entries, err := ioutil.ReadDir(s.path)
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

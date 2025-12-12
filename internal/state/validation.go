package state

import (
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// ValidationResult holds validation results
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// Validator validates state file structure and content
type Validator struct {
	strictMode bool // If true, enforce stricter validation rules
}

// NewValidator creates a new state validator
func NewValidator(strictMode bool) *Validator {
	return &Validator{
		strictMode: strictMode,
	}
}

// Validate performs comprehensive validation on a state
func (v *Validator) Validate(state *types.State) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Validate version
	if state.Version < 0 || state.Version > CurrentVersion {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "Version",
			Message: fmt.Sprintf("invalid version %d, expected 0-%d", state.Version, CurrentVersion),
			Value:   state.Version,
		})
	}

	// Validate LastRun (should not be in future)
	if state.LastRun.After(time.Now().Add(1 * time.Hour)) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "LastRun",
			Message: "LastRun timestamp is in the future",
			Value:   state.LastRun,
		})
	}

	// Validate Playbook field
	if state.Playbook == "" && len(state.Results) > 0 {
		if v.strictMode {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "Playbook",
				Message: "Playbook name is empty but results exist",
				Value:   state.Playbook,
			})
		}
	}

	// Validate Results
	for i, playResult := range state.Results {
		if valErr := v.validatePlayResult(&playResult); !valErr.Valid {
			result.Valid = false
			for _, e := range valErr.Errors {
				e.Field = fmt.Sprintf("Results[%d].%s", i, e.Field)
				result.Errors = append(result.Errors, e)
			}
		}
	}

	// Validate Metadata
	if state.Metadata != nil {
		if err := v.validateMetadata(state.Metadata); err != nil {
			result.Valid = false
			for _, e := range err.Errors {
				e.Field = "Metadata." + e.Field
				result.Errors = append(result.Errors, e)
			}
		}
	}

	// Validate Checksums
	if state.Checksums != nil {
		for key, checksum := range state.Checksums {
			if checksum == "" {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("Checksums[%s]", key),
					Message: "checksum value is empty",
					Value:   key,
				})
			}
			// SHA256 checksums should be 64 hex characters
			if len(checksum) != 64 {
				if v.strictMode {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   fmt.Sprintf("Checksums[%s]", key),
						Message: fmt.Sprintf("invalid checksum format: expected 64 hex chars, got %d", len(checksum)),
						Value:   checksum,
					})
				}
			}
		}
	}

	return result
}

// validatePlayResult validates a single PlayResult
func (v *Validator) validatePlayResult(result *types.PlayResult) *ValidationResult {
	vr := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// PlayName should not be empty
	if result.PlayName == "" && result.Name == "" {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "PlayName",
			Message: "PlayName is empty",
			Value:   result.PlayName,
		})
	}

	// Duration should be positive
	if result.Duration < 0 {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "Duration",
			Message: "Duration is negative",
			Value:   result.Duration,
		})
	}

	// EndTime should not be before StartTime
	if !result.EndTime.IsZero() && !result.StartTime.IsZero() && result.EndTime.Before(result.StartTime) {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "EndTime",
			Message: "EndTime is before StartTime",
			Value:   result.EndTime,
		})
	}

	// Validate tasks
	for i, task := range result.Tasks {
		if err := v.validateTaskResult(&task); err != nil {
			vr.Valid = false
			for _, e := range err.Errors {
				e.Field = fmt.Sprintf("Tasks[%d].%s", i, e.Field)
				vr.Errors = append(vr.Errors, e)
			}
		}
	}

	return vr
}

// validateTaskResult validates a single TaskResult
func (v *Validator) validateTaskResult(task *types.TaskResult) *ValidationResult {
	vr := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// TaskName should not be empty
	if task.TaskName == "" {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "TaskName",
			Message: "TaskName is empty",
			Value:   task.TaskName,
		})
	}

	// Module should not be empty
	if task.Module == "" {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "Module",
			Message: "Module is empty",
			Value:   task.Module,
		})
	}

	// Duration should be non-negative
	if task.Duration < 0 {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "Duration",
			Message: "Duration is negative",
			Value:   task.Duration,
		})
	}

	// Timestamp should not be in future
	if task.Timestamp.After(time.Now().Add(1 * time.Hour)) {
		vr.Valid = false
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   "Timestamp",
			Message: "Timestamp is in the future",
			Value:   task.Timestamp,
		})
	}

	return vr
}

// validateMetadata validates ExecutionMetadata
func (v *Validator) validateMetadata(metadata *types.ExecutionMetadata) *ValidationResult {
	vr := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	return vr
}

// ValidateAndRepair attempts to validate and repair common issues
func (v *Validator) ValidateAndRepair(state *types.State) (*ValidationResult, bool) {
	result := v.Validate(state)

	repaired := false

	// Attempt repairs
	if state.LastRun.IsZero() && len(state.Results) > 0 {
		// Set LastRun to the last result's end time
		state.LastRun = state.Results[len(state.Results)-1].EndTime
		repaired = true
	}

	// Initialize empty maps
	if state.Variables == nil {
		state.Variables = make(map[string]interface{})
		repaired = true
	}

	if state.Checksums == nil {
		state.Checksums = make(map[string]string)
		repaired = true
	}

	if state.Metadata == nil {
		state.Metadata = &types.ExecutionMetadata{
			User:        "unknown",
			Hostname:    "unknown",
			Environment: make(map[string]string),
			Tags:        []string{},
			ExtraVars:   make(map[string]interface{}),
		}
		repaired = true
	}

	return result, repaired
}

// String returns error details as a formatted string
func (vr *ValidationResult) String() string {
	if vr.Valid {
		return "Validation successful"
	}

	msg := fmt.Sprintf("Validation failed with %d errors:\n", len(vr.Errors))
	for i, err := range vr.Errors {
		msg += fmt.Sprintf("  %d. [%s] %s (value: %v)\n", i+1, err.Field, err.Message, err.Value)
	}
	return msg
}

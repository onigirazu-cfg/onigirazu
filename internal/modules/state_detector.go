package modules

import (
	"context"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// StateDetector defines the interface for modules to implement idempotent state checking
// This standardizes how modules detect current state and compare with desired state
type StateDetector interface {
	// GetCurrentState returns the current system state for this module's scope
	// Returns a map representing the current state or an error if state cannot be determined
	GetCurrentState(ctx context.Context, host types.Host, args map[string]interface{}) (map[string]interface{}, error)

	// CompareStates compares desired state (from args) with current state
	// Returns: (isEqual bool, differences map, error)
	// differences contains keys where current != desired for diagnostics
	CompareStates(desiredState map[string]interface{}, currentState map[string]interface{}) (bool, map[string]string, error)

	// IsSafeToSkip returns true if the module can safely skip execution when state is already correct
	// Some modules (like command, shell) should never skip even if state matches
	IsSafeToSkip(ctx context.Context, host types.Host, args map[string]interface{}) bool
}

// IdempotencyConfig defines idempotency behavior for a module
type IdempotencyConfig struct {
	// IsIdempotent indicates if the module is idempotent by nature
	IsIdempotent bool

	// StateKeyFields defines which fields are used to determine state equivalence
	// e.g., for a file module: ["path", "content", "mode", "owner"]
	// e.g., for apt module: ["name", "state"]
	StateKeyFields []string

	// CriticalFieldsForComparison are fields that must match for state to be considered equal
	// If any critical field differs, module must execute
	CriticalFieldsForComparison []string

	// PreCheckDuration is the expected time for pre-check (should be much faster than full execution)
	// Used for monitoring idempotency performance
	PreCheckDurationMs int

	// FullExecutionDurationMs is the typical duration for full execution
	// If precheck + execution takes too long, indicates inefficiency
	FullExecutionDurationMs int

	// AllowedChangedFlagStates indicates which states (true/false) are valid for this module
	// Most modules should only return true when actual changes were made
	AllowedChangedFlagStates []bool
}

// IdempotencyHelper provides utility methods for idempotent state checking
type IdempotencyHelper struct {
	config IdempotencyConfig
	// canCacheState indicates if module state can be cached during a task run
	canCacheState bool
}

// NewIdempotencyHelper creates a new idempotency helper with the given config
func NewIdempotencyHelper(config IdempotencyConfig) *IdempotencyHelper {
	return &IdempotencyHelper{
		config:        config,
		canCacheState: true, // Most modules can cache during single run
	}
}

// DefaultIdempotentConfig returns a configuration for typically idempotent modules
func DefaultIdempotentConfig(stateFields []string) IdempotencyConfig {
	return IdempotencyConfig{
		IsIdempotent:                true,
		StateKeyFields:              stateFields,
		CriticalFieldsForComparison: stateFields,
		PreCheckDurationMs:          100,                 // typical pre-check should be fast
		FullExecutionDurationMs:     1000,                // typical execution might be longer
		AllowedChangedFlagStates:    []bool{true, false}, // both are valid
	}
}

// NonIdempotentConfig returns a configuration for modules that are non-idempotent by design
// (e.g., command, shell, script, debug)
func NonIdempotentConfig() IdempotencyConfig {
	return IdempotencyConfig{
		IsIdempotent:                false,
		StateKeyFields:              []string{}, // No meaningful state to check
		CriticalFieldsForComparison: []string{},
		PreCheckDurationMs:          0,
		FullExecutionDurationMs:     500,          // typically fast but non-idempotent
		AllowedChangedFlagStates:    []bool{true}, // Always reports changed
	}
}

// ComparisonResult holds the result of state comparison
type ComparisonResult struct {
	IsEqual       bool              // true if states are equal
	Differences   map[string]string // key -> "current vs desired"
	ModuleShould  string            // "skip" or "execute"
	SkipReason    string            // why the module should skip (if applicable)
	ExecuteReason string            // why the module must execute (if applicable)
}

// StateChangeDescriptor describes what changed during module execution
type StateChangeDescriptor struct {
	FieldName     string      // which field changed
	OldValue      interface{} // previous value
	NewValue      interface{} // current value
	ChangeType    string      // "created", "modified", "deleted"
	IsSignificant bool        // true if this change is meaningful to report
}

package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SchemaMigrator handles schema versioning and migrations
type SchemaMigrator struct {
	migrations        map[int]map[int]*types.SchemaMigration // migrations[from][to] = *SchemaMigration
	supportedVersions []int                                  // List of supported schema versions
}

// NewSchemaMigrator creates a new schema migrator
func NewSchemaMigrator() *SchemaMigrator {
	return &SchemaMigrator{
		migrations:        make(map[int]map[int]*types.SchemaMigration),
		supportedVersions: []int{1, 2}, // Default supported versions
	}
}

// RegisterMigration registers a migration rule
func (sm *SchemaMigrator) RegisterMigration(migration *types.SchemaMigration) error {
	if migration == nil {
		return fmt.Errorf("migration cannot be nil")
	}

	if migration.From >= migration.To {
		return fmt.Errorf("invalid migration: from version (%d) must be less than to version (%d)", migration.From, migration.To)
	}

	if migration.From < 1 || migration.To < 1 {
		return fmt.Errorf("invalid migration: versions must be >= 1, got from=%d, to=%d", migration.From, migration.To)
	}

	if len(migration.Rules) == 0 {
		return fmt.Errorf("migration must have at least one rule")
	}

	// Initialize nested map if needed
	if _, ok := sm.migrations[migration.From]; !ok {
		sm.migrations[migration.From] = make(map[int]*types.SchemaMigration)
	}

	sm.migrations[migration.From][migration.To] = migration

	// Track supported versions
	if !containsInt(sm.supportedVersions, migration.From) {
		sm.supportedVersions = append(sm.supportedVersions, migration.From)
	}
	if !containsInt(sm.supportedVersions, migration.To) {
		sm.supportedVersions = append(sm.supportedVersions, migration.To)
	}

	return nil
}

// SetSupportedVersions sets the list of supported schema versions
func (sm *SchemaMigrator) SetSupportedVersions(versions []int) error {
	if len(versions) == 0 {
		return fmt.Errorf("must have at least one supported version")
	}

	for _, v := range versions {
		if v < 1 {
			return fmt.Errorf("invalid version: %d (must be >= 1)", v)
		}
	}

	sm.supportedVersions = versions
	return nil
}

// CheckMigrationNeeded checks if migration is needed between versions
func (sm *SchemaMigrator) CheckMigrationNeeded(fromVersion, toVersion int) bool {
	if fromVersion == toVersion {
		return false
	}

	if fromVersion > toVersion {
		return false // Don't support downgrade migrations
	}

	path, _ := sm.GetMigrationPath(fromVersion, toVersion)
	return len(path) > 0
}

// GetMigrationPath finds the migration path from one version to another
// Returns the list of migrations needed to go from fromVersion to toVersion
func (sm *SchemaMigrator) GetMigrationPath(fromVersion, toVersion int) ([]types.SchemaMigration, error) {
	if fromVersion == toVersion {
		return []types.SchemaMigration{}, nil
	}

	if fromVersion > toVersion {
		return nil, fmt.Errorf("downgrades not supported: from %d to %d", fromVersion, toVersion)
	}

	// BFS to find shortest migration path
	queue := []int{fromVersion}
	visited := make(map[int]bool)
	parent := make(map[int]int)
	parent[fromVersion] = -1
	visited[fromVersion] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == toVersion {
			// Reconstruct path
			var path []int
			node := toVersion
			for node != -1 {
				path = append([]int{node}, path...)
				node = parent[node]
			}

			// Convert path to migrations
			var migrations []types.SchemaMigration
			for i := 0; i < len(path)-1; i++ {
				from := path[i]
				to := path[i+1]
				if migration, ok := sm.migrations[from][to]; ok {
					migrations = append(migrations, *migration)
				}
			}
			return migrations, nil
		}

		// Check for direct migrations from current version
		if toMigrations, ok := sm.migrations[current]; ok {
			for nextVersion := range toMigrations {
				if !visited[nextVersion] {
					visited[nextVersion] = true
					parent[nextVersion] = current
					queue = append(queue, nextVersion)
				}
			}
		}
	}

	return nil, fmt.Errorf("no migration path found from version %d to %d", fromVersion, toVersion)
}

// ApplyMigration applies a single migration to parameters
func (sm *SchemaMigrator) ApplyMigration(params map[string]interface{}, migration types.SchemaMigration) (map[string]interface{}, error) {
	if params == nil {
		params = make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// Copy original parameters
	for k, v := range params {
		result[k] = v
	}

	// Apply each rule
	for _, rule := range migration.Rules {
		var err error
		switch rule.Type {
		case types.MigrationRuleTypeRename:
			result, err = sm.applyRenameRule(result, rule)
		case types.MigrationRuleTypeTransform:
			result, err = sm.applyTransformRule(result, rule)
		case types.MigrationRuleTypeDeprecate:
			result, err = sm.applyDeprecateRule(result, rule)
		case types.MigrationRuleTypeRemove:
			result, err = sm.applyRemoveRule(result, rule)
		case types.MigrationRuleTypeAddDefault:
			result, err = sm.applyAddDefaultRule(result, rule)
		default:
			err = fmt.Errorf("unknown migration rule type: %s", rule.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to apply migration rule: %w", err)
		}
	}

	return result, nil
}

// applyRenameRule applies a rename migration rule
func (sm *SchemaMigrator) applyRenameRule(params map[string]interface{}, rule types.MigrationRule) (map[string]interface{}, error) {
	if rule.OldParam == "" || rule.NewParam == "" {
		return nil, fmt.Errorf("rename rule requires both old_param and new_param")
	}

	if value, ok := params[rule.OldParam]; ok {
		params[rule.NewParam] = value
		delete(params, rule.OldParam)
	}

	return params, nil
}

// applyTransformRule applies a type transformation migration rule
func (sm *SchemaMigrator) applyTransformRule(params map[string]interface{}, rule types.MigrationRule) (map[string]interface{}, error) {
	param := rule.OldParam
	if param == "" {
		param = rule.NewParam
	}

	if _, ok := params[param]; !ok {
		// Parameter not provided, skip transformation
		return params, nil
	}

	value := params[param]
	var transformed interface{}
	var err error

	switch rule.ToType {
	case "string":
		transformed, err = transformToString(value, rule.FromType)
	case "integer":
		transformed, err = transformToInteger(value, rule.FromType)
	case "boolean":
		transformed, err = transformToBoolean(value, rule.FromType)
	case "array":
		transformed, err = transformToArray(value, rule.FromType)
	case "object":
		transformed, err = transformToObject(value, rule.FromType)
	default:
		err = fmt.Errorf("unsupported target type: %s", rule.ToType)
	}

	if err != nil {
		return nil, fmt.Errorf("transformation from %s to %s failed: %w", rule.FromType, rule.ToType, err)
	}

	if rule.NewParam != "" && rule.NewParam != param {
		delete(params, param)
		params[rule.NewParam] = transformed
	} else {
		params[param] = transformed
	}

	return params, nil
}

// applyDeprecateRule applies a deprecate migration rule (just logs warning)
func (sm *SchemaMigrator) applyDeprecateRule(params map[string]interface{}, rule types.MigrationRule) (map[string]interface{}, error) {
	param := rule.OldParam
	if param == "" {
		return params, nil
	}

	if _, ok := params[param]; ok {
		msg := fmt.Sprintf("Parameter '%s' is deprecated", param)
		if rule.Description != "" {
			msg += fmt.Sprintf(": %s", rule.Description)
		}
		if rule.Reason != "" {
			msg += fmt.Sprintf(" (reason: %s)", rule.Reason)
		}
		log.Printf("WARNING: %s", msg)
	}

	return params, nil
}

// applyRemoveRule applies a remove migration rule
func (sm *SchemaMigrator) applyRemoveRule(params map[string]interface{}, rule types.MigrationRule) (map[string]interface{}, error) {
	param := rule.OldParam
	if param == "" {
		return params, nil
	}

	delete(params, param)
	return params, nil
}

// applyAddDefaultRule applies an add_default migration rule
func (sm *SchemaMigrator) applyAddDefaultRule(params map[string]interface{}, rule types.MigrationRule) (map[string]interface{}, error) {
	param := rule.NewParam
	if param == "" {
		param = rule.OldParam
	}

	if param == "" {
		return nil, fmt.Errorf("add_default rule requires either new_param or old_param")
	}

	// Only add if parameter doesn't exist
	if _, ok := params[param]; !ok {
		if rule.Default != nil {
			params[param] = rule.Default
		}
	}

	return params, nil
}

// ApplyMigrations applies a sequence of migrations
func (sm *SchemaMigrator) ApplyMigrations(params map[string]interface{}, migrations []types.SchemaMigration) (map[string]interface{}, error) {
	result := params
	var err error

	for _, migration := range migrations {
		result, err = sm.ApplyMigration(result, migration)
		if err != nil {
			return nil, fmt.Errorf("failed to apply migration from v%d to v%d: %w", migration.From, migration.To, err)
		}
	}

	return result, nil
}

// ValidateVersion checks if a version is valid and supported
func (sm *SchemaMigrator) ValidateVersion(version int) error {
	if version < 1 {
		return fmt.Errorf("invalid version: %d (must be >= 1)", version)
	}

	if !containsInt(sm.supportedVersions, version) {
		return fmt.Errorf("unsupported version: %d (supported: %v)", version, sm.supportedVersions)
	}

	return nil
}

// GetLatestVersion returns the latest supported schema version
func (sm *SchemaMigrator) GetLatestVersion() int {
	latest := 0
	for _, v := range sm.supportedVersions {
		if v > latest {
			latest = v
		}
	}
	return latest
}

// GetSupportedVersions returns the list of supported versions
func (sm *SchemaMigrator) GetSupportedVersions() []int {
	return sm.supportedVersions
}

// CanMigrateTo checks if migration is possible to target version
func (sm *SchemaMigrator) CanMigrateTo(fromVersion, toVersion int) bool {
	if fromVersion == toVersion {
		return true
	}
	if fromVersion > toVersion {
		return false
	}
	_, err := sm.GetMigrationPath(fromVersion, toVersion)
	return err == nil
}

// GetMigrationInfo returns information about a migration
func (sm *SchemaMigrator) GetMigrationInfo(from, to int) *types.SchemaMigration {
	if migrations, ok := sm.migrations[from]; ok {
		if migration, ok := migrations[to]; ok {
			return migration
		}
	}
	return nil
}

// Type conversion helper functions

func transformToString(value interface{}, fromType string) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int32, int64:
		return fmt.Sprintf("%v", v), nil
	case float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	case []interface{}:
		// Join array elements
		var strs []string
		for _, item := range v {
			strs = append(strs, fmt.Sprintf("%v", item))
		}
		return strings.Join(strs, ","), nil
	case map[string]interface{}:
		// Convert to JSON string
		data, _ := json.Marshal(v)
		return string(data), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func transformToInteger(value interface{}, fromType string) (interface{}, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert string %q to integer", v)
		}
		return int(i), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to integer", v)
	}
}

func transformToBoolean(value interface{}, fromType string) (interface{}, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(v) {
		case "true", "yes", "on", "1":
			return true, nil
		case "false", "no", "off", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("cannot convert string %q to boolean", v)
		}
	case int:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to boolean", v)
	}
}

func transformToArray(value interface{}, fromType string) (interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		return v, nil
	case string:
		// Split string by comma
		parts := strings.Split(v, ",")
		result := make([]interface{}, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil
	default:
		// Wrap single value in array
		return []interface{}{v}, nil
	}
}

func transformToObject(value interface{}, fromType string) (interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		return v, nil
	case string:
		// Try to parse as JSON
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(v), &obj); err != nil {
			return nil, fmt.Errorf("cannot convert string to object: %w", err)
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to object", v)
	}
}

// containsInt checks if an integer slice contains a value
func containsInt(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// DefaultSchemaMigrator creates a migrator with built-in common migrations
func DefaultSchemaMigrator() *SchemaMigrator {
	sm := NewSchemaMigrator()

	// Built-in migration: v1 to v2
	// Example: Rename port (string) to port (integer)
	migration1To2 := &types.SchemaMigration{
		From: 1,
		To:   2,
		Rules: []types.MigrationRule{
			{
				Type:        types.MigrationRuleTypeTransform,
				OldParam:    "port",
				NewParam:    "port",
				FromType:    "string",
				ToType:      "integer",
				Description: "Convert port from string to integer",
				Reason:      "Standardize port parameter type",
			},
		},
		Notes: "Initial schema evolution: port parameter type conversion",
		Date:  "2025-01-30",
	}

	_ = sm.RegisterMigration(migration1To2)
	return sm
}

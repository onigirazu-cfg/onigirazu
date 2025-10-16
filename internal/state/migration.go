package state

import (
	"encoding/json"
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CurrentVersion is the current state file schema version
const CurrentVersion = 1

// Migrator handles state file version migrations
type Migrator struct {
	handlers map[int]MigrationHandler
}

// MigrationHandler defines a function that migrates from one version to the next
type MigrationHandler func(*types.State) error

// NewMigrator creates a new migrator with all migration handlers
func NewMigrator() *Migrator {
	m := &Migrator{
		handlers: make(map[int]MigrationHandler),
	}

	// Register migration handlers
	// Version 0 -> 1: Add Version, Metadata, Compressed fields
	m.handlers[0] = migrateFromV0ToV1

	return m
}

// Migrate migrates a state from its current version to the latest version
func (m *Migrator) Migrate(state *types.State) error {
	if state.Version == CurrentVersion {
		return nil // Already at latest version
	}

	if state.Version > CurrentVersion {
		return fmt.Errorf("state file version %d is newer than supported version %d", state.Version, CurrentVersion)
	}

	// Apply migrations sequentially
	for version := state.Version; version < CurrentVersion; version++ {
		handler, exists := m.handlers[version]
		if !exists {
			return fmt.Errorf("no migration handler for version %d->%d", version, version+1)
		}

		if err := handler(state); err != nil {
			return fmt.Errorf("migration from v%d to v%d failed: %w", version, version+1, err)
		}

		state.Version = version + 1
	}

	return nil
}

// migrateFromV0ToV1 migrates from version 0 to version 1
// This is for old state files that don't have Version, Metadata, or Compressed fields
func migrateFromV0ToV1(state *types.State) error {
	// Set default metadata if not present
	if state.Metadata == nil {
		state.Metadata = &types.ExecutionMetadata{
			User:        "unknown",
			Hostname:    "unknown",
			WorkingDir:  "",
			Environment: make(map[string]string),
			Tags:        []string{},
			ExtraVars:   make(map[string]interface{}),
		}
	}

	// Mark as not compressed since old files weren't compressed
	state.Compressed = false

	return nil
}

// MigrateJSON migrates raw JSON data to current version
// This is useful for loading state from various sources
func (m *Migrator) MigrateJSON(data []byte) (*types.State, error) {
	// First, try to unmarshal as generic map to detect version
	var rawState map[string]interface{}
	if err := json.Unmarshal(data, &rawState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state JSON: %w", err)
	}

	// Get version, default to 0 for old files
	version := 0
	if v, exists := rawState["version"]; exists {
		if vNum, ok := v.(float64); ok {
			version = int(vNum)
		}
	}

	// Unmarshal into State struct
	state := &types.State{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state data: %w", err)
	}

	// Set version for migration
	state.Version = version

	// Migrate to current version
	if err := m.Migrate(state); err != nil {
		return nil, err
	}

	return state, nil
}

// GetMigrationInfo returns information about migrations
type MigrationInfo struct {
	FromVersion int
	ToVersion   int
	Description string
}

// GetMigrationHistory returns the history of migrations
func (m *Migrator) GetMigrationHistory() []MigrationInfo {
	return []MigrationInfo{
		{
			FromVersion: 0,
			ToVersion:   1,
			Description: "Add Version, Metadata (user/hostname/environment), and Compressed fields",
		},
	}
}

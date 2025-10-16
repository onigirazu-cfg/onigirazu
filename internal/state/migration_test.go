package state

import (
	"encoding/json"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestMigratorV0ToV1(t *testing.T) {
	tests := []struct {
		name      string
		inputJSON string
		expectErr bool
		validate  func(*types.State) bool
	}{
		{
			name: "Old state file without version",
			inputJSON: `{
				"last_run": "2025-01-01T00:00:00Z",
				"playbook": "test.yml",
				"results": [],
				"variables": {},
				"checksums": {}
			}`,
			expectErr: false,
			validate: func(s *types.State) bool {
				return s.Version == 1 &&
					s.Metadata != nil &&
					!s.Compressed
			},
		},
		{
			name:      "Empty state file",
			inputJSON: `{}`,
			expectErr: false,
			validate: func(s *types.State) bool {
				return s.Version == 1 &&
					s.Metadata != nil
			},
		},
	}

	migrator := NewMigrator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := migrator.MigrateJSON([]byte(tt.inputJSON))

			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if err != nil {
				return
			}

			if !tt.validate(state) {
				t.Fatalf("Validation failed for state: %+v", state)
			}
		})
	}
}

func TestMigrationHandlers(t *testing.T) {
	migrator := NewMigrator()

	state := &types.State{
		Version: 0,
	}

	err := migrator.Migrate(state)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if state.Version != CurrentVersion {
		t.Fatalf("Expected version %d, got %d", CurrentVersion, state.Version)
	}

	if state.Metadata == nil {
		t.Fatalf("Metadata should not be nil after migration")
	}
}

func TestMigrationJSON(t *testing.T) {
	migrator := NewMigrator()

	oldState := map[string]interface{}{
		"last_run":  "2025-01-01T00:00:00Z",
		"playbook":  "test.yml",
		"results":   []interface{}{},
		"variables": map[string]interface{}{},
		"checksums": map[string]interface{}{},
	}

	jsonData, err := json.Marshal(oldState)
	if err != nil {
		t.Fatalf("Failed to marshal old state: %v", err)
	}

	state, err := migrator.MigrateJSON(jsonData)
	if err != nil {
		t.Fatalf("MigrateJSON failed: %v", err)
	}

	if state.Version != 1 {
		t.Fatalf("Expected version 1, got %d", state.Version)
	}
}

func TestMigrationHistory(t *testing.T) {
	migrator := NewMigrator()
	history := migrator.GetMigrationHistory()

	if len(history) == 0 {
		t.Fatalf("Expected migration history, got empty")
	}

	if history[0].FromVersion != 0 || history[0].ToVersion != 1 {
		t.Fatalf("Unexpected migration history: %+v", history[0])
	}
}

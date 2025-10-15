package rollback

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Snapshot represents a system state snapshot before changes
type Snapshot struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	PlaybookID  string                 `json:"playbook_id"`
	Description string                 `json:"description"`
	Resources   []ResourceSnapshot     `json:"resources"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ResourceSnapshot represents a snapshot of a single resource
type ResourceSnapshot struct {
	Type       string                 `json:"type"`        // file, package, service, user, etc.
	Identifier string                 `json:"identifier"`  // path, package name, service name, etc.
	Host       string                 `json:"host"`        // target host
	State      map[string]interface{} `json:"state"`       // current state before change
	Action     string                 `json:"action"`      // action that was performed
	Module     string                 `json:"module"`      // module that performed the action
	TaskName   string                 `json:"task_name"`   // name of the task
	Reversible bool                   `json:"reversible"`  // whether this change can be reversed
	RollbackOp *RollbackOperation     `json:"rollback_op"` // operation to reverse the change
}

// RollbackOperation defines how to reverse a change
type RollbackOperation struct {
	Module string                 `json:"module"`
	Args   map[string]interface{} `json:"args"`
	Order  int                    `json:"order"` // execution order (higher = execute first during rollback)
}

// SnapshotManager manages snapshots
type SnapshotManager struct {
	snapshotDir string
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(snapshotDir string) *SnapshotManager {
	return &SnapshotManager{
		snapshotDir: snapshotDir,
	}
}

// CreateSnapshot creates a new snapshot
func (sm *SnapshotManager) CreateSnapshot(playbookID, description string) (*Snapshot, error) {
	snapshot := &Snapshot{
		ID:          generateSnapshotID(),
		Timestamp:   time.Now(),
		PlaybookID:  playbookID,
		Description: description,
		Resources:   []ResourceSnapshot{},
		Metadata:    make(map[string]interface{}),
	}

	return snapshot, nil
}

// AddResourceSnapshot adds a resource snapshot to the snapshot
func (sm *SnapshotManager) AddResourceSnapshot(snapshot *Snapshot, resource ResourceSnapshot) {
	snapshot.Resources = append(snapshot.Resources, resource)
}

// SaveSnapshot saves a snapshot to disk
func (sm *SnapshotManager) SaveSnapshot(snapshot *Snapshot) error {
	// Create snapshot directory if it doesn't exist
	if err := os.MkdirAll(sm.snapshotDir, 0750); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Serialize snapshot
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	// Write to file
	filename := filepath.Join(sm.snapshotDir, fmt.Sprintf("snapshot_%s.json", snapshot.ID))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}

// LoadSnapshot loads a snapshot from disk
func (sm *SnapshotManager) LoadSnapshot(snapshotID string) (*Snapshot, error) {
	filename := filepath.Join(sm.snapshotDir, fmt.Sprintf("snapshot_%s.json", snapshotID))

	data, err := os.ReadFile(filename) // #nosec G304 -- snapshotID is validated
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot: %w", err)
	}

	return &snapshot, nil
}

// ListSnapshots lists all available snapshots
func (sm *SnapshotManager) ListSnapshots() ([]Snapshot, error) {
	files, err := os.ReadDir(sm.snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snapshot{}, nil
		}
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	snapshots := []Snapshot{}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sm.snapshotDir, file.Name())) // #nosec G304 -- file is from controlled directory
		if err != nil {
			continue
		}

		var snapshot Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// DeleteSnapshot deletes a snapshot
func (sm *SnapshotManager) DeleteSnapshot(snapshotID string) error {
	filename := filepath.Join(sm.snapshotDir, fmt.Sprintf("snapshot_%s.json", snapshotID))
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	return nil
}

// CleanupOldSnapshots removes snapshots older than the specified duration
func (sm *SnapshotManager) CleanupOldSnapshots(maxAge time.Duration) error {
	snapshots, err := sm.ListSnapshots()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	for _, snapshot := range snapshots {
		if snapshot.Timestamp.Before(cutoff) {
			if err := sm.DeleteSnapshot(snapshot.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateSnapshotID generates a unique snapshot ID
func generateSnapshotID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// CreateResourceSnapshot creates a resource snapshot from task execution
func CreateResourceSnapshot(ctx context.Context, task *types.Task, host *types.Host,
	currentState map[string]interface{}) ResourceSnapshot {

	resource := ResourceSnapshot{
		Type:       task.Module,
		Identifier: getResourceIdentifier(task),
		Host:       host.Name,
		State:      currentState,
		Action:     getActionFromTask(task),
		Module:     task.Module,
		TaskName:   task.Name,
		Reversible: isReversible(task.Module),
	}

	// Generate rollback operation if reversible
	if resource.Reversible {
		resource.RollbackOp = generateRollbackOperation(task, currentState)
	}

	return resource
}

// getResourceIdentifier extracts the resource identifier from task args
func getResourceIdentifier(task *types.Task) string {
	// Try common identifier fields
	identifierFields := []string{"path", "dest", "name", "src", "key"}

	for _, field := range identifierFields {
		if val, ok := task.Args[field]; ok {
			if strVal, ok := val.(string); ok {
				return strVal
			}
		}
	}

	return fmt.Sprintf("%s_%v", task.Module, task.Args)
}

// getActionFromTask determines the action from task args
func getActionFromTask(task *types.Task) string {
	if state, ok := task.Args["state"].(string); ok {
		return state
	}
	return "present"
}

// isReversible determines if a module's changes can be reversed
func isReversible(module string) bool {
	reversibleModules := map[string]bool{
		"file":       true,
		"copy":       true,
		"template":   true,
		"lineinfile": true,
		"package":    true,
		"service":    true,
		"user":       true,
		"group":      true,
		"git":        true,
		"systemd":    true,
		"cron":       true,
	}

	return reversibleModules[module]
}

// generateRollbackOperation generates a rollback operation for a task
func generateRollbackOperation(task *types.Task, currentState map[string]interface{}) *RollbackOperation {
	op := &RollbackOperation{
		Module: task.Module,
		Args:   make(map[string]interface{}),
		Order:  calculateRollbackOrder(task.Module),
	}

	switch task.Module {
	case "file", "copy", "template":
		// For file operations, restore previous state or remove if didn't exist
		if exists, ok := currentState["exists"].(bool); ok && !exists {
			// File didn't exist, remove it
			op.Args["path"] = task.Args["path"]
			op.Args["state"] = "absent"
		} else {
			// File existed, restore its content/permissions
			if content, ok := currentState["content"].(string); ok {
				op.Module = "copy"
				op.Args["dest"] = task.Args["path"]
				op.Args["content"] = content
			}
			if mode, ok := currentState["mode"].(string); ok {
				op.Args["mode"] = mode
			}
			if owner, ok := currentState["owner"].(string); ok {
				op.Args["owner"] = owner
			}
			if group, ok := currentState["group"].(string); ok {
				op.Args["group"] = group
			}
		}

	case "package":
		// For packages, reverse install/remove
		action := getActionFromTask(task)
		if action == "present" || action == "installed" {
			// Package was installed, remove it
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "absent"
		} else if action == "absent" || action == "removed" {
			// Package was removed, reinstall it
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "present"
			if version, ok := currentState["version"].(string); ok {
				op.Args["version"] = version
			}
		}

	case "service", "systemd":
		// For services, restore previous state
		if state, ok := currentState["state"].(string); ok {
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = state
		}
		if enabled, ok := currentState["enabled"].(bool); ok {
			op.Args["enabled"] = enabled
		}

	case "user":
		// For users, reverse create/remove
		action := getActionFromTask(task)
		if action == "present" {
			// User was created, remove it
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "absent"
		} else if action == "absent" {
			// User was removed, recreate it
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "present"
			// Restore user attributes from current state
			for key, value := range currentState {
				if key != "state" && key != "exists" {
					op.Args[key] = value
				}
			}
		}

	case "group":
		// Similar to user
		action := getActionFromTask(task)
		if action == "present" {
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "absent"
		} else if action == "absent" {
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "present"
			for key, value := range currentState {
				if key != "state" && key != "exists" {
					op.Args[key] = value
				}
			}
		}

	case "lineinfile":
		// For lineinfile, restore original content
		if originalLine, ok := currentState["original_line"].(string); ok {
			op.Args["path"] = task.Args["path"]
			op.Args["line"] = originalLine
			op.Args["state"] = "present"
		} else {
			// Line didn't exist, remove it
			op.Args["path"] = task.Args["path"]
			op.Args["line"] = task.Args["line"]
			op.Args["state"] = "absent"
		}

	case "git":
		// For git, checkout previous commit
		if commit, ok := currentState["commit"].(string); ok {
			op.Args["repo"] = task.Args["repo"]
			op.Args["dest"] = task.Args["dest"]
			op.Args["version"] = commit
		}

	case "cron":
		// For cron, reverse add/remove
		action := getActionFromTask(task)
		if action == "present" {
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "absent"
		} else if action == "absent" {
			op.Args["name"] = task.Args["name"]
			op.Args["state"] = "present"
			for key, value := range currentState {
				if key != "state" && key != "exists" {
					op.Args[key] = value
				}
			}
		}

	default:
		// For unknown modules, mark as non-reversible
		return nil
	}

	return op
}

// calculateRollbackOrder determines the order in which rollback operations should be executed
// Higher numbers are executed first during rollback
func calculateRollbackOrder(module string) int {
	orderMap := map[string]int{
		"service":  100, // Stop services first
		"systemd":  100,
		"cron":     90, // Remove cron jobs
		"file":     80, // Remove files
		"copy":     80,
		"template": 80,
		"git":      70, // Revert git repos
		"package":  60, // Uninstall packages
		"user":     50, // Remove users
		"group":    40, // Remove groups last
	}

	if order, ok := orderMap[module]; ok {
		return order
	}
	return 50 // Default order
}

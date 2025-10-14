package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type Manager struct {
	stateFile string
}

func New(stateFile string) *Manager {
	return &Manager{
		stateFile: stateFile,
	}
}

// LoadState loads state from file
func (m *Manager) LoadState() (*types.State, error) {
	if _, err := os.Stat(m.stateFile); os.IsNotExist(err) {
		return &types.State{
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(m.stateFile) // #nosec G304 -- stateFile is constructed from fixed state file path
	if err != nil {
		return nil, fmt.Errorf("error reading state file: %w", err)
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("error parsing state: %w", err)
	}

	return &state, nil
}

// SaveState saves state to file
func (m *Manager) SaveState(state *types.State) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing state: %w", err)
	}

	if err := os.WriteFile(m.stateFile, data, 0600); err != nil {
		return fmt.Errorf("error writing state file: %w", err)
	}

	return nil
}

// UpdateState updates state with new results
func (m *Manager) UpdateState(state *types.State, results []types.PlayResult) {
	state.Results = results
	if len(results) > 0 {
		state.LastRun = results[len(results)-1].EndTime
	} else {
		state.LastRun = time.Now()
	}
}

// CalculateChecksum calculates file checksum
func (m *Manager) CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath) // #nosec G304 -- filePath is validated by security validator
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// HasFileChanged checks if file has changed since last run
func (m *Manager) HasFileChanged(filePath string, state *types.State) (bool, error) {
	checksum, err := m.CalculateChecksum(filePath)
	if err != nil {
		return false, err
	}

	lastChecksum, exists := state.Checksums[filePath]
	if !exists {
		// New file
		state.Checksums[filePath] = checksum
		return true, nil
	}

	if checksum != lastChecksum {
		// File changed
		state.Checksums[filePath] = checksum
		return true, nil
	}

	return false, nil
}

// GetLastResults returns results from last run
func (m *Manager) GetLastResults(state *types.State) []types.PlayResult {
	return state.Results
}

// CleanupOldResults removes old results (older than 30 days)
func (m *Manager) CleanupOldResults(state *types.State) {
	cutoff := time.Now().AddDate(0, 0, -30)
	var filteredResults []types.PlayResult

	for _, result := range state.Results {
		if result.EndTime.After(cutoff) {
			filteredResults = append(filteredResults, result)
		}
	}

	state.Results = filteredResults
}

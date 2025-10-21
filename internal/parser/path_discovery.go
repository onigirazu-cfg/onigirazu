package parser

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathDiscovery provides priority-based discovery for inventory and config files
type PathDiscovery struct {
	logger interface {
		Debug(format string, args ...interface{})
		Info(format string, args ...interface{})
		Warn(format string, args ...interface{})
	}
}

// NewPathDiscovery creates a new path discovery utility
func NewPathDiscovery(logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
}) *PathDiscovery {
	return &PathDiscovery{
		logger: logger,
	}
}

// DiscoverInventoryFilePath discovers inventory file with priority system:
// 1. Priority 1: Inline inventory in playbook (checked by caller before calling this)
// 2. Priority 2: File nearby the playbook (playbookDir)
// 3. Priority 3: File in /etc/onigirazu/
func (d *PathDiscovery) DiscoverInventoryFilePath(playbookDir string) (string, int, error) {
	// List of common inventory file names (in order of preference)
	inventoryFiles := []string{
		"inventory.yml",
		"inventory.yaml",
		"inventory.toml",
		"inventory.json",
		"inventory.ini",
		"hosts",
		"hosts.yml",
		"hosts.yaml",
		"hosts.toml",
		"hosts.json",
		"hosts.ini",
		"inventory",
	}

	// Priority 2: Check in playbook directory
	if playbookDir != "" {
		for _, filename := range inventoryFiles {
			path := filepath.Join(playbookDir, filename)
			if _, err := os.Stat(path); err == nil {
				d.logger.Debug("Found inventory file (Priority 2): %s", path)
				return path, 2, nil
			}
		}
		d.logger.Debug("No inventory file found in playbook directory: %s", playbookDir)
	}

	// Priority 3: Check in /etc/onigirazu/
	etcSearchFiles := []string{"inventory.yml", "hosts.yml", "inventory"}
	for _, filename := range etcSearchFiles {
		path := filepath.Join("/etc/onigirazu", filename)
		if _, err := os.Stat(path); err == nil {
			d.logger.Debug("Found inventory file (Priority 3): %s", path)
			return path, 3, nil
		}
	}
	d.logger.Debug("No inventory file found in /etc/onigirazu/")

	return "", 0, fmt.Errorf("no inventory file found (searched: playbook_dir=%s, /etc/onigirazu/)", playbookDir)
}

// DiscoverConfigFilePath discovers config file with priority system:
// 1. Priority 1: Explicitly specified path (checked by caller before calling this)
// 2. Priority 2: File nearby the playbook (playbookDir)
// 3. Priority 3: File in /etc/onigirazu/
func (d *PathDiscovery) DiscoverConfigFilePath(playbookDir string) (string, int, error) {
	configFileName := "onigirazu.yml"

	// Priority 2: Check in playbook directory
	if playbookDir != "" {
		path := filepath.Join(playbookDir, configFileName)
		if _, err := os.Stat(path); err == nil {
			d.logger.Debug("Found config file (Priority 2): %s", path)
			return path, 2, nil
		}
		d.logger.Debug("No config file found in playbook directory: %s", path)
	}

	// Priority 3: Check in /etc/onigirazu/
	etcPath := filepath.Join("/etc/onigirazu", configFileName)
	if _, err := os.Stat(etcPath); err == nil {
		d.logger.Debug("Found config file (Priority 3): %s", etcPath)
		return etcPath, 3, nil
	}
	d.logger.Debug("No config file found in /etc/onigirazu: %s", etcPath)

	// No config file found (this is not an error - returns empty path)
	return "", 0, nil
}

// DiscoverInventoryFilePathWithPriority is like DiscoverInventoryFilePath but accepts explicit path
// and checks it as Priority 1
// Returns: (path, priority, error)
// Priority: 1 = explicit path, 2 = playbook dir, 3 = /etc/onigirazu/
func (d *PathDiscovery) DiscoverInventoryFilePathWithPriority(explicitPath, playbookDir string) (string, int, error) {
	// Priority 1: Explicit path provided
	if explicitPath != "" {
		// This should be validated by caller, but we check it exists
		if _, err := os.Stat(explicitPath); err == nil {
			d.logger.Debug("Using explicit inventory file (Priority 1): %s", explicitPath)
			return explicitPath, 1, nil
		}
		return "", 0, fmt.Errorf("explicit inventory file not found: %s", explicitPath)
	}

	// No explicit path, use normal discovery (Priority 2 & 3)
	return d.DiscoverInventoryFilePath(playbookDir)
}

// DiscoverConfigFilePathWithPriority is like DiscoverConfigFilePath but accepts explicit path
// and checks it as Priority 1
// Returns: (path, priority, error)
// Priority: 1 = explicit path, 2 = playbook dir, 3 = /etc/onigirazu/, 0 = not found (no error)
func (d *PathDiscovery) DiscoverConfigFilePathWithPriority(explicitPath, playbookDir string) (string, int, error) {
	// Priority 1: Explicit path provided
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			d.logger.Debug("Using explicit config file (Priority 1): %s", explicitPath)
			return explicitPath, 1, nil
		}
		return "", 0, fmt.Errorf("explicit config file not found: %s", explicitPath)
	}

	// No explicit path, use normal discovery (Priority 2 & 3)
	return d.DiscoverConfigFilePath(playbookDir)
}

package folderstructure

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// VariableLoader handles loading and merging variables from different sources
type VariableLoader struct {
	detector *Detector
	cache    *Cache
	mu       sync.RWMutex
}

// NewVariableLoader creates a new VariableLoader
func NewVariableLoader(detector *Detector) *VariableLoader {
	return &VariableLoader{
		detector: detector,
		cache:    NewCache(30*time.Minute, 500), // 30 min TTL for variables
	}
}

// LoadVariables loads variables from the given path with Ansible-style precedence
// The precedence order is:
// 1. defaults/main.yml and defaults/*.yml
// 2. vars/main.yml and vars/*.yml
// 3. Inline variables (if provided)
// 4. Inventory variables (if provided)
// 5. Task variables (if provided)
// 6. Playbook variables (if provided)
// 7. CLI variables (if provided)
func (vl *VariableLoader) LoadVariables(projectPath string) (*VariableSet, error) {
	vl.mu.RLock()
	if cached, found := vl.cache.Get(projectPath); found {
		vl.mu.RUnlock()
		if varSet, ok := cached.(*VariableSet); ok {
			return varSet, nil
		}
	}
	vl.mu.RUnlock()

	structure, err := vl.detector.Detect(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project structure: %w", err)
	}

	varSet := &VariableSet{
		Variables:  make(map[string]interface{}),
		Metadata:   make(map[string]VariableSource),
		Precedence: make([]string, 0),
	}

	// Load defaults
	if structure.HasDefaults {
		defaultsPath := filepath.Join(structure.RootPath, "defaults")
		if err := vl.loadVariablesFromDir(defaultsPath, varSet, "defaults", 1); err != nil {
			return nil, fmt.Errorf("failed to load defaults: %w", err)
		}
	}

	// Load vars (higher precedence than defaults)
	if structure.HasVars {
		varsPath := filepath.Join(structure.RootPath, "vars")
		if err := vl.loadVariablesFromDir(varsPath, varSet, "vars", 2); err != nil {
			return nil, fmt.Errorf("failed to load vars: %w", err)
		}
	}

	// Cache the result
	vl.mu.Lock()
	vl.cache.Set(projectPath, varSet)
	vl.mu.Unlock()

	return varSet, nil
}

// loadVariablesFromDir loads all YAML files from a directory
func (vl *VariableLoader) loadVariablesFromDir(dirPath string, varSet *VariableSet, source string, priority int) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Load main.yml first
	for _, entry := range entries {
		if entry.Name() == "main.yml" {
			if err := vl.loadVariablesFromFile(
				filepath.Join(dirPath, entry.Name()),
				varSet,
				source,
				"main.yml",
				priority,
			); err != nil {
				return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
			}
			break
		}
	}

	// Then load other yml files
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "main.yml" {
			continue
		}

		if filepath.Ext(entry.Name()) == ".yml" || filepath.Ext(entry.Name()) == ".yaml" {
			if err := vl.loadVariablesFromFile(
				filepath.Join(dirPath, entry.Name()),
				varSet,
				source,
				entry.Name(),
				priority,
			); err != nil {
				return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// loadVariablesFromFile loads variables from a single YAML file
func (vl *VariableLoader) loadVariablesFromFile(filePath string, varSet *VariableSet, source, fileName string, priority int) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var variables map[string]interface{}
	if err := yaml.Unmarshal(data, &variables); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Only update if not already set (respecting precedence)
	// but newer sources can override older ones
	for key, value := range variables {
		// Check if we should override based on precedence
		if _, ok := varSet.Variables[key]; ok {
			if meta, ok := varSet.Metadata[key]; ok {
				if priority < meta.Priority {
					// Lower priority source, don't override
					continue
				}
			} else {
				// No metadata, keep existing
				continue
			}
		}

		varSet.Variables[key] = value
		varSet.Metadata[key] = VariableSource{
			Source:   source,
			File:     fileName,
			Priority: priority,
		}
	}

	return nil
}

// MergeVariables merges multiple variable sets with proper precedence
func (vl *VariableLoader) MergeVariables(varSets ...*VariableSet) *VariableSet {
	merged := &VariableSet{
		Variables:  make(map[string]interface{}),
		Metadata:   make(map[string]VariableSource),
		Precedence: make([]string, 0),
	}

	for i, varSet := range varSets {
		priority := i + 1
		for key, value := range varSet.Variables {
			if _, ok := merged.Variables[key]; ok {
				if meta, ok := merged.Metadata[key]; ok {
					if priority < meta.Priority {
						// Higher priority wins (higher index = higher priority)
						continue
					}
				}
			} else {
				// Check if this is the first time seeing this key
				merged.Precedence = append(merged.Precedence, key)
			}

			merged.Variables[key] = value
			meta := varSet.Metadata[key]
			meta.Priority = priority
			merged.Metadata[key] = meta
		}
	}

	return merged
}

// ClearCache clears the internal cache
func (vl *VariableLoader) ClearCache() {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	vl.cache.Clear()
}

// GetVariable retrieves a single variable value
func (vl *VariableLoader) GetVariable(variables map[string]interface{}, key string) interface{} {
	return variables[key]
}

// SetVariable sets a variable value
func (vl *VariableLoader) SetVariable(variables map[string]interface{}, key string, value interface{}) {
	variables[key] = value
}

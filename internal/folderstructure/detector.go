package folderstructure

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Detector is responsible for detecting Ansible-style folder structures
type Detector struct {
	cache *Cache
	mu    sync.RWMutex

	// StandardDirs are the directories to check for
	StandardDirs []string
}

// NewDetector creates a new ProjectStructureDetector
func NewDetector() *Detector {
	return &Detector{
		cache: NewCache(1*time.Hour, 1000), // 1 hour TTL, max 1000 entries
		StandardDirs: []string{
			"defaults",
			"vars",
			"templates",
			"files",
			"handlers",
			"tasks",
		},
	}
}

// Detect detects the Ansible-style folder structure starting from the given path
// It walks up the directory tree to find the project root
func (d *Detector) Detect(startPath string) (*ProjectStructure, error) {
	d.mu.RLock()
	if cached, found := d.cache.Get(startPath); found {
		d.mu.RUnlock()
		return cached.(*ProjectStructure), nil
	}
	d.mu.RUnlock()

	// Resolve the start path
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Determine if it's a file or directory
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var searchPath string
	if fileInfo.IsDir() {
		searchPath = absPath
	} else {
		searchPath = filepath.Dir(absPath)
	}

	// Detect the project structure
	structure, err := d.detectInPath(searchPath)
	if err != nil {
		return nil, err
	}

	// Cache the result
	d.mu.Lock()
	d.cache.Set(startPath, structure)
	d.mu.Unlock()

	return structure, nil
}

// DetectProjectRoot finds the project root by looking for Ansible-style directories
// It starts from the given path and walks up the directory tree
func (d *Detector) DetectProjectRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Determine if it's a file or directory
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	var searchPath string
	if fileInfo.IsDir() {
		searchPath = absPath
	} else {
		searchPath = filepath.Dir(absPath)
	}

	// Walk up the directory tree
	for {
		structure, err := d.detectInPath(searchPath)
		if err == nil && d.hasAnyStandardDirs(structure) {
			return structure.RootPath, nil
		}

		// Move to parent directory
		parent := filepath.Dir(searchPath)
		if parent == searchPath {
			// Reached the root directory
			break
		}
		searchPath = parent
	}

	// If no structure found, return the original search path
	return searchPath, nil
}

// detectInPath detects the structure in the given path without walking up
func (d *Detector) detectInPath(path string) (*ProjectStructure, error) {
	structure := &ProjectStructure{
		RootPath:   path,
		DetectedAt: time.Now(),
		Metadata:   make(map[string]os.FileInfo),
	}

	// Check for each standard directory
	for _, dir := range d.StandardDirs {
		dirPath := filepath.Join(path, dir)
		fileInfo, err := os.Stat(dirPath)

		if err == nil && fileInfo.IsDir() {
			structure.Metadata[dir] = fileInfo

			switch dir {
			case "defaults":
				structure.HasDefaults = true
			case "vars":
				structure.HasVars = true
			case "templates":
				structure.HasTemplates = true
			case "files":
				structure.HasFiles = true
			case "handlers":
				structure.HasHandlers = true
			case "tasks":
				structure.HasTasks = true
			}
		}
	}

	return structure, nil
}

// hasAnyStandardDirs checks if the structure has any standard directories
func (d *Detector) hasAnyStandardDirs(structure *ProjectStructure) bool {
	return structure.HasDefaults ||
		structure.HasVars ||
		structure.HasTemplates ||
		structure.HasFiles ||
		structure.HasHandlers ||
		structure.HasTasks
}

// IsStructured checks if the given path has an Ansible-style structure
func (d *Detector) IsStructured(path string) (bool, error) {
	structure, err := d.Detect(path)
	if err != nil {
		return false, err
	}

	return d.hasAnyStandardDirs(structure), nil
}

// GetStandardDirs returns the list of standard directory names
func (d *Detector) GetStandardDirs() []string {
	dirs := make([]string, len(d.StandardDirs))
	copy(dirs, d.StandardDirs)
	return dirs
}

// ClearCache clears the internal cache
func (d *Detector) ClearCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cache.Clear()
}

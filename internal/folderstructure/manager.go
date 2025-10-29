package folderstructure

import (
	"fmt"
	"sync"
)

// Manager is the main interface for managing Ansible-style folder structures
type Manager struct {
	detector         *Detector
	variableLoader   *VariableLoader
	fileResolver     *FileResolver
	templateResolver *TemplateResolver
	handlerManager   *HandlerManager
	mu               sync.RWMutex
}

// NewManager creates a new Manager
func NewManager() *Manager {
	detector := NewDetector()
	return &Manager{
		detector:         detector,
		variableLoader:   NewVariableLoader(detector),
		fileResolver:     NewFileResolver(detector),
		templateResolver: NewTemplateResolver(detector),
		handlerManager:   NewHandlerManager(detector),
	}
}

// DetectStructure detects the Ansible-style folder structure at the given path
func (m *Manager) DetectStructure(path string) (*ProjectStructure, error) {
	return m.detector.Detect(path)
}

// DetectProjectRoot detects the project root starting from the given path
func (m *Manager) DetectProjectRoot(path string) (string, error) {
	return m.detector.DetectProjectRoot(path)
}

// IsStructured checks if the given path has an Ansible-style structure
func (m *Manager) IsStructured(path string) (bool, error) {
	return m.detector.IsStructured(path)
}

// LoadVariables loads variables from the given path with proper precedence
func (m *Manager) LoadVariables(path string) (*VariableSet, error) {
	return m.variableLoader.LoadVariables(path)
}

// ResolveFile resolves a file path with the given project path
func (m *Manager) ResolveFile(filePath string, projectPath string) *ResolutionResult {
	return m.fileResolver.ResolveFile(filePath, projectPath)
}

// ResolveTemplate resolves a template path with the given project path
func (m *Manager) ResolveTemplate(templatePath string, projectPath string) *ResolutionResult {
	return m.templateResolver.ResolveTemplate(templatePath, projectPath)
}

// LoadHandlers loads handlers from the given path
func (m *Manager) LoadHandlers(projectPath string) ([]*Handler, error) {
	return m.handlerManager.LoadHandlers(projectPath)
}

// GetProjectStructure retrieves the cached project structure for the given path
func (m *Manager) GetProjectStructure(path string) (*ProjectStructure, error) {
	return m.detector.Detect(path)
}

// ClearCache clears all internal caches
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.detector.ClearCache()
	m.variableLoader.ClearCache()
	m.fileResolver.ClearCache()
	m.templateResolver.ClearCache()
	m.handlerManager.ClearCache()
}

// ValidatePath validates a path within the given base path
func (m *Manager) ValidatePath(basePath string, path string) error {
	if err := m.fileResolver.ValidateFilePath(basePath, path); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	return nil
}

// GetStatistics returns statistics about the manager's caches
func (m *Manager) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"feature":    "ansible_folder_structure",
		"version":    "1.0.0",
		"components": 5,
		"detectors":  1,
		"resolvers":  2,
		"managers":   2,
	}
}

package folderstructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TemplateResolver handles resolving templates from the templates/ directory
type TemplateResolver struct {
	detector *Detector
	cache    *Cache
	mu       sync.RWMutex
}

// NewTemplateResolver creates a new TemplateResolver
func NewTemplateResolver(detector *Detector) *TemplateResolver {
	return &TemplateResolver{
		detector: detector,
		cache:    NewCache(30*time.Minute, 500), // 30 min TTL
	}
}

// ResolveTemplate resolves a template path with the following precedence:
// 1. templates/ directory (if project structure detected)
// 2. Relative to project root
// 3. Absolute path
// 4. Relative to current working directory
func (tr *TemplateResolver) ResolveTemplate(templatePath string, projectPath string) *ResolutionResult {
	cacheKey := fmt.Sprintf("template:%s:%s", templatePath, projectPath)
	tr.mu.RLock()
	if cached, found := tr.cache.Get(cacheKey); found {
		tr.mu.RUnlock()
		return cached.(*ResolutionResult)
	}
	tr.mu.RUnlock()

	result := &ResolutionResult{
		Path:  templatePath,
		Found: false,
	}

	// Check if it's an absolute path
	if filepath.IsAbs(templatePath) {
		if tr.fileExists(templatePath) {
			result.Path = templatePath
			result.Found = true
			result.Source = "absolute"
			tr.mu.Lock()
			tr.cache.Set(cacheKey, result)
			tr.mu.Unlock()
			return result
		}
	}

	// Try to detect project structure
	structure, err := tr.detector.Detect(projectPath)
	if err == nil && structure.HasTemplates {
		// Try templates/ directory
		templatesDir := filepath.Join(structure.RootPath, "templates")
		resolvedPath := filepath.Join(templatesDir, templatePath)
		if tr.fileExists(resolvedPath) {
			result.Path = resolvedPath
			result.Found = true
			result.Source = "templates"
			tr.mu.Lock()
			tr.cache.Set(cacheKey, result)
			tr.mu.Unlock()
			return result
		}
	}

	// Try relative to project root
	if structure != nil {
		relativePath := filepath.Join(structure.RootPath, templatePath)
		if tr.fileExists(relativePath) {
			result.Path = relativePath
			result.Found = true
			result.Source = "relative"
			tr.mu.Lock()
			tr.cache.Set(cacheKey, result)
			tr.mu.Unlock()
			return result
		}
	}

	// Try current working directory
	if tr.fileExists(templatePath) {
		absPath, err := filepath.Abs(templatePath)
		if err != nil {
			absPath = templatePath
		}
		result.Path = absPath
		result.Found = true
		result.Source = "cwd"
		tr.mu.Lock()
		tr.cache.Set(cacheKey, result)
		tr.mu.Unlock()
		return result
	}

	result.Found = false
	result.Error = fmt.Errorf("template not found: %s", templatePath)
	tr.mu.Lock()
	tr.cache.Set(cacheKey, result)
	tr.mu.Unlock()
	return result
}

// ResolveTemplatesInDir resolves all templates in a directory
func (tr *TemplateResolver) ResolveTemplatesInDir(dirPath string, projectPath string) ([]*ResolutionResult, error) {
	result := tr.ResolveTemplate(dirPath, projectPath)
	if !result.Found {
		return nil, fmt.Errorf("directory not found: %s", dirPath)
	}

	entries, err := os.ReadDir(result.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var results []*ResolutionResult
	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(result.Path, entry.Name())
			results = append(results, &ResolutionResult{
				Path:   filePath,
				Found:  true,
				Source: "directory",
			})
		}
	}

	return results, nil
}

// fileExists checks if a file exists
func (tr *TemplateResolver) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ClearCache clears the internal cache
func (tr *TemplateResolver) ClearCache() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.cache.Clear()
}

// ValidateTemplatePath validates that a template path is safe (prevents path traversal)
func (tr *TemplateResolver) ValidateTemplatePath(basePath string, templatePath string) error {
	// Get absolute paths
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %w", err)
	}

	absTemplate := filepath.Join(absBase, templatePath)
	absTemplate, err = filepath.Abs(absTemplate)
	if err != nil {
		return fmt.Errorf("invalid template path: %w", err)
	}

	// Check if the resolved path is within the base path
	rel, err := filepath.Rel(absBase, absTemplate)
	if err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	// Check for ../ in the relative path
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path traversal detected: %s", templatePath)
	}

	return nil
}

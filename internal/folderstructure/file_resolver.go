package folderstructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileResolver handles resolving files from the files/ directory
type FileResolver struct {
	detector *Detector
	cache    *Cache
	mu       sync.RWMutex
}

// NewFileResolver creates a new FileResolver
func NewFileResolver(detector *Detector) *FileResolver {
	return &FileResolver{
		detector: detector,
		cache:    NewCache(1*time.Hour, 1000), // 1 hour TTL
	}
}

// ResolveFile resolves a file path with the following precedence:
// 1. files/ directory (if project structure detected)
// 2. Relative to project root
// 3. Absolute path
// 4. Relative to current working directory
func (fr *FileResolver) ResolveFile(filePath string, projectPath string) *ResolutionResult {
	cacheKey := fmt.Sprintf("file:%s:%s", filePath, projectPath)
	fr.mu.RLock()
	if cached, found := fr.cache.Get(cacheKey); found {
		fr.mu.RUnlock()
		return cached.(*ResolutionResult)
	}
	fr.mu.RUnlock()

	result := &ResolutionResult{
		Path:  filePath,
		Found: false,
	}

	// Check if it's an absolute path
	if filepath.IsAbs(filePath) {
		if fr.fileExists(filePath) {
			result.Path = filePath
			result.Found = true
			result.Source = "absolute"
			fr.mu.Lock()
			fr.cache.Set(cacheKey, result)
			fr.mu.Unlock()
			return result
		}
	}

	// Try to detect project structure
	structure, err := fr.detector.Detect(projectPath)
	if err == nil && structure.HasFiles {
		// Try files/ directory
		filesDir := filepath.Join(structure.RootPath, "files")
		filePath := filepath.Join(filesDir, filePath)
		if fr.fileExists(filePath) {
			result.Path = filePath
			result.Found = true
			result.Source = "files"
			fr.mu.Lock()
			fr.cache.Set(cacheKey, result)
			fr.mu.Unlock()
			return result
		}
	}

	// Try relative to project root
	if structure != nil {
		relativePath := filepath.Join(structure.RootPath, filePath)
		if fr.fileExists(relativePath) {
			result.Path = relativePath
			result.Found = true
			result.Source = "relative"
			fr.mu.Lock()
			fr.cache.Set(cacheKey, result)
			fr.mu.Unlock()
			return result
		}
	}

	// Try current working directory
	if fr.fileExists(filePath) {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			absPath = filePath
		}
		result.Path = absPath
		result.Found = true
		result.Source = "cwd"
		fr.mu.Lock()
		fr.cache.Set(cacheKey, result)
		fr.mu.Unlock()
		return result
	}

	result.Found = false
	result.Error = fmt.Errorf("file not found: %s", filePath)
	fr.mu.Lock()
	fr.cache.Set(cacheKey, result)
	fr.mu.Unlock()
	return result
}

// ResolveFilesInDir resolves all files in a directory
func (fr *FileResolver) ResolveFilesInDir(dirPath string, projectPath string) ([]*ResolutionResult, error) {
	result := fr.ResolveFile(dirPath, projectPath)
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
func (fr *FileResolver) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ClearCache clears the internal cache
func (fr *FileResolver) ClearCache() {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.cache.Clear()
}

// ValidateFilePath validates that a file path is safe (prevents path traversal)
func (fr *FileResolver) ValidateFilePath(basePath string, filePath string) error {
	// Get absolute paths
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %w", err)
	}

	absFile := filepath.Join(absBase, filePath)
	absFile, err = filepath.Abs(absFile)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Check if the resolved path is within the base path
	rel, err := filepath.Rel(absBase, absFile)
	if err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	// Check for ../ in the relative path
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path traversal detected: %s", filePath)
	}

	return nil
}

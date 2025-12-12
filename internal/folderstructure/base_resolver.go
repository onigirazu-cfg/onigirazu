package folderstructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ResolverConfig struct {
	DirectoryName string
	CacheKeyPrefix string
	HasDirectoryField func(*ProjectStructure) bool
	TTL time.Duration
	MaxCacheSize int
	NotFoundMessage func(path string) string
}

type BaseResolver struct {
	detector *Detector
	cache    *Cache
	config   ResolverConfig
	mu       sync.RWMutex
}

func NewBaseResolver(detector *Detector, config ResolverConfig) *BaseResolver {
	return &BaseResolver{
		detector: detector,
		cache:    NewCache(config.TTL, config.MaxCacheSize),
		config:   config,
	}
}

func (br *BaseResolver) Resolve(resourcePath string, projectPath string) *ResolutionResult {
	cacheKey := fmt.Sprintf("%s:%s:%s", br.config.CacheKeyPrefix, resourcePath, projectPath)
	br.mu.RLock()
	if cached, found := br.cache.Get(cacheKey); found {
		br.mu.RUnlock()
		if result, ok := cached.(*ResolutionResult); ok {
			return result
		}
	}
	br.mu.RUnlock()

	result := &ResolutionResult{
		Path:  resourcePath,
		Found: false,
	}

	// Check if it's an absolute path
	if filepath.IsAbs(resourcePath) {
		if br.fileExists(resourcePath) {
			result.Path = resourcePath
			result.Found = true
			result.Source = "absolute"
			br.cacheResult(cacheKey, result)
			return result
		}
	}

	// Try to detect project structure
	structure, err := br.detector.Detect(projectPath)
	if err == nil && br.config.HasDirectoryField(structure) {
		// Try resource directory
		resourceDir := filepath.Join(structure.RootPath, br.config.DirectoryName)
		resolvedPath := filepath.Join(resourceDir, resourcePath)
		if br.fileExists(resolvedPath) {
			result.Path = resolvedPath
			result.Found = true
			result.Source = br.config.DirectoryName
			br.cacheResult(cacheKey, result)
			return result
		}
	}

	// Try relative to project root
	if structure != nil {
		relativePath := filepath.Join(structure.RootPath, resourcePath)
		if br.fileExists(relativePath) {
			result.Path = relativePath
			result.Found = true
			result.Source = "relative"
			br.cacheResult(cacheKey, result)
			return result
		}
	}

	// Try current working directory
	if br.fileExists(resourcePath) {
		absPath, err := filepath.Abs(resourcePath)
		if err != nil {
			absPath = resourcePath
		}
		result.Path = absPath
		result.Found = true
		result.Source = "cwd"
		br.cacheResult(cacheKey, result)
		return result
	}

	result.Found = false
	result.Error = fmt.Errorf("%s", br.config.NotFoundMessage(resourcePath))
	br.cacheResult(cacheKey, result)
	return result
}

func (br *BaseResolver) ResolveInDir(dirPath string, projectPath string) ([]*ResolutionResult, error) {
	result := br.Resolve(dirPath, projectPath)
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

func (br *BaseResolver) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (br *BaseResolver) cacheResult(cacheKey string, result *ResolutionResult) {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.cache.Set(cacheKey, result)
}

func (br *BaseResolver) ClearCache() {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.cache.Clear()
}

func (br *BaseResolver) ValidatePath(basePath string, resourcePath string) error {
	// Get absolute paths
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %w", err)
	}

	absResource := filepath.Join(absBase, resourcePath)
	absResource, err = filepath.Abs(absResource)
	if err != nil {
		return fmt.Errorf("invalid resource path: %w", err)
	}

	// Check if the resolved path is within the base path
	rel, err := filepath.Rel(absBase, absResource)
	if err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	// Check for ../ in the relative path
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path traversal detected: %s", resourcePath)
	}

	return nil
}

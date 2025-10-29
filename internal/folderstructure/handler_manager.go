package folderstructure

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// HandlerManager handles loading and managing handlers from the handlers/ directory
type HandlerManager struct {
	detector *Detector
	cache    *Cache
	mu       sync.RWMutex
}

// Handler represents a handler definition
type Handler struct {
	Name   string                   `yaml:"name"`
	Listen string                   `yaml:"listen,omitempty"`
	Notify []string                 `yaml:"notify,omitempty"`
	Tasks  []map[string]interface{} `yaml:"tasks,omitempty"`
	Block  []map[string]interface{} `yaml:"block,omitempty"`
	Always []map[string]interface{} `yaml:"always,omitempty"`
	Rescue []map[string]interface{} `yaml:"rescue,omitempty"`
	Meta   map[string]interface{}   `yaml:"meta,omitempty"`
	Tags   []string                 `yaml:"tags,omitempty"`
	Vars   map[string]interface{}   `yaml:"vars,omitempty"`
}

// NewHandlerManager creates a new HandlerManager
func NewHandlerManager(detector *Detector) *HandlerManager {
	return &HandlerManager{
		detector: detector,
		cache:    NewCache(1*time.Hour, 500), // 1 hour TTL
	}
}

// LoadHandlers loads handlers from the handlers/ directory
func (hm *HandlerManager) LoadHandlers(projectPath string) ([]*Handler, error) {
	hm.mu.RLock()
	if cached, found := hm.cache.Get(projectPath); found {
		hm.mu.RUnlock()
		return cached.([]*Handler), nil
	}
	hm.mu.RUnlock()

	structure, err := hm.detector.Detect(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project structure: %w", err)
	}

	var handlers []*Handler

	// Load handlers if directory exists
	if structure.HasHandlers {
		handlersPath := filepath.Join(structure.RootPath, "handlers")
		loadedHandlers, err := hm.loadHandlersFromDir(handlersPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load handlers: %w", err)
		}
		handlers = append(handlers, loadedHandlers...)
	}

	// Cache the result
	hm.mu.Lock()
	hm.cache.Set(projectPath, handlers)
	hm.mu.Unlock()

	return handlers, nil
}

// loadHandlersFromDir loads all handlers from a directory
func (hm *HandlerManager) loadHandlersFromDir(dirPath string) ([]*Handler, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var handlers []*Handler

	// Load main.yml first
	for _, entry := range entries {
		if entry.Name() == "main.yml" {
			loadedHandlers, err := hm.loadHandlersFromFile(filepath.Join(dirPath, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to load main.yml: %w", err)
			}
			handlers = append(handlers, loadedHandlers...)
			break
		}
	}

	// Then load other yml files
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "main.yml" {
			continue
		}

		if filepath.Ext(entry.Name()) == ".yml" || filepath.Ext(entry.Name()) == ".yaml" {
			loadedHandlers, err := hm.loadHandlersFromFile(filepath.Join(dirPath, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", entry.Name(), err)
			}
			handlers = append(handlers, loadedHandlers...)
		}
	}

	return handlers, nil
}

// loadHandlersFromFile loads handlers from a single YAML file
func (hm *HandlerManager) loadHandlersFromFile(filePath string) ([]*Handler, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var rawHandlers []map[string]interface{}
	if err := yaml.Unmarshal(data, &rawHandlers); err != nil {
		// Try single handler
		var singleHandler map[string]interface{}
		if err := yaml.Unmarshal(data, &singleHandler); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
		rawHandlers = []map[string]interface{}{singleHandler}
	}

	var handlers []*Handler
	for _, raw := range rawHandlers {
		handler := &Handler{
			Meta: make(map[string]interface{}),
			Vars: make(map[string]interface{}),
		}

		// Parse handler fields
		if name, ok := raw["name"].(string); ok {
			handler.Name = name
		}
		if listen, ok := raw["listen"].(string); ok {
			handler.Listen = listen
		}
		if tags, ok := raw["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					handler.Tags = append(handler.Tags, tagStr)
				}
			}
		}

		// Parse notify field
		if notify, ok := raw["notify"]; ok {
			switch notifyVal := notify.(type) {
			case string:
				handler.Notify = []string{notifyVal}
			case []interface{}:
				for _, n := range notifyVal {
					if nStr, ok := n.(string); ok {
						handler.Notify = append(handler.Notify, nStr)
					}
				}
			}
		}

		// Store any additional fields as metadata
		for key, value := range raw {
			if key != "name" && key != "listen" && key != "notify" && key != "tags" {
				handler.Meta[key] = value
			}
		}

		handlers = append(handlers, handler)
	}

	return handlers, nil
}

// GetHandlerByName retrieves a handler by name
func (hm *HandlerManager) GetHandlerByName(handlers []*Handler, name string) *Handler {
	for _, handler := range handlers {
		if handler.Name == name || handler.Listen == name {
			return handler
		}
	}
	return nil
}

// ClearCache clears the internal cache
func (hm *HandlerManager) ClearCache() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cache.Clear()
}

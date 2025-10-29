package folderstructure

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHandlerManagerLoadHandlers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create handlers directory
	handlersDir := filepath.Join(tmpDir, "handlers")
	if err := os.Mkdir(handlersDir, 0755); err != nil {
		t.Fatalf("failed to create handlers dir: %v", err)
	}

	// Create handlers/main.yml
	handlers := []map[string]interface{}{
		{
			"name":   "restart apache",
			"listen": "apache restart",
		},
		{
			"name":   "reload nginx",
			"listen": "nginx reload",
		},
	}

	handlersData, err := yaml.Marshal(handlers)
	if err != nil {
		t.Fatalf("failed to marshal handlers: %v", err)
	}

	handlersFile := filepath.Join(handlersDir, "main.yml")
	if err := os.WriteFile(handlersFile, handlersData, 0644); err != nil {
		t.Fatalf("failed to write handlers: %v", err)
	}

	detector := NewDetector()
	manager := NewHandlerManager(detector)

	loadedHandlers, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("failed to load handlers: %v", err)
	}

	if len(loadedHandlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(loadedHandlers))
	}

	// Check first handler
	if loadedHandlers[0].Name != "restart apache" {
		t.Errorf("expected first handler name 'restart apache', got %s", loadedHandlers[0].Name)
	}
}

func TestHandlerManagerGetHandlerByName(t *testing.T) {
	handlers := []*Handler{
		{
			Name:   "restart apache",
			Listen: "apache restart",
		},
		{
			Name:   "reload nginx",
			Listen: "nginx reload",
		},
	}

	detector := NewDetector()
	manager := NewHandlerManager(detector)

	// Test get by name
	handler := manager.GetHandlerByName(handlers, "restart apache")
	if handler == nil {
		t.Error("expected to find handler by name")
	}

	// Test get by listen
	handler = manager.GetHandlerByName(handlers, "nginx reload")
	if handler == nil {
		t.Error("expected to find handler by listen")
	}

	// Test handler not found
	handler = manager.GetHandlerByName(handlers, "nonexistent")
	if handler != nil {
		t.Error("expected handler to not be found")
	}
}

func TestHandlerManagerCaching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create handlers directory
	handlersDir := filepath.Join(tmpDir, "handlers")
	if err := os.Mkdir(handlersDir, 0755); err != nil {
		t.Fatalf("failed to create handlers dir: %v", err)
	}

	// Create handlers/main.yml
	handlers := []map[string]interface{}{
		{
			"name":   "restart apache",
			"listen": "apache restart",
		},
	}

	handlersData, err := yaml.Marshal(handlers)
	if err != nil {
		t.Fatalf("failed to marshal handlers: %v", err)
	}

	handlersFile := filepath.Join(handlersDir, "main.yml")
	if err := os.WriteFile(handlersFile, handlersData, 0644); err != nil {
		t.Fatalf("failed to write handlers: %v", err)
	}

	detector := NewDetector()
	manager := NewHandlerManager(detector)

	// First load
	handlers1, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	// Second load should come from cache
	handlers2, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	if len(handlers1) != len(handlers2) {
		t.Error("cached handlers should match original")
	}
}

func TestHandlerManagerEmptyHandlers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty handlers directory
	handlersDir := filepath.Join(tmpDir, "handlers")
	if err := os.Mkdir(handlersDir, 0755); err != nil {
		t.Fatalf("failed to create handlers dir: %v", err)
	}

	detector := NewDetector()
	manager := NewHandlerManager(detector)

	loadedHandlers, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("failed to load handlers: %v", err)
	}

	if len(loadedHandlers) != 0 {
		t.Errorf("expected 0 handlers, got %d", len(loadedHandlers))
	}
}

func TestHandlerManagerClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create handlers directory
	handlersDir := filepath.Join(tmpDir, "handlers")
	if err := os.Mkdir(handlersDir, 0755); err != nil {
		t.Fatalf("failed to create handlers dir: %v", err)
	}

	// Create handlers/main.yml
	handlers := []map[string]interface{}{
		{
			"name": "test handler",
		},
	}

	handlersData, err := yaml.Marshal(handlers)
	if err != nil {
		t.Fatalf("failed to marshal handlers: %v", err)
	}

	handlersFile := filepath.Join(handlersDir, "main.yml")
	if err := os.WriteFile(handlersFile, handlersData, 0644); err != nil {
		t.Fatalf("failed to write handlers: %v", err)
	}

	detector := NewDetector()
	manager := NewHandlerManager(detector)

	// Populate cache
	_, err = manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Clear cache
	manager.ClearCache()

	// Should still be able to load after cache clear
	handlers2, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("failed to load after cache clear: %v", err)
	}

	if len(handlers2) == 0 {
		t.Error("expected handlers after cache clear")
	}
}

func TestHandlerManagerParseNotify(t *testing.T) {
	tmpDir := t.TempDir()

	// Create handlers directory
	handlersDir := filepath.Join(tmpDir, "handlers")
	if err := os.Mkdir(handlersDir, 0755); err != nil {
		t.Fatalf("failed to create handlers dir: %v", err)
	}

	// Create handlers/main.yml with notify field
	handlers := []map[string]interface{}{
		{
			"name":   "test handler",
			"listen": "test",
			"notify": []interface{}{"restart service", "reload config"},
		},
	}

	handlersData, err := yaml.Marshal(handlers)
	if err != nil {
		t.Fatalf("failed to marshal handlers: %v", err)
	}

	handlersFile := filepath.Join(handlersDir, "main.yml")
	if err := os.WriteFile(handlersFile, handlersData, 0644); err != nil {
		t.Fatalf("failed to write handlers: %v", err)
	}

	detector := NewDetector()
	manager := NewHandlerManager(detector)

	loadedHandlers, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("failed to load handlers: %v", err)
	}

	if len(loadedHandlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(loadedHandlers))
	}

	if len(loadedHandlers[0].Notify) != 2 {
		t.Errorf("expected 2 notify items, got %d", len(loadedHandlers[0].Notify))
	}
}
